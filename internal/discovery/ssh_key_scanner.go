package discovery

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
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHKeyScanner handles discovery of SSH keys in ~/.ssh/
type SSHKeyScanner struct {
	homeDir string
	logger  observability.Logger
}

// NewSSHKeyScanner creates a new SSH key scanner
func NewSSHKeyScanner(logger observability.Logger) *SSHKeyScanner {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	return &SSHKeyScanner{
		homeDir: homeDir,
		logger:  logger,
	}
}

// SSHKeyInfo represents information about an SSH key
type SSHKeyInfo struct {
	PrivateKeyPath string
	PublicKeyPath  string
	KeyType        string
	Comment        string
	Fingerprint    string
	InSSHAgent     bool
	GitHubUsername string // Detected GitHub username
	IsOnGitHub     bool   // Whether this key is registered on GitHub
	AccountAlias   string // Suggested account alias
}

// DiscoverSSHKeys scans ~/.ssh/ directory for SSH keys and tries to determine
// which GitHub accounts they belong to
func (s *SSHKeyScanner) DiscoverSSHKeys(ctx context.Context) ([]*SSHKeyInfo, error) {
	s.logger.Info(ctx, "scanning_ssh_keys", observability.F("ssh_dir", filepath.Join(s.homeDir, ".ssh")))

	sshDir := filepath.Join(s.homeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("SSH directory not found: %s", sshDir)
	}

	// Find all SSH key files
	privateKeys, err := s.findSSHKeys(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find SSH keys: %w", err)
	}

	var keyInfos []*SSHKeyInfo

	for _, privateKeyPath := range privateKeys {
		keyInfo, err := s.analyzeSSHKey(ctx, privateKeyPath)
		if err != nil {
			s.logger.Warn(ctx, "failed_to_analyze_ssh_key",
				observability.F("path", privateKeyPath),
				observability.F("error", err.Error()))
			continue
		}

		keyInfos = append(keyInfos, keyInfo)
	}

	s.logger.Info(ctx, "ssh_keys_discovered", observability.F("count", len(keyInfos)))

	return keyInfos, nil
}

// findSSHKeys finds all private SSH key files in the SSH directory
func (s *SSHKeyScanner) findSSHKeys(sshDir string) ([]string, error) {
	files, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, err
	}

	var privateKeys []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		// Skip public keys, config files, and other non-private key files
		if strings.HasSuffix(name, ".pub") ||
			strings.HasSuffix(name, ".old") ||
			name == "config" ||
			name == "known_hosts" ||
			name == "authorized_keys" ||
			strings.HasPrefix(name, ".") {
			continue
		}

		// Check if this looks like a private key
		fullPath := filepath.Join(sshDir, name)
		if s.isPrivateSSHKey(fullPath) {
			privateKeys = append(privateKeys, fullPath)
		}
	}

	return privateKeys, nil
}

// isPrivateSSHKey checks if a file is likely a private SSH key
func (s *SSHKeyScanner) isPrivateSSHKey(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() {
		if err := file.Close(); err != nil {
			s.logger.Warn(context.Background(), "failed_to_close_file", observability.F("error", err.Error()))
		}
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return false
	}

	firstLine := scanner.Text()

	// Check for common private key headers
	privateKeyHeaders := []string{
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"-----BEGIN RSA PRIVATE KEY-----",
		"-----BEGIN DSA PRIVATE KEY-----",
		"-----BEGIN EC PRIVATE KEY-----",
		"-----BEGIN PRIVATE KEY-----",
	}

	for _, header := range privateKeyHeaders {
		if strings.Contains(firstLine, header) {
			return true
		}
	}

	return false
}

