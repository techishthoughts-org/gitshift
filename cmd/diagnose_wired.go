package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal"
)

// Wire diagnostic command handlers to CoreServices

func diagnoseBasicHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🔍 Basic System Diagnostics\n")
	fmt.Println(strings.Repeat("=", 28))

	// Basic health check
	if err := services.System.PerformHealthCheck(ctx); err != nil {
		fmt.Printf("❌ System health check failed: %v\n", err)
		return err
	}

	fmt.Printf("✅ System health check passed\n")

	// Get system info
	if sysInfo, err := services.System.GetSystemInfo(ctx); err == nil {
		fmt.Printf("\n📊 System Information:\n")
		fmt.Printf("   • Platform: %s\n", sysInfo.Platform)
		fmt.Printf("   • Version: %s\n", sysInfo.Version)
		fmt.Printf("   • Git: %s\n", sysInfo.GitVersion)
		fmt.Printf("   • SSH: %s\n", sysInfo.SSHVersion)
	}

	// Quick account check
	if accounts, err := services.Account.ListAccounts(ctx); err == nil {
		fmt.Printf("\n👥 Accounts: %d configured\n", len(accounts))

		if current, err := services.Account.GetCurrentAccount(ctx); err == nil {
			fmt.Printf("   • Current: %s\n", current.Alias)
		} else {
			fmt.Printf("   • Current: None active\n")
		}
	}

	if verbose || jsonOutput {
		return diagnoseFullHandler(cmd, args)
	}

	fmt.Printf("\n💡 Run with --detailed for comprehensive diagnostics\n")
	return nil
}

func diagnoseFullHandler(cmd *cobra.Command, args []string) error {
	parallel, _ := cmd.Flags().GetBool("parallel")
	timeout, _ := cmd.Flags().GetInt("timeout")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🔬 Comprehensive System Diagnostics\n")
	fmt.Println(strings.Repeat("=", 35))
	fmt.Printf("⚡ Parallel execution: %v\n", parallel)
	fmt.Printf("⏱️  Timeout: %ds\n", timeout)
	fmt.Println()

	report, err := services.System.RunDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("diagnostic scan failed: %w", err)
	}

	// Display overall status
	statusIcon := getOverallStatusIcon(report.Overall)
	fmt.Printf("%s Overall Status: %s\n", statusIcon, strings.ToUpper(report.Overall))
	fmt.Println()

	// Display all checks
	fmt.Printf("📋 Diagnostic Results (%d checks):\n", len(report.Checks))
	fmt.Println(strings.Repeat("-", 40))

	passCount := 0
	warnCount := 0
	failCount := 0

	for _, check := range report.Checks {
		statusIcon := getCheckStatusIcon(check.Status)
		fmt.Printf("%s %s: %s\n", statusIcon, check.Name, check.Message)

		if check.Fix != "" && check.Status != "pass" {
			fmt.Printf("   💡 %s\n", check.Fix)
		}

		switch check.Status {
		case "pass":
			passCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
		}
	}

	// Summary
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   ✅ Passed: %d\n", passCount)
	if warnCount > 0 {
		fmt.Printf("   ⚠️  Warnings: %d\n", warnCount)
	}
	if failCount > 0 {
		fmt.Printf("   ❌ Failed: %d\n", failCount)
	}

	fmt.Printf("\n%s\n", report.Summary)

	// Recommendations
	if failCount > 0 {
		fmt.Printf("\n🔧 Recommended Actions:\n")
		fmt.Printf("   • Run 'gitpersona diagnose fix --auto' to fix auto-fixable issues\n")
		fmt.Printf("   • Check specific components with targeted diagnostics\n")
	}

	if report.Overall == "critical" {
		fmt.Printf("\n🚨 CRITICAL: System requires immediate attention!\n")
	}

	return nil
}

func diagnoseHealthHandler(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	if jsonOutput {
		// TODO: Implement JSON health output
		fmt.Printf("JSON health output not implemented yet\n")
		return nil
	}

	fmt.Printf("💚 GitPersona Health Check\n")
	fmt.Println(strings.Repeat("=", 25))

	// Quick health check
	if err := services.System.PerformHealthCheck(ctx); err != nil {
		fmt.Printf("❌ UNHEALTHY: %v\n", err)
		fmt.Printf("\n💡 Run 'gitpersona diagnose full' for detailed analysis\n")
		return nil
	}

	fmt.Printf("✅ HEALTHY: All systems operational\n")

	// Quick stats
	if accounts, err := services.Account.ListAccounts(ctx); err == nil {
		fmt.Printf("👥 %d accounts configured\n", len(accounts))

		validCount := 0
		for _, account := range accounts {
			if validation, err := services.Account.ValidateAccount(ctx, account.Alias); err == nil && validation.Valid {
				validCount++
			}
		}
		fmt.Printf("✅ %d/%d accounts valid\n", validCount, len(accounts))
	}

	return nil
}

func diagnoseSSHHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	account, _ := cmd.Flags().GetString("account")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🔐 SSH Diagnostics\n")
	fmt.Println(strings.Repeat("=", 17))

	if account != "" {
		fmt.Printf("🎯 Targeting account: %s\n\n", account)
	}

	// Run SSH diagnostics
	issues, err := services.SSH.DiagnoseIssues(ctx)
	if err != nil {
		return fmt.Errorf("SSH diagnostic failed: %w", err)
	}

	if len(issues) == 0 {
		fmt.Printf("🎉 SSH system is healthy!\n")
		return nil
	}

	fmt.Printf("⚠️  Found %d SSH issues:\n\n", len(issues))

	for i, issue := range issues {
		severity := getSeverityIcon(issue.Severity)
		fmt.Printf("%d. %s %s\n", i+1, severity, issue.Description)

		if verbose {
			fmt.Printf("   💡 %s\n", issue.Fix)
			if issue.AutoFixable {
				fmt.Printf("   🤖 Auto-fixable\n")
			}
			fmt.Printf("   📊 Severity: %s\n", issue.Severity)
		}
		fmt.Println()
	}

	// Count auto-fixable issues
	autoFixCount := 0
	for _, issue := range issues {
		if issue.AutoFixable {
			autoFixCount++
		}
	}

	if autoFixCount > 0 {
		fmt.Printf("💡 %d issues can be auto-fixed with: gitpersona ssh fix --auto\n", autoFixCount)
	}

	return nil
}

func diagnoseGitHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("📁 Git Configuration Diagnostics\n")
	fmt.Println(strings.Repeat("=", 32))

	// Validate Git configuration
	validation, err := services.Git.ValidateConfig(ctx)
	if err != nil {
		return fmt.Errorf("Git diagnostic failed: %w", err)
	}

	if validation.Valid {
		fmt.Printf("✅ Git configuration is valid\n")

		// Show current config
		if config, err := services.Git.GetCurrentConfig(ctx); err == nil {
			fmt.Printf("\n📊 Current Configuration (%s):\n", config.Scope)
			fmt.Printf("   • Name: %s\n", config.Name)
			fmt.Printf("   • Email: %s\n", config.Email)
		}

		return nil
	}

	fmt.Printf("❌ Git configuration has %d issues:\n\n", len(validation.Issues))

	for i, issue := range validation.Issues {
		fmt.Printf("%d. %s\n", i+1, issue.Description)
		if verbose {
			fmt.Printf("   💡 %s\n", issue.Fix)
		}
		fmt.Println()
	}

	fmt.Printf("🔧 Most Git issues can be resolved by switching to a configured account\n")
	fmt.Printf("💡 Try: gitpersona account switch [account-name]\n")

	return nil
}

func diagnoseGitHubHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	account, _ := cmd.Flags().GetString("account")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🐙 GitHub Connectivity Diagnostics\n")
	fmt.Println(strings.Repeat("=", 33))

	var targetAccounts []*internal.Account

	if account != "" {
		acc, err := services.Account.GetAccount(ctx, account)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", account, err)
		}
		targetAccounts = []*internal.Account{acc}
	} else {
		accounts, err := services.Account.ListAccounts(ctx)
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}
		targetAccounts = accounts
	}

	if len(targetAccounts) == 0 {
		fmt.Printf("📭 No accounts to test\n")
		return nil
	}

	fmt.Printf("🧪 Testing %d accounts...\n\n", len(targetAccounts))

	for _, acc := range targetAccounts {
		fmt.Printf("Testing account: %s\n", acc.Alias)

		// Test API access
		if err := services.GitHub.TestAPIAccess(ctx, acc); err != nil {
			fmt.Printf("❌ API Access: Failed (%v)\n", err)
		} else {
			fmt.Printf("✅ API Access: OK\n")
		}

		// Test SSH access via SSH manager
		if sshResult, err := services.SSH.TestConnectivity(ctx, acc); err == nil {
			if sshResult.Success {
				fmt.Printf("✅ SSH Access: OK (%dms)\n", sshResult.Latency)
			} else {
				fmt.Printf("❌ SSH Access: Failed (%s)\n", sshResult.Message)
			}
		} else {
			fmt.Printf("❌ SSH Access: Error (%v)\n", err)
		}

		// Token validation
		if validation, err := services.GitHub.ValidateToken(ctx, acc); err == nil {
			if validation.Valid {
				fmt.Printf("✅ Token: Valid (@%s)\n", validation.Username)
				if verbose && len(validation.Scopes) > 0 {
					fmt.Printf("   🔐 Scopes: %s\n", strings.Join(validation.Scopes, ", "))
				}
			} else {
				fmt.Printf("❌ Token: %s\n", validation.Message)
			}
		} else {
			fmt.Printf("⚠️  Token: Unable to validate\n")
		}

		fmt.Println()
	}

	return nil
}

