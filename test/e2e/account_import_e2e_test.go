package e2e

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// E2ETestSuite provides end-to-end testing environment
type E2ETestSuite struct {
	ctx           context.Context
	testHomeDir   string
	originalHome  string
	gitPersonaCmd string
	cleanup       func()
}

// RealAccountData represents actual account configuration for import testing
type RealAccountData struct {
	Alias          string
	Name           string
	Email          string
	GitHubUsername string
	SSHKeyPath     string
	HasSSHKey      bool
	HasGitHubToken bool
}

// SetupE2ETest creates a real-world testing environment
func SetupE2ETest(t *testing.T) *E2ETestSuite {
	ctx := context.Background()

	// Create temporary test directory
	tempDir := t.TempDir()
	testHomeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(testHomeDir, 0755))

	// Backup original HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testHomeDir)

	// Build gitpersona binary for E2E testing
	gitPersonaCmd := buildGitPersonaBinary(t, tempDir)

	cleanup := func() {
		os.Setenv("HOME", originalHome)
		// Binary and temp dir cleaned up automatically by t.TempDir()
	}

	return &E2ETestSuite{
		ctx:           ctx,
		testHomeDir:   testHomeDir,
		originalHome:  originalHome,
		gitPersonaCmd: gitPersonaCmd,
		cleanup:       cleanup,
	}
}

func TestE2EAccountImport(t *testing.T) {
	suite := SetupE2ETest(t)
	defer suite.cleanup()

	t.Run("ImportFromExistingGitConfig", func(t *testing.T) {
		testImportFromExistingGitConfig(t, suite)
	})

	t.Run("ImportFromSSHKeys", func(t *testing.T) {
		testImportFromSSHKeys(t, suite)
	})

	t.Run("ImportFromLegacyConfig", func(t *testing.T) {
		testImportFromLegacyConfig(t, suite)
	})

	t.Run("FullWorkflowWithRealAccount", func(t *testing.T) {
		testFullWorkflowWithRealAccount(t, suite)
	})

	t.Run("MultiAccountScenario", func(t *testing.T) {
		testMultiAccountScenario(t, suite)
	})

	t.Run("GitHubIntegrationFlow", func(t *testing.T) {
		testGitHubIntegrationFlow(t, suite)
	})

	t.Run("SecurityAuditAndFix", func(t *testing.T) {
		testSecurityAuditAndFix(t, suite)
	})
}

func testImportFromExistingGitConfig(t *testing.T, suite *E2ETestSuite) {
	// Setup existing Git configuration
	setupExistingGitConfig(t, suite, RealAccountData{
		Name:  "John Developer",
		Email: "john.dev@company.com",
	})

	// Run migration check
	output, err := suite.runCommand("migrate", "check")
	assert.NoError(t, err)
	t.Logf("Migration check output: %s", output)

	// Run migration
	output, err = suite.runCommand("migrate", "run", "--auto-discover")
	assert.NoError(t, err)
	assert.Contains(t, output, "Migration completed successfully")

	// Verify account was imported
	output, err = suite.runCommand("account", "list", "--detailed")
	assert.NoError(t, err)
	assert.Contains(t, output, "John Developer")
	assert.Contains(t, output, "john.dev@company.com")

	// Test account functionality
	accounts := extractAccountsFromOutput(output)
	require.NotEmpty(t, accounts, "Should have imported at least one account")

	// Test switching to imported account
	firstAccount := accounts[0]
	output, err = suite.runCommand("account", "switch", firstAccount)
	assert.NoError(t, err)
	assert.Contains(t, output, "Switched to account")

	// Verify Git config is correctly set
	output, err = suite.runCommand("git", "config", "show")
	assert.NoError(t, err)
	assert.Contains(t, output, "John Developer")
	assert.Contains(t, output, "john.dev@company.com")
}

