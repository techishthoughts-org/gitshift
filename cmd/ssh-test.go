package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
)

var sshTestCmd = &cobra.Command{
	Use:   "ssh-test [account-alias]",
	Short: "🧪 Test SSH connectivity for GitPersona accounts",
	Long: `Test SSH connectivity and troubleshoot common SSH issues.
This command helps with:
- Testing SSH key authentication to GitHub
- Verifying known_hosts configuration  
- Checking SSH key permissions
- Troubleshooting SSH agent issues
- Validating SSH configuration

If no account is specified, it tests the currently active account.`,
	Example: `  # Test current account
  gitpersona ssh-test

  # Test specific account
  gitpersona ssh-test costaar7
  
  # Test with verbose output
  gitpersona ssh-test costaar7 --verbose
  
  # Fix known_hosts issues
  gitpersona ssh-test --fix-known-hosts`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSSHTest,
}

var (
	verbose       bool
	fixKnownHosts bool
	testAll       bool
)

func init() {
	rootCmd.AddCommand(sshTestCmd)

	sshTestCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose SSH output")
	sshTestCmd.Flags().BoolVar(&fixKnownHosts, "fix-known-hosts", false, "Automatically fix known_hosts issues")
	sshTestCmd.Flags().BoolVar(&testAll, "all", false, "Test all configured accounts")
}

func runSSHTest(cmd *cobra.Command, args []string) error {
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if testAll {
		return testAllAccounts(configManager)
	}

	var accountAlias string
	if len(args) > 0 {
		accountAlias = args[0]
	} else {
		// Use current account
		cfg := configManager.GetConfig()
		if cfg == nil || cfg.CurrentAccount == "" {
			return fmt.Errorf("no current account set. Use 'gitpersona switch <account>' or specify account name")
		}
		accountAlias = cfg.CurrentAccount
	}

	account, err := configManager.GetAccount(accountAlias)
	if err != nil {
		return fmt.Errorf("account '%s' not found", accountAlias)
	}

	fmt.Printf("🧪 Testing SSH connectivity for account: %s\n", accountAlias)
	fmt.Printf("📧 Email: %s\n", account.Email)
	fmt.Printf("🔑 SSH Key: %s\n", account.SSHKeyPath)
	fmt.Printf("────────────────────────────────────────────────────\n")

	tester := &SSHTester{
		verbose:       verbose,
		fixKnownHosts: fixKnownHosts,
	}

	return tester.TestAccount(accountAlias, account)
}

func testAllAccounts(configManager *config.Manager) error {
	accounts := configManager.ListAccounts()
	if len(accounts) == 0 {
		fmt.Println("❌ No accounts configured")
		return nil
	}

	fmt.Printf("🧪 Testing SSH connectivity for %d account(s)\n", len(accounts))
	fmt.Printf("══════════════════════════════════════════════════════\n")

	tester := &SSHTester{
		verbose:       verbose,
		fixKnownHosts: fixKnownHosts,
	}

	allPassed := true
	for _, account := range accounts {
		fmt.Printf("\n📋 Account: %s\n", account.Alias)
		fmt.Printf("────────────────────────────────────────────────────\n")

		if err := tester.TestAccount(account.Alias, account); err != nil {
			allPassed = false
		}
	}

	fmt.Printf("\n══════════════════════════════════════════════════════\n")
	if allPassed {
		fmt.Printf("✅ All accounts passed SSH connectivity tests!\n")
	} else {
		fmt.Printf("⚠️  Some accounts failed SSH connectivity tests\n")
	}

	return nil
}

type SSHTester struct {
	verbose       bool
	fixKnownHosts bool
}

func (t *SSHTester) TestAccount(alias string, account *models.Account) error {
	var failed []string

	// 1. Check if SSH key exists
	if !t.testKeyExists(account.SSHKeyPath) {
		failed = append(failed, "ssh_key_missing")
	}

	// 2. Check SSH key permissions
	if !t.testKeyPermissions(account.SSHKeyPath) {
		failed = append(failed, "ssh_key_permissions")
	}

	// 3. Check known_hosts
	if !t.testKnownHosts() {
		failed = append(failed, "known_hosts")
	}

	// 4. Test SSH connectivity
	if !t.testGitHubConnection(account.SSHKeyPath) {
		failed = append(failed, "github_connection")
	}

	// 5. Test SSH agent
	if !t.testSSHAgent(account.SSHKeyPath) {
		failed = append(failed, "ssh_agent")
	}

	if len(failed) == 0 {
		fmt.Printf("✅ All SSH tests passed for account '%s'!\n", alias)
		return nil
	} else {
		fmt.Printf("❌ SSH tests failed for account '%s': %v\n", alias, failed)
		return fmt.Errorf("SSH tests failed: %v", failed)
	}
}

func (t *SSHTester) testKeyExists(keyPath string) bool {
	fmt.Printf("🔍 Checking SSH key existence...")

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Printf(" ❌ Private key not found: %s\n", keyPath)
		return false
	}

	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		fmt.Printf(" ❌ Public key not found: %s\n", pubKeyPath)
		return false
	}

	fmt.Printf(" ✅\n")
	return true
}

