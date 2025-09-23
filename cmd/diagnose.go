package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/container"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// DiagnoseCommand provides comprehensive system diagnostics
type DiagnoseCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	fix           bool
	verbose       bool
	accountsOnly  bool
	sshOnly       bool
	gitOnly       bool
	includeSystem bool
}

// NewDiagnoseCommand creates a new diagnose command
func NewDiagnoseCommand() *DiagnoseCommand {
	cmd := &DiagnoseCommand{
		BaseCommand: commands.NewBaseCommand(
			"diagnose",
			"🔍 Comprehensive system diagnostics and issue detection",
			"diagnose [flags]",
		).WithExamples(
			"gitpersona diagnose",
			"gitpersona diagnose --fix",
			"gitpersona diagnose --accounts-only",
			"gitpersona diagnose --ssh-only --verbose",
			"gitpersona diagnose --include-system",
		).WithFlags(
			commands.Flag{Name: "fix", Short: "f", Type: "bool", Default: false, Description: "Automatically fix issues where possible"},
			commands.Flag{Name: "accounts-only", Short: "a", Type: "bool", Default: false, Description: "Only diagnose account configurations"},
			commands.Flag{Name: "ssh-only", Short: "s", Type: "bool", Default: false, Description: "Only diagnose SSH configurations"},
			commands.Flag{Name: "git-only", Short: "g", Type: "bool", Default: false, Description: "Only diagnose Git configurations"},
			commands.Flag{Name: "include-system", Short: "S", Type: "bool", Default: false, Description: "Include system-level diagnostics"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *DiagnoseCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Flags are already added by WithFlags in constructor

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.verbose = c.GetFlagBool(cmd, "verbose")
		c.fix = c.GetFlagBool(cmd, "fix")
		c.accountsOnly = c.GetFlagBool(cmd, "accounts-only")
		c.sshOnly = c.GetFlagBool(cmd, "ssh-only")
		c.gitOnly = c.GetFlagBool(cmd, "git-only")
		c.includeSystem = c.GetFlagBool(cmd, "include-system")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Run executes the diagnose command logic
func (c *DiagnoseCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()
	logger := c.GetLogger()

	c.PrintInfo(ctx, "🔍 Starting comprehensive GitPersona diagnostics...")

	// Initialize diagnostic results
	diagnosticResults := &DiagnosticResults{
		Issues:         []DiagnosticIssue{},
		Warnings:       []DiagnosticWarning{},
		AccountResults: make(map[string]*AccountDiagnostic),
		SystemHealth:   &SystemDiagnostic{},
		OverallHealth:  "unknown",
	}

	// Diagnose configuration
	if !c.sshOnly && !c.gitOnly {
		if err := c.diagnoseConfiguration(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "configuration_diagnosis_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Diagnose accounts
	if !c.sshOnly && !c.gitOnly {
		if err := c.diagnoseAccounts(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "accounts_diagnosis_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Diagnose SSH configuration
	if !c.accountsOnly && !c.gitOnly {
		if err := c.diagnoseSSH(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "ssh_diagnosis_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Diagnose Git configuration
	if !c.accountsOnly && !c.sshOnly {
		if err := c.diagnoseGit(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "git_diagnosis_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Diagnose system health
	if c.includeSystem {
		if err := c.diagnoseSystem(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "system_diagnosis_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Calculate overall health
	c.calculateOverallHealth(diagnosticResults)

	// Apply fixes if requested
	if c.fix {
		if err := c.applyFixes(ctx, container, diagnosticResults); err != nil {
			logger.Error(ctx, "fix_application_failed",
				observability.F("error", err.Error()),
			)
		}
	}

	// Display results
	c.displayResults(ctx, diagnosticResults)

	// Return error if critical issues found
	if diagnosticResults.OverallHealth == "critical" {
		return errors.New(errors.ErrCodeValidationFailed, "Critical issues detected in GitPersona configuration")
	}

	return nil
}

// diagnoseConfiguration checks the GitPersona configuration
func (c *DiagnoseCommand) diagnoseConfiguration(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	configService := container.GetConfigService()
	if configService == nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "configuration",
			Type:        "service_unavailable",
			Severity:    "critical",
			Description: "Configuration service is not available",
			Impact:      "GitPersona cannot function without configuration service",
			Fix:         "Restart GitPersona or check installation",
			Automated:   false,
		})
		return nil
	}

	// Load configuration
	if err := configService.Load(ctx); err != nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "configuration",
			Type:        "load_failed",
			Severity:    "critical",
			Description: fmt.Sprintf("Failed to load configuration: %v", err),
			Impact:      "GitPersona cannot access account configurations",
			Fix:         "Check configuration file permissions and format",
			Automated:   false,
		})
		return err
	}

	// Validate configuration
	if err := configService.Validate(ctx); err != nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "configuration",
			Type:        "validation_failed",
			Severity:    "high",
			Description: fmt.Sprintf("Configuration validation failed: %v", err),
			Impact:      "Some GitPersona features may not work correctly",
			Fix:         "Review and correct configuration file format",
			Automated:   false,
		})
	}

	return nil
}

// diagnoseAccounts checks all configured accounts
func (c *DiagnoseCommand) diagnoseAccounts(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	configService := container.GetConfigService()
	if configService == nil {
		return nil
	}

	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "accounts",
			Type:        "list_failed",
			Severity:    "high",
			Description: fmt.Sprintf("Failed to list accounts: %v", err),
			Impact:      "Cannot validate account configurations",
			Fix:         "Check configuration file integrity",
			Automated:   false,
		})
		return err
	}

	if len(accounts) == 0 {
		results.Warnings = append(results.Warnings, DiagnosticWarning{
			Category:       "accounts",
			Type:           "no_accounts",
			Description:    "No accounts configured",
			Recommendation: "Add at least one GitHub account using 'gitpersona add-github'",
		})
		return nil
	}

	// Create SSH key validator
	sshValidator := services.NewSSHKeyValidator(c.GetLogger(), nil)

	for _, account := range accounts {
		accountDiag := &AccountDiagnostic{
			Alias:    account.Alias,
			Issues:   []DiagnosticIssue{},
			Warnings: []DiagnosticWarning{},
			Valid:    true,
		}

		// Validate account fields
		if account.Name == "" {
			accountDiag.Issues = append(accountDiag.Issues, DiagnosticIssue{
				Category:    "accounts",
				Type:        "missing_name",
				Severity:    "medium",
				Description: fmt.Sprintf("Account '%s' has no display name configured", account.Alias),
				Impact:      "Git commits may have incomplete author information",
				Fix:         fmt.Sprintf("Set name: gitpersona update %s --name \"Your Name\"", account.Alias),
				Automated:   false,
			})
			accountDiag.Valid = false
		}

		if account.Email == "" {
			accountDiag.Issues = append(accountDiag.Issues, DiagnosticIssue{
				Category:    "accounts",
				Type:        "missing_email",
				Severity:    "high",
				Description: fmt.Sprintf("Account '%s' has no email configured", account.Alias),
				Impact:      "Git commits will have incomplete author information",
				Fix:         fmt.Sprintf("Set email: gitpersona update %s --email \"your@example.com\"", account.Alias),
				Automated:   false,
			})
			accountDiag.Valid = false
		}

		// Validate SSH key
		if account.SSHKeyPath != "" {
			sshResult, err := sshValidator.ValidateSSHKey(ctx, account.SSHKeyPath, account.GitHubUsername)
			if err != nil {
				accountDiag.Issues = append(accountDiag.Issues, DiagnosticIssue{
					Category:    "ssh",
					Type:        "ssh_validation_failed",
					Severity:    "high",
					Description: fmt.Sprintf("SSH key validation failed: %v", err),
					Impact:      "Account may not be able to authenticate with GitHub",
					Fix:         "Check SSH key configuration and GitHub integration",
					Automated:   false,
				})
				accountDiag.Valid = false
			} else {
				accountDiag.SSHResult = sshResult
				if !sshResult.Valid {
					accountDiag.Valid = false
					// Convert SSH issues to diagnostic issues
					for _, issue := range sshResult.Issues {
						accountDiag.Issues = append(accountDiag.Issues, DiagnosticIssue{
							Category:    "ssh",
							Type:        issue.Type,
							Severity:    issue.Severity,
							Description: issue.Description,
							Impact:      "SSH authentication may fail",
							Fix:         issue.Fix,
							Automated:   issue.Automated,
						})
					}
				}
			}
		} else {
			accountDiag.Warnings = append(accountDiag.Warnings, DiagnosticWarning{
				Category:       "ssh",
				Type:           "no_ssh_key",
				Description:    fmt.Sprintf("Account '%s' has no SSH key configured", account.Alias),
				Recommendation: "Configure SSH key for secure GitHub authentication",
			})
		}

		results.AccountResults[account.Alias] = accountDiag
	}

	return nil
}

// diagnoseSSH checks SSH configuration
func (c *DiagnoseCommand) diagnoseSSH(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	// Check SSH agent
	sshAgentService := container.GetSSHAgentService()
	if sshAgentService != nil {
		running, err := sshAgentService.IsAgentRunning(ctx)
		if err != nil {
			results.Issues = append(results.Issues, DiagnosticIssue{
				Category:    "ssh",
				Type:        "agent_check_failed",
				Severity:    "medium",
				Description: fmt.Sprintf("Failed to check SSH agent status: %v", err),
				Impact:      "Cannot determine SSH agent health",
				Fix:         "Check SSH agent configuration",
				Automated:   false,
			})
		} else if !running {
			results.Warnings = append(results.Warnings, DiagnosticWarning{
				Category:       "ssh",
				Type:           "agent_not_running",
				Description:    "SSH agent is not running",
				Recommendation: "Start SSH agent for better key management",
			})
		}

		// Check loaded keys
		if running {
			keys, err := sshAgentService.ListLoadedKeys(ctx)
			if err != nil {
				results.Issues = append(results.Issues, DiagnosticIssue{
					Category:    "ssh",
					Type:        "agent_keys_list_failed",
					Severity:    "low",
					Description: fmt.Sprintf("Failed to list loaded SSH keys: %v", err),
					Impact:      "Cannot verify which keys are loaded",
					Fix:         "Check SSH agent configuration",
					Automated:   false,
				})
			} else if len(keys) > 1 {
				results.Warnings = append(results.Warnings, DiagnosticWarning{
					Category:       "ssh",
					Type:           "multiple_keys_loaded",
					Description:    fmt.Sprintf("Multiple SSH keys loaded (%d keys)", len(keys)),
					Recommendation: "Consider using only one key at a time to avoid authentication conflicts",
				})
			}
		}
	}

	return nil
}

// diagnoseGit checks Git configuration
func (c *DiagnoseCommand) diagnoseGit(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	gitService := container.GetGitService()
	if gitService == nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "git",
			Type:        "service_unavailable",
			Severity:    "high",
			Description: "Git configuration service is not available",
			Impact:      "Cannot manage Git configuration",
			Fix:         "Check GitPersona installation and dependencies",
			Automated:   false,
		})
		return nil
	}

	// Analyze Git configuration
	var gitConfig *services.GitConfig
	var err error
	if gitConfigService, ok := gitService.(*services.GitConfigService); ok {
		gitConfig, err = gitConfigService.AnalyzeConfiguration(ctx)
		if err != nil {
			results.Issues = append(results.Issues, DiagnosticIssue{
				Category:    "git",
				Type:        "config_analysis_failed",
				Severity:    "high",
				Description: fmt.Sprintf("Failed to analyze Git configuration: %v", err),
				Impact:      "Cannot validate Git settings",
				Fix:         "Check Git installation and permissions",
				Automated:   false,
			})
			return err
		}

		// Convert Git issues to diagnostic issues
		for _, issue := range gitConfig.Issues {
			results.Issues = append(results.Issues, DiagnosticIssue{
				Category:    "git",
				Type:        issue.Type,
				Severity:    issue.Severity,
				Description: issue.Description,
				Impact:      "Git operations may not work correctly",
				Fix:         issue.Fix,
				Automated:   false, // Git issues are typically not auto-fixable
			})
		}
	}

	return nil
}

