package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
	"github.com/techishthoughts/GitPersona/internal/validation"
)

// SSHTroubleshootCommand handles SSH troubleshooting and diagnostics
type SSHTroubleshootCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	verbose    bool
	autoFix    bool
	account    string
	repository string
}

// NewSSHTroubleshootCommand creates a new SSH troubleshoot command
func NewSSHTroubleshootCommand() *SSHTroubleshootCommand {
	cmd := &SSHTroubleshootCommand{
		BaseCommand: commands.NewBaseCommand(
			"ssh-troubleshoot",
			"🔧 Diagnose and fix SSH authentication issues",
			"ssh-troubleshoot [options]",
		).WithExamples(
			"gitpersona ssh-troubleshoot",
			"gitpersona ssh-troubleshoot --account work",
			"gitpersona ssh-troubleshoot --repository user/repo --verbose",
			"gitpersona ssh-troubleshoot --auto-fix",
		).WithFlags(
			commands.Flag{Name: "verbose", Short: "v", Type: "bool", Default: false, Description: "Show detailed diagnostic information"},
			commands.Flag{Name: "auto-fix", Short: "f", Type: "bool", Default: false, Description: "Automatically fix detected issues"},
			commands.Flag{Name: "account", Short: "a", Type: "string", Default: "", Description: "Troubleshoot specific account"},
			commands.Flag{Name: "repository", Short: "r", Type: "string", Default: "", Description: "Test with specific repository"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *SSHTroubleshootCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.verbose = c.GetFlagBool(cmd, "verbose")
		c.autoFix = c.GetFlagBool(cmd, "auto-fix")
		c.account = c.GetFlagString(cmd, "account")
		c.repository = c.GetFlagString(cmd, "repository")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *SSHTroubleshootCommand) Validate(args []string) error {
	// No validation needed for this command
	return nil
}

// Run executes the SSH troubleshoot command logic
func (c *SSHTroubleshootCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get required services
	configService := container.GetConfigService()
	if configService == nil {
		return errors.New(errors.ErrCodeInternal, "config service not available")
	}

	// Load configuration
	if err := configService.Load(ctx); err != nil {
		return errors.ConfigLoadFailed(err, map[string]interface{}{
			"command": "ssh-troubleshoot",
		})
	}

	c.PrintInfo(ctx, "🔧 Starting SSH troubleshooting...")

	// Run comprehensive SSH diagnostics
	return c.runSSHDiagnostics(ctx, configService)
}

// runSSHDiagnostics runs comprehensive SSH diagnostics
func (c *SSHTroubleshootCommand) runSSHDiagnostics(ctx context.Context, configService services.ConfigurationService) error {
	// 1. Check SSH agent status
	if err := c.checkSSHAgentStatus(ctx); err != nil {
		c.PrintError(ctx, "SSH agent check failed", observability.F("error", err.Error()))
	}

	// 2. Validate SSH configuration
	if err := c.validateSSHConfiguration(ctx); err != nil {
		c.PrintError(ctx, "SSH configuration validation failed", observability.F("error", err.Error()))
	}

	// 3. Check loaded SSH keys
	if err := c.checkLoadedSSHKeys(ctx); err != nil {
		c.PrintError(ctx, "SSH keys check failed", observability.F("error", err.Error()))
	}

	// 4. Test GitHub connectivity
	if err := c.testGitHubConnectivity(ctx, configService); err != nil {
		c.PrintError(ctx, "GitHub connectivity test failed", observability.F("error", err.Error()))
	}

	// 5. Check for common issues
	if err := c.checkCommonIssues(ctx, configService); err != nil {
		c.PrintError(ctx, "Common issues check failed", observability.F("error", err.Error()))
	}

	// 6. Auto-fix if requested
	if c.autoFix {
		if err := c.autoFixIssues(ctx, configService); err != nil {
			c.PrintError(ctx, "Auto-fix failed", observability.F("error", err.Error()))
		}
	}

	// 7. Provide recommendations
	c.provideRecommendations(ctx, configService)

	return nil
}

// checkSSHAgentStatus checks the SSH agent status
func (c *SSHTroubleshootCommand) checkSSHAgentStatus(ctx context.Context) error {
	c.PrintInfo(ctx, "🔍 Checking SSH agent status...")

	// Check if SSH_AUTH_SOCK is set
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		c.PrintWarning(ctx, "SSH_AUTH_SOCK not set - SSH agent may not be running")
		c.PrintInfo(ctx, "💡 Start SSH agent with: eval $(ssh-agent)")
		return nil
	}

	// Check SSH agent response
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.PrintWarning(ctx, "SSH agent not responding or no keys loaded")
		c.PrintInfo(ctx, "💡 Start SSH agent with: eval $(ssh-agent)")
		return nil
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "The agent has no identities." {
		c.PrintInfo(ctx, "SSH agent is running but no keys loaded")
	} else {
		lines := strings.Split(outputStr, "\n")
		c.PrintSuccess(ctx, fmt.Sprintf("SSH agent is running with %d keys loaded", len(lines)))

		if c.verbose {
			for i, line := range lines {
				c.PrintInfo(ctx, fmt.Sprintf("  %d. %s", i+1, line))
			}
		}

		// Check for multiple keys (potential conflict)
		if len(lines) > 1 {
			c.PrintWarning(ctx, "Multiple SSH keys loaded - this may cause authentication conflicts")
			c.PrintInfo(ctx, "💡 Consider using SSH config with IdentitiesOnly=yes or clearing agent")
		}
	}

	return nil
}

// validateSSHConfiguration validates SSH configuration
func (c *SSHTroubleshootCommand) validateSSHConfiguration(ctx context.Context) error {
	c.PrintInfo(ctx, "🔍 Validating SSH configuration...")

	validator := validation.NewSSHValidator()
	result, err := validator.ValidateSSHConfiguration()
	if err != nil {
		return fmt.Errorf("failed to validate SSH configuration: %w", err)
	}

	if result.IsValid {
		c.PrintSuccess(ctx, "SSH configuration is valid")
	} else {
		c.PrintWarning(ctx, "SSH configuration has issues:")
		for _, issue := range result.Issues {
			severity := "⚠️"
			if issue.Severity == "critical" {
				severity = "🚨"
			} else if issue.Severity == "info" {
				severity = "ℹ️"
			}
			c.PrintInfo(ctx, fmt.Sprintf("  %s %s", severity, issue.Description))
			if c.verbose && issue.Solution != "" {
				c.PrintInfo(ctx, fmt.Sprintf("    Solution: %s", issue.Solution))
			}
		}
	}

	return nil
}

// checkLoadedSSHKeys checks the currently loaded SSH keys
func (c *SSHTroubleshootCommand) checkLoadedSSHKeys(ctx context.Context) error {
	c.PrintInfo(ctx, "🔍 Checking loaded SSH keys...")

	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.PrintInfo(ctx, "No SSH keys loaded in agent")
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.Contains(lines[0], "The agent has no identities")) {
		c.PrintInfo(ctx, "No SSH keys loaded in agent")
		return nil
	}

	c.PrintSuccess(ctx, fmt.Sprintf("Found %d loaded SSH keys", len(lines)))

	for i, line := range lines {
		c.PrintInfo(ctx, fmt.Sprintf("  %d. %s", i+1, line))

		// Check for common issues in key descriptions
		if strings.Contains(line, "Permission denied") {
			c.PrintWarning(ctx, "    ⚠️  Key has permission issues")
		}
		if strings.Contains(line, "RSA") && !strings.Contains(line, "4096") {
			c.PrintWarning(ctx, "    ⚠️  RSA key may be too weak (consider 4096-bit)")
		}
	}

	return nil
}

// testGitHubConnectivity tests GitHub connectivity
func (c *SSHTroubleshootCommand) testGitHubConnectivity(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "🔍 Testing GitHub connectivity...")

	// Get accounts to test
	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}
	if len(accounts) == 0 {
		c.PrintWarning(ctx, "No accounts configured for testing")
		return nil
	}

	// Filter by account if specified
	if c.account != "" {
		account, err := configService.GetAccount(ctx, c.account)
		if err != nil {
			c.PrintWarning(ctx, fmt.Sprintf("Account '%s' not found", c.account))
			return nil
		}
		accounts = []*models.Account{account}
	}

	// Test each account
	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			c.PrintInfo(ctx, fmt.Sprintf("Skipping account '%s' - no SSH key configured", account.Alias))
			continue
		}

		c.PrintInfo(ctx, fmt.Sprintf("Testing account '%s' (%s)...", account.Alias, account.GitHubUsername))

		// Test SSH connection
		cmd := exec.Command("ssh", "-T", "git@github.com", "-i", account.SSHKeyPath, "-o", "IdentitiesOnly=yes", "-o", "ConnectTimeout=10")
		output, err := cmd.CombinedOutput()

		outputStr := string(output)
		if strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Hi ") {
			c.PrintSuccess(ctx, fmt.Sprintf("  ✅ Account '%s' authenticated successfully", account.Alias))
		} else {
			c.PrintError(ctx, fmt.Sprintf("  ❌ Account '%s' authentication failed", account.Alias))
			if c.verbose {
				c.PrintInfo(ctx, fmt.Sprintf("    Output: %s", strings.TrimSpace(outputStr)))
			}
			if err != nil {
				c.PrintInfo(ctx, fmt.Sprintf("    Error: %v", err))
			}
		}
	}

	return nil
}

