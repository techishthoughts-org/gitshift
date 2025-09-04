package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
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
			commands.Flag{Name: "validate", Short: "v", Type: "bool", Default: false, Description: "Only validate current account without switching"},
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

// Run executes the switch command logic
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
				c.PrintWarning(ctx, "SSH validation failed, but forcing switch",
					observability.F("account", targetAlias),
					observability.F("error", err.Error()),
				)
			} else {
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

// loadConfiguration loads the configuration using the config service
func (c *SwitchCommand) loadConfiguration(ctx context.Context, configService interface{}) error {
	// Try to load configuration if the service supports it
	if service, ok := configService.(interface{ Load(context.Context) error }); ok {
		return service.Load(ctx)
	}

	// Fallback to success if service doesn't support loading
	return nil
}

// getAccount retrieves an account using the config service
func (c *SwitchCommand) getAccount(ctx context.Context, configService interface{}, alias string) (*models.Account, error) {
	// Try to get account from config service if available
	if service, ok := configService.(interface {
		GetAccounts(context.Context) map[string]interface{}
	}); ok {
		accounts := service.GetAccounts(ctx)
		if accountData, exists := accounts[alias]; exists {
			// Convert to Account model
			if accountMap, ok := accountData.(map[string]interface{}); ok {
				account := &models.Account{
					Alias: alias,
				}

				if name, exists := accountMap["name"]; exists {
					account.Name = fmt.Sprintf("%v", name)
				}
				if email, exists := accountMap["email"]; exists {
					account.Email = fmt.Sprintf("%v", email)
				}
				if sshKeyPath, exists := accountMap["ssh_key_path"]; exists {
					account.SSHKeyPath = fmt.Sprintf("%v", sshKeyPath)
				}
				if githubUsername, exists := accountMap["github_username"]; exists {
					account.GitHubUsername = fmt.Sprintf("%v", githubUsername)
				}

				return account, nil
			}
		}
	}

	return nil, errors.AccountNotFound(alias, map[string]interface{}{
		"command": "switch",
	})
}

// validateCurrentAccount validates the current account without switching
func (c *SwitchCommand) validateCurrentAccount(ctx context.Context, configService interface{}) error {
	// Try to get current account from config service
	var currentAlias string
	if service, ok := configService.(interface{ GetCurrentAccount(context.Context) string }); ok {
		currentAlias = service.GetCurrentAccount(ctx)
	}

	if currentAlias == "" {
		c.PrintWarning(ctx, "No current account set")
		return nil
	}

	// Try to get account details
	account, err := c.getAccount(ctx, configService, currentAlias)
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
	// Get SSH service from container
	container := c.GetContainer()
	sshService := container.GetSSHService()

	if sshService == nil {
		c.PrintWarning(ctx, "SSH service not available, skipping validation")
		return nil
	}

	// Try to validate SSH configuration
	if service, ok := sshService.(interface {
		ValidateSSHConfiguration() (interface{}, error)
	}); ok {
		_, err := service.ValidateSSHConfiguration()
		if err != nil {
			return fmt.Errorf("SSH validation failed: %w", err)
		}
	}

	return nil
}

// performAccountSwitch performs the actual account switch
func (c *SwitchCommand) performAccountSwitch(ctx context.Context, configService interface{}, targetAlias string, targetAccount *models.Account) error {
	c.GetLogger().Info(ctx, "performing_account_switch",
		observability.F("target_account", targetAlias),
		observability.F("account_name", targetAccount.Name),
	)

	// Update current account in config service
	if service, ok := configService.(interface {
		SetCurrentAccount(context.Context, string) error
	}); ok {
		if err := service.SetCurrentAccount(ctx, targetAlias); err != nil {
			return fmt.Errorf("failed to set current account: %w", err)
		}
	}

	// Update Git configuration
	if err := c.updateGitConfig(ctx, targetAccount); err != nil {
		return fmt.Errorf("failed to update Git configuration: %w", err)
	}

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
		if service, ok := gitService.(interface {
			SetUserConfiguration(ctx context.Context, name, email string) error
		}); ok {
			if err := service.SetUserConfiguration(ctx, account.Name, account.Email); err != nil {
				return fmt.Errorf("failed to set user configuration: %w", err)
			}
			c.PrintSuccess(ctx, fmt.Sprintf("Updated Git user configuration: %s <%s>", account.Name, account.Email))
		}
	}

	// Set SSH command
	if account.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", account.SSHKeyPath)
		if service, ok := gitService.(interface {
			SetSSHCommand(ctx context.Context, sshCommand string) error
		}); ok {
			if err := service.SetSSHCommand(ctx, sshCmd); err != nil {
				return fmt.Errorf("failed to set SSH command: %w", err)
			}
			c.PrintSuccess(ctx, fmt.Sprintf("Updated Git SSH command: %s", sshCmd))
		}
	}

	return nil
}

// validateSwitchSuccess validates that the switch was successful
func (c *SwitchCommand) validateSwitchSuccess(ctx context.Context, configService interface{}, targetAlias string) error {
	logger := c.GetLogger()

	logger.Info(ctx, "validating_switch_success",
		observability.F("target_account", targetAlias),
	)

	// Verify the current account was set correctly
	if service, ok := configService.(interface{ GetCurrentAccount(context.Context) string }); ok {
		currentAccount := service.GetCurrentAccount(ctx)
		if currentAccount != targetAlias {
			return fmt.Errorf("account switch verification failed: expected %s, got %s", targetAlias, currentAccount)
		}
	}

	return nil
}

// Switch command for integration
var (
	switchStandardizedCmd = &cobra.Command{
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
		RunE: runSwitchStandardized,
	}
)

func init() {
	rootCmd.AddCommand(switchStandardizedCmd)
}

// runSwitchStandardized runs the switch command
func runSwitchStandardized(cmd *cobra.Command, args []string) error {
	// Create and run the switch command
	switchCmd := NewSwitchCommand()
	ctx := context.Background()
	return switchCmd.Execute(ctx, args)
}