// diagnoseSystem checks system-level configuration
func (c *DiagnoseCommand) diagnoseSystem(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	// Check Git installation
	if out, err := execCommand(ctx, "git", "--version"); err != nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "system",
			Type:        "git_not_found",
			Severity:    "critical",
			Description: "Git is not installed or not in PATH",
			Impact:      "GitPersona cannot function without Git",
			Fix:         "Install Git: https://git-scm.com/downloads",
			Automated:   false,
		})
	} else {
		results.SystemHealth.GitVersion = strings.TrimSpace(string(out))
	}

	// Check SSH installation
	if out, err := execCommand(ctx, "ssh", "-V"); err != nil {
		results.Issues = append(results.Issues, DiagnosticIssue{
			Category:    "system",
			Type:        "ssh_not_found",
			Severity:    "critical",
			Description: "SSH is not installed or not in PATH",
			Impact:      "Cannot use SSH keys for GitHub authentication",
			Fix:         "Install SSH client",
			Automated:   false,
		})
	} else {
		results.SystemHealth.SSHVersion = strings.TrimSpace(string(out))
	}

	// Check GitHub CLI
	if out, err := execCommand(ctx, "gh", "--version"); err != nil {
		results.Warnings = append(results.Warnings, DiagnosticWarning{
			Category:       "system",
			Type:           "gh_cli_not_found",
			Description:    "GitHub CLI (gh) is not installed",
			Recommendation: "Install GitHub CLI for enhanced GitHub integration",
		})
	} else {
		results.SystemHealth.GitHubCLIVersion = strings.TrimSpace(string(out))
	}

	return nil
}

