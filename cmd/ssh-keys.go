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

var sshKeysCmd = &cobra.Command{
	Use:   "ssh-keys",
	Short: "Manage SSH keys and resolve authentication issues",
	Long: `Manage SSH keys and automatically resolve SSH authentication issues.

This command helps you manage SSH keys for multiple GitHub accounts, diagnose SSH problems,
and automatically configure the correct SSH key for each account.

Examples:
  gitpersona ssh-keys list              # List all SSH keys
  gitpersona ssh-keys diagnose          # Diagnose SSH authentication issues
  gitpersona ssh-keys test [account]    # Test SSH connection for an account
  gitpersona ssh-keys generate [account] # Generate new SSH key for an account
  gitpersona ssh-keys setup [account]   # Setup SSH key for an account`,
}

func init() {
	rootCmd.AddCommand(sshKeysCmd)
}

func executeSSHKeys(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("action required. Use: list, diagnose, test, generate, or setup")
	}

	action := args[0]

	switch action {
	case "list":
		return listSSHKeys()
	case "diagnose":
		return diagnoseSSHKeys()
	case "test":
		if len(args) < 2 {
			return fmt.Errorf("account name required for test action")
		}
		return testSSHConnection(args[1])
	case "generate":
		if len(args) < 2 {
			return fmt.Errorf("account name required for generate action")
		}
		return generateSSHKey(args[1])
	case "setup":
		if len(args) < 2 {
			return fmt.Errorf("account name required for setup action")
		}
		return setupSSHKey(args[1])
	default:
		return fmt.Errorf("invalid action: %s. Use: list, diagnose, test, generate, or setup", action)
	}
}

// listSSHKeys lists all available SSH keys
func listSSHKeys() error {
	fmt.Println("🔑 Available SSH Keys:")
	fmt.Println("=======================")

	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	files, err := os.ReadDir(sshDir)
	if err != nil {
		return fmt.Errorf("failed to read SSH directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".pub") {
			keyPath := filepath.Join(sshDir, file.Name())
			content, err := os.ReadFile(keyPath)
			if err != nil {
				continue
			}

			// Extract email/username from key comment
			parts := strings.Fields(string(content))
			if len(parts) >= 3 {
				comment := parts[2]
				fmt.Printf("📁 %s\n", file.Name())
				fmt.Printf("   Comment: %s\n", comment)
				fmt.Printf("   Path: %s\n", keyPath)
				fmt.Println()
			}
		}
	}

	return nil
}

// diagnoseSSHKeys diagnoses SSH authentication issues
func diagnoseSSHKeys() error {
	fmt.Println("🔍 Diagnosing SSH authentication issues...")
	fmt.Println("")

	// Check SSH agent
	fmt.Println("🔐 SSH Agent Status:")
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("❌ SSH agent not running or no keys loaded")
		fmt.Println("💡 Run: eval $(ssh-agent -s) && ssh-add ~/.ssh/id_rsa")
	} else {
		fmt.Println("✅ SSH agent running with keys:")
		fmt.Println(string(output))
	}

	fmt.Println("")

	// Check GitHub SSH connection
	fmt.Println("🌐 Testing GitHub SSH Connection:")
	cmd = exec.Command("ssh", "-T", "git@github.com")
	output, _ = cmd.CombinedOutput()

	if strings.Contains(string(output), "successfully authenticated") {
		// Extract username from output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Hi") && strings.Contains(line, "!") {
				fmt.Printf("✅ %s\n", strings.TrimSpace(line))
				break
			}
		}
	} else {
		fmt.Println("❌ GitHub SSH connection failed")
		fmt.Println("💡 Check your SSH keys and GitHub account settings")
	}

	return nil
}

// testSSHConnection tests SSH connection for a specific account
func testSSHConnection(account string) error {
	fmt.Printf("🧪 Testing SSH connection for account: %s\n", account)

	// Get account SSH key path
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	config := configManager.GetConfig()
	accountData, exists := config.Accounts[account]
	if !exists {
		return fmt.Errorf("account '%s' not found", account)
	}

	if accountData.SSHKeyPath == "" {
		return fmt.Errorf("account '%s' has no SSH key configured", account)
	}

	// Test SSH connection with specific key
	cmd := exec.Command("ssh", "-i", accountData.SSHKeyPath, "-T", "git@github.com")
	output, _ := cmd.CombinedOutput()

	if strings.Contains(string(output), "successfully authenticated") {
		fmt.Println("✅ SSH connection successful")
		// Extract username from output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Hi") && strings.Contains(line, "!") {
				fmt.Printf("   Authenticated as: %s\n", strings.TrimSpace(line))
				break
			}
		}
	} else {
		fmt.Println("❌ SSH connection failed")
		fmt.Printf("   Error: %s\n", strings.TrimSpace(string(output)))
		fmt.Println("💡 Check if the SSH key is correct and added to GitHub")
	}

	return nil
}

// generateSSHKey generates a new SSH key for an account
func generateSSHKey(account string) error {
	fmt.Printf("🔑 Generating new SSH key for account: %s\n", account)

	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	keyName := fmt.Sprintf("id_ed25519_%s", account)
	keyPath := filepath.Join(sshDir, keyName)

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("SSH key already exists: %s", keyPath)
	}

	// Generate new key
	email := fmt.Sprintf("%s@users.noreply.github.com", account)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-C", email)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("💡 Press Enter to accept default passphrase (empty)")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	fmt.Printf("✅ SSH key generated: %s\n", keyPath)
	fmt.Printf("📋 Public key: %s.pub\n", keyPath)

	// Show the public key
	publicKeyPath := keyPath + ".pub"
	content, err := os.ReadFile(publicKeyPath)
	if err == nil {
		fmt.Println("🔑 Public key content:")
		fmt.Println(string(content))
		fmt.Println("💡 Add this key to your GitHub account")
	}

	return nil
}

// setupSSHKey sets up SSH key for an account
func setupSSHKey(account string) error {
	fmt.Printf("⚙️  Setting up SSH key for account: %s\n", account)

	// Get account SSH key path
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	config := configManager.GetConfig()
	accountData, exists := config.Accounts[account]
	if !exists {
		return fmt.Errorf("account '%s' not found", account)
	}

	if accountData.SSHKeyPath == "" {
		return fmt.Errorf("account '%s' has no SSH key configured", account)
	}

	// Check if key exists
	if _, err := os.Stat(accountData.SSHKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found: %s", accountData.SSHKeyPath)
	}

	// Add key to SSH agent
	fmt.Printf("🔄 Adding key to SSH agent: %s\n", accountData.SSHKeyPath)
	cmd := exec.Command("ssh-add", accountData.SSHKeyPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add key to SSH agent: %w", err)
	}

	fmt.Println("✅ SSH key added to agent")

	// Test connection
	fmt.Println("🧪 Testing connection...")
	return testSSHConnection(account)
}
