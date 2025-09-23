package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

var environmentCmd = &cobra.Command{
	Use:   "environment",
	Short: "Manage GitPersona environment configuration",
	Long: `Manage GitPersona environment configuration for MCP servers and shell integration.

This command helps configure the environment so that GitPersona tokens
are available to MCP servers and shell sessions without external dependencies.`,
}

var setupEnvCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up GitPersona environment configuration",
	Long: `Set up GitPersona environment configuration.

This command configures:
- MCP server environment variables
- Shell environment integration
- Token management for the current account`,
	RunE: runSetupEnvironment,
}

var validateEnvCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate GitPersona environment configuration",
	Long:  `Validate the current GitPersona environment setup and provide recommendations.`,
	RunE:  runValidateEnvironment,
}

var cleanupEnvCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up GitPersona environment configuration",
	Long:  `Clean up GitPersona environment configuration and remove outdated files.`,
	RunE:  runCleanupEnvironment,
}

var statusEnvCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GitPersona environment status",
	Long:  `Show the current status of GitPersona environment configuration.`,
	RunE:  runEnvironmentStatus,
}

func init() {
	rootCmd.AddCommand(environmentCmd)

	environmentCmd.AddCommand(setupEnvCmd)
	environmentCmd.AddCommand(validateEnvCmd)
	environmentCmd.AddCommand(cleanupEnvCmd)
	environmentCmd.AddCommand(statusEnvCmd)

	// Flags for setup command
	setupEnvCmd.Flags().StringP("account", "a", "", "Account to set up environment for")
	setupEnvCmd.Flags().BoolP("force", "f", false, "Force setup even if configuration exists")
	setupEnvCmd.Flags().BoolP("mcp-only", "m", false, "Only set up MCP server configuration")
	setupEnvCmd.Flags().BoolP("shell-only", "s", false, "Only set up shell environment")
}

func runSetupEnvironment(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Get account
	account, _ := cmd.Flags().GetString("account")
	if account == "" {
		// Try to get current account
		configManager := config.NewManager()
		if err := configManager.Load(); err == nil {
			if currentAccount, err := configManager.GetCurrentAccount(); err == nil {
				account = currentAccount.Alias
			}
		}
		if account == "" {
			account = "default"
		}
	}

	// Initialize services
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	environmentService := services.NewEnvironmentService(logger, tokenStorage)

	// Get token for account
	token, err := tokenStorage.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("no token found for account '%s'. Run 'gitpersona github-token set %s' first", account, account)
	}

	// Check flags
	mcpOnly, _ := cmd.Flags().GetBool("mcp-only")
	shellOnly, _ := cmd.Flags().GetBool("shell-only")
	force, _ := cmd.Flags().GetBool("force")

	fmt.Printf("Setting up GitPersona environment for account: %s\n", account)

	// Set up MCP server configuration
	if !shellOnly {
		fmt.Printf("📋 Configuring MCP servers...\n")
		if err := environmentService.UpdateMCPServerConfig(ctx, account, token); err != nil {
			if !force {
				return fmt.Errorf("failed to update MCP server config: %w", err)
			}
			fmt.Printf("⚠️  Warning: Failed to update MCP server config: %v\n", err)
		} else {
			fmt.Printf("✅ MCP server configuration updated\n")
		}
	}

	// Set up shell environment
	if !mcpOnly {
		fmt.Printf("🐚 Configuring shell environment...\n")
		if err := environmentService.UpdateShellEnvironment(ctx, account, token); err != nil {
			if !force {
				return fmt.Errorf("failed to update shell environment: %w", err)
			}
			fmt.Printf("⚠️  Warning: Failed to update shell environment: %v\n", err)
		} else {
			fmt.Printf("✅ Shell environment configuration updated\n")
		}
	}

	fmt.Printf("\n🎉 Environment setup completed for account: %s\n", account)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Restart Claude Code to reload MCP servers\n")
	fmt.Printf("2. Source your shell configuration: source ~/.config/gitpersona/environment\n")
	fmt.Printf("3. Run 'gitpersona environment validate' to verify setup\n")

	return nil
}