// calculateOverallHealth determines the overall system health
func (c *DiagnoseCommand) calculateOverallHealth(results *DiagnosticResults) {
	criticalCount := 0
	highCount := 0

	for _, issue := range results.Issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
	}

	// Check account health
	invalidAccounts := 0
	for _, accountDiag := range results.AccountResults {
		if !accountDiag.Valid {
			invalidAccounts++
		}
	}

	if criticalCount > 0 {
		results.OverallHealth = "critical"
	} else if highCount > 2 || invalidAccounts > len(results.AccountResults)/2 {
		results.OverallHealth = "poor"
	} else if highCount > 0 || invalidAccounts > 0 || len(results.Warnings) > 3 {
		results.OverallHealth = "fair"
	} else if len(results.Issues) == 0 && len(results.Warnings) <= 1 {
		results.OverallHealth = "excellent"
	} else {
		results.OverallHealth = "good"
	}
}

// applyFixes automatically fixes issues where possible
func (c *DiagnoseCommand) applyFixes(ctx context.Context, container *container.SimpleContainer, results *DiagnosticResults) error {
	c.PrintInfo(ctx, "🔧 Applying automatic fixes...")

	fixedCount := 0
	sshValidator := services.NewSSHKeyValidator(c.GetLogger(), nil)

	for alias, accountDiag := range results.AccountResults {
		if accountDiag.SSHResult != nil {
			if err := sshValidator.FixSSHKeyIssues(ctx, accountDiag.SSHResult); err != nil {
				c.PrintWarning(ctx, fmt.Sprintf("Failed to fix SSH issues for account '%s': %v", alias, err))
			} else {
				fixedCount++
			}
		}
	}

	c.PrintSuccess(ctx, fmt.Sprintf("Applied %d automatic fixes", fixedCount))
	return nil
}

