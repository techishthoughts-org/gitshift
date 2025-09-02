package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured GitHub accounts",
	Long: `List all configured GitHub accounts with their details.

The output shows the alias, name, email, and additional information for each account.
The current active account is marked with an asterisk (*).

Examples:
  gitpersona list
  gitpersona list --format table
  gitpersona list --format json`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("No accounts configured. Use 'gitpersona add' to add an account.")
			return nil
		}

		format, _ := cmd.Flags().GetString("format")
		currentAccount := configManager.GetConfig().CurrentAccount

		switch format {
		case "json":
			return printAccountsJSON(accounts)
		case "table":
			return printAccountsTable(accounts, currentAccount)
		default:
			return printAccountsDefault(accounts, currentAccount)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("format", "f", "default", "Output format (default, table, json)")
}

func printAccountsDefault(accounts []*models.Account, currentAccount string) error {
	// Sort accounts by alias
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Alias < accounts[j].Alias
	})

	fmt.Println("Configured GitHub Accounts:")
	fmt.Println()

	for _, account := range accounts {
		marker := "  "
		if account.Alias == currentAccount {
			marker = "* "
		}

		fmt.Printf("%s%s\n", marker, account.Alias)
		fmt.Printf("    Name:  %s\n", account.Name)
		fmt.Printf("    Email: %s\n", account.Email)

		if account.GitHubUsername != "" {
			fmt.Printf("    GitHub: @%s\n", account.GitHubUsername)
		} else {
			fmt.Printf("    GitHub: (not set)\n")
		}

		if account.SSHKeyPath != "" {
			fmt.Printf("    SSH Key: %s\n", account.SSHKeyPath)
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

	if currentAccount != "" {
		fmt.Printf("Current active account: %s\n", currentAccount)
	} else {
		fmt.Println("No active account set")
	}

	return nil
}

func printAccountsTable(accounts []*models.Account, currentAccount string) error {
	// Sort accounts by alias
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Alias < accounts[j].Alias
	})

	// Calculate column widths
	aliasWidth := len("ALIAS")
	nameWidth := len("NAME")
	emailWidth := len("EMAIL")
	githubWidth := len("GITHUB")
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
		githubDisplay := account.GitHubUsername
		if githubDisplay == "" {
			githubDisplay = "(not set)"
		}
		if len(githubDisplay) > githubWidth {
			githubWidth = len(githubDisplay)
		}
		if len(account.Description) > descWidth {
			descWidth = len(account.Description)
		}
	}

	// Add padding
	aliasWidth += 2
	nameWidth += 2
	emailWidth += 2
	githubWidth += 2
	descWidth += 2

	// Print header
	fmt.Printf("%-1s %-*s %-*s %-*s %-*s %-*s\n", "", aliasWidth, "ALIAS", nameWidth, "NAME", emailWidth, "EMAIL", githubWidth, "GITHUB", descWidth, "DESCRIPTION")
	fmt.Printf("%-1s %s %s %s %s %s\n", "",
		strings.Repeat("-", aliasWidth),
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", emailWidth),
		strings.Repeat("-", githubWidth),
		strings.Repeat("-", descWidth))

	// Print accounts
	for _, account := range accounts {
		marker := " "
		if account.Alias == currentAccount {
			marker = "*"
		}

		githubDisplay := account.GitHubUsername
		if githubDisplay == "" {
			githubDisplay = "(not set)"
		}

		fmt.Printf("%-1s %-*s %-*s %-*s %-*s %-*s\n",
			marker,
			aliasWidth, account.Alias,
			nameWidth, account.Name,
			emailWidth, account.Email,
			githubWidth, githubDisplay,
			descWidth, account.Description)
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
	// Sort accounts by alias
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Alias < accounts[j].Alias
	})

	fmt.Println("[")
	for i, account := range accounts {
		fmt.Printf("  {\n")
		fmt.Printf("    \"alias\": \"%s\",\n", account.Alias)
		fmt.Printf("    \"name\": \"%s\",\n", account.Name)
		fmt.Printf("    \"email\": \"%s\",\n", account.Email)
		fmt.Printf("    \"github_username\": \"%s\",\n", account.GitHubUsername)
		if account.SSHKeyPath != "" {
			fmt.Printf("    \"ssh_key_path\": \"%s\",\n", account.SSHKeyPath)
		}
		if account.Description != "" {
			fmt.Printf("    \"description\": \"%s\",\n", account.Description)
		}
		fmt.Printf("    \"is_default\": %t,\n", account.IsDefault)
		fmt.Printf("    \"created_at\": \"%s\"", account.CreatedAt.Format(time.RFC3339))
		if account.LastUsed != nil {
			fmt.Printf(",\n    \"last_used\": \"%s\"", account.LastUsed.Format(time.RFC3339))
		}
		fmt.Printf("\n  }")

		if i < len(accounts)-1 {
			fmt.Println(",")
		} else {
			fmt.Println()
		}
	}
	fmt.Println("]")

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