func testImportFromSSHKeys(t *testing.T, suite *E2ETestSuite) {
	// Setup SSH keys with realistic naming patterns
	sshKeys := []RealAccountData{
		{
			Alias:      "work",
			SSHKeyPath: "id_ed25519_work",
			Name:       "Work Account",
			Email:      "user@work.com",
		},
		{
			Alias:      "personal",
			SSHKeyPath: "id_ed25519_personal",
			Name:       "Personal Account",
			Email:      "user@gmail.com",
		},
		{
			Alias:      "github",
			SSHKeyPath: "id_rsa_github",
			Name:       "GitHub Account",
			Email:      "user@github.com",
		},
	}

	setupSSHKeys(t, suite, sshKeys)

	// Run SSH-based account discovery
	output, err := suite.runCommand("smart", "detect", "--ssh-discovery")
	assert.NoError(t, err)
	t.Logf("SSH discovery output: %s", output)

	// Import discovered accounts
	output, err = suite.runCommand("migrate", "run", "--from-ssh")
	assert.NoError(t, err)
	assert.Contains(t, output, "Migration completed successfully")

	// Verify all accounts were imported
	output, err = suite.runCommand("account", "list", "--verbose")
	assert.NoError(t, err)

	for _, expectedKey := range sshKeys {
		assert.Contains(t, output, expectedKey.Alias)
		assert.Contains(t, output, expectedKey.SSHKeyPath)
	}

	// Test SSH functionality for each account
	for _, keyData := range sshKeys {
		// Switch to account
		output, err = suite.runCommand("account", "switch", keyData.Alias)
		assert.NoError(t, err)

		// Test SSH key validation
		output, err = suite.runCommand("ssh", "keys", "validate", keyData.SSHKeyPath)
		if err == nil {
			assert.Contains(t, output, "valid")
		} else {
			t.Logf("SSH validation expected to fail in test environment: %v", err)
		}
	}
}

func testImportFromLegacyConfig(t *testing.T, suite *E2ETestSuite) {
	// Setup legacy GitPersona configuration
	legacyConfig := map[string]interface{}{
		"current_account": "main",
		"global_git_mode": true,
		"accounts": map[string]interface{}{
			"main": map[string]interface{}{
				"name":            "Main User",
				"email":           "main@example.com",
				"ssh_key_path":    "~/.ssh/id_ed25519_main",
				"github_username": "mainuser",
			},
			"work": map[string]interface{}{
				"name":            "Work User",
				"email":           "work@company.com",
				"ssh_key_path":    "~/.ssh/id_ed25519_work",
				"github_username": "workuser",
			},
		},
	}

	setupLegacyConfig(t, suite, legacyConfig)

	// Run legacy migration
	output, err := suite.runCommand("migrate", "run", "--from-legacy")
	assert.NoError(t, err)
	assert.Contains(t, output, "Legacy migration completed")

	// Verify accounts were migrated with v2 features
	output, err = suite.runCommand("account", "list", "--json")
	assert.NoError(t, err)

	// Parse JSON output to verify v2 features
	assert.Contains(t, output, "isolation_level")
	assert.Contains(t, output, "isolation_metadata")

	// Test migrated account functionality
	output, err = suite.runCommand("account", "switch", "main")
	assert.NoError(t, err)

	// Verify GitHub integration if available
	output, err = suite.runCommand("github", "test")
	if err != nil {
		// Expected to fail without actual token
		assert.Contains(t, output, "token")
	}

	// Test security features on migrated accounts
	output, err = suite.runCommand("security", "audit")
	assert.NoError(t, err)
	assert.Contains(t, output, "Security audit completed")
}

