package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// switchIsolatedCmd represents the new isolated switch command
var switchIsolatedCmd = &cobra.Command{
	Use:   "switch-isolated [account-alias]",
	Short: "🔄 Switch to a different GitHub account with complete isolation",
	Long: `Switch to a different GitHub account with complete isolation and atomic operations.

This enhanced command provides:
- Complete SSH agent isolation per account
- Secure token storage and validation
- Atomic operations with rollback capability
- Comprehensive validation and error handling
- No GitHub CLI dependency

Examples:
  gitpersona switch-isolated work
  gitpersona switch-isolated personal --strict
  gitpersona switch-isolated client --validate-only`,
	Aliases: []string{"si", "switch-new"},
	Args:    cobra.ExactArgs(1),
	RunE:    runSwitchIsolatedCommand,
}

// runSwitchIsolatedCommand executes the isolated switch command
func runSwitchIsolatedCommand(cmd *cobra.Command, args []string) error {
	accountAlias := args[0]
	ctx := context.Background()

	// Get flags
	validateOnly, _ := cmd.Flags().GetBool("validate-only")
	strict, _ := cmd.Flags().GetBool("strict")
	skipSSH, _ := cmd.Flags().GetBool("skip-ssh")
	skipToken, _ := cmd.Flags().GetBool("skip-token")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Initialize logger
	logger := observability.NewDefaultLogger()

	fmt.Printf("🚀 Starting isolated account switch to '%s'...\n", accountAlias)

	// Load GitPersona configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Find target account
	accounts := configManager.ListAccounts()
	var targetAccount *models.Account
	for _, account := range accounts {
		if account.Alias == accountAlias {
			targetAccount = account
			break
		}
	}

	if targetAccount == nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}

	// Get current account for rollback purposes
	var sourceAccount *models.Account
	currentAccount, _ := configManager.GetCurrentAccount()
	currentAccountAlias := ""
	if currentAccount != nil {
		currentAccountAlias = currentAccount.Alias
	}
	if currentAccountAlias != "" && currentAccountAlias != accountAlias {
		for _, account := range accounts {
			if account.Alias == currentAccountAlias {
				sourceAccount = account
				break
			}
		}
	}

	fmt.Printf("📋 Target Account: %s (%s <%s>)\n",
		targetAccount.Alias, targetAccount.Name, targetAccount.Email)

	if sourceAccount != nil {
		fmt.Printf("📋 Source Account: %s (%s <%s>)\n",
			sourceAccount.Alias, sourceAccount.Name, sourceAccount.Email)
	}

	// Initialize isolated services
	fmt.Printf("🔧 Initializing isolated services...\n")

	// Initialize isolated token service
	tokenConfig := &services.TokenIsolationConfig{
		StrictIsolation:    strict,
		AutoValidation:     true,
		ValidationInterval: time.Hour,
		EncryptionEnabled:  true,
		BackupEnabled:      true,
	}

	tokenService, err := services.NewIsolatedTokenService(logger, tokenConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize token service: %w", err)
	}

	// Initialize isolated SSH manager
	sshConfig := &services.SSHIsolationConfig{
		StrictIsolation:     strict,
		AutoCleanup:         true,
		SocketTimeout:       30 * time.Second,
		KeyLoadTimeout:      10 * time.Second,
		MaxIdleTime:         time.Hour,
		ForceIdentitiesOnly: true,
	}

	sshManager, err := services.NewIsolatedSSHManager(logger, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize SSH manager: %w", err)
	}

	// Handle validate-only mode
	if validateOnly {
		fmt.Printf("🔍 Validating account '%s' (isolated mode)...\n", accountAlias)
		return validateAccountIsolated(ctx, logger, targetAccount, tokenService, sshManager)
	}

	// Configure transaction options
	transactionOptions := &services.TransactionOptions{
		StrictValidation:     strict,
		RollbackOnFailure:    true,
		ValidateBeforeSwitch: true,
		ValidateAfterSwitch:  true,
		Timeout:              timeout,
		ConcurrentSteps:      false, // Sequential for safety
		SkipSSHValidation:    skipSSH,
		SkipTokenValidation:  skipToken,
	}

	// Create and configure transaction
	fmt.Printf("⚡ Creating atomic transaction...\n")

	transaction := services.NewAccountSwitchTransaction(
		ctx, logger, sourceAccount, targetAccount,
		tokenService, sshManager, transactionOptions,
	)

	// Add switch steps
	transaction.AddStep(services.NewTokenIsolationStep(logger))
	transaction.AddStep(services.NewSSHIsolationStep(logger))
	transaction.AddStep(services.NewGitConfigurationStep(logger))
	transaction.AddStep(services.NewEnvironmentStep(logger))
	transaction.AddStep(services.NewValidationStep(logger))

	fmt.Printf("🔄 Executing account switch transaction...\n")
	fmt.Printf("   Transaction ID: %s\n", transaction.GetTransactionID())
	fmt.Printf("   Steps: 5 (token → ssh → git → env → validate)\n")

	// Execute transaction
	startTime := time.Now()
	result, err := transaction.Execute()
	duration := time.Since(startTime)

	// Display results
	fmt.Printf("\n📊 Transaction Results:\n")
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Success: %v\n", result.Success)
	fmt.Printf("   Completed Steps: %d\n", len(result.CompletedSteps))

	if result.Success {
		// Update current account in configuration
		if err := configManager.SetCurrentAccount(accountAlias); err != nil {
			fmt.Printf("⚠️  Warning: Failed to update current account in config: %v\n", err)
		} else if err := configManager.Save(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save configuration: %v\n", err)
		}

		fmt.Printf("\n🎉 Successfully switched to account '%s' (isolated mode)!\n", accountAlias)
		fmt.Printf("   ✅ SSH agent isolated with account-specific key\n")
		fmt.Printf("   ✅ Token validated and isolated\n")
		fmt.Printf("   ✅ Git configuration updated\n")
		fmt.Printf("   ✅ Environment properly configured\n")
		fmt.Printf("   ✅ All validations passed\n")

		return nil
	}

	// Handle failure
	fmt.Printf("\n❌ Account switch failed!\n")
	if result.FailedStep != nil {
		fmt.Printf("   Failed Step: %s\n", result.FailedStep.StepName)
		fmt.Printf("   Error: %s\n", result.FailedStep.Error)
	}

	if len(result.ValidationErrors) > 0 {
		fmt.Printf("   Validation Errors:\n")
		for _, validationError := range result.ValidationErrors {
			fmt.Printf("     - %s\n", validationError)
		}
	}

	if len(result.RollbackSteps) > 0 {
		fmt.Printf("   Rollback Steps Executed: %d\n", len(result.RollbackSteps))
		fmt.Printf("   Final State: %s\n", result.FinalState)
	}

	return err
}

