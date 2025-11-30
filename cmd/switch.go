package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
	"github.com/techishthoughts/gitshift/internal/ssh"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [account-alias]",
	Short: "🔄 Switch to a different Git account (GitHub, GitLab, etc.)",
	Long: `Switch to a different Git platform account with comprehensive configuration.

This command will:
- Switch SSH configuration and keys
- Update Git user.name and user.email
- Update platform token environment (if applicable)
- Test the connection

Works with all supported platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)
- Bitbucket (coming soon)
- Custom Git platforms

Examples:
  # Switch to GitHub account
  gitshift switch work-github
  gitshift switch personal-github --force

  # Switch to GitLab account
  gitshift switch work-gitlab
  gitshift switch personal-gitlab --validate-only

  # Switch to self-hosted GitLab
  gitshift switch company-gitlab

  # Switch to GitHub Enterprise
  gitshift switch enterprise`,
	Aliases: []string{"s", "use"},
	Args:    cobra.ExactArgs(1),
	RunE:    runSwitchCommand,
}

// runSwitchCommand executes the switch command
func runSwitchCommand(cmd *cobra.Command, args []string) error {
	accountAlias := args[0]

	// Get flags
	validateOnly, _ := cmd.Flags().GetBool("validate")
	force, _ := cmd.Flags().GetBool("force")

	// Load gitshift configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Handle validate-only mode
	if validateOnly {
		return validateAccount(configManager, accountAlias)
	}

	// Find the account
	accounts := configManager.ListAccounts()
	var targetAccount *models.Account
	for _, account := range accounts {
		if account.Alias == accountAlias {
			targetAccount = account
			break
		}
	}

	if targetAccount == nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}

	fmt.Printf("🔄 Switching to account '%s'...\n", accountAlias)
	fmt.Printf("   Name: %s\n", targetAccount.Name)
	fmt.Printf("   Email: %s\n", targetAccount.Email)

	// 1. Switch SSH configuration if SSH key is configured
	if targetAccount.SSHKeyPath != "" {
		if _, err := os.Stat(targetAccount.SSHKeyPath); err != nil {
			if force {
				fmt.Printf("⚠️  SSH key not found at %s, skipping SSH switch (--force enabled)\n", targetAccount.SSHKeyPath)
			} else {
				return fmt.Errorf("SSH key not found at %s: %w", targetAccount.SSHKeyPath, err)
			}
		} else {
			// SSH key exists, proceed with switch
			fmt.Printf("🔑 Switching SSH configuration with proper isolation...\n")
			sshManager := ssh.NewManager()

			// Create a context with timeout for SSH operations
			_, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			if err := sshManager.SwitchToAccount(accountAlias, targetAccount.SSHKeyPath); err != nil {
				if force {
					fmt.Printf("⚠️  SSH switch failed: %v (continuing due to --force)\n", err)
				} else {
					return fmt.Errorf("SSH switch failed: %w", err)
				}
			} else {
				fmt.Printf("✅ SSH configuration updated with complete isolation\n")
				fmt.Printf("   • SSH config configured for account: %s\n", accountAlias)
				fmt.Printf("   • SSH agent cleared and key loaded: %s\n", targetAccount.SSHKeyPath)
				fmt.Printf("   • SSH connection tested successfully\n")
			}
		}
	} else {
		fmt.Printf("ℹ️  No SSH key configured for this account\n")
		fmt.Printf("   Consider running: gitshift ssh-keys generate %s\n", accountAlias)
	}

	// 2. Update Git configuration
	fmt.Printf("🔧 Updating Git configuration...\n")
	if err := updateGitConfig(targetAccount); err != nil {
		if force {
			fmt.Printf("⚠️  Git config update failed: %v (continuing due to --force)\n", err)
		} else {
			return fmt.Errorf("failed to update Git configuration: %w", err)
		}
	} else {
		fmt.Printf("✅ Git configuration updated\n")
	}

	// 2.5 Update GPG configuration if account has GPG key
	if targetAccount.HasGPGKey() {
		fmt.Printf("🔐 Configuring GPG signing...\n")
		if err := updateGPGConfig(targetAccount); err != nil {
			if force {
				fmt.Printf("⚠️  GPG config update failed: %v (continuing due to --force)\n", err)
			} else {
				fmt.Printf("⚠️  GPG config update failed: %v\n", err)
				fmt.Printf("   Git configuration updated but GPG signing may not work\n")
			}
		} else {
			if targetAccount.IsGPGEnabled() {
				fmt.Printf("✅ GPG signing enabled (key: %s)\n", targetAccount.GPGKeyID)
			} else {
				fmt.Printf("ℹ️  GPG key configured but automatic signing is disabled\n")
				fmt.Printf("   To enable: gitshift gpg-keygen %s --enable\n", accountAlias)
			}
		}
	} else {
		// No GPG key, disable signing
		fmt.Printf("🔓 Disabling GPG signing (no GPG key configured)...\n")
		if err := disableGPGSigning(); err != nil {
			fmt.Printf("⚠️  Failed to disable GPG signing: %v\n", err)
		}
	}

	// 3. Update current account in gitshift config
	fmt.Printf("📝 Updating gitshift configuration...\n")
	if err := configManager.SetCurrentAccount(accountAlias); err != nil {
		return fmt.Errorf("failed to set current account: %w", err)
	}
	if err := configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	fmt.Printf("✅ gitshift configuration updated\n")

	// 4. Update GitHub token if using GitHub CLI
	fmt.Printf("🔐 Switching GitHub CLI authentication...\n")
	if err := switchGitHubCLI(accountAlias); err != nil {
		if force {
			fmt.Printf("⚠️  GitHub CLI switch failed: %v (continuing due to --force)\n", err)
		} else {
			fmt.Printf("⚠️  GitHub CLI switch failed: %v\n", err)
			fmt.Printf("   You may need to authenticate manually: gh auth login\n")
		}
	} else {
		fmt.Printf("✅ GitHub CLI authentication updated\n")
	}

	// 5. Test the setup (unless forcing)
	if !force {
		fmt.Printf("🧪 Testing configuration...\n")
		if err := testConfiguration(targetAccount); err != nil {
			fmt.Printf("⚠️  Configuration test failed: %v\n", err)
			fmt.Printf("   The switch completed but there may be issues\n")
		} else {
			fmt.Printf("✅ Configuration test passed\n")
		}
	}

	fmt.Printf("\n🎉 Successfully switched to account '%s'!\n", accountAlias)
	fmt.Printf("   You can now use Git with the %s account configuration\n", accountAlias)

	return nil
}

