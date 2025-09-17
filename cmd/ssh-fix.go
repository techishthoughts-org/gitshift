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
)

// SSHFixCommand handles automatic SSH key generation and configuration
type SSHFixCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	account string
	force   bool
}

// NewSSHFixCommand creates a new SSH fix command
func NewSSHFixCommand() *SSHFixCommand {
	cmd := &SSHFixCommand{
		BaseCommand: commands.NewBaseCommand(
			"ssh-fix",
			"🔧 Automatically fix SSH key issues for accounts",
			"ssh-fix [options]",
		).WithExamples(
			"gitpersona ssh-fix",
			"gitpersona ssh-fix --account work",
			"gitpersona ssh-fix --force",
		).WithFlags(
			commands.Flag{Name: "account", Short: "a", Type: "string", Default: "", Description: "Fix SSH key for specific account"},
			commands.Flag{Name: "force", Short: "f", Type: "bool", Default: false, Description: "Force regeneration of SSH keys"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *SSHFixCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.account = c.GetFlagString(cmd, "account")
		c.force = c.GetFlagBool(cmd, "force")

		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *SSHFixCommand) Validate(args []string) error {
	// No validation needed for this command
	return nil
}

// Run executes the SSH fix command logic
func (c *SSHFixCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get required services
	configService := container.GetConfigService()
	if configService == nil {
		return errors.New(errors.ErrCodeInternal, "config service not available")
	}

	// Load configuration
	if err := configService.Load(ctx); err != nil {
		return errors.ConfigLoadFailed(err, map[string]interface{}{
			"command": "ssh-fix",
		})
	}

	c.PrintInfo(ctx, "🔧 Starting SSH key fix process...")

	// Get accounts to fix
	accounts, err := configService.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		c.PrintWarning(ctx, "No accounts configured")
		c.PrintInfo(ctx, "💡 Add accounts first with: gitpersona add-github USERNAME")
		return nil
	}

	// Filter by account if specified
	if c.account != "" {
		account, err := configService.GetAccount(ctx, c.account)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", c.account, err)
		}
		accounts = []*models.Account{account}
	}

	// Fix each account
	fixedCount := 0
	for _, account := range accounts {
		if err := c.fixAccountSSH(ctx, account); err != nil {
			c.PrintError(ctx, "Failed to fix SSH for account",
				observability.F("account", account.Alias),
				observability.F("error", err.Error()),
			)
		} else {
			fixedCount++
		}
	}

	c.PrintSuccess(ctx, fmt.Sprintf("Fixed SSH keys for %d account(s)", fixedCount))
	return nil
}

// Execute is the main entry point for the command
func (c *SSHFixCommand) Execute(ctx context.Context, args []string) error {
	// Validate arguments
	if err := c.Validate(args); err != nil {
		return err
	}

	// Execute the command logic
	return c.Run(ctx, args)
}

// fixAccountSSH fixes SSH key issues for a specific account
func (c *SSHFixCommand) fixAccountSSH(ctx context.Context, account *models.Account) error {
	c.PrintInfo(ctx, fmt.Sprintf("🔍 Checking account '%s' (%s)...", account.Alias, account.GitHubUsername))

	// Check if SSH key exists and is valid
	needsNewKey := false
	if account.SSHKeyPath == "" {
		c.PrintWarning(ctx, "No SSH key configured for account")
		needsNewKey = true
	} else if _, err := os.Stat(account.SSHKeyPath); os.IsNotExist(err) {
		c.PrintWarning(ctx, "SSH key file not found",
			observability.F("ssh_key_path", account.SSHKeyPath),
		)
		needsNewKey = true
	} else if c.force {
		c.PrintInfo(ctx, "Force regeneration requested")
		needsNewKey = true
	}

	if !needsNewKey {
		c.PrintSuccess(ctx, "SSH key is already configured and exists")
		return nil
	}

	// Generate new SSH key
	return c.generateSSHKeyForAccount(ctx, account)
}

// generateSSHKeyForAccount generates a new SSH key for the account
func (c *SSHFixCommand) generateSSHKeyForAccount(ctx context.Context, account *models.Account) error {
	c.PrintInfo(ctx, "🔑 Generating new SSH key...")

	// Determine key path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	keyPath := filepath.Join(sshDir, fmt.Sprintf("id_ed25519_%s", account.Alias))

	// Remove existing key if force is enabled
	if c.force {
		_ = os.Remove(keyPath)
		_ = os.Remove(keyPath + ".pub")
	}

	// Generate SSH key
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", account.Email, "-f", keyPath, "-N", "")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %w, output: %s", err, string(output))
	}

	c.PrintSuccess(ctx, "SSH key generated successfully",
		observability.F("ssh_key_path", keyPath),
	)

	// Update account configuration
	account.SSHKeyPath = keyPath
	configService := c.GetContainer().GetConfigService()
	if err := configService.SetAccount(ctx, account); err != nil {
		c.PrintWarning(ctx, "Failed to save account configuration",
			observability.F("error", err.Error()),
		)
	} else {
		c.PrintSuccess(ctx, "Account configuration updated")
	}

	// Try to add to GitHub if authenticated
	if c.isGitHubCLIAuthenticated() {
		c.PrintInfo(ctx, "🚀 Adding SSH key to GitHub account...")
		addCmd := exec.Command("gh", "ssh-key", "add", keyPath+".pub", "--title", fmt.Sprintf("gitpersona-%s-%s", account.Alias, account.GitHubUsername))
		if err := addCmd.Run(); err != nil {
			c.PrintWarning(ctx, "Failed to add SSH key to GitHub",
				observability.F("error", err.Error()),
			)
			c.PrintInfo(ctx, "💡 Please add this key manually: https://github.com/settings/keys")
		} else {
			c.PrintSuccess(ctx, "SSH key added to GitHub account!")
		}
	} else {
		c.PrintInfo(ctx, "💡 Please add this SSH key to your GitHub account:")
		c.PrintInfo(ctx, "   https://github.com/settings/keys")
	}

	// Test the new key
	c.PrintInfo(ctx, "🧪 Testing SSH connection...")
	testCmd := exec.Command("ssh", "-T", "git@github.com", "-i", keyPath, "-o", "IdentitiesOnly=yes", "-o", "ConnectTimeout=10")
	testOutput, _ := testCmd.CombinedOutput()
	if strings.Contains(string(testOutput), "successfully authenticated") || strings.Contains(string(testOutput), "Hi ") {
		c.PrintSuccess(ctx, "SSH connection test successful!")
	} else {
		c.PrintWarning(ctx, "SSH connection test failed",
			observability.F("output", string(testOutput)),
		)
	}

	return nil
}

