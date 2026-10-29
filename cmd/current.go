package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// currentCmd represents the current command
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "👤 Show the current active Git account",
	Long: `Display the currently active Git platform account configuration.

This command shows which account is currently active in gitshift, including
its alias, name, email, and platform.

Works with all supported platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)`,
	Aliases: []string{"c", "whoami"},
	RunE:    runCurrentCommand,
}

// runCurrentCommand executes the current command
func runCurrentCommand(cmd *cobra.Command, args []string) error {
	configManager := newConfigManager()
	// --short is read by shell prompts on every render: read-only, alias only,
	// and no error when nothing has been applied yet.
	if short, _ := cmd.Flags().GetBool("short"); short {
		if err := configManager.LoadReadOnly(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if alias := configManager.GetConfig().CurrentAccount; alias != "" {
			fmt.Fprintln(cmd.OutOrStdout(), alias)
		}
		return nil
	}
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	account, err := configManager.GetCurrentAccount()
	if err != nil {
		return fmt.Errorf("failed to get current account: %w", err)
	}

	// Check if we should output JSON
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if jsonOutput {
		// Output in JSON format
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		entry := newAccountJSON(account)
		entry.SSHKeyPath = ""
		entry.GPGKeyFingerprint = ""
		if err := encoder.Encode(entry); err != nil {
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
	currentCmd.Flags().BoolP("short", "s", false, "Print only the active alias (for shell prompts)")
	rootCmd.AddCommand(currentCmd)
}