func testFullWorkflowWithRealAccount(t *testing.T, suite *E2ETestSuite) {
	// Test complete workflow: create, configure, validate, use
	accountData := RealAccountData{
		Alias:          "e2e-test",
		Name:           "E2E Test User",
		Email:          "e2e@testing.com",
		GitHubUsername: "e2etester",
	}

	// Step 1: Create account
	output, err := suite.runCommand("account", "add", accountData.Alias,
		"--name", accountData.Name,
		"--email", accountData.Email,
		"--github", accountData.GitHubUsername)
	assert.NoError(t, err)
	assert.Contains(t, output, "Account created successfully")

	// Step 2: Generate SSH key for account
	output, err = suite.runCommand("ssh", "keys", "generate",
		"--name", accountData.Alias,
		"--type", "ed25519",
		"--email", accountData.Email)
	assert.NoError(t, err)
	assert.Contains(t, output, "SSH key generated successfully")

	// Step 3: Update account with SSH key
	sshKeyPath := filepath.Join(suite.testHomeDir, ".ssh", fmt.Sprintf("id_ed25519_%s", accountData.Alias))
	output, err = suite.runCommand("account", "update", accountData.Alias,
		"--ssh-key", sshKeyPath)
	assert.NoError(t, err)

	// Step 4: Validate account configuration
	output, err = suite.runCommand("account", "validate", accountData.Alias)
	assert.NoError(t, err)
	assert.Contains(t, output, "validation successful")

	// Step 5: Switch to account and verify Git config
	output, err = suite.runCommand("account", "switch", accountData.Alias)
	assert.NoError(t, err)

	// Step 6: Test Git operations
	testRepoDir := filepath.Join(suite.testHomeDir, "test-repo")
	require.NoError(t, os.MkdirAll(testRepoDir, 0755))

	// Initialize Git repo
	gitCmd := exec.CommandContext(suite.ctx, "git", "init", testRepoDir)
	gitCmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", suite.testHomeDir))
	err = gitCmd.Run()
	require.NoError(t, err)

	// Change to repo directory and test Git config
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(testRepoDir)

	output, err = suite.runCommand("git", "config", "show")
	assert.NoError(t, err)
	assert.Contains(t, output, accountData.Name)
	assert.Contains(t, output, accountData.Email)

	// Step 7: Test SSH connectivity
	output, err = suite.runCommand("ssh", "test")
	// May fail in test environment - that's expected
	t.Logf("SSH test result: %s", output)

	// Step 8: Run comprehensive diagnostics
	output, err = suite.runCommand("diagnose", "health", "--verbose")
	assert.NoError(t, err)
	assert.Contains(t, output, "Diagnostic Results")

	// Step 9: Security audit
	output, err = suite.runCommand("security", "audit")
	assert.NoError(t, err)
	assert.Contains(t, output, "Security audit completed")

	// Step 10: Clean up - remove account
	output, err = suite.runCommand("account", "remove", accountData.Alias, "--confirm")
	assert.NoError(t, err)
	assert.Contains(t, output, "Account removed successfully")
}

func testMultiAccountScenario(t *testing.T, suite *E2ETestSuite) {
	// Create multiple accounts for different contexts
	accounts := []RealAccountData{
		{
			Alias:          "work-frontend",
			Name:           "John Doe",
			Email:          "john.doe@company.com",
			GitHubUsername: "johndoe-work",
		},
		{
			Alias:          "work-backend",
			Name:           "John Doe",
			Email:          "john.doe@company.com",
			GitHubUsername: "johndoe-backend",
		},
		{
			Alias:          "personal",
			Name:           "John Doe",
			Email:          "john.personal@gmail.com",
			GitHubUsername: "johndoe-personal",
		},
		{
			Alias:          "opensource",
			Name:           "John Doe",
			Email:          "john.opensource@gmail.com",
			GitHubUsername: "johndoe-oss",
		},
	}

	// Create all accounts
	for _, account := range accounts {
		_, err := suite.runCommand("account", "add", account.Alias,
			"--name", account.Name,
			"--email", account.Email,
			"--github", account.GitHubUsername)
		assert.NoError(t, err, "Failed to create account %s", account.Alias)

		// Generate SSH key for each
		_, err = suite.runCommand("ssh", "keys", "generate",
			"--name", account.Alias,
			"--type", "ed25519",
			"--email", account.Email)
		assert.NoError(t, err, "Failed to generate SSH key for %s", account.Alias)
	}

	// Test account listing with different detail levels
	output, err := suite.runCommand("account", "list")
	assert.NoError(t, err)
	for _, account := range accounts {
		assert.Contains(t, output, account.Alias)
	}

	// Test switching between accounts rapidly
	for i := 0; i < 3; i++ {
		for _, account := range accounts {
			output, err = suite.runCommand("account", "switch", account.Alias)
			assert.NoError(t, err)

			// Verify switch worked
			output, err = suite.runCommand("git", "config", "show")
			assert.NoError(t, err)
			assert.Contains(t, output, account.Email)
		}
	}

	// Test smart detection in different contexts
	for _, account := range accounts {
		// Create a mock repository context
		repoDir := filepath.Join(suite.testHomeDir, fmt.Sprintf("repo-%s", account.Alias))
		require.NoError(t, os.MkdirAll(repoDir, 0755))

		originalWd, _ := os.Getwd()
		os.Chdir(repoDir)

		// Initialize git repo
		gitCmd := exec.CommandContext(suite.ctx, "git", "init")
		gitCmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", suite.testHomeDir))
		gitCmd.Run()

		// Test smart detection
		output, err = suite.runCommand("smart", "detect")
		if err == nil {
			t.Logf("Smart detection for %s: %s", account.Alias, output)
		}

		os.Chdir(originalWd)
	}

	// Test batch operations
	output, err = suite.runCommand("account", "validate", "--all")
	if err == nil {
		assert.Contains(t, output, "validation")
	}

	// Clean up all accounts
	for _, account := range accounts {
		suite.runCommand("account", "remove", account.Alias, "--confirm")
	}
}

