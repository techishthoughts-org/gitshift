package validation

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
)

// SSHValidator handles comprehensive SSH key validation and troubleshooting
type SSHValidator struct {
	sshConfigPath string
	sshDir        string
	timeout       time.Duration
}

// NewSSHValidator creates a new SSH validator instance
func NewSSHValidator() *SSHValidator {
	homeDir, _ := os.UserHomeDir()
	return &SSHValidator{
		sshConfigPath: filepath.Join(homeDir, ".ssh", "config"),
		sshDir:        filepath.Join(homeDir, ".ssh"),
		timeout:       30 * time.Second,
	}
}

// ValidationResult contains the results of SSH validation
type ValidationResult struct {
	IsValid         bool              `json:"is_valid"`
	Issues          []ValidationIssue `json:"issues"`
	Recommendations []string          `json:"recommendations"`
	SSHKeys         []SSHKeyInfo      `json:"ssh_keys"`
	ConfigIssues    []SSHConfigIssue  `json:"config_issues"`
}

// ValidationIssue represents a specific validation problem
type ValidationIssue struct {
	Severity    string `json:"severity"` // "critical", "warning", "info"
	Category    string `json:"category"` // "authentication", "configuration", "security"
	Description string `json:"description"`
	Solution    string `json:"solution"`
	Code        string `json:"code"` // Example fix code
}

// SSHKeyInfo contains information about an SSH key
type SSHKeyInfo struct {
	Path        string   `json:"path"`
	Type        string   `json:"type"`
	Fingerprint string   `json:"fingerprint"`
	Email       string   `json:"email"`
	GitHubUser  string   `json:"github_user"`
	IsValid     bool     `json:"is_valid"`
	Issues      []string `json:"issues"`
}

