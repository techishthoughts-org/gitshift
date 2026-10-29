package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/gitidentity"
	"github.com/techishthoughts/gitshift/internal/models"
	"github.com/techishthoughts/gitshift/internal/ssh"
	"github.com/techishthoughts/gitshift/pkg/platform"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [account-alias]",
	Short: "🔄 Switch to a GitHub or GitLab account",
	Long: `Switch to a different Git platform account with comprehensive configuration.

This command will:
- Switch SSH configuration and keys
- Update global Git identity and the current repository when applicable
- Update optional GPG signing configuration
- Select the matching GitHub CLI account for GitHub profiles
- Validate the resulting configuration and connection

Works with all supported platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)

Examples:
  # Switch to GitHub account
  gitshift switch work-github
  gitshift switch personal-github --force

  # Switch to GitLab account
  gitshift switch work-gitlab
  gitshift switch personal-gitlab --validate

  # Switch to self-hosted GitLab
  gitshift switch company-gitlab

  # Switch to GitHub Enterprise
  gitshift switch enterprise`,
	Aliases: []string{"s", "use"},
	Args:    cobra.ExactArgs(1),
	RunE:    runSwitchCommand,
}

// runSwitchCommand executes the switch command
func runSwitchCommand(cmd *cobra.Command, args []string) error {
	accountAlias := args[0]
	validateOnly, _ := cmd.Flags().GetBool("validate")
	force, _ := cmd.Flags().GetBool("force")
	skipValidation, _ := cmd.Flags().GetBool("skip-validation")

	configManager := newConfigManager()
	release, err := configManager.AcquireSwitchLock()
	if err != nil {
		return fmt.Errorf("failed to acquire switch transaction: %w", err)
	}
	defer release()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if validateOnly {
		return validateAccount(configManager, accountAlias)
	}
	return switchAccount(cmd, configManager, accountAlias, force, skipValidation, false)
}

