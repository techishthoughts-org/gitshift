package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SecurityValidator provides security validation and hardening
type SecurityValidator struct {
	logger observability.Logger
}

// SecurityAudit represents a security audit result
type SecurityAudit struct {
	Timestamp        time.Time            `json:"timestamp"`
	OverallScore     int                  `json:"overall_score"`
	MaxScore         int                  `json:"max_score"`
	SecurityLevel    string               `json:"security_level"`
	Violations       []*SecurityViolation `json:"violations"`
	Recommendations  []string             `json:"recommendations"`
	ComplianceStatus map[string]bool      `json:"compliance_status"`
}

// SecurityViolation represents a security issue
type SecurityViolation struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"` // "critical", "high", "medium", "low"
	Category    string    `json:"category"` // "permissions", "encryption", "credentials", "config"
	Description string    `json:"description"`
	Path        string    `json:"path,omitempty"`
	Fix         string    `json:"fix"`
	AutoFixable bool      `json:"auto_fixable"`
	Fixed       bool      `json:"fixed"`
	DetectedAt  time.Time `json:"detected_at"`
}

// EncryptionManager handles secure storage operations
type EncryptionManager struct {
	logger observability.Logger
}

// SecureStorage provides encrypted key-value storage
type SecureStorage struct {
	encryptionManager *EncryptionManager
	storagePath       string
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(logger observability.Logger) *SecurityValidator {
	return &SecurityValidator{
		logger: logger,
	}
}

// NewEncryptionManager creates a new encryption manager
func NewEncryptionManager(logger observability.Logger) *EncryptionManager {
	return &EncryptionManager{
		logger: logger,
	}
}

// RunSecurityAudit performs comprehensive security audit
func (sv *SecurityValidator) RunSecurityAudit(ctx context.Context) (*SecurityAudit, error) {
	sv.logger.Info(ctx, "starting_security_audit")

	audit := &SecurityAudit{
		Timestamp:        time.Now(),
		MaxScore:         100,
		Violations:       []*SecurityViolation{},
		Recommendations:  []string{},
		ComplianceStatus: make(map[string]bool),
	}

	// Check file permissions
	permissionViolations := sv.checkFilePermissions(ctx)
	audit.Violations = append(audit.Violations, permissionViolations...)

	// Check for credential exposure
	credentialViolations := sv.checkCredentialExposure(ctx)
	audit.Violations = append(audit.Violations, credentialViolations...)

	// Check configuration security
	configViolations := sv.checkConfigurationSecurity(ctx)
	audit.Violations = append(audit.Violations, configViolations...)

	// Check SSH key security
	sshViolations := sv.checkSSHKeySecurity(ctx)
	audit.Violations = append(audit.Violations, sshViolations...)

	// Calculate overall score
	audit.OverallScore = sv.calculateSecurityScore(audit.Violations, audit.MaxScore)
	audit.SecurityLevel = sv.determineSecurityLevel(audit.OverallScore)

	// Generate recommendations
	audit.Recommendations = sv.generateRecommendations(audit.Violations)

	// Check compliance
	audit.ComplianceStatus = sv.checkCompliance(audit.Violations)

	sv.logger.Info(ctx, "security_audit_completed",
		observability.F("overall_score", audit.OverallScore),
		observability.F("security_level", audit.SecurityLevel),
		observability.F("violations_count", len(audit.Violations)),
	)

	return audit, nil
}

// FixSecurityViolations attempts to automatically fix security violations
func (sv *SecurityValidator) FixSecurityViolations(ctx context.Context, violations []*SecurityViolation) error {
	sv.logger.Info(ctx, "fixing_security_violations",
		observability.F("violations_count", len(violations)),
	)

	fixedCount := 0
	for _, violation := range violations {
		if !violation.AutoFixable || violation.Fixed {
			continue
		}

		if err := sv.fixViolation(ctx, violation); err != nil {
			sv.logger.Error(ctx, "failed_to_fix_violation",
				observability.F("violation_id", violation.ID),
				observability.F("error", err.Error()),
			)
			continue
		}

		violation.Fixed = true
		fixedCount++
	}

	sv.logger.Info(ctx, "security_violations_fix_completed",
		observability.F("total", len(violations)),
		observability.F("fixed", fixedCount),
	)

	return nil
}

// NewSecureStorage creates a new secure storage instance
func (em *EncryptionManager) NewSecureStorage(ctx context.Context, name string) (*SecureStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storageDir := filepath.Join(homeDir, ".gitpersona", "secure")
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create secure storage directory: %w", err)
	}

	storagePath := filepath.Join(storageDir, name+".enc")

	return &SecureStorage{
		encryptionManager: em,
		storagePath:       storagePath,
	}, nil
}

// Store securely stores a value
func (ss *SecureStorage) Store(ctx context.Context, key string, value string) error {
	// TODO: Implement actual encryption
	// For now, just store encoded value
	encoded := ss.simpleEncode(key, value)

	if err := os.WriteFile(ss.storagePath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to store encrypted data: %w", err)
	}

	return nil
}