// SSHConfigIssue represents SSH configuration problems
type SSHConfigIssue struct {
	Line        int    `json:"line"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Fix         string `json:"fix"`
}

// ValidateSSHConfiguration performs comprehensive SSH validation
func (v *SSHValidator) ValidateSSHConfiguration() (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:         true,
		Issues:          []ValidationIssue{},
		Recommendations: []string{},
		SSHKeys:         []SSHKeyInfo{},
		ConfigIssues:    []SSHConfigIssue{},
	}

	// 1. Validate SSH directory structure
	if err := v.validateSSHDirectory(result); err != nil {
		return result, err
	}

	// 2. Validate SSH configuration file
	if err := v.validateSSHConfig(result); err != nil {
		return result, err
	}

	// 3. Validate SSH keys
	if err := v.validateSSHKeys(result); err != nil {
		return result, err
	}

	// 4. Test GitHub authentication for each key
	if err := v.testGitHubAuthentication(result); err != nil {
		return result, err
	}

	// 5. Check for configuration conflicts
	if err := v.checkConfigurationConflicts(result); err != nil {
		return result, err
	}

	// 6. Generate recommendations
	v.generateRecommendations(result)

	return result, nil
}

// validateSSHDirectory checks SSH directory structure and permissions
func (v *SSHValidator) validateSSHDirectory(result *ValidationResult) error {
	// Check if SSH directory exists
	if _, err := os.Stat(v.sshDir); os.IsNotExist(err) {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity:    "critical",
			Category:    "configuration",
			Description: "SSH directory does not exist",
			Solution:    "Create SSH directory with proper permissions",
			Code:        "mkdir -p ~/.ssh && chmod 700 ~/.ssh",
		})
		result.IsValid = false
		return nil
	}

	// Check SSH directory permissions
	if info, err := os.Stat(v.sshDir); err == nil {
		if info.Mode().Perm()&0077 != 0 {
			result.Issues = append(result.Issues, ValidationIssue{
				Severity:    "critical",
				Category:    "security",
				Description: "SSH directory has overly permissive permissions",
				Solution:    "Restrict SSH directory permissions to owner only",
				Code:        "chmod 700 ~/.ssh",
			})
			result.IsValid = false
		}
	}

	return nil
}

// validateSSHConfig validates SSH configuration file
func (v *SSHValidator) validateSSHConfig(result *ValidationResult) error {
	if _, err := os.Stat(v.sshConfigPath); os.IsNotExist(err) {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity:    "warning",
			Category:    "configuration",
			Description: "SSH config file does not exist",
			Solution:    "Create SSH config file for better key management",
			Code:        "touch ~/.ssh/config && chmod 600 ~/.ssh/config",
		})
		return nil
	}

	file, err := os.Open(v.sshConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open SSH config: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	hostBlocks := make(map[string][]string)
	currentHost := ""

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for Host directive
		if strings.HasPrefix(line, "Host ") {
			currentHost = strings.TrimSpace(strings.TrimPrefix(line, "Host "))
			hostBlocks[currentHost] = []string{}
			continue
		}

		// Check for problematic configurations
		if currentHost == "github.com" && strings.Contains(line, "IdentityFile") {
			result.ConfigIssues = append(result.ConfigIssues, SSHConfigIssue{
				Line:        lineNum,
				Description: "Default github.com host configuration may override specific key selections",
				Severity:    "warning",
				Fix:         "Use specific host aliases (e.g., github-example) instead of default github.com",
			})
		}

		// Check for IdentitiesOnly setting
		if strings.Contains(line, "IdentitiesOnly") && !strings.Contains(line, "yes") {
			result.ConfigIssues = append(result.ConfigIssues, SSHConfigIssue{
				Line:        lineNum,
				Description: "IdentitiesOnly not set to 'yes' may cause key selection issues",
				Severity:    "info",
				Fix:         "Add 'IdentitiesOnly yes' to ensure only specified keys are used",
			})
		}

		if currentHost != "" {
			hostBlocks[currentHost] = append(hostBlocks[currentHost], line)
		}
	}

	// Check for missing host configurations
	if _, exists := hostBlocks["github.com"]; exists {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity:    "warning",
			Category:    "configuration",
			Description: "Default github.com host configuration detected",
			Solution:    "Consider using specific host aliases for different GitHub accounts",
			Code: `Host github-example
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_example
    IdentitiesOnly yes`,
		})
	}

	return scanner.Err()
}

// validateSSHKeys validates individual SSH keys
func (v *SSHValidator) validateSSHKeys(result *ValidationResult) error {
	// Find all SSH keys
	pattern := filepath.Join(v.sshDir, "id_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find SSH keys: %w", err)
	}

	for _, keyPath := range matches {
		// Skip public key files
		if strings.HasSuffix(keyPath, ".pub") {
			continue
		}

		keyInfo, err := v.analyzeSSHKey(keyPath)
		if err != nil {
			continue // Skip keys we can't analyze
		}

		result.SSHKeys = append(result.SSHKeys, keyInfo)
	}

	return nil
}

// analyzeSSHKey analyzes a single SSH key
func (v *SSHValidator) analyzeSSHKey(keyPath string) (SSHKeyInfo, error) {
	keyInfo := SSHKeyInfo{
		Path:    keyPath,
		IsValid: true,
		Issues:  []string{},
	}

	// Check key file permissions
	if info, err := os.Stat(keyPath); err == nil {
		if info.Mode().Perm()&0077 != 0 {
			keyInfo.Issues = append(keyInfo.Issues, "Key file has overly permissive permissions")
			keyInfo.IsValid = false
		}
	}

	// Get key fingerprint
	if fingerprint, err := v.getKeyFingerprint(keyPath); err == nil {
		keyInfo.Fingerprint = fingerprint
	} else {
		keyInfo.Issues = append(keyInfo.Issues, "Failed to get key fingerprint")
		keyInfo.IsValid = false
	}

	// Determine key type
	if keyType, err := v.getKeyType(keyPath); err == nil {
		keyInfo.Type = keyType
	} else {
		keyInfo.Issues = append(keyInfo.Issues, "Failed to determine key type")
		keyInfo.IsValid = false
	}

	// Extract email from key comment
	if email, err := v.extractEmailFromKey(keyPath); err == nil {
		keyInfo.Email = email
	} else {
		keyInfo.Issues = append(keyInfo.Issues, "No email found in key comment")
	}

	return keyInfo, nil
}

// getKeyFingerprint gets the fingerprint of an SSH key
func (v *SSHValidator) getKeyFingerprint(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse fingerprint from output
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}

	return "", fmt.Errorf("unexpected fingerprint format")
}

// getKeyType determines the type of SSH key
func (v *SSHValidator) getKeyType(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse key type from output
	parts := strings.Fields(string(output))
	if len(parts) >= 3 {
		return parts[2], nil
	}

	return "", fmt.Errorf("unexpected key type format")
}

// extractEmailFromKey extracts email from key comment
func (v *SSHValidator) extractEmailFromKey(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse email from output
	parts := strings.Fields(string(output))
	if len(parts) >= 4 {
		email := parts[3]
		// Validate email format
		if strings.Contains(email, "@") {
			return email, nil
		}
	}

	return "", fmt.Errorf("no valid email found")
}

// testGitHubAuthentication tests GitHub authentication for each key
func (v *SSHValidator) testGitHubAuthentication(result *ValidationResult) error {
	for i, keyInfo := range result.SSHKeys {
		// Test authentication with timeout
		githubUser, err := v.testGitHubKey(keyInfo.Path)
		if err != nil {
			keyInfo.Issues = append(keyInfo.Issues, fmt.Sprintf("GitHub authentication failed: %v", err))
			keyInfo.IsValid = false
		} else {
			keyInfo.GitHubUser = githubUser
		}

		// Update the key info in results
		result.SSHKeys[i] = keyInfo
	}

	return nil
}

// testGitHubKey tests a single SSH key with GitHub
func (v *SSHValidator) testGitHubKey(keyPath string) (string, error) {
	// Create a test command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", "-T", "git@github.com", "-i", keyPath, "-o", "IdentitiesOnly=yes")
	output, err := cmd.Output()

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("authentication timeout")
		}
		return "", err
	}

	// Parse GitHub username from output
	// Expected format: "Hi username! You've successfully authenticated..."
	re := regexp.MustCompile(`Hi (\w+)!`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("unexpected authentication response format")
}

// checkConfigurationConflicts checks for SSH configuration conflicts
func (v *SSHValidator) checkConfigurationConflicts(result *ValidationResult) error {
	// Check for duplicate key usage
	keyUsage := make(map[string][]string)
	for _, keyInfo := range result.SSHKeys {
		if keyInfo.GitHubUser != "" {
			keyUsage[keyInfo.GitHubUser] = append(keyUsage[keyInfo.GitHubUser], keyInfo.Path)
		}
	}

	// Report conflicts
	for user, keys := range keyUsage {
		if len(keys) > 1 {
			result.Issues = append(result.Issues, ValidationIssue{
				Severity:    "warning",
				Category:    "configuration",
				Description: fmt.Sprintf("Multiple SSH keys authenticating as %s", user),
				Solution:    "Ensure each key is associated with only one GitHub account",
				Code:        fmt.Sprintf("Keys: %s", strings.Join(keys, ", ")),
			})
		}
	}

	// Check for SSH agent key conflicts
	if err := v.checkSSHAgentConflicts(result); err != nil {
		return err
	}

	// Check for default github.com host configuration conflicts
	if err := v.checkDefaultGitHubHostConflicts(result); err != nil {
		return err
	}

	return nil
}

// checkSSHAgentConflicts checks for SSH agent key conflicts
func (v *SSHValidator) checkSSHAgentConflicts(result *ValidationResult) error {
	// Check if SSH agent is running and has multiple keys
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		// SSH agent not running or no keys - this is not necessarily a problem
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	keyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "The agent has no identities") {
			keyCount++
		}
	}

	if keyCount > 1 {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity:    "critical",
			Category:    "authentication",
			Description: fmt.Sprintf("SSH agent has %d keys loaded, which may cause authentication conflicts", keyCount),
			Solution:    "Clear SSH agent and load only the required key, or use SSH config with IdentitiesOnly",
			Code:        "ssh-add -D && ssh-add ~/.ssh/your_key",
		})
	}

	return nil
}

// checkDefaultGitHubHostConflicts checks for problematic default github.com configurations
func (v *SSHValidator) checkDefaultGitHubHostConflicts(result *ValidationResult) error {
	// Check if there's a default github.com configuration that might conflict
	if _, err := os.Stat(v.sshConfigPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(v.sshConfigPath)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	hasDefaultGitHubHost := false
	hasIdentitiesOnly := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for default github.com host
		if strings.HasPrefix(line, "Host ") {
			host := strings.TrimSpace(strings.TrimPrefix(line, "Host "))
			if host == "github.com" {
				hasDefaultGitHubHost = true
			}
		}

		// Check for IdentitiesOnly setting
		if strings.Contains(line, "IdentitiesOnly") && strings.Contains(line, "yes") {
			hasIdentitiesOnly = true
		}
	}

	// Report issues with default github.com configuration
	if hasDefaultGitHubHost && !hasIdentitiesOnly {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity:    "critical",
			Category:    "authentication",
			Description: "Default github.com host configuration without IdentitiesOnly may cause key selection conflicts",
			Solution:    "Add 'IdentitiesOnly yes' to your github.com host configuration or use specific host aliases",
			Code: `Host github.com
    HostName github.com
    User git
    IdentityFile ~/.ssh/your_key
    IdentitiesOnly yes`,
		})
	}

	return nil
}

// generateRecommendations generates improvement recommendations
func (v *SSHValidator) generateRecommendations(result *ValidationResult) {
	if !result.IsValid {
		result.Recommendations = append(result.Recommendations,
			"Fix critical issues before proceeding with GitHub operations")
	}

	// Check for missing IdentitiesOnly settings
	hasIdentitiesOnly := false
	for _, keyInfo := range result.SSHKeys {
		if keyInfo.IsValid {
			hasIdentitiesOnly = true
			break
		}
	}

	if !hasIdentitiesOnly {
		result.Recommendations = append(result.Recommendations,
			"Add 'IdentitiesOnly yes' to SSH config to prevent key selection conflicts")
	}

	// Check for host alias usage
	if len(result.ConfigIssues) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Use specific host aliases (e.g., github-example) instead of default github.com")
	}

	// Security recommendations
	result.Recommendations = append(result.Recommendations,
		"Ensure all SSH keys have restricted permissions (600)",
		"Use ED25519 keys for better security and performance",
		"Regularly rotate SSH keys and remove unused ones")
}

// GenerateSSHConfig generates a recommended SSH configuration
func (v *SSHValidator) GenerateSSHConfig(accounts []models.Account) string {
	var config strings.Builder

	config.WriteString("# GitPersona Generated SSH Configuration\n")
	config.WriteString("# Generated on: " + time.Now().Format(time.RFC3339) + "\n")
	config.WriteString("# This configuration prevents SSH key conflicts by using specific host aliases\n\n")

	// Add a default github.com configuration if there's only one account
	if len(accounts) == 1 && accounts[0].SSHKeyPath != "" {
		account := accounts[0]
		config.WriteString("# Default GitHub configuration (single account)\n")
		config.WriteString("Host github.com\n")
		config.WriteString("    HostName github.com\n")
		config.WriteString("    User git\n")
		config.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
		config.WriteString("    IdentitiesOnly yes\n")
		config.WriteString("    UseKeychain yes\n")
		config.WriteString("    AddKeysToAgent yes\n")
		config.WriteString("    ServerAliveInterval 60\n")
		config.WriteString("    ServerAliveCountMax 3\n\n")
	} else if len(accounts) > 1 {
		// For multiple accounts, use specific host aliases
		config.WriteString("# Multiple GitHub accounts - use specific host aliases\n")
		config.WriteString("# Usage: git clone git@github-work:user/repo.git\n\n")

		for _, account := range accounts {
			if account.SSHKeyPath == "" {
				continue
			}

			hostAlias := fmt.Sprintf("github-%s", account.Alias)
			config.WriteString(fmt.Sprintf("# %s GitHub Account (%s)\n", account.Name, account.GitHubUsername))
			config.WriteString(fmt.Sprintf("Host %s\n", hostAlias))
			config.WriteString("    HostName github.com\n")
			config.WriteString("    User git\n")
			config.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
			config.WriteString("    IdentitiesOnly yes\n")
			config.WriteString("    UseKeychain yes\n")
			config.WriteString("    AddKeysToAgent yes\n")
			config.WriteString("    ServerAliveInterval 60\n")
			config.WriteString("    ServerAliveCountMax 3\n\n")
		}
	}

	return config.String()
}

// GenerateSSHConfigForAccount generates SSH config for a specific account
func (v *SSHValidator) GenerateSSHConfigForAccount(account models.Account) string {
	var config strings.Builder

	config.WriteString("# SSH Configuration for " + account.Name + "\n")
	config.WriteString("# Generated on: " + time.Now().Format(time.RFC3339) + "\n\n")

	if account.SSHKeyPath == "" {
		config.WriteString("# No SSH key configured for this account\n")
		return config.String()
	}

	// Generate both default and alias configurations
	config.WriteString("# Default github.com configuration\n")
	config.WriteString("Host github.com\n")
	config.WriteString("    HostName github.com\n")
	config.WriteString("    User git\n")
	config.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
	config.WriteString("    IdentitiesOnly yes\n")
	config.WriteString("    UseKeychain yes\n")
	config.WriteString("    AddKeysToAgent yes\n")
	config.WriteString("    ServerAliveInterval 60\n")
	config.WriteString("    ServerAliveCountMax 3\n\n")

	// Also generate an alias for explicit usage
	hostAlias := fmt.Sprintf("github-%s", account.Alias)
	config.WriteString(fmt.Sprintf("# Alias for explicit usage: git@%s\n", hostAlias))
	config.WriteString(fmt.Sprintf("Host %s\n", hostAlias))
	config.WriteString("    HostName github.com\n")
	config.WriteString("    User git\n")
	config.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
	config.WriteString("    IdentitiesOnly yes\n")
	config.WriteString("    UseKeychain yes\n")
	config.WriteString("    AddKeysToAgent yes\n")
	config.WriteString("    ServerAliveInterval 60\n")
	config.WriteString("    ServerAliveCountMax 3\n\n")

	return config.String()
}

// FixSSHPermissions fixes SSH file permissions
func (v *SSHValidator) FixSSHPermissions() error {
	// Fix SSH directory permissions
	if err := os.Chmod(v.sshDir, 0700); err != nil {
		return fmt.Errorf("failed to fix SSH directory permissions: %w", err)
	}

	// Fix SSH config permissions
	if _, err := os.Stat(v.sshConfigPath); err == nil {
		if err := os.Chmod(v.sshConfigPath, 0600); err != nil {
			return fmt.Errorf("failed to fix SSH config permissions: %w", err)
		}
	}

	// Fix private key permissions
	pattern := filepath.Join(v.sshDir, "id_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find SSH keys: %w", err)
	}

	for _, keyPath := range matches {
		// Skip public key files
		if strings.HasSuffix(keyPath, ".pub") {
			continue
		}

		if err := os.Chmod(keyPath, 0600); err != nil {
			return fmt.Errorf("failed to fix key permissions for %s: %w", keyPath, err)
		}
	}

	return nil
}