// displayResults shows the diagnostic results in a user-friendly format
func (c *DiagnoseCommand) displayResults(ctx context.Context, results *DiagnosticResults) {
	// Overall health status
	healthEmoji := c.getHealthEmoji(results.OverallHealth)
	c.PrintInfo(ctx, fmt.Sprintf("\n%s Overall Health: %s", healthEmoji, strings.ToUpper(results.OverallHealth)))

	// Summary
	c.PrintInfo(ctx, "\n📊 Summary:")
	c.PrintInfo(ctx, fmt.Sprintf("  • Issues: %d", len(results.Issues)))
	c.PrintInfo(ctx, fmt.Sprintf("  • Warnings: %d", len(results.Warnings)))
	c.PrintInfo(ctx, fmt.Sprintf("  • Accounts: %d configured", len(results.AccountResults)))

	// Critical and high severity issues first
	if len(results.Issues) > 0 {
		c.PrintInfo(ctx, "\n🚨 Issues Found:")

		// Sort issues by severity
		sort.Slice(results.Issues, func(i, j int) bool {
			severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
			return severityOrder[results.Issues[i].Severity] < severityOrder[results.Issues[j].Severity]
		})

		for _, issue := range results.Issues {
			emoji := c.getSeverityEmoji(issue.Severity)
			c.PrintError(ctx, fmt.Sprintf("  %s [%s] %s: %s", emoji, strings.ToUpper(issue.Severity), issue.Category, issue.Description))
			if c.verbose {
				c.PrintInfo(ctx, fmt.Sprintf("    Impact: %s", issue.Impact))
				c.PrintInfo(ctx, fmt.Sprintf("    Fix: %s", issue.Fix))
			}
		}
	}

	// Account-specific results
	if len(results.AccountResults) > 0 && !c.sshOnly && !c.gitOnly {
		c.PrintInfo(ctx, "\n👤 Account Status:")
		for alias, accountDiag := range results.AccountResults {
			status := "✅ Valid"
			if !accountDiag.Valid {
				status = "❌ Issues"
			}
			c.PrintInfo(ctx, fmt.Sprintf("  • %s: %s", alias, status))

			if c.verbose && len(accountDiag.Issues) > 0 {
				for _, issue := range accountDiag.Issues {
					c.PrintError(ctx, fmt.Sprintf("    - %s", issue.Description))
				}
			}
		}
	}

	// Warnings
	if len(results.Warnings) > 0 {
		c.PrintInfo(ctx, "\n⚠️  Warnings:")
		for _, warning := range results.Warnings {
			c.PrintWarning(ctx, fmt.Sprintf("  • %s: %s", warning.Category, warning.Description))
			if c.verbose && warning.Recommendation != "" {
				c.PrintInfo(ctx, fmt.Sprintf("    Recommendation: %s", warning.Recommendation))
			}
		}
	}

	// System information
	if c.includeSystem && results.SystemHealth != nil {
		c.PrintInfo(ctx, "\n🖥️  System Information:")
		if results.SystemHealth.GitVersion != "" {
			c.PrintInfo(ctx, fmt.Sprintf("  • Git: %s", results.SystemHealth.GitVersion))
		}
		if results.SystemHealth.SSHVersion != "" {
			c.PrintInfo(ctx, fmt.Sprintf("  • SSH: %s", results.SystemHealth.SSHVersion))
		}
		if results.SystemHealth.GitHubCLIVersion != "" {
			c.PrintInfo(ctx, fmt.Sprintf("  • GitHub CLI: %s", results.SystemHealth.GitHubCLIVersion))
		}
	}

	// Recommendations
	if !c.fix && c.hasFixableIssues(results) {
		c.PrintInfo(ctx, "\n💡 Run 'gitpersona diagnose --fix' to automatically resolve fixable issues")
	}
}

