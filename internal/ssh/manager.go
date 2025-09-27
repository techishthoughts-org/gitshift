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

	// 5. Test the connection
	if err := m.TestConnection(); err != nil {
		// Don't fail if connection test fails, might be network issue
		fmt.Printf("⚠️  Warning: SSH connection test failed: %v\n", err)
	}

	return nil
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
		existingContent = string(content)
	}

	// Build the new optimized SSH config
	newConfig := m.buildIsolatedSSHConfig(accountAlias, keyPath, existingContent)

	// Backup existing config if it exists and isn't already a GitPersona config
	if existingContent != "" && !strings.Contains(existingContent, "GitPersona SSH Configuration") {
		backupPath := m.configPath + ".backup-" + fmt.Sprintf("%d", os.Getpid())
		if err := os.WriteFile(backupPath, []byte(existingContent), 0600); err != nil {
			fmt.Printf("⚠️  Warning: failed to backup SSH config: %v\n", err)
		}
	}

	// Write the updated config
	if err := os.WriteFile(m.configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// buildIsolatedSSHConfig creates an SSH config with proper multi-account isolation
func (m *Manager) buildIsolatedSSHConfig(accountAlias, keyPath, existingConfig string) string {
	var result strings.Builder

	// Add GitPersona header with clear identification
	result.WriteString("# GitPersona SSH Configuration\n")
	result.WriteString("# This configuration ensures proper isolation between multiple GitHub accounts\n")
	result.WriteString(fmt.Sprintf("# Current active account: %s\n", accountAlias))
	result.WriteString("# DO NOT EDIT MANUALLY - Managed by GitPersona\n\n")

	// Add strict GitHub configuration with complete isolation
	result.WriteString("# Primary GitHub host with complete isolation\n")
	result.WriteString("Host github.com\n")
	result.WriteString("    HostName github.com\n")
	result.WriteString("    User git\n")
	result.WriteString(fmt.Sprintf("    IdentityFile %s\n", keyPath))
	result.WriteString("    IdentitiesOnly yes\n")
	result.WriteString("    PreferredAuthentications publickey\n")
	result.WriteString("    PubkeyAuthentication yes\n")
	result.WriteString("    AddKeysToAgent no\n") // Prevent automatic key loading
	result.WriteString("    UseKeychain no\n")    // Disable keychain integration
	result.WriteString("    StrictHostKeyChecking accept-new\n")
	result.WriteString("    UserKnownHostsFile ~/.ssh/known_hosts\n")
	result.WriteString("    ConnectTimeout 10\n")
	result.WriteString("    ServerAliveInterval 60\n")
	result.WriteString("    ServerAliveCountMax 3\n")
	result.WriteString("\n")

	// Add catch-all for any GitHub subdomain or SSH variant
	result.WriteString("# Catch-all for GitHub variants\n")
	result.WriteString("Host *.github.com github-*\n")
	result.WriteString("    HostName github.com\n")
	result.WriteString("    User git\n")
	result.WriteString(fmt.Sprintf("    IdentityFile %s\n", keyPath))
	result.WriteString("    IdentitiesOnly yes\n")
	result.WriteString("    PreferredAuthentications publickey\n")
	result.WriteString("    PubkeyAuthentication yes\n")
	result.WriteString("    AddKeysToAgent no\n")
	result.WriteString("    UseKeychain no\n")
	result.WriteString("\n")

	// Preserve non-GitHub configurations from existing config
	if existingConfig != "" {
		result.WriteString("# Preserved non-GitHub configurations\n")
		preservedConfig := m.preserveNonGitHubConfig(existingConfig)
		if preservedConfig != "" {
			result.WriteString(preservedConfig)
			result.WriteString("\n")
		}
	}

	// Add global defaults that don't conflict with GitHub isolation
	result.WriteString("# Global SSH defaults (non-conflicting)\n")
	result.WriteString("Host *\n")
	result.WriteString("    Protocol 2\n")
	result.WriteString("    Compression yes\n")
	result.WriteString("    TCPKeepAlive yes\n")
	result.WriteString("    ServerAliveInterval 60\n")
	result.WriteString("    ServerAliveCountMax 3\n")
	result.WriteString("    # Note: Key management is handled per-host above\n\n")

	return result.String()
}

// preserveNonGitHubConfig extracts and preserves non-GitHub host configurations
func (m *Manager) preserveNonGitHubConfig(existingConfig string) string {
	var result strings.Builder
	lines := strings.Split(existingConfig, "\n")
	inGitHubSection := false
	inGitPersonaSection := false
	currentHostLines := []string{}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip GitPersona managed sections
		if strings.Contains(trimmedLine, "GitPersona") {
			inGitPersonaSection = true
			continue
		}
		if inGitPersonaSection && trimmedLine == "" {
			inGitPersonaSection = false
			continue
		}
		if inGitPersonaSection {
			continue
		}

		// Detect GitHub-related host entries
		if strings.HasPrefix(trimmedLine, "Host ") {
			// Process previous host if it wasn't GitHub-related
			if len(currentHostLines) > 0 && !inGitHubSection {
				for _, hostLine := range currentHostLines {
					result.WriteString(hostLine + "\n")
				}
			}

			// Reset for new host
			currentHostLines = []string{line}
			inGitHubSection = strings.Contains(strings.ToLower(trimmedLine), "github")
		} else if strings.HasPrefix(trimmedLine, "    ") || strings.HasPrefix(trimmedLine, "\t") || trimmedLine == "" {
			// Add configuration line to current host
			currentHostLines = append(currentHostLines, line)
		} else {
			// Not a host configuration, add directly if not in GitHub section
			if !inGitHubSection {
				result.WriteString(line + "\n")
			}
		}
	}

	// Process final host if it wasn't GitHub-related
	if len(currentHostLines) > 0 && !inGitHubSection {
		for _, hostLine := range currentHostLines {
			result.WriteString(hostLine + "\n")
		}
	}

	return result.String()
}

// TestConnection tests the SSH connection to GitHub
func (m *Manager) TestConnection() error {
	cmd := exec.Command("ssh", "-T", "git@github.com")
	output, err := cmd.CombinedOutput()

	// SSH connection to GitHub should fail with exit code 1 but give us the auth message
	if err != nil {
		if strings.Contains(string(output), "successfully authenticated") {
			return nil // This is the expected behavior
		}
		return fmt.Errorf("SSH connection test failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetLoadedKeys returns the list of currently loaded SSH keys
func (m *Manager) GetLoadedKeys() ([]string, error) {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// No keys loaded
		if strings.Contains(string(output), "no identities") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}
