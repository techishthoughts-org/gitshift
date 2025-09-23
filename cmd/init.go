package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/github"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// initCmd represents the init command for shell integration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize shell integration for automatic account switching",
	Long: `Initialize shell integration for automatic account switching.

This command outputs shell commands that should be evaluated in your shell
to enable automatic Git account switching based on project configuration.

Add this to your shell configuration file (.bashrc, .zshrc, etc.):
  eval "$(gitpersona init)"

The command will:
1. Check for a .gitpersona.yaml file in the current directory
2. Automatically switch to the specified account if found
3. Set up the appropriate Git configuration and SSH settings

Examples:
  gitpersona init
  eval "$(gitpersona init)"`,
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
					// reweite .gitconfig with current account info
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
		if err := configManager.SetCurrentAccount(account.Alias); err != nil {
			return fmt.Errorf("failed to set current account: %w", err)
		}

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
  gitpersona current
  gitpersona current --verbose`,
	Aliases: []string{"status", "whoami"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		gitManager := git.NewManager()
		verbose, _ := cmd.Flags().GetBool("verbose")
		recommendations, _ := cmd.Flags().GetBool("recommendations")

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
			fmt.Printf("\n💡 Suggestion: Run 'gitpersona switch %s' to match project configuration\n", projectConfig.Account)
		} else if (verbose || recommendations) && gitManager.IsGitRepo(currentDir) {
			// Show auto-detection recommendation
			accounts := configManager.ListAccounts()
			if len(accounts) > 0 {
				if detectionResult, err := performDetectionSuggestion(gitManager, accounts); err == nil && detectionResult.RecommendedAccount != nil {
					if currentAccount == "" || detectionResult.RecommendedAccount.Alias != currentAccount {
						fmt.Printf("\n💡 Auto-Detection: Switch to '%s' (%.0f%% confidence)\n",
							detectionResult.RecommendedAccount.Alias, detectionResult.Confidence*100)
						fmt.Println("   Run: gitpersona auto-detect")
					} else {
						fmt.Println("\n✅ Using recommended account for this repository")
					}
				}
			}
		}

		return nil
	},
}

// autoIdentifyCmd represents the auto-identify command for immediate account switching
var autoIdentifyCmd = &cobra.Command{
	Use:   "auto-identify",
	Short: "🔍 Automatically identify and switch to the best matching account",
	Long: `Automatically identify and switch to the best matching account based on local context.

This command will:
1. 🔍 Analyze your current Git configuration
2. 🎯 Find the best matching account from your saved accounts
3. ⚡ Switch to that account immediately
4. 🚀 Set up Git config and SSH settings
5. ✅ Show you what was configured

Perfect for:
- Quick account switching in any directory
- Automatic setup when entering new projects
- Immediate account identification without shell integration

Examples:
  gitpersona auto-identify
  gitpersona auto-identify --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("no accounts configured yet. Add your first account with: gitpersona add-github <username>")
		}

		gitManager := git.NewManager()
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		if verbose {
			fmt.Println("🔍 Analyzing current directory and Git configuration...")
		}

		// Check for project-specific configuration first
		projectConfig, err := configManager.LoadProjectConfig(currentDir)
		if err == nil && projectConfig.Account != "" {
			account, err := configManager.GetAccount(projectConfig.Account)
			if err == nil {
				fmt.Printf("🎯 Project-specific account detected: %s\n", account.Alias)
				return autoSwitchToAccount(account, gitManager, verbose)
			}
		}

		// Try automatic identification
		if gitManager.IsGitRepo(currentDir) {
			if verbose {
				fmt.Println("📁 Git repository detected, analyzing configuration...")
			}

			// Get current Git configuration
			currentName, currentEmail, err := gitManager.GetCurrentConfig()
			if err == nil {
				if verbose {
					fmt.Printf("📋 Current Git config: name='%s', email='%s'\n", currentName, currentEmail)
				}

				// Try to find matching account
				bestMatch := findBestMatchingAccount(configManager, currentName, currentEmail)
				if bestMatch != nil {
					fmt.Printf("🔍 Auto-detected account: %s (from Git config)\n", bestMatch.Alias)
					fmt.Printf("   Name: %s\n", bestMatch.Name)
					fmt.Printf("   Email: %s\n", bestMatch.Email)
					return autoSwitchToAccount(bestMatch, gitManager, verbose)
				}
			}

			// Check for SSH key in use
			if verbose {
				fmt.Println("🔑 Checking SSH agent for active keys...")
			}
			sshKeyPath := detectCurrentSSHKey()
			if sshKeyPath != "" {
				if verbose {
					fmt.Printf("🔑 SSH key detected: %s\n", sshKeyPath)
				}
				// Try to find account by SSH key
				account := findAccountBySSHKey(configManager, sshKeyPath)
				if account != nil {
					fmt.Printf("🔑 SSH key matched account: %s\n", account.Alias)
					return autoSwitchToAccount(account, gitManager, verbose)
				}
			}
		}

		// Fallback: use current account if available
		if currentAccount := configManager.GetConfig().CurrentAccount; currentAccount != "" {
			account, err := configManager.GetAccount(currentAccount)
			if err == nil {
				fmt.Printf("📍 Using current account: %s\n", account.Alias)
				return autoSwitchToAccount(account, gitManager, verbose)
			}
		}

		// No automatic identification possible
		fmt.Println("❌ Could not automatically identify an account")
		fmt.Println("\n💡 Available accounts:")
		accounts := configManager.ListAccounts()
		for _, acc := range accounts {
			fmt.Printf("   • %s (@%s) - %s\n", acc.Alias, acc.GitHubUsername, acc.Email)
		}
		fmt.Println("\n🚀 To switch manually:")
		fmt.Println("   gitpersona switch <alias>")
		fmt.Println("   gitpersona add-github <username>  # Add new account")

		return nil
	},
}

