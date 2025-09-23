package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project-specific GitHub account configurations",
	Long: `Manage project-specific GitHub account configurations.

Project configurations allow you to automatically use different GitHub accounts
for different projects by creating a .gitpersona.yaml file in the project root.

Examples:
  gitpersona project set work
  gitpersona project show
  gitpersona project remove
  gitpersona project list
  gitpersona project detect`,
}

// projectSetCmd sets the account for the current project
var projectSetCmd = &cobra.Command{
	Use:   "set [alias]",
	Short: "Set the GitHub account for the current project",
	Long: `Set the GitHub account to use for the current project.

This creates a .gitpersona.yaml file in the current directory that specifies
which account should be used when working in this project.

Examples:
  gitpersona project set work
  gitpersona project set personal`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Verify the account exists
		account, err := configManager.GetAccount(alias)
		if err != nil {
			return fmt.Errorf("account '%s' not found. Use 'gitpersona list' to see available accounts", alias)
		}

		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Create project configuration
		projectConfig := models.NewProjectConfig(alias)

		// Save project configuration
		if err := configManager.SaveProjectConfig(currentDir, projectConfig); err != nil {
			return fmt.Errorf("failed to save project configuration: %w", err)
		}

		fmt.Printf("✅ Project configured to use account '%s'\n", alias)
		fmt.Printf("   Account: %s (%s - %s)\n", account.Alias, account.Name, account.Email)
		fmt.Printf("   Configuration saved to: .gitpersona.yaml\n")
		fmt.Printf("\n💡 Run 'eval \"$(gitpersona init)\"' in your shell to enable automatic switching\n")

		return nil
	},
}

// projectShowCmd shows the current project configuration
var projectShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current project configuration",
	Long: `Show the GitHub account configuration for the current project.

This displays the contents of the .gitpersona.yaml file if it exists,
along with the account details.

Examples:
  gitpersona project show`,
	Aliases: []string{"info", "status"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Load project configuration
		projectConfig, err := configManager.LoadProjectConfig(currentDir)
		if err != nil {
			fmt.Printf("No project configuration found in: %s\n", currentDir)
			fmt.Printf("Use 'gitpersona project set <alias>' to configure this project\n")
			return nil
		}

		// Get account details
		account, err := configManager.GetAccount(projectConfig.Account)
		if err != nil {
			fmt.Printf("❌ Project configured for account '%s', but account not found\n", projectConfig.Account)
			fmt.Printf("Use 'gitpersona list' to see available accounts\n")
			return nil
		}

		fmt.Printf("📁 Project Configuration\n")
		fmt.Printf("   Directory: %s\n", currentDir)
		fmt.Printf("   Account: %s\n", account.Alias)
		fmt.Printf("   Name: %s\n", account.Name)
		fmt.Printf("   Email: %s\n", account.Email)
		if account.GitHubUsername != "" {
			fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
		}
		if account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.SSHKeyPath)
		}
		if account.Description != "" {
			fmt.Printf("   Description: %s\n", account.Description)
		}
		fmt.Printf("   Configured: %s\n", formatTime(projectConfig.CreatedAt))

		return nil
	},
}

// projectRemoveCmd removes the project configuration
var projectRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove the project configuration",
	Long: `Remove the GitHub account configuration for the current project.

This deletes the .gitpersona.yaml file from the current directory.

Examples:
  gitpersona project remove`,
	Aliases: []string{"rm", "delete", "unset"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current directory
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		configFile := fmt.Sprintf("%s/.gitpersona.yaml", currentDir)

		// Check if file exists
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			fmt.Printf("No project configuration found in: %s\n", currentDir)
			return nil
		}

		// Ask for confirmation unless --force flag is used
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to remove the project configuration? [y/N]: ")
			confirmation := promptForInput("")
			if confirmation != "y" && confirmation != "Y" && confirmation != "yes" && confirmation != "Yes" {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		// Remove the file
		if err := os.Remove(configFile); err != nil {
			return fmt.Errorf("failed to remove project configuration: %w", err)
		}

		fmt.Printf("✅ Project configuration removed\n")
		fmt.Printf("   Deleted: %s\n", configFile)

		return nil
	},
}

// projectListCmd lists all configured projects
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured projects",
	Long: `List all projects that have GitPersona configurations.

This command scans for .gitpersona.yaml files and shows all configured projects.

Examples:
  gitpersona project list
  gitpersona project list --path ~/dev`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		scanPath, _ := cmd.Flags().GetString("path")
		if scanPath == "" {
			scanPath = "."
		}

		fmt.Printf("📁 Scanning for configured projects in: %s\n", scanPath)

		projects, err := findConfiguredProjects(scanPath)
		if err != nil {
			return fmt.Errorf("failed to scan for projects: %w", err)
		}

		if len(projects) == 0 {
			fmt.Println("   No configured projects found")
			fmt.Println("\n💡 Use 'gitpersona project-wizard --scan' to configure projects")
			return nil
		}

		fmt.Printf("📈 Found %d configured projects:\n\n", len(projects))

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		for _, project := range projects {
			projectConfig, err := configManager.LoadProjectConfig(project.Path)
			if err != nil {
				fmt.Printf("❌ %s - Failed to load config\n", project.Name)
				continue
			}

			account, err := configManager.GetAccount(projectConfig.Account)
			if err != nil {
				fmt.Printf("⚠️  %s - Account '%s' not found\n", project.Name, projectConfig.Account)
				continue
			}

			fmt.Printf("✅ %s\n", project.Name)
			fmt.Printf("   Account: %s (%s)\n", account.Alias, account.Email)
			fmt.Printf("   Path: %s\n", project.Path)
			fmt.Printf("   Configured: %s\n\n", formatTime(projectConfig.CreatedAt))
		}

		return nil
	},
}

// projectDetectCmd detects the best account for current project
var projectDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect the best account for the current project",
	Long: `Analyze the current project and recommend the best GitHub account.

This command performs the same analysis as auto-detect but only shows
recommendations without making changes.

Examples:
  gitpersona project detect
  gitpersona project detect --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("❌ No accounts configured")
			return nil
		}

		gitManager := git.NewManager()
		if !gitManager.IsGitRepository() {
			fmt.Println("❌ Not in a Git repository")
			return nil
		}

		detectionResult, err := performDetection(gitManager, accounts, verbose)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}

		displayDetectionResults(detectionResult, verbose)

		if detectionResult.RecommendedAccount != nil {
			fmt.Printf("\n💡 To configure this project:\n")
			fmt.Printf("   gitpersona project set %s\n", detectionResult.RecommendedAccount.Alias)
		}

		return nil
	},
}

// ProjectInfo represents a configured project
type ProjectInfo struct {
	Name string
	Path string
}

// findConfiguredProjects finds all projects with .gitpersona.yaml files
func findConfiguredProjects(rootDir string) ([]ProjectInfo, error) {
	var projects []ProjectInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		if !info.IsDir() && info.Name() == ".gitpersona.yaml" {
			projectDir := filepath.Dir(path)
			projectName := filepath.Base(projectDir)
			projects = append(projects, ProjectInfo{
				Name: projectName,
				Path: projectDir,
			})
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
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectSetCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectRemoveCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectDetectCmd)

	projectRemoveCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
	projectListCmd.Flags().String("path", ".", "Path to scan for configured projects")
	projectDetectCmd.Flags().BoolP("verbose", "v", false, "Show detailed analysis")
}
