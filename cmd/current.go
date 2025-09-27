package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
)

// currentCmd represents the current command
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "👤 Show the current active GitHub account",
	Long: `Display the currently active GitHub account configuration.

This command shows which account is currently active in gitshift, including
its alias, name, and email.`,
	Aliases: []string{"c", "whoami"},
	RunE:    runCurrentCommand,
}

// runCurrentCommand executes the current command
func runCurrentCommand(cmd *cobra.Command, args []string) error {
	// Get the current account alias
	alias, err := getCurrentAccount()
	if err != nil {
		return fmt.Errorf("failed to get current account: %w", err)
	}

	// Load the configuration
	configManager := config.NewManager()

	// Get the account details
	account, err := configManager.GetAccount(alias)
	if err != nil {
		return fmt.Errorf("failed to get account details: %w", err)
	}

	// Check if we should output JSON
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if jsonOutput {
		// Output in JSON format
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(account); err != nil {
			return fmt.Errorf("failed to encode account as JSON: %w", err)
		}
		return nil
	}

	// Display the current account information in a human-readable format
	fmt.Println("\n🤖 Current Active Account")
	fmt.Println("───────────────────────")
	fmt.Printf("👤 \033[1mAlias:\033[0m  %s\n", account.Alias)
	fmt.Printf("👤 \033[1mName:\033[0m   %s\n", account.Name)
	fmt.Printf("📧 \033[1mEmail:\033[0m  %s\n", account.Email)
	if account.GitHubUsername != "" {
		fmt.Printf("🐙 \033[1mGitHub:\033[0m @%s\n", account.GitHubUsername)
	}
	if account.SSHKeyPath != "" {
		fmt.Printf("🔑 \033[1mSSH Key:\033[0m %s\n", account.SSHKeyPath)
	}
	fmt.Println()

	return nil
}

func init() {
	// Add the --json flag
	currentCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	rootCmd.AddCommand(currentCmd)
}
