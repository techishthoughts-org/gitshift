package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// diagnoseUnifiedCmd consolidates all diagnostic and troubleshooting operations
var diagnoseUnifiedCmd = &cobra.Command{
	Use:   "diagnose [mode]",
	Short: "🏥 Unified diagnostics and troubleshooting",
	Long: `Complete system diagnostics with multiple modes and comprehensive analysis:

🔍 DIAGNOSTIC MODES:
  basic               - Basic system health check (default)
  full                - Comprehensive diagnostic suite
  health              - Quick health status check
  ssh                 - SSH-specific diagnostics
  git                 - Git configuration diagnostics
  github              - GitHub connectivity diagnostics

🛠️ TROUBLESHOOTING:
  issues              - Show all detected issues
  fix                 - Auto-fix detected issues
  doctor              - Interactive troubleshooting guide

📊 REPORTING:
  report              - Generate diagnostic report
  export [file]       - Export diagnostics to file

Examples:
  gitpersona diagnose              # Basic diagnostic
  gitpersona diagnose full         # Comprehensive diagnostics
  gitpersona diagnose ssh          # SSH-only diagnostics
  gitpersona diagnose fix --auto   # Auto-fix all issues
  gitpersona diagnose report --json

Use 'gitpersona diagnose [mode] --help' for detailed information.`,
	Args: cobra.RangeArgs(0, 1),
	RunE: diagnoseHandler,
}

// Diagnostic subcommands
var (
	diagnoseBasicCmd = &cobra.Command{
		Use:   "basic",
		Short: "Basic system health check",
		Long:  `Perform basic system health checks covering essential functionality.`,
		RunE:  diagnoseBasicHandler,
	}

	diagnoseFullCmd = &cobra.Command{
		Use:   "full",
		Short: "Comprehensive diagnostic suite",
		Long:  `Run complete diagnostic suite covering all system components.`,
		RunE:  diagnoseFullHandler,
	}

	diagnoseHealthCmd = &cobra.Command{
		Use:   "health",
		Short: "Quick health status check",
		Long:  `Quick health status check with pass/fail summary.`,
		RunE:  diagnoseHealthHandler,
	}

	diagnoseSSHCmd = &cobra.Command{
		Use:   "ssh",
		Short: "SSH-specific diagnostics",
		Long:  `Run diagnostics focused on SSH configuration and connectivity.`,
		RunE:  diagnoseSSHHandler,
	}

	diagnoseGitCmd = &cobra.Command{
		Use:   "git",
		Short: "Git configuration diagnostics",
		Long:  `Run diagnostics focused on Git configuration and repository setup.`,
		RunE:  diagnoseGitHandler,
	}

	diagnoseGitHubCmd = &cobra.Command{
		Use:   "github",
		Short: "GitHub connectivity diagnostics",
		Long:  `Run diagnostics focused on GitHub API and SSH connectivity.`,
		RunE:  diagnoseGitHubHandler,
	}

	diagnoseIssuesCmd = &cobra.Command{
		Use:   "issues",
		Short: "Show all detected issues",
		Long:  `Display all currently detected issues with severity levels.`,
		RunE:  diagnoseIssuesHandler,
	}

	diagnoseFixCmd = &cobra.Command{
		Use:   "fix",
		Short: "Auto-fix detected issues",
		Long:  `Automatically fix detected issues where possible.`,
		RunE:  diagnoseFixHandler,
	}

	diagnoseDoctorCmd = &cobra.Command{
		Use:   "doctor",
		Short: "Interactive troubleshooting guide",
		Long:  `Interactive troubleshooting guide with step-by-step assistance.`,
		RunE:  diagnoseDoctorHandler,
	}

	diagnoseReportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate diagnostic report",
		Long:  `Generate comprehensive diagnostic report with all findings.`,
		RunE:  diagnoseReportHandler,
	}

	diagnoseExportCmd = &cobra.Command{
		Use:   "export [file]",
		Short: "Export diagnostics to file",
		Long:  `Export diagnostic results to file in specified format.`,
		Args:  cobra.ExactArgs(1),
		RunE:  diagnoseExportHandler,
	}
)

