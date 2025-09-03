package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/container"
)

// validateGitConfiguration validates Git configuration
func validateGitConfiguration(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("🔍 Analyzing Git configuration...")

	// Get the service container
	serviceContainer := container.GetGlobalSimpleContainer()
	if serviceContainer == nil {
		return fmt.Errorf("service container not available")
	}

	// Get Git service
	gitService := serviceContainer.GetGitService()
	if gitService == nil {
		fmt.Println("⚠️  Git service not available - showing example analysis")
		displayMockGitAnalysis()
		return nil
	}

	// Analyze configuration
	if service, ok := gitService.(interface {
		AnalyzeConfiguration(context.Context) (interface{}, error)
	}); ok {
		config, err := service.AnalyzeConfiguration(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to analyze Git configuration: %v\n", err)
			return err
		}

		displayGitAnalysisResults(config)

		// Auto-fix if requested
		autoFix, _ := cmd.Flags().GetBool("auto-fix")
		if autoFix {
			return autoFixGitIssues(ctx, gitService, config)
		}
	} else {
		// Fallback to mock analysis
		displayMockGitAnalysis()
	}

	return nil
}

func displayGitAnalysisResults(config interface{}) {
	fmt.Println("📊 Git Configuration Analysis Results:")
	fmt.Println()

	if configMap, ok := config.(map[string]interface{}); ok {
		// Display user configuration
		if user, exists := configMap["user"]; exists {
			fmt.Println("👤 User Configuration:")
			if userMap, ok := user.(map[string]interface{}); ok {
				for key, value := range userMap {
					fmt.Printf("  %s: %v\n", key, value)
				}
			}
			fmt.Println()
		}

		// Display issues
		if issues, exists := configMap["issues"]; exists {
			fmt.Println("⚠️  Configuration Issues:")
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

						fmt.Printf("  %s %s\n", severity, description)
					}
				} else {
					fmt.Println("✅ No configuration issues found")
				}
			}
		}
	}
}

func displayMockGitAnalysis() {
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

func autoFixGitIssues(ctx context.Context, gitService interface{}, config interface{}) error {
	fmt.Println("🔧 Auto-fixing Git configuration issues...")

	if service, ok := gitService.(interface {
		FixConfiguration(context.Context, interface{}) error
	}); ok {
		err := service.FixConfiguration(ctx, config)
		if err != nil {
			fmt.Printf("❌ Failed to auto-fix issues: %v\n", err)
			return err
		}
		fmt.Println("✅ Issues fixed successfully")
	} else {
		fmt.Println("⚠️  Manual fixes required - auto-fix not available")
	}

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
		RunE: validateGitConfiguration,
	}

	validateGitFlags = struct {
		autoFix bool
		verbose bool
	}{}
)

func init() {
	validateGitCmd.Flags().BoolVarP(&validateGitFlags.autoFix, "auto-fix", "f", false, "Automatically fix detected issues")
	validateGitCmd.Flags().BoolVarP(&validateGitFlags.verbose, "verbose", "v", false, "Show detailed information")

	rootCmd.AddCommand(validateGitCmd)
}
