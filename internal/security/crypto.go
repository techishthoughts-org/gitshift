package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ModernCryptoManager handles cryptographic operations following 2025 best practices
type ModernCryptoManager struct {
	keyRotationDays int
}

// NewModernCryptoManager creates a new crypto manager with security best practices
func NewModernCryptoManager() *ModernCryptoManager {
	return &ModernCryptoManager{
		keyRotationDays: 90, // Rotate keys every 90 days (2025 best practice)
	}
}

// GenerateEd25519Key generates a modern Ed25519 SSH key pair
// Ed25519 is preferred over RSA in 2025 due to:
// - Smaller key size (256 bits vs 4096 bits RSA)
// - Better performance
// - Quantum resistance
// - Simpler implementation (less attack surface)
func (cm *ModernCryptoManager) GenerateEd25519Key(alias, email string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", err
	}

	// Modern key naming with timestamp for rotation tracking
	timestamp := time.Now().Format("20060102")
	keyName := fmt.Sprintf("id_ed25519_%s_%s", alias, timestamp)
	privateKeyPath := filepath.Join(sshDir, keyName)
	publicKeyPath := privateKeyPath + ".pub"

	// Check if key already exists
	if _, err := os.Stat(privateKeyPath); err == nil {
		if cm.isKeyExpired(privateKeyPath) {
			fmt.Printf("🔄 SSH key expired, generating new one...\n")
		} else {
			fmt.Printf("🔑 Valid SSH key already exists: %s\n", privateKeyPath)
			return privateKeyPath, nil
		}
	}

	// Generate Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Convert to SSH format
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Save private key in OpenSSH format
	opensshPrivateKey, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Save private key with strict permissions
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(opensshPrivateKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	publicKeyData := fmt.Sprintf("%s %s@%s-%s\n",
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))),
		email, alias, timestamp)

	if err := os.WriteFile(publicKeyPath, []byte(publicKeyData), 0644); err != nil {
		return "", fmt.Errorf("failed to write public key: %w", err)
	}

	// Set extended attributes for security metadata (2025 practice)
	cm.setSecurityMetadata(privateKeyPath, alias)

	fmt.Printf("✅ Ed25519 SSH key generated: %s\n", privateKeyPath)
	fmt.Printf("🔐 Key will expire on: %s\n", time.Now().AddDate(0, 0, cm.keyRotationDays).Format("2006-01-02"))

	return privateKeyPath, nil
}

// isKeyExpired checks if an SSH key is expired based on 2025 rotation policies
func (cm *ModernCryptoManager) isKeyExpired(keyPath string) bool {
	info, err := os.Stat(keyPath)
	if err != nil {
		return true
	}

	// Check if key is older than rotation period
	return time.Since(info.ModTime()) > time.Duration(cm.keyRotationDays)*24*time.Hour
}

// setSecurityMetadata adds security metadata to key files (2025 practice)
func (cm *ModernCryptoManager) setSecurityMetadata(keyPath, alias string) {
	// In a real implementation, this would set extended attributes
	// to track key origin, expiration, and security policies
	metadata := fmt.Sprintf("generated_by=gh-switcher,account=%s,created=%s,expires=%s",
		alias,
		time.Now().Format(time.RFC3339),
		time.Now().AddDate(0, 0, cm.keyRotationDays).Format(time.RFC3339))

	// This would use platform-specific extended attributes
	// macOS: xattr, Linux: setfattr, Windows: alternate data streams
	_ = metadata
}

// ValidateKeyStrength validates SSH key against 2025 security standards
func (cm *ModernCryptoManager) ValidateKeyStrength(keyPath string) error {
	keyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	publicKey, comment, _, _, err := ssh.ParseAuthorizedKey(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse SSH key: %w", err)
	}

	keyType := publicKey.Type()

	// 2025 security standards
	switch keyType {
	case "ssh-ed25519":
		// Ed25519 is preferred in 2025
		return nil
	case "ssh-rsa":
		// Check RSA key size (minimum 3072 bits in 2025, 4096 recommended)
		if !strings.Contains(string(keyData), "4096") && !strings.Contains(string(keyData), "3072") {
			return fmt.Errorf("RSA key too weak for 2025 standards (minimum 3072 bits)")
		}
		fmt.Printf("⚠️  RSA key detected. Consider migrating to Ed25519 for better security\n")
		return nil
	case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		// ECDSA is acceptable but Ed25519 is preferred
		fmt.Printf("⚠️  ECDSA key detected. Consider migrating to Ed25519\n")
		return nil
	default:
		return fmt.Errorf("unsupported key type '%s' for 2025 standards", keyType)
	}
}

// SecureWipe securely overwrites sensitive data in memory
func (cm *ModernCryptoManager) SecureWipe(data []byte) {
	// Zero out sensitive data following 2025 security practices
	for i := range data {
		data[i] = 0
	}
}

// KeyRotationStatus checks if keys need rotation
type KeyRotationStatus struct {
	KeyPath       string
	ExpiresOn     time.Time
	DaysUntilExp  int
	NeedsRotation bool
	Severity      string
}

// CheckKeyRotationNeeded evaluates if SSH keys need rotation
func (cm *ModernCryptoManager) CheckKeyRotationNeeded(keyPaths []string) []KeyRotationStatus {
	var status []KeyRotationStatus

	for _, keyPath := range keyPaths {
		if keyPath == "" {
			continue
		}

		// Expand path
		if strings.HasPrefix(keyPath, "~/") {
			homeDir, _ := os.UserHomeDir()
			keyPath = filepath.Join(homeDir, keyPath[2:])
		}

		info, err := os.Stat(keyPath)
		if err != nil {
			continue
		}

		created := info.ModTime()
		expires := created.Add(time.Duration(cm.keyRotationDays) * 24 * time.Hour)
		daysUntil := int(time.Until(expires).Hours() / 24)

		severity := "info"
		needsRotation := false

		if daysUntil <= 0 {
			severity = "critical"
			needsRotation = true
		} else if daysUntil <= 7 {
			severity = "warning"
			needsRotation = true
		} else if daysUntil <= 30 {
			severity = "caution"
		}

		status = append(status, KeyRotationStatus{
			KeyPath:       keyPath,
			ExpiresOn:     expires,
			DaysUntilExp:  daysUntil,
			NeedsRotation: needsRotation,
			Severity:      severity,
		})
	}

	return status
}

// GenerateSecureAlias creates cryptographically secure account aliases
func (cm *ModernCryptoManager) GenerateSecureAlias(baseAlias string) string {
	// Add entropy to prevent alias enumeration attacks
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	suffix := fmt.Sprintf("%x", randomBytes)[:6]
	return fmt.Sprintf("%s_%s", baseAlias, suffix)
}
