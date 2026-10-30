package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	gitshiftconfig "github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/gitidentity"
	"github.com/techishthoughts/gitshift/internal/models"
	"github.com/techishthoughts/gitshift/internal/ssh"
	"github.com/techishthoughts/gitshift/pkg/platform"
)

var autoGitIncludesCmd = &cobra.Command{
	Use:   "git-includes [on|off|status]",
	Short: "Let Git pick the identity per rule directory with includeIf",
	Long: `Let Git choose the identity for each auto-switch rule directory by itself.

'on' writes one include file per account under the GitShift config directory
(git/<alias>.gitconfig) with the account's name, email, Git SSH command and
commit signing, and adds an includeIf "gitdir:<rule dir>/" entry for every rule
at the end of your global Git config. Every terminal then uses the right
identity for its directory at the same time, without switching. Entries are
kept in sync whenever accounts or rules change.

'off' removes the entries and files. 'status' (the default) shows them.

Switching still sets the GitHub CLI account, which Git cannot choose per
directory. gitshift's Host block in ~/.ssh/config is removed, since it would
offer the global account's key ahead of each directory's.`,
	Example: `  gitshift auto git-includes on
  gitshift auto git-includes status
  gitshift auto git-includes off`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"on", "off", "status"},
	RunE: func(cmd *cobra.Command, args []string) error {
		action := "status"
		if len(args) == 1 {
			action = args[0]
		}
		manager := newConfigManager()
		out := cmd.OutOrStdout()
		switch action {
		case "on":
			if err := manager.Load(); err != nil {
				return fmt.Errorf("load configuration: %w", err)
			}
			if err := manager.SetGitIncludes(true); err != nil {
				return fmt.Errorf("enable Git includes: %w", err)
			}
			// Sync again so a failure is reported as an error, not a warning.
			if err := gitidentity.Sync(manager.GetConfig(), gitIncludeDir(manager)); err != nil {
				return fmt.Errorf("write Git includes: %w", err)
			}
			if err := removeManagedSSHHosts(manager.GetConfig()); err != nil {
				return fmt.Errorf("remove managed SSH host: %w", err)
			}
			fmt.Fprintln(out, "Git includes on: Git now picks the identity per rule directory.")
		case "off":
			if err := manager.Load(); err != nil {
				return fmt.Errorf("load configuration: %w", err)
			}
			if err := manager.SetGitIncludes(false); err != nil {
				return fmt.Errorf("disable Git includes: %w", err)
			}
			if err := gitidentity.Remove(gitIncludeDir(manager)); err != nil {
				return fmt.Errorf("remove Git includes: %w", err)
			}
			fmt.Fprintln(out, "Git includes off: managed includeIf entries and files removed.")
			return nil
		case "status":
			if err := manager.LoadReadOnly(); err != nil {
				return fmt.Errorf("load configuration: %w", err)
			}
		default:
			return fmt.Errorf("unknown action %q; use on, off or status", action)
		}
		return printGitIncludes(cmd, manager)
	},
}

func printGitIncludes(cmd *cobra.Command, manager *gitshiftconfig.Manager) error {
	out := cmd.OutOrStdout()
	enabled := manager.GetConfig().AutoSwitch.GitIncludes
	installed, err := gitidentity.Installed(gitIncludeDir(manager))
	if err != nil {
		return err
	}
	state := "off"
	if enabled {
		state = "on"
	}
	fmt.Fprintf(out, "Git includes: %s\n", state)
	for _, include := range installed {
		fmt.Fprintf(out, "%s -> %s (%s)\n", include.Dir, include.Account, include.File)
	}
	if enabled {
		inSync, err := gitidentity.InSync(manager.GetConfig(), gitIncludeDir(manager))
		if err != nil {
			return err
		}
		if !inSync {
			fmt.Fprintln(out, "Out of sync with the rules, or other settings follow the includes; run 'gitshift auto git-includes on'.")
		}
	}
	if !enabled && len(installed) > 0 {
		fmt.Fprintln(out, "Entries remain from an earlier setup; run 'gitshift auto git-includes off' to remove them.")
	}
	return nil
}

// removeManagedSSHHosts drops gitshift's Host block for every account's host.
func removeManagedSSHHosts(cfg *models.Config) error {
	manager := ssh.NewManager()
	for _, account := range cfg.Accounts {
		provider, err := platform.NewFactory().GetPlatform(platform.Type(account.GetPlatform()), account.GetDomain())
		if err != nil {
			return err
		}
		if err := manager.RemoveSSHConfig(provider.GetSSHHost()); err != nil {
			return err
		}
	}
	return nil
}

func gitIncludeDir(manager *gitshiftconfig.Manager) string {
	return filepath.Join(manager.ConfigDir(), "git")
}

func init() {
	autoCmd.AddCommand(autoGitIncludesCmd)
}
