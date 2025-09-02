package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/github"
)

// overviewCmd provides a complete overview of all accounts and their repositories
var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Show complete overview of all accounts and their repositories",
	Long: `Display a comprehensive overview of all configured GitHub accounts
including their repositories, configuration status, and connectivity.

This command provides a dashboard view showing:
- Account details (name, email, GitHub username)
- Repository counts and recent projects
- SSH key configuration status
- Current active account
- Connectivity verification

Examples:
  gitpersona overview
  gitpersona overview --detailed`,
	Aliases: []string{"dashboard", "summary"},
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("❌ No accounts configured.")
			fmt.Println("💡 Add your first account:")
			fmt.Println("   • gitpersona add-github username --email your@example.com")
			fmt.Println("   • gitpersona discover --auto-import")
			return nil
		}

		detailed, _ := cmd.Flags().GetBool("detailed")
		currentAccount := configManager.GetConfig().CurrentAccount

		fmt.Println("🔄 GitPersona - Overview")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println()

		// Get GitHub client for repo fetching
		githubClient, authError := getAuthenticatedGitHubClient()
		if authError != nil {
			fmt.Printf("⚠️  Limited API access: %v\n", authError)
			githubClient = github.NewClient("")
		}

		// Sort accounts by alias
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].Alias < accounts[j].Alias
		})

		totalRepos := 0

		for i, account := range accounts {
			if i > 0 {
				fmt.Println()
			}

			// Account header
			marker := "  "
			if account.Alias == currentAccount {
				marker = "▶ "
				fmt.Printf("%s🎯 %s (ACTIVE)\n", marker, account.Alias)
			} else {
				fmt.Printf("%s📋 %s\n", marker, account.Alias)
			}

			// Basic info
			fmt.Printf("     👤 %s <%s>\n", account.Name, account.Email)
			if account.GitHubUsername != "" {
				fmt.Printf("     🐙 @%s\n", account.GitHubUsername)
			}

			// SSH key status
			if account.SSHKeyPath != "" {
				fmt.Printf("     🔑 %s\n", account.SSHKeyPath)
			} else {
				fmt.Printf("     🔑 No SSH key configured\n")
			}

			// Repository info
			if account.GitHubUsername != "" {
				fmt.Printf("     📦 Fetching repositories...\n")
				repos, err := githubClient.FetchUserRepositories(account.GitHubUsername)
				if err != nil {
					fmt.Printf("     ❌ Could not fetch repositories: %v\n", err)
				} else {
					fmt.Printf("     ✅ %d repositories found\n", len(repos))
					totalRepos += len(repos)

					if detailed && len(repos) > 0 {
						// Show top 3 repositories
						sort.Slice(repos, func(i, j int) bool {
							return repos[i].Stars > repos[j].Stars
						})

						showCount := 3
						if len(repos) < showCount {
							showCount = len(repos)
						}

						fmt.Printf("     📊 Top repositories:\n")
						for j := 0; j < showCount; j++ {
							repo := repos[j]
							fmt.Printf("        • %s", repo.Name)
							if repo.Private {
								fmt.Printf(" 🔒")
							}
							if repo.Stars > 0 {
								fmt.Printf(" (⭐ %d)", repo.Stars)
							}
							if repo.Language != "" {
								fmt.Printf(" [%s]", repo.Language)
							}
							fmt.Println()
						}

						if len(repos) > showCount {
							fmt.Printf("        ... and %d more\n", len(repos)-showCount)
						}
					}
				}
			} else {
				fmt.Printf("     ❌ No GitHub username configured\n")
			}

			// Last used info
			if account.LastUsed != nil {
				fmt.Printf("     🕒 Last used: %s\n", account.LastUsed.Format("Jan 02 15:04"))
			}
		}

		// Summary
		fmt.Println()
		fmt.Println("📊 Summary")
		fmt.Println("-" + strings.Repeat("-", 30))
		fmt.Printf("Total accounts: %d\n", len(accounts))
		fmt.Printf("Total repositories: %d\n", totalRepos)
		if currentAccount != "" {
			fmt.Printf("Active account: %s\n", currentAccount)
		} else {
			fmt.Println("Active account: None")
		}

		// Quick actions
		fmt.Println()
		fmt.Println("💡 Quick actions:")
		fmt.Println("   • Switch account: gitpersona switch")
		fmt.Println("   • Add new account: gitpersona add-github username")
		fmt.Println("   • List repositories: gitpersona repos")
		fmt.Println("   • Launch TUI: gitpersona")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(overviewCmd)

	overviewCmd.Flags().Bool("detailed", false, "Show detailed repository information")
}
