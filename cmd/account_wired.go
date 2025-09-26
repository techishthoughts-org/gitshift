package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal"
)

// Wire account command handlers to CoreServices

func accountListHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	status, _ := cmd.Flags().GetBool("status")
	detailed, _ := cmd.Flags().GetBool("detailed")
	filter, _ := cmd.Flags().GetString("filter")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	accounts, err := services.Account.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Printf("📭 No accounts configured\n")
		fmt.Printf("💡 Add your first account with: gitpersona account add [name]\n")
		return nil
	}

	// Get current account
	currentAccount, _ := services.Account.GetCurrentAccount(ctx)
	currentAlias := ""
	if currentAccount != nil {
		currentAlias = currentAccount.Alias
	}

	if jsonOutput {
		return outputAccountsJSON(accounts, currentAlias)
	}

	// Filter accounts if requested
	filteredAccounts := filterAccounts(accounts, filter)

	fmt.Printf("👥 GitPersona Accounts (%d total)\n", len(accounts))
	fmt.Println(strings.Repeat("=", 40))

	for _, account := range filteredAccounts {
		displayAccount(account, currentAlias, detailed || verbose, status)
	}

	if len(filteredAccounts) < len(accounts) {
		fmt.Printf("\n🔍 Showing %d/%d accounts (filtered by: %s)\n",
			len(filteredAccounts), len(accounts), filter)
	}

	return nil
}

func accountShowHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	var account *internal.Account
	var err error

	if len(args) > 0 {
		account, err = services.Account.GetAccount(ctx, args[0])
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", args[0], err)
		}
	} else {
		account, err = services.Account.GetCurrentAccount(ctx)
		if err != nil {
			return fmt.Errorf("no current account set: %w", err)
		}
	}

	fmt.Printf("👤 Account Details: %s\n", account.Alias)
	fmt.Println(strings.Repeat("=", 30))

	displayAccountDetails(account, verbose)

	// Show validation status
	if validation, err := services.Account.ValidateAccount(ctx, account.Alias); err == nil {
		fmt.Println("\n🔍 Validation Status:")
		if validation.Valid {
			fmt.Printf("✅ Account is valid and ready to use\n")
		} else {
			fmt.Printf("❌ Account has %d issues:\n", len(validation.Issues))
			for i, issue := range validation.Issues {
				fmt.Printf("  %d. %s\n", i+1, issue)
			}
		}
	}

	return nil
}

func accountCurrentHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	account, err := services.Account.GetCurrentAccount(ctx)
	if err != nil {
		fmt.Printf("❌ No account currently active\n")
		fmt.Printf("💡 Switch to an account with: gitpersona account switch [name]\n")
		return nil
	}

	fmt.Printf("🎯 Current Account: %s\n", account.Alias)

	if verbose {
		fmt.Println()
		displayAccountDetails(account, true)
	} else {
		fmt.Printf("👤 %s <%s>\n", account.Name, account.Email)
		if account.GitHubUsername != "" {
			fmt.Printf("🐙 @%s\n", account.GitHubUsername)
		}
	}

	return nil
}

