package gpg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Manager handles GPG key generation and management
type Manager struct {
	homeDir string
	gpgDir  string
}

// NewManager creates a new GPG manager
func NewManager() *Manager {
	// Try os.UserHomeDir() first (cross-platform)
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		// Fallback to HOME environment variable
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			// Last resort: use current directory (will likely fail, but better than "~")
			homeDir = "."
		}
	}

	return &Manager{
		homeDir: homeDir,
		gpgDir:  filepath.Join(homeDir, ".gnupg"),
	}
}

// GenerateKeyParams contains parameters for GPG key generation
type GenerateKeyParams struct {
	Alias      string
	Name       string
	Email      string
	KeyType    string // "RSA", "ECC", "DSA"
	KeyLength  int    // 2048, 4096 for RSA
	Passphrase string
	ExpireDate string // "0" for no expiration, "1y", "2y", etc.
	Force      bool
}

// KeyInfo contains information about a generated GPG key
type KeyInfo struct {
	KeyID       string
	Fingerprint string
	KeyType     string
	KeySize     int
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	Email       string
	Name        string
}

// GenerateKey generates a new GPG key with the specified parameters
func (m *Manager) GenerateKey(params GenerateKeyParams) (*KeyInfo, error) {
	// Ensure GPG directory exists
	if err := os.MkdirAll(m.gpgDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create GPG directory: %w", err)
	}

	// Validate parameters
	if err := m.validateKeyParams(params); err != nil {
		return nil, err
	}

	// Check if GPG is installed
	if err := m.checkGPGInstalled(); err != nil {
		return nil, err
	}

	// Build the GPG batch parameter file content
	batchParams := m.buildBatchParams(params)

	// Warn about passphrase security if using passphrase
	if params.Passphrase != "" {
		fmt.Printf("⚠️  Warning: Passphrase will be temporarily stored in a file during key generation\n")
		fmt.Printf("   For maximum security, consider using passwordless keys or manual GPG key generation\n")
	}

	fmt.Printf("🔧 Generating %s GPG key with %d bits...\n", params.KeyType, params.KeyLength)

	// Create a temporary file for batch parameters with secure permissions
	tmpFile, err := os.CreateTemp("", "gpg-batch-*.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Set restrictive permissions immediately (owner read/write only)
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to set secure permissions on temp file: %w", err)
	}

	// Securely clean up temp file
	defer func() {
		// Overwrite sensitive data before deletion if passphrase was used
		if params.Passphrase != "" {
			if f, err := os.OpenFile(tmpFile.Name(), os.O_WRONLY, 0600); err == nil {
				// Overwrite with zeros
				zeros := make([]byte, 4096)
				f.WriteAt(zeros, 0)
				f.Close()
			}
		}
		os.Remove(tmpFile.Name())
	}()

	// Write batch parameters to temp file
	if _, err := tmpFile.WriteString(batchParams); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write batch params: %w", err)
	}
	tmpFile.Close()

	// Execute gpg --batch --gen-key
	cmd := exec.Command("gpg", "--batch", "--gen-key", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("GPG key generation failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("✅ GPG key generated successfully\n")

	// Small delay to ensure key is fully written to keyring
	time.Sleep(100 * time.Millisecond)

	// Extract key ID from GPG output or keyring
	// To avoid race condition with multiple keys with same email,
	// we get the most recently created key
	keyInfo, err := m.getMostRecentKeyByEmail(params.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key info: %w", err)
	}

	return keyInfo, nil
}

// sanitizeGPGInput removes characters that could break GPG batch format
func sanitizeGPGInput(s string) string {
	// Remove newlines, carriage returns, and null bytes
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\x00", "")
	// Remove control characters (ASCII 0-31 except space)
	var result strings.Builder
	for _, r := range s {
		if r >= 32 || r == '\t' {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

// buildBatchParams creates the GPG batch parameter content
func (m *Manager) buildBatchParams(params GenerateKeyParams) string {
	var buf bytes.Buffer

	// Key type
	switch params.KeyType {
	case "RSA":
		buf.WriteString("Key-Type: RSA\n")
		buf.WriteString(fmt.Sprintf("Key-Length: %d\n", params.KeyLength))
		buf.WriteString("Subkey-Type: RSA\n")
		buf.WriteString(fmt.Sprintf("Subkey-Length: %d\n", params.KeyLength))
	case "ECC":
		buf.WriteString("Key-Type: EDDSA\n")
		buf.WriteString("Key-Curve: Ed25519\n")
		buf.WriteString("Subkey-Type: ECDH\n")
		buf.WriteString("Subkey-Curve: Cv25519\n")
	case "DSA":
		buf.WriteString("Key-Type: DSA\n")
		buf.WriteString("Key-Length: 2048\n")
		buf.WriteString("Subkey-Type: ELG-E\n")
		buf.WriteString("Subkey-Length: 2048\n")
	}

	// User identity - sanitize all user-provided input
	buf.WriteString(fmt.Sprintf("Name-Real: %s\n", sanitizeGPGInput(params.Name)))
	buf.WriteString(fmt.Sprintf("Name-Email: %s\n", sanitizeGPGInput(params.Email)))
	buf.WriteString(fmt.Sprintf("Name-Comment: gitshift-%s\n", sanitizeGPGInput(params.Alias)))

	// Expiration - validate and set
	if params.ExpireDate == "" {
		params.ExpireDate = "0" // No expiration by default
	}
	// Note: Validation happens in validateKeyParams, but double-check format
	if err := validateExpireDate(params.ExpireDate); err != nil {
		// Should not reach here if validateKeyParams was called, but be safe
		fmt.Printf("⚠️  Warning: Invalid expiration date format, using no expiration\n")
		params.ExpireDate = "0"
	}
	buf.WriteString(fmt.Sprintf("Expire-Date: %s\n", params.ExpireDate))

	// Passphrase handling
	if params.Passphrase == "" {
		// No passphrase - passwordless key
		buf.WriteString("%no-protection\n")
	} else {
		buf.WriteString(fmt.Sprintf("Passphrase: %s\n", params.Passphrase))
	}

	// Commit the parameters
	buf.WriteString("%commit\n")

	return buf.String()
}

// validateExpireDate validates the GPG expiration date format
func validateExpireDate(expireDate string) error {
	if expireDate == "" || expireDate == "0" {
		return nil // No expiration is valid
	}

	// GPG accepts formats: <n>, <n>w (weeks), <n>m (months), <n>y (years)
	// where n is a positive number
	matched, _ := regexp.MatchString(`^\d+(w|m|y)?$`, expireDate)
	if !matched {
		return fmt.Errorf("invalid expiration date format: %s (use: 0 for never, or 1y, 2m, 3w, etc.)", expireDate)
	}

	return nil
}

// validateKeyParams validates the key generation parameters
func (m *Manager) validateKeyParams(params GenerateKeyParams) error {
	if params.Name == "" {
		return fmt.Errorf("name is required")
	}

	if params.Email == "" {
		return fmt.Errorf("email is required")
	}

	// Validate expiration date format
	if err := validateExpireDate(params.ExpireDate); err != nil {
		return err
	}

	// Validate key type
	switch params.KeyType {
	case "RSA":
		if params.KeyLength < 2048 {
			return fmt.Errorf("RSA key size must be at least 2048 bits")
		}
		if params.KeyLength != 2048 && params.KeyLength != 4096 {
			fmt.Printf("⚠️  Warning: Recommended RSA key sizes are 2048 or 4096 bits\n")
		}
	case "ECC":
		// ECC keys have fixed sizes based on curve (Ed25519 = 256 bits equivalent)
		// Note: KeyLength is ignored for ECC, curve determines the actual size
		// We don't modify params.KeyLength to avoid mutating input
	case "DSA":
		if params.KeyLength < 2048 {
			return fmt.Errorf("DSA key size must be at least 2048 bits")
		}
	default:
		return fmt.Errorf("unsupported key type: %s (supported: RSA, ECC, DSA)", params.KeyType)
	}

	return nil
}

// checkGPGInstalled checks if GPG is installed and available
func (m *Manager) checkGPGInstalled() error {
	cmd := exec.Command("gpg", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("GPG is not installed. Please install GPG:\n" +
			"  macOS:    brew install gnupg\n" +
			"  Ubuntu:   sudo apt install gnupg\n" +
			"  Fedora:   sudo dnf install gnupg2\n" +
			"  Windows:  Download from https://gpg4win.org")
	}

	// Extract version for display
	versionLine := strings.Split(string(output), "\n")[0]
	fmt.Printf("📦 Using %s\n", versionLine)

	return nil
}

// getKeyInfoByEmail retrieves key information by email address
func (m *Manager) getKeyInfoByEmail(email string) (*KeyInfo, error) {
	// List keys with the specified email
	cmd := exec.Command("gpg", "--list-keys", "--with-colons", email)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	return m.parseKeyInfo(output)
}

// getMostRecentKeyByEmail retrieves the most recently created key for an email
// This avoids race conditions when multiple keys exist with the same email
func (m *Manager) getMostRecentKeyByEmail(email string) (*KeyInfo, error) {
	// List all keys with the specified email
	cmd := exec.Command("gpg", "--list-keys", "--with-colons", email)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Parse all keys
	keys, err := m.parseAllKeys(output)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys found for email: %s", email)
	}

	// Find the most recently created key
	mostRecent := keys[0]
	for _, key := range keys[1:] {
		if key.CreatedAt.After(mostRecent.CreatedAt) {
			mostRecent = key
		}
	}

	return &mostRecent, nil
}

// parseAllKeys parses multiple keys from GPG output
func (m *Manager) parseAllKeys(output []byte) ([]KeyInfo, error) {
	var keys []KeyInfo
	var currentKey *KeyInfo

	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")

		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub": // Start of a new public key
			// Save previous key if exists
			if currentKey != nil && currentKey.KeyID != "" {
				keys = append(keys, *currentKey)
			}
			// Start new key
			currentKey = &KeyInfo{}
			if len(fields) >= 5 {
				// Field index 4: Key ID
				currentKey.KeyID = fields[4]

				// Field index 5: Creation date (Unix timestamp)
				if fields[5] != "" {
					if timestamp := parseInt64(fields[5]); timestamp > 0 {
						currentKey.CreatedAt = time.Unix(timestamp, 0)
					}
				}

				// Field index 6: Expiration date (Unix timestamp)
				if len(fields) > 6 && fields[6] != "" {
					if timestamp := parseInt64(fields[6]); timestamp > 0 {
						expiresAt := time.Unix(timestamp, 0)
						currentKey.ExpiresAt = &expiresAt
					}
				}

				// Field index 2: Key length
				if len(fields) > 2 {
					currentKey.KeySize = parseInt(fields[2])
				}

				// Field index 3: Key algorithm
				if len(fields) > 3 {
					currentKey.KeyType = m.mapKeyAlgorithm(fields[3])
				}
			}

		case "fpr": // Fingerprint
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}

		case "uid": // User ID
			if currentKey != nil && len(fields) >= 10 {
				uidStr := fields[9]
				if name, email := m.parseUID(uidStr); name != "" {
					currentKey.Name = name
					currentKey.Email = email
				}
			}
		}
	}

	// Add the last key
	if currentKey != nil && currentKey.KeyID != "" {
		keys = append(keys, *currentKey)
	}

	return keys, nil
}