// analyzeSSHKey performs comprehensive analysis of an SSH key
func (s *SSHKeyScanner) analyzeSSHKey(ctx context.Context, privateKeyPath string) (*SSHKeyInfo, error) {
	keyInfo := &SSHKeyInfo{
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  privateKeyPath + ".pub",
	}

	// Extract basic key information
	if err := s.extractKeyInfo(keyInfo); err != nil {
		return nil, fmt.Errorf("failed to extract key info: %w", err)
	}

	// Determine GitHub username from key name or comment
	keyInfo.GitHubUsername = s.extractGitHubUsername(privateKeyPath, keyInfo.Comment)
	keyInfo.AccountAlias = s.generateAccountAlias(keyInfo.GitHubUsername, privateKeyPath)

	// Check if key is loaded in SSH agent
	keyInfo.InSSHAgent = s.isKeyInSSHAgent(privateKeyPath)

	// Test if key is registered on GitHub
	keyInfo.IsOnGitHub = s.testGitHubKeyRegistration(ctx, privateKeyPath, keyInfo.GitHubUsername)

	s.logger.Info(ctx, "ssh_key_analyzed",
		observability.F("path", privateKeyPath),
		observability.F("github_username", keyInfo.GitHubUsername),
		observability.F("account_alias", keyInfo.AccountAlias),
		observability.F("in_ssh_agent", keyInfo.InSSHAgent),
		observability.F("is_on_github", keyInfo.IsOnGitHub))

	return keyInfo, nil
}

// extractKeyInfo extracts basic information from the SSH key
func (s *SSHKeyScanner) extractKeyInfo(keyInfo *SSHKeyInfo) error {
	// Read public key to get type and comment
	if _, err := os.Stat(keyInfo.PublicKeyPath); err == nil {
		content, err := os.ReadFile(keyInfo.PublicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read public key: %w", err)
		}

		parts := strings.Fields(strings.TrimSpace(string(content)))
		if len(parts) >= 2 {
			keyInfo.KeyType = parts[0]
			if len(parts) >= 3 {
				keyInfo.Comment = strings.Join(parts[2:], " ")
			}
		}
	}

	// Get fingerprint
	fingerprint, err := s.getKeyFingerprint(keyInfo.PrivateKeyPath)
	if err == nil {
		keyInfo.Fingerprint = fingerprint
	}

	return nil
}

// extractGitHubUsername attempts to extract GitHub username from key path or comment
func (s *SSHKeyScanner) extractGitHubUsername(keyPath, comment string) string {
	// First, try to extract from the key filename
	filename := filepath.Base(keyPath)

	// Pattern 1: id_ed25519_username or id_rsa_username
	if strings.HasPrefix(filename, "id_ed25519_") {
		return strings.TrimPrefix(filename, "id_ed25519_")
	}
	if strings.HasPrefix(filename, "id_rsa_") {
		return strings.TrimPrefix(filename, "id_rsa_")
	}

	// Pattern 2: Just the username (like "costaar7", "thukabjj")
	if filename != "id_ed25519" && filename != "id_rsa" && filename != "id_dsa" {
		// Check if it looks like a username (alphanumeric, possibly with hyphens/underscores)
		usernamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if usernamePattern.MatchString(filename) {
			return filename
		}
	}

	// Pattern 3: Extract from comment (email or username)
	if comment != "" {
		// Check if comment contains an email with a username part that looks like GitHub username
		emailPattern := regexp.MustCompile(`([a-zA-Z0-9_-]+)@`)
		if matches := emailPattern.FindStringSubmatch(comment); len(matches) > 1 {
			return matches[1]
		}

		// Check if comment is just a username
		usernamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if usernamePattern.MatchString(comment) {
			return comment
		}
	}

	return ""
}

// generateAccountAlias generates a suggested account alias
func (s *SSHKeyScanner) generateAccountAlias(githubUsername, keyPath string) string {
	if githubUsername != "" {
		return githubUsername
	}

	// Fallback to key filename
	filename := filepath.Base(keyPath)
	filename = strings.TrimPrefix(filename, "id_ed25519_")
	filename = strings.TrimPrefix(filename, "id_rsa_")

	if filename == "id_ed25519" || filename == "id_rsa" || filename == "id_dsa" {
		return "default"
	}

	return filename
}

// getKeyFingerprint gets the SSH key fingerprint
func (s *SSHKeyScanner) getKeyFingerprint(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-lf", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Parse output: "2048 SHA256:... keypath (RSA)"
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}

	return "", fmt.Errorf("failed to parse fingerprint")
}

// isKeyInSSHAgent checks if the key is currently loaded in SSH agent
func (s *SSHKeyScanner) isKeyInSSHAgent(keyPath string) bool {
	// Get fingerprint of our key
	fingerprint, err := s.getKeyFingerprint(keyPath)
	if err != nil {
		return false
	}

	// List keys in SSH agent
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check if our fingerprint is in the agent
	return strings.Contains(string(output), fingerprint)
}

