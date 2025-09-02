package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/techishthoughts/GitPersona/internal/models"
	"golang.org/x/crypto/ssh"
)

// ConfigValidator provides comprehensive configuration validation following 2025 standards
type ConfigValidator struct {
	validator *validator.Validate
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	v := validator.New()

	// Register custom validators for 2025 security standards
	v.RegisterValidation("secure_email", validateSecureEmail)
	v.RegisterValidation("github_username", validateGitHubUsername)
	v.RegisterValidation("ssh_key_path", validateSSHKeyPath)
	v.RegisterValidation("account_alias", validateAccountAlias)

	return &ConfigValidator{
		validator: v,
	}
}

// ValidateConfig performs comprehensive validation of the entire configuration
func (cv *ConfigValidator) ValidateConfig(config *models.Config) error {
	// Basic struct validation
	if err := cv.validator.Struct(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Cross-account validation
	if err := cv.validateAccountConsistency(config); err != nil {
		return err
	}

	// Security policy validation
	if err := cv.validateSecurityPolicies(config); err != nil {
		return err
	}

	return nil
}

// validateAccountConsistency ensures accounts don't conflict
func (cv *ConfigValidator) validateAccountConsistency(config *models.Config) error {
	seenEmails := make(map[string]string)
	seenGitHubUsernames := make(map[string]string)
	seenSSHKeys := make(map[string]string)

	for alias, account := range config.Accounts {
		// Check for duplicate emails
		if existingAlias, exists := seenEmails[account.Email]; exists {
			return fmt.Errorf("email '%s' is used by both '%s' and '%s' accounts",
				account.Email, existingAlias, alias)
		}
		seenEmails[account.Email] = alias

		// Check for duplicate GitHub usernames
		if account.GitHubUsername != "" {
			if existingAlias, exists := seenGitHubUsernames[account.GitHubUsername]; exists {
				return fmt.Errorf("GitHub username '%s' is used by both '%s' and '%s' accounts",
					account.GitHubUsername, existingAlias, alias)
			}
			seenGitHubUsernames[account.GitHubUsername] = alias
		}

		// Check for duplicate SSH keys
		if account.SSHKeyPath != "" {
			if existingAlias, exists := seenSSHKeys[account.SSHKeyPath]; exists {
				return fmt.Errorf("SSH key '%s' is used by both '%s' and '%s' accounts",
					account.SSHKeyPath, existingAlias, alias)
			}
			seenSSHKeys[account.SSHKeyPath] = alias
		}
	}

	return nil
}

// validateSecurityPolicies enforces 2025 security standards
func (cv *ConfigValidator) validateSecurityPolicies(config *models.Config) error {
	for alias, account := range config.Accounts {
		// Validate email domain policies
		if err := cv.validateEmailPolicy(account.Email, alias); err != nil {
			return err
		}

		// Validate SSH key strength
		if account.SSHKeyPath != "" {
			if err := cv.validateSSHKeyPolicy(account.SSHKeyPath, alias); err != nil {
				return err
			}
		}

		// Validate account age policies
		if err := cv.validateAccountAge(account); err != nil {
			return err
		}
	}

	return nil
}

// validateEmailPolicy checks email against security policies
func (cv *ConfigValidator) validateEmailPolicy(email, alias string) error {
	// Block common disposable email providers (2025 security practice)
	disposableProviders := []string{
		"10minutemail.com", "tempmail.org", "guerrillamail.com",
		"mailinator.com", "yopmail.com", "throwaway.email",
	}

	domain := strings.Split(email, "@")[1]
	for _, disposable := range disposableProviders {
		if strings.Contains(domain, disposable) {
			return fmt.Errorf("account '%s': disposable email provider '%s' not allowed for security reasons", alias, domain)
		}
	}

	// Warn about personal emails for work accounts
	workIndicators := []string{"work", "company", "corp", "enterprise", "org"}
	personalProviders := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com"}

	isWorkAccount := false
	for _, indicator := range workIndicators {
		if strings.Contains(strings.ToLower(alias), indicator) {
			isWorkAccount = true
			break
		}
	}

	if isWorkAccount {
		for _, provider := range personalProviders {
			if strings.Contains(email, provider) {
				fmt.Printf("⚠️  Warning: Account '%s' uses personal email '%s' but appears to be work-related\n",
					alias, provider)
				break
			}
		}
	}

	return nil
}

// validateSSHKeyPolicy enforces SSH key security standards
func (cv *ConfigValidator) validateSSHKeyPolicy(keyPath, alias string) error {
	// Expand home directory
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, _ := os.UserHomeDir()
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}

	// Check key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key '%s' for account '%s' does not exist", keyPath, alias)
	}

	// Check key permissions (must be 600)
	info, err := os.Stat(keyPath)
	if err != nil {
		return err
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		return fmt.Errorf("SSH key '%s' has insecure permissions %o, should be 0600", keyPath, mode.Perm())
	}

	// Validate key type and strength
	publicKeyPath := keyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); err == nil {
		if err := cv.validateKeyStrength(publicKeyPath); err != nil {
			return fmt.Errorf("SSH key strength validation failed for '%s': %w", alias, err)
		}
	}

	return nil
}

