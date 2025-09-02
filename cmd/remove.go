package cmd

import (
	"fmt"

	"github.com/arthurcosta/GitPersona/internal/config"
	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [alias]",
	Short: "Remove a GitHub account",
	Long: `Remove a GitHub account from the configuration.

This will permanently delete the account configuration. If the account
being removed is currently active, the system will automatically switch
to another account if available.

Examples:
  gh-switcher remove work
  gh-switcher remove personal`,
	Aliases: []string{"rm", "delete", "del"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if the account exists
		account, err := configManager.GetAccount(alias)
		if err != nil {
			return fmt.Errorf("account '%s' not found", alias)
		}

		// Ask for confirmation unless --force flag is used
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to remove account '%s'? (%s - %s) [y/N]: ",
				account.Alias, account.Name, account.Email)

			confirmation := promptForInput("")
			if confirmation != "y" && confirmation != "Y" && confirmation != "yes" && confirmation != "Yes" {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		// Remove the account
		if err := configManager.RemoveAccount(alias); err != nil {
			return fmt.Errorf("failed to remove account: %w", err)
		}

		fmt.Printf("✅ Successfully removed account '%s'\n", alias)

		// Check if there are any accounts left
		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("No accounts remaining. Use 'gh-switcher add' to add a new account.")
		} else {
			currentAccount := configManager.GetConfig().CurrentAccount
			if currentAccount != "" {
				fmt.Printf("Current active account: %s\n", currentAccount)
			} else {
				fmt.Println("No active account set. Use 'gh-switcher switch' to select one.")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}