// checkCommonIssues checks for common SSH issues
func (c *SSHTroubleshootCommand) checkCommonIssues(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "🔍 Checking for common SSH issues...")

	// Check SSH directory permissions
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil {
		perm := info.Mode().Perm()
		if perm&0077 != 0 {
			c.PrintWarning(ctx, "SSH directory has overly permissive permissions")
			c.PrintInfo(ctx, "💡 Fix with: chmod 700 ~/.ssh")
		} else {
			c.PrintSuccess(ctx, "SSH directory permissions are correct")
		}
	}

	// Check SSH config file
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	if _, err := os.Stat(sshConfigPath); err == nil {
		// Check for problematic configurations
		content, err := os.ReadFile(sshConfigPath)
		if err == nil {
			configStr := string(content)
			if strings.Contains(configStr, "Host github.com") && !strings.Contains(configStr, "IdentitiesOnly yes") {
				c.PrintWarning(ctx, "SSH config has github.com host without IdentitiesOnly")
				c.PrintInfo(ctx, "💡 This may cause key selection conflicts")
			}
		}
	} else {
		c.PrintInfo(ctx, "No SSH config file found")
		c.PrintInfo(ctx, "💡 Consider creating one with: gitpersona ssh-config generate --apply")
	}

	return nil
}

// autoFixIssues automatically fixes detected issues
func (c *SSHTroubleshootCommand) autoFixIssues(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "🔧 Auto-fixing detected issues...")

	// Fix SSH permissions
	validator := validation.NewSSHValidator()
	if err := validator.FixSSHPermissions(); err != nil {
		c.PrintWarning(ctx, "Failed to fix SSH permissions", observability.F("error", err.Error()))
	} else {
		c.PrintSuccess(ctx, "SSH permissions fixed")
	}

	// Generate and apply SSH config if needed
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		c.PrintInfo(ctx, "Generating SSH configuration...")

		accounts, err := configService.ListAccounts(ctx)
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}
		modelAccounts := make([]models.Account, len(accounts))
		for i, acc := range accounts {
			modelAccounts[i] = *acc
		}

		configContent := validator.GenerateSSHConfig(modelAccounts)
		if err := os.WriteFile(sshConfigPath, []byte(configContent), 0600); err != nil {
			c.PrintWarning(ctx, "Failed to write SSH config", observability.F("error", err.Error()))
		} else {
			c.PrintSuccess(ctx, "SSH configuration generated and applied")
		}
	}

	return nil
}

