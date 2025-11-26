package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

// Manager handles Git configuration operations
type Manager struct {
	useSSH bool
}

// NewManager creates a new Git manager
func NewManager() *Manager {
	return &Manager{
		useSSH: false, // Default to HTTPS for reliability
	}
}

// NewSSHManager creates a Git manager configured to use SSH
func NewSSHManager() *Manager {
	return &Manager{
		useSSH: true,
	}
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
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// SetLocalConfig sets the Git configuration for the current repository
func (m *Manager) SetLocalConfig(account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

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
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

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
	defer func() { _ = file.Close() }()

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
	// Always read global configuration to ensure consistency
	cmd := exec.Command("git", "config", "--global", "--get", key)
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

// SetRemoteURL sets the remote URL for the current repository
func (m *Manager) SetRemoteURL(remoteName, repoURL string) error {
	// Ensure we have the right protocol
	finalURL := m.normalizeURL(repoURL)

	cmd := exec.Command("git", "remote", "set-url", remoteName, finalURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set remote URL: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// normalizeURL ensures the URL uses the appropriate protocol
func (m *Manager) normalizeURL(url string) string {
	// Detect the platform from the URL
	domain := m.extractDomain(url)
	if domain == "" {
		// Unknown format, return as-is
		return url
	}

	// Extract the repo path from any format
	repoPath := m.extractRepoPath(url, domain)
	if repoPath == "" {
		return url
	}

	// Remove .git suffix if present
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Return in the appropriate format
	if m.useSSH {
		return fmt.Sprintf("git@%s:%s.git", domain, repoPath)
	} else {
		return fmt.Sprintf("https://%s/%s.git", domain, repoPath)
	}
}

// extractDomain extracts the domain from a Git URL
func (m *Manager) extractDomain(url string) string {
	// SSH format: git@domain:path
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			return strings.TrimPrefix(parts[0], "git@")
		}
	}

	// HTTPS format: https://domain/path
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		// Parse the URL to extract the host
		if idx := strings.Index(url[8:], "/"); idx != -1 {
			return url[8 : 8+idx]
		}
	}

	// Common Git hosting platforms
	commonDomains := []string{"github.com", "gitlab.com", "bitbucket.org"}
	for _, domain := range commonDomains {
		if strings.Contains(url, domain) {
			return domain
		}
	}

	return ""
}

// extractRepoPath extracts the repository path from a URL given a domain
func (m *Manager) extractRepoPath(url, domain string) string {
	// SSH format: git@domain:owner/repo.git
	if strings.HasPrefix(url, fmt.Sprintf("git@%s:", domain)) {
		return strings.TrimPrefix(url, fmt.Sprintf("git@%s:", domain))
	}

	// HTTPS format: https://domain/owner/repo.git
	if strings.HasPrefix(url, fmt.Sprintf("https://%s/", domain)) {
		return strings.TrimPrefix(url, fmt.Sprintf("https://%s/", domain))
	}

	if strings.HasPrefix(url, fmt.Sprintf("http://%s/", domain)) {
		return strings.TrimPrefix(url, fmt.Sprintf("http://%s/", domain))
	}

	return ""
}

// GetCurrentRemoteURL gets the current remote URL
func (m *Manager) GetCurrentRemoteURL(remoteName string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepository checks if the current directory is a git repository
func (m *Manager) IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// TestGitOperation tests if git operations work correctly
func (m *Manager) TestGitOperation() error {
	if !m.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Test basic git operation
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// SafeFetch performs a safe git fetch operation
func (m *Manager) SafeFetch(remoteName string) error {
	if !m.IsGitRepository() {
		return fmt.Errorf("not in a git repository")
	}

	// Use HTTPS for fetch to avoid SSH issues
	originalURL, err := m.GetCurrentRemoteURL(remoteName)
	if err != nil {
		return err
	}

	// Temporarily switch to HTTPS if using SSH
	httpsURL := m.convertToHTTPS(originalURL)
	if httpsURL != originalURL {
		// Switch to HTTPS
		if err := m.SetRemoteURL(remoteName, httpsURL); err != nil {
			return fmt.Errorf("failed to switch to HTTPS: %w", err)
		}

		// Restore original URL after fetch
		defer func() {
			_ = m.SetRemoteURL(remoteName, originalURL)
		}()
	}

	cmd := exec.Command("git", "fetch", remoteName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// convertToHTTPS converts any Git SSH URL to HTTPS format
func (m *Manager) convertToHTTPS(url string) string {
	// Extract domain and repo path
	domain := m.extractDomain(url)
	if domain == "" {
		return url
	}

	repoPath := m.extractRepoPath(url, domain)
	if repoPath == "" {
		return url
	}

	repoPath = strings.TrimSuffix(repoPath, ".git")
	return fmt.Sprintf("https://%s/%s.git", domain, repoPath)
}

// SetUserConfig sets the git user configuration
func (m *Manager) SetUserConfig(name, email string) error {
	if name != "" {
		cmd := exec.Command("git", "config", "--global", "user.name", name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
	}

	if email != "" {
		cmd := exec.Command("git", "config", "--global", "user.email", email)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
	}

	return nil
}

// GetUserConfig gets the current git user configuration
func (m *Manager) GetUserConfig() (name, email string, err error) {
	nameCmd := exec.Command("git", "config", "--global", "user.name")
	nameOutput, nameErr := nameCmd.Output()
	if nameErr == nil {
		name = strings.TrimSpace(string(nameOutput))
	}

	emailCmd := exec.Command("git", "config", "--global", "user.email")
	emailOutput, emailErr := emailCmd.Output()
	if emailErr == nil {
		email = strings.TrimSpace(string(emailOutput))
	}

	if nameErr != nil && emailErr != nil {
		return "", "", fmt.Errorf("failed to get git config: name=%v, email=%v", nameErr, emailErr)
	}

	return name, email, nil
}

// ClearSSHConfig removes problematic SSH configurations
func (m *Manager) ClearSSHConfig() error {
	// Remove global SSH command
	if err := exec.Command("git", "config", "--global", "--unset", "core.sshcommand").Run(); err != nil {
		log.Printf("Warning: failed to unset global git config: %v", err)
	}

	// Remove local SSH command
	if err := exec.Command("git", "config", "--local", "--unset", "core.sshcommand").Run(); err != nil {
		log.Printf("Warning: failed to unset local git config: %v", err)
	}

	// Remove any GIT_SSH_COMMAND environment variable
	if err := os.Unsetenv("GIT_SSH_COMMAND"); err != nil {
		log.Printf("Warning: failed to unset GIT_SSH_COMMAND: %v", err)
	}

	return nil
}

// SetGPGConfig sets the GPG signing configuration for Git
func (m *Manager) SetGPGConfig(account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	// Only configure GPG if the account has a GPG key
	if !account.HasGPGKey() {
		// No GPG key, disable signing
		return m.DisableGPGSigning()
	}

	// Set the signing key
	if err := m.setGPGSigningKey(account.GPGKeyID, true); err != nil {
		return fmt.Errorf("failed to set GPG signing key: %w", err)
	}

	// Enable or disable automatic signing based on account preferences
	if account.IsGPGEnabled() {
		if err := m.EnableGPGSigning(); err != nil {
			return fmt.Errorf("failed to enable GPG signing: %w", err)
		}
	} else {
		if err := m.DisableGPGSigning(); err != nil {
			return fmt.Errorf("failed to disable GPG signing: %w", err)
		}
	}

	return nil
}

// setGPGSigningKey sets the user.signingkey configuration
func (m *Manager) setGPGSigningKey(keyID string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, "user.signingkey", keyID)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git config failed: %w", err)
	}

	return nil
}

// EnableGPGSigning enables automatic GPG commit and tag signing
func (m *Manager) EnableGPGSigning() error {
	// Enable commit signing
	cmd := exec.Command("git", "config", "--global", "commit.gpgsign", "true")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable commit.gpgsign: %w", err)
	}

	// Enable tag signing
	cmd = exec.Command("git", "config", "--global", "tag.gpgsign", "true")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable tag.gpgsign: %w", err)
	}

	return nil
}

// DisableGPGSigning disables automatic GPG commit and tag signing
func (m *Manager) DisableGPGSigning() error {
	// Disable commit signing
	cmd := exec.Command("git", "config", "--global", "commit.gpgsign", "false")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to disable commit.gpgsign: %v", err)
	}

	// Disable tag signing
	cmd = exec.Command("git", "config", "--global", "tag.gpgsign", "false")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to disable tag.gpgsign: %v", err)
	}

	return nil
}

