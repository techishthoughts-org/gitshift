package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/discovery"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

var discoverAccountsCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover existing GitHub accounts from SSH keys and Git configuration",
	Long: `Automatically discover GitHub accounts by scanning:
- SSH keys in ~/.ssh/ directory
- Git configuration files
- GitHub CLI authentication
- SSH configuration

This command will analyze your existing setup and suggest accounts to add to GitPersona.`,
	Example: `  # Discover all accounts
  gitpersona discover

  # Discover accounts and show detailed analysis
  gitpersona discover --verbose

  # Discover accounts and automatically add them
  gitpersona discover --auto-add`,
	RunE: runDiscoverAccounts,
}

var (
	discoverVerbose bool
	discoverAutoAdd bool
	discoverSSHOnly bool
)

func init() {
	rootCmd.AddCommand(discoverAccountsCmd)

	discoverAccountsCmd.Flags().BoolVarP(&discoverVerbose, "verbose", "v", false, "Show detailed discovery analysis")
	discoverAccountsCmd.Flags().BoolVar(&discoverAutoAdd, "auto-add", false, "Automatically add discovered accounts")
	discoverAccountsCmd.Flags().BoolVar(&discoverSSHOnly, "ssh-only", false, "Only scan SSH keys")
}

func runDiscoverAccounts(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	logger := observability.NewDefaultLogger()

	// Print header
	fmt.Println("🔍 GitPersona Account Discovery")
	fmt.Println("==================================")
	fmt.Println()

	if discoverSSHOnly {
		return runSSHKeyDiscovery(ctx, logger)
	}

	return runFullDiscovery(ctx, logger)
}

func runSSHKeyDiscovery(ctx context.Context, logger observability.Logger) error {
	fmt.Println("🔑 Scanning SSH keys in ~/.ssh/...")
	fmt.Println()

	scanner := discovery.NewSSHKeyScanner(logger)

	// Discover SSH keys
	keys, err := scanner.DiscoverSSHKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover SSH keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("❌ No SSH keys found in ~/.ssh/")
		return nil
	}

	fmt.Printf("✅ Found %d SSH key(s)\n\n", len(keys))

	// Display discovered keys
	for i, key := range keys {
		fmt.Printf("🔑 SSH Key #%d\n", i+1)
		fmt.Printf("   Path: %s\n", key.PrivateKeyPath)
		fmt.Printf("   Type: %s\n", key.KeyType)

		if key.Comment != "" {
			fmt.Printf("   Comment: %s\n", key.Comment)
		}

		if key.GitHubUsername != "" {
			fmt.Printf("   GitHub Username: %s\n", key.GitHubUsername)
		} else {
			fmt.Printf("   GitHub Username: (not detected)\n")
		}

		fmt.Printf("   Account Alias: %s\n", key.AccountAlias)
		fmt.Printf("   In SSH Agent: %v\n", key.InSSHAgent)
		fmt.Printf("   Registered on GitHub: %v\n", key.IsOnGitHub)

		if discoverVerbose && key.Fingerprint != "" {
			fmt.Printf("   Fingerprint: %s\n", key.Fingerprint)
		}

		fmt.Println()
	}

	// Test GitHub connectivity for each key
	fmt.Println("🧪 Testing GitHub connectivity...")
	fmt.Println()

	githubKeys := 0
	for _, key := range keys {
		if key.IsOnGitHub {
			githubKeys++
			fmt.Printf("✅ %s: Connected to GitHub", key.AccountAlias)
			if key.GitHubUsername != "" {
				fmt.Printf(" (username: %s)", key.GitHubUsername)
			}
			fmt.Println()
		} else {
			fmt.Printf("❌ %s: Not connected to GitHub or key not registered\n", key.AccountAlias)
		}
	}

	fmt.Println()
	fmt.Printf("📊 Summary: %d total keys, %d connected to GitHub\n", len(keys), githubKeys)

	// Create account suggestions
	if githubKeys > 0 {
		fmt.Println()
		fmt.Println("💡 Suggested accounts to add:")
		fmt.Println()

		accounts, err := scanner.CreateAccountsFromSSHKeys(ctx, keys)
		if err != nil {
			return fmt.Errorf("failed to create accounts from SSH keys: %w", err)
		}

		for _, account := range accounts {
			fmt.Printf("📋 Account: %s\n", account.Account.Alias)
			if account.Account.GitHubUsername != "" {
				fmt.Printf("   GitHub: @%s\n", account.Account.GitHubUsername)
			}
			if account.Account.Name != "" {
				fmt.Printf("   Name: %s\n", account.Account.Name)
			}
			if account.Account.Email != "" {
				fmt.Printf("   Email: %s\n", account.Account.Email)
			}
			fmt.Printf("   SSH Key: %s\n", account.Account.SSHKeyPath)
			fmt.Printf("   Confidence: %d/10\n", account.Confidence)
			fmt.Println()
		}

		if discoverAutoAdd {
			fmt.Println("🚀 Auto-adding discovered accounts...")
			return addDiscoveredAccounts(ctx, accounts)
		} else {
			fmt.Println("💡 To add these accounts, run:")
			fmt.Println("   gitpersona discover --auto-add")
			fmt.Println()
			fmt.Println("   Or add them manually:")
			for _, account := range accounts {
				fmt.Printf("   gitpersona add-github %s", account.Account.Alias)
				if account.Account.GitHubUsername != "" {
					fmt.Printf(" --username %s", account.Account.GitHubUsername)
				}
				if account.Account.Name != "" {
					fmt.Printf(" --name \"%s\"", account.Account.Name)
				}
				if account.Account.Email != "" {
					fmt.Printf(" --email \"%s\"", account.Account.Email)
				}
				fmt.Printf(" --ssh-key \"%s\"", account.Account.SSHKeyPath)
				fmt.Println()
			}
		}
	}

	return nil
}