// provideRecommendations provides troubleshooting recommendations
func (c *SSHTroubleshootCommand) provideRecommendations(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "💡 Troubleshooting Recommendations:")

	// Check if we have multiple accounts
	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}
	if len(accounts) > 1 {
		c.PrintInfo(ctx, "  • You have multiple GitHub accounts configured")
		c.PrintInfo(ctx, "  • Use SSH config with specific host aliases to prevent conflicts")
		c.PrintInfo(ctx, "  • Run: gitpersona ssh-config generate --apply")
	}

	// Check current account
	currentAccount := configService.GetCurrentAccount(ctx)
	if currentAccount != "" {
		c.PrintInfo(ctx, fmt.Sprintf("  • Current account: %s", currentAccount))
		account, err := configService.GetAccount(ctx, currentAccount)
		if err == nil && account.SSHKeyPath != "" {
			c.PrintInfo(ctx, fmt.Sprintf("  • Current SSH key: %s", account.SSHKeyPath))
		}
	}

	// General recommendations
	c.PrintInfo(ctx, "  • If you get 'Repository not found' errors:")
	c.PrintInfo(ctx, "    - Check if the correct SSH key is loaded")
	c.PrintInfo(ctx, "    - Verify the key is associated with the right GitHub account")
	c.PrintInfo(ctx, "    - Use 'gitpersona ssh-agent --cleanup' to clear conflicts")
	c.PrintInfo(ctx, "  • For persistent issues:")
	c.PrintInfo(ctx, "    - Run: gitpersona ssh-config generate --apply")
	c.PrintInfo(ctx, "    - Test with: ssh -T git@github.com -v")

	return nil
}