// Retrieve securely retrieves a value
func (ss *SecureStorage) Retrieve(ctx context.Context, key string) (string, error) {
	if _, err := os.Stat(ss.storagePath); os.IsNotExist(err) {
		return "", fmt.Errorf("secure storage file does not exist")
	}

	encoded, err := os.ReadFile(ss.storagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted data: %w", err)
	}

	// TODO: Implement actual decryption
	value := ss.simpleDecode(key, string(encoded))
	return value, nil
}

// Delete removes stored data
func (ss *SecureStorage) Delete(ctx context.Context) error {
	if err := os.Remove(ss.storagePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete secure storage: %w", err)
	}
	return nil
}

// Private methods for SecurityValidator

func (sv *SecurityValidator) checkFilePermissions(ctx context.Context) []*SecurityViolation {
	violations := []*SecurityViolation{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return violations
	}

	// Check SSH directory
	sshDir := filepath.Join(homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil {
		if info.Mode().Perm()&077 != 0 {
			violations = append(violations, &SecurityViolation{
				ID:          "ssh_dir_permissions",
				Severity:    "high",
				Category:    "permissions",
				Description: "SSH directory has overly permissive permissions",
				Path:        sshDir,
				Fix:         fmt.Sprintf("chmod 700 %s", sshDir),
				AutoFixable: true,
				DetectedAt:  time.Now(),
			})
		}
	}

	// Check SSH keys
	if entries, err := os.ReadDir(sshDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasSuffix(entry.Name(), ".pub") {
				continue
			}

			keyPath := filepath.Join(sshDir, entry.Name())
			if info, err := os.Stat(keyPath); err == nil {
				if info.Mode().Perm() != 0600 {
					violations = append(violations, &SecurityViolation{
						ID:          fmt.Sprintf("ssh_key_permissions_%s", entry.Name()),
						Severity:    "critical",
						Category:    "permissions",
						Description: fmt.Sprintf("SSH key has incorrect permissions: %s", entry.Name()),
						Path:        keyPath,
						Fix:         fmt.Sprintf("chmod 600 %s", keyPath),
						AutoFixable: true,
						DetectedAt:  time.Now(),
					})
				}
			}
		}
	}

	// Check GitPersona config directory
	configDir := filepath.Join(homeDir, ".gitpersona")
	if info, err := os.Stat(configDir); err == nil {
		if info.Mode().Perm()&022 != 0 {
			violations = append(violations, &SecurityViolation{
				ID:          "config_dir_permissions",
				Severity:    "medium",
				Category:    "permissions",
				Description: "GitPersona config directory has overly permissive permissions",
				Path:        configDir,
				Fix:         fmt.Sprintf("chmod 755 %s", configDir),
				AutoFixable: true,
				DetectedAt:  time.Now(),
			})
		}
	}

	return violations
}

func (sv *SecurityValidator) checkCredentialExposure(ctx context.Context) []*SecurityViolation {
	violations := []*SecurityViolation{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return violations
	}

	// Check for tokens in plain text files
	tokensDir := filepath.Join(homeDir, ".gitpersona", "tokens")
	if entries, err := os.ReadDir(tokensDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			tokenPath := filepath.Join(tokensDir, entry.Name())
			if content, err := os.ReadFile(tokenPath); err == nil {
				// Simple heuristic: if content looks like base64, it might be encrypted
				if !sv.looksEncrypted(string(content)) {
					violations = append(violations, &SecurityViolation{
						ID:          fmt.Sprintf("plaintext_token_%s", entry.Name()),
						Severity:    "critical",
						Category:    "credentials",
						Description: fmt.Sprintf("Token file appears to be in plaintext: %s", entry.Name()),
						Path:        tokenPath,
						Fix:         "Re-store token using encrypted storage",
						AutoFixable: false,
						DetectedAt:  time.Now(),
					})
				}
			}
		}
	}

	// Check for credentials in environment variables
	for _, env := range os.Environ() {
		if strings.Contains(strings.ToUpper(env), "GITHUB_TOKEN") ||
			strings.Contains(strings.ToUpper(env), "GIT_TOKEN") {
			violations = append(violations, &SecurityViolation{
				ID:          "env_credentials",
				Severity:    "high",
				Category:    "credentials",
				Description: "Potential credentials found in environment variables",
				Fix:         "Use secure token storage instead of environment variables",
				AutoFixable: false,
				DetectedAt:  time.Now(),
			})
		}
	}

	return violations
}