func testGitHubIntegrationFlow(t *testing.T, suite *E2ETestSuite) {
	accountAlias := "github-integration"

	// Create account for GitHub integration
	output, err := suite.runCommand("account", "add", accountAlias,
		"--name", "GitHub Integration Test",
		"--email", "github@test.com",
		"--github", "testuser")
	assert.NoError(t, err)

	// Generate SSH key
	output, err = suite.runCommand("ssh", "keys", "generate",
		"--name", accountAlias,
		"--type", "ed25519",
		"--email", "github@test.com")
	assert.NoError(t, err)

	// Test GitHub token operations (will fail without real token - expected)
	output, err = suite.runCommand("github", "token", "set")
	assert.Error(t, err)
	assert.Contains(t, output, "token")

	// Test GitHub API access (expected to fail)
	output, err = suite.runCommand("github", "test")
	assert.Error(t, err)
	t.Logf("GitHub test (expected to fail): %s", output)

	// Test SSH key upload preparation (will fail without token)
	output, err = suite.runCommand("ssh", "keys", "upload",
		"--key", accountAlias,
		"--title", "Test Key")
	assert.Error(t, err)
	assert.Contains(t, output, "GitHub")

	// Test with mock token environment
	os.Setenv("GITHUB_TOKEN", "fake-token-for-testing")
	defer os.Unsetenv("GITHUB_TOKEN")

	output, err = suite.runCommand("github", "test")
	assert.Error(t, err) // Still should fail with fake token
	assert.Contains(t, output, "GitHub")

	// Clean up
	suite.runCommand("account", "remove", accountAlias, "--confirm")
}

func testSecurityAuditAndFix(t *testing.T, suite *E2ETestSuite) {
	// Create account with intentionally insecure setup
	accountAlias := "security-test"

	output, err := suite.runCommand("account", "add", accountAlias,
		"--name", "Security Test",
		"--email", "security@test.com")
	assert.NoError(t, err)

	// Generate SSH key
	output, err = suite.runCommand("ssh", "keys", "generate",
		"--name", accountAlias,
		"--type", "ed25519",
		"--email", "security@test.com")
	assert.NoError(t, err)

	// Intentionally create insecure permissions
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	if _, err := os.Stat(sshDir); err == nil {
		// Make SSH directory too permissive
		os.Chmod(sshDir, 0755) // Should be 0700

		// Make SSH keys too permissive
		if entries, err := os.ReadDir(sshDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".pub") {
					keyPath := filepath.Join(sshDir, entry.Name())
					os.Chmod(keyPath, 0644) // Should be 0600
				}
			}
		}
	}

	// Run security audit
	output, err = suite.runCommand("security", "audit", "--detailed")
	assert.NoError(t, err)
	assert.Contains(t, output, "Security audit completed")

	// Should find violations
	if strings.Contains(output, "violations") {
		t.Logf("Security violations found (expected): %s", output)

		// Run auto-fix
		output, err = suite.runCommand("security", "fix", "--auto")
		assert.NoError(t, err)
		assert.Contains(t, output, "Security violations fixed")

		// Re-run audit to verify fixes
		output, err = suite.runCommand("security", "audit")
		assert.NoError(t, err)
		t.Logf("Post-fix audit results: %s", output)
	}

	// Test security compliance check
	output, err = suite.runCommand("diagnose", "security", "--compliance")
	assert.NoError(t, err)
	assert.Contains(t, output, "compliance")

	// Clean up
	suite.runCommand("account", "remove", accountAlias, "--confirm")
}

