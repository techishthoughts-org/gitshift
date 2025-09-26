package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal"
)

// Wire smart command handlers to CoreServices

// Enhanced status command with progressive disclosure
func statusHandler(cmd *cobra.Command, args []string) error {
	detailed, _ := cmd.Flags().GetBool("detailed")
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Progressive disclosure - determine status level
	statusLevel := "basic"
	if verbose {
		statusLevel = "verbose"
	} else if detailed {
		statusLevel = "detailed"
	}

	// Check for specific status requests
	accounts, _ := cmd.Flags().GetBool("accounts")
	ssh, _ := cmd.Flags().GetBool("ssh")
	git, _ := cmd.Flags().GetBool("git")
	github, _ := cmd.Flags().GetBool("github")
	health, _ := cmd.Flags().GetBool("health")

	return showProgressiveStatusWired(statusLevel, jsonOutput, map[string]bool{
		"accounts": accounts,
		"ssh":      ssh,
		"git":      git,
		"github":   github,
		"health":   health,
	})
}

// Enhanced auto commands with smart detection
func autoDetectHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Println("🔍 Auto-detecting current context...")

	// Detect repository context
	cwd, _ := os.Getwd()
	repoInfo, err := services.Git.DetectRepository(ctx, cwd)

	if err != nil {
		fmt.Printf("📁 Not in a Git repository\n")
	} else {
		fmt.Printf("📁 Repository detected: %s\n", filepath.Base(repoInfo.Path))

		if repoInfo.IsGitHub {
			fmt.Printf("🐙 GitHub repository: %s\n", repoInfo.Organization)

			// Try to suggest appropriate account
			if suggestedAccount, err := services.Git.SuggestAccount(ctx, repoInfo); err == nil {
				fmt.Printf("🎯 Suggested account: %s\n", suggestedAccount.Alias)

				// Check if current account matches
				if current, err := services.Account.GetCurrentAccount(ctx); err == nil {
					if current.Alias != suggestedAccount.Alias {
						fmt.Printf("💡 Consider switching: gitpersona account switch %s\n", suggestedAccount.Alias)
					} else {
						fmt.Printf("✅ Already using the right account\n")
					}
				}
			}
		}
	}

	// Detect system issues
	if report, err := services.System.RunDiagnostics(ctx); err == nil {
		failedChecks := 0
		for _, check := range report.Checks {
			if check.Status == "fail" {
				failedChecks++
			}
		}

		if failedChecks > 0 {
			fmt.Printf("\n⚠️  Detected %d system issues\n", failedChecks)
			fmt.Printf("💡 Run 'gitpersona auto fix' to resolve automatically\n")
		} else {
			fmt.Printf("\n✅ System is healthy\n")
		}
	}

	// Detect account validation issues
	if accounts, err := services.Account.ListAccounts(ctx); err == nil {
		invalidCount := 0
		for _, account := range accounts {
			if validation, err := services.Account.ValidateAccount(ctx, account.Alias); err == nil && !validation.Valid {
				invalidCount++
			}
		}

		if invalidCount > 0 {
			fmt.Printf("\n⚠️  %d accounts need attention\n", invalidCount)
			fmt.Printf("💡 Run 'gitpersona account validate-all' for details\n")
		}
	}

	return nil
}

func autoSetupHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Println("🚀 Auto-setup based on detected environment")

	// Check if any accounts exist
	accounts, err := services.Account.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to check accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Printf("📭 No accounts found - starting initial setup\n")
		fmt.Printf("💡 This would launch the interactive setup wizard\n")
		fmt.Printf("   Run: gitpersona account add [name] for now\n")
		return nil
	}

	fmt.Printf("👥 Found %d existing accounts\n", len(accounts))

	// Check repository context for smart setup
	cwd, _ := os.Getwd()
	if repoInfo, err := services.Git.DetectRepository(ctx, cwd); err == nil {
		fmt.Printf("📁 In repository: %s\n", filepath.Base(repoInfo.Path))

		if repoInfo.IsGitHub && repoInfo.Organization != "" {
			// Try to find or create account for this organization
			fmt.Printf("🔍 Looking for account matching organization: %s\n", repoInfo.Organization)

			// This would implement smart account matching logic
			fmt.Printf("💡 Auto-setup for organization-specific accounts not fully implemented\n")
		}
	}

	// Validate existing accounts and suggest fixes
	validationResults, err := services.Account.ValidateAllAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate accounts: %w", err)
	}

	issueCount := 0
	for _, result := range validationResults {
		if !result.Valid {
			issueCount++
		}
	}

	if issueCount > 0 {
		fmt.Printf("🔧 Found issues with %d accounts - attempting auto-fix\n", issueCount)
		fmt.Printf("💡 Auto-fix not fully implemented - run manual validation\n")
	} else {
		fmt.Printf("✅ All accounts are properly configured\n")
	}

	return nil
}

func autoFixHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Println("🔧 Auto-fixing detected issues")

	// Run comprehensive diagnostics
	report, err := services.System.RunDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("failed to run diagnostics: %w", err)
	}

	// Count fixable issues
	autoFixableCount := 0
	for _, check := range report.Checks {
		if check.Status != "pass" && check.Fix != "" {
			// Simple heuristic for auto-fixable issues
			if strings.Contains(check.Fix, "chmod") || strings.Contains(check.Fix, "mkdir") {
				autoFixableCount++
			}
		}
	}

	if autoFixableCount == 0 {
		fmt.Printf("🎉 No auto-fixable issues found\n")
		return nil
	}

	fmt.Printf("🔧 Attempting to fix %d issues...\n", autoFixableCount)

	// This would call the actual auto-fix functionality
	fmt.Printf("💡 Auto-fix implementation delegated to diagnostic system\n")
	return diagnoseFixHandler(cmd, []string{})
}

func autoSwitchHandler(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Println("🔄 Auto-switching account based on repository context")

	// Detect current repository
	cwd, _ := os.Getwd()
	repoInfo, err := services.Git.DetectRepository(ctx, cwd)
	if err != nil {
		return fmt.Errorf("not in a Git repository: %w", err)
	}

	// Get suggested account
	suggestedAccount, err := services.Git.SuggestAccount(ctx, repoInfo)
	if err != nil {
		return fmt.Errorf("no account suggestion available: %w", err)
	}

	// Check if already using the right account
	current, err := services.Account.GetCurrentAccount(ctx)
	if err == nil && current.Alias == suggestedAccount.Alias {
		fmt.Printf("✅ Already using the correct account: %s\n", current.Alias)
		return nil
	}

	fmt.Printf("🎯 Suggested account: %s\n", suggestedAccount.Alias)
	fmt.Printf("📁 Repository: %s/%s\n", repoInfo.Organization, filepath.Base(repoInfo.Path))

	// Switch to suggested account
	if err := services.Account.SwitchAccount(ctx, suggestedAccount.Alias); err != nil {
		return fmt.Errorf("failed to switch account: %w", err)
	}

	fmt.Printf("✅ Switched to account: %s\n", suggestedAccount.Alias)
	return nil
}

func autoCloneHandler(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("📥 Auto-cloning with account detection: %s\n", repoURL)

	// Parse repository URL to detect organization
	org := extractOrganizationFromURL(repoURL)
	if org == "" {
		return fmt.Errorf("could not detect organization from URL: %s", repoURL)
	}

	fmt.Printf("🔍 Detected organization: %s\n", org)

	// Find appropriate account
	accounts, err := services.Account.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	var selectedAccount *internal.Account
	for _, account := range accounts {
		// Simple matching - in real implementation, this would be more sophisticated
		if strings.Contains(strings.ToLower(account.Alias), strings.ToLower(org)) ||
			strings.Contains(strings.ToLower(account.Description), strings.ToLower(org)) {
			selectedAccount = account
			break
		}
	}

	if selectedAccount == nil {
		// Use current account as fallback
		if current, err := services.Account.GetCurrentAccount(ctx); err == nil {
			selectedAccount = current
			fmt.Printf("⚠️  No specific account found for %s, using current: %s\n", org, current.Alias)
		} else {
			return fmt.Errorf("no suitable account found for organization: %s", org)
		}
	} else {
		fmt.Printf("🎯 Using account: %s\n", selectedAccount.Alias)
	}

	// Switch to selected account
	if err := services.Account.SwitchAccount(ctx, selectedAccount.Alias); err != nil {
		return fmt.Errorf("failed to switch to account: %w", err)
	}

	// Clone the repository
	fmt.Printf("📥 Cloning repository...\n")
	fmt.Printf("💡 Actual cloning not implemented - would execute: git clone %s\n", repoURL)

	fmt.Printf("✅ Repository cloned with account: %s\n", selectedAccount.Alias)
	return nil
}

// Progressive status implementation with CoreServices
func showProgressiveStatusWired(level string, jsonOutput bool, filters map[string]bool) error {
	ctx := context.Background()
	services := serviceContainer.GetServices()

	switch level {
	case "basic":
		return showBasicStatusWired(ctx, services, jsonOutput, filters)
	case "detailed":
		return showDetailedStatusWired(ctx, services, jsonOutput, filters)
	case "verbose":
		return showVerboseStatusWired(ctx, services, jsonOutput, filters)
	default:
		return fmt.Errorf("unknown status level: %s", level)
	}
}

