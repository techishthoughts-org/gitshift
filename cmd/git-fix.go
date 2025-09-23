package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/git"
)

// gitFixCmd represents the git-fix command
var gitFixCmd = &cobra.Command{
	Use:   "git-fix",
	Short: "🔧 Fix common Git configuration issues",
	Long: `Fix common Git configuration issues that prevent proper operation.

This command will:
- Clear problematic SSH configurations that cause fork errors
- Test Git operations
- Switch to HTTPS if SSH is not working
- Clean up environment variables

Examples:
  gitpersona git-fix
  gitpersona git-fix --use-ssh
  gitpersona git-fix --test-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		useSSH, _ := cmd.Flags().GetBool("use-ssh")
		testOnly, _ := cmd.Flags().GetBool("test-only")

		var gitManager *git.Manager
		if useSSH {
			gitManager = git.NewSSHManager()
		} else {
			gitManager = git.NewManager()
		}

		fmt.Printf("🔧 Git Configuration Fix Tool\n")
		fmt.Printf("=============================\n\n")

		// Test if we're in a git repository
		if !gitManager.IsGitRepository() {
			fmt.Printf("ℹ️  Not in a Git repository - skipping repository-specific fixes\n")
		} else {
			fmt.Printf("✅ Git repository detected\n")
		}

		if testOnly {
			return runGitTests(gitManager)
		}

		// 1. Clear problematic SSH configurations
		fmt.Printf("🧹 Clearing problematic SSH configurations...\n")
		if err := gitManager.ClearSSHConfig(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to clear SSH config: %v\n", err)
		} else {
			fmt.Printf("✅ SSH configuration cleared\n")
		}

		// 2. Test Git operations
		fmt.Printf("🧪 Testing Git operations...\n")
		if err := gitManager.TestGitOperation(); err != nil {
			fmt.Printf("❌ Git test failed: %v\n", err)

			if !useSSH {
				fmt.Printf("💡 Consider running with --use-ssh if you need SSH functionality\n")
			}
		} else {
			fmt.Printf("✅ Git operations working correctly\n")
		}

		// 3. Fix remote URL if in a repository
		if gitManager.IsGitRepository() {
			fmt.Printf("🔗 Checking remote configuration...\n")

			currentURL, err := gitManager.GetCurrentRemoteURL("origin")
			if err != nil {
				fmt.Printf("⚠️  Could not get remote URL: %v\n", err)
			} else {
				fmt.Printf("   Current remote: %s\n", currentURL)

				// Set the appropriate URL format
				if err := gitManager.SetRemoteURL("origin", currentURL); err != nil {
					fmt.Printf("⚠️  Failed to normalize remote URL: %v\n", err)
				} else {
					newURL, _ := gitManager.GetCurrentRemoteURL("origin")
					if newURL != currentURL {
						fmt.Printf("✅ Remote URL updated to: %s\n", newURL)
					} else {
						fmt.Printf("✅ Remote URL is already correct\n")
					}
				}
			}

			// 4. Test fetch operation
			fmt.Printf("📡 Testing fetch operation...\n")
			if err := gitManager.SafeFetch("origin"); err != nil {
				fmt.Printf("⚠️  Fetch test failed: %v\n", err)
				fmt.Printf("   This may be due to authentication or network issues\n")
			} else {
				fmt.Printf("✅ Fetch operation successful\n")
			}
		}

		// 5. Show current Git user configuration
		fmt.Printf("👤 Checking Git user configuration...\n")
		name, email, err := gitManager.GetUserConfig()
		if err != nil {
			fmt.Printf("⚠️  Failed to get Git user config: %v\n", err)
		} else {
			if name == "" && email == "" {
				fmt.Printf("⚠️  No Git user configuration found\n")
				fmt.Printf("   Set with: git config --global user.name \"Your Name\"\n")
				fmt.Printf("   Set with: git config --global user.email \"your@email.com\"\n")
			} else {
				fmt.Printf("✅ Git user: %s <%s>\n", name, email)
			}
		}

		fmt.Printf("\n🎉 Git configuration fix completed!\n")
		fmt.Printf("   If you still experience issues, try running with --use-ssh or --test-only\n")

		return nil
	},
}

// runGitTests runs comprehensive Git tests
func runGitTests(gitManager *git.Manager) error {
	fmt.Printf("🧪 Running comprehensive Git tests...\n\n")

	tests := []struct {
		name string
		test func() error
	}{
		{"Repository Detection", func() error {
			if !gitManager.IsGitRepository() {
				return fmt.Errorf("not in a Git repository")
			}
			return nil
		}},
		{"Git Status", func() error {
			return gitManager.TestGitOperation()
		}},
		{"Remote URL Check", func() error {
			if !gitManager.IsGitRepository() {
				return nil // Skip if not in repo
			}
			_, err := gitManager.GetCurrentRemoteURL("origin")
			return err
		}},
		{"User Configuration", func() error {
			name, email, err := gitManager.GetUserConfig()
			if err != nil {
				return err
			}
			if name == "" || email == "" {
				return fmt.Errorf("incomplete user configuration")
			}
			return nil
		}},
	}

	passed := 0
	total := len(tests)

	for _, test := range tests {
		fmt.Printf("   Testing %s... ", test.name)
		if err := test.test(); err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
		} else {
			fmt.Printf("✅ Passed\n")
			passed++
		}
	}

	fmt.Printf("\n📊 Test Results: %d/%d passed\n", passed, total)

	if passed == total {
		fmt.Printf("🎉 All tests passed! Git configuration is working correctly.\n")
	} else {
		fmt.Printf("⚠️  Some tests failed. Run 'gitpersona git-fix' to attempt fixes.\n")
	}

	return nil
}

func init() {
	gitFixCmd.Flags().Bool("use-ssh", false, "Configure Git to use SSH instead of HTTPS")
	gitFixCmd.Flags().Bool("test-only", false, "Only run tests without making changes")

	rootCmd.AddCommand(gitFixCmd)
}
