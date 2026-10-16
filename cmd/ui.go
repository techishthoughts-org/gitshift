package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	gitshiftconfig "github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
	gitshifttui "github.com/techishthoughts/gitshift/internal/tui"
	"golang.org/x/term"
)

var terminalCheck = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

var uiCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"tui"},
	Short:   "Open the interactive terminal interface",
	Long: `Open GitShift's keyboard-driven terminal interface.

The UI manages accounts, validated switching, directory rules, and explicit
discovery imports. Unsafe force options and key generation remain CLI-only.
Both input and output must be attached to a terminal.`,
	Args: cobra.NoArgs,
	RunE: runUICommand,
}

func runUICommand(cmd *cobra.Command, _ []string) error {
	// Check before constructing or loading the service: redirected invocations
	// must not create a configuration file as a side effect.
	if !terminalCheck() {
		return fmt.Errorf("interactive UI requires a TTY for stdin and stdout")
	}
	inline, _ := cmd.Flags().GetBool("inline")
	service := &uiService{
		manager: newConfigManager(),
		switchFn: func(ctx context.Context, manager *gitshiftconfig.Manager, alias string) error {
			adapter := &cobra.Command{}
			adapter.SetContext(ctx)
			return switchAccount(adapter, manager, alias, false, false, true)
		},
	}
	model := gitshifttui.NewWithContext(cmd.Context(), service, inline)
	program := tea.NewProgram(model, tea.WithoutSignals())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-signals:
			program.Send(gitshifttui.QuitRequestedMsg{})
		case <-done:
		}
	}()
	_, err := program.Run()
	close(done)
	signal.Stop(signals)
	return err
}

type uiService struct {
	manager  *gitshiftconfig.Manager
	switchFn func(context.Context, *gitshiftconfig.Manager, string) error
}

func (s *uiService) Load(_ context.Context) (gitshifttui.Snapshot, error) {
	if err := s.manager.LoadReadOnly(); err != nil {
		return gitshifttui.Snapshot{}, err
	}
	cfg := s.manager.GetConfig()
	result := gitshifttui.Snapshot{Current: cfg.CurrentAccount}
	for _, account := range s.manager.ListAccounts() {
		result.Accounts = append(result.Accounts, accountDTO(account, cfg.CurrentAccount))
	}
	for _, rule := range s.manager.ListAutoSwitchRules() {
		result.Rules = append(result.Rules, gitshifttui.Rule{Path: rule.Path, Account: rule.Account})
	}
	return result, nil
}

func accountDTO(a *models.Account, current string) gitshifttui.Account {
	return gitshifttui.Account{Alias: a.Alias, Name: a.Name, Email: a.Email, Username: a.GetUsername(), Platform: a.GetPlatform(), Domain: a.GetDomain(), SSHKeyPath: a.SSHKeyPath, GPGKeyID: a.GPGKeyID, Description: a.Description, Current: a.Alias == current, Default: a.IsDefault}
}

func (s *uiService) writable() error { return s.manager.Load() }