// switchToAccount switches to the specified account and shows the configuration
func autoSwitchToAccount(account *models.Account, gitManager *git.Manager, verbose bool) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("\n✅ Switching to account: %s\n", account.Alias)
	fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
	fmt.Printf("   Name: %s\n", account.Name)
	fmt.Printf("   Email: %s\n", account.Email)

	// Set local git config if in a git repo
	if gitManager.IsGitRepo(currentDir) {
		if verbose {
			fmt.Println("📝 Setting local Git configuration...")
		}

		// Set local git config
		if err := gitManager.SetLocalConfig(account); err != nil {
			fmt.Printf("⚠️  Failed to set local Git config: %v\n", err)
		} else {
			fmt.Println("✅ Local Git configuration updated")
		}
	}

	// Set SSH command if SSH key is specified
	if account.SSHKeyPath != "" {
		if verbose {
			fmt.Println("🔑 Configuring SSH key...")
		}

		sshCommand := gitManager.GenerateSSHCommand(account.SSHKeyPath)
		if sshCommand != "" {
			fmt.Printf("🔑 SSH command: %s\n", sshCommand)
			fmt.Println("💡 Add this to your shell: export GIT_SSH_COMMAND=\"" + sshCommand + "\"")
		}
	}

	// Update the current account
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := configManager.SetCurrentAccount(account.Alias); err != nil {
		return fmt.Errorf("failed to set current account: %w", err)
	}

	fmt.Printf("\n🎉 Successfully switched to account: %s\n", account.Alias)

	if verbose {
		fmt.Println("\n📋 Current Git configuration:")
		if name, email, err := gitManager.GetCurrentConfig(); err == nil {
			fmt.Printf("   user.name:  %s\n", name)
			fmt.Printf("   user.email: %s\n", email)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(autoIdentifyCmd)

	currentCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
	currentCmd.Flags().BoolP("recommendations", "r", false, "Show account recommendations")
	autoIdentifyCmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
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

// findBestMatchingAccount finds the best matching account based on Git config
func findBestMatchingAccount(configManager *config.Manager, gitName, gitEmail string) *models.Account {
	accounts := configManager.ListAccounts()
	var bestMatch *models.Account
	var bestScore int

	for _, account := range accounts {
		score := 0

		// Exact email match gets highest score
		if account.Email == gitEmail {
			score += 100
		}

		// Partial email match (domain)
		if gitEmail != "" && account.Email != "" {
			accountDomain := extractDomain(account.Email)
			gitDomain := extractDomain(gitEmail)
			if accountDomain == gitDomain {
				score += 50
			}
		}

		// Name similarity
		if account.Name == gitName {
			score += 75
		} else if gitName != "" && account.Name != "" {
			// Check if names are similar (case-insensitive)
			if strings.EqualFold(account.Name, gitName) {
				score += 60
			}
		}

		// GitHub username in email
		if account.GitHubUsername != "" && strings.Contains(gitEmail, account.GitHubUsername) {
			score += 25
		}

		if score > bestScore {
			bestScore = score
			bestMatch = account
		}
	}

	// Only return if we have a reasonable match
	if bestScore >= 25 {
		return bestMatch
	}

	return nil
}

// findAccountBySSHKey finds an account by SSH key path
func findAccountBySSHKey(configManager *config.Manager, sshKeyPath string) *models.Account {
	accounts := configManager.ListAccounts()
	for _, account := range accounts {
		if account.SSHKeyPath == sshKeyPath {
			return account
		}
	}
	return nil
}

// detectCurrentSSHKey tries to detect which SSH key is currently in use
func detectCurrentSSHKey() string {
	// Check SSH agent for loaded keys
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse SSH agent output to find key paths
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "~/.ssh/") || strings.Contains(line, "/.ssh/") {
			// Extract key path from SSH agent output
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				keyPath := parts[2]
				// Expand ~ to home directory
				if strings.HasPrefix(keyPath, "~") {
					home, err := os.UserHomeDir()
					if err == nil {
						keyPath = filepath.Join(home, keyPath[1:])
					}
				}
				return keyPath
			}
		}
	}

	return ""
}

// extractDomain extracts domain from email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// DetectionSuggestionResult holds the results of account detection for suggestions
type DetectionSuggestionResult struct {
	RecommendedAccount *models.Account `json:"recommended_account"`
	Confidence         float64         `json:"confidence"`
}

// performDetectionSuggestion performs a simple account detection for suggestions
func performDetectionSuggestion(gitManager *git.Manager, accounts []*models.Account) (*DetectionSuggestionResult, error) {
	result := &DetectionSuggestionResult{}

	// Get remote URL if available
	remoteURL, err := gitManager.GetCurrentRemoteURL("origin")
	if err != nil {
		return result, nil // No remote URL, no recommendation
	}

	var bestMatch *models.Account
	bestScore := 0.0

	// Analyze each account for matches
	for _, account := range accounts {
		score := 0.0

		// Factor 1: Remote URL matching (highest weight)
		if account.GitHubUsername != "" && strings.Contains(remoteURL, account.GitHubUsername) {
			score += 0.8
		}

		// Factor 2: SSH key accessibility
		if account.SSHKeyPath != "" && isSSHKeyAccessible(account.SSHKeyPath) {
			score += 0.2
		}

		// Factor 3: Current account preference
		if account.IsDefault {
			score += 0.1
		}

		if score > bestScore {
			bestScore = score
			bestMatch = account
		}
	}

	// Only recommend if we have a good match
	if bestScore >= 0.6 {
		result.RecommendedAccount = bestMatch
		result.Confidence = bestScore
	}

	return result, nil
}