func init() {
	rootCmd.AddCommand(diagnoseUnifiedCmd)

	// Add subcommands
	diagnoseUnifiedCmd.AddCommand(diagnoseBasicCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseFullCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseHealthCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseSSHCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseGitCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseGitHubCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseIssuesCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseFixCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseDoctorCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseReportCmd)
	diagnoseUnifiedCmd.AddCommand(diagnoseExportCmd)

	// Global flags
	diagnoseUnifiedCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	diagnoseUnifiedCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	diagnoseUnifiedCmd.PersistentFlags().Bool("quiet", false, "Suppress non-critical output")
	diagnoseUnifiedCmd.PersistentFlags().StringP("account", "a", "", "Target specific account")

	// Specific flags
	diagnoseFullCmd.Flags().Bool("parallel", true, "Run checks in parallel")
	diagnoseFullCmd.Flags().Int("timeout", 300, "Timeout for full diagnostics (seconds)")

	diagnoseFixCmd.Flags().Bool("auto", false, "Apply fixes automatically without confirmation")
	diagnoseFixCmd.Flags().Bool("force", false, "Force fixes even for dangerous operations")
	diagnoseFixCmd.Flags().StringSlice("types", []string{}, "Fix only specific issue types")

	diagnoseReportCmd.Flags().String("format", "text", "Report format (text, json, html)")
	diagnoseReportCmd.Flags().StringP("output", "o", "", "Output file for report")

	diagnoseExportCmd.Flags().String("format", "json", "Export format (json, yaml, csv)")
}

// Main handler - routes to basic diagnostics if no subcommand
func diagnoseHandler(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Default to basic diagnostics
		return diagnoseBasicHandler(cmd, args)
	}

	mode := args[0]
	switch mode {
	case "basic":
		return diagnoseBasicHandler(cmd, args[1:])
	case "full":
		return diagnoseFullHandler(cmd, args[1:])
	case "health":
		return diagnoseHealthHandler(cmd, args[1:])
	case "ssh":
		return diagnoseSSHHandler(cmd, args[1:])
	case "git":
		return diagnoseGitHandler(cmd, args[1:])
	case "github":
		return diagnoseGitHubHandler(cmd, args[1:])
	case "fix":
		return diagnoseFixHandler(cmd, args[1:])
	case "issues":
		return diagnoseIssuesHandler(cmd, args[1:])
	case "doctor":
		return diagnoseDoctorHandler(cmd, args[1:])
	case "report":
		return diagnoseReportHandler(cmd, args[1:])
	default:
		return fmt.Errorf("unknown diagnostic mode: %s", mode)
	}
}

// Handler implementations

func diagnoseIssuesHandler(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	fmt.Printf("⚠️  Showing detected issues (json: %v)\n", jsonOutput)

	// TODO: Implement issue listing from all services
	return fmt.Errorf("not implemented yet - will aggregate issues from all services")
}

func diagnoseDoctorHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("👨‍⚕️ Starting interactive troubleshooting guide")

	// TODO: Implement interactive troubleshooting
	return fmt.Errorf("not implemented yet - will implement interactive doctor")
}

func diagnoseReportHandler(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")

	fmt.Printf("📊 Generating diagnostic report (format: %s, output: %s)\n", format, outputFile)

	// TODO: Implement comprehensive reporting
	return fmt.Errorf("not implemented yet - will generate comprehensive report")
}

func diagnoseExportHandler(cmd *cobra.Command, args []string) error {
	filename := args[0]
	format, _ := cmd.Flags().GetString("format")

	fmt.Printf("💾 Exporting diagnostics to: %s (format: %s)\n", filename, format)

	// TODO: Implement diagnostic export
	return fmt.Errorf("not implemented yet - will export diagnostic data")
}

// Hide old diagnostic commands during transition
func init() {
	// Hide legacy commands to reduce noise
	hideCommand("troubleshoot")
	hideCommand("health")
	hideCommand("doctor")
}
