package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/github"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// SwitchCommand handles account switching with proper validation and logging
type SwitchCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	validateOnly   bool
	skipValidation bool
	force          bool
}

// NewSwitchCommand creates a new switch command
func NewSwitchCommand() *SwitchCommand {
	cmd := &SwitchCommand{
		BaseCommand: commands.NewBaseCommand(
			"switch",
			"🔄 Switch to a different GitHub account",
			"switch [alias]",
		).WithExamples(
			"gitpersona switch personal",
			"gitpersona switch work --validate-only",
			"gitpersona switch personal --force",
		).WithFlags(
			commands.Flag{Name: "validate", Short: "V", Type: "bool", Default: false, Description: "Only validate current account without switching"},
			commands.Flag{Name: "skip-validation", Short: "s", Type: "bool", Default: false, Description: "Skip SSH validation (not recommended)"},
			commands.Flag{Name: "force", Short: "f", Type: "bool", Default: false, Description: "Force switch even if validation fails"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *SwitchCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.validateOnly = c.GetFlagBool(cmd, "validate")
		c.skipValidation = c.GetFlagBool(cmd, "skip-validation")
		c.force = c.GetFlagBool(cmd, "force")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *SwitchCommand) Validate(args []string) error {
	if c.validateOnly {
		// No arguments needed for validation-only mode
		return nil
	}

	if len(args) == 0 {
		return errors.New(errors.ErrCodeMissingRequired, "account alias is required").
			WithContext("field", "alias")
	}

	if len(args) > 1 {
		return errors.New(errors.ErrCodeInvalidInput, "too many arguments").
			WithContext("expected", 1).
			WithContext("provided", len(args))
	}

	return nil
}

// loadConfiguration loads the configuration using the config service
func (c *SwitchCommand) loadConfiguration(ctx context.Context, configService services.ConfigurationService) error {
	if configService == nil {
		return fmt.Errorf("config service not available")
	}
	return configService.Load(ctx)
}

// getAccount retrieves an account using the config service
func (c *SwitchCommand) getAccount(ctx context.Context, configService services.ConfigurationService, alias string) (*models.Account, error) {
	if configService == nil {
		return nil, errors.New(errors.ErrCodeInternal, "config service not available")
	}

	return configService.GetAccount(ctx, alias)
}

// validateCurrentAccount validates the current account without switching
func (c *SwitchCommand) validateCurrentAccount(ctx context.Context, configService services.ConfigurationService) error {
	if configService == nil {
		return errors.New(errors.ErrCodeInternal, "config service not available")
	}

	currentAlias := configService.GetCurrentAccount(ctx)
	if currentAlias == "" {
		c.PrintWarning(ctx, "No current account set")
		return nil
	}

	// Get account details
	account, err := configService.GetAccount(ctx, currentAlias)
	if err != nil {
		c.PrintWarning(ctx, "Could not retrieve current account details",
			observability.F("account", currentAlias),
			observability.F("error", err.Error()),
		)
		return nil
	}

	// Display current account information
	c.PrintInfo(ctx, fmt.Sprintf("Current account: %s", currentAlias),
		observability.F("account", currentAlias),
		observability.F("name", account.Name),
		observability.F("email", account.Email),
	)

	// Validate SSH configuration
	if err := c.validateAccountSSH(ctx, account); err != nil {
		c.PrintWarning(ctx, "SSH validation failed for current account",
			observability.F("account", currentAlias),
			observability.F("error", err.Error()),
		)
	} else {
		c.PrintSuccess(ctx, "SSH validation passed for current account",
			observability.F("account", currentAlias),
		)
	}

	return nil
}

// validateAccountSSH validates the SSH configuration for an account
func (c *SwitchCommand) validateAccountSSH(ctx context.Context, account *models.Account) error {
	// Get SSH agent service from container
	container := c.GetContainer()
	sshAgentService := container.GetSSHAgentService()

	if sshAgentService == nil {
		c.PrintWarning(ctx, "SSH agent service not available, skipping validation")
		return nil
	}

	// If no SSH key is configured, provide helpful guidance
	if account.SSHKeyPath == "" {
		c.PrintWarning(ctx, "No SSH key configured for account",
			observability.F("account", account.Alias),
			observability.F("github_username", account.GitHubUsername),
		)
		c.PrintInfo(ctx, "💡 To fix this issue:")
		c.PrintInfo(ctx, "  1. Generate SSH key: gitpersona add-github "+account.GitHubUsername+" --overwrite")
		c.PrintInfo(ctx, "  2. Or add existing key: gitpersona config set --account "+account.Alias+" ssh_key_path /path/to/key")
		c.PrintInfo(ctx, "  3. Then run: gitpersona switch "+account.Alias)
		return fmt.Errorf("SSH key not configured for account %s", account.Alias)
	}

	// Check if SSH key file actually exists
	if _, err := os.Stat(account.SSHKeyPath); os.IsNotExist(err) {
		c.PrintError(ctx, "SSH key file not found",
			observability.F("account", account.Alias),
			observability.F("ssh_key_path", account.SSHKeyPath),
		)
		c.PrintInfo(ctx, "💡 To fix this issue:")
		c.PrintInfo(ctx, "  1. Generate new SSH key: gitpersona add-github "+account.GitHubUsername+" --overwrite")
		c.PrintInfo(ctx, "  2. Or update key path: gitpersona config set --account "+account.Alias+" ssh_key_path /correct/path/to/key")
		return fmt.Errorf("SSH key file not found: %s", account.SSHKeyPath)
	}

	// Check for SSH key conflicts before validation
	if err := c.checkSSHKeyConflicts(ctx, account); err != nil {
		c.PrintWarning(ctx, "SSH key conflicts detected",
			observability.F("error", err.Error()),
		)
		c.PrintInfo(ctx, "💡 Consider running: gitpersona ssh-agent --cleanup")
	}

	// Use the ValidateSSHConnectionWithRetry method from SSH agent service
	c.PrintInfo(ctx, "Validating SSH connection with retry mechanism...",
		observability.F("ssh_key", account.SSHKeyPath),
	)

	if err := sshAgentService.ValidateSSHConnectionWithRetry(ctx, account.SSHKeyPath); err != nil {
		// Provide specific error messages for common SSH issues
		if strings.Contains(err.Error(), "Repository not found") {
			return c.handleRepositoryNotFoundError(ctx, account, err)
		}
		if strings.Contains(err.Error(), "Permission denied") {
			return c.handlePermissionDeniedError(ctx, account, err)
		}
		return fmt.Errorf("SSH validation failed: %w", err)
	}

	c.PrintSuccess(ctx, "SSH validation successful",
		observability.F("ssh_key", account.SSHKeyPath),
	)
	return nil
}

// checkSSHKeyConflicts checks for potential SSH key conflicts
func (c *SwitchCommand) checkSSHKeyConflicts(ctx context.Context, account *models.Account) error {
	// Check if SSH agent has multiple keys loaded
	container := c.GetContainer()
	sshAgentService := container.GetSSHAgentService()

	if service, ok := sshAgentService.(interface {
		ListLoadedKeys(ctx context.Context) ([]string, error)
	}); ok {
		keys, err := service.ListLoadedKeys(ctx)
		if err != nil {
			return fmt.Errorf("failed to check loaded keys: %w", err)
		}

		if len(keys) > 1 {
			return fmt.Errorf("SSH agent has %d keys loaded, which may cause authentication conflicts", len(keys))
		}
	}

	return nil
}

// handleRepositoryNotFoundError provides specific guidance for repository not found errors
func (c *SwitchCommand) handleRepositoryNotFoundError(ctx context.Context, account *models.Account, err error) error {
	c.PrintError(ctx, "Repository not found - this usually indicates SSH key authentication issues",
		observability.F("account", account.Alias),
		observability.F("ssh_key", account.SSHKeyPath),
	)

	c.PrintInfo(ctx, "🔍 Troubleshooting steps:")
	c.PrintInfo(ctx, "  1. Verify the SSH key is associated with the correct GitHub account")
	c.PrintInfo(ctx, "  2. Check if multiple SSH keys are loaded in the agent")
	c.PrintInfo(ctx, "  3. Ensure your SSH config uses IdentitiesOnly=yes")
	c.PrintInfo(ctx, "  4. Try: gitpersona ssh-agent --cleanup")

	// Generate SSH config suggestion
	if account.SSHKeyPath != "" {
		c.PrintInfo(ctx, "💡 Recommended SSH config:")
		c.PrintInfo(ctx, "   Host github.com")
		c.PrintInfo(ctx, "     HostName github.com")
		c.PrintInfo(ctx, "     User git")
		c.PrintInfo(ctx, fmt.Sprintf("     IdentityFile %s", account.SSHKeyPath))
		c.PrintInfo(ctx, "     IdentitiesOnly yes")
	}

	return fmt.Errorf("SSH authentication failed - wrong key may be used: %w", err)
}

// handlePermissionDeniedError provides specific guidance for permission denied errors
func (c *SwitchCommand) handlePermissionDeniedError(ctx context.Context, account *models.Account, err error) error {
	c.PrintError(ctx, "Permission denied - SSH key may not be properly configured",
		observability.F("account", account.Alias),
		observability.F("ssh_key", account.SSHKeyPath),
	)

	c.PrintInfo(ctx, "🔍 Troubleshooting steps:")
	c.PrintInfo(ctx, "  1. Verify the public key is added to your GitHub account")
	c.PrintInfo(ctx, "  2. Check SSH key permissions: chmod 600 "+account.SSHKeyPath)
	c.PrintInfo(ctx, "  3. Test SSH connection: ssh -T git@github.com -i "+account.SSHKeyPath)
	c.PrintInfo(ctx, "  4. Ensure the key is not password-protected or remove passphrase")

	return fmt.Errorf("SSH permission denied: %w", err)
}

// performAccountSwitch performs the actual account switch
func (c *SwitchCommand) performAccountSwitch(ctx context.Context, configService services.ConfigurationService, targetAlias string, targetAccount *models.Account) error {
	c.GetLogger().Info(ctx, "performing_account_switch",
		observability.F("target_account", targetAlias),
		observability.F("account_name", targetAccount.Name),
	)

	// Update current account in config service
	if err := configService.SetCurrentAccount(ctx, targetAlias); err != nil {
		return fmt.Errorf("failed to set current account: %w", err)
	}

	// Manage SSH agent for the account
	if err := c.manageSSHAgent(ctx, targetAccount); err != nil {
		c.PrintWarning(ctx, "SSH agent management failed, but continuing with switch",
			observability.F("error", err.Error()),
		)
	}

	// Update Git configuration
	if err := c.updateGitConfig(ctx, targetAccount); err != nil {
		return fmt.Errorf("failed to update Git configuration: %w", err)
	}

	// Update GitHub token in zsh_secrets
	if err := c.updateGitHubTokenInZshSecrets(ctx, targetAccount); err != nil {
		c.PrintWarning(ctx, "Failed to update GitHub token in zsh_secrets, but continuing with switch",
			observability.F("error", err.Error()),
		)
	}

	return nil
}

// manageSSHAgent manages the SSH agent for the account
func (c *SwitchCommand) manageSSHAgent(ctx context.Context, account *models.Account) error {
	container := c.GetContainer()
	sshAgentService := container.GetSSHAgentService()

	if sshAgentService == nil {
		c.PrintWarning(ctx, "SSH agent service not available, skipping SSH agent management")
		return nil
	}

	// If no SSH key is configured, skip SSH agent management
	if account.SSHKeyPath == "" {
		c.PrintInfo(ctx, "No SSH key configured for account, skipping SSH agent management")
		return nil
	}

	c.PrintInfo(ctx, "Managing SSH agent for account",
		observability.F("ssh_key", account.SSHKeyPath),
	)

	// Switch to the account's SSH key with socket cleanup (this will clear other keys and load only this one)
	if err := sshAgentService.SwitchToAccountWithCleanup(ctx, account.SSHKeyPath); err != nil {
		c.PrintWarning(ctx, "SSH agent switch encountered an issue",
			observability.F("error", err.Error()),
			observability.F("ssh_key", account.SSHKeyPath),
		)

		// Provide helpful error message based on error type
		if strings.Contains(err.Error(), "socket") {
			c.PrintInfo(ctx, "💡 Try running: gitpersona ssh-agent --cleanup")
		} else if strings.Contains(err.Error(), "permission") {
			c.PrintInfo(ctx, "💡 Check SSH key permissions: chmod 600 "+account.SSHKeyPath)
		} else if strings.Contains(err.Error(), "not found") {
			c.PrintInfo(ctx, "💡 SSH key file not found: "+account.SSHKeyPath)
		}

		// Don't fail the entire switch - SSH agent issues shouldn't block the account switch
		c.PrintInfo(ctx, "Continuing with account switch despite SSH agent issue")
		return nil
	}

	// Note: SSH validation is now handled in validateAccountSSH before the switch

	c.PrintSuccess(ctx, "SSH agent configured for account",
		observability.F("ssh_key", account.SSHKeyPath),
	)

	return nil
}

// Implement the Run method that was missing
func (c *SwitchCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get required services
	configService := container.GetConfigService()
	if configService == nil {
		return errors.New(errors.ErrCodeInternal, "config service not available")
	}

	// Load configuration
	if err := c.loadConfiguration(ctx, configService); err != nil {
		return errors.ConfigLoadFailed(err, map[string]interface{}{
			"command": "switch",
		})
	}

	if c.validateOnly {
		return c.validateCurrentAccount(ctx, configService)
	}

	// Get target account
	targetAlias := args[0]
	targetAccount, err := c.getAccount(ctx, configService, targetAlias)
	if err != nil {
		return errors.AccountNotFound(targetAlias, map[string]interface{}{
			"command": "switch",
		})
	}

	// Validate account SSH configuration
	if !c.skipValidation {
		if err := c.validateAccountSSH(ctx, targetAccount); err != nil {
			if c.force {
				c.PrintWarning(ctx, "SSH validation failed, but forcing switch due to --force flag",
					observability.F("account", targetAlias),
					observability.F("error", err.Error()),
				)
				c.PrintInfo(ctx, "⚠️  Warning: Proceeding with switch despite validation failure")
			} else {
				c.PrintError(ctx, "SSH validation failed. Use --force to bypass validation",
					observability.F("account", targetAlias),
					observability.F("error", err.Error()),
				)
				return errors.SSHValidationFailed(err, map[string]interface{}{
					"account": targetAlias,
					"command": "switch",
				})
			}
		}
	}

	// Perform the account switch
	if err := c.performAccountSwitch(ctx, configService, targetAlias, targetAccount); err != nil {
		return errors.Wrap(err, errors.ErrCodeAccountSwitchFailed, "failed to switch account").
			WithContext("account", targetAlias)
	}

	// Validate switch success
	if err := c.validateSwitchSuccess(ctx, configService, targetAlias); err != nil {
		return errors.Wrap(err, errors.ErrCodeAccountSwitchFailed, "switch completed but validation failed").
			WithContext("account", targetAlias)
	}

	// Success
	c.PrintSuccess(ctx, fmt.Sprintf("Successfully switched to account '%s'", targetAlias),
		observability.F("account", targetAlias),
		observability.F("name", targetAccount.Name),
		observability.F("email", targetAccount.Email),
	)

	return nil
}

// updateGitConfig updates the Git configuration for the account
func (c *SwitchCommand) updateGitConfig(ctx context.Context, account *models.Account) error {
	container := c.GetContainer()
	gitService := container.GetGitService()

	if gitService == nil {
		return fmt.Errorf("git service not available")
	}

	// Set user configuration
	if account.Name != "" || account.Email != "" {
		if err := gitService.SetUserConfiguration(ctx, account.Name, account.Email); err != nil {
			return fmt.Errorf("failed to set user configuration: %w", err)
		}
		c.PrintSuccess(ctx, fmt.Sprintf("Updated Git user configuration: %s <%s>", account.Name, account.Email))
	}

	// Set SSH command
	if account.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", account.SSHKeyPath)
		if err := gitService.SetSSHCommand(ctx, sshCmd); err != nil {
			return fmt.Errorf("failed to set SSH command: %w", err)
		}
		c.PrintSuccess(ctx, fmt.Sprintf("Updated Git SSH command: %s", sshCmd))
	}

	return nil
}

// updateGitHubTokenInZshSecrets updates the GITHUB_TOKEN in zsh_secrets file
func (c *SwitchCommand) updateGitHubTokenInZshSecrets(ctx context.Context, account *models.Account) error {
	container := c.GetContainer()
	zshSecretsService := container.GetZshSecretsService()

	if zshSecretsService == nil {
		c.PrintInfo(ctx, "Zsh secrets service not available, skipping GitHub token update")
		return nil
	}

	c.PrintInfo(ctx, "Updating GitHub token in zsh_secrets...",
		observability.F("account", account.Alias),
	)

	// Get the current GitHub token from GitHub CLI
	// For now, we'll use the existing GitHub client to get the token
	githubClient := github.NewClient("")
	token, err := githubClient.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Update the token in zsh_secrets
	if err := zshSecretsService.UpdateGitHubToken(ctx, token); err != nil {
		return fmt.Errorf("failed to update GitHub token in zsh_secrets: %w", err)
	}

	c.PrintSuccess(ctx, "Updated GitHub token in zsh_secrets",
		observability.F("account", account.Alias),
	)

	// Optionally reload the zsh_secrets file
	if err := zshSecretsService.ReloadZshSecrets(ctx); err != nil {
		c.PrintWarning(ctx, "Failed to reload zsh_secrets file",
			observability.F("error", err.Error()),
		)
		// Don't fail the entire operation if reload fails
	}

	return nil
}

// validateSwitchSuccess validates that the switch was successful
func (c *SwitchCommand) validateSwitchSuccess(ctx context.Context, configService services.ConfigurationService, targetAlias string) error {
	logger := c.GetLogger()

	logger.Info(ctx, "validating_switch_success",
		observability.F("target_account", targetAlias),
	)

	// Verify the current account was set correctly
	currentAccount := configService.GetCurrentAccount(ctx)
	if currentAccount != targetAlias {
		return fmt.Errorf("account switch verification failed: expected %s, got %s", targetAlias, currentAccount)
	}

	return nil
}

// Switch command for integration
var (
	switchCmd = &cobra.Command{
		Use:     "switch [alias]",
		Aliases: []string{"s", "use"},
		Short:   "🔄 Switch to a different GitHub account",
		Long: `🔄 Switch to a Different GitHub Account

This command switches your active GitHub account and validates the SSH configuration
to ensure everything works correctly before making the switch.

The command will:
1. Validate SSH configuration for the target account
2. Test GitHub authentication
3. Update Git configuration
4. Verify the switch was successful

Examples:
  gitpersona switch personal     # Switch to personal account
  gitpersona switch work         # Switch to work account
  gitpersona switch --validate   # Validate current account without switching`,
		Args: cobra.MaximumNArgs(1),
		RunE: runSwitch,
	}
)

func init() {
	// Add flags to the command
	switchCmd.Flags().BoolP("validate", "V", false, "Only validate current account without switching")
	switchCmd.Flags().BoolP("skip-validation", "s", false, "Skip SSH validation (not recommended)")
	switchCmd.Flags().BoolP("force", "f", false, "Force switch even if validation fails")

	rootCmd.AddCommand(switchCmd)
}

// runSwitch runs the switch command using the service-oriented approach
func runSwitch(cmd *cobra.Command, args []string) error {
	// Create a new switch command instance
	switchCmd := NewSwitchCommand()

	// Get flag values and set them
	switchCmd.validateOnly, _ = cmd.Flags().GetBool("validate")
	switchCmd.skipValidation, _ = cmd.Flags().GetBool("skip-validation")
	switchCmd.force, _ = cmd.Flags().GetBool("force")

	// Validate arguments
	if err := switchCmd.Validate(args); err != nil {
		return err
	}

	// Execute using the service-oriented implementation
	ctx := context.Background()
	return switchCmd.Run(ctx, args)
}
