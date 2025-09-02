package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
)

// completeCmd represents the complete command
var completeCmd = &cobra.Command{
	Use:   "complete [alias]",
	Short: "Complete a pending account with missing information",
	Long: `Complete a pending account by providing the missing required information.

This command converts a pending account (discovered but incomplete) into an active account.

Examples:
  gitpersona complete personal --name "John Doe" --email "john@example.com"
  gitpersona complete work --name "Work User" --email "work@company.com"`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]
		name, _ := cmd.Flags().GetString("name")
		email, _ := cmd.Flags().GetString("email")

		if name == "" || email == "" {
			return fmt.Errorf("both --name and --email are required")
		}

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if the pending account exists
		pending, err := configManager.GetPendingAccount(alias)
		if err != nil {
			return fmt.Errorf("pending account '%s' not found: %w", alias, err)
		}

		fmt.Printf("📋 Completing pending account '%s':\n", alias)
		fmt.Printf("   GitHub: @%s\n", pending.GitHubUsername)
		fmt.Printf("   Source: %s\n", pending.Source)
		fmt.Printf("   Missing fields: %v\n", pending.MissingFields)
		fmt.Printf("   Adding: Name='%s', Email='%s'\n", name, email)

		// Complete the pending account
		account, err := configManager.CompletePendingAccount(alias, name, email)
		if err != nil {
			return fmt.Errorf("failed to complete account: %w", err)
		}

		fmt.Printf("✅ Account '%s' completed successfully!\n", alias)
		fmt.Printf("   Name: %s\n", account.Name)
		fmt.Printf("   Email: %s\n", account.Email)
		fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
		if account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
		}

		// Auto-sync: Test SSH connectivity if SSH key is available
		if account.SSHKeyPath != "" {
			fmt.Println("\n🔐 Testing SSH connectivity...")
			if err := testSSHForAccount(account); err != nil {
				fmt.Printf("   ⚠️  SSH test failed: %v\n", err)
			} else {
				fmt.Printf("   ✅ SSH test passed!\n")
			}
		}

		fmt.Printf("\n💡 Account is now active and ready to use!\n")
		fmt.Printf("   Use 'gitpersona switch %s' to activate it\n", alias)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(completeCmd)

	completeCmd.Flags().StringP("name", "n", "", "Git user name (required)")
	completeCmd.Flags().StringP("email", "e", "", "Git user email (required)")
	completeCmd.MarkFlagRequired("name")
	completeCmd.MarkFlagRequired("email")
}
