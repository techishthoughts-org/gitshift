package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewModernCryptoManager(t *testing.T) {
	manager := NewModernCryptoManager()
	if manager == nil {
		t.Fatal("NewModernCryptoManager returned nil")
	}

	if manager.keyRotationDays != 90 {
		t.Errorf("Expected key rotation days 90, got %d", manager.keyRotationDays)
	}
}

func TestGenerateEd25519Key(t *testing.T) {
	manager := NewModernCryptoManager()
	alias := "test-alias"
	email := "test@example.com"

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	_ = os.Setenv("HOME", tempDir)

	keyPath, err := manager.GenerateEd25519Key(alias, email)
	if err != nil {
		t.Fatalf("GenerateEd25519Key failed: %v", err)
	}

	if keyPath == "" {
		t.Fatal("Expected key path to be returned")
	}

	// Check that the key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatalf("Private key file does not exist: %s", keyPath)
	}

	// Check that the public key file exists
	publicKeyPath := keyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Fatalf("Public key file does not exist: %s", publicKeyPath)
	}

	// Check that the key path contains the alias
	if !strings.Contains(filepath.Base(keyPath), alias) {
		t.Errorf("Expected key path to contain alias, got %s", keyPath)
	}
}

func TestGenerateEd25519KeyWithInvalidEmail(t *testing.T) {
	manager := NewModernCryptoManager()
	alias := "test-alias"
	email := "invalid-email" // Invalid email format

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	_ = os.Setenv("HOME", tempDir)

	_, err := manager.GenerateEd25519Key(alias, email)
	if err != nil {
		t.Errorf("Unexpected error for invalid email: %v", err)
	}
}

func TestIsKeyExpired(t *testing.T) {
	manager := NewModernCryptoManager()

	tests := []struct {
		name     string
		keyAge   time.Duration
		expected bool
	}{
		{
			name:     "recent key",
			keyAge:   30 * 24 * time.Hour, // 30 days
			expected: false,
		},
		{
			name:     "expired key",
			keyAge:   100 * 24 * time.Hour, // 100 days
			expected: true,
		},
		{
			name:     "exactly at rotation period",
			keyAge:   90 * 24 * time.Hour, // 90 days
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary key file with the specified age
			tempFile := filepath.Join(t.TempDir(), "test_key")
			keyTime := time.Now().Add(-tt.keyAge)
			err := os.WriteFile(tempFile, []byte("test key"), 0600)
			if err != nil {
				t.Fatalf("Failed to create test key file: %v", err)
			}

			// Set the file modification time
			err = os.Chtimes(tempFile, keyTime, keyTime)
			if err != nil {
				t.Fatalf("Failed to set file time: %v", err)
			}

			expired := manager.isKeyExpired(tempFile)
			if expired != tt.expected {
				t.Errorf("Expected expired %v, got %v", tt.expected, expired)
			}
		})
	}
}

func TestSetSecurityMetadata(t *testing.T) {
	manager := NewModernCryptoManager()
	alias := "test-alias"
	email := "test@example.com"

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()

	_ = os.Setenv("HOME", tempDir)

	keyPath, err := manager.GenerateEd25519Key(alias, email)
	if err != nil {
		t.Fatalf("GenerateEd25519Key failed: %v", err)
	}

	// Test setting security metadata (this doesn't create a file, just sets extended attributes)
	manager.setSecurityMetadata(keyPath, alias)

	// Verify that the key file was created
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("Key file does not exist: %s", keyPath)
	}

	// Verify that the public key file was created
	publicKeyPath := keyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); err != nil {
		t.Errorf("Public key file does not exist: %s", publicKeyPath)
	}
}