// parseKeyInfo parses GPG key information from colon-separated output
func (m *Manager) parseKeyInfo(output []byte) (*KeyInfo, error) {
	info := &KeyInfo{}
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")

		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub": // Public key
			if len(fields) >= 5 {
				// Field 5 (index 4): Key ID
				info.KeyID = fields[4]

				// Field 6 (index 5): Creation date (Unix timestamp)
				if len(fields) > 5 && fields[5] != "" {
					if timestamp := parseInt64(fields[5]); timestamp > 0 {
						info.CreatedAt = time.Unix(timestamp, 0)
					}
				}

				// Field 7 (index 6): Expiration date (Unix timestamp)
				if len(fields) > 6 && fields[6] != "" {
					if timestamp := parseInt64(fields[6]); timestamp > 0 {
						expiresAt := time.Unix(timestamp, 0)
						info.ExpiresAt = &expiresAt
					}
				}

				// Field 3 (index 2): Key length
				if len(fields) > 2 {
					info.KeySize = parseInt(fields[2])
				}

				// Field 4 (index 3): Key algorithm
				if len(fields) > 3 {
					info.KeyType = m.mapKeyAlgorithm(fields[3])
				}
			}

		case "fpr": // Fingerprint
			if len(fields) >= 10 {
				info.Fingerprint = fields[9]
			}

		case "uid": // User ID
			if len(fields) >= 10 {
				uidStr := fields[9]
				// Extract name and email from UID string
				if name, email := m.parseUID(uidStr); name != "" {
					info.Name = name
					info.Email = email
				}
			}
		}
	}

	if info.KeyID == "" {
		return nil, fmt.Errorf("failed to extract key ID from GPG output")
	}

	return info, nil
}