func runValidateEnvironment(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Initialize services
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	environmentService := services.NewEnvironmentService(logger, tokenStorage)

	// Validate environment
	result, err := environmentService.ValidateEnvironmentSetup(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate environment: %w", err)
	}

	fmt.Printf("GitPersona Environment Validation Results\n")
	fmt.Printf("==========================================\n\n")

	// MCP Configuration
	fmt.Printf("📋 MCP Server Configuration:\n")
	if result.MCPConfigExists {
		fmt.Printf("  ✅ MCP configuration found\n")
		for _, path := range result.MCPConfigPaths {
			fmt.Printf("     - %s\n", path)
		}
	} else {
		fmt.Printf("  ❌ No MCP configuration found\n")
	}

	// Shell Configuration
	fmt.Printf("\n🐚 Shell Environment:\n")
	if result.ShellConfigExists {
		fmt.Printf("  ✅ Shell configuration found\n")
	} else {
		fmt.Printf("  ❌ No shell configuration found\n")
	}

	// Token Status
	fmt.Printf("\n🔑 Token Status:\n")
	if result.CurrentToken != "" {
		fmt.Printf("  ✅ GitHub token available (%s)\n", result.CurrentToken)
		fmt.Printf("  📍 Source: %s\n", result.TokenSource)
	} else {
		fmt.Printf("  ❌ No GitHub token found\n")
	}

	// Issues
	if len(result.Issues) > 0 {
		fmt.Printf("\n⚠️  Issues Found:\n")
		for _, issue := range result.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	// Recommendations
	if len(result.Recommendations) > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		for _, rec := range result.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}

	if len(result.Issues) == 0 && result.MCPConfigExists && result.ShellConfigExists {
		fmt.Printf("\n🎉 Environment validation passed! Everything looks good.\n")
	}

	return nil
}

func runCleanupEnvironment(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Initialize services
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	environmentService := services.NewEnvironmentService(logger, tokenStorage)

	fmt.Printf("🧹 Cleaning up GitPersona environment configuration...\n")

	// Get list of accounts with tokens
	accounts, err := tokenStorage.ListTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}

	// Clean up configuration for each account
	for _, account := range accounts {
		fmt.Printf("  Cleaning up configuration for account: %s\n", account)
		if err := environmentService.CleanupMCPConfig(ctx, account); err != nil {
			fmt.Printf("    ⚠️  Warning: Failed to cleanup config for %s: %v\n", account, err)
		}
	}

	fmt.Printf("✅ Environment cleanup completed\n")
	fmt.Printf("\nNote: You may need to restart Claude Code and reload your shell\n")

	return nil
}

func runEnvironmentStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Initialize services
	tokenStorage, err := services.NewTokenStorageService(logger)
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	environmentService := services.NewEnvironmentService(logger, tokenStorage)

	fmt.Printf("GitPersona Environment Status\n")
	fmt.Printf("============================\n\n")

	// Get current account
	currentAccount := "unknown"
	configManager := config.NewManager()
	if err := configManager.Load(); err == nil {
		if account, err := configManager.GetCurrentAccount(); err == nil {
			currentAccount = account.Alias
		}
	}

	fmt.Printf("📊 Current Account: %s\n\n", currentAccount)

	// Validate environment
	result, err := environmentService.ValidateEnvironmentSetup(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to validate environment: %v\n", err)
		return nil
	}

	// Quick status overview
	fmt.Printf("🔍 Quick Status:\n")
	fmt.Printf("  MCP Configuration: %s\n", getEnvironmentStatusIcon(result.MCPConfigExists))
	fmt.Printf("  Shell Environment: %s\n", getEnvironmentStatusIcon(result.ShellConfigExists))
	fmt.Printf("  GitHub Token: %s\n", getEnvironmentStatusIcon(result.CurrentToken != ""))

	// Token information
	if result.CurrentToken != "" {
		fmt.Printf("\n🔑 Token Information:\n")
		fmt.Printf("  Token: %s\n", result.CurrentToken)
		fmt.Printf("  Source: %s\n", result.TokenSource)
	}

	// List stored tokens
	accounts, err := tokenStorage.ListTokens(ctx)
	if err == nil && len(accounts) > 0 {
		fmt.Printf("\n💾 Stored Tokens:\n")
		for _, account := range accounts {
			if metadata, err := tokenStorage.GetTokenMetadata(ctx, account); err == nil {
				fmt.Printf("  %s (%s)\n", account, metadata.TokenPrefix)
			} else {
				fmt.Printf("  %s\n", account)
			}
		}
	}

	// Overall status
	if result.MCPConfigExists && result.ShellConfigExists && result.CurrentToken != "" {
		fmt.Printf("\n🎉 Status: All systems operational!\n")
	} else {
		fmt.Printf("\n⚠️  Status: Configuration needed\n")
		fmt.Printf("Run 'gitpersona environment validate' for detailed recommendations\n")
	}

	return nil
}

func getEnvironmentStatusIcon(status bool) string {
	if status {
		return "✅ Configured"
	}
	return "❌ Not configured"
}