// UnsetGPGConfig removes GPG signing configuration from Git
func (m *Manager) UnsetGPGConfig() error {
	// Unset signing key
	if err := exec.Command("git", "config", "--global", "--unset", "user.signingkey").Run(); err != nil {
		log.Printf("Warning: failed to unset user.signingkey: %v", err)
	}

	// Disable signing
	return m.DisableGPGSigning()
}

// GetGPGConfig returns the current GPG configuration
func (m *Manager) GetGPGConfig() (signingKey string, commitSign, tagSign bool, err error) {
	// Get signing key
	cmd := exec.Command("git", "config", "--global", "--get", "user.signingkey")
	if output, err := cmd.Output(); err == nil {
		signingKey = strings.TrimSpace(string(output))
	}

	// Get commit.gpgsign
	cmd = exec.Command("git", "config", "--global", "--get", "commit.gpgsign")
	if output, err := cmd.Output(); err == nil {
		commitSign = strings.TrimSpace(string(output)) == "true"
	}

	// Get tag.gpgsign
	cmd = exec.Command("git", "config", "--global", "--get", "tag.gpgsign")
	if output, err := cmd.Output(); err == nil {
		tagSign = strings.TrimSpace(string(output)) == "true"
	}

	return signingKey, commitSign, tagSign, nil
}