// switchAccount is the single transaction implementation used by manual and
// directory-triggered switching. The caller must hold AcquireSwitchLock.
func switchAccount(cmd *cobra.Command, configManager *config.Manager, accountAlias string, force, skipValidation, quiet bool) error {
	outputf := func(format string, args ...any) {
		if !quiet {
			fmt.Printf(format, args...)
		}
	}
	targetAccount, err := configManager.GetAccount(accountAlias)
	if err != nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}
	if configManager.GetConfig().AutoSwitch.GitIncludes {
		// Switching and its rollback rewrite global keys, which can append
		// sections after the includes and override them. Re-sync on every
		// outcome so the includes stay last.
		defer func() {
			if err := gitidentity.Sync(configManager.GetConfig(), gitIncludeDir(configManager)); err != nil {
				fmt.Fprintf(os.Stderr, "warning: Git includes not updated: %v; run 'gitshift auto git-includes on'\n", err)
			}
		}()
	}
	if err := targetAccount.Validate(); err != nil {
		return fmt.Errorf("account preflight failed: %w", err)
	}
	if targetAccount.SSHKeyPath != "" {
		targetAccount.SSHKeyPath, err = models.NormalizeSSHKeyPath(targetAccount.SSHKeyPath)
		if err != nil {
			return fmt.Errorf("account preflight failed: %w", err)
		}
		if err := ssh.CheckPrivateKeyFile(targetAccount.SSHKeyPath); err != nil {
			return fmt.Errorf("account preflight failed: %w", err)
		}
	}
	if err := checkCommitSigning(targetAccount); err != nil && !force {
		return fmt.Errorf("signing preflight failed: %w", err)
	}
	provider, err := platform.NewFactory().GetPlatform(platform.Type(targetAccount.GetPlatform()), targetAccount.GetDomain())
	if err != nil {
		return fmt.Errorf("platform preflight failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()
	if targetAccount.SSHKeyPath != "" && !skipValidation {
		if err := ssh.NewManager().TestConnectionToPlatformContext(ctx, provider.GetSSHHost(), targetAccount.SSHKeyPath); err != nil {
			if !force {
				return fmt.Errorf("SSH preflight failed: %w", err)
			}
			outputf("⚠️  SSH validation failed; continuing due to --force: %v\n", err)
		}
	}

	previousState, err := captureSwitchState(targetAccount, force)
	if err != nil {
		return fmt.Errorf("switch preflight failed: %w", err)
	}
	rollback := func(cause error) error {
		if rollbackErr := previousState.restore(); rollbackErr != nil {
			return fmt.Errorf("%w (rollback also failed: %v)", cause, rollbackErr)
		}
		return cause
	}

	outputf("🔄 Switching to account '%s'...\n", accountAlias)
	outputf("   Name: %s\n", targetAccount.Name)
	outputf("   Email: %s\n", targetAccount.Email)

	if targetAccount.SSHKeyPath != "" {
		outputf("🔑 Updating SSH configuration for %s...\n", provider.GetSSHHost())
		if err := ssh.NewManager().SwitchToAccountContext(ctx, accountAlias, targetAccount.SSHKeyPath, provider.GetSSHHost(), false); err != nil {
			return rollback(fmt.Errorf("SSH switch failed: %w", err))
		}
		outputf("✅ SSH configuration updated; shared agent and shell startup files unchanged\n")
	} else {
		outputf("ℹ️  No SSH key configured for this account\n")
		outputf("   Consider running: gitshift ssh-keygen %s\n", accountAlias)
	}

	outputf("🔧 Updating Git configuration...\n")
	if err := updateGitConfig(targetAccount, !configManager.GetConfig().AutoSwitch.GitIncludes); err != nil {
		return rollback(fmt.Errorf("failed to update Git configuration: %w", err))
	} else {
		outputf("✅ Git configuration updated\n")
	}

	signingFormat, _ := targetAccount.CommitSigning()
	if err := applyCommitSigning(targetAccount); err != nil {
		if force {
			outputf("⚠️  Commit signing update failed: %v (continuing due to --force)\n", err)
		} else {
			return rollback(fmt.Errorf("commit signing update failed: %w", err))
		}
	} else {
		switch signingFormat {
		case models.SigningFormatSSH, models.SigningFormatGPG:
			outputf("🔐 Commit and tag signing enabled (%s)\n", signingFormat)
		default:
			outputf("🔓 Commit signing off for this account\n")
			outputf("   To enable: gitshift update %s --signing ssh\n", accountAlias)
		}
	}

	if targetAccount.GetPlatform() == "github" {
		outputf("🔐 Switching GitHub CLI authentication...\n")
		if err := switchGitHubCLI(targetAccount.GetDomain(), targetAccount.GetUsername()); err != nil {
			if force {
				outputf("⚠️  GitHub CLI switch failed: %v (continuing due to --force)\n", err)
			} else {
				return rollback(err)
			}
		} else {
			previousState.ghChanged = true
			outputf("✅ GitHub CLI authentication updated\n")
		}
	}

	if !skipValidation && !force {
		outputf("🧪 Testing configuration...\n")
		if err := testConfiguration(targetAccount); err != nil {
			return rollback(fmt.Errorf("configuration validation failed: %w", err))
		}
		outputf("✅ Configuration test passed\n")
	}

	// Persist last so failed side effects never advertise the target as current.
	if err := configManager.SetCurrentAccount(accountAlias); err != nil {
		return rollback(fmt.Errorf("failed to set current account: %w", err))
	}
	outputf("✅ gitshift configuration updated\n")

	outputf("\n🎉 Successfully switched to account '%s'!\n", accountAlias)
	outputf("   You can now use Git with the %s account configuration\n", accountAlias)

	return nil
}

type configValueSnapshot struct {
	scope  string
	key    string
	values []string
}

type switchState struct {
	gitValues  []configValueSnapshot
	sshPath    string
	sshData    []byte
	sshMode    os.FileMode
	sshExisted bool
	ghDomain   string
	ghUser     string
	ghChanged  bool
}

func captureSwitchState(target *models.Account, allowMissingGitHubState bool) (*switchState, error) {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	state := &switchState{sshPath: filepath.Join(home, ".ssh", "config")}
	if info, err := os.Stat(state.sshPath); err == nil {
		state.sshExisted = true
		state.sshMode = info.Mode().Perm()
		state.sshData, err = os.ReadFile(state.sshPath)
		if err != nil {
			return nil, fmt.Errorf("snapshot SSH config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect SSH config: %w", err)
	}

	keys := []string{"user.name", "user.email", "core.sshCommand", "user.signingkey", "commit.gpgsign", "tag.gpgsign", "gpg.format"}
	scopes := []string{"--global"}
	if isGitRepo() {
		scopes = append(scopes, "--local")
	}
	for _, scope := range scopes {
		for _, key := range keys {
			values, err := readGitConfigValues(scope, key)
			if err != nil {
				return nil, fmt.Errorf("snapshot git config %s: %w", key, err)
			}
			state.gitValues = append(state.gitValues, configValueSnapshot{scope: scope, key: key, values: values})
		}
	}

	if target.GetPlatform() == "github" {
		if _, err := exec.LookPath("gh"); err != nil {
			if allowMissingGitHubState {
				return state, nil
			}
			return nil, fmt.Errorf("GitHub CLI not found")
		}
		cmd := exec.Command("gh", "auth", "status", "--active", "--hostname", target.GetDomain(), "--json", "hosts", "--jq", ".hosts | add | .[0].login // empty")
		output, err := cmd.Output()
		if err != nil {
			if allowMissingGitHubState {
				return state, nil
			}
			return nil, fmt.Errorf("determine current GitHub CLI account: %w", err)
		}
		state.ghDomain = target.GetDomain()
		state.ghUser = strings.TrimSpace(string(output))
		if state.ghUser == "" {
			if allowMissingGitHubState {
				state.ghDomain = ""
				return state, nil
			}
			return nil, fmt.Errorf("GitHub CLI returned an empty active username")
		}
	}

	return state, nil
}

func readGitConfigValues(scope, key string) ([]string, error) {
	output, err := exec.Command("git", "config", scope, "--get-all", key).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	text := strings.TrimSuffix(string(output), "\n")
	if text == "" {
		return []string{""}, nil
	}
	return strings.Split(text, "\n"), nil
}

func (s *switchState) restore() error {
	var restoreErrors []string
	if s.ghChanged && s.ghUser != "" {
		if output, err := exec.Command("gh", "auth", "switch", "--hostname", s.ghDomain, "--user", s.ghUser).CombinedOutput(); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("GitHub CLI: %v (%s)", err, strings.TrimSpace(string(output))))
		}
	}
	for i := len(s.gitValues) - 1; i >= 0; i-- {
		snapshot := s.gitValues[i]
		_ = exec.Command("git", "config", snapshot.scope, "--unset-all", snapshot.key).Run()
		for _, value := range snapshot.values {
			if err := exec.Command("git", "config", snapshot.scope, "--add", snapshot.key, value).Run(); err != nil {
				restoreErrors = append(restoreErrors, fmt.Sprintf("git %s: %v", snapshot.key, err))
			}
		}
	}
	if s.sshExisted {
		if err := os.WriteFile(s.sshPath, s.sshData, s.sshMode); err != nil {
			restoreErrors = append(restoreErrors, fmt.Sprintf("SSH config: %v", err))
		}
	} else if err := os.Remove(s.sshPath); err != nil && !os.IsNotExist(err) {
		restoreErrors = append(restoreErrors, fmt.Sprintf("SSH config: %v", err))
	}
	if len(restoreErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(restoreErrors, "; "))
	}
	return nil
}

