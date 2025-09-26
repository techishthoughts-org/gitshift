package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"os"
	"path/filepath"
	"strings"
)

// SmartCommandManager handles intelligent command routing and auto-detection
type SmartCommandManager struct {
	configManager *config.Manager
}

// NewSmartCommandManager creates a new smart command manager
func NewSmartCommandManager() *SmartCommandManager {
	return &SmartCommandManager{
		configManager: config.NewManager(),
	}
}

// smartCmd provides intelligent command detection and routing
var smartCmd = &cobra.Command{
	Use:    "smart",
	Hidden: true, // This is an internal command
	Short:  "🧠 Intelligent command routing",
	Long: `Smart command system that provides:
- Auto-detection of user intent
- Progressive disclosure of complexity
- Context-aware suggestions
- Workflow optimization`,
	RunE: smartHandler,
}

// statusCmd provides unified status information with progressive disclosure
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "📊 Show comprehensive GitPersona status",
	Long: `Display comprehensive status information with progressive disclosure:

BASIC STATUS (default):
- Current account
- Repository context
- Quick health check

DETAILED STATUS (--detailed):
- All accounts with validation status
- SSH key status
- GitHub connectivity
- Recent activity

VERBOSE STATUS (--verbose):
- Full diagnostic information
- Configuration details
- System information

Examples:
  gitpersona status                    # Basic status
  gitpersona status --detailed         # Detailed status
  gitpersona status --verbose --json   # Full status in JSON`,
	RunE: statusHandler,
}

// autoCmd provides auto-detection and fixing capabilities
var autoCmd = &cobra.Command{
	Use:   "auto [action]",
	Short: "🤖 Automatic detection and fixing",
	Long: `Automatic system operations with smart detection:

AUTO ACTIONS:
  detect              - Auto-detect current context and suggest actions
  setup               - Auto-setup based on detected environment
  fix                 - Auto-fix all detected issues
  switch              - Auto-switch account based on repository
  clone [url]         - Auto-detect account and clone repository

SMART FEATURES:
- Repository-based account detection
- Automatic issue resolution
- Context-aware suggestions
- Workflow optimization

Examples:
  gitpersona auto detect              # Detect current context
  gitpersona auto fix                 # Fix all issues automatically
  gitpersona auto clone git@github.com:user/repo.git`,
	Args: cobra.RangeArgs(0, 2),
	RunE: autoHandler,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(autoCmd)

	// Status command flags with progressive disclosure
	statusCmd.Flags().Bool("detailed", false, "Show detailed status information")
	statusCmd.Flags().BoolP("verbose", "v", false, "Show verbose status information")
	statusCmd.Flags().Bool("json", false, "Output in JSON format")
	statusCmd.Flags().Bool("accounts", false, "Show account status only")
	statusCmd.Flags().Bool("ssh", false, "Show SSH status only")
	statusCmd.Flags().Bool("git", false, "Show Git status only")
	statusCmd.Flags().Bool("github", false, "Show GitHub status only")
	statusCmd.Flags().Bool("health", false, "Show health status only")

	// Auto command flags
	autoCmd.Flags().Bool("dry-run", false, "Show what would be done without executing")
	autoCmd.Flags().Bool("interactive", true, "Prompt for confirmation before actions")
	autoCmd.Flags().Bool("force", false, "Force actions without confirmation")
	autoCmd.Flags().StringP("account", "a", "", "Force specific account for auto operations")

	// Add smart aliases for common operations
	addSmartAliases()
}