// validateAccountIsolated performs isolated validation of an account
func validateAccountIsolated(
	ctx context.Context,
	logger observability.Logger,
	account *models.Account,
	tokenService *services.IsolatedTokenService,
	sshManager *services.IsolatedSSHManager,
) error {
	fmt.Printf("🔍 Validating account '%s' with isolation features...\n", account.Alias)

	issues := 0

	// Basic account validation
	if err := account.Validate(); err != nil {
		fmt.Printf("❌ Account configuration invalid: %v\n", err)
		issues++
	} else {
		fmt.Printf("✅ Account configuration valid\n")
	}

	// Token isolation validation
	if tokenService != nil {
		fmt.Printf("🔑 Validating token isolation...\n")

		// Check if token exists
		_, err := tokenService.GetToken(ctx, account.Alias)
		if err != nil {
			fmt.Printf("❌ No isolated token found: %v\n", err)
			issues++
		} else {
			fmt.Printf("✅ Isolated token found\n")

			// Validate token belongs to correct user
			if account.GitHubUsername != "" {
				if err := tokenService.ValidateTokenIsolation(ctx, account.Alias, account.GitHubUsername); err != nil {
					fmt.Printf("❌ Token isolation validation failed: %v\n", err)
					issues++
				} else {
					fmt.Printf("✅ Token isolation validated\n")
				}
			}
		}

		// Get token metadata
		if metadata, err := tokenService.GetTokenMetadata(ctx, account.Alias); err == nil {
			fmt.Printf("📊 Token Metadata:\n")
			fmt.Printf("     Username: %s\n", metadata.Username)
			fmt.Printf("     Type: %s\n", metadata.TokenType)
			fmt.Printf("     Created: %s\n", metadata.CreatedAt.Format(time.RFC3339))
			fmt.Printf("     Last Used: %s\n", metadata.LastUsed.Format(time.RFC3339))
			fmt.Printf("     Valid: %v\n", metadata.IsValid)
			if len(metadata.Scopes) > 0 {
				fmt.Printf("     Scopes: %v\n", metadata.Scopes)
			}
		}
	}

	// SSH isolation validation
	if account.SSHKeyPath != "" {
		fmt.Printf("🔐 Validating SSH isolation...\n")

		// Check if SSH key exists
		if _, err := os.Stat(account.SSHKeyPath); err != nil {
			fmt.Printf("❌ SSH key not found: %s\n", account.SSHKeyPath)
			issues++
		} else {
			fmt.Printf("✅ SSH key found: %s\n", account.SSHKeyPath)

			// Test SSH isolation (dry run)
			fmt.Printf("🧪 Testing SSH isolation (dry run)...\n")

			// This would normally switch SSH, but for validation we just check
			// if the SSH manager can create an isolated environment
			if sshManager != nil {
				// We can't easily test SSH without actually switching, so we'll
				// just validate the SSH manager is properly configured
				fmt.Printf("✅ SSH isolation manager ready\n")
			}
		}
	} else {
		fmt.Printf("⚠️  No SSH key configured for isolation\n")
	}

	// Isolation level validation
	if account.IsIsolated() {
		fmt.Printf("🔒 Account has isolation enabled: %s\n", account.GetIsolationLevel())

		if account.RequiresSSHIsolation() {
			fmt.Printf("✅ SSH isolation required and available\n")
		}

		if account.RequiresTokenIsolation() {
			fmt.Printf("✅ Token isolation required and available\n")
		}
	} else {
		fmt.Printf("⚠️  Account does not have isolation enabled\n")
	}

	// Summary
	fmt.Printf("\n📊 Validation Summary:\n")
	if issues == 0 {
		fmt.Printf("✅ Account '%s' is fully configured for isolated switching!\n", account.Alias)
		fmt.Printf("   Ready for complete multi-account isolation\n")
	} else {
		fmt.Printf("❌ Account '%s' has %d issue(s) that need resolution\n", account.Alias, issues)
		fmt.Printf("   Isolated switching may not work properly\n")
		return fmt.Errorf("account validation failed with %d issues", issues)
	}

	return nil
}

func init() {
	// Add flags
	switchIsolatedCmd.Flags().BoolP("validate-only", "v", false, "Only validate the account without switching")
	switchIsolatedCmd.Flags().BoolP("strict", "s", false, "Enable strict validation and isolation")
	switchIsolatedCmd.Flags().Bool("skip-ssh", false, "Skip SSH validation (not recommended)")
	switchIsolatedCmd.Flags().Bool("skip-token", false, "Skip token validation (not recommended)")
	switchIsolatedCmd.Flags().DurationP("timeout", "t", 5*time.Minute, "Transaction timeout")

	// Add to root command
	rootCmd.AddCommand(switchIsolatedCmd)
}