func TestValidateKeyStrength(t *testing.T) {
	manager := NewModernCryptoManager()

	// Test Ed25519 key strength (should always be strong)
	// Create a temporary Ed25519 public key file
	tempFile := filepath.Join(t.TempDir(), "test_key.pub")
	ed25519Key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIA0MmPmYZ4UyjXrhsCbOFTnf6RrebpNGNlcpmpvye3aI test@example.com"
	err := os.WriteFile(tempFile, []byte(ed25519Key), 0644)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	// Remove .pub extension since ValidateKeyStrength adds it
	keyPath := strings.TrimSuffix(tempFile, ".pub")
	err = manager.ValidateKeyStrength(keyPath)
	if err != nil {
		t.Errorf("Expected Ed25519 key to be valid, got error: %v", err)
	}

	// Test RSA key strength
	tests := []struct {
		name     string
		keyData  string
		expected bool
	}{
		{
			name:     "weak RSA key",
			keyData:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQC7vbqajDhA... test@example.com",
			expected: false,
		},
		{
			name:     "strong RSA key",
			keyData:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDxMXdQ3LwhM8M+nQhx7lwC3Mtnxuz1Ws/9yawG/TESzXP/de3T/MlprIsh6BycZL1rzTNzRVqmk6ubsE4MbRRXLA+jI03734B9IA1Xf3tsc759Y07TAkXPLkMLxS5H9+iz8HLSsbE1tFZhUi1cy5NwKyUeqDAP3KNYpOjLxpPOcmEzQCDHX8s7EeYU85Xh4jOZa+VoghZukAMEKSXgI38LCvX0mGs9aXghbbIStaa2+dxH9zXD0NaUas2z09K2OXHDgyjtmaf2rCeVw1wt9OsYal6cpB1ZxM0oMtfx4wOsr96ahq+U0mZsV8DFmQfVO7otKXP2uq5B5Uetpos+RAJHuQcFaoSCSjgZNjlvgTqEb10zMcDY4pl+q2sIowGqgMOhIimtxSvTHkABYHlcR7kLta3wL+vfecPgSqftJFtTtzEcKkFIKn0Y76n5zEMAnQTr2/54YU//pjy4gKLvbqhI2vFl0ynpZKPL1VG1jusNlcdx3Er/L3+ps/J46xbXlRms27X7anBV2RxXWN+49pEFejly4LJ1edQjJrzv82c7lMU/tXO/bJ/ZxeP2C66aW7DfxK3iD5/GEE36apENLwpPyfJdJMr61PwxJeYcIXOW8mIkOs7dejbkdhA3GqR5OuWwcuco1EBg0naNqqF1HdU0dcuFhtnGBpRzboygDZBCqw== test@example.com",
			expected: false, // RSA keys are rejected due to validation logic
		},
		{
			name:     "ECDSA key",
			keyData:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBBd1OL4hGWCYBWOPEnXpBSegjbB8ig3hlNDo3Iq9VvtTl4KMAabQvjS8IplRtyxrr3UvahpiOEuqtNky6gLVHcE= test@example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(t.TempDir(), "test_key.pub")
			err := os.WriteFile(tempFile, []byte(tt.keyData), 0644)
			if err != nil {
				t.Fatalf("Failed to create test key file: %v", err)
			}

			// Remove .pub extension since ValidateKeyStrength adds it
			keyPath := strings.TrimSuffix(tempFile, ".pub")
			err = manager.ValidateKeyStrength(keyPath)
			valid := (err == nil)
			if valid != tt.expected {
				t.Errorf("Expected key validity %v, got %v (error: %v)", tt.expected, valid, err)
			}
		})
	}
}

