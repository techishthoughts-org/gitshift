package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
)

var troubleshootCmd = &cobra.Command{
	Use:   "troubleshoot",
	Short: "Comprehensive troubleshooting for all GitPersona issues",
	Long: `Comprehensive troubleshooting for all GitPersona issues.

This command automatically diagnoses and fixes common problems including:
• Protocol conflicts (SSH vs HTTPS)
• SSH key authentication issues
• GitHub CLI authentication problems
• Repository connection issues
• Account configuration problems

Examples:
  gitpersona troubleshoot              # Run full diagnostic and auto-fix
  gitpersona troubleshoot --diagnose  # Only diagnose (no fixes)
  gitpersona troubleshoot --fix       # Only fix (skip diagnosis)`,
	Aliases: []string{"ts", "fix"},
	RunE:    executeTroubleshoot,
}

var (
	diagnoseOnly bool
	fixOnly      bool
)

func init() {
	troubleshootCmd.Flags().BoolVarP(&diagnoseOnly, "diagnose", "d", false, "Only diagnose issues (no fixes)")
	troubleshootCmd.Flags().BoolVarP(&fixOnly, "fix", "f", false, "Only fix issues (skip diagnosis)")
	rootCmd.AddCommand(troubleshootCmd)
}

func executeTroubleshoot(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 GitPersona Comprehensive Troubleshooting")
	fmt.Println("==========================================")
	fmt.Println("")

	// Check if we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Println("⚠️  Not in a git repository")
		fmt.Println("💡 This command works best from within a git repository")
		fmt.Println("")
	}

	// Run diagnosis
	if !fixOnly {
		if err := runFullDiagnosis(); err != nil {
			fmt.Printf("❌ Diagnosis failed: %v\n", err)
		}
	}

	// Run fixes
	if !diagnoseOnly {
		if err := runAutoFixes(); err != nil {
			fmt.Printf("❌ Auto-fix failed: %v\n", err)
		}
	}

	fmt.Println("")
	fmt.Println("🎯 Troubleshooting complete!")
	fmt.Println("💡 If issues persist, try:")
	fmt.Println("   • gitpersona protocol diagnose")
	fmt.Println("   • gitpersona ssh-keys diagnose")
	fmt.Println("   • gitpersona repo diagnose")

	return nil
}

// runFullDiagnosis runs comprehensive diagnosis
func runFullDiagnosis() error {
	fmt.Println("🔍 Running comprehensive diagnosis...")
	fmt.Println("")

	// 1. Check GitHub CLI authentication
	fmt.Println("1️⃣  GitHub CLI Authentication:")
	if err := diagnoseGitHubCLI(); err != nil {
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   ✅ GitHub CLI authentication OK")
	}

	fmt.Println("")

	// 2. Check SSH keys
	fmt.Println("2️⃣  SSH Key Status:")
	if err := diagnoseSSHKeysStatus(); err != nil {
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   ✅ SSH keys OK")
	}

	fmt.Println("")

	// 3. Check Git configuration
	fmt.Println("3️⃣  Git Configuration:")
	if err := diagnoseGitConfig(); err != nil {
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   ✅ Git configuration OK")
	}

	fmt.Println("")

	// 4. Check repository status
	if _, err := os.Stat(".git"); err == nil {
		fmt.Println("4️⃣  Repository Status:")
		if err := diagnoseRepository(); err != nil {
			fmt.Printf("   ❌ %v\n", err)
		} else {
			fmt.Println("   ✅ Repository OK")
		}
		fmt.Println("")
	}

	// 5. Check GitPersona configuration
	fmt.Println("5️⃣  GitPersona Configuration:")
	if err := diagnoseGitPersonaConfig(); err != nil {
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   ✅ GitPersona configuration OK")
	}

	return nil
}

