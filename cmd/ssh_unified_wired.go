package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// Global service container - initialized during app startup
var serviceContainer *internal.ServiceContainer

// InitializeServices initializes the service container
func InitializeServices() error {
	logger := observability.NewLogger(observability.LogLevelInfo)
	serviceContainer = internal.NewServiceContainer(logger)
	return nil
}

// Wire all SSH command handlers to use CoreServices
func init() {
	// Ensure services are initialized
	if serviceContainer == nil {
		InitializeServices()
	}
}

// Updated SSH command handlers using CoreServices

func sshKeysListHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	keys, err := services.SSH.ListKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	if jsonOutput {
		return outputKeysJSON(keys)
	}

	fmt.Printf("🔑 SSH Keys (%d found)\n", len(keys))
	fmt.Println(strings.Repeat("=", 50))

	for _, key := range keys {
		status := "❌"
		if key.Valid {
			status = "✅"
		}

		fmt.Printf("%s %s\n", status, key.Path)
		if verbose || !key.Valid {
			fmt.Printf("   Type: %s, Size: %d bits\n", key.Type, key.Size)
			fmt.Printf("   Fingerprint: %s\n", key.Fingerprint)
			if key.Email != "" {
				fmt.Printf("   Email: %s\n", key.Email)
			}
			if !key.Valid {
				fmt.Printf("   ⚠️  Issues: Key validation failed\n")
			}
		}
		fmt.Println()
	}

	return nil
}

