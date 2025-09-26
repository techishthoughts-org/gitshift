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

// SwitchToAccount switches SSH configuration to use the specified account
func (m *Manager) SwitchToAccount(accountAlias, keyPath string) error {
	// 1. Update SSH config to use the specific key for github.com
	if err := m.updateGitHubSSHConfig(accountAlias, keyPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	// 2. Clear SSH agent to force re-authentication
	if err := m.clearSSHAgent(); err != nil {
		return fmt.Errorf("failed to clear SSH agent: %w", err)
	}

	// 3. Add only the specific key to agent
	if err := m.addKeyToAgent(keyPath); err != nil {
		return fmt.Errorf("failed to add key to agent: %w", err)
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

// updateGitHubSSHConfig updates the SSH config to use the specific key for GitHub
func (m *Manager) updateGitHubSSHConfig(accountAlias, keyPath string) error {
	// Read current SSH config
	content, err := os.ReadFile(m.configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	// Create a completely new SSH config optimized for GitPersona
	newConfig := m.buildOptimizedSSHConfig(accountAlias, keyPath, string(content))

	// Write the updated config back
	if err := os.WriteFile(m.configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// buildOptimizedSSHConfig creates an optimized SSH config for GitPersona account switching
func (m *Manager) buildOptimizedSSHConfig(accountAlias, keyPath, existingConfig string) string {
	var result strings.Builder

	// Add GitPersona header
	result.WriteString("# GitPersona SSH Configuration\n")
	result.WriteString("# This configuration prevents SSH key conflicts when using multiple GitHub accounts\n")
	result.WriteString("# Last updated for account: " + accountAlias + "\n\n")

	// Add GitHub host configuration with strict isolation
	result.WriteString("Host github.com\n")
	result.WriteString("    HostName github.com\n")
	result.WriteString("    User git\n")
	result.WriteString(fmt.Sprintf("    IdentityFile %s\n", keyPath))
	result.WriteString("    IdentitiesOnly yes\n")
	result.WriteString("    PreferredAuthentications publickey\n")
	result.WriteString("    AddKeysToAgent no\n") // Prevent automatic key loading
	result.WriteString("    UseKeychain no\n")    // Prevent keychain integration
	result.WriteString("\n")

	// Preserve any non-GitHub host configurations from existing config
	if existingConfig != "" {
		lines := strings.Split(existingConfig, "\n")
		inGitHubSection := false

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Skip GitPersona header comments and github.com sections
			if strings.HasPrefix(trimmedLine, "# GitPersona") ||
				strings.HasPrefix(trimmedLine, "Host github.com") ||
				strings.Contains(trimmedLine, "github-") {
				if strings.HasPrefix(trimmedLine, "Host github.com") || strings.Contains(trimmedLine, "github-") {
					inGitHubSection = true
				}
				continue
			}

			// Check if we're leaving a GitHub section
			if inGitHubSection && strings.HasPrefix(trimmedLine, "Host ") && !strings.Contains(trimmedLine, "github") {
				inGitHubSection = false
			}

			// Skip lines that are part of GitHub sections
			if inGitHubSection {
				continue
			}

			// Preserve other host configurations
			if trimmedLine != "" || !inGitHubSection {
				result.WriteString(line + "\n")
			}
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
