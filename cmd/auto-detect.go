package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// autoDetectCmd represents the auto-detect command
var autoDetectCmd = &cobra.Command{
	Use:   "auto-detect",
	Short: "🤖 Automatically detect and switch to the appropriate account",
	Long: `Automatically detect which GitHub account should be used based on:

- Current repository's remote URL
- Project directory name patterns
- Git commit history analysis
- SSH key matching

This command will:
- Analyze the current Git repository
- Match it against configured accounts
- Automatically switch to the best matching account
- Update Git configuration accordingly

Examples:
  gitpersona auto-detect
  gitpersona auto-detect --dry-run
  gitpersona auto-detect --force`,
	Aliases: []string{"detect", "auto"},
	RunE:    runAutoDetectCommand,
}

// runAutoDetectCommand executes the auto-detect logic
func runAutoDetectCommand(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Println("🤖 GitPersona Auto-Detection")
	fmt.Println("=" + strings.Repeat("=", 40))

	// Load GitPersona configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	accounts := configManager.ListAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ No accounts configured")
		fmt.Println("   Add accounts using: gitpersona add-github <username>")
		return nil
	}

	// Initialize Git manager
	gitManager := git.NewManager()

	// Check if we're in a Git repository
	if !gitManager.IsGitRepository() {
		fmt.Println("❌ Not in a Git repository")
		fmt.Println("   Navigate to a Git project directory and try again")
		return nil
	}

	fmt.Println("✅ Git repository detected")

	// Perform detection analysis
	detectionResult, err := performDetection(gitManager, accounts, verbose)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Display results
	displayDetectionResults(detectionResult, verbose)

	// Apply changes if not dry-run
	if !dryRun {
		if detectionResult.RecommendedAccount != nil {
			currentAccount, _ := configManager.GetCurrentAccount()

			// Check if we need to switch
			if currentAccount == nil || currentAccount.Alias != detectionResult.RecommendedAccount.Alias || force {
				fmt.Printf("\n🔄 Switching to account '%s'...\n", detectionResult.RecommendedAccount.Alias)

				// Use the switch command functionality
				if err := switchToAccount(configManager, detectionResult.RecommendedAccount, force); err != nil {
					return fmt.Errorf("failed to switch accounts: %w", err)
				}

				fmt.Printf("✅ Successfully switched to '%s'\n", detectionResult.RecommendedAccount.Alias)
			} else {
				fmt.Printf("\n✅ Already using the recommended account '%s'\n", currentAccount.Alias)
			}

			// Offer to create project configuration if confidence is high
			if detectionResult.Confidence >= 0.8 {
				if err := offerProjectConfiguration(configManager, detectionResult); err != nil {
					if verbose {
						fmt.Printf("⚠️  Failed to create project configuration: %v\n", err)
					}
				}
			}
		} else {
			fmt.Println("\n⚠️  No suitable account found for automatic switching")
		}
	} else {
		fmt.Println("\n🔍 Dry-run mode: No changes made")
		if detectionResult.RecommendedAccount != nil && detectionResult.Confidence >= 0.8 {
			fmt.Printf("💡 Would offer to create project configuration for '%s'\n", detectionResult.RecommendedAccount.Alias)
		}
	}

	return nil
}

// DetectionResult holds the results of account detection
type DetectionResult struct {
	RecommendedAccount *models.Account `json:"recommended_account"`
	Matches            []AccountMatch  `json:"matches"`
	RemoteURL          string          `json:"remote_url"`
	ProjectPath        string          `json:"project_path"`
	Confidence         float64         `json:"confidence"`
}

// AccountMatch represents a potential account match
type AccountMatch struct {
	Account    *models.Account `json:"account"`
	Score      float64         `json:"score"`
	Reasons    []string        `json:"reasons"`
	Confidence string          `json:"confidence"`
}

// performDetection analyzes the current repository and finds matching accounts
func performDetection(gitManager *git.Manager, accounts []*models.Account, verbose bool) (*DetectionResult, error) {
	result := &DetectionResult{
		Matches: []AccountMatch{},
	}

	// Get current directory and remote URL
	currentDir := getCurrentDirectory()
	result.ProjectPath = currentDir

	remoteURL, err := gitManager.GetCurrentRemoteURL("origin")
	if err != nil {
		if verbose {
			fmt.Printf("⚠️  Could not get remote URL: %v\n", err)
		}
	} else {
		result.RemoteURL = remoteURL
		if verbose {
			fmt.Printf("📡 Remote URL: %s\n", remoteURL)
		}
	}

	// Analyze each account for matches
	for _, account := range accounts {
		match := analyzeAccountMatch(account, remoteURL, currentDir, verbose)
		if match.Score > 0 {
			result.Matches = append(result.Matches, match)
		}
	}

	// Sort matches by score (highest first)
	for i := 0; i < len(result.Matches)-1; i++ {
		for j := i + 1; j < len(result.Matches); j++ {
			if result.Matches[i].Score < result.Matches[j].Score {
				result.Matches[i], result.Matches[j] = result.Matches[j], result.Matches[i]
			}
		}
	}

	// Select recommended account
	if len(result.Matches) > 0 && result.Matches[0].Score >= 0.6 {
		result.RecommendedAccount = result.Matches[0].Account
		result.Confidence = result.Matches[0].Score
	}

	return result, nil
}

