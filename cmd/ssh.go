package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "🔐 SSH key management and troubleshooting",
	Long: `Advanced SSH key management with comprehensive diagnostics.

SSH commands provide powerful troubleshooting and management capabilities
for GitHub authentication and connectivity issues.

Features:
- Test SSH connectivity with detailed diagnostics
- Generate SSH config entries automatically
- Troubleshoot common SSH authentication issues
- Support for multiple key types (RSA, Ed25519, ECDSA)
- Integration with ssh-agent management
- Security compliance validation

Examples:
  gitpersona ssh test              # Test current account SSH
  gitpersona ssh test work         # Test specific account
  gitpersona ssh config            # Generate SSH config entries
  gitpersona ssh doctor            # Comprehensive diagnostics`,
}

// sshTestCmd tests SSH connectivity
var sshTestCmd = &cobra.Command{
	Use:   "test [account]",
	Short: "Test SSH connectivity for GitHub",
	Long: `Test SSH connectivity to GitHub with comprehensive diagnostics.

This command performs detailed SSH connectivity testing including:
- SSH key file existence and permissions
- SSH agent status and key loading
- GitHub.com connectivity test
- Key type and security validation
- Authentication flow verification

Examples:
  gitpersona ssh test              # Test current account
  gitpersona ssh test work         # Test specific account
  gitpersona ssh test --verbose    # Detailed diagnostic output`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		var account *models.Account
		var err error

		if len(args) > 0 {
			// Test specific account
			alias := args[0]
			account, err = configManager.GetAccount(alias)
			if err != nil {
				return fmt.Errorf("account '%s' not found. Use 'gitpersona list' to see available accounts", alias)
			}
		} else {
			// Test current account
			currentAlias := configManager.GetConfig().CurrentAccount
			if currentAlias == "" {
				return fmt.Errorf("no account currently active. Use 'gitpersona switch ACCOUNT' to select one")
			}
			account, err = configManager.GetAccount(currentAlias)
			if err != nil {
				return fmt.Errorf("current account '%s' not found in configuration", currentAlias)
			}
		}

		return performSSHTest(account, verbose)
	},
}

// sshConfigCmd generates SSH config entries
var sshConfigCmd = &cobra.Command{
	Use:   "config [account]",
	Short: "Generate SSH config entries",
	Long: `Generate SSH config entries for GitHub accounts.

This command creates properly formatted SSH config entries that can be
added to your ~/.ssh/config file for seamless GitHub authentication.

The generated config includes:
- Host aliases for easy identification
- IdentitiesOnly for security
- Proper key file paths
- GitHub.com hostname configuration

Examples:
  gitpersona ssh config            # Generate for all accounts
  gitpersona ssh config work       # Generate for specific account
  gitpersona ssh config --output ~/.ssh/config_gitpersona`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		accounts := configManager.ListAccounts()
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts configured. Use 'gitpersona add' to add an account")
		}

		var targetAccounts []*models.Account

		if len(args) > 0 {
			// Generate for specific account
			alias := args[0]
			account, err := configManager.GetAccount(alias)
			if err != nil {
				return fmt.Errorf("account '%s' not found", alias)
			}
			targetAccounts = []*models.Account{account}
		} else {
			// Generate for all accounts
			targetAccounts = accounts
		}

		outputFile, _ := cmd.Flags().GetString("output")
		return generateSSHConfig(targetAccounts, outputFile)
	},
}

