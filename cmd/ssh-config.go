package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
	"github.com/techishthoughts/GitPersona/internal/validation"
)

// SSHConfigCommand handles SSH configuration management
type SSHConfigCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	generate bool
	backup   bool
	apply    bool
	account  string
}

// NewSSHConfigCommand creates a new SSH config command
func NewSSHConfigCommand() *SSHConfigCommand {
	cmd := &SSHConfigCommand{
		BaseCommand: commands.NewBaseCommand(
			"ssh-config",
			"⚙️  Manage SSH configuration to prevent key conflicts",
			"ssh-config [command]",
		).WithExamples(
			"gitpersona ssh-config generate",
			"gitpersona ssh-config generate --account work",
			"gitpersona ssh-config apply --backup",
		).WithFlags(
			commands.Flag{Name: "generate", Short: "g", Type: "bool", Default: false, Description: "Generate SSH configuration"},
			commands.Flag{Name: "backup", Short: "b", Type: "bool", Default: false, Description: "Create backup before applying changes"},
			commands.Flag{Name: "apply", Short: "a", Type: "bool", Default: false, Description: "Apply generated configuration to ~/.ssh/config"},
			commands.Flag{Name: "account", Short: "c", Type: "string", Default: "", Description: "Generate config for specific account only"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *SSHConfigCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.generate, _ = cmd.Flags().GetBool("generate")
		c.backup, _ = cmd.Flags().GetBool("backup")
		c.apply, _ = cmd.Flags().GetBool("apply")
		c.account, _ = cmd.Flags().GetString("account")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *SSHConfigCommand) Validate(args []string) error {
	// No validation needed for this command
	return nil
}

// Execute is the main entry point for the command
func (c *SSHConfigCommand) Execute(ctx context.Context, args []string) error {
	// Validate arguments
	if err := c.Validate(args); err != nil {
		return err
	}

	// Execute the command logic
	return c.Run(ctx, args)
}

// Run executes the SSH config command logic
func (c *SSHConfigCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get required services
	configService := container.GetConfigService()
	if configService == nil {
		return errors.New(errors.ErrCodeInternal, "config service not available")
	}

	// Load configuration
	if err := configService.Load(ctx); err != nil {
		return errors.ConfigLoadFailed(err, map[string]interface{}{
			"command": "ssh-config",
		})
	}

	if c.generate {
		return c.generateSSHConfig(ctx, configService)
	}

	if c.apply {
		return c.applySSHConfig(ctx, configService)
	}

	// Default: show current SSH config status
	return c.showSSHConfigStatus(ctx, configService)
}

// generateSSHConfig generates SSH configuration
func (c *SSHConfigCommand) generateSSHConfig(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "🔧 Generating SSH configuration...")

	// Get accounts
	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}
	if len(accounts) == 0 {
		return errors.New(errors.ErrCodeInternal, "no accounts configured")
	}

	// Filter by account if specified
	if c.account != "" {
		account, err := configService.GetAccount(ctx, c.account)
		if err != nil {
			return errors.AccountNotFound(c.account, map[string]interface{}{
				"command": "ssh-config",
			})
		}
		accounts = []*models.Account{account}
	}

	// Generate SSH config
	validator := validation.NewSSHValidator()
	var configContent string

	if len(accounts) == 1 {
		configContent = validator.GenerateSSHConfigForAccount(*accounts[0])
	} else {
		// Convert to slice of models.Account
		modelAccounts := make([]models.Account, len(accounts))
		for i, acc := range accounts {
			modelAccounts[i] = *acc
		}
		configContent = validator.GenerateSSHConfig(modelAccounts)
	}

	// Display the generated configuration
	c.PrintSuccess(ctx, "SSH configuration generated successfully")
	c.PrintInfo(ctx, "📄 Generated SSH Configuration:")
	fmt.Println()
	fmt.Println(configContent)

	// Show usage instructions
	c.PrintInfo(ctx, "💡 Usage Instructions:")
	if len(accounts) == 1 {
		c.PrintInfo(ctx, "  • This configuration will make the specified key the default for github.com")
		c.PrintInfo(ctx, "  • All git operations will use this key automatically")
	} else {
		c.PrintInfo(ctx, "  • Use host aliases for specific accounts:")
		for _, account := range accounts {
			if account.SSHKeyPath != "" {
				c.PrintInfo(ctx, fmt.Sprintf("    - git@github-%s:user/repo.git (for %s)", account.Alias, account.Name))
			}
		}
	}

	if c.apply {
		c.PrintInfo(ctx, "🔄 Applying configuration...")
		return c.applyGeneratedConfig(ctx, configContent)
	}

	return nil
}

// applySSHConfig applies SSH configuration
func (c *SSHConfigCommand) applySSHConfig(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "🔄 Applying SSH configuration...")

	// Generate configuration first
	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}
	if len(accounts) == 0 {
		return errors.New(errors.ErrCodeInternal, "no accounts configured")
	}

	validator := validation.NewSSHValidator()
	modelAccounts := make([]models.Account, len(accounts))
	for i, acc := range accounts {
		modelAccounts[i] = *acc
	}
	configContent := validator.GenerateSSHConfig(modelAccounts)

	return c.applyGeneratedConfig(ctx, configContent)
}

