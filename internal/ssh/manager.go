package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Manager handles SSH configuration and key management
type Manager struct {
	homeDir    string
	configPath string
}

// NewManager creates a new SSH manager
func NewManager() *Manager {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "~"
	}

	return &Manager{
		homeDir:    homeDir,
		configPath: filepath.Join(homeDir, ".ssh", "config"),
	}
}

// SwitchToAccount switches SSH configuration to use the specified account with improved isolation
func (m *Manager) SwitchToAccount(accountAlias, keyPath string) error {
	// 1. Validate key exists and fix permissions
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}

	// Fix key permissions if needed
	if info, err := os.Stat(keyPath); err == nil {
		if info.Mode().Perm() != 0600 {
			if err := os.Chmod(keyPath, 0600); err != nil {
				return fmt.Errorf("failed to fix SSH key permissions: %w", err)
			}
		}
	}

	// 2. Update SSH config with improved isolation
	if err := m.updateGitHubSSHConfigV2(accountAlias, keyPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	// 3. Clear SSH agent and load only the required key
	if err := m.clearSSHAgent(); err != nil {
		// Don't fail if SSH agent operations fail
		fmt.Printf("⚠️  Warning: SSH agent clear failed: %v\n", err)
	}

	// 4. Add only the specific key to agent
	if err := m.addKeyToAgent(keyPath); err != nil {
		// Don't fail if SSH agent operations fail, SSH config should be enough
		fmt.Printf("⚠️  Warning: SSH agent key loading failed: %v\n", err)
	}

	// 5. Update shell configuration with GIT_SSH_COMMAND
	if err := m.updateShellConfig(accountAlias, keyPath); err != nil {
		// Don't fail the entire operation if shell config update fails
		fmt.Printf("⚠️  Warning: failed to update shell configuration: %v\n", err)
	} else {
		fmt.Printf("✅ Shell configuration updated for account: %s\n", accountAlias)
	}

	// 6. Test the connection (don't fail on error)
	if err := m.TestConnection(); err != nil {
		fmt.Printf("⚠️  Warning: SSH connection test failed: %v\n", err)
	}

	return nil
}

// GetLoadedKeys returns the list of currently loaded SSH keys
func (m *Manager) GetLoadedKeys() ([]string, error) {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "The agent has no identities") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var keys []string
	for _, line := range lines {
		if parts := strings.Fields(line); len(parts) >= 3 {
			keys = append(keys, strings.TrimSpace(parts[2]))
		}
	}
	return keys, nil
}

// detectShell detects the user's shell and returns shell type and config file path
func (m *Manager) detectShell() (shellType, configPath string, err error) {
	// On Windows, we don't update shell config (not reliable)
	if runtime.GOOS == "windows" {
		return "", "", fmt.Errorf("shell config update not supported on Windows - please set GIT_SSH_COMMAND environment variable manually:\n" +
			"  PowerShell: $env:GIT_SSH_COMMAND = 'ssh -i <path> -o IdentitiesOnly=yes'\n" +
			"  CMD:        set GIT_SSH_COMMAND=ssh -i <path> -o IdentitiesOnly=yes")
	}

	// Try to detect shell from SHELL environment variable
	shellEnv := os.Getenv("SHELL")

	// Determine shell type and config file
	switch {
	case strings.Contains(shellEnv, "zsh"):
		// ZSH: check for .zshrc, then .zsh_secrets
		zshrc := filepath.Join(m.homeDir, ".zshrc")
		if _, err := os.Stat(zshrc); err == nil {
			return "zsh", zshrc, nil
		}
		// Fall back to .zsh_secrets for backward compatibility
		return "zsh", filepath.Join(m.homeDir, ".zsh_secrets"), nil

	case strings.Contains(shellEnv, "bash"):
		// Bash: check for .bashrc on Linux, .bash_profile on macOS
		if runtime.GOOS == "darwin" {
			bashProfile := filepath.Join(m.homeDir, ".bash_profile")
			if _, err := os.Stat(bashProfile); err == nil {
				return "bash", bashProfile, nil
			}
		}
		bashrc := filepath.Join(m.homeDir, ".bashrc")
		return "bash", bashrc, nil

	case strings.Contains(shellEnv, "fish"):
		// Fish shell uses a different directory structure
		fishConfig := filepath.Join(m.homeDir, ".config", "fish", "config.fish")
		return "fish", fishConfig, nil

	case strings.Contains(shellEnv, "ksh"):
		// Korn shell
		return "ksh", filepath.Join(m.homeDir, ".kshrc"), nil

	default:
		// Default fallback: try .profile (POSIX standard)
		profile := filepath.Join(m.homeDir, ".profile")
		return "posix", profile, nil
	}
}