// validateAccount validates an account configuration
func validateAccount(configManager *config.Manager, accountAlias string) error {
	accounts := configManager.ListAccounts()
	var targetAccount *models.Account
	for _, account := range accounts {
		if account.Alias == accountAlias {
			targetAccount = account
			break
		}
	}

	if targetAccount == nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}

	fmt.Printf("🔍 Validating account '%s'...\n", accountAlias)

	issues := 0

	// Check basic configuration
	if targetAccount.Name == "" {
		fmt.Printf("❌ Missing display name\n")
		issues++
	} else {
		fmt.Printf("✅ Display name: %s\n", targetAccount.Name)
	}

	if targetAccount.Email == "" {
		fmt.Printf("❌ Missing email address\n")
		issues++
	} else {
		fmt.Printf("✅ Email: %s\n", targetAccount.Email)
	}

	// Check SSH key
	if targetAccount.SSHKeyPath == "" {
		fmt.Printf("⚠️  No SSH key configured\n")
	} else {
		keyPath, normalizeErr := models.NormalizeSSHKeyPath(targetAccount.SSHKeyPath)
		if normalizeErr != nil {
			fmt.Printf("❌ Invalid SSH key path: %v\n", normalizeErr)
			issues++
		} else if err := ssh.CheckPrivateKeyFile(keyPath); err != nil {
			fmt.Printf("❌ %v\n", err)
			issues++
		} else {
			fmt.Printf("✅ SSH key found: %s\n", keyPath)

			// Test SSH connection
			sshManager := ssh.NewManager()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := sshManager.TestConnectionToPlatformContext(ctx, targetAccount.GetDomain(), keyPath); err != nil {
				fmt.Printf("❌ SSH connection test failed: %v\n", err)
				issues++
			} else {
				fmt.Printf("✅ SSH connection test passed\n")
			}
		}
	}

	if targetAccount.GetPlatform() == "github" {
		if err := checkGitHubCLIAccount(targetAccount.GetDomain(), targetAccount.GetUsername()); err != nil {
			fmt.Printf("❌ %v\n", err)
			issues++
		} else {
			fmt.Printf("✅ GitHub CLI logged in as @%s on %s\n", targetAccount.GetUsername(), targetAccount.GetDomain())
		}
	}

	if format, key := targetAccount.CommitSigning(); format != "" {
		if err := checkCommitSigning(targetAccount); err != nil {
			fmt.Printf("❌ %v\n", err)
			issues++
		} else {
			fmt.Printf("✅ Commit signing (%s) key available: %s\n", format, key)
		}
	}

	if issues == 0 {
		fmt.Printf("\n✅ Account '%s' is valid and ready to use!\n", accountAlias)
	} else {
		fmt.Printf("\n❌ Account '%s' has %d issue(s) that need to be resolved\n", accountAlias, issues)
		return fmt.Errorf("account validation failed")
	}

	return nil
}