// SSH troubleshoot command for integration
var (
	sshTroubleshootCmd = &cobra.Command{
		Use:     "ssh-troubleshoot [options]",
		Aliases: []string{"ssh-trouble", "ssh-diagnose"},
		Short:   "🔧 Diagnose and fix SSH authentication issues",
		Long: `🔧 Diagnose and Fix SSH Authentication Issues

This command provides comprehensive SSH troubleshooting to help resolve
authentication problems, especially those related to multiple SSH keys and
GitHub account conflicts.

Features:
- Check SSH agent status and loaded keys
- Validate SSH configuration files
- Test GitHub connectivity for all accounts
- Detect common configuration issues
- Auto-fix permissions and generate configs
- Provide specific troubleshooting steps

Common Issues Resolved:
- "Repository not found" errors due to wrong SSH key
- Multiple SSH keys causing authentication conflicts
- Missing or incorrect SSH configuration
- Permission issues with SSH files
- SSH agent problems

Examples:
  gitpersona ssh-troubleshoot                    # Full diagnostic
  gitpersona ssh-troubleshoot --account work     # Test specific account
  gitpersona ssh-troubleshoot --auto-fix         # Auto-fix issues
  gitpersona ssh-troubleshoot --verbose          # Detailed output`,
		Args: cobra.NoArgs,
		RunE: runSSHTroubleshoot,
	}
)

func init() {
	// Add flags to the command
	sshTroubleshootCmd.Flags().BoolP("verbose", "v", false, "Show detailed diagnostic information")
	sshTroubleshootCmd.Flags().BoolP("auto-fix", "f", false, "Automatically fix detected issues")
	sshTroubleshootCmd.Flags().StringP("account", "a", "", "Troubleshoot specific account")
	sshTroubleshootCmd.Flags().StringP("repository", "r", "", "Test with specific repository")

	rootCmd.AddCommand(sshTroubleshootCmd)
}

// runSSHTroubleshoot runs the SSH troubleshoot command
func runSSHTroubleshoot(cmd *cobra.Command, args []string) error {
	// Create and run the SSH troubleshoot command
	sshTroubleshootCmd := NewSSHTroubleshootCommand()
	ctx := context.Background()
	return sshTroubleshootCmd.Execute(ctx, args)
}
