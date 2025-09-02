package cmd

import (
	"fmt"
	"os"

	"github.com/arthurcosta/GitPersona/internal/config"
	"github.com/arthurcosta/GitPersona/internal/git"
	"github.com/arthurcosta/GitPersona/internal/github"
	"github.com/spf13/cobra"
)

// initCmd represents the init command for shell integration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize shell integration for automatic account switching",
	Long: `Initialize shell integration for automatic account switching.

This command outputs shell commands that should be evaluated in your shell
to enable automatic Git account switching based on project configuration.

Add this to your shell configuration file (.bashrc, .zshrc, etc.):
  eval "$(gh-switcher init)"

The command will:
1. Check for a .gh-switcher.yaml file in the current directory
2. Automatically switch to the specified account if found
3. Set up the appropriate Git configuration and SSH settings

Examples:
  gh-switcher init
  eval "$(gh-switcher init)"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			// If config doesn't exist, just return empty output
			return nil
		}

		gitManager := git.NewManager()

		// Check for project configuration in current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return nil // Silent failure for init command
		}

		projectConfig, err := configManager.LoadProjectConfig(currentDir)
		if err != nil {
			// No project config found, check if we're in a git repo
			if gitManager.IsGitRepo(currentDir) {
				// Output current account info if available
				if currentAccount := configManager.GetConfig().CurrentAccount; currentAccount != "" {
					account, err := configManager.GetAccount(currentAccount)
					if err == nil && account.SSHKeyPath != "" {
						fmt.Printf("export GIT_SSH_COMMAND=\"%s\"\n", gitManager.GenerateSSHCommand(account.SSHKeyPath))
					}
				}
			}
			return nil
		}

		// Get the account specified in project config
		account, err := configManager.GetAccount(projectConfig.Account)
		if err != nil {
			fmt.Fprintf(os.Stderr, "# Warning: Account '%s' specified in project config not found\n", projectConfig.Account)
			return nil
		}

		// Generate shell commands to switch to the account
		fmt.Printf("# Switching to account '%s' for this project\n", account.Alias)

		// Check if we're in a git repository
		if gitManager.IsGitRepo(currentDir) {
			// Set local git config
			fmt.Printf("git config --local user.name \"%s\"\n", account.Name)
			fmt.Printf("git config --local user.email \"%s\"\n", account.Email)
		}

		// Set SSH command if SSH key is specified
		if account.SSHKeyPath != "" {
			sshCommand := gitManager.GenerateSSHCommand(account.SSHKeyPath)
			if sshCommand != "" {
				fmt.Printf("export GIT_SSH_COMMAND=\"%s\"\n", sshCommand)
			}
		}

		// Update the current account
		configManager.SetCurrentAccount(account.Alias)

		return nil
	},
}

// currentCmd shows the current account status
var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active GitHub account",
	Long: `Show the current active GitHub account and Git configuration.

This command displays:
- The currently active GitHub account
- Current Git user.name and user.email configuration
- SSH key configuration (if applicable)
- Project-specific configuration (if in a configured project)

Examples:
  gh-switcher current
  gh-switcher current --verbose`,
	Aliases: []string{"status", "whoami"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		gitManager := git.NewManager()
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Check for project configuration
		projectConfig, projectErr := configManager.LoadProjectConfig(currentDir)

		// Get current Git configuration
		gitName, gitEmail, gitErr := gitManager.GetCurrentConfig()

		// Display project information
		if verbose {
			fmt.Printf("Current Directory: %s\n", currentDir)
			fmt.Printf("Is Git Repository: %t\n", gitManager.IsGitRepo(currentDir))
		}

		// Display project configuration
		if projectErr == nil {
			fmt.Printf("📁 Project Account: %s\n", projectConfig.Account)
		} else if verbose {
			fmt.Println("📁 Project Account: None configured")
		}

		// Display active account
		currentAccount := configManager.GetConfig().CurrentAccount
		if currentAccount != "" {
			account, err := configManager.GetAccount(currentAccount)
			if err != nil {
				fmt.Printf("❌ Current Account: %s (not found in configuration)\n", currentAccount)
			} else {
				fmt.Printf("👤 Current Account: %s\n", account.Alias)
				if verbose {
					fmt.Printf("   Name: %s\n", account.Name)
					fmt.Printf("   Email: %s\n", account.Email)
					if account.GitHubUsername != "" {
						fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)

						// Show repository count if possible
						if githubClient, err := getAuthenticatedGitHubClient(); err == nil {
							if repos, err := githubClient.FetchUserRepositories(account.GitHubUsername); err == nil {
								fmt.Printf("   Repositories: %d total\n", len(repos))
							}
						}
					}
					if account.SSHKeyPath != "" {
						fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
					}
					if account.LastUsed != nil {
						fmt.Printf("   Last Used: %s\n", formatTime(*account.LastUsed))
					}
				}
			}
		} else {
			fmt.Println("👤 Current Account: None set")
		}

		// Display Git configuration
		if gitErr == nil {
			fmt.Printf("🔧 Git Configuration:\n")
			fmt.Printf("   user.name:  %s\n", gitName)
			fmt.Printf("   user.email: %s\n", gitEmail)
		} else if verbose {
			fmt.Printf("🔧 Git Configuration: Not available (%v)\n", gitErr)
		}

		// Display SSH configuration
		if currentAccount != "" {
			account, err := configManager.GetAccount(currentAccount)
			if err == nil && account.SSHKeyPath != "" {
				sshCommand := gitManager.GenerateSSHCommand(account.SSHKeyPath)
				if sshCommand != "" {
					fmt.Printf("🔑 SSH Configuration: %s\n", sshCommand)
				}
			}
		}

		// Show suggestions
		if projectErr == nil && projectConfig.Account != currentAccount {
			fmt.Printf("\n💡 Suggestion: Run 'gh-switcher switch %s' to match project configuration\n", projectConfig.Account)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(currentCmd)

	currentCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
}

// getAuthenticatedGitHubClient tries to get an authenticated GitHub client
func getAuthenticatedGitHubClient() (*github.Client, error) {
	tempClient := github.NewClient("")

	// Check if already authenticated
	if token, err := tempClient.GetGitHubToken(); err == nil && token != "" {
		return github.NewClient(token), nil
	}

	return nil, fmt.Errorf("not authenticated - run 'gh auth login' first")
}