// parseUID extracts name and email from GPG UID string
// Example: "John Doe (gitshift-work) <john@company.com>"
func (m *Manager) parseUID(uid string) (name, email string) {
	// Extract email
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
func (m *Manager) mapKeyAlgorithm(algo string) string {
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

// ExportPublicKey exports the public key in ASCII armor format
func (m *Manager) ExportPublicKey(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--armor", "--export", keyID)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to export public key: %w", err)
	}

	return string(output), nil
}

// SavePublicKeyToFile exports the public key to a file
func (m *Manager) SavePublicKeyToFile(keyID, alias string) (string, error) {
	pubKey, err := m.ExportPublicKey(keyID)
	if err != nil {
		return "", err
	}

	// Save to ~/.gnupg/gitshift-{alias}-public.asc
	filename := filepath.Join(m.gpgDir, fmt.Sprintf("gitshift-%s-public.asc", alias))
	if err := os.WriteFile(filename, []byte(pubKey), 0644); err != nil {
		return "", fmt.Errorf("failed to save public key: %w", err)
	}

	return filename, nil
}

// ListKeys lists all GPG keys in the keyring
func (m *Manager) ListKeys() ([]KeyInfo, error) {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	var keys []KeyInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var currentKey *KeyInfo

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")

		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub": // Start of a new public key
			if currentKey != nil && currentKey.KeyID != "" {
				keys = append(keys, *currentKey)
			}
			currentKey = &KeyInfo{}
			if len(fields) >= 12 {
				currentKey.KeyID = fields[4]
				currentKey.KeySize = parseInt(fields[2])
				currentKey.KeyType = m.mapKeyAlgorithm(fields[3])

				if fields[5] != "" {
					if timestamp := parseInt64(fields[5]); timestamp > 0 {
						currentKey.CreatedAt = time.Unix(timestamp, 0)
					}
				}

				if fields[6] != "" {
					if timestamp := parseInt64(fields[6]); timestamp > 0 {
						expiresAt := time.Unix(timestamp, 0)
						currentKey.ExpiresAt = &expiresAt
					}
				}
			}

		case "fpr": // Fingerprint
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}

		case "uid": // User ID
			if currentKey != nil && len(fields) >= 10 {
				uidStr := fields[9]
				if name, email := m.parseUID(uidStr); name != "" {
					currentKey.Name = name
					currentKey.Email = email
				}
			}
		}
	}

	// Add the last key
	if currentKey != nil && currentKey.KeyID != "" {
		keys = append(keys, *currentKey)
	}

	return keys, nil
}

