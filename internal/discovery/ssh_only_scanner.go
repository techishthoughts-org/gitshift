package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/techishthoughts/gitshift/internal/models"
)

// SSHOnlyScanner handles SSH-only discovery without any GitHub API/CLI integration
type SSHOnlyScanner struct {
	homeDir string
}

// NewSSHOnlyScanner creates a new SSH-only scanner
func NewSSHOnlyScanner() *SSHOnlyScanner {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	return &SSHOnlyScanner{
		homeDir: homeDir,
	}
}

// ScanSSHKeys scans ~/.ssh for existing SSH keys and extracts account information
func (s *SSHOnlyScanner) ScanSSHKeys() ([]*DiscoveredAccount, error) {
	var discovered []*DiscoveredAccount

	fmt.Println("🔍 Scanning SSH keys in ~/.ssh directory (SSH-only mode)...")

	sshDir := filepath.Join(s.homeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		return discovered, nil
	}

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "id_") || strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())
		pubKeyPath := keyPath + ".pub"

		// Only process if both private and public key exist
		if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
			continue
		}

		// Extract information from SSH key
		account := s.createAccountFromSSHKey(keyPath, pubKeyPath)
		if account != nil {
			discovered = append(discovered, account)
		}
	}

	fmt.Printf("✅ Found %d SSH key(s)\n", len(discovered))
	return discovered, nil
}

// createAccountFromSSHKey creates a discovered account from SSH key information
func (s *SSHOnlyScanner) createAccountFromSSHKey(privateKeyPath, publicKeyPath string) *DiscoveredAccount {
	// Extract email from public key comment
	email := s.extractEmailFromPublicKey(publicKeyPath)
	if email == "" {
		fmt.Printf("⚠️  Skipping SSH key %s (no email found)\n", filepath.Base(privateKeyPath))
		return nil
	}

	// Extract username from filename (e.g., id_ed25519_costaar7 -> costaar7)
	filename := filepath.Base(privateKeyPath)
	username := s.extractUsernameFromKeyFilename(filename)

	// Generate display name from email
	name := s.generateNameFromEmail(email)

	// Generate alias (prefer username from filename, fallback to email prefix)
	alias := username
	if alias == "" {
		alias = strings.Split(email, "@")[0]
	}

	confidence := 8 // High confidence for SSH keys with email
	if username != "" {
		confidence = 9 // Even higher if we have clear username
	}

	fmt.Printf("🔑 Found SSH key: %s -> %s (%s)\n", alias, name, email)

	return &DiscoveredAccount{
		Account: &models.Account{
			Alias:          alias,
			Name:           name,
			Email:          email,
			GitHubUsername: username,
			SSHKeyPath:     privateKeyPath,
			Description:    "Discovered from SSH key",
		},
		Source:     "ssh",
		Confidence: confidence,
	}
}

// extractEmailFromPublicKey extracts email from SSH public key comment
func (s *SSHOnlyScanner) extractEmailFromPublicKey(pubKeyPath string) string {
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return ""
	}

	// SSH public key format: <key-type> <base64-key> <comment>
	parts := strings.Fields(strings.TrimSpace(string(content)))
	if len(parts) >= 3 {
		comment := parts[2]
		// Check if comment looks like an email
		if strings.Contains(comment, "@") && strings.Contains(comment, ".") {
			return comment
		}
	}

	return ""
}

// extractUsernameFromKeyFilename extracts username from SSH key filename
func (s *SSHOnlyScanner) extractUsernameFromKeyFilename(filename string) string {
	// Pattern: id_ed25519_username or id_rsa_username
	if strings.HasPrefix(filename, "id_ed25519_") {
		return strings.TrimPrefix(filename, "id_ed25519_")
	}
	if strings.HasPrefix(filename, "id_rsa_") {
		return strings.TrimPrefix(filename, "id_rsa_")
	}
	return ""
}

// generateNameFromEmail generates a display name from email address
func (s *SSHOnlyScanner) generateNameFromEmail(email string) string {
	if email == "" || !strings.Contains(email, "@") {
		return ""
	}

	// Create a title caser for English
	titleCaser := cases.Title(language.English)

	// Extract the part before @ symbol
	parts := strings.Split(email, "@")
	namePart := parts[0]

	// Convert email prefix to a readable name format
	if strings.Contains(namePart, ".") {
		// Split by dots and capitalize each part
		nameParts := strings.Split(namePart, ".")
		for i, part := range nameParts {
			nameParts[i] = titleCaser.String(strings.ToLower(part))
		}
		return strings.Join(nameParts, " ")
	} else {
		// Just capitalize the single part
		return titleCaser.String(strings.ToLower(namePart))
	}
}
