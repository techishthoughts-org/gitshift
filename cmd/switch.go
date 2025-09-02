package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [alias]",
	Short: "Switch to a different GitHub account",
	Long: `Switch to a different GitHub account with intelligent behavior:

• With alias: Switch to the specified account
• No alias + 2 accounts: Automatically switch to the other account
• No alias + 1 account: Show current account (no switching needed)
• No alias + 3+ accounts: Show available options and suggest commands

The command will globally update the Git configuration (user.name and user.email)
and set up the SSH configuration for the selected account.

Examples:
  gitpersona switch work          # Switch to specific account
  gitpersona switch               # Smart switch (auto-switch between 2 accounts)
  gitpersona switch personal      # Switch to personal account`,
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
			// Smart switching logic
			selectedAlias = smartSwitchLogic(accounts, configManager.GetConfig().CurrentAccount)

			// If smart switch didn't provide a valid alias, show error
			if selectedAlias == "" || selectedAlias == configManager.GetConfig().CurrentAccount {
				fmt.Println("💡 Use 'gitpersona switch <alias>' to switch to a specific account")
				return fmt.Errorf("no account selected for switching")
			}
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

// smartSwitchLogic implements intelligent account switching
func smartSwitchLogic(accounts []*models.Account, currentAccount string) string {
	if len(accounts) == 0 {
		return ""
	}

	if len(accounts) == 1 {
		// Only one account, no need to switch
		fmt.Printf("ℹ️  Only one account configured: %s\n", accounts[0].Alias)
		fmt.Println("💡 No switching needed - you're already using the only available account")
		return accounts[0].Alias
	}

	if len(accounts) == 2 {
		// Two accounts - smart switch between them
		var otherAccount *models.Account
		for _, account := range accounts {
			if account.Alias != currentAccount {
				otherAccount = account
				break
			}
		}

		if otherAccount != nil {
			fmt.Printf("🔄 Smart switching: %s → %s\n", currentAccount, otherAccount.Alias)
			fmt.Printf("   Current: %s (%s)\n", currentAccount, getAccountSummary(accounts, currentAccount))
			fmt.Printf("   Switching to: %s (%s)\n", otherAccount.Alias, getAccountSummary(accounts, otherAccount.Alias))
			return otherAccount.Alias
		}
	}

	// More than 2 accounts - show current and suggest options
	fmt.Printf("📋 Current account: %s\n", currentAccount)
	fmt.Println("Available accounts:")
	for _, account := range accounts {
		marker := "  "
		if account.Alias == currentAccount {
			marker = "* "
		}
		fmt.Printf("%s%s (%s)\n", marker, account.Alias, getAccountSummary(accounts, account.Alias))
	}
	fmt.Printf("\n💡 Use 'gitpersona switch <alias>' to switch to a specific account\n")
	fmt.Printf("   Example: gitpersona switch %s\n", getFirstNonCurrentAccount(accounts, currentAccount))

	// For multiple accounts, we need user input
	// Return current account to avoid errors, but suggest manual selection
	return currentAccount
}

// getAccountSummary returns a brief summary of an account
func getAccountSummary(accounts []*models.Account, alias string) string {
	for _, account := range accounts {
		if account.Alias == alias {
			if account.GitHubUsername != "" {
				return fmt.Sprintf("@%s", account.GitHubUsername)
			}
			if account.Name != "" {
				return account.Name
			}
			return "unnamed"
		}
	}
	return "unknown"
}

// getFirstNonCurrentAccount returns the first account that's not the current one
func getFirstNonCurrentAccount(accounts []*models.Account, currentAccount string) string {
	for _, account := range accounts {
		if account.Alias != currentAccount {
			return account.Alias
		}
	}
	return ""
}
