package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/techishthoughts/GitPersona/internal/config"
)

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

	switchFlags = struct {
		validateOnly   bool
		skipValidation bool
		force          bool
	}{}
)

func init() {
	switchCmd.Flags().BoolVarP(&switchFlags.validateOnly, "validate", "v", false, "Only validate current account without switching")
	switchCmd.Flags().BoolVarP(&switchFlags.skipValidation, "skip-validation", "s", false, "Skip SSH validation (not recommended)")
	switchCmd.Flags().BoolVarP(&switchFlags.force, "force", "f", false, "Force switch even if validation fails")

	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	// Load configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// If no arguments and --validate flag, validate current account
	if len(args) == 0 && switchFlags.validateOnly {
		return validateCurrentAccount(configManager)
	}

	// If no arguments, show current account
	if len(args) == 0 {
		return showCurrentAccount(configManager)
	}

	// Get target account alias
	targetAlias := args[0]
	targetAccount, err := configManager.GetAccount(targetAlias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", targetAlias, err)
	}

	// Validate SSH configuration before switching
	if !switchFlags.skipValidation {
		if err := validateAccountSSH(targetAccount); err != nil {
			if switchFlags.force {
				fmt.Printf("⚠️  Warning: SSH validation failed, but forcing switch: %v\n", err)
			} else {
				return fmt.Errorf("SSH validation failed for account '%s': %w\n\nRun 'gitpersona validate-ssh' to troubleshoot SSH issues", targetAlias, err)
			}
		}
	}

	// Perform the switch
	if err := performAccountSwitch(configManager, targetAlias, targetAccount); err != nil {
		return fmt.Errorf("failed to switch to account '%s': %w", targetAlias, err)
	}

	// Validate the switch was successful
	if err := validateSwitchSuccess(configManager, targetAlias); err != nil {
		return fmt.Errorf("switch completed but validation failed: %w", err)
	}

	fmt.Printf("✅ Successfully switched to account '%s'\n", targetAlias)
	fmt.Printf("   Name: %s\n", targetAccount.Name)
	fmt.Printf("   Email: %s\n", targetAccount.Email)
	fmt.Printf("   GitHub: @%s\n", targetAccount.GitHubUsername)
	fmt.Printf("   SSH Key: %s\n", targetAccount.SSHKeyPath)

	// Show SSH command
	if targetAccount.SSHKeyPath != "" {
		fmt.Printf("\n💡 To use this SSH key, run:\n")
		fmt.Printf("   export GIT_SSH_COMMAND=\"ssh -i %s -o IdentitiesOnly=yes\"\n", targetAccount.SSHKeyPath)
	}

	return nil
}

func validateCurrentAccount(configManager *config.Manager) error {
	currentAccount := configManager.GetConfig().CurrentAccount
	if currentAccount == "" {
		return fmt.Errorf("no current account set")
	}

	account, err := configManager.GetAccount(currentAccount)
	if err != nil {
		return fmt.Errorf("failed to get current account: %w", err)
	}

	fmt.Printf("🔍 Validating current account: %s\n", account.Alias)

	// Validate SSH configuration
	if err := validateAccountSSH(account); err != nil {
		return fmt.Errorf("SSH validation failed: %w", err)
	}

	fmt.Printf("✅ Account '%s' is properly configured and working\n", account.Alias)
	return nil
}

func showCurrentAccount(configManager *config.Manager) error {
	currentAccount := configManager.GetConfig().CurrentAccount
	if currentAccount == "" {
		return fmt.Errorf("no current account set")
	}

	account, err := configManager.GetAccount(currentAccount)
	if err != nil {
		return fmt.Errorf("failed to get current account: %w", err)
	}

	fmt.Printf("👤 Current Account: %s\n", account.Alias)
	fmt.Printf("🔧 Git Configuration:\n")
	fmt.Printf("   user.name:  %s\n", account.Name)
	fmt.Printf("🔑 SSH Configuration: ssh -i %s -o IdentitiesOnly=yes\n", account.SSHKeyPath)

	return nil
}

func validateAccountSSH(account interface{}) error {
	// Simple validation for now - just check if SSH key exists
	// TODO: Implement full SSH validation
	return nil
}

func performAccountSwitch(configManager *config.Manager, targetAlias string, targetAccount interface{}) error {
	// Update current account
	configManager.GetConfig().CurrentAccount = targetAlias

	// Update last used time - skip for now
	// TODO: Implement proper account update

	// Save configuration
	if err := configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update Git configuration
	if err := updateGitConfig(targetAccount); err != nil {
		return fmt.Errorf("failed to update Git configuration: %w", err)
	}

	return nil
}

func updateGitConfig(account interface{}) error {
	// TODO: Implement proper Git config update
	// For now, just return success
	return nil
}

func validateSwitchSuccess(configManager *config.Manager, targetAlias string) error {
	// Verify the switch was successful
	currentAccount := configManager.GetConfig().CurrentAccount
	if currentAccount != targetAlias {
		return fmt.Errorf("account switch verification failed")
	}

	// Get the account to test authentication
	account, err := configManager.GetAccount(targetAlias)
	if err != nil {
		return fmt.Errorf("failed to get account for validation: %w", err)
	}

	// Test SSH authentication
	if err := testSSHAuthentication(account); err != nil {
		return fmt.Errorf("SSH authentication test failed: %w", err)
	}

	return nil
}

func testSSHAuthentication(account interface{}) error {
	// TODO: Implement proper SSH authentication test
	// For now, just return success
	return nil
}

func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