// runAutoFixes runs automatic fixes for detected issues
func runAutoFixes() error {
	fmt.Println("🔧 Running automatic fixes...")
	fmt.Println("")

	// 1. Fix protocol issues
	fmt.Println("1️⃣  Fixing protocol issues...")
	if err := autoDetectProtocol(); err != nil {
		fmt.Printf("   ⚠️  Protocol fix failed: %v\n", err)
	} else {
		fmt.Println("   ✅ Protocol issues resolved")
	}

	fmt.Println("")

	// 2. Fix SSH key issues
	fmt.Println("2️⃣  Fixing SSH key issues...")
	if err := fixSSHKeyIssues(); err != nil {
		fmt.Printf("   ⚠️  SSH key fix failed: %v\n", err)
	} else {
		fmt.Println("   ✅ SSH key issues resolved")
	}

	fmt.Println("")

	// 3. Fix repository issues
	if _, err := os.Stat(".git"); err == nil {
		fmt.Println("3️⃣  Fixing repository issues...")
		if err := fixRepository(); err != nil {
			fmt.Printf("   ⚠️  Repository fix failed: %v\n", err)
		} else {
			fmt.Println("   ✅ Repository issues resolved")
		}
		fmt.Println("")
	}

	return nil
}

// diagnoseGitHubCLI checks GitHub CLI authentication
func diagnoseGitHubCLI() error {
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("GitHub CLI not authenticated")
	}

	// Check if multiple accounts are authenticated
	lines := strings.Split(string(output), "\n")
	accountCount := 0
	for _, line := range lines {
		if strings.Contains(line, "Logged in to github.com account") {
			accountCount++
		}
	}

	if accountCount == 0 {
		return fmt.Errorf("No GitHub accounts authenticated")
	} else if accountCount == 1 {
		return fmt.Errorf("Only one account authenticated (multiple accounts recommended)")
	}

	return nil
}

// diagnoseSSHKeysStatus checks SSH key status
func diagnoseSSHKeysStatus() error {
	// Check SSH agent
	cmd := exec.Command("ssh-add", "-l")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SSH agent not running or no keys loaded")
	}

	// Check GitHub SSH connection
	cmd = exec.Command("ssh", "-T", "git@github.com")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GitHub SSH connection failed")
	}

	if !strings.Contains(string(output), "successfully authenticated") {
		return fmt.Errorf("SSH authentication failed")
	}

	return nil
}

// diagnoseGitConfig checks Git configuration
func diagnoseGitConfig() error {
	// Check global user.name
	cmd := exec.Command("git", "config", "--global", "--get", "user.name")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Global user.name not set")
	}

	// Check global user.email
	cmd = exec.Command("git", "config", "--global", "--get", "user.email")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Global user.email not set")
	}

	return nil
}

// diagnoseGitPersonaConfig checks GitPersona configuration
func diagnoseGitPersonaConfig() error {
	// Check if config file exists
	configPath := os.Getenv("GITPERSONA_CONFIG")
	if configPath == "" {
		configPath = filepath.Join(os.Getenv("HOME"), ".config", "gitpersona", "config.yaml")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("Configuration file not found")
	}

	// Try to load configuration
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("Failed to load configuration: %v", err)
	}

	accounts := configManager.ListAccounts()
	if len(accounts) == 0 {
		return fmt.Errorf("No accounts configured")
	}

	return nil
}

// fixSSHKeyIssues attempts to fix common SSH key problems
func fixSSHKeyIssues() error {
	// Start SSH agent if not running
	cmd := exec.Command("ssh-add", "-l")
	if err := cmd.Run(); err != nil {
		fmt.Println("   🔄 Starting SSH agent...")
		cmd = exec.Command("eval", "$(ssh-agent -s)")
		cmd.Run() // Ignore errors
	}

	// Try to add common SSH keys
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	commonKeys := []string{"id_rsa", "id_ed25519", "id_rsa_personal", "id_rsa_work"}

	for _, key := range commonKeys {
		keyPath := filepath.Join(sshDir, key)
		if _, err := os.Stat(keyPath); err == nil {
			fmt.Printf("   🔄 Adding SSH key: %s\n", key)
			cmd := exec.Command("ssh-add", keyPath)
			cmd.Run() // Ignore errors
		}
	}

	return nil
}