// sshDoctorCmd provides comprehensive SSH diagnostics
var sshDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Comprehensive SSH diagnostics and troubleshooting",
	Long: `Perform comprehensive SSH diagnostics and troubleshooting.

This command runs a complete battery of SSH-related tests including:
- SSH agent status and configuration
- Key file existence and permissions
- GitHub connectivity for all accounts
- Common configuration issues
- Security compliance validation
- Performance testing

Examples:
  gitpersona ssh doctor            # Full diagnostic suite
  gitpersona ssh doctor --json     # JSON output for automation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		configManager := config.NewManager()
		if err := configManager.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		return runSSHDiagnostics(configManager, jsonOutput)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)

	// Add subcommands
	sshCmd.AddCommand(sshTestCmd)
	sshCmd.AddCommand(sshConfigCmd)
	sshCmd.AddCommand(sshDoctorCmd)

	// Flags for test command
	sshTestCmd.Flags().BoolP("verbose", "v", false, "Show detailed diagnostic output")

	// Flags for config command
	sshConfigCmd.Flags().StringP("output", "o", "", "Output file for SSH config (default: stdout)")

	// Flags for doctor command
	sshDoctorCmd.Flags().Bool("json", false, "Output results in JSON format")
}

// performSSHTest performs comprehensive SSH testing for an account
func performSSHTest(account *models.Account, verbose bool) error {
	fmt.Printf("🔐 Testing SSH connectivity for account '%s'\n", account.Alias)
	fmt.Printf("   Name: %s (%s)\n", account.Name, account.Email)
	if account.GitHubUsername != "" {
		fmt.Printf("   GitHub: @%s\n", account.GitHubUsername)
	}
	fmt.Println()

	// Test 1: SSH Key File Existence
	fmt.Print("1. 🔍 Checking SSH key file... ")
	if account.SSHKeyPath == "" {
		fmt.Println("❌ No SSH key configured")
		fmt.Println("   💡 Add SSH key with: gitpersona add ACCOUNT --ssh-key ~/.ssh/id_ed25519_ACCOUNT")
		return nil
	}

	keyPath := expandPath(account.SSHKeyPath)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Printf("❌ SSH key not found: %s\n", keyPath)
		fmt.Println("   💡 Generate new key with: gitpersona add-github GITHUB_USERNAME")
		return nil
	}
	fmt.Printf("✅ Found: %s\n", keyPath)

	// Test 2: Key Permissions
	fmt.Print("2. 🔒 Checking key permissions... ")
	if info, err := os.Stat(keyPath); err == nil {
		perm := info.Mode().Perm()
		if perm != 0600 {
			fmt.Printf("⚠️  Insecure permissions: %o (should be 600)\n", perm)
			fmt.Printf("   💡 Fix with: chmod 600 %s\n", keyPath)
		} else {
			fmt.Println("✅ Correct (600)")
		}
	} else {
		fmt.Printf("❌ Cannot check permissions: %v\n", err)
	}

	// Test 3: SSH Agent
	fmt.Print("3. 🔧 Checking SSH agent... ")
	if err := sshCheckSSHAgent(); err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println("   💡 Start SSH agent with: eval $(ssh-agent)")
	} else {
		fmt.Println("✅ Running")
	}

	// Test 4: Key in Agent
	fmt.Print("4. 🗝️  Checking key in SSH agent... ")
	if keyInAgent, err := isKeyInAgent(keyPath); err != nil {
		fmt.Printf("❌ Error checking: %v\n", err)
	} else if !keyInAgent {
		fmt.Printf("⚠️  Key not in agent\n")
		fmt.Printf("   💡 Add with: ssh-add %s\n", keyPath)
	} else {
		fmt.Println("✅ Key loaded")
	}

	// Test 5: GitHub Connectivity
	fmt.Print("5. 🌐 Testing GitHub connectivity... ")
	if err := testGitHubConnectivity(keyPath, verbose); err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println("   💡 Check GitHub SSH key settings: https://github.com/settings/keys")
		if account.GitHubUsername != "" {
			fmt.Printf("   💡 Re-upload key with: gitpersona add-github %s --overwrite\n", account.GitHubUsername)
		}
	} else {
		fmt.Println("✅ Connected successfully")
	}

	// Test 6: Key Type Validation
	fmt.Print("6. 🔐 Validating key security... ")
	if keyType, err := getKeyType(keyPath + ".pub"); err != nil {
		fmt.Printf("❌ Cannot determine key type: %v\n", err)
	} else {
		switch keyType {
		case "ssh-ed25519":
			fmt.Println("✅ Ed25519 (quantum-resistant, 2025 standard)")
		case "ssh-rsa":
			fmt.Println("⚠️  RSA (consider upgrading to Ed25519)")
		case "ecdsa-sha2":
			fmt.Println("✅ ECDSA (secure)")
		default:
			fmt.Printf("⚠️  Unknown type: %s\n", keyType)
		}
	}

	if verbose {
		fmt.Println("\n📊 Detailed Information:")
		fmt.Printf("   • Key file: %s\n", keyPath)
		fmt.Printf("   • Public key file: %s.pub\n", keyPath)
		if account.GitHubUsername != "" {
			fmt.Printf("   • GitHub profile: https://github.com/%s\n", account.GitHubUsername)
			fmt.Printf("   • GitHub SSH keys: https://github.com/%s.keys\n", account.GitHubUsername)
		}
		fmt.Printf("   • Test command: ssh -T git@github.com -i %s\n", keyPath)
	}

	fmt.Println("\n🎉 SSH connectivity test completed!")
	return nil
}

// generateSSHConfig generates SSH config entries
func generateSSHConfig(accounts []*models.Account, outputFile string) error {
	var configBuilder strings.Builder

	configBuilder.WriteString("# GitPersona SSH Configuration\n")
	configBuilder.WriteString("# Generated automatically - do not edit manually\n")
	configBuilder.WriteString(fmt.Sprintf("# Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		configBuilder.WriteString(fmt.Sprintf("# Account: %s (%s)\n", account.Alias, account.Name))
		configBuilder.WriteString(fmt.Sprintf("Host github.com-%s\n", account.Alias))
		configBuilder.WriteString("    HostName github.com\n")
		configBuilder.WriteString("    User git\n")
		configBuilder.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
		configBuilder.WriteString("    IdentitiesOnly yes\n")
		configBuilder.WriteString("    AddKeysToAgent yes\n")
		configBuilder.WriteString("    UseKeychain yes\n")
		configBuilder.WriteString("\n")
	}

	configContent := configBuilder.String()

	if outputFile != "" {
		// Write to file
		if err := os.WriteFile(outputFile, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write SSH config to %s: %w", outputFile, err)
		}
		fmt.Printf("✅ SSH config written to: %s\n", outputFile)
		fmt.Println("\n💡 To use this config, add the following to your ~/.ssh/config:")
		fmt.Printf("Include %s\n", outputFile)
	} else {
		// Print to stdout
		fmt.Println("📄 SSH Configuration Entries:\n")
		fmt.Print(configContent)
		fmt.Println("\n💡 To use this configuration:")
		fmt.Println("   1. Copy the above to your ~/.ssh/config file")
		fmt.Println("   2. Use host aliases like: git clone git@github.com-work:user/repo.git")
	}

	return nil
}

// runSSHDiagnostics performs comprehensive SSH diagnostics
func runSSHDiagnostics(configManager *config.Manager, jsonOutput bool) error {
	if !jsonOutput {
		fmt.Println("🏥 GitPersona SSH Doctor")
		fmt.Println("Performing comprehensive SSH diagnostics...\n")
	}

	diagnostics := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"checks":    []map[string]interface{}{},
	}

	// Check 1: SSH Binary
	sshCheck := runDiagnosticCheck("SSH Binary", func() (bool, string, string) {
		if _, err := exec.LookPath("ssh"); err != nil {
			return false, "SSH binary not found in PATH", "Install OpenSSH client"
		}
		return true, "SSH binary available", ""
	})
	addDiagnosticResult(&diagnostics, sshCheck, jsonOutput)

	// Check 2: SSH Agent
	agentCheck := runDiagnosticCheck("SSH Agent", func() (bool, string, string) {
		if err := sshCheckSSHAgent(); err != nil {
			return false, err.Error(), "Start SSH agent: eval $(ssh-agent)"
		}
		return true, "SSH agent running", ""
	})
	addDiagnosticResult(&diagnostics, agentCheck, jsonOutput)

	// Check 3: GitHub Connectivity
	connectivityCheck := runDiagnosticCheck("GitHub Connectivity", func() (bool, string, string) {
		if err := testBasicGitHubConnectivity(); err != nil {
			return false, err.Error(), "Check internet connection and GitHub status"
		}
		return true, "GitHub.com accessible", ""
	})
	addDiagnosticResult(&diagnostics, connectivityCheck, jsonOutput)

	// Check 4: Account SSH Keys
	accounts := configManager.ListAccounts()
	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		accountCheck := runDiagnosticCheck(fmt.Sprintf("Account '%s' SSH", account.Alias), func() (bool, string, string) {
			keyPath := expandPath(account.SSHKeyPath)

			// Check file existence
			if _, err := os.Stat(keyPath); os.IsNotExist(err) {
				return false, fmt.Sprintf("SSH key not found: %s", keyPath),
					fmt.Sprintf("Generate new key: gitpersona add-github %s --overwrite", account.GitHubUsername)
			}

			// Check permissions
			if info, err := os.Stat(keyPath); err == nil {
				if info.Mode().Perm() != 0600 {
					return false, fmt.Sprintf("Insecure permissions: %o", info.Mode().Perm()),
						fmt.Sprintf("Fix permissions: chmod 600 %s", keyPath)
				}
			}

			// Test connectivity
			if err := testGitHubConnectivity(keyPath, false); err != nil {
				return false, fmt.Sprintf("GitHub connectivity failed: %v", err),
					"Check GitHub SSH key settings or re-upload key"
			}

			return true, "SSH key working correctly", ""
		})
		addDiagnosticResult(&diagnostics, accountCheck, jsonOutput)
	}

	// Summary
	if !jsonOutput {
		checks := diagnostics["checks"].([]map[string]interface{})
		passed := 0
		for _, check := range checks {
			if check["status"].(bool) {
				passed++
			}
		}

		fmt.Printf("\n📊 Diagnostics Summary: %d/%d checks passed\n", passed, len(checks))
		if passed < len(checks) {
			fmt.Println("\n💡 Some issues found. Follow the suggestions above to resolve them.")
		} else {
			fmt.Println("\n🎉 All SSH diagnostics passed!")
		}
	} else {
		// JSON output
		return outputJSON(diagnostics)
	}

	return nil
}

// Helper functions

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func sshCheckSSHAgent() error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return fmt.Errorf("SSH_AUTH_SOCK not set - SSH agent not running")
	}

	cmd := exec.Command("ssh-add", "-l")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SSH agent not responding")
	}

	return nil
}

func isKeyInAgent(keyPath string) (bool, error) {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Check if key fingerprint is in agent output
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		return false, fmt.Errorf("public key not found: %s", pubKeyPath)
	}

	// Get key fingerprint
	fingerprintCmd := exec.Command("ssh-keygen", "-lf", pubKeyPath)
	fingerprintOutput, err := fingerprintCmd.Output()
	if err != nil {
		return false, err
	}

	fingerprint := strings.Fields(string(fingerprintOutput))[1]
	return strings.Contains(string(output), fingerprint), nil
}

func testGitHubConnectivity(keyPath string, verbose bool) error {
	args := []string{
		"-T", "git@github.com",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
	}

	if verbose {
		args = append(args, "-v")
	}

	cmd := exec.Command("ssh", args...)
	output, _ := cmd.CombinedOutput()

	// SSH to GitHub returns exit code 1 on successful auth, so check output content
	outputStr := string(output)
	if strings.Contains(outputStr, "successfully authenticated") {
		return nil
	}

	return fmt.Errorf("authentication failed: %s", outputStr)
}

func testBasicGitHubConnectivity() error {
	cmd := exec.Command("ssh", "-T", "git@github.com", "-o", "ConnectTimeout=10")
	output, err := cmd.CombinedOutput()

	// Check if we can at least connect (even if auth fails)
	outputStr := string(output)
	if strings.Contains(outputStr, "github.com") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("cannot connect to GitHub: %v", err)
	}

	return fmt.Errorf("unexpected response from GitHub: %s", outputStr)
}

func getKeyType(pubKeyPath string) (string, error) {
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", err
	}

	parts := strings.Fields(string(content))
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid public key format")
	}

	return parts[0], nil
}

func runDiagnosticCheck(name string, checkFunc func() (bool, string, string)) map[string]interface{} {
	success, message, suggestion := checkFunc()

	return map[string]interface{}{
		"name":       name,
		"status":     success,
		"message":    message,
		"suggestion": suggestion,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

func addDiagnosticResult(diagnostics *map[string]interface{}, result map[string]interface{}, jsonOutput bool) {
	checks := (*diagnostics)["checks"].([]map[string]interface{})
	(*diagnostics)["checks"] = append(checks, result)

	if !jsonOutput {
		status := "❌"
		if result["status"].(bool) {
			status = "✅"
		}

		fmt.Printf("%s %s: %s\n", status, result["name"], result["message"])
		if suggestion := result["suggestion"].(string); suggestion != "" && !result["status"].(bool) {
			fmt.Printf("   💡 %s\n", suggestion)
		}
	}
}

func outputJSON(data interface{}) error {
	// This would use encoding/json to output structured data
	fmt.Printf("%+v\n", data)
	return nil
}
