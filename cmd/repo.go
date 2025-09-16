package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories and resolve authentication issues",
	Long: `Manage repositories and automatically resolve authentication issues.

This command helps you manage Git repositories, diagnose connection problems,
and automatically resolve protocol and authentication issues.

Examples:
  gitpersona repo diagnose              # Diagnose repository issues
  gitpersona repo fix                   # Auto-fix repository issues
  gitpersona repo test                  # Test repository connection
  gitpersona repo setup [url]          # Setup new repository with best protocol`,
	RunE: executeRepo,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}

func executeRepo(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("action required. Use: diagnose, fix, test, or setup")
	}

	action := args[0]

	switch action {
	case "diagnose":
		return diagnoseRepository()
	case "fix":
		return fixRepository()
	case "test":
		return testRepository()
	case "setup":
		if len(args) < 2 {
			return fmt.Errorf("repository URL required for setup action")
		}
		return setupRepository(args[1])
	default:
		return fmt.Errorf("invalid action: %s. Use: diagnose, fix, test, or setup", action)
	}
}

// diagnoseRepository diagnoses repository issues
func diagnoseRepository() error {
	fmt.Println("🔍 Diagnosing repository issues...")
	fmt.Println("")

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Check current remote
	fmt.Println("🔗 Git Remote Configuration:")
	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("❌ Failed to get remote configuration")
	} else {
		fmt.Println(string(output))
	}

	fmt.Println("")

	// Check GitHub CLI authentication
	fmt.Println("📊 GitHub CLI Status:")
	cmd = exec.Command("gh", "auth", "status")
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("❌ GitHub CLI not authenticated")
		fmt.Println("💡 Run: gh auth login")
	} else {
		fmt.Println(string(output))
	}

	fmt.Println("")

	// Check GitHub CLI protocol setting
	fmt.Println("⚙️  GitHub CLI Protocol Setting:")
	cmd = exec.Command("gh", "config", "get", "git_protocol")
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("❌ Failed to get protocol setting")
	} else {
		protocol := strings.TrimSpace(string(output))
		fmt.Printf("Current protocol: %s\n", protocol)
	}

	fmt.Println("")

	// Test connection
	fmt.Println("🧪 Testing Connection:")
	if err := testRepository(); err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		fmt.Println("💡 Try running: gitpersona repo fix")
	} else {
		fmt.Println("✅ Connection successful")
	}

	return nil
}

// fixRepository automatically fixes repository issues
func fixRepository() error {
	fmt.Println("🔧 Auto-fixing repository issues...")
	fmt.Println("")

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Get current remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current remote URL: %w", err)
	}

	currentURL := strings.TrimSpace(string(output))
	fmt.Printf("Current remote: %s\n", currentURL)

	// Try HTTPS first (more reliable for multiple accounts)
	fmt.Println("🔄 Trying HTTPS protocol...")
	if err := switchToHTTPS(); err != nil {
		fmt.Printf("⚠️  HTTPS failed: %v\n", err)
	} else {
		// Test HTTPS
		if err := testRepository(); err == nil {
			fmt.Println("✅ HTTPS protocol working - issue resolved!")
			return nil
		}
		fmt.Println("⚠️  HTTPS not working, trying SSH...")
	}

	// Try SSH if HTTPS failed
	fmt.Println("🔄 Trying SSH protocol...")
	if err := switchToSSH(); err != nil {
		fmt.Printf("⚠️  SSH failed: %v\n", err)
	} else {
		// Test SSH
		if err := testRepository(); err == nil {
			fmt.Println("✅ SSH protocol working - issue resolved!")
			return nil
		}
	}

	return fmt.Errorf("auto-fix failed. Both protocols failed. Please check your authentication manually")
}

// testRepository tests the repository connection
func testRepository() error {
	// Try a dry-run fetch
	cmd := exec.Command("git", "fetch", "origin", "--dry-run")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fetch test failed: %w", err)
	}

	return nil
}

// setupRepository sets up a new repository with the best protocol
func setupRepository(url string) error {
	fmt.Printf("🚀 Setting up repository: %s\n", url)
	fmt.Println("")

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); err == nil {
		return fmt.Errorf("already in a git repository")
	}

	// Initialize git repository
	fmt.Println("🔄 Initializing git repository...")
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Add remote origin
	fmt.Println("🔄 Adding remote origin...")
	cmd = exec.Command("git", "remote", "add", "origin", url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add remote origin: %w", err)
	}

	// Try to determine best protocol
	fmt.Println("🤖 Detecting best protocol...")

	// Check if URL is SSH or HTTPS
	if strings.HasPrefix(url, "git@github.com:") {
		fmt.Println("✅ Using SSH protocol")
		cmd = exec.Command("gh", "config", "set", "git_protocol", "ssh")
		_ = cmd.Run() // Ignore errors
	} else if strings.HasPrefix(url, "https://github.com/") {
		fmt.Println("✅ Using HTTPS protocol")
		cmd = exec.Command("gh", "config", "set", "git_protocol", "https")
		_ = cmd.Run() // Ignore errors
	} else {
		fmt.Println("⚠️  Unknown URL format, defaulting to HTTPS")
		cmd = exec.Command("gh", "config", "set", "git_protocol", "https")
		_ = cmd.Run() // Ignore errors
	}

	// Test connection
	fmt.Println("🧪 Testing connection...")
	if err := testRepository(); err != nil {
		fmt.Printf("⚠️  Connection test failed: %v\n", err)
		fmt.Println("💡 You may need to authenticate or check the repository URL")
	} else {
		fmt.Println("✅ Repository setup successful!")
	}

	return nil
}
