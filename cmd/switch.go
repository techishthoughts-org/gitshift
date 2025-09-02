package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/tui"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [alias]",
	Short: "Switch to a different GitHub account",
	Long: `Switch to a different GitHub account. If no alias is provided,
an interactive TUI will be shown to select an account.

The command will globally update the Git configuration (user.name and user.email)
and set up the SSH configuration for the selected account.

Examples:
  gitpersona switch work
  gitpersona switch personal
  gitpersona switch (opens interactive TUI)`,
	Aliases: []string{"s", "use"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts configured. Use 'gitpersona add' to add an account")
		}

		var selectedAlias string

		// If alias provided as argument, use it
		if len(args) > 0 {
			selectedAlias = args[0]
		} else {
			// Launch TUI to select account
			selectedAccount, err := tui.SelectAccount(accounts, configManager.GetConfig().CurrentAccount)
			if err != nil {
				return fmt.Errorf("failed to select account: %w", err)
			}
			if selectedAccount == nil {
				return fmt.Errorf("no account selected")
			}
			selectedAlias = selectedAccount.Alias
		}

		// Get the account
		account, err := configManager.GetAccount(selectedAlias)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", selectedAlias, err)
		}

		// Switch to the account (always global)
		if err := switchToAccount(configManager, account); err != nil {
			return fmt.Errorf("failed to switch to account '%s': %w", selectedAlias, err)
		}

		fmt.Printf("✅ Switched to account '%s'\n", selectedAlias)
		fmt.Printf("   Name: %s\n", account.Name)
		fmt.Printf("   Email: %s\n", account.Email)

		if account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
			fmt.Printf("\n💡 To use this SSH key, run:\n")
			fmt.Printf("   export GIT_SSH_COMMAND=\"ssh -i %s -o IdentitiesOnly=yes\"\n", account.SSHKeyPath)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

// switchToAccount handles the actual account switching logic (always global)
func switchToAccount(configManager *config.Manager, account *models.Account) error {
	gitManager := git.NewManager()

	// Always set global Git configuration
	if err := gitManager.SetGlobalConfig(account); err != nil {
		return fmt.Errorf("failed to set global git config: %w", err)
	}

	// Update current account in config
	if err := configManager.SetCurrentAccount(account.Alias); err != nil {
		return fmt.Errorf("failed to update current account: %w", err)
	}

	return nil
}
