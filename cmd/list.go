package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured Git platform accounts",
	Long: `List all configured Git platform accounts with their details.

The output shows the alias, name, email, platform, and additional information for each account.
The current active account is marked with an asterisk (*).

Displays accounts from all platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)
- Bitbucket (coming soon)
- Custom Git platforms

Examples:
  # List all accounts
  gitshift list

  # List in table format
  gitshift list --format table

  # List in JSON format
  gitshift list --format json`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		pendingAccounts := configManager.ListPendingAccounts()

		if len(accounts) == 0 && len(pendingAccounts) == 0 {
			fmt.Println("No accounts configured. Use 'gitshift add' to add an account.")
			return nil
		}

		format, _ := cmd.Flags().GetString("format")
		currentAccount := configManager.GetConfig().CurrentAccount

		// Show active accounts
		if len(accounts) > 0 {
			switch format {
			case "json":
				return printAccountsJSON(accounts)
			case "table":
				return printAccountsTable(accounts, currentAccount)
			default:
				if err := printAccountsDefault(accounts, currentAccount); err != nil {
					return err
				}
			}
		}

		// Show pending accounts if any
		if len(pendingAccounts) > 0 {
			fmt.Println()
			fmt.Println("📋 Pending Accounts (need completion):")
			fmt.Println()

			for _, pending := range pendingAccounts {
				fmt.Printf("  %s\n", pending.Alias)
				if pending.GitHubUsername != "" {
					fmt.Printf("    GitHub: @%s\n", pending.GitHubUsername)
				}
				fmt.Printf("    Missing: %v\n", pending.MissingFields)
				fmt.Printf("    Source: %s\n", pending.Source)
				fmt.Printf("    💡 Complete with: gitshift complete %s --name \"Your Name\" --email \"your@email.com\"\n", pending.Alias)
				fmt.Println()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("format", "f", "default", "Output format (default, table, json)")
}

func printAccountsDefault(accounts []*models.Account, currentAccount string) error {
	// Group accounts by platform
	accountsByPlatform := groupAccountsByPlatform(accounts)

	// Sort platforms for consistent output
	platforms := make([]string, 0, len(accountsByPlatform))
	for platform := range accountsByPlatform {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	// Print accounts grouped by platform
	for _, platform := range platforms {
		platformAccounts := accountsByPlatform[platform]
		// Sort accounts within platform by alias
		sort.Slice(platformAccounts, func(i, j int) bool {
			return platformAccounts[i].Alias < platformAccounts[j].Alias
		})

		platformDisplay := getPlatformDisplay(platform)
		fmt.Printf("%s\n", platformDisplay)
		fmt.Println()

		for _, account := range platformAccounts {
			marker := "  "
			if account.Alias == currentAccount {
				marker = "* "
			}

			fmt.Printf("%s%s\n", marker, account.Alias)
			fmt.Printf("    Name:  %s\n", account.Name)
			fmt.Printf("    Email: %s\n", account.Email)

			// Display platform-specific username
			username := account.GetUsername()
			if username != "" {
				platformName := strings.ToUpper(platform[:1]) + platform[1:]
				fmt.Printf("    %s: @%s\n", platformName, username)
			} else {
				platformName := strings.ToUpper(platform[:1]) + platform[1:]
				fmt.Printf("    %s: (not set)\n", platformName)
			}

			// Display domain if different from default
			domain := account.GetDomain()
			defaultDomain := getDefaultDomain(platform)
			if domain != "" && domain != defaultDomain {
				fmt.Printf("    Domain: %s\n", domain)
			}

			if account.SSHKeyPath != "" {
				fmt.Printf("    SSH Key: %s\n", account.SSHKeyPath)
			}

			// Display GPG status
			if account.HasGPGKey() {
				gpgStatus := "🔐 GPG: Enabled"
				if !account.IsGPGEnabled() {
					gpgStatus = "🔓 GPG: Configured (signing disabled)"
				}
				fmt.Printf("    %s (%s)\n", gpgStatus, account.GPGKeyID)
				if account.IsGPGKeyExpired() {
					fmt.Printf("       ⚠️  Key expired: %s\n", account.GPGKeyExpiry.Format("2006-01-02"))
				}
			} else {
				fmt.Printf("    ⚪ GPG: Not configured\n")
			}

			if account.Description != "" {
				fmt.Printf("    Description: %s\n", account.Description)
			}

			if account.LastUsed != nil {
				fmt.Printf("    Last Used: %s\n", formatTime(*account.LastUsed))
			}

			fmt.Printf("    Created: %s\n", formatTime(account.CreatedAt))
			fmt.Println()
		}
	}

	if currentAccount != "" {
		fmt.Printf("Current active account: %s\n", currentAccount)
	} else {
		fmt.Println("No active account set")
	}

	return nil
}

func printAccountsTable(accounts []*models.Account, currentAccount string) error {
	// Group accounts by platform
	accountsByPlatform := groupAccountsByPlatform(accounts)

	// Sort platforms for consistent output
	platforms := make([]string, 0, len(accountsByPlatform))
	for platform := range accountsByPlatform {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	// Calculate column widths
	aliasWidth := len("ALIAS")
	nameWidth := len("NAME")
	emailWidth := len("EMAIL")
	platformWidth := len("PLATFORM")
	usernameWidth := len("USERNAME")
	descWidth := len("DESCRIPTION")

	for _, account := range accounts {
		if len(account.Alias) > aliasWidth {
			aliasWidth = len(account.Alias)
		}
		if len(account.Name) > nameWidth {
			nameWidth = len(account.Name)
		}
		if len(account.Email) > emailWidth {
			emailWidth = len(account.Email)
		}
		platformDisplay := strings.ToUpper(account.GetPlatform())
		if len(platformDisplay) > platformWidth {
			platformWidth = len(platformDisplay)
		}
		usernameDisplay := account.GetUsername()
		if usernameDisplay == "" {
			usernameDisplay = "(not set)"
		}
		if len(usernameDisplay) > usernameWidth {
			usernameWidth = len(usernameDisplay)
		}
		if len(account.Description) > descWidth {
			descWidth = len(account.Description)
		}
	}

	// Add padding
	aliasWidth += 2
	nameWidth += 2
	emailWidth += 2
	platformWidth += 2
	usernameWidth += 2
	descWidth += 2

	// Print accounts grouped by platform
	for _, platform := range platforms {
		platformAccounts := accountsByPlatform[platform]
		// Sort accounts within platform by alias
		sort.Slice(platformAccounts, func(i, j int) bool {
			return platformAccounts[i].Alias < platformAccounts[j].Alias
		})

		platformDisplay := getPlatformDisplay(platform)
		fmt.Printf("\n%s\n", platformDisplay)

		// Print header
		fmt.Printf("%-1s %-*s %-*s %-*s %-*s %-*s %-*s\n", "", aliasWidth, "ALIAS", nameWidth, "NAME", emailWidth, "EMAIL", platformWidth, "PLATFORM", usernameWidth, "USERNAME", descWidth, "DESCRIPTION")
		fmt.Printf("%-1s %s %s %s %s %s %s\n", "",
			strings.Repeat("-", aliasWidth),
			strings.Repeat("-", nameWidth),
			strings.Repeat("-", emailWidth),
			strings.Repeat("-", platformWidth),
			strings.Repeat("-", usernameWidth),
			strings.Repeat("-", descWidth))

		// Print accounts
		for _, account := range platformAccounts {
			marker := " "
			if account.Alias == currentAccount {
				marker = "*"
			}

			usernameDisplay := account.GetUsername()
			if usernameDisplay == "" {
				usernameDisplay = "(not set)"
			}
			platformDisplay := strings.ToUpper(account.GetPlatform())

			fmt.Printf("%-1s %-*s %-*s %-*s %-*s %-*s %-*s\n",
				marker,
				aliasWidth, account.Alias,
				nameWidth, account.Name,
				emailWidth, account.Email,
				platformWidth, platformDisplay,
				usernameWidth, usernameDisplay,
				descWidth, account.Description)
		}
	}

	fmt.Println()
	if currentAccount != "" {
		fmt.Printf("* Current active account: %s\n", currentAccount)
	} else {
		fmt.Println("No active account set")
	}

	return nil
}

func printAccountsJSON(accounts []*models.Account) error {
	// Group accounts by platform
	accountsByPlatform := groupAccountsByPlatform(accounts)

	// Sort platforms for consistent output
	platforms := make([]string, 0, len(accountsByPlatform))
	for platform := range accountsByPlatform {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	fmt.Println("{")
	platformIndex := 0
	for _, platform := range platforms {
		platformAccounts := accountsByPlatform[platform]
		// Sort accounts within platform by alias
		sort.Slice(platformAccounts, func(i, j int) bool {
			return platformAccounts[i].Alias < platformAccounts[j].Alias
		})

		fmt.Printf("  \"%s\": [\n", platform)
		for i, account := range platformAccounts {
			fmt.Printf("    {\n")
			fmt.Printf("      \"alias\": \"%s\",\n", account.Alias)
			fmt.Printf("      \"name\": \"%s\",\n", account.Name)
			fmt.Printf("      \"email\": \"%s\",\n", account.Email)
			fmt.Printf("      \"platform\": \"%s\",\n", account.GetPlatform())
			if account.GetDomain() != "" {
				fmt.Printf("      \"domain\": \"%s\",\n", account.GetDomain())
			}
			username := account.GetUsername()
			if username != "" {
				fmt.Printf("      \"username\": \"%s\",\n", username)
			}
			// Keep github_username for backward compatibility
			if account.GitHubUsername != "" {
				fmt.Printf("      \"github_username\": \"%s\",\n", account.GitHubUsername)
			}
			if account.SSHKeyPath != "" {
				fmt.Printf("      \"ssh_key_path\": \"%s\",\n", account.SSHKeyPath)
			}
			if account.HasGPGKey() {
				fmt.Printf("      \"gpg_key_id\": \"%s\",\n", account.GPGKeyID)
				fmt.Printf("      \"gpg_enabled\": %t,\n", account.IsGPGEnabled())
				if account.GPGKeyFingerprint != "" {
					fmt.Printf("      \"gpg_fingerprint\": \"%s\",\n", account.GPGKeyFingerprint)
				}
			}
			if account.Description != "" {
				fmt.Printf("      \"description\": \"%s\",\n", account.Description)
			}
			fmt.Printf("      \"is_default\": %t,\n", account.IsDefault)
			fmt.Printf("      \"created_at\": \"%s\"", account.CreatedAt.Format(time.RFC3339))
			if account.LastUsed != nil {
				fmt.Printf(",\n      \"last_used\": \"%s\"", account.LastUsed.Format(time.RFC3339))
			}
			fmt.Printf("\n    }")

			if i < len(platformAccounts)-1 {
				fmt.Println(",")
			} else {
				fmt.Println()
			}
		}
		fmt.Printf("  ]")

		if platformIndex < len(platforms)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
		platformIndex++
	}
	fmt.Println("}")

	return nil
}

func formatTime(t time.Time) string {
	now := time.Now()

	if t.After(now.Add(-24 * time.Hour)) {
		return t.Format("Today 15:04")
	}

	if t.After(now.Add(-7 * 24 * time.Hour)) {
		return t.Format("Mon 15:04")
	}

	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}

	return t.Format("Jan 02, 2006")
}

// groupAccountsByPlatform groups accounts by their platform
func groupAccountsByPlatform(accounts []*models.Account) map[string][]*models.Account {
	grouped := make(map[string][]*models.Account)
	for _, account := range accounts {
		platform := account.GetPlatform()
		grouped[platform] = append(grouped[platform], account)
	}
	return grouped
}

// getPlatformDisplay returns a formatted platform header with emoji
func getPlatformDisplay(platform string) string {
	platformName := strings.ToUpper(platform[:1]) + platform[1:]
	var emoji string
	switch platform {
	case "github":
		emoji = "🐙"
	case "gitlab":
		emoji = "🦊"
	case "bitbucket":
		emoji = "🪣"
	default:
		emoji = "🔧"
	}
	return fmt.Sprintf("%s %s Accounts", emoji, platformName)
}

// getDefaultDomain returns the default domain for a platform
func getDefaultDomain(platform string) string {
	switch platform {
	case "github":
		return "github.com"
	case "gitlab":
		return "gitlab.com"
	case "bitbucket":
		return "bitbucket.org"
	default:
		return ""
	}
}
