package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Auto-discover existing Git accounts on your system",
	Long: `Automatically discover and import existing Git accounts from:

- Every authenticated GitHub CLI account on every configured host
- SSH keys in ~/.ssh/ directory
- GPG signing keys from system keyring
- Matches provider, SSH, and GPG evidence only when email ownership is unambiguous

GitHub CLI identities are keyed by provider, hostname, and username, so multiple
accounts on github.com and GitHub Enterprise hosts remain separate. Discovery
does not switch the active GitHub CLI account or store provider tokens.

By default, complete results with confidence 6/10 or higher are imported without
a confirmation prompt, and incomplete higher-confidence results may be saved as
pending metadata. Use --dry-run for a guaranteed read-only scan. --auto-import
also permits otherwise complete results below the normal confidence threshold.

Examples:
  # Discover all Git accounts (SSH + GPG)
  gitshift discover

  # Auto-import discovered accounts
  gitshift discover --auto-import

  # Dry run to see what would be discovered
  gitshift discover --dry-run

  # Assign aliases by stable identity (repeat --alias for more accounts)
  gitshift discover --alias 'github|github.com|octocat=personal'

  # Prompt for aliases, accepting a suggestion with Enter
  gitshift discover --prompt-aliases`,
	RunE: runDiscoverCommand,
}

func runDiscoverCommand(cmd *cobra.Command, args []string) error {
	discoveryService := discovery.NewAccountDiscovery()
	return runDiscoverCommandWithScanner(cmd, args, discoveryService.ScanExistingAccounts)
}

