package discovery

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
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
	output  io.Writer
}

// NewGPGScanner creates a new GPG scanner
func NewGPGScanner() *GPGScanner {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	return &GPGScanner{
		homeDir: homeDir,
		output:  os.Stdout,
	}
}

// GPGKeyInfo represents discovered GPG key information
type GPGKeyInfo struct {
	KeyID        string
	Fingerprint  string
	KeyType      string
	KeySize      int
	CreatedAt    time.Time
	ExpiresAt    *time.Time
	Email        string
	Name         string
	Capabilities string // Signing capabilities
}

// ScanGPGKeys scans the GPG keyring for existing secret keys
func (s *GPGScanner) ScanGPGKeys() ([]*DiscoveredAccount, error) {
	return s.ScanGPGKeysContext(context.Background())
}

// ScanGPGKeysContext is the cancelable form used by interactive clients.
func (s *GPGScanner) ScanGPGKeysContext(ctx context.Context) ([]*DiscoveredAccount, error) {
	if s.output == nil {
		s.output = io.Discard
	}
	var discovered []*DiscoveredAccount

	fmt.Fprintln(s.output, "🔐 Scanning GPG keyring for secret keys...")

	// Check if GPG is installed
	if !s.isGPGInstalledContext(ctx) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fmt.Fprintln(s.output, "⚠️  GPG not installed, skipping GPG key discovery")
		return discovered, nil
	}

	// List all secret keys (keys we can sign with)
	keys, err := s.listSecretKeysContext(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Filter for signing-capable keys and create discovered accounts
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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

	fmt.Fprintf(s.output, "✅ Found %d GPG signing key(s)\n", len(discovered))
	return discovered, nil
}

// isGPGInstalled checks if GPG is available on the system
func (s *GPGScanner) isGPGInstalled() bool {
	return s.isGPGInstalledContext(context.Background())
}

func (s *GPGScanner) isGPGInstalledContext(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "gpg", "--version")
	err := cmd.Run()
	return err == nil
}

// listSecretKeys retrieves all secret (private) GPG keys from the keyring
func (s *GPGScanner) listSecretKeys() ([]GPGKeyInfo, error) {
	return s.listSecretKeysContext(context.Background())
}

func (s *GPGScanner) listSecretKeysContext(ctx context.Context) ([]GPGKeyInfo, error) {
	// Use --list-secret-keys to only get keys we can sign with
	// --with-colons for machine-readable output
	// --fingerprint to include fingerprints
	cmd := exec.CommandContext(ctx, "gpg", "--list-secret-keys", "--with-colons", "--fingerprint")
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

			// Need at least 5 fields to get KeyID (field index 4)
			if len(fields) >= 5 {
				// Field 5 (index 4): Key ID
				if fields[4] != "" {
					currentKey.KeyID = fields[4]
				}

				// Field 3 (index 2): Key length
				if len(fields) > 2 && fields[2] != "" {
					currentKey.KeySize = parseInt(fields[2])
				}

				// Field 4 (index 3): Key algorithm
				if len(fields) > 3 && fields[3] != "" {
					currentKey.KeyType = s.mapKeyAlgorithm(fields[3])
				}

				// Field 6 (index 5): Creation date (Unix timestamp)
				if len(fields) > 5 && fields[5] != "" {
					if timestamp := parseInt64(fields[5]); timestamp > 0 {
						currentKey.CreatedAt = time.Unix(timestamp, 0)
					}
				}

				// Field 7 (index 6): Expiration date (Unix timestamp)
				if len(fields) > 6 && fields[6] != "" {
					if timestamp := parseInt64(fields[6]); timestamp > 0 {
						expiresAt := time.Unix(timestamp, 0)
						currentKey.ExpiresAt = &expiresAt
					}
				}

				// Field 12 (index 11): Key capabilities
				if len(fields) > 11 && fields[11] != "" {
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

	// Safely display KeyID (truncate if longer than 16 chars, use full if shorter)
	keyIDDisplay := key.KeyID
	if len(key.KeyID) > 16 {
		keyIDDisplay = key.KeyID[:16]
	}

	fmt.Fprintf(s.output, "🔐 Found GPG key: %s -> %s (%s) [%s, %s, %s]\n",
		alias, key.Name, key.Email, keyIDDisplay, status, platform)

	return &DiscoveredAccount{
		Account: &models.Account{
			Alias:             alias,
			Name:              key.Name,
			Email:             key.Email,
			GPGKeyID:          key.KeyID,
			GPGKeyFingerprint: key.Fingerprint,
			GPGKeyType:        key.KeyType,
			GPGKeySize:        key.KeySize,
			GPGKeyExpiry:      key.ExpiresAt,
			GPGEnabled:        true,
			Platform:          platform,
			Description:       "Discovered from GPG keyring",
		},
		Source:     "gpg",
		Confidence: confidence,
	}
}

// detectPlatformFromEmail detects the Git platform based on email domain
func (s *GPGScanner) detectPlatformFromEmail(email string) string {
	return detectPlatform(email)
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