// updateShellConfig updates the shell configuration to set GIT_SSH_COMMAND
func (m *Manager) updateShellConfig(accountAlias, keyPath string) error {
	// Detect shell and get config file path
	shellType, configPath, err := m.detectShell()
	if err != nil {
		return err
	}

	// Read existing content
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read shell config: %w", err)
	}

	// Create backup
	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Remove existing GIT_SSH_COMMAND if it exists
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Remove both export and set variants (for fish shell)
		if !strings.HasPrefix(trimmed, "export GIT_SSH_COMMAND=") &&
			!strings.HasPrefix(trimmed, "set -x GIT_SSH_COMMAND") {
			newLines = append(newLines, line)
		}
	}

	// Add new GIT_SSH_COMMAND based on shell type
	var commandLine string
	if shellType == "fish" {
		// Fish shell uses 'set -x' for environment variables
		commandLine = fmt.Sprintf(`set -x GIT_SSH_COMMAND "ssh -i %s -o IdentitiesOnly=yes"`, keyPath)
	} else {
		// Bash, ZSH, KSH, and POSIX shells use 'export'
		commandLine = fmt.Sprintf(`export GIT_SSH_COMMAND="ssh -i %s -o IdentitiesOnly=yes"`, keyPath)
	}

	// Add comment before the command for clarity
	newLines = append(newLines, "")
	newLines = append(newLines, "# gitshift: SSH key configuration")
	newLines = append(newLines, commandLine)

	// Add a newline at the end
	newLines = append(newLines, "")

	// Write back to file
	if err := os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		// Try to restore backup if write fails
		_ = os.WriteFile(configPath, content, 0600)
		return fmt.Errorf("failed to write shell config: %w", err)
	}

	// Print helpful message about what was done
	fmt.Printf("📝 Updated %s config: %s\n", shellType, configPath)
	fmt.Printf("💡 Run 'source %s' or restart your terminal to apply changes\n", configPath)

	return nil
}