// isGitHubCLIAuthenticated checks if GitHub CLI is authenticated
func (c *SSHFixCommand) isGitHubCLIAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

// SSH fix command for integration
var (
	sshFixCmd = &cobra.Command{
		Use:     "ssh-fix [options]",
		Aliases: []string{"ssh-repair", "ssh-generate"},
		Short:   "🔧 Automatically fix SSH key issues for accounts",
		Long: `🔧 Automatically Fix SSH Key Issues

This command automatically detects and fixes SSH key problems for your configured
GitHub accounts. It can generate missing SSH keys, update account configurations,
and even add keys to your GitHub account automatically.

Features:
- Detect missing or invalid SSH keys
- Generate new ED25519 SSH keys
- Update account configurations automatically
- Add keys to GitHub account (if authenticated)
- Test SSH connectivity
- Force regeneration of existing keys

Common Use Cases:
- Fix "SSH key not found" errors
- Regenerate SSH keys after system changes
- Set up SSH keys for newly discovered accounts
- Repair broken SSH configurations

Examples:
  gitpersona ssh-fix                    # Fix all accounts
  gitpersona ssh-fix --account work     # Fix specific account
  gitpersona ssh-fix --force            # Force regenerate all keys`,
		Args: cobra.NoArgs,
		RunE: runSSHFix,
	}
)

func init() {
	// Add flags to the command
	sshFixCmd.Flags().StringP("account", "a", "", "Fix SSH key for specific account")
	sshFixCmd.Flags().BoolP("force", "f", false, "Force regeneration of SSH keys")

	rootCmd.AddCommand(sshFixCmd)
}

// runSSHFix runs the SSH fix command
func runSSHFix(cmd *cobra.Command, args []string) error {
	// Create and run the SSH fix command
	sshFixCmd := NewSSHFixCommand()
	ctx := context.Background()
	return sshFixCmd.Execute(ctx, args)
}