// testGitHubKeyRegistration tests if the SSH key is registered on GitHub
func (s *SSHKeyScanner) testGitHubKeyRegistration(ctx context.Context, keyPath, githubUsername string) bool {
	s.logger.Info(ctx, "testing_github_key_registration",
		observability.F("key_path", keyPath),
		observability.F("github_username", githubUsername))

	// Test direct GitHub SSH connection
	cmd := exec.Command("ssh", "-i", keyPath, "-T", "git@github.com", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=10")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	s.logger.Info(ctx, "github_ssh_test_result",
		observability.F("key_path", keyPath),
		observability.F("output", outputStr),
		observability.F("error", err))

	// GitHub SSH responds with a message like:
	// "Hi username! You've successfully authenticated, but GitHub does not provide shell access."
	if strings.Contains(outputStr, "successfully authenticated") {
		// Extract the actual GitHub username from the response
		re := regexp.MustCompile(`Hi ([^!]+)!`)
		if matches := re.FindStringSubmatch(outputStr); len(matches) > 1 {
			detectedUsername := matches[1]
			s.logger.Info(ctx, "github_username_detected",
				observability.F("key_path", keyPath),
				observability.F("detected_username", detectedUsername))
			return true
		}
		return true
	}

	return false
}

// CreateAccountsFromSSHKeys creates account configurations from discovered SSH keys
func (s *SSHKeyScanner) CreateAccountsFromSSHKeys(ctx context.Context, keys []*SSHKeyInfo) ([]*DiscoveredAccount, error) {
	var accounts []*DiscoveredAccount

	for _, key := range keys {
		if !key.IsOnGitHub {
			s.logger.Warn(ctx, "skipping_key_not_on_github", observability.F("path", key.PrivateKeyPath))
			continue
		}

		account := &DiscoveredAccount{
			Account: &models.Account{
				Alias:          key.AccountAlias,
				Name:           "", // Will be enriched later
				Email:          "", // Will be enriched later
				SSHKeyPath:     key.PrivateKeyPath,
				GitHubUsername: key.GitHubUsername,
				Description:    "Discovered SSH key for GitHub account",
			},
			Source:     "~/.ssh/ key discovery",
			Confidence: 9, // High confidence since key is verified on GitHub
		}

		// Try to enrich with GitHub API data if we have the username
		if key.GitHubUsername != "" {
			enrichedData := s.enrichFromGitHubAPI(ctx, key.GitHubUsername)
			if enrichedData.Name != "" {
				account.Account.Name = enrichedData.Name
			}
			if enrichedData.Email != "" {
				account.Account.Email = enrichedData.Email
			}
		}

		accounts = append(accounts, account)

		s.logger.Info(ctx, "account_created_from_ssh_key",
			observability.F("alias", account.Account.Alias),
			observability.F("github_username", account.Account.GitHubUsername),
			observability.F("ssh_key_path", account.Account.SSHKeyPath))
	}

	return accounts, nil
}

// enrichFromGitHubAPI tries to get additional user information from GitHub API
func (s *SSHKeyScanner) enrichFromGitHubAPI(ctx context.Context, username string) *models.Account {
	s.logger.Info(ctx, "enriching_from_github_api", observability.F("username", username))

	// Try to get user info from GitHub API using curl (more reliable than gh command)
	cmd := exec.Command("curl", "-s", fmt.Sprintf("https://api.github.com/users/%s", username))

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd = exec.CommandContext(timeoutCtx, "curl", "-s", fmt.Sprintf("https://api.github.com/users/%s", username))
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Warn(ctx, "github_api_request_failed",
			observability.F("username", username),
			observability.F("error", err.Error()))
		return &models.Account{}
	}

	// Simple parsing - look for name and email in JSON response
	outputStr := string(output)

	var name, email string

	// Extract name using regex (simple approach)
	nameRegex := regexp.MustCompile(`"name":\s*"([^"]*)"`)
	if matches := nameRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
		name = matches[1]
	}

	// Note: GitHub API doesn't expose email in public profile for most users
	// We'll leave email empty and let it be filled by other discovery methods

	s.logger.Info(ctx, "github_api_enrichment_result",
		observability.F("username", username),
		observability.F("name", name))

	return &models.Account{
		Name:  name,
		Email: email,
	}
}
