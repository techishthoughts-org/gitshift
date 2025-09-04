package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
)

// ValidateGitCommand handles Git configuration validation
type ValidateGitCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	autoFix bool
	verbose bool
}

// NewValidateGitCommand creates a new validate-git command
func NewValidateGitCommand() *ValidateGitCommand {
	cmd := &ValidateGitCommand{
		BaseCommand: commands.NewBaseCommand(
			"validate-git",
			"🔍 Validate and fix Git configuration issues",
			"validate-git",
		).WithExamples(
			"gitpersona validate-git",
			"gitpersona validate-git --auto-fix",
			"gitpersona validate-git --verbose",
		).WithFlags(
			commands.Flag{Name: "auto-fix", Short: "f", Type: "bool", Default: false, Description: "Automatically fix detected issues"},
			commands.Flag{Name: "verbose", Short: "v", Type: "bool", Default: false, Description: "Show detailed information"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *ValidateGitCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.autoFix = c.GetFlagBool(cmd, "auto-fix")
		c.verbose = c.GetFlagBool(cmd, "verbose")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Run executes the validate-git command logic
func (c *ValidateGitCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get Git service
	gitService := container.GetGitService()
	if gitService == nil {
		return fmt.Errorf("git service not available")
	}

	c.PrintInfo(ctx, "🔍 Analyzing Git configuration...")

	// Analyze Git configuration
	if service, ok := gitService.(interface {
		AnalyzeConfiguration(context.Context) (interface{}, error)
	}); ok {
		config, err := service.AnalyzeConfiguration(ctx)
		if err != nil {
			return fmt.Errorf("failed to analyze Git configuration: %w", err)
		}

		// Display analysis results
		if err := c.displayGitAnalysis(ctx, config); err != nil {
			return fmt.Errorf("failed to display analysis: %w", err)
		}

		// Auto-fix if requested
		if c.autoFix {
			if err := c.autoFixGitIssues(ctx, gitService, config); err != nil {
				return fmt.Errorf("failed to auto-fix issues: %w", err)
			}
		}
	} else {
		// Fallback to mock analysis
		c.displayMockGitAnalysis()
	}

	return nil
}

// displayGitAnalysis displays the Git configuration analysis results
func (c *ValidateGitCommand) displayGitAnalysis(ctx context.Context, config interface{}) error {
	// Try to extract information from the config
	if configMap, ok := config.(map[string]interface{}); ok {
		// Display user configuration
		if user, exists := configMap["user"]; exists {
			if userMap, ok := user.(map[string]interface{}); ok {
				c.PrintInfo(ctx, "👤 User Configuration:")
				if name, exists := userMap["name"]; exists {
					c.PrintInfo(ctx, fmt.Sprintf("  name: %v", name))
				}
				if email, exists := userMap["email"]; exists {
					c.PrintInfo(ctx, fmt.Sprintf("  email: %v", email))
				}
			}
		}

		// Display SSH configuration
		if sshCommand, exists := configMap["ssh_command"]; exists {
			c.PrintInfo(ctx, "🔐 SSH Configuration:")
			c.PrintInfo(ctx, fmt.Sprintf("  Command: %v", sshCommand))
		}

		// Display issues
		if issues, exists := configMap["issues"]; exists {
			c.PrintInfo(ctx, "⚠️  Configuration Issues:")
			if issueList, ok := issues.([]map[string]interface{}); ok {
				if len(issueList) > 0 {
					for _, issue := range issueList {
						severity := "⚠️"
						if s, exists := issue["severity"]; exists {
							if s == "high" {
								severity = "🚨"
							} else if s == "low" {
								severity = "ℹ️"
							}
						}

						description := "Unknown issue"
						if d, exists := issue["description"]; exists {
							description = fmt.Sprintf("%v", d)
						}

						c.PrintInfo(ctx, fmt.Sprintf("  %s %s", severity, description))
					}
				} else {
					c.PrintSuccess(ctx, "✅ No configuration issues found")
				}
			}
		}
	}

	return nil
}

// displayMockGitAnalysis displays mock analysis results
func (c *ValidateGitCommand) displayMockGitAnalysis() {
	fmt.Println("📊 Git Configuration Analysis Results:")
	fmt.Println()
	fmt.Println("👤 User Configuration:")
	fmt.Println("  name: Example User")
	fmt.Println("  email: user@example.com")
	fmt.Println()
	fmt.Println("🔐 SSH Configuration:")
	fmt.Println("  Command: ssh -i ~/.ssh/id_example_key -o IdentitiesOnly=yes")
	fmt.Println()
	fmt.Println("✅ No configuration issues found")
	fmt.Println()
	fmt.Println("💡 This is a demo. Install services for full functionality.")
}

// autoFixGitIssues automatically fixes Git configuration issues
func (c *ValidateGitCommand) autoFixGitIssues(ctx context.Context, gitService interface{}, config interface{}) error {
	c.PrintInfo(ctx, "🔧 Auto-fixing Git configuration issues...")

	if service, ok := gitService.(interface {
		FixConfiguration(context.Context, interface{}) error
	}); ok {
		err := service.FixConfiguration(ctx, config)
		if err != nil {
			c.PrintError(ctx, fmt.Sprintf("❌ Failed to auto-fix issues: %v", err))
			return err
		}
		c.PrintSuccess(ctx, "✅ Issues fixed successfully")
	} else {
		c.PrintWarning(ctx, "⚠️  Manual fixes required - auto-fix not available")
	}

	return nil
}

// Validate Git command for integration
var (
	validateGitStandardizedCmd = &cobra.Command{
		Use:     "validate-git",
		Aliases: []string{"vg", "git-check"},
		Short:   "🔍 Validate and fix Git configuration issues",
		Long: `🔍 Validate and Fix Git Configuration Issues

This command analyzes your Git configuration for common problems:
- Duplicate SSH command configurations
- Wrong SSH key references
- Missing user configuration
- Credential helper conflicts
- Remote repository issues

Examples:
  gitpersona validate-git              # Analyze configuration
  gitpersona validate-git --auto-fix   # Analyze and auto-fix issues
  gitpersona validate-git --verbose    # Show detailed information`,
		Args: cobra.NoArgs,
		RunE: runValidateGitStandardized,
	}
)

func init() {
	rootCmd.AddCommand(validateGitStandardizedCmd)
}

// runValidateGitStandardized runs the validate-git command
func runValidateGitStandardized(cmd *cobra.Command, args []string) error {
	// Create and run the validate-git command
	validateCmd := NewValidateGitCommand()
	ctx := context.Background()
	return validateCmd.Execute(ctx, args)
}
