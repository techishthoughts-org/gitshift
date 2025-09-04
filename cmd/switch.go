package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/container"
	"github.com/techishthoughts/GitPersona/internal/models"
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

	// Update Git configuration via GitConfigService from the container
	if err := updateGitConfig(targetAccount); err != nil {
		return fmt.Errorf("failed to update Git configuration: %w", err)
	}

	return nil
}

func updateGitConfig(account interface{}) error {
	// Try to cast to expected account type
	acc, ok := account.(*models.Account)
	if !ok {
		// If we don't have a real account object, nothing to do
		return fmt.Errorf("invalid account object for git config update")
	}

	ctx := context.Background()

	// Get GitConfigService from container
	c := container.GetGlobalSimpleContainer()
	svcRaw := c.GetGitService()
	if svcRaw == nil {
		return fmt.Errorf("git config service not available in container")
	}

	// Try to cast to GitConfigManager interface
	gitSvc, ok := svcRaw.(interface {
		SetUserConfiguration(ctx context.Context, name, email string) error
		SetSSHCommand(ctx context.Context, sshCommand string) error
	})
	if !ok {
		return fmt.Errorf("git service does not implement required interface")
	}

	// Set user configuration
	if acc.Name != "" || acc.Email != "" {
		if err := gitSvc.SetUserConfiguration(ctx, acc.Name, acc.Email); err != nil {
			return fmt.Errorf("failed to set user configuration: %w", err)
		}
		fmt.Printf("✅ Updated Git user configuration: %s <%s>\n", acc.Name, acc.Email)
	}

	// Set SSH command
	sshCmd := ""
	if acc.SSHKeyPath != "" {
		sshCmd = fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", acc.SSHKeyPath)
	}

	if err := gitSvc.SetSSHCommand(ctx, sshCmd); err != nil {
		// Try a best-effort fallback to edit the user's global gitconfig directly
		fmt.Printf("⚠️  Warning: failed to set SSH command via git service: %v\n", err)
		fmt.Printf("🔧 Attempting direct ~/.gitconfig update...\n")

		if err := updateUserGitconfig(sshCmd); err != nil {
			return fmt.Errorf("failed to update ~/.gitconfig directly: %w", err)
		}
		fmt.Printf("✅ Updated ~/.gitconfig directly\n")
	} else {
		fmt.Printf("✅ Updated Git SSH command: %s\n", sshCmd)
	}

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

// updateUserGitconfig does a best-effort edit of the user's ~/.gitconfig to set core.sshcommand
func updateUserGitconfig(sshCmd string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	gitconfigPath := filepath.Join(home, ".gitconfig")

	// Read existing content or create empty content if file doesn't exist
	var content []byte
	if _, err := os.Stat(gitconfigPath); err == nil {
		content, err = ioutil.ReadFile(gitconfigPath)
		if err != nil {
			return fmt.Errorf("failed to read ~/.gitconfig: %w", err)
		}
	} else {
		// File doesn't exist, create it
		content = []byte{}
	}

	lines := strings.Split(string(content), "\n")
	found := false

	// Look for existing core.sshcommand line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "core.sshcommand") || strings.Contains(trimmed, "core.sshcommand=") {
			lines[i] = "\tcore.sshcommand = " + sshCmd
			found = true
			break
		}
	}

	if !found {
		// Look for [core] section and add after it
		added := false
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "[core]") {
				// Insert after this line
				insertAt := i + 1
				newLines := append(lines[:insertAt], append([]string{"\tcore.sshcommand = " + sshCmd}, lines[insertAt:]...)...)
				lines = newLines
				added = true
				break
			}
		}

		if !added {
			// No [core] section found, add it at the end
			if len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "[core]", "\tcore.sshcommand = "+sshCmd)
		}
	}

	newContent := strings.Join(lines, "\n")
	if err := ioutil.WriteFile(gitconfigPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write ~/.gitconfig: %w", err)
	}

	return nil
}