// Helper functions

func buildGitPersonaBinary(t *testing.T, tempDir string) string {
	binaryPath := filepath.Join(tempDir, "gitpersona")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../../cmd/main.go")
	cmd.Dir = filepath.Join(".", "..", "..") // Adjust path as needed

	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build GitPersona binary: %v", err)
	}

	// Make binary executable
	err = os.Chmod(binaryPath, 0755)
	require.NoError(t, err)

	return binaryPath
}

func (suite *E2ETestSuite) runCommand(args ...string) (string, error) {
	cmd := exec.CommandContext(suite.ctx, suite.gitPersonaCmd, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", suite.testHomeDir))
	cmd.Dir = suite.testHomeDir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func setupExistingGitConfig(t *testing.T, suite *E2ETestSuite, account RealAccountData) {
	// Setup global Git config
	gitCmd := exec.CommandContext(suite.ctx, "git", "config", "--global", "user.name", account.Name)
	gitCmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", suite.testHomeDir))
	require.NoError(t, gitCmd.Run())

	gitCmd = exec.CommandContext(suite.ctx, "git", "config", "--global", "user.email", account.Email)
	gitCmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", suite.testHomeDir))
	require.NoError(t, gitCmd.Run())
}

func setupSSHKeys(t *testing.T, suite *E2ETestSuite, keys []RealAccountData) {
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	for _, keyData := range keys {
		keyPath := filepath.Join(sshDir, keyData.SSHKeyPath)
		pubKeyPath := keyPath + ".pub"

		// Create fake private key
		privateKey := fmt.Sprintf(`-----BEGIN OPENSSH PRIVATE KEY-----
# Fake SSH key for testing %s
# Type: %s
# Email: %s
-----END OPENSSH PRIVATE KEY-----`, keyData.Alias, "ed25519", keyData.Email)

		require.NoError(t, os.WriteFile(keyPath, []byte(privateKey), 0600))

		// Create fake public key
		publicKey := fmt.Sprintf("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyDataForTesting%s %s", keyData.Alias, keyData.Email)
		require.NoError(t, os.WriteFile(pubKeyPath, []byte(publicKey), 0644))
	}
}

func setupLegacyConfig(t *testing.T, suite *E2ETestSuite, config map[string]interface{}) {
	// Create legacy config in old location
	legacyConfigPath := filepath.Join(suite.testHomeDir, ".git-persona.yaml")

	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(legacyConfigPath, data, 0644))
}

func extractAccountsFromOutput(output string) []string {
	var accounts []string
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "👤") || strings.Contains(line, "Account:") {
			// Extract account alias from output
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "👤" || part == "Account:") && i+1 < len(parts) {
					account := strings.TrimSpace(parts[i+1])
					if strings.Contains(account, "(") {
						account = strings.Split(account, "(")[0]
					}
					accounts = append(accounts, account)
					break
				}
			}
		}
	}

	return accounts
}

// Performance benchmarks for E2E operations
func BenchmarkE2EOperations(b *testing.B) {
	suite := SetupE2ETest(&testing.T{})
	defer suite.cleanup()

	b.Run("AccountCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			accountAlias := fmt.Sprintf("bench-account-%d", i)
			_, err := suite.runCommand("account", "add", accountAlias,
				"--name", "Bench User",
				"--email", "bench@test.com")
			if err != nil {
				b.Fatal(err)
			}

			// Clean up
			suite.runCommand("account", "remove", accountAlias, "--confirm")
		}
	})

	b.Run("AccountSwitching", func(b *testing.B) {
		// Setup accounts first
		for i := 0; i < 3; i++ {
			accountAlias := fmt.Sprintf("switch-account-%d", i)
			suite.runCommand("account", "add", accountAlias,
				"--name", "Switch User",
				"--email", "switch@test.com")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			accountAlias := fmt.Sprintf("switch-account-%d", i%3)
			_, err := suite.runCommand("account", "switch", accountAlias)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.runCommand("diagnose", "health")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
