package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// pendingCmd represents the pending command
var pendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List pending accounts that need manual completion",
	Long: `List all pending accounts that were discovered but need manual completion.

Pending accounts are accounts that were found during discovery but are missing
required information (like name or email) and couldn't be automatically imported.

Examples:
  gitpersona pending
  gitpersona pending --format json`,

	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		pendingAccounts := configManager.ListPendingAccounts()
		if len(pendingAccounts) == 0 {
			fmt.Println("✅ No pending accounts found.")
			fmt.Println("💡 All discovered accounts have been imported or completed.")
			return nil
		}

		format, _ := cmd.Flags().GetString("format")

		switch format {
		case "json":
			return printPendingAccountsJSON(pendingAccounts)
		case "table":
			return printPendingAccountsTable(pendingAccounts)
		default:
			return printPendingAccountsDefault(pendingAccounts)
		}
	},
}

func printPendingAccountsDefault(accounts []*models.PendingAccount) error {
	fmt.Printf("📋 Found %d pending account(s) that need completion:\n\n", len(accounts))

	for i, account := range accounts {
		fmt.Printf("📋 Pending Account %d:\n", i+1)
		fmt.Printf("   Alias: %s\n", account.Alias)
		if account.GitHubUsername != "" {
			fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
		}
		fmt.Printf("   Source: %s\n", account.Source)
		fmt.Printf("   Confidence: %d/10\n", account.Confidence)
		fmt.Printf("   Missing: %v\n", account.MissingFields)
		fmt.Printf("   Created: %s\n", formatTime(account.CreatedAt))

		if len(account.PartialData) > 0 {
			fmt.Printf("   Partial Data:\n")
			for key, value := range account.PartialData {
				fmt.Printf("     %s: %s\n", key, value)
			}
		}

		fmt.Printf("   💡 Complete with: gitpersona complete %s --name \"Your Name\" --email \"your@email.com\"\n", account.Alias)
		fmt.Println()
	}

	return nil
}

func printPendingAccountsTable(accounts []*models.PendingAccount) error {
	// Calculate column widths
	aliasWidth := len("ALIAS")
	githubWidth := len("GITHUB")
	sourceWidth := len("SOURCE")
	confidenceWidth := len("CONF")
	missingWidth := len("MISSING")

	for _, account := range accounts {
		if len(account.Alias) > aliasWidth {
			aliasWidth = len(account.Alias)
		}
		if len(account.GitHubUsername) > githubWidth {
			githubWidth = len(account.GitHubUsername)
		}
		if len(account.Source) > sourceWidth {
			sourceWidth = len(account.Source)
		}
		if len(fmt.Sprintf("%d", account.Confidence)) > confidenceWidth {
			confidenceWidth = len(fmt.Sprintf("%d", account.Confidence))
		}
		missingStr := fmt.Sprintf("%v", account.MissingFields)
		if len(missingStr) > missingWidth {
			missingWidth = len(missingStr)
		}
	}

	// Print header
	fmt.Printf("%-*s | %-*s | %-*s | %-*s | %-*s\n",
		aliasWidth, "ALIAS",
		githubWidth, "GITHUB",
		sourceWidth, "SOURCE",
		confidenceWidth, "CONF",
		missingWidth, "MISSING")
	fmt.Println(strings.Repeat("-", aliasWidth+githubWidth+sourceWidth+confidenceWidth+missingWidth+8))

	// Print accounts
	for _, account := range accounts {
		fmt.Printf("%-*s | %-*s | %-*s | %-*s | %-*s\n",
			aliasWidth, account.Alias,
			githubWidth, account.GitHubUsername,
			sourceWidth, account.Source,
			confidenceWidth, fmt.Sprintf("%d", account.Confidence),
			missingWidth, fmt.Sprintf("%v", account.MissingFields))
	}

	return nil
}

func printPendingAccountsJSON(accounts []*models.PendingAccount) error {
	// Simple JSON output
	fmt.Println("[")
	for i, account := range accounts {
		if i > 0 {
			fmt.Println(",")
		}
		fmt.Printf("  {\n")
		fmt.Printf("    \"alias\": \"%s\",\n", account.Alias)
		fmt.Printf("    \"github_username\": \"%s\",\n", account.GitHubUsername)
		fmt.Printf("    \"source\": \"%s\",\n", account.Source)
		fmt.Printf("    \"confidence\": %d,\n", account.Confidence)
		fmt.Printf("    \"missing_fields\": %v,\n", account.MissingFields)
		fmt.Printf("    \"created_at\": \"%s\"\n", account.CreatedAt.Format(time.RFC3339))
		fmt.Printf("  }")
	}
	fmt.Println("\n]")
	return nil
}

func init() {
	rootCmd.AddCommand(pendingCmd)

	pendingCmd.Flags().StringP("format", "f", "default", "Output format (default, table, json)")
}