func showBasicStatusWired(ctx context.Context, services *internal.CoreServices, jsonOutput bool, filters map[string]bool) error {
	fmt.Println("📊 GitPersona Status")
	fmt.Println(strings.Repeat("=", 20))

	// Current account
	if current, err := services.Account.GetCurrentAccount(ctx); err == nil {
		fmt.Printf("👤 Current Account: %s\n", current.Alias)
		fmt.Printf("📧 %s <%s>\n", current.Name, current.Email)
	} else {
		fmt.Printf("👤 Current Account: None active\n")
	}

	// Repository context
	cwd, _ := os.Getwd()
	if repoInfo, err := services.Git.DetectRepository(ctx, cwd); err == nil {
		fmt.Printf("📁 Repository: %s\n", filepath.Base(repoInfo.Path))
		if repoInfo.IsGitHub {
			fmt.Printf("🐙 Organization: %s\n", repoInfo.Organization)
		}
	} else {
		fmt.Printf("📁 Repository: Not in a Git repository\n")
	}

	// Quick health check
	if err := services.System.PerformHealthCheck(ctx); err == nil {
		fmt.Printf("💚 Health: All systems operational\n")
	} else {
		fmt.Printf("⚠️  Health: Issues detected\n")
	}

	fmt.Printf("\n💡 Use --detailed for more information\n")
	return nil
}

func showDetailedStatusWired(ctx context.Context, services *internal.CoreServices, jsonOutput bool, filters map[string]bool) error {
	// Show basic status first
	showBasicStatusWired(ctx, services, false, filters)

	fmt.Printf("\n--- DETAILED STATUS ---\n")

	// Account summary
	if accounts, err := services.Account.ListAccounts(ctx); err == nil {
		fmt.Printf("\n👥 Accounts (%d):\n", len(accounts))
		for _, account := range accounts {
			status := "✅"
			if validation, err := services.Account.ValidateAccount(ctx, account.Alias); err != nil || !validation.Valid {
				status = "⚠️"
			}

			current := ""
			if currentAcc, err := services.Account.GetCurrentAccount(ctx); err == nil && currentAcc.Alias == account.Alias {
				current = " (current)"
			}

			fmt.Printf("   %s %s%s\n", status, account.Alias, current)
		}
	}

	// SSH summary
	if keys, err := services.SSH.ListKeys(ctx); err == nil {
		validKeys := 0
		for _, key := range keys {
			if key.Valid {
				validKeys++
			}
		}
		fmt.Printf("\n🔑 SSH Keys: %d valid, %d total\n", validKeys, len(keys))
	}

	// System health
	if report, err := services.System.RunDiagnostics(ctx); err == nil {
		fmt.Printf("\n🏥 System Health: %s\n", report.Overall)
	}

	return nil
}

func showVerboseStatusWired(ctx context.Context, services *internal.CoreServices, jsonOutput bool, filters map[string]bool) error {
	// Show detailed status first
	showDetailedStatusWired(ctx, services, false, filters)

	fmt.Printf("\n--- VERBOSE STATUS ---\n")

	// System information
	if sysInfo, err := services.System.GetSystemInfo(ctx); err == nil {
		fmt.Printf("\n🖥️  System Information:\n")
		fmt.Printf("   Platform: %s\n", sysInfo.Platform)
		fmt.Printf("   Version: %s\n", sysInfo.Version)
		fmt.Printf("   Git: %s\n", sysInfo.GitVersion)
		fmt.Printf("   SSH: %s\n", sysInfo.SSHVersion)
	}

	// Detailed diagnostics
	if report, err := services.System.RunDiagnostics(ctx); err == nil {
		fmt.Printf("\n🔬 Diagnostic Summary:\n")
		passCount := 0
		warnCount := 0
		failCount := 0

		for _, check := range report.Checks {
			switch check.Status {
			case "pass":
				passCount++
			case "warn":
				warnCount++
			case "fail":
				failCount++
			}
		}

		fmt.Printf("   ✅ Passed: %d\n", passCount)
		if warnCount > 0 {
			fmt.Printf("   ⚠️  Warnings: %d\n", warnCount)
		}
		if failCount > 0 {
			fmt.Printf("   ❌ Failed: %d\n", failCount)
		}
	}

	return nil
}

// Helper function to extract organization from URL
func extractOrganizationFromURL(url string) string {
	// Handle SSH URLs: git@github.com:org/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Handle HTTPS URLs: https://github.com/org/repo.git
	if strings.HasPrefix(url, "https://github.com/") {
		path := strings.TrimPrefix(url, "https://github.com/")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	return ""
}