// DeleteKey deletes a GPG key from the keyring
func (m *Manager) DeleteKey(keyID string, deleteSecret bool) error {
	// Delete secret key first if requested
	if deleteSecret {
		cmd := exec.Command("gpg", "--batch", "--yes", "--delete-secret-keys", keyID)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to delete secret key: %w", err)
		}
	}

	// Delete public key
	cmd := exec.Command("gpg", "--batch", "--yes", "--delete-keys", keyID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	return nil
}

// GetKeyFingerprint retrieves the fingerprint for a key ID
func (m *Manager) GetKeyFingerprint(keyID string) (string, error) {
	cmd := exec.Command("gpg", "--with-colons", "--fingerprint", keyID)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get key fingerprint: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) >= 10 && fields[0] == "fpr" {
			return fields[9], nil
		}
	}

	return "", fmt.Errorf("fingerprint not found for key ID: %s", keyID)
}

// TestGPGKey tests if a GPG key can be used for signing
func (m *Manager) TestGPGKey(keyID string) error {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "gpg-test-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer os.Remove(tmpFile.Name() + ".asc")

	testContent := []byte("gitshift GPG test signature")
	if _, err := tmpFile.Write(testContent); err != nil {
		return fmt.Errorf("failed to write test content: %w", err)
	}
	tmpFile.Close()

	// Try to sign the file
	cmd := exec.Command("gpg", "--batch", "--yes", "--local-user", keyID, "--armor", "--detach-sign", tmpFile.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GPG signing test failed: %w", err)
	}

	fmt.Printf("✅ GPG key %s can sign successfully\n", keyID)
	return nil
}