func accountCreateHandler(cmd *cobra.Command, args []string) error {
	alias := args[0]
	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	githubUsername, _ := cmd.Flags().GetString("github-username")
	sshKey, _ := cmd.Flags().GetString("ssh-key")
	description, _ := cmd.Flags().GetString("description")
	generateKey, _ := cmd.Flags().GetBool("generate-key")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Validate required fields
	if name == "" {
		return fmt.Errorf("name is required (use --name)")
	}
	if email == "" {
		return fmt.Errorf("email is required (use --email)")
	}

	fmt.Printf("🔨 Creating account '%s'\n", alias)
	fmt.Println(strings.Repeat("-", 30))

	req := internal.CreateAccountRequest{
		Alias:          alias,
		Name:           name,
		Email:          email,
		GitHubUsername: githubUsername,
		SSHKeyPath:     sshKey,
		Description:    description,
	}

	// Generate SSH key if requested
	if generateKey && sshKey == "" {
		fmt.Printf("🔑 Generating SSH key...\n")
		keyPath := generateSSHKeyPath(alias, "ed25519")
		req.SSHKeyPath = keyPath
	}

	account, err := services.Account.CreateAccount(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	fmt.Printf("✅ Account created successfully!\n\n")
	displayAccountDetails(account, true)

	// Suggest next steps
	fmt.Println("\n💡 Next steps:")
	fmt.Printf("   1. Switch to this account: gitpersona account switch %s\n", alias)
	if account.SSHKeyPath != "" {
		fmt.Printf("   2. Upload SSH key to GitHub: cat %s.pub\n", account.SSHKeyPath)
		fmt.Printf("   3. Test connectivity: gitpersona ssh test %s\n", alias)
	}

	return nil
}

func accountSwitchHandler(cmd *cobra.Command, args []string) error {
	alias := args[0]

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Validate account exists
	account, err := services.Account.GetAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	fmt.Printf("🔄 Switching to account '%s'\n", alias)

	if err := services.Account.SwitchAccount(ctx, alias); err != nil {
		return fmt.Errorf("failed to switch account: %w", err)
	}

	fmt.Printf("✅ Switched to account '%s'\n", account.Alias)
	fmt.Printf("👤 %s <%s>\n", account.Name, account.Email)

	// Show quick validation
	if validation, err := services.Account.ValidateAccount(ctx, alias); err == nil {
		if validation.Valid {
			fmt.Printf("💚 Account is ready to use\n")
		} else {
			fmt.Printf("⚠️  Account has %d issues - run 'gitpersona account validate %s' for details\n",
				len(validation.Issues), alias)
		}
	}

	return nil
}

func accountUpdateHandler(cmd *cobra.Command, args []string) error {
	alias := args[0]
	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	githubUsername, _ := cmd.Flags().GetString("github-username")
	sshKey, _ := cmd.Flags().GetString("ssh-key")
	description, _ := cmd.Flags().GetString("description")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Validate account exists
	_, err := services.Account.GetAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	updates := internal.AccountUpdates{}
	updateCount := 0

	if name != "" {
		updates.Name = &name
		updateCount++
	}
	if email != "" {
		updates.Email = &email
		updateCount++
	}
	if githubUsername != "" {
		updates.GitHubUsername = &githubUsername
		updateCount++
	}
	if sshKey != "" {
		updates.SSHKeyPath = &sshKey
		updateCount++
	}
	if description != "" {
		updates.Description = &description
		updateCount++
	}

	if updateCount == 0 {
		return fmt.Errorf("no updates specified - use --name, --email, --github-username, --ssh-key, or --description")
	}

	fmt.Printf("✏️  Updating account '%s' (%d changes)\n", alias, updateCount)

	if err := services.Account.UpdateAccount(ctx, alias, updates); err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	fmt.Printf("✅ Account updated successfully\n")

	// Show updated account
	if account, err := services.Account.GetAccount(ctx, alias); err == nil {
		fmt.Println()
		displayAccountDetails(account, false)
	}

	return nil
}

func accountDeleteHandler(cmd *cobra.Command, args []string) error {
	alias := args[0]
	force, _ := cmd.Flags().GetBool("force")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	// Validate account exists
	account, err := services.Account.GetAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	fmt.Printf("🗑️  Deleting account '%s'\n", alias)
	fmt.Printf("👤 %s <%s>\n", account.Name, account.Email)

	if !force {
		fmt.Printf("\n⚠️  This action cannot be undone. Continue? [y/N]: ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// Handle scan error silently
		}
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Operation cancelled")
			return nil
		}
	}

	if err := services.Account.DeleteAccount(ctx, alias); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	fmt.Printf("✅ Account deleted successfully\n")

	if account.SSHKeyPath != "" {
		fmt.Printf("💡 SSH keys remain at: %s\n", account.SSHKeyPath)
		fmt.Printf("   Remove manually if no longer needed\n")
	}

	return nil
}

func accountValidateHandler(cmd *cobra.Command, args []string) error {
	var alias string
	if len(args) > 0 {
		alias = args[0]
	}

	fix, _ := cmd.Flags().GetBool("fix")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	var account *internal.Account
	var err error

	if alias != "" {
		account, err = services.Account.GetAccount(ctx, alias)
		if err != nil {
			return fmt.Errorf("account '%s' not found: %w", alias, err)
		}
	} else {
		account, err = services.Account.GetCurrentAccount(ctx)
		if err != nil {
			return fmt.Errorf("no current account set: %w", err)
		}
		alias = account.Alias
	}

	fmt.Printf("✅ Validating account '%s'\n", alias)
	fmt.Println(strings.Repeat("=", 30))

	validation, err := services.Account.ValidateAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if validation.Valid {
		fmt.Printf("🎉 Account is valid and ready to use!\n")
		return nil
	}

	fmt.Printf("❌ Account has %d issues:\n\n", len(validation.Issues))

	for i, issue := range validation.Issues {
		fmt.Printf("%d. %s\n", i+1, issue)
	}

	if fix {
		fmt.Printf("\n🔧 Auto-fix is not implemented yet\n")
		fmt.Printf("💡 Please resolve issues manually\n")
	}

	fmt.Printf("\n💡 Suggestions:\n")
	fmt.Printf("   • Test SSH: gitpersona ssh test %s\n", alias)
	fmt.Printf("   • Fix SSH issues: gitpersona ssh diagnose\n")
	fmt.Printf("   • Validate GitHub access: gitpersona github test %s\n", alias)

	return nil
}