// isGitRepo checks if the current directory is a Git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// updateGitConfig updates the Git user configuration (both global and local if in a repo)
// updateGitConfig sets the global identity and, unless Git includes choose the
// identity per directory, the current repository's local identity too.
func updateGitConfig(account *models.Account, writeLocal bool) error {
	// Set global configuration
	if account.Name != "" {
		cmd := exec.Command("git", "config", "--global", "user.name", account.Name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git user.name: %w", err)
		}
	}

	if account.Email != "" {
		cmd := exec.Command("git", "config", "--global", "user.email", account.Email)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git user.email: %w", err)
		}
	}

	// Set SSH command to use the account's SSH key for proper isolation
	if account.SSHKeyPath != "" {
		sshCommand := isolatedGitSSHCommand(account.SSHKeyPath)
		cmd := exec.Command("git", "config", "--global", "core.sshCommand", sshCommand)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git core.sshCommand: %w", err)
		}
	} else if err := clearGitConfigValue("--global", "core.sshCommand"); err != nil {
		return fmt.Errorf("failed to clear global git core.sshCommand: %w", err)
	}

	// Check if we're in a Git repository and set local config too. Local values
	// would override Git includes, so they are skipped when includes are on.
	if writeLocal && isGitRepo() {
		if account.Name != "" {
			cmd := exec.Command("git", "config", "--local", "user.name", account.Name)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git user.name: %w", err)
			}
		}

		if account.Email != "" {
			cmd := exec.Command("git", "config", "--local", "user.email", account.Email)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git user.email: %w", err)
			}
		}

		// Also set SSH command locally for better isolation
		if account.SSHKeyPath != "" {
			sshCommand := isolatedGitSSHCommand(account.SSHKeyPath)
			cmd := exec.Command("git", "config", "--local", "core.sshCommand", sshCommand)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git core.sshCommand: %w", err)
			}
		} else if err := clearGitConfigValue("--local", "core.sshCommand"); err != nil {
			return fmt.Errorf("failed to clear local git core.sshCommand: %w", err)
		}
	}

	return nil
}