// applyGeneratedConfig applies the generated configuration to ~/.ssh/config
func (c *SSHConfigCommand) applyGeneratedConfig(ctx context.Context, configContent string) error {
	// Get SSH config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")

	// Create backup if requested
	if c.backup {
		if err := c.createBackup(ctx, sshConfigPath); err != nil {
			c.PrintWarning(ctx, "Failed to create backup, but continuing",
				observability.F("error", err.Error()),
			)
		}
	}

	// Ensure SSH directory exists
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Write configuration
	if err := os.WriteFile(sshConfigPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	c.PrintSuccess(ctx, "SSH configuration applied successfully",
		observability.F("path", sshConfigPath),
	)

	// Validate the applied configuration
	if err := c.validateAppliedConfig(ctx); err != nil {
		c.PrintWarning(ctx, "Configuration applied but validation failed",
			observability.F("error", err.Error()),
		)
	}

	return nil
}

// createBackup creates a backup of the current SSH config
func (c *SSHConfigCommand) createBackup(ctx context.Context, sshConfigPath string) error {
	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		// No existing config to backup
		return nil
	}

	backupPath := sshConfigPath + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
	if err := os.Rename(sshConfigPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	c.PrintSuccess(ctx, "Backup created successfully",
		observability.F("backup_path", backupPath),
	)
	return nil
}

// validateAppliedConfig validates the applied SSH configuration
func (c *SSHConfigCommand) validateAppliedConfig(ctx context.Context) error {
	validator := validation.NewSSHValidator()
	result, err := validator.ValidateSSHConfiguration()
	if err != nil {
		return fmt.Errorf("failed to validate SSH configuration: %w", err)
	}

	if !result.IsValid {
		c.PrintWarning(ctx, "SSH configuration has issues:")
		for _, issue := range result.Issues {
			c.PrintInfo(ctx, fmt.Sprintf("  • %s: %s", issue.Severity, issue.Description))
		}
	} else {
		c.PrintSuccess(ctx, "SSH configuration validation passed")
	}

	return nil
}

// showSSHConfigStatus shows the current SSH configuration status
func (c *SSHConfigCommand) showSSHConfigStatus(ctx context.Context, configService services.ConfigurationService) error {
	c.PrintInfo(ctx, "📊 SSH Configuration Status")

	// Get SSH config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")

	// Check if SSH config exists
	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		c.PrintWarning(ctx, "SSH config file does not exist",
			observability.F("path", sshConfigPath),
		)
		c.PrintInfo(ctx, "💡 Run 'gitpersona ssh-config generate --apply' to create one")
		return nil
	}

	// Validate current configuration
	validator := validation.NewSSHValidator()
	result, err := validator.ValidateSSHConfiguration()
	if err != nil {
		return fmt.Errorf("failed to validate SSH configuration: %w", err)
	}

	// Display status
	if result.IsValid {
		c.PrintSuccess(ctx, "SSH configuration is valid")
	} else {
		c.PrintWarning(ctx, "SSH configuration has issues")
		for _, issue := range result.Issues {
			severity := "⚠️"
			if issue.Severity == "critical" {
				severity = "🚨"
			} else if issue.Severity == "info" {
				severity = "ℹ️"
			}
			c.PrintInfo(ctx, fmt.Sprintf("  %s %s", severity, issue.Description))
		}
	}

	// Show recommendations
	if len(result.Recommendations) > 0 {
		c.PrintInfo(ctx, "💡 Recommendations:")
		for _, rec := range result.Recommendations {
			c.PrintInfo(ctx, fmt.Sprintf("  • %s", rec))
		}
	}

	return nil
}

// SSH config management command for integration
var (
	sshConfigManagementCmd = &cobra.Command{
		Use:     "ssh-config [command]",
		Aliases: []string{"sc", "sshconf"},
		Short:   "⚙️  Manage SSH configuration to prevent key conflicts",
		Long: `⚙️  Manage SSH Configuration to Prevent Key Conflicts

This command helps you manage your SSH configuration to prevent authentication
conflicts when using multiple GitHub accounts. It can generate, validate, and
apply SSH configurations that ensure the correct keys are used for each account.

Features:
- Generate SSH config entries for all or specific accounts
- Apply configurations with automatic backup
- Validate existing SSH configurations
- Detect and warn about potential conflicts
- Provide specific host aliases for multiple accounts

Examples:
  gitpersona ssh-config                    # Show current SSH config status
  gitpersona ssh-config --generate         # Generate SSH config for all accounts
  gitpersona ssh-config --generate --account work  # Generate for specific account
  gitpersona ssh-config --apply --backup   # Apply config with backup`,
		Args: cobra.NoArgs,
		RunE: runSSHConfig,
	}
)

func init() {
	// Add flags to the command
	sshConfigManagementCmd.Flags().BoolP("generate", "g", false, "Generate SSH configuration")
	sshConfigManagementCmd.Flags().BoolP("backup", "b", false, "Create backup before applying changes")
	sshConfigManagementCmd.Flags().BoolP("apply", "a", false, "Apply generated configuration to ~/.ssh/config")
	sshConfigManagementCmd.Flags().StringP("account", "c", "", "Generate config for specific account only")

	rootCmd.AddCommand(sshConfigManagementCmd)
}

// runSSHConfig runs the SSH config command
func runSSHConfig(cmd *cobra.Command, args []string) error {
	// Create and run the SSH config command
	sshConfigCmd := NewSSHConfigCommand()

	// Get flag values from cobra command
	generate, _ := cmd.Flags().GetBool("generate")
	backup, _ := cmd.Flags().GetBool("backup")
	apply, _ := cmd.Flags().GetBool("apply")
	account, _ := cmd.Flags().GetString("account")

	// Set flags on our command
	sshConfigCmd.generate = generate
	sshConfigCmd.backup = backup
	sshConfigCmd.apply = apply
	sshConfigCmd.account = account

	ctx := context.Background()
	return sshConfigCmd.Run(ctx, args)
}