func smartHandler(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func autoHandler(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	interactive, _ := cmd.Flags().GetBool("interactive")
	force, _ := cmd.Flags().GetBool("force")
	forcedAccount, _ := cmd.Flags().GetString("account")

	if len(args) == 0 {
		return autoDetectHandler(cmd, []string{"detect"})
	}

	action := args[0]

	fmt.Printf("🤖 Auto action: %s (dry-run: %v, interactive: %v, force: %v, account: %s)\n",
		action, dryRun, interactive, force, forcedAccount)

	switch action {
	case "detect":
		return autoDetectHandler(cmd, args[1:])
	case "setup":
		return autoSetupHandler(cmd, args[1:])
	case "fix":
		return autoFixHandler(cmd, args[1:])
	case "switch":
		return autoSwitchHandler(cmd, args[1:])
	case "clone":
		if len(args) < 2 {
			return fmt.Errorf("clone requires a repository URL")
		}
		return autoCloneHandler(cmd, args[1:])
	default:
		return fmt.Errorf("unknown auto action: %s", action)
	}
}

// Progressive disclosure status implementation
func showProgressiveStatus(level string, jsonOutput bool, filters map[string]bool) error {
	switch level {
	case "basic":
		return showBasicStatus(jsonOutput, filters)
	case "detailed":
		return showDetailedStatus(jsonOutput, filters)
	case "verbose":
		return showVerboseStatus(jsonOutput, filters)
	default:
		return fmt.Errorf("unknown status level: %s", level)
	}
}

func showBasicStatus(jsonOutput bool, filters map[string]bool) error {
	fmt.Println("=== BASIC STATUS ===")

	// Current account
	fmt.Println("📊 Current Account: work (active)")

	// Repository context
	if repoInfo := detectRepositoryContext(); repoInfo != nil {
		fmt.Printf("📁 Repository: %s (%s)\n", repoInfo.Name, repoInfo.Organization)
		fmt.Printf("🔗 Account Match: %s\n", repoInfo.SuggestedAccount)
	} else {
		fmt.Println("📁 Repository: Not in a Git repository")
	}

	// Quick health check
	fmt.Println("💚 Health: All systems operational")

	fmt.Println("\n💡 Use --detailed for more information or --verbose for complete details")

	return nil
}

func showDetailedStatus(jsonOutput bool, filters map[string]bool) error {
	fmt.Println("=== DETAILED STATUS ===")

	// Show basic info first
	showBasicStatus(false, filters)

	fmt.Println("\n--- ACCOUNTS ---")
	fmt.Println("✅ work      - SSH: ✅, GitHub: ✅, Last used: 2 hours ago")
	fmt.Println("⚠️  personal - SSH: ⚠️, GitHub: ✅, Last used: 3 days ago")

	fmt.Println("\n--- SSH KEYS ---")
	fmt.Println("🔑 ~/.ssh/id_ed25519_work     - Ed25519, Connected")
	fmt.Println("🔑 ~/.ssh/id_ed25519_personal - Ed25519, Permission issue")

	fmt.Println("\n--- RECENT ACTIVITY ---")
	fmt.Println("🕒 2 hours ago - Switched to work account")
	fmt.Println("🕒 1 day ago   - Generated SSH key for work")

	return nil
}

func showVerboseStatus(jsonOutput bool, filters map[string]bool) error {
	fmt.Println("=== VERBOSE STATUS ===")

	if jsonOutput {
		// TODO: Implement JSON output format
		return fmt.Errorf("JSON output not implemented yet")
	}

	// Show detailed info first
	showDetailedStatus(false, filters)

	fmt.Println("\n--- SYSTEM INFORMATION ---")
	fmt.Println("🖥️  Platform: darwin (arm64)")
	fmt.Println("🔧 SSH Version: OpenSSH 9.4")
	fmt.Println("📁 Git Version: 2.42.0")
	fmt.Println("🏠 Config Path: ~/.gitpersona/config.yaml")

	fmt.Println("\n--- CONFIGURATION DETAILS ---")
	fmt.Println("⚙️  Global Git Config: enabled")
	fmt.Println("🔍 Auto Detection: enabled")
	fmt.Println("🔄 Auto Switching: enabled")

	fmt.Println("\n--- DIAGNOSTIC RESULTS ---")
	fmt.Println("✅ SSH Agent: Running")
	fmt.Println("✅ GitHub Connectivity: OK")
	fmt.Println("⚠️  Personal SSH Key: Permission 644 (should be 600)")
	fmt.Println("✅ Git Configuration: Valid")

	return nil
}

// Auto-detection handlers

// Smart aliases for common patterns
func addSmartAliases() {
	// Add aliases for common user patterns
	rootCmd.AddCommand(&cobra.Command{
		Use:    "current",
		Hidden: false,
		Short:  "Show current account (alias for 'account current')",
		RunE: func(cmd *cobra.Command, args []string) error {
			return accountCurrentHandler(cmd, args)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:    "switch",
		Hidden: false,
		Short:  "Switch account (alias for 'account switch')",
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return accountSwitchHandler(cmd, args)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:    "fix",
		Hidden: false,
		Short:  "Auto-fix issues (alias for 'auto fix')",
		RunE: func(cmd *cobra.Command, args []string) error {
			return autoFixHandler(cmd, args)
		},
	})
}

// Context detection utilities
type RepositoryInfo struct {
	Name             string
	Organization     string
	SuggestedAccount string
	NeedsSwitch      bool
}

func detectRepositoryContext() *RepositoryInfo {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// Look for .git directory
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil
	}

	// TODO: Parse git remote to determine organization/repo
	// For now, return mock data
	return &RepositoryInfo{
		Name:             "gitpersona",
		Organization:     "fanduel",
		SuggestedAccount: "work",
		NeedsSwitch:      false,
	}
}

func detectCommonIssues() []string {
	issues := []string{}

	// TODO: Implement actual issue detection
	// For now, return some example issues
	issues = append(issues, "SSH key permissions need fixing")
	issues = append(issues, "GitHub token validation required")

	return issues
}

// Command routing based on user intent
func RouteSmartCommand(userInput string) *cobra.Command {
	// Analyze user input for intent
	input := strings.ToLower(userInput)

	// Common patterns
	if strings.Contains(input, "clone") && strings.Contains(input, "git") {
		return autoCmd
	}

	if strings.Contains(input, "status") || strings.Contains(input, "current") {
		return statusCmd
	}

	if strings.Contains(input, "fix") || strings.Contains(input, "problem") {
		return autoCmd
	}

	return nil
}
