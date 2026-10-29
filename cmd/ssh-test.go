package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
	"github.com/techishthoughts/gitshift/internal/ssh"
)

var sshTestCmd = &cobra.Command{
	Use:   "ssh-test [account-alias]",
	Short: "🧪 Test SSH connectivity for Git platform accounts",
	Long: `Test SSH connectivity and troubleshoot common SSH issues.
This command helps with:
- Testing SSH key authentication to Git platforms (GitHub, GitLab, etc.)
- Verifying known_hosts configuration
- Checking SSH key permissions
- Troubleshooting SSH agent issues
- Validating SSH configuration

Works with all supported platforms:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)

If no account is specified, it tests the currently active account.`,
	Example: `  # Test current account
  gitshift ssh-test

  # Test GitHub account
  gitshift ssh-test work-github

  # Test GitLab account
  gitshift ssh-test work-gitlab

  # Test with verbose output
  gitshift ssh-test work-github --verbose

  # Test all accounts
  gitshift ssh-test --all`,
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
	sshTestCmd.Flags().BoolVar(&fixKnownHosts, "fix-known-hosts", false, "Deprecated: ssh-test no longer modifies known_hosts")
	_ = sshTestCmd.Flags().MarkHidden("fix-known-hosts")
	sshTestCmd.Flags().BoolVar(&testAll, "all", false, "Test all configured accounts")
}

func runSSHTest(cmd *cobra.Command, args []string) error {
	if fixKnownHosts {
		return fmt.Errorf("--fix-known-hosts is no longer supported; ssh-test is read-only")
	}
	configManager := newConfigManager()
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
			return fmt.Errorf("no current account set. Use 'gitshift switch <account>' or specify account name")
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

	var failures []error
	for _, account := range accounts {
		fmt.Printf("\n📋 Account: %s\n", account.Alias)
		fmt.Printf("────────────────────────────────────────────────────\n")

		if err := tester.TestAccount(account.Alias, account); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", account.Alias, err))
		}
	}

	fmt.Printf("\n══════════════════════════════════════════════════════\n")
	if len(failures) == 0 {
		fmt.Printf("✅ All accounts passed SSH connectivity tests!\n")
	} else {
		fmt.Printf("⚠️  Some accounts failed SSH connectivity tests\n")
		return errors.Join(failures...)
	}

	return nil
}

type SSHTester struct {
	verbose       bool
	fixKnownHosts bool
}

func (t *SSHTester) TestAccount(alias string, account *models.Account) error {
	var failed []string
	keyPath, err := models.NormalizeSSHKeyPath(account.SSHKeyPath)
	if err != nil {
		return err
	}

	// 1. Check if SSH key exists
	if !t.testKeyExists(keyPath) {
		failed = append(failed, "ssh_key_missing")
	}

	// 2. Check SSH key permissions
	if !t.testKeyPermissions(keyPath) {
		failed = append(failed, "ssh_key_permissions")
	}

	// 3. Check known_hosts
	if !t.testKnownHosts(account.GetDomain()) {
		failed = append(failed, "known_hosts")
	}

	// 4. Test SSH connectivity
	if !t.testPlatformConnection(keyPath, account.GetDomain()) {
		failed = append(failed, "platform_connection")
	}

	// 5. Test SSH agent
	if !t.testSSHAgent(keyPath) {
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

	if err := ssh.CheckPrivateKeyFile(keyPath); err != nil {
		fmt.Printf(" ❌ %v\n", err)
		return false
	}

	// Check public key permissions (should be 644)
	pubKeyPath := keyPath + ".pub"
	if info, err := os.Stat(pubKeyPath); err == nil {
		perm := info.Mode().Perm()
		if perm != 0644 {
			fmt.Printf(" ⚠️  Public key has permissions: %o (recommended: 644)\n", perm)
		}
	}

	fmt.Printf(" ✅\n")
	return true
}

func (t *SSHTester) testKnownHosts(domain string) bool {
	fmt.Printf("🌐 Checking known_hosts for %s...", domain)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf(" ❌ Cannot get home directory: %v\n", err)
		return false
	}

	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHostsPath); err != nil {
		fmt.Printf(" ❌ Cannot read known_hosts: %v\n", err)
		return false
	}

	if !knownHostsContains(knownHostsPath, domain) {
		fmt.Printf(" ❌ %s not found in known_hosts\n", domain)
		return false
	}

	fmt.Printf(" ✅\n")
	return true
}

// knownHostsContains asks ssh-keygen, which understands hashed entries
// (HashKnownHosts yes) as well as plain ones.
func knownHostsContains(knownHostsPath, domain string) bool {
	output, err := exec.Command("ssh-keygen", "-F", domain, "-f", knownHostsPath).Output()
	return err == nil && len(strings.TrimSpace(string(output))) > 0
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

func (t *SSHTester) testPlatformConnection(keyPath, domain string) bool {
	fmt.Printf("🔗 Testing %s SSH connection...", domain)

	args := []string{
		"-F", os.DevNull,
		"-i", keyPath,
		"-o", "ConnectTimeout=10",
		"-o", "IdentitiesOnly=yes",
		"-o", "IdentityAgent=none",
		"-o", "StrictHostKeyChecking=yes",
		"-T", "git@" + domain,
	}

	if t.verbose {
		args = append([]string{"-v"}, args...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// SSH to GitHub should return exit code 1 with success message
	if err == nil || strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Welcome to GitLab") {
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
			fmt.Printf("   💡 Verify the provider fingerprint, then update ~/.ssh/known_hosts with ssh-keygen/ssh-keyscan\n")
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
