package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/models"
)

// Manager handles Git configuration operations
type Manager struct{}

// NewManager creates a new Git manager
func NewManager() *Manager {
	return &Manager{}
}

// IsGitRepo checks if the current directory is a Git repository
func (m *Manager) IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}

	// Check if we're inside a git worktree
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

// SetLocalConfig sets the Git configuration for the current repository
func (m *Manager) SetLocalConfig(account *models.Account) error {
	if err := m.setUserName(account.Name, false); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	if err := m.setUserEmail(account.Email, false); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	return nil
}

// SetGlobalConfig sets the global Git configuration
func (m *Manager) SetGlobalConfig(account *models.Account) error {
	if err := m.setUserName(account.Name, true); err != nil {
		return fmt.Errorf("failed to set global user.name: %w", err)
	}

	if err := m.setUserEmail(account.Email, true); err != nil {
		return fmt.Errorf("failed to set global user.email: %w", err)
	}

	return nil
}

// GetCurrentConfig returns the current Git configuration
func (m *Manager) GetCurrentConfig() (name, email string, err error) {
	name, err = m.getConfigValue("user.name")
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.name: %w", err)
	}

	email, err = m.getConfigValue("user.email")
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.email: %w", err)
	}

	return name, email, nil
}

// GenerateSSHCommand generates the SSH command for Git with the specified key
func (m *Manager) GenerateSSHCommand(sshKeyPath string) string {
	if sshKeyPath == "" {
		return ""
	}

	// Expand home directory if needed
	if strings.HasPrefix(sshKeyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			sshKeyPath = filepath.Join(homeDir, sshKeyPath[2:])
		}
	}

	// Check if the SSH key exists
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return ""
	}

	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", sshKeyPath)
}

// ValidateSSHKey checks if the SSH key file exists and is readable
func (m *Manager) ValidateSSHKey(sshKeyPath string) error {
	if sshKeyPath == "" {
		return nil // SSH key is optional
	}

	// Expand home directory if needed
	if strings.HasPrefix(sshKeyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		sshKeyPath = filepath.Join(homeDir, sshKeyPath[2:])
	}

	// Check if file exists
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		return models.ErrSSHKeyNotFound
	}

	// Check if file is readable
	file, err := os.Open(sshKeyPath)
	if err != nil {
		return fmt.Errorf("SSH key file is not readable: %w", err)
	}
	defer file.Close()

	return nil
}

// GetGitVersion returns the Git version
func (m *Manager) GetGitVersion() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", models.ErrGitNotFound
	}

	return strings.TrimSpace(string(output)), nil
}

// setUserName sets the Git user.name configuration
func (m *Manager) setUserName(name string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, "user.name", name)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git config failed: %w", err)
	}

	return nil
}

// setUserEmail sets the Git user.email configuration
func (m *Manager) setUserEmail(email string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, "user.email", email)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git config failed: %w", err)
	}

	return nil
}

// getConfigValue retrieves a Git configuration value
func (m *Manager) getConfigValue(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the remote URL for the current repository
func (m *Manager) GetRemoteURL(remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.Command("git", "config", "--get", fmt.Sprintf("remote.%s.url", remote))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current Git branch
func (m *Manager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