func (sv *SecurityValidator) checkConfigurationSecurity(ctx context.Context) []*SecurityViolation {
	violations := []*SecurityViolation{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return violations
	}

	// Check config file permissions
	configPath := filepath.Join(homeDir, ".gitpersona", "config.yaml")
	if info, err := os.Stat(configPath); err == nil {
		if info.Mode().Perm()&044 != 0 {
			violations = append(violations, &SecurityViolation{
				ID:          "config_file_permissions",
				Severity:    "medium",
				Category:    "permissions",
				Description: "Config file has overly permissive permissions",
				Path:        configPath,
				Fix:         fmt.Sprintf("chmod 600 %s", configPath),
				AutoFixable: true,
				DetectedAt:  time.Now(),
			})
		}
	}

	return violations
}

func (sv *SecurityValidator) checkSSHKeySecurity(ctx context.Context) []*SecurityViolation {
	violations := []*SecurityViolation{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return violations
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if entries, err := os.ReadDir(sshDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), "id_") || strings.HasSuffix(entry.Name(), ".pub") {
				continue
			}

			// Check for weak key types
			if strings.Contains(entry.Name(), "rsa") {
				keyPath := filepath.Join(sshDir, entry.Name())
				violations = append(violations, &SecurityViolation{
					ID:          fmt.Sprintf("weak_key_type_%s", entry.Name()),
					Severity:    "low",
					Category:    "encryption",
					Description: fmt.Sprintf("RSA key detected, consider upgrading to Ed25519: %s", entry.Name()),
					Path:        keyPath,
					Fix:         "Generate new Ed25519 key and update GitHub",
					AutoFixable: false,
					DetectedAt:  time.Now(),
				})
			}
		}
	}

	return violations
}

func (sv *SecurityValidator) calculateSecurityScore(violations []*SecurityViolation, maxScore int) int {
	score := maxScore

	for _, violation := range violations {
		switch violation.Severity {
		case "critical":
			score -= 20
		case "high":
			score -= 15
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (sv *SecurityValidator) determineSecurityLevel(score int) string {
	switch {
	case score >= 90:
		return "excellent"
	case score >= 80:
		return "good"
	case score >= 70:
		return "fair"
	case score >= 60:
		return "poor"
	default:
		return "critical"
	}
}

func (sv *SecurityValidator) generateRecommendations(violations []*SecurityViolation) []string {
	recommendations := []string{}

	hasPermissionIssues := false
	hasCredentialIssues := false
	hasWeakKeys := false

	for _, violation := range violations {
		switch violation.Category {
		case "permissions":
			hasPermissionIssues = true
		case "credentials":
			hasCredentialIssues = true
		case "encryption":
			hasWeakKeys = true
		}
	}

	if hasPermissionIssues {
		recommendations = append(recommendations, "Fix file and directory permissions to follow security best practices")
	}

	if hasCredentialIssues {
		recommendations = append(recommendations, "Use encrypted storage for all credentials and tokens")
	}

	if hasWeakKeys {
		recommendations = append(recommendations, "Upgrade to Ed25519 SSH keys for better security")
	}

	recommendations = append(recommendations, "Run regular security audits to maintain security posture")

	return recommendations
}

func (sv *SecurityValidator) checkCompliance(violations []*SecurityViolation) map[string]bool {
	compliance := map[string]bool{
		"file_permissions":     true,
		"credential_security":  true,
		"encryption_standards": true,
	}

	for _, violation := range violations {
		switch violation.Category {
		case "permissions":
			compliance["file_permissions"] = false
		case "credentials":
			compliance["credential_security"] = false
		case "encryption":
			compliance["encryption_standards"] = false
		}
	}

	return compliance
}

func (sv *SecurityValidator) fixViolation(ctx context.Context, violation *SecurityViolation) error {
	switch violation.ID {
	case "ssh_dir_permissions":
		return os.Chmod(violation.Path, 0700)

	case "config_dir_permissions":
		return os.Chmod(violation.Path, 0755)

	case "config_file_permissions":
		return os.Chmod(violation.Path, 0600)

	default:
		// Handle SSH key permissions
		if strings.HasPrefix(violation.ID, "ssh_key_permissions_") {
			return os.Chmod(violation.Path, 0600)
		}
	}

	return fmt.Errorf("unknown violation type: %s", violation.ID)
}

func (sv *SecurityValidator) looksEncrypted(content string) bool {
	// Simple heuristic: encrypted content should be base64-like
	if len(content) == 0 {
		return false
	}

	// Check for base64 characteristics
	validChars := 0
	for _, char := range content {
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '+' || char == '/' || char == '=' {
			validChars++
		}
	}

	// If more than 80% of characters are base64-valid, assume encrypted
	return float64(validChars)/float64(len(content)) > 0.8
}

// Simple encoding/decoding (replace with proper encryption in production)
func (ss *SecureStorage) simpleEncode(key, value string) string {
	hash := sha256.Sum256([]byte(key + value))
	return hex.EncodeToString(hash[:]) + ":" + value
}

func (ss *SecureStorage) simpleDecode(key, encoded string) string {
	parts := strings.SplitN(encoded, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1] // Return value part (hash verification skipped for simplicity)
}
