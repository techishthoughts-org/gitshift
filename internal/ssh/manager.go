package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// updateShellConfig updates the shell configuration to set GIT_SSH_COMMAND
func (m *Manager) updateShellConfig(accountAlias, keyPath string) error {
	// Path to the shell config file
	configPath := filepath.Join(m.homeDir, ".zsh_secrets")

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
		if !strings.HasPrefix(strings.TrimSpace(line), "export GIT_SSH_COMMAND=") {
			newLines = append(newLines, line)
		}
	}

	// Add new GIT_SSH_COMMAND
	newLines = append(newLines, fmt.Sprintf(`export GIT_SSH_COMMAND="ssh -i %s -o IdentitiesOnly=yes"`, keyPath))

	// Add a newline at the end if the file wasn't empty
	if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
		newLines = append(newLines, "")
	}

	// Write back to file
	if err := os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		// Try to restore backup if write fails
		_ = os.WriteFile(configPath, content, 0600)
		return fmt.Errorf("failed to write shell config: %w", err)
	}

	return nil
}

// SourceShellConfig sources the shell configuration to apply changes
func (m *Manager) SourceShellConfig() error {
	configPath := filepath.Join(m.homeDir, ".zsh_secrets")
	cmd := exec.Command("zsh", "-c", fmt.Sprintf("source %s", configPath))
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

// TestConnection tests the SSH connection to GitHub
func (m *Manager) TestConnection() error {
	// Test SSH connection to GitHub
	testCmd := exec.Command("ssh", "-T", "git@github.com")
	output, err := testCmd.CombinedOutput()
	outputStr := string(output)

	// GitHub returns success (0) but with a message when authentication succeeds
	// but shell access is not granted (which is the expected behavior)
	if err == nil || strings.Contains(outputStr, "successfully authenticated") {
		return nil
	}

	return fmt.Errorf("SSH connection test failed: %w\nOutput: %s", err, outputStr)
}

// updateGitHubSSHConfigV2 updates the SSH config with improved multi-account isolation
func (m *Manager) updateGitHubSSHConfigV2(accountAlias, keyPath string) error {
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
	newConfig := m.buildIsolatedSSHConfig(accountAlias, keyPath, existingContent)

	// Write the updated config
	if err := os.WriteFile(m.configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// buildIsolatedSSHConfig creates an SSH config with proper multi-account isolation
func (m *Manager) buildIsolatedSSHConfig(accountAlias, keyPath, existingConfig string) string {
	// Start with the gitshift header
	config := "# gitshift Managed Config - DO NOT EDIT MANUALLY\n"
	config += "# This file is automatically generated by gitshift\n\n"

	// Preserve non-GitHub configurations
	config += m.preserveNonGitHubConfig(existingConfig)

	// Add GitHub host configuration
	config += fmt.Sprintf(`# GitHub account: %s
Host github.com
    HostName github.com
    User git
    IdentityFile %s
    IdentitiesOnly yes
    AddKeysToAgent yes
    UseKeychain yes

`, accountAlias, keyPath)

	return config
}

// preserveNonGitHubConfig extracts and preserves non-GitHub host configurations
func (m *Manager) preserveNonGitHubConfig(existingConfig string) string {
	if existingConfig == "" {
		return ""
	}

	var preserved []string
	lines := strings.Split(existingConfig, "\n")
	inGitHubSection := false
	skipUntilNextHost := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Check if we're entering a GitHub host section
		if strings.HasPrefix(line, "Host ") {
			hostLine := strings.TrimSpace(strings.TrimPrefix(line, "Host"))
			hosts := strings.Fields(hostLine)

			// Check if any of the hosts is github.com
			isGitHubHost := false
			for _, host := range hosts {
				if host == "github.com" || strings.Contains(host, "github") {
					isGitHubHost = true
					break
				}
			}

			if isGitHubHost {
				inGitHubSection = true
				skipUntilNextHost = true
				continue
			}

			// If we were in a GitHub section and found a new host, we're done
			if inGitHubSection {
				inGitHubSection = false
			}

			// If we were skipping until next host, stop skipping
			if skipUntilNextHost {
				skipUntilNextHost = false
			}
		}

		// Skip lines in GitHub sections or when we're in skip mode
		if inGitHubSection || skipUntilNextHost {
			continue
		}

		// Preserve all other lines
		preserved = append(preserved, lines[i])
	}

	if len(preserved) > 0 {
		// Ensure there's a blank line before adding GitHub config
		if preserved[len(preserved)-1] != "" {
			preserved = append(preserved, "")
		}
		return strings.Join(preserved, "\n") + "\n"
	}

	return ""
}
