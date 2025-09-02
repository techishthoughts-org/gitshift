package cmd

import (
	"fmt"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/discovery"
	"github.com/spf13/cobra"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Auto-discover existing Git accounts on your system",
	Long: `Automatically discover and import existing Git accounts from:

- Global ~/.gitconfig
- Git config files in ~/.config/git/
- SSH keys configured for GitHub
- GitHub CLI (gh) authentication

Examples:
  gitpersona discover
  gitpersona discover --auto-import`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if accounts already exist
		existingAccounts := configManager.ListAccounts()
		if len(existingAccounts) > 0 {
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				fmt.Printf("⚠️  You already have %d account(s) configured.\n", len(existingAccounts))
				fmt.Println("Use --overwrite to replace existing accounts.")
				fmt.Println("\nExisting accounts:")
				for _, acc := range existingAccounts {
					fmt.Printf("  - %s (%s - %s)\n", acc.Alias, acc.Name, acc.Email)
				}
				return nil
			}
		}

		// Discover accounts
		discovery := discovery.NewAccountDiscovery()
		fmt.Println("🔍 Scanning system for existing Git accounts...")

		discovered, err := discovery.ScanExistingAccounts()
		if err != nil {
			return fmt.Errorf("failed to discover accounts: %w", err)
		}

		if len(discovered) == 0 {
			fmt.Println("❌ No existing Git accounts found on your system.")
			fmt.Println("💡 Use 'gitpersona add-github username' for automatic setup!")
			return nil
		}

		fmt.Printf("✅ Found %d potential account(s):\n\n", len(discovered))

		imported := 0
		for i, account := range discovered {
			fmt.Printf("📋 Account %d:\n", i+1)
			fmt.Printf("   Alias: %s\n", account.Alias)
			if account.Name != "" {
				fmt.Printf("   Name: %s\n", account.Name)
			}
			if account.Email != "" {
				fmt.Printf("   Email: %s\n", account.Email)
			}
			if account.GitHubUsername != "" {
				fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
			}
			fmt.Printf("   Source: %s\n", account.Source)
			fmt.Printf("   Confidence: %d/10\n", account.Confidence)

			autoImport, _ := cmd.Flags().GetBool("auto-import")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if !dryRun && (autoImport || account.Confidence >= 8) && account.Name != "" && account.Email != "" {
				if err := configManager.AddAccount(account.Account); err != nil {
					fmt.Printf("   ❌ Failed to import: %v\n", err)
				} else {
					fmt.Printf("   ✅ Imported successfully!\n")
					imported++
				}
			} else if dryRun {
				fmt.Printf("   🔍 Would import (dry run mode)\n")
			} else {
				fmt.Printf("   ⏭️  Skipped (low confidence or missing data)\n")
			}

			fmt.Println()
		}

		if imported > 0 {
			fmt.Printf("🎉 Successfully imported %d account(s)!\n", imported)
			fmt.Println("💡 Use 'gitpersona list' to see all accounts")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)

	discoverCmd.Flags().Bool("dry-run", false, "Show what would be discovered without importing")
	discoverCmd.Flags().Bool("auto-import", false, "Automatically import suitable accounts")
	discoverCmd.Flags().Bool("overwrite", false, "Allow discovery even when accounts already exist")
}