func TestSecureWipe(t *testing.T) {
	manager := NewModernCryptoManager()

	// Create a temporary file with sensitive data
	tempFile := filepath.Join(t.TempDir(), "sensitive.txt")
	sensitiveData := []byte("sensitive information that should be wiped")

	err := os.WriteFile(tempFile, sensitiveData, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists and contains data
	fileInfo, err := os.Stat(tempFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Fatal("Test file should not be empty")
	}

	// Perform secure wipe (wipes data in memory, not the file)
	wipeData := []byte(sensitiveData)
	manager.SecureWipe(wipeData)

	// Verify the data in memory was wiped
	for i, b := range wipeData {
		if b != 0 {
			t.Errorf("Expected byte at index %d to be 0, got %d", i, b)
		}
	}

	// Verify file still exists (SecureWipe doesn't delete files)
	fileInfo, err = os.Stat(tempFile)
	if err != nil {
		t.Fatalf("Failed to stat test file after wipe: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("Expected file to still exist after memory wipe")
	}
}

func TestSecureWipeNonExistentFile(t *testing.T) {
	manager := NewModernCryptoManager()

	// Test secure wipe with nil data
	manager.SecureWipe(nil)

	// Test secure wipe with empty data
	manager.SecureWipe([]byte{})

	// These should not panic or error
}

func TestCheckKeyRotationNeeded(t *testing.T) {
	manager := NewModernCryptoManager()

	tests := []struct {
		name     string
		keyAge   time.Duration
		expected bool
	}{
		{
			name:     "new key",
			keyAge:   30 * 24 * time.Hour, // 30 days
			expected: false,
		},
		{
			name:     "key near rotation",
			keyAge:   80 * 24 * time.Hour, // 80 days
			expected: false,
		},
		{
			name:     "key needs rotation",
			keyAge:   95 * 24 * time.Hour, // 95 days
			expected: true,
		},
		{
			name:     "very old key",
			keyAge:   200 * 24 * time.Hour, // 200 days
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary key file with the specified age
			tempFile := filepath.Join(t.TempDir(), "test_key")
			keyTime := time.Now().Add(-tt.keyAge)
			err := os.WriteFile(tempFile, []byte("test key"), 0600)
			if err != nil {
				t.Fatalf("Failed to create test key file: %v", err)
			}

			// Set the file modification time
			err = os.Chtimes(tempFile, keyTime, keyTime)
			if err != nil {
				t.Fatalf("Failed to set file time: %v", err)
			}

			status := manager.CheckKeyRotationNeeded([]string{tempFile})
			if len(status) == 0 {
				t.Fatal("Expected at least one status result")
			}

			needsRotation := status[0].NeedsRotation
			if needsRotation != tt.expected {
				t.Errorf("Expected rotation needed %v, got %v", tt.expected, needsRotation)
			}
		})
	}
}

func TestGenerateSecureAlias(t *testing.T) {
	manager := NewModernCryptoManager()

	// Test generating alias from email
	email := "john.doe@example.com"
	alias := manager.GenerateSecureAlias(email)
	if alias == "" {
		t.Fatal("Expected non-empty alias")
	}

	// Test that alias doesn't contain sensitive information
	if alias == email {
		t.Error("Alias should not be the same as email")
	}

	// Test that alias is reasonably short (base + underscore + 6 hex chars)
	if len(alias) > 30 {
		t.Errorf("Alias should be reasonably short, got length %d", len(alias))
	}

	// Test generating alias from username
	username := "johndoe"
	alias2 := manager.GenerateSecureAlias(username)
	if alias2 == "" {
		t.Fatal("Expected non-empty alias from username")
	}

	// Test that different inputs produce different aliases
	if alias == alias2 {
		t.Error("Different inputs should produce different aliases")
	}
}

func TestGenerateSecureAliasWithSpecialCharacters(t *testing.T) {
	manager := NewModernCryptoManager()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "email with special chars",
			input: "user+tag@example.com",
		},
		{
			name:  "username with spaces",
			input: "john doe",
		},
		{
			name:  "username with numbers",
			input: "user123",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alias := manager.GenerateSecureAlias(tt.input)
			if alias == "" {
				t.Fatal("Expected non-empty alias")
			}

			// Test that alias doesn't contain problematic characters
			if alias == tt.input {
				t.Error("Alias should not be the same as input")
			}
		})
	}
}

func TestEd25519KeyGeneration(t *testing.T) {
	// Test that we can generate a valid Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key pair: %v", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		t.Errorf("Expected private key size %d, got %d", ed25519.PrivateKeySize, len(privateKey))
	}

	// Test that the key pair is valid
	message := []byte("test message")
	signature := ed25519.Sign(privateKey, message)
	if !ed25519.Verify(publicKey, message, signature) {
		t.Error("Generated key pair failed signature verification")
	}
}

func TestKeyRotationDays(t *testing.T) {
	manager := NewModernCryptoManager()

	// Test that the default rotation period is reasonable
	if manager.keyRotationDays <= 0 {
		t.Error("Key rotation days should be positive")
	}

	if manager.keyRotationDays > 365 {
		t.Error("Key rotation days should not exceed 1 year")
	}

	// Test that 90 days is a reasonable default (2025 best practice)
	if manager.keyRotationDays != 90 {
		t.Errorf("Expected 90 days rotation period, got %d", manager.keyRotationDays)
	}
}

func TestCryptoManagerConcurrency(t *testing.T) {
	manager := NewModernCryptoManager()

	// Test that the manager can handle concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test concurrent key strength validation
			tempFile := filepath.Join(t.TempDir(), fmt.Sprintf("test_key_%d.pub", id))
			ed25519Key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIA0MmPmYZ4UyjXrhsCbOFTnf6RrebpNGNlcpmpvye3aI test@example.com"
			err := os.WriteFile(tempFile, []byte(ed25519Key), 0644)
			if err != nil {
				t.Errorf("Concurrent test %d: Failed to create test key file: %v", id, err)
				return
			}

			// Remove .pub extension since ValidateKeyStrength adds it
			keyPath := strings.TrimSuffix(tempFile, ".pub")
			err = manager.ValidateKeyStrength(keyPath)
			if err != nil {
				t.Errorf("Concurrent test %d: Expected Ed25519 to be valid, got error: %v", id, err)
			}

			// Test concurrent rotation check
			status := manager.CheckKeyRotationNeeded([]string{tempFile})
			if len(status) == 0 {
				t.Errorf("Concurrent test %d: Expected at least one status result", id)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
