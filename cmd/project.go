package cmd

import (
	"fmt"
	"os"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/spf13/cobra"
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
  gitpersona project remove`,
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

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectSetCmd)
	projectCmd.AddCommand(projectShowCmd)
	projectCmd.AddCommand(projectRemoveCmd)

	projectRemoveCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}
