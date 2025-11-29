package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/techishthoughts/gitshift/internal/models"
)

// GPGScanner handles GPG key discovery from the system keyring
type GPGScanner struct {
	homeDir string
}

// NewGPGScanner creates a new GPG scanner
func NewGPGScanner() *GPGScanner {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	return &GPGScanner{
		homeDir: homeDir,
	}
}

// GPGKeyInfo represents discovered GPG key information
type GPGKeyInfo struct {
	KeyID       string
	Fingerprint string
	KeyType     string
	KeySize     int
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	Email       string
	Name        string
	Capabilities string // Signing capabilities
}

// ScanGPGKeys scans the GPG keyring for existing secret keys
func (s *GPGScanner) ScanGPGKeys() ([]*DiscoveredAccount, error) {
	var discovered []*DiscoveredAccount

	fmt.Println("🔐 Scanning GPG keyring for secret keys...")

	// Check if GPG is installed
	if !s.isGPGInstalled() {
		fmt.Println("⚠️  GPG not installed, skipping GPG key discovery")
		return discovered, nil
	}

	// List all secret keys (keys we can sign with)
	keys, err := s.listSecretKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Filter for signing-capable keys and create discovered accounts
	for _, key := range keys {
		// Only include keys that can sign (have 'S' capability)
		if !strings.Contains(key.Capabilities, "S") {
			continue
		}

		// Skip keys without email
		if key.Email == "" {
			continue
		}

		account := s.createAccountFromGPGKey(key)
		if account != nil {
			discovered = append(discovered, account)
		}
	}

	fmt.Printf("✅ Found %d GPG signing key(s)\n", len(discovered))
	return discovered, nil
}

// isGPGInstalled checks if GPG is available on the system
func (s *GPGScanner) isGPGInstalled() bool {
	cmd := exec.Command("gpg", "--version")
	err := cmd.Run()
	return err == nil
}

// listSecretKeys retrieves all secret (private) GPG keys from the keyring
func (s *GPGScanner) listSecretKeys() ([]GPGKeyInfo, error) {
	// Use --list-secret-keys to only get keys we can sign with
	// --with-colons for machine-readable output
	// --fingerprint to include fingerprints
	cmd := exec.Command("gpg", "--list-secret-keys", "--with-colons", "--fingerprint")
	output, err := cmd.Output()
	if err != nil {
		// If error is just "no secret keys", return empty list
		if strings.Contains(err.Error(), "exit status") {
			return []GPGKeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to execute gpg command: %w", err)
	}

	return s.parseGPGOutput(output)
}

// parseGPGOutput parses the colon-separated GPG output format
func (s *GPGScanner) parseGPGOutput(output []byte) ([]GPGKeyInfo, error) {
	var keys []GPGKeyInfo
	var currentKey *GPGKeyInfo

	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")

		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "sec": // Secret key (primary key)
			// Save previous key if exists
			if currentKey != nil && currentKey.KeyID != "" {
				keys = append(keys, *currentKey)
			}

			// Start new key
			currentKey = &GPGKeyInfo{}

			if len(fields) >= 12 {
				// Field 5 (index 4): Key ID
				currentKey.KeyID = fields[4]

				// Field 3 (index 2): Key length
				currentKey.KeySize = parseInt(fields[2])

				// Field 4 (index 3): Key algorithm
				currentKey.KeyType = s.mapKeyAlgorithm(fields[3])

				// Field 6 (index 5): Creation date (Unix timestamp)
				if fields[5] != "" {
					if timestamp := parseInt64(fields[5]); timestamp > 0 {
						currentKey.CreatedAt = time.Unix(timestamp, 0)
					}
				}

				// Field 7 (index 6): Expiration date (Unix timestamp)
				if fields[6] != "" {
					if timestamp := parseInt64(fields[6]); timestamp > 0 {
						expiresAt := time.Unix(timestamp, 0)
						currentKey.ExpiresAt = &expiresAt
					}
				}

				// Field 12 (index 11): Key capabilities
				if len(fields) > 11 {
					currentKey.Capabilities = fields[11]
				}
			}

		case "fpr": // Fingerprint
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}

		case "uid": // User ID (name and email)
			if currentKey != nil && len(fields) >= 10 {
				uidStr := fields[9]
				// Parse UID to extract name and email
				if name, email := s.parseUID(uidStr); email != "" {
					// Only set if not already set (use first UID)
					if currentKey.Email == "" {
						currentKey.Name = name
						currentKey.Email = email
					}
				}
			}
		}
	}

	// Add the last key
	if currentKey != nil && currentKey.KeyID != "" {
		keys = append(keys, *currentKey)
	}

	return keys, scanner.Err()
}

