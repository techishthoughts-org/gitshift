package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
)

// updateCmd represents the update command for modifying account information
var updateCmd = &cobra.Command{
	Use:   "update [alias]",
	Short: "🔄 Update account information",
	Long: `Update account information for an existing Git platform account.

This command allows you to modify:
- Display name
- Email address
- GitHub username (works for any platform)
- SSH key path
- Description
- Default account status

Works with accounts from all platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)
- Bitbucket (coming soon)
- Custom Git platforms

Examples:
  # Update GitHub account
  gitshift update work-github --name "New Name"
  gitshift update personal-github --email "new@email.com"

  # Update GitLab account
  gitshift update work-gitlab --name "New Name"
  gitshift update personal-gitlab --github-username "newusername"

  # Update SSH key
  gitshift update work --ssh-key "~/.ssh/new_key" --description "Updated description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if account exists
		existingAccount, err := configManager.GetAccount(alias)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", alias, err)
		}

		// Get update flags
		newName, _ := cmd.Flags().GetString("name")
		newEmail, _ := cmd.Flags().GetString("email")
		newGitHubUsername, _ := cmd.Flags().GetString("github-username")
		newSSHKey, _ := cmd.Flags().GetString("ssh-key")
		newDescription, _ := cmd.Flags().GetString("description")
		newPlatform, _ := cmd.Flags().GetString("platform")
		autoGPG, _ := cmd.Flags().GetBool("auto-gpg")
		setDefault, _ := cmd.Flags().GetBool("default")

		// Check if any updates were requested
		if newName == "" && newEmail == "" && newGitHubUsername == "" && newSSHKey == "" && newDescription == "" && newPlatform == "" && !autoGPG && !setDefault {
			fmt.Printf("📋 Current account information for '%s':\n", alias)
			fmt.Printf("   Name: %s\n", existingAccount.Name)
			fmt.Printf("   Email: %s\n", existingAccount.Email)
			fmt.Printf("   Platform: %s\n", existingAccount.GetPlatform())
			if existingAccount.GetUsername() != "" {
				fmt.Printf("   Username: @%s\n", existingAccount.GetUsername())
			} else if existingAccount.GitHubUsername != "" {
				fmt.Printf("   GitHub: @%s\n", existingAccount.GitHubUsername)
			}
			if existingAccount.SSHKeyPath != "" {
				fmt.Printf("   SSH Key: %s\n", existingAccount.SSHKeyPath)
			}
			if existingAccount.HasGPGKey() {
				fmt.Printf("   GPG Key: %s (%s)\n", existingAccount.GPGKeyID, existingAccount.GPGKeyType)
			} else {
				fmt.Printf("   GPG Key: Not configured\n")
			}
			if existingAccount.Description != "" {
				fmt.Printf("   Description: %s\n", existingAccount.Description)
			}
			fmt.Printf("   Default: %t\n", configManager.GetConfig().CurrentAccount == alias)
			fmt.Println("\n💡 Use flags to update specific fields:")
			fmt.Println("   --name, --email, --github-username, --ssh-key, --description, --platform, --auto-gpg, --default")
			return nil
		}

		// Create updated account - preserve all existing fields including GPG
		updatedAccount := &models.Account{
			Alias:              existingAccount.Alias,
			Name:               existingAccount.Name,
			Email:              existingAccount.Email,
			SSHKeyPath:         existingAccount.SSHKeyPath,
			GitHubUsername:     existingAccount.GitHubUsername,
			Username:           existingAccount.Username,
			Platform:           existingAccount.Platform,
			Domain:             existingAccount.Domain,
			Description:        existingAccount.Description,
			CreatedAt:          existingAccount.CreatedAt,
			LastUsed:           existingAccount.LastUsed,
			GPGKeyID:           existingAccount.GPGKeyID,
			GPGKeyFingerprint:  existingAccount.GPGKeyFingerprint,
			GPGKeyType:         existingAccount.GPGKeyType,
			GPGKeySize:         existingAccount.GPGKeySize,
			GPGKeyExpiry:       existingAccount.GPGKeyExpiry,
			GPGEnabled:         existingAccount.GPGEnabled,
		}

		// Apply updates
		changes := []string{}
		if newName != "" && newName != existingAccount.Name {
			updatedAccount.Name = newName
			changes = append(changes, fmt.Sprintf("name: %s → %s", existingAccount.Name, newName))
		}
		if newEmail != "" && newEmail != existingAccount.Email {
			updatedAccount.Email = newEmail
			changes = append(changes, fmt.Sprintf("email: %s → %s", existingAccount.Email, newEmail))
		}
		if newGitHubUsername != "" && newGitHubUsername != existingAccount.GitHubUsername {
			updatedAccount.GitHubUsername = newGitHubUsername
			changes = append(changes, fmt.Sprintf("GitHub username: @%s → @%s", existingAccount.GitHubUsername, newGitHubUsername))
		}
		if newSSHKey != "" && newSSHKey != existingAccount.SSHKeyPath {
			updatedAccount.SSHKeyPath = newSSHKey
			changes = append(changes, fmt.Sprintf("SSH key: %s → %s", existingAccount.SSHKeyPath, newSSHKey))
		}
		if newDescription != "" && newDescription != existingAccount.Description {
			updatedAccount.Description = newDescription
			changes = append(changes, fmt.Sprintf("description: %s → %s", existingAccount.Description, newDescription))
		}
		if newPlatform != "" && newPlatform != existingAccount.GetPlatform() {
			updatedAccount.Platform = newPlatform
			changes = append(changes, fmt.Sprintf("platform: %s → %s", existingAccount.GetPlatform(), newPlatform))
		}

		// Auto-associate GPG key if requested
		if autoGPG && updatedAccount.Email != "" {
			gpgScanner := discovery.NewGPGScanner()
			gpgAccounts, err := gpgScanner.ScanGPGKeys()
			if err == nil {
				// Find GPG key matching the account email
				for _, gpgAcc := range gpgAccounts {
					if strings.EqualFold(gpgAcc.Email, updatedAccount.Email) {
						updatedAccount.GPGKeyID = gpgAcc.GPGKeyID
						updatedAccount.GPGKeyFingerprint = gpgAcc.GPGKeyFingerprint
						updatedAccount.GPGKeyType = gpgAcc.GPGKeyType
						updatedAccount.GPGKeySize = gpgAcc.GPGKeySize
						updatedAccount.GPGKeyExpiry = gpgAcc.GPGKeyExpiry
						updatedAccount.GPGEnabled = true
						changes = append(changes, fmt.Sprintf("GPG key: associated (%s)", gpgAcc.GPGKeyID))
						break
					}
				}
			}
		}

		// Validate updated account
		if err := updatedAccount.Validate(); err != nil {
			return fmt.Errorf("updated account validation failed: %w", err)
		}

		// Remove old account and add updated one
		if err := configManager.RemoveAccount(alias); err != nil {
			return fmt.Errorf("failed to remove old account: %w", err)
		}

		if err := configManager.AddAccount(updatedAccount); err != nil {
			return fmt.Errorf("failed to add updated account: %w", err)
		}

		// Set as default if requested
		if setDefault {
			if err := configManager.SetCurrentAccount(alias); err != nil {
				fmt.Printf("⚠️  Failed to set as default: %v\n", err)
			} else {
				changes = append(changes, "set as default account")
			}
		}

		// Show summary of changes
		if len(changes) > 0 {
			fmt.Printf("✅ Successfully updated account '%s':\n", alias)
			for _, change := range changes {
				fmt.Printf("   • %s\n", change)
			}
		} else {
			fmt.Printf("ℹ️  No changes made to account '%s'\n", alias)
		}

		// Show final account information
		fmt.Printf("\n📋 Updated account information:\n")
		fmt.Printf("   Name: %s\n", updatedAccount.Name)
		fmt.Printf("   Email: %s\n", updatedAccount.Email)
		fmt.Printf("   Platform: %s\n", updatedAccount.GetPlatform())
		if updatedAccount.GetUsername() != "" {
			fmt.Printf("   Username: @%s\n", updatedAccount.GetUsername())
		} else if updatedAccount.GitHubUsername != "" {
			fmt.Printf("   GitHub: @%s\n", updatedAccount.GitHubUsername)
		}
		if updatedAccount.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", updatedAccount.SSHKeyPath)
		}
		if updatedAccount.HasGPGKey() {
			fmt.Printf("   GPG Key: %s (%s)\n", updatedAccount.GPGKeyID, updatedAccount.GPGKeyType)
			if updatedAccount.IsGPGEnabled() {
				fmt.Printf("   GPG Signing: Enabled\n")
			}
		}
		if updatedAccount.Description != "" {
			fmt.Printf("   Description: %s\n", updatedAccount.Description)
		}
		fmt.Printf("   Default: %t\n", configManager.GetConfig().CurrentAccount == alias)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("name", "n", "", "New display name for the account")
	updateCmd.Flags().StringP("email", "e", "", "New email address for the account")
	updateCmd.Flags().StringP("github-username", "g", "", "New GitHub username (without @)")
	updateCmd.Flags().StringP("ssh-key", "k", "", "New SSH key path")
	updateCmd.Flags().StringP("description", "d", "", "New description for the account")
	updateCmd.Flags().StringP("platform", "p", "", "Set platform (github, gitlab, bitbucket)")
	updateCmd.Flags().Bool("auto-gpg", false, "Automatically find and associate GPG key by email")
	updateCmd.Flags().Bool("default", false, "Set this account as the default")
}
