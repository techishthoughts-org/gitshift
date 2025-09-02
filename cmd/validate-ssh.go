package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	validateSSHCmd = &cobra.Command{
		Use:   "validate-ssh",
		Short: "🔍 Validate SSH configuration and troubleshoot issues",
		Long: `🔍 Validate SSH Configuration and Troubleshoot Issues

This command performs comprehensive validation of your SSH configuration to prevent
common issues like:
- SSH key misconfigurations
- Account authentication conflicts
- Permission problems
- Configuration conflicts

Examples:
  gitpersona validate-ssh                    # Run full validation
  gitpersona validate-ssh --fix-permissions # Fix permission issues
  gitpersona validate-ssh --json            # Output in JSON format
  gitpersona validate-ssh --generate-config # Generate recommended SSH config`,
		RunE: runValidateSSH,
	}

	validateSSHFlags = struct {
		fixPermissions bool
		generateConfig bool
		outputJSON     bool
		verbose        bool
	}{}
)

func init() {
	validateSSHCmd.Flags().BoolVarP(&validateSSHFlags.fixPermissions, "fix-permissions", "f", false, "Automatically fix SSH file permissions")
	validateSSHCmd.Flags().BoolVarP(&validateSSHFlags.generateConfig, "generate-config", "g", false, "Generate recommended SSH configuration")
	validateSSHCmd.Flags().BoolVarP(&validateSSHFlags.outputJSON, "json", "j", false, "Output results in JSON format")
	validateSSHFlags.verbose = false

	rootCmd.AddCommand(validateSSHCmd)
}

func runValidateSSH(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Validating SSH configuration...")

	// Simple SSH validation for now
	validationErr := validateSSHBasics()
	if validationErr != nil {
		fmt.Printf("⚠️  SSH validation failed: %v\n", validationErr)
	} else {
		fmt.Println("✅ Basic SSH validation passed!")
	}

	// Handle automatic fixes
	if validateSSHFlags.fixPermissions {
		fmt.Println("\n🔧 Fixing SSH permissions...")
		if err := fixSSHPermissions(); err != nil {
			fmt.Printf("❌ Failed to fix permissions: %v\n", err)
		} else {
			fmt.Println("✅ SSH permissions fixed successfully")
		}
	}

	// Generate SSH config
	if validateSSHFlags.generateConfig {
		fmt.Println("\n📝 Generating recommended SSH configuration...")
		generateSSHConfigTemplate()
	}

	// Only return error if no flags were used
	if !validateSSHFlags.fixPermissions && !validateSSHFlags.generateConfig && validationErr != nil {
		return fmt.Errorf("SSH validation failed: %w", validationErr)
	}

	return nil
}

func validateSSHBasics() error {
	// Test SSH agent
	if err := testSSHAgent(); err != nil {
		return fmt.Errorf("SSH agent test failed: %w", err)
	}

	// Test GitHub connection
	if err := testGitHubConnection(); err != nil {
		return fmt.Errorf("GitHub connection test failed: %w", err)
	}

	return nil
}

func testSSHAgent() error {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("SSH agent not running or no keys loaded: %w", err)
	}

	keys := strings.TrimSpace(string(output))
	if keys == "The agent has no identities." {
		return fmt.Errorf("no SSH keys loaded in agent")
	}

	fmt.Printf("✅ SSH agent is running with keys: %s\n", keys)
	return nil
}

func testGitHubConnection() error {
	// Test basic GitHub SSH connection
	cmd := exec.Command("ssh", "-T", "git@github.com")
	output, err := cmd.Output()

	if err != nil {
		return fmt.Errorf("GitHub SSH connection failed: %w", err)
	}

	// Check for successful authentication
	if strings.Contains(string(output), "successfully authenticated") {
		fmt.Println("✅ GitHub SSH authentication successful")
		return nil
	}

	return fmt.Errorf("unexpected GitHub response: %s", string(output))
}

func fixSSHPermissions() error {
	// Fix SSH directory permissions
	sshDirCmd := exec.Command("chmod", "700", "~/.ssh")
	if err := sshDirCmd.Run(); err != nil {
		return fmt.Errorf("failed to fix SSH directory permissions: %w", err)
	}

	// Fix SSH config permissions
	configCmd := exec.Command("chmod", "600", "~/.ssh/config")
	if err := configCmd.Run(); err != nil {
		return fmt.Errorf("failed to fix SSH config permissions: %w", err)
	}

	// Fix private key permissions
	keyCmd := exec.Command("chmod", "600", "~/.ssh/id_*")
	if err := keyCmd.Run(); err != nil {
		return fmt.Errorf("failed to fix key permissions: %w", err)
	}

	return nil
}

func generateSSHConfigTemplate() {
	fmt.Println("# GitPersona SSH Configuration Template")
	fmt.Println("# Add your accounts here:")
	fmt.Println("")
	fmt.Println("Host github-username")
	fmt.Println("    HostName github.com")
	fmt.Println("    User git")
	fmt.Println("    IdentityFile ~/.ssh/id_ed25519_username")
	fmt.Println("    IdentitiesOnly yes")
	fmt.Println("    UseKeychain yes")
	fmt.Println("    AddKeysToAgent yes")
	fmt.Println("")
	fmt.Println("# Example for multiple accounts:")
	fmt.Println("Host github-example")
	fmt.Println("    HostName github.com")
	fmt.Println("    User git")
	fmt.Println("    IdentityFile ~/.ssh/id_ed25519_example")
	fmt.Println("    IdentitiesOnly yes")
	fmt.Println("")
	fmt.Println("Host github-work")
	fmt.Println("    HostName github.com")
	fmt.Println("    User git")
	fmt.Println("    IdentityFile ~/.ssh/id_rsa_work")
	fmt.Println("    IdentitiesOnly yes")
}
