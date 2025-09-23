package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// projectWizardCmd represents the project-wizard command
var projectWizardCmd = &cobra.Command{
	Use:   "project-wizard",
	Short: "🧙 Interactive wizard to configure projects with GitPersona",
	Long: `Interactive wizard to set up GitPersona project configurations.

This wizard will:
- Analyze your current project
- Show account recommendations
- Create project configuration files
- Set up automatic account switching
- Configure multiple projects at once

Examples:
  gitpersona project-wizard                    # Configure current directory
  gitpersona project-wizard /path/to/project   # Configure specific project
  gitpersona project-wizard --scan ~/dev       # Scan and configure multiple projects`,
	Aliases: []string{"wizard", "setup-project"},
	RunE:    runProjectWizardCommand,
}

// runProjectWizardCommand executes the project wizard
func runProjectWizardCommand(cmd *cobra.Command, args []string) error {
	scanDir, _ := cmd.Flags().GetString("scan")
	interactive, _ := cmd.Flags().GetBool("interactive")
	force, _ := cmd.Flags().GetBool("force")

	// Load configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	accounts := configManager.ListAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ No accounts configured")
		fmt.Println("   Add accounts first: gitpersona add-github <username>")
		return nil
	}

	fmt.Println("🧙 GitPersona Project Configuration Wizard")
	fmt.Println("=" + strings.Repeat("=", 45))

	if scanDir != "" {
		return runBulkProjectScan(scanDir, configManager, accounts, force)
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	targetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	return runSingleProjectWizard(targetDir, configManager, accounts, interactive, force)
}

// runSingleProjectWizard configures a single project
func runSingleProjectWizard(projectPath string, configManager *config.Manager, accounts []*models.Account, interactive, force bool) error {
	fmt.Printf("📁 Configuring project: %s\n", filepath.Base(projectPath))
	fmt.Printf("   Path: %s\n", projectPath)

	// Check if config already exists
	existingConfig, err := configManager.LoadProjectConfig(projectPath)
	if err == nil && existingConfig.Account != "" && !force {
		fmt.Printf("✅ Project already configured for account: %s\n", existingConfig.Account)
		if !interactive {
			fmt.Println("   Use --force to reconfigure")
			return nil
		}
		fmt.Print("   Reconfigure? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			return nil
		}
	}

	gitManager := git.NewManager()

	// Check if it's a Git repository
	if !gitManager.IsGitRepository() {
		fmt.Println("⚠️  Not a Git repository")
		if interactive {
			fmt.Print("   Continue anyway? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				return nil
			}
		} else {
			fmt.Println("   Skipping non-Git directory")
			return nil
		}
	}

	// Perform auto-detection
	fmt.Println("\n🔍 Analyzing project for account recommendations...")
	detectionResult, err := performDetection(gitManager, accounts, false)
	if err != nil {
		fmt.Printf("⚠️  Auto-detection failed: %v\n", err)
	}

	// Show results and get user choice
	selectedAccount := selectAccountInteractive(detectionResult, accounts, interactive)
	if selectedAccount == nil {
		fmt.Println("   Skipped project configuration")
		return nil
	}

	// Create project configuration
	projectConfig := &models.ProjectConfig{
		Account:   selectedAccount.Alias,
		CreatedAt: time.Now(),
	}

	if err := configManager.SaveProjectConfig(projectPath, projectConfig); err != nil {
		return fmt.Errorf("failed to save project configuration: %w", err)
	}

	fmt.Printf("✅ Created project configuration: %s/.gitpersona.yaml\n", projectPath)
	fmt.Printf("   Account: %s\n", selectedAccount.Alias)
	fmt.Println("   Future directory changes will automatically use this account")

	return nil
}

