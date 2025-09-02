package cmd

import (
	"fmt"
	"sort"

	"github.com/thukabjj/GitPersona/internal/config"
	"github.com/thukabjj/GitPersona/internal/github"
	"github.com/thukabjj/GitPersona/internal/models"
	"github.com/spf13/cobra"
)

// reposCmd represents the repos command
var reposCmd = &cobra.Command{
	Use:   "repos [account-alias]",
	Short: "List repositories for GitHub accounts",
	Long: `List repositories for configured GitHub accounts.

This command fetches repositories from GitHub API to verify account
access and show available projects. If no account is specified,
it shows repositories for all configured accounts.

Examples:
  gh-switcher repos work       # Show work account repos
  gh-switcher repos personal   # Show personal account repos
  gh-switcher repos           # Show all accounts' repos
  gh-switcher repos --private  # Include private repositories`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			fmt.Println("❌ No accounts configured. Add an account first.")
			return nil
		}

		// Get command flags
		includePrivate, _ := cmd.Flags().GetBool("private")
		showStars, _ := cmd.Flags().GetBool("stars")
		limit, _ := cmd.Flags().GetInt("limit")

		// Determine which accounts to check
		var targetAccounts []*models.Account
		if len(args) > 0 {
			// Specific account
			account, err := configManager.GetAccount(args[0])
			if err != nil {
				return fmt.Errorf("account '%s' not found", args[0])
			}
			targetAccounts = append(targetAccounts, account)
		} else {
			// All accounts
			targetAccounts = accounts
		}

		// Get authenticated GitHub client
		githubClient, err := getAuthenticatedGitHubClient()
		if err != nil {
			fmt.Printf("⚠️  Using unauthenticated access: %v\n", err)
			githubClient = github.NewClient("")
		}

		totalRepos := 0

		for i, account := range targetAccounts {
			if i > 0 {
				fmt.Println() // Spacing between accounts
			}

			fmt.Printf("🔍 Fetching repositories for @%s (%s)...\n",
				account.GitHubUsername, account.Alias)

			if account.GitHubUsername == "" {
				fmt.Printf("❌ No GitHub username configured for account '%s'\n", account.Alias)
				fmt.Printf("   Update with: gh-switcher add %s --github-username USERNAME --overwrite\n", account.Alias)
				continue
			}

			repos, err := githubClient.FetchUserRepositories(account.GitHubUsername)
			if err != nil {
				fmt.Printf("❌ Failed to fetch repositories: %v\n", err)
				continue
			}

			// Filter repositories
			var filteredRepos []*github.Repository
			for _, repo := range repos {
				if !includePrivate && repo.Private {
					continue
				}
				filteredRepos = append(filteredRepos, repo)
			}

			// Sort by stars (descending) or name
			if showStars {
				sort.Slice(filteredRepos, func(i, j int) bool {
					return filteredRepos[i].Stars > filteredRepos[j].Stars
				})
			} else {
				sort.Slice(filteredRepos, func(i, j int) bool {
					return filteredRepos[i].Name < filteredRepos[j].Name
				})
			}

			// Apply limit
			if limit > 0 && len(filteredRepos) > limit {
				filteredRepos = filteredRepos[:limit]
			}

			fmt.Printf("✅ Found %d repositories for @%s:\n", len(filteredRepos), account.GitHubUsername)
			fmt.Println()

			// Display repositories
			for j, repo := range filteredRepos {
				if j >= 10 && limit == 0 { // Show max 10 by default unless limit specified
					remaining := len(filteredRepos) - 10
					fmt.Printf("   ... and %d more repositories\n", remaining)
					break
				}

				// Repository info
				fmt.Printf("   📦 %s", repo.Name)

				if repo.Private {
					fmt.Printf(" 🔒")
				}
				if repo.Fork {
					fmt.Printf(" 🍴")
				}
				if repo.Archived {
					fmt.Printf(" 📦")
				}

				fmt.Printf("\n")

				if repo.Description != "" {
					fmt.Printf("      %s\n", repo.Description)
				}

				// Language and stats
				var stats []string
				if repo.Language != "" {
					stats = append(stats, fmt.Sprintf("📝 %s", repo.Language))
				}
				if repo.Stars > 0 {
					stats = append(stats, fmt.Sprintf("⭐ %d", repo.Stars))
				}
				if repo.Forks > 0 {
					stats = append(stats, fmt.Sprintf("🍴 %d", repo.Forks))
				}
				stats = append(stats, fmt.Sprintf("🕒 %s", repo.UpdatedAt))

				if len(stats) > 0 {
					fmt.Printf("      %s\n", joinWithSeparator(stats, " • "))
				}

				fmt.Printf("      🔗 %s\n", repo.HTMLURL)
				fmt.Println()
			}

			totalRepos += len(filteredRepos)
		}

		fmt.Printf("📊 Total repositories shown: %d\n", totalRepos)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reposCmd)

	reposCmd.Flags().Bool("private", false, "Include private repositories")
	reposCmd.Flags().Bool("stars", false, "Sort by star count (descending)")
	reposCmd.Flags().IntP("limit", "l", 0, "Limit number of repositories shown per account (0 = no limit)")
}

// joinWithSeparator joins strings with a separator
func joinWithSeparator(items []string, separator string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += separator
		}
		result += item
	}
	return result
}
