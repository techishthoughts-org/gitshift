package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
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

// Register the command
func init() {
	diagnoseCmd := NewDiagnoseCommand().CreateCobraCommand()
	rootCmd.AddCommand(diagnoseCmd)
}