func runFullDiscovery(ctx context.Context, logger observability.Logger) error {
	fmt.Println("🔍 Running comprehensive account discovery...")
	fmt.Println()

	discovery := discovery.NewAccountDiscovery()

	accounts, err := discovery.ScanExistingAccounts()
	if err != nil {
		return fmt.Errorf("failed to discover accounts: %w", err)
	}

	if len(accounts) == 0 {
		fmt.Println("❌ No accounts discovered")
		fmt.Println("💡 Try running: gitpersona discover --ssh-only")
		return nil
	}

	fmt.Printf("✅ Discovered %d account(s)\n\n", len(accounts))

	// Display discovered accounts
	for i, account := range accounts {
		fmt.Printf("👤 Account #%d: %s\n", i+1, account.Account.Alias)

		if account.Account.Name != "" {
			fmt.Printf("   Name: %s\n", account.Account.Name)
		}

		if account.Account.Email != "" {
			fmt.Printf("   Email: %s\n", account.Account.Email)
		}

		if account.Account.GitHubUsername != "" {
			fmt.Printf("   GitHub: @%s\n", account.Account.GitHubUsername)
		}

		if account.Account.SSHKeyPath != "" {
			fmt.Printf("   SSH Key: %s\n", account.Account.SSHKeyPath)
		}

		fmt.Printf("   Source: %s\n", account.Source)
		fmt.Printf("   Confidence: %d/10\n", account.Confidence)

		if account.Conflicting {
			fmt.Printf("   ⚠️  Conflicting data found - review before adding\n")
		}

		fmt.Println()
	}

	if discoverAutoAdd {
		fmt.Println("🚀 Auto-adding discovered accounts...")
		return addDiscoveredAccounts(ctx, accounts)
	} else {
		fmt.Println("💡 To add these accounts, run:")
		fmt.Println("   gitpersona discover --auto-add")
	}

	return nil
}

func addDiscoveredAccounts(ctx context.Context, accounts []*discovery.DiscoveredAccount) error {
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Println("🔄 Adding discovered accounts...")
	fmt.Println()

	added := 0
	for _, account := range accounts {
		// Skip accounts with low confidence or missing critical data
		if account.Confidence < 7 {
			fmt.Printf("⏭️  Skipping %s (low confidence: %d/10)\n", account.Account.Alias, account.Confidence)
			continue
		}

		if account.Account.GitHubUsername == "" && account.Account.Email == "" {
			fmt.Printf("⏭️  Skipping %s (missing GitHub username and email)\n", account.Account.Alias)
			continue
		}

		// Set created timestamp if not set
		if account.Account.CreatedAt.IsZero() {
			account.Account.CreatedAt = time.Now()
		}

		// Add the account to configuration
		if err := configManager.AddAccount(account.Account); err != nil {
			fmt.Printf("❌ Failed to add account %s: %v\n", account.Account.Alias, err)
			continue
		}

		fmt.Printf("✅ Successfully added account: %s\n", account.Account.Alias)
		if account.Account.Name != "" {
			fmt.Printf("   Name: %s\n", account.Account.Name)
		}
		if account.Account.Email != "" {
			fmt.Printf("   Email: %s\n", account.Account.Email)
		}
		if account.Account.GitHubUsername != "" {
			fmt.Printf("   GitHub: @%s\n", account.Account.GitHubUsername)
		}
		added++
	}

	fmt.Println()
	if added > 0 {
		fmt.Printf("🎉 Successfully added %d account(s)!\n", added)
		fmt.Println("💡 Use 'gitpersona list' to see your accounts")

		// Set the first added account as current if no current account is set
		if configManager.GetConfig().CurrentAccount == "" && len(accounts) > 0 {
			firstAccount := accounts[0]
			if err := configManager.SetCurrentAccount(firstAccount.Account.Alias); err == nil {
				fmt.Printf("📌 Set '%s' as current account\n", firstAccount.Account.Alias)
			}
		}
	} else {
		fmt.Println("⚠️  No accounts were added")
	}

	return nil
}