// validateAccount validates an account configuration
func validateAccount(configManager *config.Manager, accountAlias string) error {
	accounts := configManager.ListAccounts()
	var targetAccount *models.Account
	for _, account := range accounts {
		if account.Alias == accountAlias {
			targetAccount = account
			break
		}
	}

	if targetAccount == nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}

	fmt.Printf("🔍 Validating account '%s'...\n", accountAlias)

	issues := 0

	// Check basic configuration
	if targetAccount.Name == "" {
		fmt.Printf("❌ Missing display name\n")
		issues++
	} else {
		fmt.Printf("✅ Display name: %s\n", targetAccount.Name)
	}

	if targetAccount.Email == "" {
		fmt.Printf("❌ Missing email address\n")
		issues++
	} else {
		fmt.Printf("✅ Email: %s\n", targetAccount.Email)
	}

	// Check SSH key
	if targetAccount.SSHKeyPath == "" {
		fmt.Printf("⚠️  No SSH key configured\n")
	} else {
		if _, err := os.Stat(targetAccount.SSHKeyPath); err != nil {
			fmt.Printf("❌ SSH key not found: %s\n", targetAccount.SSHKeyPath)
			issues++
		} else {
			fmt.Printf("✅ SSH key found: %s\n", targetAccount.SSHKeyPath)

			// Test SSH connection
			sshManager := ssh.NewManager()
			if err := sshManager.TestConnection(); err != nil {
				fmt.Printf("⚠️  SSH connection test failed: %v\n", err)
			} else {
				fmt.Printf("✅ SSH connection test passed\n")
			}
		}
	}

	if issues == 0 {
		fmt.Printf("\n✅ Account '%s' is valid and ready to use!\n", accountAlias)
	} else {
		fmt.Printf("\n❌ Account '%s' has %d issue(s) that need to be resolved\n", accountAlias, issues)
		return fmt.Errorf("account validation failed")
	}

	return nil
}

// isGitRepo checks if the current directory is a Git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// updateGitConfig updates the Git user configuration (both global and local if in a repo)
func updateGitConfig(account *models.Account) error {
	// Set global configuration
	if account.Name != "" {
		cmd := exec.Command("git", "config", "--global", "user.name", account.Name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git user.name: %w", err)
		}
	}

	if account.Email != "" {
		cmd := exec.Command("git", "config", "--global", "user.email", account.Email)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git user.email: %w", err)
		}
	}

	// Set SSH command to use the account's SSH key for proper isolation
	if account.SSHKeyPath != "" {
		sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", account.SSHKeyPath)
		cmd := exec.Command("git", "config", "--global", "core.sshCommand", sshCommand)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global git core.sshCommand: %w", err)
		}
	}

	// Check if we're in a Git repository and set local config too
	if isGitRepo() {
		if account.Name != "" {
			cmd := exec.Command("git", "config", "--local", "user.name", account.Name)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git user.name: %w", err)
			}
		}

		if account.Email != "" {
			cmd := exec.Command("git", "config", "--local", "user.email", account.Email)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git user.email: %w", err)
			}
		}

		// Also set SSH command locally for better isolation
		if account.SSHKeyPath != "" {
			sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", account.SSHKeyPath)
			cmd := exec.Command("git", "config", "--local", "core.sshCommand", sshCommand)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to set local git core.sshCommand: %w", err)
			}
		}
	}

	return nil
}

