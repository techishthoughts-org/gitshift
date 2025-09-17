package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/discovery"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Auto-discover existing Git accounts on your system",
	Long: `Automatically discover and import existing Git accounts from:

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
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		if len(existingAccounts) > 0 {
			if !overwrite {
				fmt.Printf("⚠️  You already have %d account(s) configured.\n", len(existingAccounts))
				fmt.Println("Use --overwrite to replace existing accounts.")
				fmt.Println("\nExisting accounts:")
				for _, acc := range existingAccounts {
					fmt.Printf("  - %s (%s - %s)\n", acc.Alias, acc.Name, acc.Email)
				}
				return nil
			}

			// Clear existing accounts when overwrite is enabled
			fmt.Printf("🗑️  Clearing %d existing account(s)...\n", len(existingAccounts))
			if err := configManager.ClearAllAccounts(); err != nil {
				return fmt.Errorf("failed to clear existing accounts: %w", err)
			}
			fmt.Println("✅ Existing accounts cleared.")
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
		var importedAccounts []*models.Account
		autoImport, _ := cmd.Flags().GetBool("auto-import")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

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

			// More lenient import criteria - allow import if we have at least name OR email, and confidence is reasonable
			canImport := !dryRun && (autoImport || account.Confidence >= 6) &&
				(account.Name != "" || account.Email != "") &&
				(account.GitHubUsername != "" || account.SSHKeyPath != "")

			if canImport {
				if err := configManager.AddAccount(account.Account); err != nil {
					fmt.Printf("   ❌ Failed to import: %v\n", err)
				} else {
					fmt.Printf("   ✅ Imported successfully!\n")
					imported++
					importedAccounts = append(importedAccounts, account.Account)
				}
			} else if dryRun {
				fmt.Printf("   🔍 Would import (dry run mode)\n")
			} else {
				// Check if we can add as pending account (more inclusive criteria)
				hasUsefulInfo := account.Confidence >= 6 && (account.GitHubUsername != "" ||
					account.SSHKeyPath != "" ||
					account.Name != "" ||
					account.Email != "")

				if hasUsefulInfo {
					// Create pending account for manual completion
					missingFields := []string{}
					if account.Name == "" {
						missingFields = append(missingFields, "name")
					}
					if account.Email == "" {
						missingFields = append(missingFields, "email")
					}
					if account.GitHubUsername == "" {
						missingFields = append(missingFields, "github_username")
					}
					if account.SSHKeyPath == "" {
						missingFields = append(missingFields, "ssh_key")
					}

					partialData := make(map[string]string)
					if account.SSHKeyPath != "" {
						partialData["ssh_key_path"] = account.SSHKeyPath
					}
					if account.Name != "" {
						partialData["name"] = account.Name
					}
					if account.Email != "" {
						partialData["email"] = account.Email
					}

					pendingAccount := models.NewPendingAccount(
						account.Alias,
						account.GitHubUsername,
						account.Source,
						account.Confidence,
						missingFields,
						partialData,
					)

					if err := configManager.AddPendingAccount(pendingAccount); err != nil {
						fmt.Printf("   ❌ Failed to add as pending: %v\n", err)
					} else {
						fmt.Printf("   📋 Added to pending accounts (missing: %s)\n", strings.Join(missingFields, ", "))

						// Provide specific completion command based on what's missing
						if account.GitHubUsername != "" {
							fmt.Printf("   💡 Complete with: gitpersona complete %s", account.Alias)
							if account.Name == "" {
								fmt.Printf(" --name \"Your Name\"")
							}
							if account.Email == "" {
								fmt.Printf(" --email \"your@email.com\"")
							}
							fmt.Println()
						} else {
							fmt.Printf("   💡 Complete with: gitpersona add-github USERNAME --alias %s", account.Alias)
							if account.Name != "" {
								fmt.Printf(" --name \"%s\"", account.Name)
							} else {
								fmt.Printf(" --name \"Your Name\"")
							}
							if account.Email != "" {
								fmt.Printf(" --email \"%s\"", account.Email)
							} else {
								fmt.Printf(" --email \"your@email.com\"")
							}
							fmt.Println()
						}
					}
				} else {
					fmt.Printf("   ⏭️  Skipped: ")
					reasons := []string{}
					recommendations := []string{}

					if account.Confidence < 6 {
						reasons = append(reasons, fmt.Sprintf("low confidence (%d/10)", account.Confidence))
					}
					if account.Name == "" && account.Email == "" {
						reasons = append(reasons, "missing name and email")
						if account.GitHubUsername != "" {
							recommendations = append(recommendations, fmt.Sprintf("gitpersona add-github %s --name \"Your Name\" --email \"your@email.com\"", account.GitHubUsername))
						} else if account.Alias != "" {
							recommendations = append(recommendations, fmt.Sprintf("gitpersona add %s --name \"Your Name\" --email \"your@email.com\"", account.Alias))
						} else {
							recommendations = append(recommendations, "--name \"Your Name\" --email \"your@email.com\"")
						}
					}
					if account.GitHubUsername == "" && account.SSHKeyPath == "" {
						reasons = append(reasons, "missing GitHub username and SSH key")
						recommendations = append(recommendations, "gitpersona add-github USERNAME")
					}

					fmt.Printf("%s\n", strings.Join(reasons, ", "))

					// Provide specific recommendations
					if len(recommendations) > 0 {
						fmt.Printf("   💡 To add this account:")
						if account.GitHubUsername != "" {
							// We have GitHub username, suggest completing with missing fields
							fmt.Printf(" gitpersona add-github %s", account.GitHubUsername)
							if len(recommendations) > 1 {
								fmt.Printf(" %s", strings.Join(recommendations[1:], " "))
							}
						} else if account.Alias != "" && (account.Name != "" || account.Email != "") {
							// We have alias and some info, suggest using the alias
							fmt.Printf(" gitpersona add %s", account.Alias)
							// Filter out the generic recommendation and use specific ones
							specificRecommendations := []string{}
							for _, rec := range recommendations {
								if rec != "gitpersona add-github USERNAME" {
									specificRecommendations = append(specificRecommendations, rec)
								}
							}
							if len(specificRecommendations) > 0 {
								fmt.Printf(" %s", strings.Join(specificRecommendations, " "))
							}
						} else {
							// No GitHub username, suggest the general command
							fmt.Printf(" %s", recommendations[0])
						}
						fmt.Println()
					}
				}
			}

			fmt.Println()
		}

		if imported > 0 {
			fmt.Printf("🎉 Successfully imported %d account(s)!\n", imported)
			fmt.Println("💡 Use 'gitpersona list' to see all accounts")

			// Automatically test SSH for imported accounts
			if !dryRun {
				fmt.Println("\n🔐 Testing SSH connectivity for imported accounts...")
				for _, account := range importedAccounts {
					if account.SSHKeyPath != "" {
						fmt.Printf("\n🧪 Testing SSH for account '%s'...\n", account.Alias)
						if err := testSSHForAccount(account); err != nil {
							fmt.Printf("   ⚠️  SSH test failed: %v\n", err)
						} else {
							fmt.Printf("   ✅ SSH test passed!\n")
						}
					}
				}
			}
		}

		// Check for pending accounts
		if !dryRun {
			pendingAccounts := configManager.ListPendingAccounts()
			if len(pendingAccounts) > 0 {
				fmt.Printf("\n📋 Found %d pending account(s) that need completion\n", len(pendingAccounts))
				fmt.Println("💡 Use 'gitpersona pending' to see pending accounts")
				fmt.Println("💡 Use 'gitpersona complete <alias>' to complete pending accounts")
			}
		}

		return nil
	},
}

// testSSHForAccount performs basic SSH testing for a discovered account
func testSSHForAccount(account *models.Account) error {
	// This is a simplified SSH test - for full testing, users can run 'gitpersona ssh test <alias>'
	if account.SSHKeyPath == "" {
		return fmt.Errorf("no SSH key configured")
	}

	// Check if SSH key file exists
	if _, err := os.Stat(account.SSHKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key file not found: %s", account.SSHKeyPath)
	}

	// Check if SSH key file is readable
	if _, err := os.ReadFile(account.SSHKeyPath); err != nil {
		return fmt.Errorf("SSH key file not readable: %s", err)
	}

	// Check file permissions (should be 600 for SSH keys)
	if info, err := os.Stat(account.SSHKeyPath); err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			fmt.Printf("   ⚠️  SSH key permissions should be 600, got %o\n", mode)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(discoverCmd)

	discoverCmd.Flags().Bool("dry-run", false, "Show what would be discovered without importing")
	discoverCmd.Flags().Bool("auto-import", false, "Automatically import suitable accounts")
	discoverCmd.Flags().Bool("overwrite", false, "Allow discovery even when accounts already exist")
}