func accountValidateAllHandler(cmd *cobra.Command, args []string) error {
	fix, _ := cmd.Flags().GetBool("fix")

	ctx := context.Background()
	services := serviceContainer.GetServices()

	fmt.Printf("✅ Validating all accounts\n")
	fmt.Println(strings.Repeat("=", 25))

	validations, err := services.Account.ValidateAllAccounts(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(validations) == 0 {
		fmt.Printf("📭 No accounts to validate\n")
		return nil
	}

	validCount := 0
	totalIssues := 0

	for _, validation := range validations {
		status := "❌"
		if validation.Valid {
			status = "✅"
			validCount++
		} else {
			totalIssues += len(validation.Issues)
		}

		fmt.Printf("%s %s", status, validation.Account)

		if !validation.Valid {
			fmt.Printf(" (%d issues)", len(validation.Issues))
		}
		fmt.Println()
	}

	fmt.Printf("\n📊 Summary: %d/%d accounts valid", validCount, len(validations))
	if totalIssues > 0 {
		fmt.Printf(", %d total issues", totalIssues)
	}
	fmt.Println()

	if totalIssues > 0 && fix {
		fmt.Printf("\n🔧 Auto-fix is not implemented yet\n")
		fmt.Printf("💡 Run individual account validation for details\n")
	}

	return nil
}

// Helper functions

func displayAccount(account *internal.Account, currentAlias string, detailed, showStatus bool) {
	indicator := "  "
	if account.Alias == currentAlias {
		indicator = "▶ "
	}

	fmt.Printf("%s%s", indicator, account.Alias)

	if account.Alias == currentAlias {
		fmt.Printf(" (current)")
	}

	if detailed {
		fmt.Printf("\n     👤 %s <%s>\n", account.Name, account.Email)
		if account.GitHubUsername != "" {
			fmt.Printf("     🐙 @%s\n", account.GitHubUsername)
		}
		if account.SSHKeyPath != "" {
			fmt.Printf("     🔑 %s\n", account.SSHKeyPath)
		}
		if account.LastUsed != nil {
			fmt.Printf("     🕒 Last used: %s\n", formatTimeAgo(*account.LastUsed))
		}
	}

	fmt.Println()
}

func displayAccountDetails(account *internal.Account, verbose bool) {
	fmt.Printf("Alias: %s\n", account.Alias)
	fmt.Printf("Name: %s\n", account.Name)
	fmt.Printf("Email: %s\n", account.Email)

	if account.GitHubUsername != "" {
		fmt.Printf("GitHub: @%s\n", account.GitHubUsername)
	}

	if account.SSHKeyPath != "" {
		fmt.Printf("SSH Key: %s\n", account.SSHKeyPath)
	}

	if account.Description != "" {
		fmt.Printf("Description: %s\n", account.Description)
	}

	if verbose {
		fmt.Printf("Created: %s\n", formatTimeAgo(account.CreatedAt))
		if account.LastUsed != nil {
			fmt.Printf("Last Used: %s\n", formatTimeAgo(*account.LastUsed))
		}
		fmt.Printf("Active: %v\n", account.IsActive)
	}
}

func filterAccounts(accounts []*internal.Account, filter string) []*internal.Account {
	if filter == "" {
		return accounts
	}

	filtered := make([]*internal.Account, 0)
	for _, account := range accounts {
		switch filter {
		case "active":
			if account.IsActive {
				filtered = append(filtered, account)
			}
		case "inactive":
			if !account.IsActive {
				filtered = append(filtered, account)
			}
		default:
			// Default: include all
			filtered = append(filtered, account)
		}
	}

	return filtered
}

func formatTimeAgo(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

func outputAccountsJSON(accounts []*internal.Account, currentAlias string) error {
	// TODO: Implement JSON output
	fmt.Printf("JSON output not implemented yet\n")
	return nil
}