func sshKeysGenerateHandler(cmd *cobra.Command, args []string) error {
	keyType, _ := cmd.Flags().GetString("type")
	accountName, _ := cmd.Flags().GetString("account")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Get current account if not specified
	var account *internal.Account
	var err error
	if accountName != "" {
		account, err = services.Account.GetAccount(ctx, accountName)
	} else {
		account, err = services.Account.GetCurrentAccount(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Generate SSH key path
	keyPath := generateSSHKeyPath(account.Alias, keyType)

	req := internal.GenerateKeyRequest{
		Type:      keyType,
		Email:     account.Email,
		KeyPath:   keyPath,
		Overwrite: false,
	}

	fmt.Printf("🔨 Generating %s SSH key for account '%s'\n", keyType, account.Alias)
	fmt.Printf("📧 Email: %s\n", account.Email)
	fmt.Printf("🔑 Key path: %s\n", keyPath)
	fmt.Println()

	key, err := services.SSH.GenerateKey(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	fmt.Printf("✅ SSH key generated successfully!\n")
	fmt.Printf("🔑 Private key: %s\n", key.Path)
	fmt.Printf("🔓 Public key: %s.pub\n", key.Path)
	fmt.Printf("🔒 Type: %s\n", key.Type)
	fmt.Printf("🆔 Fingerprint: %s\n", key.Fingerprint)
	fmt.Println()

	// Update account with new key path
	updates := internal.AccountUpdates{
		SSHKeyPath: &key.Path,
	}

	if err := services.Account.UpdateAccount(ctx, account.Alias, updates); err != nil {
		fmt.Printf("⚠️  Warning: Failed to update account with new key path: %v\n", err)
	} else {
		fmt.Printf("✅ Account '%s' updated with new SSH key\n", account.Alias)
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Printf("   1. Upload public key to GitHub: gitpersona ssh keys upload %s\n", key.Path)
	fmt.Printf("   2. Test connectivity: gitpersona ssh test %s\n", account.Alias)

	return nil
}

func sshKeysDeleteHandler(cmd *cobra.Command, args []string) error {
	keyPath := args[0]
	force, _ := cmd.Flags().GetBool("force")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Validate key exists
	keyInfo, err := services.SSH.ValidateKey(ctx, keyPath)
	if err != nil {
		return fmt.Errorf("SSH key not found or invalid: %w", err)
	}

	fmt.Printf("🗑️  Deleting SSH key: %s\n", keyPath)
	fmt.Printf("🔒 Type: %s (%d bits)\n", keyInfo.Type, keyInfo.Size)
	fmt.Printf("🆔 Fingerprint: %s\n", keyInfo.Fingerprint)

	if !force {
		fmt.Print("\n⚠️  This action cannot be undone. Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Operation cancelled")
			return nil
		}
	}

	if err := services.SSH.DeleteKey(ctx, keyPath); err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	fmt.Printf("✅ SSH key deleted successfully\n")
	fmt.Printf("💡 Remember to remove the key from GitHub: https://github.com/settings/keys\n")

	return nil
}

func sshKeysValidateHandler(cmd *cobra.Command, args []string) error {
	keyPath := args[0]
	verbose, _ := cmd.Flags().GetBool("verbose")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("✅ Validating SSH key: %s\n", keyPath)
	fmt.Println()

	keyInfo, err := services.SSH.ValidateKey(ctx, keyPath)
	if err != nil {
		fmt.Printf("❌ Key validation failed: %v\n", err)
		return err
	}

	status := "✅ Valid"
	if !keyInfo.Valid {
		status = "❌ Invalid"
	}

	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Type: %s (%d bits)\n", keyInfo.Type, keyInfo.Size)
	fmt.Printf("Fingerprint: %s\n", keyInfo.Fingerprint)
	if keyInfo.Email != "" {
		fmt.Printf("Email: %s\n", keyInfo.Email)
	}

	if verbose {
		fmt.Println("\n📊 Detailed Information:")
		fmt.Printf("   • Private key: %s (exists: %v, readable: %v)\n",
			keyInfo.Path, keyInfo.Exists, keyInfo.Readable)
		fmt.Printf("   • Public key: %s.pub\n", keyInfo.Path)

		// Check permissions
		if info, err := os.Stat(keyInfo.Path); err == nil {
			perm := info.Mode().Perm()
			permStatus := "✅ Secure (600)"
			if perm != 0600 {
				permStatus = fmt.Sprintf("⚠️  Insecure (%o)", perm)
			}
			fmt.Printf("   • Permissions: %s\n", permStatus)
		}
	}

	if keyInfo.Valid {
		fmt.Println("\n🎉 SSH key is valid and ready to use!")
	}

	return nil
}

func sshTestHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetInt("timeout")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	var account *internal.Account
	var err error

	if len(args) > 0 {
		// Test specific account
		account, err = services.Account.GetAccount(ctx, args[0])
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", args[0], err)
		}
	} else {
		// Test current account
		account, err = services.Account.GetCurrentAccount(ctx)
		if err != nil {
			return fmt.Errorf("no current account set: %w", err)
		}
	}

	fmt.Printf("🔍 Testing SSH connectivity for account '%s'\n", account.Alias)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("👤 Name: %s\n", account.Name)
	fmt.Printf("📧 Email: %s\n", account.Email)
	if account.GitHubUsername != "" {
		fmt.Printf("🐙 GitHub: @%s\n", account.GitHubUsername)
	}
	fmt.Printf("🔑 SSH Key: %s\n", account.SSHKeyPath)
	fmt.Printf("⏱️  Timeout: %d seconds\n", timeout)
	fmt.Println()

	result, err := services.SSH.TestConnectivity(ctx, account)
	if err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}

	if result.Success {
		fmt.Printf("✅ SSH connectivity successful!\n")
		fmt.Printf("⚡ Latency: %dms\n", result.Latency)

		if githubUser, exists := result.Details["github_username"]; exists {
			fmt.Printf("👋 GitHub says: Hi %s!\n", githubUser)
		}
	} else {
		fmt.Printf("❌ SSH connectivity failed\n")
		fmt.Printf("💔 Error: %s\n", result.Message)

		if verbose {
			if output, exists := result.Details["output"]; exists {
				fmt.Printf("\n📋 SSH Output:\n%s\n", output)
			}
		}
	}

	if verbose && len(result.Details) > 0 {
		fmt.Println("\n📊 Additional Details:")
		for key, value := range result.Details {
			if key != "output" { // Already shown above
				fmt.Printf("   • %s: %v\n", key, value)
			}
		}
	}

	return nil
}

func sshDiagnoseHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	full, _ := cmd.Flags().GetBool("full")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🏥 SSH Diagnostics\n")
	fmt.Println(strings.Repeat("=", 30))
	if full {
		fmt.Println("Running comprehensive SSH diagnostics...")
	} else {
		fmt.Println("Running basic SSH diagnostics...")
	}
	fmt.Println()

	issues, err := services.SSH.DiagnoseIssues(ctx)
	if err != nil {
		return fmt.Errorf("failed to diagnose SSH issues: %w", err)
	}

	if jsonOutput {
		return outputIssuesJSON(issues)
	}

	if len(issues) == 0 {
		fmt.Printf("🎉 No SSH issues detected! Everything looks good.\n")
		return nil
	}

	fmt.Printf("⚠️  Found %d SSH issues:\n\n", len(issues))

	criticalCount := 0
	for i, issue := range issues {
		severity := getSeverityIcon(issue.Severity)
		fmt.Printf("%d. %s %s\n", i+1, severity, issue.Description)

		if verbose || issue.Severity == "high" {
			fmt.Printf("   💡 Fix: %s\n", issue.Fix)
			if issue.AutoFixable {
				fmt.Printf("   🤖 Auto-fixable: Yes\n")
			}
		}

		if issue.Severity == "high" {
			criticalCount++
		}
		fmt.Println()
	}

	fmt.Printf("📊 Summary: %d issues (%d critical)\n", len(issues), criticalCount)

	if criticalCount > 0 {
		fmt.Printf("🚨 Critical issues require immediate attention!\n")
	}

	autoFixableCount := 0
	for _, issue := range issues {
		if issue.AutoFixable {
			autoFixableCount++
		}
	}

	if autoFixableCount > 0 {
		fmt.Printf("💡 %d issues can be auto-fixed with: gitpersona ssh fix --auto\n", autoFixableCount)
	}

	return nil
}

func sshFixHandler(cmd *cobra.Command, args []string) error {
	auto, _ := cmd.Flags().GetBool("auto")
	force, _ := cmd.Flags().GetBool("force")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🔧 SSH Auto-Fix\n")
	fmt.Println(strings.Repeat("=", 20))

	// First, diagnose issues
	issues, err := services.SSH.DiagnoseIssues(ctx)
	if err != nil {
		return fmt.Errorf("failed to diagnose issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Printf("🎉 No SSH issues found to fix!\n")
		return nil
	}

	autoFixableIssues := make([]*internal.SSHIssue, 0)
	for _, issue := range issues {
		if issue.AutoFixable {
			autoFixableIssues = append(autoFixableIssues, issue)
		}
	}

	fmt.Printf("Found %d issues, %d auto-fixable\n\n", len(issues), len(autoFixableIssues))

	if len(autoFixableIssues) == 0 {
		fmt.Printf("❌ No auto-fixable issues found\n")
		fmt.Printf("💡 Manual fixes required - run 'gitpersona ssh diagnose --verbose' for details\n")
		return nil
	}

	if !auto && !force {
		fmt.Printf("Auto-fixable issues:\n")
		for i, issue := range autoFixableIssues {
			fmt.Printf("  %d. %s\n", i+1, issue.Description)
		}

		fmt.Printf("\nProceed with auto-fix? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Auto-fix cancelled")
			return nil
		}
	}

	fmt.Printf("🔧 Applying fixes...\n\n")

	if err := services.SSH.FixIssues(ctx, autoFixableIssues); err != nil {
		return fmt.Errorf("failed to fix issues: %w", err)
	}

	// Count fixed issues
	fixedCount := 0
	for _, issue := range autoFixableIssues {
		if issue.Fixed {
			fixedCount++
			fmt.Printf("✅ Fixed: %s\n", issue.Description)
		} else {
			fmt.Printf("❌ Failed to fix: %s\n", issue.Description)
		}
	}

	fmt.Printf("\n🎉 Auto-fix completed: %d/%d issues resolved\n", fixedCount, len(autoFixableIssues))

	if fixedCount < len(autoFixableIssues) {
		fmt.Printf("⚠️  Some issues could not be fixed automatically\n")
	}

	return nil
}

// Helper functions

func generateSSHKeyPath(alias, keyType string) string {
	homeDir, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.ssh/id_%s_%s", homeDir, keyType, alias)
}

func getSeverityIcon(severity string) string {
	switch severity {
	case "high":
		return "🚨"
	case "medium":
		return "⚠️"
	case "low":
		return "ℹ️"
	default:
		return "•"
	}
}

func outputKeysJSON(keys []*internal.SSHKeyInfo) error {
	// TODO: Implement JSON output for keys
	fmt.Printf("JSON output not implemented yet\n")
	return nil
}

func outputIssuesJSON(issues []*internal.SSHIssue) error {
	// TODO: Implement JSON output for issues
	fmt.Printf("JSON output not implemented yet\n")
	return nil
}