// isolatedGitSSHCommand pins Git to the account key while keeping the user's
// SSH config (ports, ProxyJump, agents holding this key). IdentitiesOnly stops
// ssh from offering unrelated agent keys first.
func isolatedGitSSHCommand(keyPath string) string {
	return gitidentity.SSHCommand(keyPath)
}

func clearGitConfigValue(scope, key string) error {
	check := exec.Command("git", "config", scope, "--get-all", key)
	if err := check.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return err
	}
	return exec.Command("git", "config", scope, "--unset-all", key).Run()
}

// switchGitHubCLI switches the GitHub CLI authentication
// checkGitHubCLIAccount reports whether gh holds a working login for username
// on domain, without switching the active account.
func checkGitHubCLIAccount(domain, username string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) not found")
	}
	if username == "" {
		return fmt.Errorf("account has no username for GitHub CLI")
	}
	query := fmt.Sprintf(`[.hosts[]?[]? | select(.login == %s) | .state] | first // empty`, strconv.Quote(username))
	output, err := exec.Command("gh", "auth", "status", "--hostname", domain, "--json", "hosts", "--jq", query).Output()
	state := strings.TrimSpace(string(output))
	switch {
	case state == "success":
		return nil
	case state == "" && err == nil:
		return fmt.Errorf("GitHub CLI is not logged in as @%s on %s; run 'gh auth login --hostname %s'", username, domain, domain)
	case state == "":
		return fmt.Errorf("cannot read GitHub CLI status for %s: %w", domain, err)
	default:
		return fmt.Errorf("GitHub CLI login for @%s on %s is %s; run 'gh auth refresh --hostname %s'", username, domain, state, domain)
	}
}

func switchGitHubCLI(domain, username string) error {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI not found")
	}

	// Try to switch to the account
	cmd := exec.Command("gh", "auth", "switch", "--hostname", domain, "--user", username)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the account doesn't exist in gh auth, that's OK
		if strings.Contains(string(output), "not found") {
			return fmt.Errorf("account '%s' not found in GitHub CLI for %s - run 'gh auth login --hostname %s'", username, domain, domain)
		}
		return fmt.Errorf("gh auth switch failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// testConfiguration tests the current configuration
func testConfiguration(account *models.Account) error {
	// Test Git configuration
	nameCmd := exec.Command("git", "config", "--global", "user.name")
	nameOutput, err := nameCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.name: %w", err)
	}

	emailCmd := exec.Command("git", "config", "--global", "user.email")
	emailOutput, err := emailCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.email: %w", err)
	}

	gitName := strings.TrimSpace(string(nameOutput))
	gitEmail := strings.TrimSpace(string(emailOutput))

	if account.Name != "" && gitName != account.Name {
		return fmt.Errorf("git user.name mismatch: expected '%s', got '%s'", account.Name, gitName)
	}

	if account.Email != "" && gitEmail != account.Email {
		return fmt.Errorf("git user.email mismatch: expected '%s', got '%s'", account.Email, gitEmail)
	}

	// Test SSH if key is configured
	if account.SSHKeyPath != "" {
		if _, err := os.Stat(account.SSHKeyPath); err == nil {
			sshManager := ssh.NewManager()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := sshManager.TestConnectionToPlatformContext(ctx, account.GetDomain(), account.SSHKeyPath); err != nil {
				return fmt.Errorf("SSH connection test failed: %w", err)
			}
		}
	}

	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// getCurrentAccount returns the currently active account alias, or an error if not found
