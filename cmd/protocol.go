package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var protocolCmd = &cobra.Command{
	Use:   "protocol",
	Short: "Manage Git protocol settings and resolve authentication issues",
	Long: `Manage Git protocol settings and automatically resolve authentication issues.

This command helps you switch between SSH and HTTPS protocols, diagnose connection problems,
and automatically resolve common authentication issues when working with multiple GitHub accounts.

Examples:
  gitpersona protocol https          # Switch to HTTPS protocol
  gitpersona protocol ssh            # Switch to SSH protocol
  gitpersona protocol diagnose       # Diagnose current protocol issues
  gitpersona protocol auto           # Auto-detect and use best protocol
  gitpersona protocol test           # Test current protocol connection`,
	RunE: executeProtocol,
}

func init() {
	rootCmd.AddCommand(protocolCmd)
}

func executeProtocol(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("protocol type required. Use: https, ssh, diagnose, auto, or test")
	}

	protocolType := args[0]

	switch protocolType {
	case "https":
		return switchToHTTPS()
	case "ssh":
		return switchToSSH()
	case "diagnose":
		return diagnoseProtocol()
	case "auto":
		return autoDetectProtocol()
	case "test":
		return testProtocol()
	default:
		return fmt.Errorf("invalid protocol type: %s. Use: https, ssh, diagnose, auto, or test", protocolType)
	}
}

// switchToHTTPS switches the current repository to use HTTPS protocol
func switchToHTTPS() error {
	fmt.Println("🔄 Switching to HTTPS protocol...")

	// Set GitHub CLI to use HTTPS
	cmd := exec.Command("gh", "config", "set", "git_protocol", "https")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set GitHub CLI protocol to HTTPS: %w", err)
	}

	// Get current remote URL
	cmd = exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current remote URL: %w", err)
	}

	currentURL := strings.TrimSpace(string(output))

	// Convert SSH URL to HTTPS if needed
	if strings.HasPrefix(currentURL, "git@github.com:") {
		newURL := strings.Replace(currentURL, "git@github.com:", "https://github.com/", 1)
		newURL = strings.Replace(newURL, ".git", ".git", 1)

		cmd = exec.Command("git", "remote", "set-url", "origin", newURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update remote URL: %w", err)
		}

		fmt.Printf("✅ Updated remote URL: %s\n", newURL)
	} else if strings.HasPrefix(currentURL, "https://github.com/") {
		fmt.Println("✅ Already using HTTPS protocol")
	} else {
		fmt.Printf("⚠️  Unknown remote URL format: %s\n", currentURL)
	}

	fmt.Println("✅ Switched to HTTPS protocol")
	fmt.Println("💡 Benefits:")
	fmt.Println("   • Works with GitHub CLI authentication")
	fmt.Println("   • No SSH key conflicts")
	fmt.Println("   • Automatic token management")

	return nil
}

// switchToSSH switches the current repository to use SSH protocol
func switchToSSH() error {
	fmt.Println("🔄 Switching to SSH protocol...")

	// Set GitHub CLI to use SSH
	cmd := exec.Command("gh", "config", "set", "git_protocol", "ssh")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set GitHub CLI protocol to SSH: %w", err)
	}

	// Get current remote URL
	cmd = exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current remote URL: %w", err)
	}

	currentURL := strings.TrimSpace(string(output))

	// Convert HTTPS URL to SSH if needed
	if strings.HasPrefix(currentURL, "https://github.com/") {
		newURL := strings.Replace(currentURL, "https://github.com/", "git@github.com:", 1)

		cmd = exec.Command("git", "remote", "set-url", "origin", newURL)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update remote URL: %w", err)
		}

		fmt.Printf("✅ Updated remote URL: %s\n", newURL)
	} else if strings.HasPrefix(currentURL, "git@github.com:") {
		fmt.Println("✅ Already using SSH protocol")
	} else {
		fmt.Printf("⚠️  Unknown remote URL format: %s\n", currentURL)
	}

	fmt.Println("✅ Switched to SSH protocol")
	fmt.Println("💡 Benefits:")
	fmt.Println("   • More secure")
	fmt.Println("   • No need to enter credentials")
	fmt.Println("   • Better for automation")

	return nil
}

// diagnoseProtocol diagnoses current protocol issues
func diagnoseProtocol() error {
	fmt.Println("🔍 Diagnosing protocol issues...")
	fmt.Println("")

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Check GitHub CLI authentication
	fmt.Println("📊 GitHub CLI Status:")
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("❌ GitHub CLI not authenticated")
		fmt.Println("💡 Run: gh auth login")
	} else {
		fmt.Println(string(output))
	}

	fmt.Println("")

	// Check current remote
	fmt.Println("🔗 Git Remote Configuration:")
	cmd = exec.Command("git", "remote", "-v")
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("❌ Failed to get remote configuration")
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
	if err := testProtocol(); err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		fmt.Println("💡 Try switching protocols or check authentication")
	} else {
		fmt.Println("✅ Connection successful")
	}

	return nil
}

// autoDetectProtocol automatically detects and uses the best protocol
func autoDetectProtocol() error {
	fmt.Println("🤖 Auto-detecting best protocol...")

	// First try HTTPS (more reliable for multiple accounts)
	fmt.Println("🔄 Trying HTTPS protocol...")
	if err := switchToHTTPS(); err != nil {
		fmt.Printf("⚠️  HTTPS failed: %v\n", err)
	} else {
		// Test HTTPS
		if err := testProtocol(); err == nil {
			fmt.Println("✅ HTTPS protocol working - keeping it")
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
		if err := testProtocol(); err == nil {
			fmt.Println("✅ SSH protocol working - keeping it")
			return nil
		}
	}

	return fmt.Errorf("both protocols failed. Please check your authentication")
}

// testProtocol tests the current protocol connection
func testProtocol() error {
	// Try a dry-run fetch
	cmd := exec.Command("git", "fetch", "origin", "--dry-run")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fetch test failed: %w", err)
	}

	return nil
}