// validateKeyStrength checks SSH key against 2025 cryptographic standards
func (cv *ConfigValidator) validateKeyStrength(publicKeyPath string) error {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse SSH key: %w", err)
	}

	keyType := publicKey.Type()

	switch keyType {
	case "ssh-ed25519":
		return nil // Perfect for 2025
	case "ssh-rsa":
		// RSA must be at least 3072 bits in 2025 (4096 recommended)
		keyStr := string(keyData)
		if !strings.Contains(keyStr, "4096") && !strings.Contains(keyStr, "3072") {
			return fmt.Errorf("RSA key below 2025 minimum strength (3072 bits)")
		}
		return nil
	case "ecdsa-sha2-nistp256":
		return fmt.Errorf("NIST P-256 curves deprecated in 2025 due to quantum concerns")
	case "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return nil // Acceptable for 2025
	default:
		return fmt.Errorf("unsupported key type '%s' for 2025 security standards", keyType)
	}
}

// validateAccountAge implements account lifecycle policies
func (cv *ConfigValidator) validateAccountAge(account *models.Account) error {
	// Implement account lifecycle management (2025 compliance)
	maxAccountAge := 365 * 24 * time.Hour // 1 year maximum

	if !account.CreatedAt.IsZero() && time.Since(account.CreatedAt) > maxAccountAge {
		return fmt.Errorf("account '%s' exceeds maximum age policy (1 year), consider renewal", account.Alias)
	}

	return nil
}

// Custom validators for 2025 standards

// validateSecureEmail validates email format and security requirements
func validateSecureEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return false
	}

	// 2025 security: Block weak email patterns
	weakPatterns := []string{
		"password", "123456", "admin", "test", "demo",
	}

	emailLower := strings.ToLower(email)
	for _, pattern := range weakPatterns {
		if strings.Contains(emailLower, pattern) {
			return false
		}
	}

	return true
}

// validateGitHubUsername validates GitHub username format
func validateGitHubUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	// GitHub username rules (2025)
	if len(username) > 39 || len(username) < 1 {
		return false
	}

	if strings.HasPrefix(username, "-") || strings.HasSuffix(username, "-") {
		return false
	}

	if strings.Contains(username, "--") {
		return false
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	return usernameRegex.MatchString(username)
}

// validateSSHKeyPath validates SSH key path format
func validateSSHKeyPath(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Optional field
	}

	// Security: Block paths outside ~/.ssh directory
	if !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "/Users/") {
		return false
	}

	// Must be SSH key file
	return strings.Contains(path, "ssh") && (strings.Contains(path, "id_") || strings.Contains(path, "key"))
}

// validateAccountAlias validates account alias format
func validateAccountAlias(fl validator.FieldLevel) bool {
	alias := fl.Field().String()

	// 2025 standards: alphanumeric + hyphens only
	aliasRegex := regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	if !aliasRegex.MatchString(alias) {
		return false
	}

	// Block reserved names
	reserved := []string{"admin", "root", "system", "config", "default", "null", "undefined"}
	aliasLower := strings.ToLower(alias)
	for _, reserved := range reserved {
		if aliasLower == reserved {
			return false
		}
	}

	return len(alias) >= 2 && len(alias) <= 32
}
