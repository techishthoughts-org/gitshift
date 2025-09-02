package cmd

import (
	"fmt"
	"strings"

	"github.com/arthurcosta/GitPersona/internal/config"
	"github.com/arthurcosta/GitPersona/internal/git"
	"github.com/arthurcosta/GitPersona/internal/github"
	"github.com/arthurcosta/GitPersona/internal/models"
	"github.com/spf13/cobra"
)

// addGithubCmd represents the add-github command for automatic GitHub integration
var addGithubCmd = &cobra.Command{
	Use:   "add-github [github-username]",
	Short: "Add a GitHub account with automatic setup",
	Long: `Add a GitHub account by providing just the GitHub username.
This command will AUTOMATICALLY:

1. 🔐 Authenticate with GitHub (full OAuth permissions)
2. 🔍 Fetch user information from GitHub API
3. 🔑 Generate SSH keys automatically
4. ⬆️  Upload SSH keys to your GitHub account
5. 🎯 Set up complete local configuration
6. 🌐 Configure for global Git usage
7. 🔄 Switch to the account immediately
8. ✅ Ready to use immediately!

ZERO manual steps required - just provide the GitHub username!

Examples:
  gh-switcher add-github thukabjj --email "arthur.alvesdeveloper@gmail.com"
  gh-switcher add-github octocat --alias work
  gh-switcher add-github costaar7 --email "arthur.costa@fanduel.com" --alias work
  gh-switcher add-github username --no-auth  # skip authentication

Features:
- Automatic GitHub OAuth authentication with full permissions
- Automatic user info fetching from GitHub API
- Automatic SSH key generation and upload to GitHub
- Clipboard integration for easy manual copying (if needed)
- Global Git configuration by default
- Immediate account switching after setup`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		githubUsername := strings.TrimPrefix(args[0], "@")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		gitManager := git.NewManager()

		// Get command flags
		alias, _ := cmd.Flags().GetString("alias")
		email, _ := cmd.Flags().GetString("email")
		skipAuth, _ := cmd.Flags().GetBool("no-auth")
		skipSSH, _ := cmd.Flags().GetBool("skip-ssh")

		// Initialize GitHub client with authentication by default
		var githubClient *github.Client

		if !skipAuth {
			// Authenticate with GitHub by default for best experience
			fmt.Println("🔐 Setting up authenticated GitHub access...")
			tempClient := github.NewClient("")
			if err := tempClient.AuthenticateWithGitHub(); err != nil {
				fmt.Printf("⚠️  Authentication failed: %v\n", err)
				fmt.Println("💡 Falling back to unauthenticated access (limited functionality)")
				githubClient = github.NewClient("")
			} else {
				// Get authenticated token
				if token, err := tempClient.GetGitHubToken(); err == nil {
					githubClient = github.NewClient(token)
					fmt.Println("✅ Using authenticated GitHub API access")
					fmt.Println("🎯 Full access enabled: SSH key auto-upload, private emails, etc.")
				} else {
					fmt.Println("⚠️  Token retrieval failed, using unauthenticated access")
					githubClient = github.NewClient("")
				}
			}
		} else {
			githubClient = github.NewClient("")
			fmt.Println("💡 Using unauthenticated GitHub API access (--no-auth specified)")
			fmt.Println("   Limited functionality: No SSH key auto-upload, no private emails")
		}

		// Setup account from GitHub username
		fmt.Printf("🚀 Setting up GitHub account for @%s...\n\n", githubUsername)

		var account *models.Account
		var err error

		if skipSSH {
			// Fetch user info only, skip SSH key generation
			userInfo, err := githubClient.FetchUserInfo(githubUsername)
			if err != nil {
				return fmt.Errorf("failed to fetch GitHub user info: %w", err)
			}

			// Generate alias if not provided
			if alias == "" {
				if userInfo.Name != "" {
					parts := strings.Fields(userInfo.Name)
					if len(parts) > 0 {
						alias = strings.ToLower(parts[0])
					}
				}
				if alias == "" {
					alias = strings.ToLower(userInfo.Login)
				}
			}

			// Handle email with priority: provided > GitHub API > no-reply
			finalEmail := email // from flag
			if finalEmail != "" {
				fmt.Printf("✅ Using provided email: %s\n", finalEmail)
			} else {
				finalEmail = userInfo.Email
				if finalEmail == "" {
					finalEmail = fmt.Sprintf("%s@users.noreply.github.com", githubUsername)
					fmt.Printf("💡 Using GitHub no-reply email: %s\n", finalEmail)
				}
			}

			account = models.NewAccount(alias, userInfo.Name, finalEmail, "")
			account.GitHubUsername = userInfo.Login
			account.Description = fmt.Sprintf("GitHub @%s (no SSH)", userInfo.Login)

		} else {
			// Full automatic setup with SSH key generation
			account, err = githubClient.SetupAccountFromUsername(githubUsername, alias, email)
			if err != nil {
				return fmt.Errorf("failed to setup account: %w", err)
			}
		}

		// Validate account
		if err := account.Validate(); err != nil {
			return fmt.Errorf("account validation failed: %w", err)
		}

		// Check if account already exists
		if existingAccount, err := configManager.GetAccount(account.Alias); err == nil {
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				fmt.Printf("❌ Account '%s' already exists:\n", account.Alias)
				fmt.Printf("   Name: %s\n", existingAccount.Name)
				fmt.Printf("   Email: %s\n", existingAccount.Email)
				fmt.Println("   Use --overwrite to replace it")
				return nil
			}

			// Remove existing account
			if err := configManager.RemoveAccount(account.Alias); err != nil {
				return fmt.Errorf("failed to remove existing account: %w", err)
			}
			fmt.Printf("🔄 Replaced existing account '%s'\n", account.Alias)
		}

		// Add account to configuration
		if err := configManager.AddAccount(account); err != nil {
			return fmt.Errorf("failed to add account: %w", err)
		}

		fmt.Printf("\n🎉 Successfully added GitHub account '%s'!\n", account.Alias)
		fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
		fmt.Printf("   Name: %s\n", account.Name)
		fmt.Printf("   Email: %s\n", account.Email)
		if account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
		}

		// Automatically switch to this account globally
		switchGlobally, _ := cmd.Flags().GetBool("switch")
		setDefault := len(configManager.ListAccounts()) == 1 // First account becomes default

		if switchGlobally || setDefault {
			fmt.Printf("\n🌐 Setting '%s' as global Git configuration...\n", account.Alias)

			if err := gitManager.SetGlobalConfig(account); err != nil {
				fmt.Printf("⚠️  Failed to set global Git config: %v\n", err)
			} else {
				// Update current account
				if err := configManager.SetCurrentAccount(account.Alias); err != nil {
					fmt.Printf("⚠️  Failed to set current account: %v\n", err)
				} else {
					fmt.Printf("✅ Global Git configuration updated!\n")

					// Show current git config
					fmt.Println("\n📋 Current Git configuration:")
					if name, email, err := gitManager.GetCurrentConfig(); err == nil {
						fmt.Printf("   user.name:  %s\n", name)
						fmt.Printf("   user.email: %s\n", email)
					}
				}
			}
		}

		// Show next steps
		fmt.Println("\n💡 Next steps:")

		if account.SSHKeyPath != "" {
			fmt.Println("   1. Add the SSH key to your GitHub account:")
			fmt.Println("      → https://github.com/settings/keys")
			fmt.Println("   2. Test SSH connection:")
			fmt.Printf("      → ssh -T git@github.com -i %s\n", account.SSHKeyPath)
		}

		if !switchGlobally && !setDefault {
			fmt.Printf("   • Switch to this account: gh-switcher switch %s\n", account.Alias)
		}

		fmt.Println("   • List all accounts: gh-switcher list")
		fmt.Println("   • Set up shell integration: eval \"$(gh-switcher init)\"")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(addGithubCmd)

	addGithubCmd.Flags().StringP("alias", "a", "", "Custom alias for the account (auto-generated if not provided)")
	addGithubCmd.Flags().StringP("email", "e", "", "Email address for the account (fetched from GitHub if not provided)")
	addGithubCmd.Flags().Bool("no-auth", false, "Skip GitHub authentication (limited functionality)")
	addGithubCmd.Flags().Bool("skip-ssh", false, "Skip SSH key generation (manual setup required)")
	addGithubCmd.Flags().Bool("overwrite", false, "Overwrite existing account with same alias")
	addGithubCmd.Flags().Bool("switch", true, "Switch to this account immediately after creation (default: true)")
}