func (t *SSHTester) testKeyPermissions(keyPath string) bool {
	fmt.Printf("🔒 Checking SSH key permissions...")

	// Check private key permissions (should be 600)
	if info, err := os.Stat(keyPath); err == nil {
		perm := info.Mode().Perm()
		if perm != 0600 {
			fmt.Printf(" ❌ Private key has wrong permissions: %o (should be 600)\n", perm)

			// Try to fix permissions
			if err := os.Chmod(keyPath, 0600); err != nil {
				fmt.Printf("   ⚠️  Failed to fix permissions: %v\n", err)
				return false
			} else {
				fmt.Printf("   ✅ Fixed private key permissions\n")
			}
		}
	} else {
		fmt.Printf(" ❌ Cannot check private key permissions: %v\n", err)
		return false
	}

	// Check public key permissions (should be 644)
	pubKeyPath := keyPath + ".pub"
	if info, err := os.Stat(pubKeyPath); err == nil {
		perm := info.Mode().Perm()
		if perm != 0644 {
			fmt.Printf(" ⚠️  Public key has permissions: %o (recommended: 644)\n", perm)
			if err := os.Chmod(pubKeyPath, 0644); err == nil {
				fmt.Printf("   ✅ Fixed public key permissions\n")
			}
		}
	}

	fmt.Printf(" ✅\n")
	return true
}

func (t *SSHTester) testKnownHosts() bool {
	fmt.Printf("🌐 Checking known_hosts for GitHub...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf(" ❌ Cannot get home directory: %v\n", err)
		return false
	}

	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		if t.fixKnownHosts {
			fmt.Printf(" ⚠️  known_hosts not found, creating...\n")
			return t.fixGitHubKnownHosts(knownHostsPath)
		}
		fmt.Printf(" ❌ Cannot read known_hosts: %v\n", err)
		return false
	}

	contentStr := string(content)
	hasGitHub := strings.Contains(contentStr, "github.com")

	if !hasGitHub {
		if t.fixKnownHosts {
			fmt.Printf(" ⚠️  GitHub not in known_hosts, adding...\n")
			return t.fixGitHubKnownHosts(knownHostsPath)
		}
		fmt.Printf(" ❌ GitHub not found in known_hosts\n")
		return false
	}

	fmt.Printf(" ✅\n")
	return true
}

func (t *SSHTester) fixGitHubKnownHosts(knownHostsPath string) bool {
	keyManager := &SSHKeyManager{}
	if err := keyManager.SetupKnownHosts(); err != nil {
		fmt.Printf("   ❌ Failed to setup known_hosts: %v\n", err)
		return false
	}
	fmt.Printf("   ✅ Added GitHub to known_hosts\n")
	return true
}

func (t *SSHTester) testGitHubConnection(keyPath string) bool {
	fmt.Printf("🔗 Testing GitHub SSH connection...")

	args := []string{
		"-i", keyPath,
		"-o", "ConnectTimeout=10",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-T", "git@github.com",
	}

	if t.verbose {
		args = append([]string{"-v"}, args...)
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// SSH to GitHub should return exit code 1 with success message
	if strings.Contains(outputStr, "successfully authenticated") {
		if t.verbose {
			fmt.Printf(" ✅\n   Output: %s\n", outputStr)
		} else {
			fmt.Printf(" ✅\n")
		}
		return true
	}

	fmt.Printf(" ❌ Connection failed\n")
	if t.verbose {
		fmt.Printf("   Command: ssh %s\n", strings.Join(args, " "))
		fmt.Printf("   Output: %s\n", outputStr)
		fmt.Printf("   Error: %v\n", err)
	} else {
		// Show key troubleshooting info
		fmt.Printf("   💡 Try running with --verbose for more details\n")
		if strings.Contains(outputStr, "Permission denied") {
			fmt.Printf("   💡 Permission denied - check if key is added to GitHub\n")
		}
		if strings.Contains(outputStr, "Host key verification failed") {
			fmt.Printf("   💡 Host key issue - try --fix-known-hosts\n")
		}
	}

	return false
}

func (t *SSHTester) testSSHAgent(keyPath string) bool {
	fmt.Printf("🔐 Checking SSH agent...")

	// Check if ssh-agent is running
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		fmt.Printf(" ⚠️  SSH agent not detected (SSH_AUTH_SOCK not set)\n")
		return true // This is not critical
	}

	// List keys in agent
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf(" ⚠️  Cannot list SSH agent keys: %v\n", err)
		return true // Not critical
	}

	outputStr := string(output)

	// Get key fingerprint
	fingerprintCmd := exec.Command("ssh-keygen", "-lf", keyPath)
	fingerprintOutput, err := fingerprintCmd.Output()
	if err != nil {
		fmt.Printf(" ⚠️  Cannot get key fingerprint: %v\n", err)
		return true
	}

	fingerprint := strings.Fields(string(fingerprintOutput))
	if len(fingerprint) < 2 {
		fmt.Printf(" ⚠️  Cannot parse key fingerprint\n")
		return true
	}

	keyFingerprint := fingerprint[1] // SHA256:...

	if strings.Contains(outputStr, keyFingerprint) {
		fmt.Printf(" ✅ Key loaded in SSH agent\n")
	} else {
		fmt.Printf(" ⚠️  Key not loaded in SSH agent\n")
		fmt.Printf("   💡 Run: ssh-add %s\n", keyPath)
	}

	return true
}