// analyzeAccountMatch analyzes how well an account matches the current repository
func analyzeAccountMatch(account *models.Account, remoteURL, projectPath string, verbose bool) AccountMatch {
	match := AccountMatch{
		Account: account,
		Score:   0.0,
		Reasons: []string{},
	}

	// Factor 1: Remote URL matching (highest weight)
	if remoteURL != "" && account.GitHubUsername != "" {
		if strings.Contains(remoteURL, account.GitHubUsername) {
			match.Score += 0.8
			match.Reasons = append(match.Reasons, "GitHub username matches remote URL")
		}
	}

	// Factor 2: SSH key accessibility
	if account.SSHKeyPath != "" {
		if isSSHKeyAccessible(account.SSHKeyPath) {
			match.Score += 0.2
			match.Reasons = append(match.Reasons, "SSH key is accessible")
		}
	}

	// Factor 3: Project path patterns (organization/company names)
	if account.GitHubUsername != "" {
		projectName := filepath.Base(projectPath)
		if strings.Contains(strings.ToLower(projectName), strings.ToLower(account.GitHubUsername)) {
			match.Score += 0.3
			match.Reasons = append(match.Reasons, "Project name contains username")
		}
	}

	// Factor 4: Current account preference (small bonus)
	if account.IsDefault {
		match.Score += 0.1
		match.Reasons = append(match.Reasons, "Default account")
	}

	// Factor 5: Recent usage
	if !account.LastUsed.IsZero() {
		match.Score += 0.1
		match.Reasons = append(match.Reasons, "Recently used account")
	}

	// Determine confidence level
	if match.Score >= 0.8 {
		match.Confidence = "Very High"
	} else if match.Score >= 0.6 {
		match.Confidence = "High"
	} else if match.Score >= 0.4 {
		match.Confidence = "Medium"
	} else if match.Score >= 0.2 {
		match.Confidence = "Low"
	} else {
		match.Confidence = "Very Low"
	}

	return match
}

// displayDetectionResults shows the detection analysis results
func displayDetectionResults(result *DetectionResult, verbose bool) {
	fmt.Println("\n🔍 Detection Results:")

	if result.RemoteURL != "" {
		fmt.Printf("  📡 Remote: %s\n", result.RemoteURL)
	}
	fmt.Printf("  📁 Project: %s\n", filepath.Base(result.ProjectPath))

	if len(result.Matches) == 0 {
		fmt.Println("  ❌ No matching accounts found")
		return
	}

	fmt.Printf("\n🎯 Account Matches (%d found):\n", len(result.Matches))

	for i, match := range result.Matches {
		emoji := "⭐"
		if i == 0 && match.Score >= 0.6 {
			emoji = "🥇"
		} else if i == 1 {
			emoji = "🥈"
		} else if i == 2 {
			emoji = "🥉"
		}

		fmt.Printf("  %s %s - Score: %.1f (%s confidence)\n",
			emoji, match.Account.Alias, match.Score, match.Confidence)

		if verbose {
			for _, reason := range match.Reasons {
				fmt.Printf("     • %s\n", reason)
			}
		}
	}

	if result.RecommendedAccount != nil {
		fmt.Printf("\n✅ Recommended: %s (%.1f%% confidence)\n",
			result.RecommendedAccount.Alias, result.Confidence*100)
	}
}

// Helper functions
func getCurrentDirectory() string {
	if dir, err := filepath.Abs("."); err == nil {
		return dir
	}
	return "."
}

func isSSHKeyAccessible(keyPath string) bool {
	// Expand home directory
	if strings.HasPrefix(keyPath, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			keyPath = filepath.Join(home, keyPath[2:])
		}
	}

	// Test if key can be loaded
	cmd := exec.Command("ssh-keygen", "-l", "-f", keyPath)
	return cmd.Run() == nil
}

func switchToAccount(configManager *config.Manager, account *models.Account, force bool) error {
	// Update current account in GitPersona config
	if err := configManager.SetCurrentAccount(account.Alias); err != nil {
		return err
	}

	// Save configuration
	if err := configManager.Save(); err != nil {
		return err
	}

	// Update Git configuration
	gitManager := git.NewManager()
	if err := gitManager.SetUserConfig(account.Name, account.Email); err != nil && !force {
		return err
	}

	return nil
}

func init() {
	autoDetectCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
	autoDetectCmd.Flags().BoolP("force", "f", false, "Force account switch even if already using recommended account")
	autoDetectCmd.Flags().BoolP("verbose", "v", false, "Show detailed analysis information")

	rootCmd.AddCommand(autoDetectCmd)
}

// offerProjectConfiguration offers to create a project configuration file
func offerProjectConfiguration(configManager *config.Manager, result *DetectionResult) error {
	// Check if project config already exists
	projectConfig, err := configManager.LoadProjectConfig(result.ProjectPath)
	if err == nil && projectConfig.Account != "" {
		return nil // Project config already exists
	}

	fmt.Printf("\n💡 High confidence match detected (%.0f%%)\n", result.Confidence*100)
	fmt.Printf("📁 Would you like to save '%s' as the default account for this project?\n", result.RecommendedAccount.Alias)
	fmt.Print("   This will create a .gitpersona.yaml file for automatic switching. (y/N): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		// Handle scan error silently
	}

	if strings.ToLower(strings.TrimSpace(response)) == "y" {
		projectConfig := &models.ProjectConfig{
			Account:   result.RecommendedAccount.Alias,
			CreatedAt: time.Now(),
		}

		if err := configManager.SaveProjectConfig(result.ProjectPath, projectConfig); err != nil {
			return err
		}

		fmt.Printf("✅ Created project configuration: %s/.gitpersona.yaml\n", result.ProjectPath)
		fmt.Println("   Future directory changes will automatically use this account")
		return nil
	}

	fmt.Println("   Skipped project configuration creation")
	return nil
}