// switchGitHubCLI switches the GitHub CLI authentication
func switchGitHubCLI(accountAlias string) error {
	// Check if gh CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI not found")
	}

	// Try to switch to the account
	cmd := exec.Command("gh", "auth", "switch", "--user", accountAlias)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the account doesn't exist in gh auth, that's OK
		if strings.Contains(string(output), "not found") {
			return fmt.Errorf("account '%s' not found in GitHub CLI - run 'gh auth login' to add it", accountAlias)
		}
		return fmt.Errorf("gh auth switch failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// testConfiguration tests the current configuration
func testConfiguration(account *models.Account) error {
	// Test Git configuration
	nameCmd := exec.Command("git", "config", "--global", "user.name")
	nameOutput, err := nameCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.name: %w", err)
	}

	emailCmd := exec.Command("git", "config", "--global", "user.email")
	emailOutput, err := emailCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.email: %w", err)
	}

	gitName := strings.TrimSpace(string(nameOutput))
	gitEmail := strings.TrimSpace(string(emailOutput))

	if account.Name != "" && gitName != account.Name {
		return fmt.Errorf("git user.name mismatch: expected '%s', got '%s'", account.Name, gitName)
	}

	if account.Email != "" && gitEmail != account.Email {
		return fmt.Errorf("git user.email mismatch: expected '%s', got '%s'", account.Email, gitEmail)
	}

	// Test SSH if key is configured
	if account.SSHKeyPath != "" {
		if _, err := os.Stat(account.SSHKeyPath); err == nil {
			sshManager := ssh.NewManager()
			if err := sshManager.TestConnection(); err != nil {
				return fmt.Errorf("SSH connection test failed: %w", err)
			}
		}
	}

	return nil
}

// getCurrentAccount returns the currently active account alias, or an error if not found
func getCurrentAccount() (string, error) {
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return "", fmt.Errorf("failed to load configuration: %w", err)
	}

	currentAccount := configManager.GetConfig().CurrentAccount
	if currentAccount == "" {
		return "", fmt.Errorf("no active account set")
	}

	return currentAccount, nil
}

// updateGPGConfig updates Git GPG signing configuration for the account
func updateGPGConfig(account *models.Account) error {
	if !account.HasGPGKey() {
		return fmt.Errorf("account has no GPG key configured")
	}

	// Set the signing key
	cmd := exec.Command("git", "config", "--global", "user.signingkey", account.GPGKeyID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set user.signingkey: %w", err)
	}

	// Enable or disable automatic signing based on account preference
	if account.IsGPGEnabled() {
		// Enable commit signing
		cmd = exec.Command("git", "config", "--global", "commit.gpgsign", "true")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable commit.gpgsign: %w", err)
		}

		// Enable tag signing
		cmd = exec.Command("git", "config", "--global", "tag.gpgsign", "true")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable tag.gpgsign: %w", err)
		}
	} else {
		// Disable automatic signing but keep the key configured
		cmd = exec.Command("git", "config", "--global", "commit.gpgsign", "false")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disable commit.gpgsign: %w", err)
		}

		cmd = exec.Command("git", "config", "--global", "tag.gpgsign", "false")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disable tag.gpgsign: %w", err)
		}
	}

	return nil
}

// disableGPGSigning disables GPG signing in Git configuration
func disableGPGSigning() error {
	// Unset the signing key
	cmd := exec.Command("git", "config", "--global", "--unset", "user.signingkey")
	_ = cmd.Run() // Ignore error if key was not set

	// Disable commit signing
	cmd = exec.Command("git", "config", "--global", "commit.gpgsign", "false")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable commit.gpgsign: %w", err)
	}

	// Disable tag signing
	cmd = exec.Command("git", "config", "--global", "tag.gpgsign", "false")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable tag.gpgsign: %w", err)
	}

	return nil
}

func init() {
	switchCmd.Flags().BoolP("validate", "V", false, "Only validate the account without switching")
	switchCmd.Flags().BoolP("force", "f", false, "Force switch even if validation fails")
	switchCmd.Flags().BoolP("skip-validation", "s", false, "Skip SSH validation (not recommended)")

	rootCmd.AddCommand(switchCmd)
}
