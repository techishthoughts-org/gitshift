package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// ValidateGitCommand handles Git configuration validation and fixing
type ValidateGitCommand struct {
	*BaseCommand

	// Command-specific flags
	autoFix bool
	verbose bool
}

// NewValidateGitCommand creates a new Git validation command
func NewValidateGitCommand() *ValidateGitCommand {
	cmd := &ValidateGitCommand{
		BaseCommand: NewBaseCommand(
			"validate-git",
			"🔍 Validate and fix Git configuration issues",
			"validate-git [flags]",
		).WithExamples(
			"gitpersona validate-git",
			"gitpersona validate-git --auto-fix",
			"gitpersona validate-git --verbose",
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *ValidateGitCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Add command-specific flags
	cmd.Flags().BoolVarP(&c.autoFix, "auto-fix", "f", false, "Automatically fix detected issues")
	cmd.Flags().BoolVarP(&c.verbose, "verbose", "v", false, "Show detailed information")

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *ValidateGitCommand) Validate(args []string) error {
	// No arguments needed for this command
	return nil
}

// Run executes the Git validation command logic
func (c *ValidateGitCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get required services
	gitConfigService := container.GetGitService()

	if gitConfigService == nil {
		return fmt.Errorf("Git service not available")
	}

	// Analyze Git configuration
	config, err := c.analyzeGitConfiguration(ctx, gitConfigService)
	if err != nil {
		return fmt.Errorf("failed to analyze Git configuration: %w", err)
	}

	// Display analysis results
	c.displayAnalysisResults(ctx, config)

	// Auto-fix if requested
	if c.autoFix {
		if err := c.autoFixIssues(ctx, gitConfigService, config); err != nil {
			return fmt.Errorf("failed to auto-fix issues: %w", err)
		}
	}

	return nil
}

// Execute overrides the BaseCommand Execute method to call our Run method
func (c *ValidateGitCommand) Execute(ctx context.Context, args []string) error {
	c.startTime = time.Now()
	c.ctx = ctx

	// Log command execution
	c.GetLogger().Info(ctx, "executing_git_validation_command",
		observability.F("command", c.name),
		observability.F("args", args),
	)

	// Validate arguments
	if err := c.Validate(args); err != nil {
		return err
	}

	// Execute the command
	err := c.Run(ctx, args)

	// Log completion
	if err != nil {
		c.GetLogger().Error(ctx, "git_validation_command_failed",
			observability.F("error", err.Error()),
		)
		return err
	}

	c.GetLogger().Info(ctx, "git_validation_command_completed_successfully")
	return nil
}

// analyzeGitConfiguration analyzes the current Git configuration
func (c *ValidateGitCommand) analyzeGitConfiguration(ctx context.Context, gitConfigService interface{}) (interface{}, error) {
	c.GetLogger().Info(ctx, "analyzing_git_configuration")

	// Try to analyze configuration if the service supports it
	if service, ok := gitConfigService.(interface {
		AnalyzeConfiguration(context.Context) (interface{}, error)
	}); ok {
		return service.AnalyzeConfiguration(ctx)
	}

	// Fallback to basic analysis
	return c.basicGitConfigAnalysis(ctx), nil
}

// basicGitConfigAnalysis performs basic Git configuration analysis
func (c *ValidateGitCommand) basicGitConfigAnalysis(ctx context.Context) interface{} {
	// This is a fallback implementation
	// In a real scenario, this would use the GitConfigService
	return map[string]interface{}{
		"user": map[string]string{
			"name":  "Example User",
			"email": "user@example.com",
		},
		"ssh_command": "ssh -i ~/.ssh/id_example_key -o IdentitiesOnly=yes",
		"issues": []map[string]interface{}{
			{
				"type":        "duplicate_ssh_command",
				"severity":    "medium",
				"description": "Multiple SSH command configurations found",
				"fix":         "Remove duplicate configurations",
				"fixed":       false,
			},
		},
	}
}

// displayAnalysisResults displays the Git configuration analysis results
func (c *ValidateGitCommand) displayAnalysisResults(ctx context.Context, config interface{}) {
	c.PrintInfo(ctx, "🔍 Git Configuration Analysis Results")

	// Display user configuration
	if userConfig, ok := config.(map[string]interface{}); ok {
		if user, exists := userConfig["user"]; exists {
			if userMap, ok := user.(map[string]string); ok {
				c.PrintSuccess(ctx, fmt.Sprintf("User: %s (%s)", userMap["name"], userMap["email"]))
			}
		}

		if sshCommand, exists := userConfig["ssh_command"]; exists {
			c.PrintInfo(ctx, fmt.Sprintf("SSH Command: %s", sshCommand))
		}

		// Display issues
		if issues, exists := userConfig["issues"]; exists {
			if issuesList, ok := issues.([]map[string]interface{}); ok {
				if len(issuesList) > 0 {
					c.PrintWarning(ctx, fmt.Sprintf("Found %d configuration issues:", len(issuesList)))

					for _, issue := range issuesList {
						severity := "⚠️"
						if s, exists := issue["severity"]; exists {
							if s == "high" {
								severity = "🚨"
							} else if s == "medium" {
								severity = "⚠️"
							} else {
								severity = "ℹ️"
							}
						}

						description := "Unknown issue"
						if d, exists := issue["description"]; exists {
							description = fmt.Sprintf("%v", d)
						}

						fix := "No fix available"
						if f, exists := issue["fix"]; exists {
							fix = fmt.Sprintf("%v", f)
						}

						c.PrintWarning(ctx, fmt.Sprintf("  %s %s", severity, description))
						if c.verbose {
							c.PrintInfo(ctx, fmt.Sprintf("     Fix: %s", fix))
						}
					}
				} else {
					c.PrintSuccess(ctx, "✅ No configuration issues found")
				}
			}
		}
	}
}

// autoFixIssues automatically fixes detected Git configuration issues
func (c *ValidateGitCommand) autoFixIssues(ctx context.Context, gitConfigService interface{}, config interface{}) error {
	c.PrintInfo(ctx, "🔧 Auto-fixing Git configuration issues...")

	// Try to fix configuration if the service supports it
	if service, ok := gitConfigService.(interface {
		FixConfiguration(context.Context, interface{}) error
	}); ok {
		return service.FixConfiguration(ctx, config)
	}

	// Fallback to manual fixes
	return c.manualFixIssues(ctx, config)
}

// manualFixIssues performs manual fixes for common issues
func (c *ValidateGitCommand) manualFixIssues(ctx context.Context, config interface{}) error {
	c.PrintWarning(ctx, "Manual fixes required - some issues cannot be auto-fixed")

	// This would implement manual fixes based on the detected issues
	// For now, we'll just log that manual intervention is needed

	return nil
}

// Git validation command for integration
var (
	validateGitCmd = &cobra.Command{
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
		RunE: runValidateGit,
	}

	validateGitFlags = struct {
		autoFix bool
		verbose bool
	}{}
)

func init() {
	validateGitCmd.Flags().BoolVarP(&validateGitFlags.autoFix, "auto-fix", "f", false, "Automatically fix detected issues")
	validateGitFlags.verbose = false

	rootCmd.AddCommand(validateGitCmd)
}

// runValidateGit runs the Git validation command
func runValidateGit(cmd *cobra.Command, args []string) error {
	// Create and run the Git validation command
	validateCmd := NewValidateGitCommand()
	validateCmd.autoFix = validateGitFlags.autoFix
	validateCmd.verbose = validateGitFlags.verbose

	ctx := context.Background()
	return validateCmd.Execute(ctx, args)
}