func (s *uiService) SaveAccount(_ context.Context, input gitshifttui.AccountInput) error {
	if err := input.Validate(); err != nil {
		return err
	}
	if err := s.writable(); err != nil {
		return err
	}
	sshPath, err := models.NormalizeSSHKeyPath(input.SSHKeyPath)
	if err != nil {
		return err
	}
	if input.OriginalAlias == "" {
		a := models.NewAccount(input.Alias, input.Name, input.Email, sshPath)
		a.Platform = input.Platform
		a.Domain = input.Domain
		a.SetUsername(input.Username)
		a.Description = input.Description
		return s.manager.AddAccount(a)
	}
	a, err := s.manager.GetAccount(input.OriginalAlias)
	if err != nil {
		return err
	}
	updated := *a
	updated.Name = input.Name
	updated.Email = input.Email
	updated.SSHKeyPath = sshPath
	updated.Platform = input.Platform
	updated.Domain = input.Domain
	updated.SetUsername(input.Username)
	updated.Description = input.Description
	return s.manager.UpdateAccount(&updated)
}
func (s *uiService) RemoveAccount(_ context.Context, alias string) error {
	if err := s.writable(); err != nil {
		return err
	}
	return s.manager.RemoveAccount(alias)
}
func (s *uiService) SwitchAccount(ctx context.Context, alias string) error {
	release, err := s.manager.AcquireSwitchLock()
	if err != nil {
		return err
	}
	defer release()
	if err = s.manager.Load(); err != nil {
		return err
	}
	return s.switchFn(ctx, s.manager, alias)
}
func (s *uiService) AddRule(_ context.Context, rule gitshifttui.Rule) error {
	if err := s.writable(); err != nil {
		return err
	}
	_, err := s.manager.AddAutoSwitchRule(rule.Path, rule.Account)
	return err
}
func (s *uiService) RemoveRule(_ context.Context, path string) error {
	if err := s.writable(); err != nil {
		return err
	}
	return s.manager.RemoveAutoSwitchRule(path)
}
func (s *uiService) Discover(ctx context.Context) ([]gitshifttui.Discovery, error) {
	found, err := discovery.NewSilentAccountDiscovery().ScanExistingAccountsContext(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]gitshifttui.Discovery, 0, len(found))
	for _, d := range found {
		if d == nil || d.Account == nil {
			continue
		}
		result = append(result, gitshifttui.Discovery{Identity: d.IdentityKey, Alias: d.Alias, Name: d.Name, Email: d.Email, Username: d.GetUsername(), Platform: d.GetPlatform(), Domain: d.GetDomain(), SSHKeyPath: d.SSHKeyPath, GPGKeyID: d.GPGKeyID, GPGKeyFingerprint: d.GPGKeyFingerprint, GPGKeyType: d.GPGKeyType, GPGKeySize: d.GPGKeySize, GPGEnabled: d.GPGEnabled, Source: d.Source, Confidence: d.Confidence})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Identity < result[j].Identity })
	return result, nil
}
func (s *uiService) Import(_ context.Context, selected []gitshifttui.Discovery) error {
	if len(selected) == 0 {
		return fmt.Errorf("select at least one account")
	}
	accounts := make([]*models.Account, 0, len(selected))
	seen := map[string]bool{}
	for _, d := range selected {
		if err := models.ValidateAlias(d.Alias); err != nil {
			return fmt.Errorf("invalid alias %q: %w", d.Alias, err)
		}
		key := strings.ToLower(d.Alias)
		if seen[key] {
			return fmt.Errorf("duplicate alias %q", d.Alias)
		}
		seen[key] = true
		a := models.NewAccount(d.Alias, d.Name, d.Email, d.SSHKeyPath)
		a.Platform = d.Platform
		a.Domain = d.Domain
		a.SetUsername(d.Username)
		a.Description = "Discovered from " + d.Source
		a.GPGKeyID = d.GPGKeyID
		a.GPGKeyFingerprint = d.GPGKeyFingerprint
		a.GPGKeyType = d.GPGKeyType
		a.GPGKeySize = d.GPGKeySize
		a.GPGEnabled = d.GPGEnabled
		if d.Identity != "" {
			a.AccountMetadata = map[string]string{"discovery_identity": d.Identity}
		}
		a.CreatedAt = time.Now().UTC()
		if err := a.Validate(); err != nil {
			return fmt.Errorf("cannot import %q: %w", d.Alias, err)
		}
		accounts = append(accounts, a)
	}
	if err := s.writable(); err != nil {
		return err
	}
	return s.manager.AddAccounts(accounts)
}

func init() {
	uiCmd.Flags().Bool("inline", false, "render below the current terminal content instead of using the alternate screen")
	rootCmd.AddCommand(uiCmd)
}