// Helper functions
func (c *DiagnoseCommand) getHealthEmoji(health string) string {
	switch health {
	case "excellent":
		return "🟢"
	case "good":
		return "🟡"
	case "fair":
		return "🟠"
	case "poor":
		return "🔴"
	case "critical":
		return "💀"
	default:
		return "❓"
	}
}

func (c *DiagnoseCommand) getSeverityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "💥"
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "❓"
	}
}

func (c *DiagnoseCommand) hasFixableIssues(results *DiagnosticResults) bool {
	for _, issue := range results.Issues {
		if issue.Automated {
			return true
		}
	}

	for _, accountDiag := range results.AccountResults {
		if accountDiag.SSHResult != nil {
			for _, issue := range accountDiag.SSHResult.Issues {
				if issue.Automated {
					return true
				}
			}
		}
	}

	return false
}

// Data structures
type DiagnosticResults struct {
	Issues         []DiagnosticIssue
	Warnings       []DiagnosticWarning
	AccountResults map[string]*AccountDiagnostic
	SystemHealth   *SystemDiagnostic
	OverallHealth  string
}

type DiagnosticIssue struct {
	Category    string `json:"category"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Fix         string `json:"fix"`
	Automated   bool   `json:"automated"`
}

type DiagnosticWarning struct {
	Category       string `json:"category"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	Recommendation string `json:"recommendation"`
}

type AccountDiagnostic struct {
	Alias     string                           `json:"alias"`
	Valid     bool                             `json:"valid"`
	Issues    []DiagnosticIssue                `json:"issues"`
	Warnings  []DiagnosticWarning              `json:"warnings"`
	SSHResult *services.SSHKeyValidationResult `json:"ssh_result,omitempty"`
}

type SystemDiagnostic struct {
	GitVersion       string `json:"git_version"`
	SSHVersion       string `json:"ssh_version"`
	GitHubCLIVersion string `json:"github_cli_version"`
}

