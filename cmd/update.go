package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
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

		configManager := newConfigManager()
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
		newUsername, _ := cmd.Flags().GetString("username")
		newGitHubUsername, _ := cmd.Flags().GetString("github-username")
		newSSHKey, _ := cmd.Flags().GetString("ssh-key")
		newDescription, _ := cmd.Flags().GetString("description")
		newPlatform, _ := cmd.Flags().GetString("platform")
		newDomain, _ := cmd.Flags().GetString("domain")
		newAPIEndpoint, _ := cmd.Flags().GetString("api-endpoint")
		autoGPG, _ := cmd.Flags().GetBool("auto-gpg")
		setDefault, _ := cmd.Flags().GetBool("default")

		// Check if any updates were requested
		if newUsername != "" && newGitHubUsername != "" && newUsername != newGitHubUsername {
			return fmt.Errorf("--username and --github-username must match when both are supplied")
		}
		if newUsername == "" {
			newUsername = newGitHubUsername
		}
		domainChanged := cmd.Flags().Changed("domain")
		apiEndpointChanged := cmd.Flags().Changed("api-endpoint")

		if newName == "" && newEmail == "" && newUsername == "" && newSSHKey == "" && newDescription == "" && newPlatform == "" && !domainChanged && !apiEndpointChanged && !autoGPG && !setDefault && !cmd.Flags().Changed("gpg-signing") {
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
			fmt.Printf("   Default: %t\n", existingAccount.IsDefault)
			fmt.Println("\n💡 Use flags to update specific fields:")
			fmt.Println("   --name, --email, --username, --ssh-key, --description, --platform, --domain, --api-endpoint, --auto-gpg, --gpg-signing, --default")
			return nil
		}

		// Copy the complete account so fields unknown to this command are retained.
		updatedAccount := *existingAccount

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
		if newUsername != "" && newUsername != existingAccount.GetUsername() {
			updatedAccount.SetUsername(strings.TrimPrefix(newUsername, "@"))
			changes = append(changes, fmt.Sprintf("username: @%s → @%s", existingAccount.GetUsername(), updatedAccount.GetUsername()))
		}
		if newSSHKey != "" {
			keyPath, err := models.NormalizeSSHKeyPath(newSSHKey)
			if err != nil {
				return fmt.Errorf("invalid --ssh-key: %w", err)
			}
			if _, err := os.Stat(keyPath); err != nil {
				return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
			}
			if keyPath != existingAccount.SSHKeyPath {
				updatedAccount.SSHKeyPath = keyPath
				changes = append(changes, fmt.Sprintf("SSH key: %s → %s", existingAccount.SSHKeyPath, keyPath))
			}
		}
		if newDescription != "" && newDescription != existingAccount.Description {
			updatedAccount.Description = newDescription
			changes = append(changes, fmt.Sprintf("description: %s → %s", existingAccount.Description, newDescription))
		}
		if newPlatform != "" && newPlatform != existingAccount.GetPlatform() {
			updatedAccount.Platform = newPlatform
			if !domainChanged {
				updatedAccount.Domain = ""
			}
			if !apiEndpointChanged {
				updatedAccount.APIEndpoint = ""
			}
			changes = append(changes, fmt.Sprintf("platform: %s → %s", existingAccount.GetPlatform(), newPlatform))
		}
		if domainChanged && newDomain != existingAccount.Domain {
			updatedAccount.Domain = newDomain
			changes = append(changes, fmt.Sprintf("domain: %s → %s", existingAccount.GetDomain(), updatedAccount.GetDomain()))
		}
		if apiEndpointChanged && newAPIEndpoint != existingAccount.APIEndpoint {
			updatedAccount.APIEndpoint = newAPIEndpoint
			changes = append(changes, "API endpoint updated")
		}

		// Auto-associate GPG key if requested
		if autoGPG {
			if updatedAccount.Email == "" {
				return fmt.Errorf("--auto-gpg needs an account email to match")
			}
			gpgAccounts, err := scanGPGKeys()
			if err != nil {
				return fmt.Errorf("--auto-gpg: scan GPG keys: %w", err)
			}
			match := matchGPGKey(gpgAccounts, updatedAccount.Email)
			if match == nil {
				return fmt.Errorf("--auto-gpg: no GPG secret key with email %s; create one with 'gitshift gpg-keygen %s'", updatedAccount.Email, alias)
			}
			updatedAccount.GPGKeyID = match.GPGKeyID
			updatedAccount.GPGKeyFingerprint = match.GPGKeyFingerprint
			updatedAccount.GPGKeyType = match.GPGKeyType
			updatedAccount.GPGKeySize = match.GPGKeySize
			updatedAccount.GPGKeyExpiry = match.GPGKeyExpiry
			updatedAccount.GPGEnabled = true
			changes = append(changes, fmt.Sprintf("GPG key: associated (%s)", match.GPGKeyID))
		}

		if cmd.Flags().Changed("gpg-signing") {
			signing, _ := cmd.Flags().GetBool("gpg-signing")
			switch {
			case signing && !updatedAccount.HasGPGKey():
				return fmt.Errorf("--gpg-signing needs a GPG key; run 'gitshift update %s --auto-gpg' or 'gitshift gpg-keygen %s'", alias, alias)
			case signing != updatedAccount.GPGEnabled:
				updatedAccount.GPGEnabled = signing
				changes = append(changes, fmt.Sprintf("GPG signing: %s", map[bool]string{true: "enabled", false: "disabled"}[signing]))
			}
		}

		if setDefault && !updatedAccount.IsDefault {
			updatedAccount.IsDefault = true
			changes = append(changes, "set as default account")
		}

		// Validate updated account
		if err := updatedAccount.Validate(); err != nil {
			return fmt.Errorf("updated account validation failed: %w", err)
		}

		if err := configManager.UpdateAccount(&updatedAccount); err != nil {
			return fmt.Errorf("failed to update account: %w", err)
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
		fmt.Printf("   Default: %t\n", updatedAccount.IsDefault)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("name", "n", "", "New display name for the account")
	updateCmd.Flags().StringP("email", "e", "", "New email address for the account")
	updateCmd.Flags().StringP("github-username", "g", "", "New GitHub username (without @)")
	updateCmd.Flags().String("username", "", "New platform username (without @)")
	updateCmd.Flags().StringP("ssh-key", "k", "", "New SSH key path")
	updateCmd.Flags().StringP("description", "d", "", "New description for the account")
	updateCmd.Flags().StringP("platform", "p", "", "Set platform (github or gitlab)")
	updateCmd.Flags().String("domain", "", "Set provider domain; use an empty value to restore the platform default")
	updateCmd.Flags().String("api-endpoint", "", "Set provider API endpoint; use an empty value to clear it")
	updateCmd.Flags().Bool("auto-gpg", false, "Automatically find and associate GPG key by email")
	updateCmd.Flags().Bool("default", false, "Set this account as the default")
	updateCmd.Flags().Bool("gpg-signing", false, "Sign commits and tags with the account's GPG key (--gpg-signing=false to stop)")
}

// scanGPGKeys is replaced in tests.
var scanGPGKeys = func() ([]*discovery.DiscoveredAccount, error) {
	return discovery.NewGPGScanner().ScanGPGKeys()
}

func matchGPGKey(keys []*discovery.DiscoveredAccount, email string) *discovery.DiscoveredAccount {
	for _, key := range keys {
		if key != nil && key.Account != nil && key.GPGKeyID != "" && strings.EqualFold(key.Email, email) {
			return key
		}
	}
	return nil
}