// runBulkProjectScan scans and configures multiple projects
func runBulkProjectScan(scanDir string, configManager *config.Manager, accounts []*models.Account, force bool) error {
	fmt.Printf("🔍 Scanning for Git projects in: %s\n", scanDir)

	projects, err := findGitProjects(scanDir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("   No Git projects found")
		return nil
	}

	fmt.Printf("📊 Found %d Git projects\n", len(projects))

	gitManager := git.NewManager()
	var configured, skipped, failed int

	for i, project := range projects {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(projects), filepath.Base(project))

		// Check existing config
		if _, err := configManager.LoadProjectConfig(project); err == nil && !force {
			fmt.Println("   ✅ Already configured (use --force to reconfigure)")
			skipped++
			continue
		}

		// Change to project directory for analysis
		originalDir, _ := os.Getwd()
		os.Chdir(project)

		// Perform detection
		detectionResult, err := performDetection(gitManager, accounts, false)
		if err != nil {
			fmt.Printf("   ⚠️  Detection failed: %v\n", err)
			failed++
			os.Chdir(originalDir)
			continue
		}

		// Auto-configure if high confidence
		if detectionResult.RecommendedAccount != nil && detectionResult.Confidence >= 0.8 {
			projectConfig := &models.ProjectConfig{
				Account:   detectionResult.RecommendedAccount.Alias,
				CreatedAt: time.Now(),
			}

			if err := configManager.SaveProjectConfig(project, projectConfig); err != nil {
				fmt.Printf("   ❌ Failed to save config: %v\n", err)
				failed++
			} else {
				fmt.Printf("   ✅ Configured for account: %s (%.0f%% confidence)\n",
					detectionResult.RecommendedAccount.Alias, detectionResult.Confidence*100)
				configured++
			}
		} else {
			fmt.Println("   ⚠️  Low confidence - manual configuration needed")
			skipped++
		}

		os.Chdir(originalDir)
	}

	fmt.Printf("\n📊 Bulk scan complete:\n")
	fmt.Printf("   ✅ Configured: %d\n", configured)
	fmt.Printf("   ⚠️  Skipped: %d\n", skipped)
	fmt.Printf("   ❌ Failed: %d\n", failed)

	if skipped > 0 {
		fmt.Println("\n💡 For manual configuration of skipped projects:")
		fmt.Println("   gitpersona project-wizard /path/to/project --interactive")
	}

	return nil
}

// selectAccountInteractive helps user select an account
func selectAccountInteractive(detectionResult *DetectionResult, accounts []*models.Account, interactive bool) *models.Account {
	if detectionResult.RecommendedAccount != nil {
		fmt.Printf("\n🎯 Recommended account: %s (%.0f%% confidence)\n",
			detectionResult.RecommendedAccount.Alias, detectionResult.Confidence*100)

		if detectionResult.Confidence >= 0.8 && !interactive {
			fmt.Println("   Using recommended account (high confidence)")
			return detectionResult.RecommendedAccount
		}
	}

	if !interactive {
		if detectionResult.RecommendedAccount != nil && detectionResult.Confidence >= 0.6 {
			return detectionResult.RecommendedAccount
		}
		return nil
	}

	// Interactive selection
	fmt.Println("\n👤 Available accounts:")
	for i, account := range accounts {
		emoji := "  "
		if detectionResult.RecommendedAccount != nil && account.Alias == detectionResult.RecommendedAccount.Alias {
			emoji = "⭐"
		}
		fmt.Printf("   %s[%d] %s - %s <%s>\n", emoji, i+1, account.Alias, account.Name, account.Email)
	}
	fmt.Printf("   [0] Skip configuration\n")

	for {
		fmt.Print("\nSelect account [0-" + strconv.Itoa(len(accounts)) + "]: ")
		var choice string
		fmt.Scanln(&choice)

		if choice == "0" || choice == "" {
			return nil
		}

		if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(accounts) {
			return accounts[idx-1]
		}

		fmt.Println("   Invalid choice. Please try again.")
	}
}

// findGitProjects recursively finds Git projects in a directory
func findGitProjects(rootDir string) ([]string, error) {
	var projects []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		if info.IsDir() && info.Name() == ".git" {
			projectDir := filepath.Dir(path)
			projects = append(projects, projectDir)
			return filepath.SkipDir // Don't go into .git directory
		}

		// Skip hidden directories and common build/cache directories
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") && info.Name() != "." ||
			info.Name() == "node_modules" || info.Name() == "vendor" ||
			info.Name() == "target" || info.Name() == "build") {
			return filepath.SkipDir
		}

		return nil
	})

	return projects, err
}

func init() {
	projectWizardCmd.Flags().String("scan", "", "Scan directory for multiple Git projects")
	projectWizardCmd.Flags().BoolP("interactive", "i", false, "Interactive mode with prompts")
	projectWizardCmd.Flags().BoolP("force", "f", false, "Force reconfiguration of existing projects")

	rootCmd.AddCommand(projectWizardCmd)
}