func runDiscoverCommandWithScanner(cmd *cobra.Command, args []string, scan func() ([]*discovery.DiscoveredAccount, error)) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	overwrite, _ := cmd.Flags().GetBool("overwrite")
	aliasSpecs, _ := cmd.Flags().GetStringArray("alias")
	promptAliases, _ := cmd.Flags().GetBool("prompt-aliases")
	customizeAliases := len(aliasSpecs) > 0 || promptAliases

	configManager := newConfigManager()
	var loadErr error
	if dryRun || customizeAliases {
		loadErr = configManager.LoadReadOnly()
	} else {
		loadErr = configManager.Load()
	}
	if loadErr != nil {
		return fmt.Errorf("failed to load configuration: %w", loadErr)
	}

	// Check if accounts already exist
	existingAccounts := configManager.ListAccounts()
	existingPending := configManager.ListPendingAccounts()

	if len(existingAccounts) > 0 && !dryRun {
		if !overwrite && !customizeAliases {
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
	fmt.Println("🔍 Scanning system for existing Git accounts...")

	discovered, err := scan()
	if err != nil {
		return fmt.Errorf("failed to discover accounts: %w", err)
	}

	if len(discovered) == 0 {
		fmt.Println("❌ No existing Git accounts found on your system.")
		fmt.Println("💡 Use 'gitshift add ALIAS --name NAME --email EMAIL --username USERNAME' to add an account.")
		return nil
	}

	if customizeAliases {
		aliases, err := planDiscoveryAliases(cmd, discovered, aliasSpecs, promptAliases, overwrite, existingAccounts, existingPending)
		if err != nil {
			return fmt.Errorf("plan discovered aliases: %w", err)
		}
		for i, alias := range aliases {
			discovered[i].Alias = alias
		}

		// Alias input and the complete plan are validated against a read-only
		// snapshot first. Only a successful, non-dry-run plan may initialize,
		// migrate, chmod, or otherwise prepare configuration for persistence.
		if !dryRun {
			configManager = newConfigManager()
			if err := configManager.Load(); err != nil {
				return fmt.Errorf("failed to prepare configuration for import: %w", err)
			}
		}
	}

	fmt.Printf("✅ Found %d potential account(s):\n\n", len(discovered))

	imported := 0
	var importedAccounts []*models.Account
	var replacementPending []*models.PendingAccount
	autoImport, _ := cmd.Flags().GetBool("auto-import")
	for i, account := range discovered {
		fmt.Printf("📋 Account %d:\n", i+1)
		fmt.Printf("   Alias: %s\n", account.Alias)
		if account.Name != "" {
			fmt.Printf("   Name: %s\n", account.Name)
		}
		if account.Email != "" {
			fmt.Printf("   Email: %s\n", account.Email)
		}

		// Display platform
		platform := account.GetPlatform()
		platformEmoji := "🐙" // GitHub octopus
		if platform == "gitlab" {
			platformEmoji = "🦊" // GitLab fox
		}
		fmt.Printf("   Platform: %s %s\n", platformEmoji, strings.ToUpper(platform[:1])+platform[1:])
		if account.Domain != "" {
			fmt.Printf("   Host: %s\n", account.Domain)
		}

		if account.GitHubUsername != "" {
			fmt.Printf("   Username: @%s\n", account.GitHubUsername)
		}

		// Display SSH key information
		if account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
		}

		// Display GPG key information
		if account.GPGKeyID != "" {
			fmt.Printf("   GPG Key: %s (%s, %d bits)\n", account.GPGKeyID, account.GPGKeyType, account.GPGKeySize)
			if account.GPGKeyExpiry != nil {
				fmt.Printf("   GPG Expiry: %s\n", account.GPGKeyExpiry.Format("2006-01-02"))
			}
		}

		fmt.Printf("   Source: %s\n", account.Source)
		if account.IdentityKey != "" {
			fmt.Printf("   Identity: %s\n", account.IdentityKey)
		}
		fmt.Printf("   Confidence: %d/10\n", account.Confidence)

		// More lenient import criteria - allow import if we have at least name OR email, and confidence is reasonable
		canImport := !dryRun && (autoImport || account.Confidence >= 6) &&
			(account.Name != "" || account.Email != "") &&
			(account.GitHubUsername != "" || account.SSHKeyPath != "")

		if canImport {
			if overwrite {
				if err := account.Account.Validate(); err != nil {
					fmt.Printf("   ❌ Failed to import: %v\n", err)
				} else {
					fmt.Printf("   ✅ Ready to import.\n")
					imported++
					importedAccounts = append(importedAccounts, account.Account)
				}
			} else if err := configManager.AddAccount(account.Account); err != nil {
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

				if overwrite {
					replacementPending = append(replacementPending, pendingAccount)
					fmt.Printf("   📋 Ready to add as pending (missing: %s)\n", strings.Join(missingFields, ", "))
				} else if err := configManager.AddPendingAccount(pendingAccount); err != nil {
					fmt.Printf("   ❌ Failed to add as pending: %v\n", err)
				} else {
					fmt.Printf("   📋 Added to pending accounts (missing: %s)\n", strings.Join(missingFields, ", "))

					// Provide specific completion command based on what's missing
					if account.GitHubUsername != "" {
						fmt.Printf("   💡 Recreate with: gitshift add %s --username %s", account.Alias, account.GitHubUsername)
						if account.Name == "" {
							fmt.Printf(" --name \"Your Name\"")
						}
						if account.Email == "" {
							fmt.Printf(" --email \"your@email.com\"")
						}
						fmt.Println()
					} else {
						fmt.Printf("   💡 Recreate with: gitshift add %s --username USERNAME", account.Alias)
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
						recommendations = append(recommendations, fmt.Sprintf("gitshift add %s --username %s --name \"Your Name\" --email \"your@email.com\"", account.Alias, account.GitHubUsername))
					} else if account.Alias != "" {
						recommendations = append(recommendations, fmt.Sprintf("gitshift add %s --name \"Your Name\" --email \"your@email.com\"", account.Alias))
					} else {
						recommendations = append(recommendations, "--name \"Your Name\" --email \"your@email.com\"")
					}
				}
				if account.GitHubUsername == "" && account.SSHKeyPath == "" {
					reasons = append(reasons, "missing GitHub username and SSH key")
					recommendations = append(recommendations, fmt.Sprintf("gitshift add %s --username USERNAME", account.Alias))
				}

				fmt.Printf("%s\n", strings.Join(reasons, ", "))

				// Provide specific recommendations
				if len(recommendations) > 0 {
					fmt.Printf("   💡 To add this account:")
					if account.GitHubUsername != "" {
						// We have GitHub username, suggest completing with missing fields
						fmt.Printf(" gitshift add %s --username %s", account.Alias, account.GitHubUsername)
						if len(recommendations) > 1 {
							fmt.Printf(" %s", strings.Join(recommendations[1:], " "))
						}
					} else if account.Alias != "" && (account.Name != "" || account.Email != "") {
						// We have alias and some info, suggest using the alias
						fmt.Printf(" gitshift add %s", account.Alias)
						// Filter out the generic recommendation and use specific ones
						specificRecommendations := []string{}
						for _, rec := range recommendations {
							if rec != fmt.Sprintf("gitshift add %s --username USERNAME", account.Alias) {
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

	if overwrite && !dryRun {
		if len(importedAccounts)+len(replacementPending) == 0 {
			fmt.Println("⚠️  No importable accounts were found; existing configuration was preserved.")
		} else {
			fmt.Printf("🗑️  Replacing %d existing account(s)...\n", len(existingAccounts))
			if err := configManager.ReplaceAccounts(importedAccounts, replacementPending); err != nil {
				return fmt.Errorf("failed to replace existing accounts: %w", err)
			}
			fmt.Println("✅ Existing accounts replaced.")
		}
	}

	if imported > 0 {
		fmt.Printf("🎉 Successfully imported %d account(s)!\n", imported)
		fmt.Println("💡 Use 'gitshift list' to see all accounts")

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
			fmt.Println("💡 Use 'gitshift list' to see pending accounts")
			fmt.Println("💡 Recreate pending entries with 'gitshift add <alias> --name NAME --email EMAIL --username USERNAME'")
		}
	}

	return nil
}

// testSSHForAccount performs basic SSH testing for a discovered account
func testSSHForAccount(account *models.Account) error {
	// This is a simplified SSH test - for full testing, users can run 'gitshift ssh test <alias>'
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
	discoverCmd.Flags().Bool("auto-import", false, "Also import complete results below the normal confidence threshold")
	discoverCmd.Flags().Bool("overwrite", false, "Allow discovery even when accounts already exist")
	discoverCmd.Flags().StringArray("alias", nil, "Set an alias with SELECTOR=ALIAS (repeatable; selector is an identity key or unique suggested alias)")
	discoverCmd.Flags().Bool("prompt-aliases", false, "Prompt for aliases not assigned with --alias")
}