// createAccountFromGPGKey creates a discovered account from GPG key information
func (s *GPGScanner) createAccountFromGPGKey(key GPGKeyInfo) *DiscoveredAccount {
	if key.Email == "" {
		return nil
	}

	// Generate alias from email prefix
	alias := strings.Split(key.Email, "@")[0]

	// Determine confidence based on key information
	confidence := 7 // Base confidence for GPG keys

	// Higher confidence if key is recent (within last 2 years)
	if time.Since(key.CreatedAt) < 2*365*24*time.Hour {
		confidence = 8
	}

	// Lower confidence if key is expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		confidence = 5
	}

	status := "active"
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		status = "expired"
	}

	// Detect platform from email domain
	platform := s.detectPlatformFromEmail(key.Email)

	fmt.Printf("🔐 Found GPG key: %s -> %s (%s) [%s, %s, %s]\n",
		alias, key.Name, key.Email, key.KeyID[:16], status, platform)

	return &DiscoveredAccount{
		Account: &models.Account{
			Alias:              alias,
			Name:               key.Name,
			Email:              key.Email,
			GPGKeyID:           key.KeyID,
			GPGKeyFingerprint:  key.Fingerprint,
			GPGKeyType:         key.KeyType,
			GPGKeySize:         key.KeySize,
			GPGKeyExpiry:       key.ExpiresAt,
			GPGEnabled:         true,
			Platform:           platform,
			Description:        "Discovered from GPG keyring",
		},
		Source:     "gpg",
		Confidence: confidence,
	}
}

// detectPlatformFromEmail detects the Git platform based on email domain
func (s *GPGScanner) detectPlatformFromEmail(email string) string {
	if email == "" {
		return "github" // Default
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "github"
	}
	domain := strings.ToLower(parts[1])

	// Direct platform detection
	if strings.Contains(domain, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(domain, "github") {
		return "github"
	}
	if strings.Contains(domain, "bitbucket") {
		return "bitbucket"
	}

	// Corporate emails likely use GitLab or GitHub Enterprise
	commonProviders := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "icloud.com"}
	for _, provider := range commonProviders {
		if domain == provider {
			return "github" // Personal email defaults to GitHub
		}
	}

	// Corporate email - default to GitLab
	return "gitlab"
}

// parseUID extracts name and email from GPG UID string
// Example: "John Doe (comment) <john@example.com>"
func (s *GPGScanner) parseUID(uid string) (name, email string) {
	// Extract email using regex
	emailRegex := regexp.MustCompile(`<([^>]+)>`)
	if matches := emailRegex.FindStringSubmatch(uid); len(matches) >= 2 {
		email = matches[1]
	}

	// Extract name (everything before email or comment)
	nameRegex := regexp.MustCompile(`^([^(<]+)`)
	if matches := nameRegex.FindStringSubmatch(uid); len(matches) >= 2 {
		name = strings.TrimSpace(matches[1])
	}

	return name, email
}

// mapKeyAlgorithm maps GPG algorithm codes to human-readable names
func (s *GPGScanner) mapKeyAlgorithm(algo string) string {
	switch algo {
	case "1":
		return "RSA"
	case "17":
		return "DSA"
	case "22":
		return "EdDSA"
	default:
		return "Unknown"
	}
}

// parseInt parses a string to int
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// parseInt64 parses a string to int64
func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}