// Helper function to execute commands
func execCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// Simple command execution without context
func execCommandSimple(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// Simple diagnose command using traditional Cobra pattern
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "🔍 Comprehensive system diagnostics and issue detection",
	Long: `Diagnose GitPersona configuration and detect potential issues.

This command performs comprehensive diagnostics on:
- Account configurations and validity
- SSH key setup and authentication
- Git configuration and integration
- GitHub connectivity and permissions

Examples:
  gitpersona diagnose
  gitpersona diagnose --fix
  gitpersona diagnose --verbose
  gitpersona diagnose --accounts-only
  gitpersona diagnose --ssh-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		verbose, _ := cmd.Flags().GetBool("verbose")
		fix, _ := cmd.Flags().GetBool("fix")
		accountsOnly, _ := cmd.Flags().GetBool("accounts-only")
		sshOnly, _ := cmd.Flags().GetBool("ssh-only")
		gitOnly, _ := cmd.Flags().GetBool("git-only")
		includeSystem, _ := cmd.Flags().GetBool("include-system")

		return runDiagnose(verbose, fix, accountsOnly, sshOnly, gitOnly, includeSystem)
	},
}

// runDiagnose executes the diagnose logic
func runDiagnose(verbose, fix, accountsOnly, sshOnly, gitOnly, includeSystem bool) error {
	fmt.Println("🔍 GitPersona Diagnostics")
	fmt.Println("=" + strings.Repeat("=", 50))

	issues := 0
	warnings := 0

	// Basic system checks
	if !accountsOnly && !sshOnly && !gitOnly {
		fmt.Println("\n📊 System Health:")

		// Check Git installation
		if _, err := execCommandSimple("git", "--version"); err != nil {
			fmt.Println("  ❌ Git: Not found or not accessible")
			issues++
		} else {
			fmt.Println("  ✅ Git: Available")
		}

		// Check SSH
		if _, err := execCommandSimple("ssh", "-V"); err != nil {
			fmt.Println("  ❌ SSH: Not found or not accessible")
			issues++
		} else {
			fmt.Println("  ✅ SSH: Available")
		}

		// Check GitHub CLI
		if _, err := execCommandSimple("gh", "--version"); err != nil {
			fmt.Println("  ⚠️  GitHub CLI: Not found (recommended for enhanced integration)")
			warnings++
		} else {
			fmt.Println("  ✅ GitHub CLI: Available")
		}
	}

	// SSH diagnostics
	if !accountsOnly && !gitOnly {
		fmt.Println("\n🔐 SSH Configuration:")

		// Check SSH agent
		if _, err := execCommandSimple("ssh-add", "-l"); err != nil {
			fmt.Println("  ⚠️  SSH Agent: No keys loaded or agent not running")
			warnings++
		} else {
			// Count loaded keys
			out, _ := execCommandSimple("ssh-add", "-l")
			keyCount := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
			if keyCount > 5 {
				fmt.Printf("  ⚠️  SSH Agent: %d keys loaded (too many - may cause conflicts)\n", keyCount)
				warnings++
			} else {
				fmt.Printf("  ✅ SSH Agent: %d keys loaded\n", keyCount)
			}
		}

		// Check SSH config
		sshConfigPath := fmt.Sprintf("%s/.ssh/config", homeDir())
		if _, err := os.Stat(sshConfigPath); err != nil {
			fmt.Println("  ⚠️  SSH Config: No ~/.ssh/config file found")
			warnings++
		} else {
			fmt.Println("  ✅ SSH Config: ~/.ssh/config exists")
		}
	}

	// Account diagnostics
	if !sshOnly && !gitOnly {
		fmt.Println("\n👤 Account Configuration:")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			fmt.Printf("  ❌ Config Load: Failed to load configuration: %v\n", err)
			issues++
		} else {
			accounts := configManager.ListAccounts()
			if len(accounts) == 0 {
				fmt.Println("  ⚠️  Accounts: No accounts configured")
				warnings++
			} else {
				fmt.Printf("  ✅ Accounts: %d configured\n", len(accounts))

				// Validate each account
				for _, account := range accounts {
					if account.Name == "" || account.Email == "" {
						fmt.Printf("    ⚠️  %s: Missing name or email\n", account.Alias)
						warnings++
					} else if account.SSHKeyPath == "" {
						fmt.Printf("    ⚠️  %s: No SSH key configured\n", account.Alias)
						warnings++
					} else {
						fmt.Printf("    ✅ %s: Properly configured\n", account.Alias)
					}
				}
			}
		}
	}

	// Git diagnostics
	if !accountsOnly && !sshOnly {
		fmt.Println("\n🔧 Git Configuration:")

		// Check global Git config
		if name, err := execCommandSimple("git", "config", "--global", "user.name"); err != nil || len(strings.TrimSpace(string(name))) == 0 {
			fmt.Println("  ⚠️  Global Name: Not set")
			warnings++
		} else {
			fmt.Printf("  ✅ Global Name: %s\n", strings.TrimSpace(string(name)))
		}

		if email, err := execCommandSimple("git", "config", "--global", "user.email"); err != nil || len(strings.TrimSpace(string(email))) == 0 {
			fmt.Println("  ⚠️  Global Email: Not set")
			warnings++
		} else {
			fmt.Printf("  ✅ Global Email: %s\n", strings.TrimSpace(string(email)))
		}

		// Check current repository if we're in one
		if _, err := execCommandSimple("git", "rev-parse", "--git-dir"); err == nil {
			fmt.Println("  ✅ Current Directory: Git repository detected")

			// Check remote URL
			if remote, err := execCommandSimple("git", "remote", "get-url", "origin"); err == nil {
				remoteStr := strings.TrimSpace(string(remote))
				if strings.Contains(remoteStr, "github.com") {
					fmt.Printf("  ✅ Remote: %s\n", remoteStr)
				} else {
					fmt.Printf("  ⚠️  Remote: %s (not GitHub)\n", remoteStr)
					warnings++
				}
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 52))

	if issues == 0 && warnings == 0 {
		fmt.Println("🟢 Overall Health: EXCELLENT")
		fmt.Println("All systems functioning properly!")
	} else if issues == 0 && warnings <= 2 {
		fmt.Println("🟡 Overall Health: GOOD")
		fmt.Printf("Found %d warning(s) - minor issues detected\n", warnings)
	} else if issues <= 1 && warnings <= 5 {
		fmt.Println("🟠 Overall Health: FAIR")
		fmt.Printf("Found %d issue(s) and %d warning(s)\n", issues, warnings)
	} else {
		fmt.Println("🔴 Overall Health: POOR")
		fmt.Printf("Found %d issue(s) and %d warning(s)\n", issues, warnings)
	}

	if fix && (issues > 0 || warnings > 0) {
		fmt.Println("\n🔧 Applying automatic fixes...")
		fixedCount := applyAutoFixes(verbose)
		fmt.Printf("✅ Applied %d automatic fixes\n", fixedCount)
	} else if issues > 0 || warnings > 0 {
		fmt.Println("\n💡 Run 'gitpersona diagnose --fix' to automatically resolve fixable issues")
	}

	return nil
}

// applyAutoFixes automatically fixes common issues
func applyAutoFixes(verbose bool) int {
	fixedCount := 0

	fmt.Println("\n🔧 Auto-Fix Operations:")

	// Fix 1: Clear SSH agent if too many keys loaded
	if out, err := execCommandSimple("ssh-add", "-l"); err == nil {
		keyCount := len(strings.Split(strings.TrimSpace(string(out)), "\n"))
		if keyCount > 5 {
			if verbose {
				fmt.Printf("  🧹 Clearing SSH agent (%d keys loaded)...\n", keyCount)
			}
			if _, err := execCommandSimple("ssh-add", "-D"); err == nil {
				fmt.Println("  ✅ SSH agent cleared")
				fixedCount++
			} else {
				fmt.Println("  ❌ Failed to clear SSH agent")
			}
		}
	}

	// Fix 2: Clear problematic Git SSH configurations
	if verbose {
		fmt.Println("  🧹 Clearing problematic Git SSH configurations...")
	}
	gitSSHFixed := false
	if _, err := execCommandSimple("git", "config", "--global", "--unset", "core.sshcommand"); err == nil {
		gitSSHFixed = true
	}
	if _, err := execCommandSimple("git", "config", "--local", "--unset", "core.sshcommand"); err == nil {
		gitSSHFixed = true
	}
	os.Unsetenv("GIT_SSH_COMMAND")
	if gitSSHFixed {
		fmt.Println("  ✅ Git SSH configuration cleared")
		fixedCount++
	}

	// Fix 3: Set missing Git configuration from first valid account
	configManager := config.NewManager()
	if err := configManager.Load(); err == nil {
		accounts := configManager.ListAccounts()
		for _, account := range accounts {
			if account.Name != "" && account.Email != "" {
				// Check if global config is missing
				name, nameErr := execCommandSimple("git", "config", "--global", "user.name")
				email, emailErr := execCommandSimple("git", "config", "--global", "user.email")

				needNameFix := nameErr != nil || len(strings.TrimSpace(string(name))) == 0
				needEmailFix := emailErr != nil || len(strings.TrimSpace(string(email))) == 0

				if needNameFix || needEmailFix {
					if verbose {
						fmt.Printf("  🔧 Setting Git config from account '%s'...\n", account.Alias)
					}
					if needNameFix {
						if _, err := execCommandSimple("git", "config", "--global", "user.name", account.Name); err == nil {
							fmt.Printf("  ✅ Set Git user.name: %s\n", account.Name)
							fixedCount++
						}
					}
					if needEmailFix {
						if _, err := execCommandSimple("git", "config", "--global", "user.email", account.Email); err == nil {
							fmt.Printf("  ✅ Set Git user.email: %s\n", account.Email)
							fixedCount++
						}
					}
				}
				break // Use first valid account
			}
		}

		// Fix 3b: Fix incomplete account configurations
		for _, account := range accounts {
			if account.Name == "" || account.Email == "" {
				if verbose {
					fmt.Printf("  🔧 Fixing incomplete account '%s'...\n", account.Alias)
				}
				// Try to get the missing info from git config if available
				gitName, _ := execCommandSimple("git", "config", "--global", "user.name")
				gitEmail, _ := execCommandSimple("git", "config", "--global", "user.email")

				updated := false
				if account.Name == "" && len(strings.TrimSpace(string(gitName))) > 0 {
					account.Name = strings.TrimSpace(string(gitName))
					updated = true
					fmt.Printf("  ✅ Set account name: %s\n", account.Name)
				}
				if account.Email == "" && len(strings.TrimSpace(string(gitEmail))) > 0 {
					account.Email = strings.TrimSpace(string(gitEmail))
					updated = true
					fmt.Printf("  ✅ Set account email: %s\n", account.Email)
				}
				if updated {
					// Update account in config by removing and re-adding
					if err := configManager.RemoveAccount(account.Alias); err == nil {
						if err := configManager.AddAccount(account); err == nil {
							if err := configManager.Save(); err == nil {
								fixedCount++
							}
						}
					}
				}
			}
		}
	}

	// Fix 4: Ensure SSH config directory exists
	sshDir := fmt.Sprintf("%s/.ssh", homeDir())
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		if verbose {
			fmt.Println("  📁 Creating SSH directory...")
		}
		if err := os.MkdirAll(sshDir, 0700); err == nil {
			fmt.Println("  ✅ Created ~/.ssh directory")
			fixedCount++
		}
	}

	// Fix 5: Load SSH key for current account if available
	if configManager := config.NewManager(); configManager.Load() == nil {
		currentAccount, _ := configManager.GetCurrentAccount()
		if currentAccount != nil && currentAccount.SSHKeyPath != "" {
			if _, err := os.Stat(currentAccount.SSHKeyPath); err == nil {
				// Check if key is already loaded
				if out, err := execCommandSimple("ssh-add", "-l"); err != nil || !strings.Contains(string(out), currentAccount.SSHKeyPath) {
					if verbose {
						fmt.Printf("  🔑 Loading SSH key for current account '%s'...\n", currentAccount.Alias)
					}
					if _, err := execCommandSimple("ssh-add", currentAccount.SSHKeyPath); err == nil {
						fmt.Printf("  ✅ Loaded SSH key: %s\n", currentAccount.SSHKeyPath)
						fixedCount++
					}
				}
			}
		}
	}

	if fixedCount == 0 {
		fmt.Println("  ℹ️  No fixable issues found")
	}

	return fixedCount
}

// Helper function to get home directory
func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "~"
}

// Register the command
func init() {
	diagnoseCmd.Flags().BoolP("fix", "f", false, "Automatically fix issues where possible")
	diagnoseCmd.Flags().BoolP("accounts-only", "a", false, "Only diagnose account configurations")
	diagnoseCmd.Flags().BoolP("ssh-only", "s", false, "Only diagnose SSH configurations")
	diagnoseCmd.Flags().BoolP("git-only", "g", false, "Only diagnose Git configurations")
	diagnoseCmd.Flags().BoolP("include-system", "S", false, "Include system-level diagnostics")
	diagnoseCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	diagnoseCmd.Flags().BoolP("json", "j", false, "Output in JSON format")

	rootCmd.AddCommand(diagnoseCmd)
}
