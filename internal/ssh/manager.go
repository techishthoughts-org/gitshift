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
	// 1. Clear SSH agent
	if err := m.clearSSHAgent(); err != nil {
		return fmt.Errorf("failed to clear SSH agent: %w", err)
	}

	// 2. Add the specific key to agent
	if err := m.addKeyToAgent(keyPath); err != nil {
		return fmt.Errorf("failed to add key to agent: %w", err)
	}

	// 3. Update SSH config to prioritize this account's key
	if err := m.updateSSHConfig(accountAlias, keyPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
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

// updateSSHConfig updates the SSH config to prioritize the account's key
func (m *Manager) updateSSHConfig(accountAlias, keyPath string) error {
	// Read current SSH config
	content, err := os.ReadFile(m.configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	configText := string(content)

	// Find the GitHub section and update the default IdentityFile
	lines := strings.Split(configText, "\n")
	newLines := make([]string, 0, len(lines))
	inGitHubSection := false
	foundIdentityFile := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the github.com section
		if strings.HasPrefix(trimmedLine, "Host github.com") {
			inGitHubSection = true
			newLines = append(newLines, line)
			continue
		}

		// Check if we're leaving the github.com section
		if inGitHubSection && strings.HasPrefix(trimmedLine, "Host ") && !strings.Contains(trimmedLine, "github") {
			inGitHubSection = false
		}

		// If we're in the github.com section and find IdentityFile, update it
		if inGitHubSection && strings.HasPrefix(trimmedLine, "IdentityFile") {
			newLines = append(newLines, fmt.Sprintf("    IdentityFile %s", keyPath))
			foundIdentityFile = true
			continue
		}

		newLines = append(newLines, line)
	}

	// If we didn't find an IdentityFile line in the github.com section, we need to add it
	if inGitHubSection && !foundIdentityFile {
		// Find the end of the github.com section and add IdentityFile before it
		for i := len(newLines) - 1; i >= 0; i-- {
			if strings.HasPrefix(strings.TrimSpace(newLines[i]), "Host github.com") {
				// Insert IdentityFile after the Host line
				insertIndex := i + 1
				// Find where to insert (after existing properties)
				for j := insertIndex; j < len(newLines); j++ {
					line := strings.TrimSpace(newLines[j])
					if line == "" || strings.HasPrefix(line, "Host ") {
						insertIndex = j
						break
					}
					insertIndex = j + 1
				}

				// Insert the IdentityFile line
				newLine := fmt.Sprintf("    IdentityFile %s", keyPath)
				newLines = append(newLines[:insertIndex], append([]string{newLine}, newLines[insertIndex:]...)...)
				break
			}
		}
	}

	// Write the updated config back
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(m.configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
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