func diagnoseFixHandler(cmd *cobra.Command, args []string) error {
	auto, _ := cmd.Flags().GetBool("auto")
	force, _ := cmd.Flags().GetBool("force")
	types, _ := cmd.Flags().GetStringSlice("types")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("🔧 System Auto-Fix\n")
	fmt.Println(strings.Repeat("=", 16))

	// Get system diagnostics to find issues
	report, err := services.System.RunDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("failed to run diagnostics: %w", err)
	}

	// Convert diagnostic checks to system issues
	issues := convertChecksToIssues(report.Checks, types)

	if len(issues) == 0 {
		fmt.Printf("🎉 No fixable issues found!\n")
		return nil
	}

	autoFixableIssues := make([]*internal.SystemIssue, 0)
	for _, issue := range issues {
		if issue.AutoFixable {
			autoFixableIssues = append(autoFixableIssues, issue)
		}
	}

	fmt.Printf("Found %d issues, %d auto-fixable\n\n", len(issues), len(autoFixableIssues))

	if len(autoFixableIssues) == 0 {
		fmt.Printf("❌ No auto-fixable issues found\n")
		fmt.Printf("💡 Manual intervention required for:\n")
		for i, issue := range issues {
			fmt.Printf("   %d. %s\n", i+1, issue.Description)
		}
		return nil
	}

	if !auto && !force {
		fmt.Printf("Auto-fixable issues:\n")
		for i, issue := range autoFixableIssues {
			fmt.Printf("   %d. %s\n", i+1, issue.Description)
		}

		fmt.Printf("\nProceed with auto-fix? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Handle scan error silently
		}
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Auto-fix cancelled")
			return nil
		}
	}

	fmt.Printf("🔧 Applying fixes...\n\n")

	if err := services.System.AutoFix(ctx, autoFixableIssues); err != nil {
		return fmt.Errorf("auto-fix failed: %w", err)
	}

	fmt.Printf("✅ Auto-fix completed\n")
	fmt.Printf("💡 Run diagnostics again to verify fixes\n")

	return nil
}

// Helper functions

func getOverallStatusIcon(status string) string {
	switch status {
	case "healthy":
		return "💚"
	case "issues":
		return "⚠️"
	case "critical":
		return "🚨"
	default:
		return "❓"
	}
}

func getCheckStatusIcon(status string) string {
	switch status {
	case "pass":
		return "✅"
	case "warn":
		return "⚠️"
	case "fail":
		return "❌"
	default:
		return "❓"
	}
}

func convertChecksToIssues(checks []*internal.DiagnosticCheck, typeFilter []string) []*internal.SystemIssue {
	issues := make([]*internal.SystemIssue, 0)

	for _, check := range checks {
		if check.Status == "pass" {
			continue
		}

		// Apply type filter if specified
		if len(typeFilter) > 0 {
			found := false
			for _, t := range typeFilter {
				if strings.Contains(strings.ToLower(check.Name), strings.ToLower(t)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		severity := "medium"
		if check.Status == "fail" {
			severity = "high"
		} else if check.Status == "warn" {
			severity = "low"
		}

		issue := &internal.SystemIssue{
			Type:        strings.ToLower(strings.ReplaceAll(check.Name, " ", "_")),
			Severity:    severity,
			Description: check.Message,
			AutoFixable: check.Fix != "" && (strings.Contains(check.Fix, "chmod") || strings.Contains(check.Fix, "mkdir")),
		}

		issues = append(issues, issue)
	}

	return issues
}