func getCurrentAccount() (string, error) {
	configManager := newConfigManager()
	if err := configManager.Load(); err != nil {
		return "", fmt.Errorf("failed to load configuration: %w", err)
	}

	currentAccount := configManager.GetConfig().CurrentAccount
	if currentAccount == "" {
		return "", fmt.Errorf("no active account set")
	}

	return currentAccount, nil
}

// applyCommitSigning sets Git's global signing settings for the account:
// gpg.format, user.signingkey, commit.gpgsign and tag.gpgsign.
func applyCommitSigning(account *models.Account) error {
	set, unset, err := gitidentity.SigningSettings(account)
	if err != nil {
		return err
	}
	for _, setting := range set {
		if output, err := exec.Command("git", "config", "--global", setting.Key, setting.Value).CombinedOutput(); err != nil {
			return fmt.Errorf("set %s: %w: %s", setting.Key, err, strings.TrimSpace(string(output)))
		}
	}
	for _, name := range unset {
		if err := clearGitConfigValue("--global", name); err != nil {
			return fmt.Errorf("unset %s: %w", name, err)
		}
	}
	return nil
}

// checkCommitSigning reports problems that would make signed commits fail.
func checkCommitSigning(account *models.Account) error {
	format, key := account.CommitSigning()
	switch format {
	case models.SigningFormatSSH:
		keyPath, err := models.NormalizeSSHKeyPath(key)
		if err != nil {
			return fmt.Errorf("SSH signing key: %w", err)
		}
		if strings.HasSuffix(keyPath, ".pub") {
			if _, err := os.Stat(keyPath); err != nil {
				return fmt.Errorf("SSH signing key not found at %s: %w", keyPath, err)
			}
		} else if err := ssh.CheckPrivateKeyFile(keyPath); err != nil {
			return fmt.Errorf("SSH signing key: %w", err)
		}
		return checkGitSupportsSSHSigning()
	case models.SigningFormatGPG:
		if err := exec.Command("gpg", "--batch", "--list-secret-keys", "--", key).Run(); err != nil {
			return fmt.Errorf("GPG secret key %s not available: %w", key, err)
		}
	}
	return nil
}

// checkGitSupportsSSHSigning requires Git 2.34, the first release with
// gpg.format=ssh.
func checkGitSupportsSSHSigning() error {
	output, err := exec.Command("git", "version").Output()
	if err != nil {
		return fmt.Errorf("read git version: %w", err)
	}
	if !gitVersionAtLeast(string(output), 2, 34) {
		return fmt.Errorf("SSH commit signing needs Git 2.34 or later; found %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func gitVersionAtLeast(output string, major, minor int) bool {
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return false
	}
	parts := strings.SplitN(fields[2], ".", 3)
	if len(parts) < 2 {
		return false
	}
	gotMajor, errMajor := strconv.Atoi(parts[0])
	gotMinor, errMinor := strconv.Atoi(parts[1])
	if errMajor != nil || errMinor != nil {
		return false
	}
	return gotMajor > major || (gotMajor == major && gotMinor >= minor)
}

func init() {
	switchCmd.Flags().BoolP("validate", "V", false, "Only validate the account without switching")
	switchCmd.Flags().BoolP("force", "f", false, "Force switch even if validation fails")
	switchCmd.Flags().BoolP("skip-validation", "s", false, "Skip SSH validation (not recommended)")

	rootCmd.AddCommand(switchCmd)
}