// SourceShellConfig sources the shell configuration to apply changes
func (m *Manager) SourceShellConfig() error {
	// Detect shell and get config file path
	shellType, configPath, err := m.detectShell()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	// Build the appropriate source command for each shell
	switch shellType {
	case "fish":
		// Fish shell uses 'source' command
		cmd = exec.Command("fish", "-c", fmt.Sprintf("source %s", configPath))
	case "bash":
		// Bash uses 'source' or '.'
		cmd = exec.Command("bash", "-c", fmt.Sprintf("source %s", configPath))
	case "zsh":
		// ZSH uses 'source'
		cmd = exec.Command("zsh", "-c", fmt.Sprintf("source %s", configPath))
	case "ksh":
		// KSH uses '.'
		cmd = exec.Command("ksh", "-c", fmt.Sprintf(". %s", configPath))
	default:
		// POSIX shells use '.'
		cmd = exec.Command("sh", "-c", fmt.Sprintf(". %s", configPath))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// clearSSHAgent removes all keys from the SSH agent
func (m *Manager) clearSSHAgent() error {
	cmd := exec.Command("ssh-add", "-D")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// It's OK if there are no keys to remove
		if strings.Contains(string(output), "no identities") {
			return nil
		}
		return fmt.Errorf("ssh-add -D failed: %w", err)
	}
	return nil
}

// addKeyToAgent adds a specific key to the SSH agent
func (m *Manager) addKeyToAgent(keyPath string) error {
	cmd := exec.Command("ssh-add", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add %s failed: %w\nOutput: %s", keyPath, err, string(output))
	}
	return nil
}

// TestConnection tests the SSH connection to GitHub (deprecated, use TestConnectionToPlatform)
func (m *Manager) TestConnection() error {
	return m.TestConnectionToPlatform("github.com")
}

// TestConnectionToPlatform tests the SSH connection to a specific platform domain
func (m *Manager) TestConnectionToPlatform(domain string) error {
	// Test SSH connection to the specified domain
	testCmd := exec.Command("ssh", "-T", fmt.Sprintf("git@%s", domain))
	output, err := testCmd.CombinedOutput()
	outputStr := string(output)

	// Different platforms return different success messages
	// GitHub: "successfully authenticated"
	// GitLab: "Welcome to GitLab"
	// Both return non-zero exit codes despite successful authentication
	successIndicators := []string{
		"successfully authenticated",
		"Welcome to GitLab",
		"logged in as",
		"authenticated",
	}

	for _, indicator := range successIndicators {
		if strings.Contains(outputStr, indicator) {
			return nil
		}
	}

	// If no error and output suggests success
	if err == nil {
		return nil
	}

	return fmt.Errorf("SSH connection test to %s failed: %w\nOutput: %s", domain, err, outputStr)
}

// updateGitHubSSHConfigV2 updates the SSH config with improved multi-account isolation
// Deprecated: Use UpdateSSHConfig with platform domain instead
func (m *Manager) updateGitHubSSHConfigV2(accountAlias, keyPath string) error {
	return m.UpdateSSHConfig(accountAlias, keyPath, "github.com")
}

// UpdateSSHConfig updates the SSH config for a specific platform domain
func (m *Manager) UpdateSSHConfig(accountAlias, keyPath, domain string) error {
	// Ensure SSH directory exists
	sshDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Read current SSH config (if exists)
	existingContent := ""
	if content, err := os.ReadFile(m.configPath); err == nil {
		if !strings.Contains(string(content), "# gitshift Managed Config") {
			// Backup existing config if it's not already managed by gitshift
			backupPath := m.configPath + ".backup"
			if err := os.WriteFile(backupPath, content, 0600); err != nil {
				return fmt.Errorf("failed to backup SSH config: %w", err)
			}
		}
		existingContent = string(content)
	}

	// Build the new config
	newConfig := m.buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, domain, existingContent)

	// Write the updated config
	if err := os.WriteFile(m.configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// buildIsolatedSSHConfig creates an SSH config with proper multi-account isolation
// Deprecated: Use buildIsolatedSSHConfigForPlatform instead
func (m *Manager) buildIsolatedSSHConfig(accountAlias, keyPath, existingConfig string) string {
	return m.buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, "github.com", existingConfig)
}

// buildIsolatedSSHConfigForPlatform creates an SSH config for a specific platform
func (m *Manager) buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, domain, existingConfig string) string {
	// Start with the gitshift header
	config := "# gitshift Managed Config - DO NOT EDIT MANUALLY\n"
	config += "# This file is automatically generated by gitshift\n\n"

	// Preserve non-platform configurations
	config += m.preserveNonPlatformConfig(existingConfig, domain)

	// Determine platform name for comment
	platformName := "Git hosting"
	switch domain {
	case "github.com":
		platformName = "GitHub"
	case "gitlab.com":
		platformName = "GitLab"
	case "bitbucket.org":
		platformName = "Bitbucket"
	}

	// Add platform host configuration
	config += fmt.Sprintf(`# %s account: %s
Host %s
    HostName %s
    User git
    IdentityFile %s
    IdentitiesOnly yes
    AddKeysToAgent yes
    UseKeychain yes

`, platformName, accountAlias, domain, domain, keyPath)

	return config
}

// preserveNonGitHubConfig extracts and preserves non-GitHub host configurations
// Deprecated: Use preserveNonPlatformConfig instead
func (m *Manager) preserveNonGitHubConfig(existingConfig string) string {
	return m.preserveNonPlatformConfig(existingConfig, "github.com")
}

// preserveNonPlatformConfig extracts and preserves configurations not related to the specified domain
func (m *Manager) preserveNonPlatformConfig(existingConfig, targetDomain string) string {
	if existingConfig == "" {
		return ""
	}

	var preserved []string
	lines := strings.Split(existingConfig, "\n")
	inTargetSection := false
	skipUntilNextHost := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Check if we're entering a target platform host section
		if strings.HasPrefix(line, "Host ") {
			hostLine := strings.TrimSpace(strings.TrimPrefix(line, "Host"))
			hosts := strings.Fields(hostLine)

			// Check if any of the hosts matches the target domain
			isTargetHost := false
			for _, host := range hosts {
				if host == targetDomain || strings.Contains(host, targetDomain) {
					isTargetHost = true
					break
				}
			}

			if isTargetHost {
				inTargetSection = true
				skipUntilNextHost = true
				continue
			}

			// If we were in a target section and found a new host, we're done
			if inTargetSection {
				inTargetSection = false
			}

			// If we were skipping until next host, stop skipping
			if skipUntilNextHost {
				skipUntilNextHost = false
			}
		}

		// Skip lines in target sections or when we're in skip mode
		if inTargetSection || skipUntilNextHost {
			continue
		}

		// Preserve all other lines
		preserved = append(preserved, lines[i])
	}

	if len(preserved) > 0 {
		// Ensure there's a blank line before adding platform config
		if preserved[len(preserved)-1] != "" {
			preserved = append(preserved, "")
		}
		return strings.Join(preserved, "\n") + "\n"
	}

	return ""
}
