package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// IsolatedTokenService provides completely isolated token management per account
type IsolatedTokenService struct {
	logger       observability.Logger
	storageDir   string
	encKey       []byte
	accountMutex sync.RWMutex
	accounts     map[string]*IsolatedAccountToken
}

// IsolatedAccountToken represents a token with complete isolation metadata
type IsolatedAccountToken struct {
	AccountAlias    string            `json:"account_alias"`
	EncryptedToken  string            `json:"encrypted_token"`
	Nonce           string            `json:"nonce"`
	Username        string            `json:"username"`
	TokenType       string            `json:"token_type"`
	Scopes          []string          `json:"scopes"`
	CreatedAt       time.Time         `json:"created_at"`
	LastUsed        time.Time         `json:"last_used"`
	LastValidated   time.Time         `json:"last_validated"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty"`
	IsValid         bool              `json:"is_valid"`
	ValidationCount int               `json:"validation_count"`
	Metadata        map[string]string `json:"metadata"`
}

// TokenIsolationConfig defines isolation parameters
type TokenIsolationConfig struct {
	StrictIsolation    bool          `json:"strict_isolation"`
	AutoValidation     bool          `json:"auto_validation"`
	ValidationInterval time.Duration `json:"validation_interval"`
	EncryptionEnabled  bool          `json:"encryption_enabled"`
	BackupEnabled      bool          `json:"backup_enabled"`
}

// NewIsolatedTokenService creates a new isolated token service
func NewIsolatedTokenService(logger observability.Logger, config *TokenIsolationConfig) (*IsolatedTokenService, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	storageDir := filepath.Join(homeDir, ".config", "gitpersona", "isolated-tokens")

	// Ensure storage directory exists with strict permissions
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	service := &IsolatedTokenService{
		logger:     logger,
		storageDir: storageDir,
		accounts:   make(map[string]*IsolatedAccountToken),
	}

	// Initialize encryption
	if config.EncryptionEnabled {
		if err := service.initializeEncryption(); err != nil {
			return nil, fmt.Errorf("failed to initialize encryption: %w", err)
		}
	}

	// Load existing tokens
	if err := service.loadExistingTokens(context.Background()); err != nil {
		logger.Warn(context.Background(), "failed_to_load_existing_tokens",
			observability.F("error", err.Error()),
		)
	}

	return service, nil
}

// StoreToken stores a token with complete isolation
func (s *IsolatedTokenService) StoreToken(ctx context.Context, accountAlias, token, username string) error {
	s.logger.Info(ctx, "storing_isolated_token",
		observability.F("account", accountAlias),
		observability.F("username", username),
	)

	// Encrypt token
	encryptedToken, nonce, err := s.encryptToken(token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Create isolated token record
	isolatedToken := &IsolatedAccountToken{
		AccountAlias:    accountAlias,
		EncryptedToken:  base64.StdEncoding.EncodeToString(encryptedToken),
		Nonce:           base64.StdEncoding.EncodeToString(nonce),
		Username:        username,
		TokenType:       s.detectTokenType(token),
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
		LastValidated:   time.Now(),
		IsValid:         true,
		ValidationCount: 0,
		Metadata:        make(map[string]string),
	}

	// Store in memory
	s.accountMutex.Lock()
	s.accounts[accountAlias] = isolatedToken
	s.accountMutex.Unlock()

	// Persist to disk
	if err := s.persistToken(ctx, accountAlias, isolatedToken); err != nil {
		return fmt.Errorf("failed to persist token: %w", err)
	}

	s.logger.Info(ctx, "isolated_token_stored_successfully",
		observability.F("account", accountAlias),
		observability.F("token_type", isolatedToken.TokenType),
	)

	return nil
}

// GetToken retrieves a token with strict isolation enforcement
func (s *IsolatedTokenService) GetToken(ctx context.Context, accountAlias string) (string, error) {
	s.logger.Info(ctx, "retrieving_isolated_token",
		observability.F("account", accountAlias),
	)

	s.accountMutex.RLock()
	isolatedToken, exists := s.accounts[accountAlias]
	s.accountMutex.RUnlock()

	if !exists {
		// Try to load from disk
		if err := s.loadTokenFromDisk(ctx, accountAlias); err != nil {
			return "", fmt.Errorf("no token found for account '%s': %w", accountAlias, err)
		}

		s.accountMutex.RLock()
		isolatedToken, exists = s.accounts[accountAlias]
		s.accountMutex.RUnlock()

		if !exists {
			return "", fmt.Errorf("no token found for account: %s", accountAlias)
		}
	}

	// Validate token is still valid
	if !isolatedToken.IsValid {
		return "", fmt.Errorf("token for account '%s' is marked as invalid", accountAlias)
	}

	// Check if validation is needed
	if time.Since(isolatedToken.LastValidated) > 24*time.Hour {
		s.logger.Info(ctx, "token_validation_needed",
			observability.F("account", accountAlias),
			observability.F("last_validated", isolatedToken.LastValidated.Format(time.RFC3339)),
		)
	}

	// Decrypt token
	token, err := s.decryptToken(isolatedToken.EncryptedToken, isolatedToken.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token for account '%s': %w", accountAlias, err)
	}

	// Update last used time
	s.accountMutex.Lock()
	isolatedToken.LastUsed = time.Now()
	s.accountMutex.Unlock()

	s.logger.Info(ctx, "isolated_token_retrieved_successfully",
		observability.F("account", accountAlias),
		observability.F("username", isolatedToken.Username),
	)

	return token, nil
}

// ValidateTokenIsolation ensures token belongs to the correct account
func (s *IsolatedTokenService) ValidateTokenIsolation(ctx context.Context, accountAlias, expectedUsername string) error {
	s.logger.Info(ctx, "validating_token_isolation",
		observability.F("account", accountAlias),
		observability.F("expected_username", expectedUsername),
	)

	s.accountMutex.RLock()
	isolatedToken, exists := s.accounts[accountAlias]
	s.accountMutex.RUnlock()

	if !exists {
		return fmt.Errorf("no token found for account: %s", accountAlias)
	}

	if isolatedToken.Username != expectedUsername {
		s.logger.Error(ctx, "token_isolation_violation",
			observability.F("account", accountAlias),
			observability.F("expected_username", expectedUsername),
			observability.F("actual_username", isolatedToken.Username),
		)

		// Mark token as invalid
		s.accountMutex.Lock()
		isolatedToken.IsValid = false
		s.accountMutex.Unlock()

		return fmt.Errorf("token isolation violation: account '%s' token belongs to '%s', expected '%s'",
			accountAlias, isolatedToken.Username, expectedUsername)
	}

	s.accountMutex.Lock()
	isolatedToken.LastValidated = time.Now()
	isolatedToken.ValidationCount++
	s.accountMutex.Unlock()

	s.logger.Info(ctx, "token_isolation_validated",
		observability.F("account", accountAlias),
		observability.F("username", expectedUsername),
		observability.F("validation_count", isolatedToken.ValidationCount),
	)

	return nil
}

// DeleteToken removes a token with secure cleanup
func (s *IsolatedTokenService) DeleteToken(ctx context.Context, accountAlias string) error {
	s.logger.Info(ctx, "deleting_isolated_token",
		observability.F("account", accountAlias),
	)

	// Remove from memory
	s.accountMutex.Lock()
	delete(s.accounts, accountAlias)
	s.accountMutex.Unlock()

	// Remove from disk
	tokenPath := filepath.Join(s.storageDir, accountAlias+".json")
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	// Remove backup if it exists
	backupPath := tokenPath + ".backup"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn(ctx, "failed_to_delete_token_backup",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
	}

	s.logger.Info(ctx, "isolated_token_deleted_successfully",
		observability.F("account", accountAlias),
	)

	return nil
}

// ListTokens returns all accounts with tokens
func (s *IsolatedTokenService) ListTokens(ctx context.Context) ([]string, error) {
	s.accountMutex.RLock()
	accounts := make([]string, 0, len(s.accounts))
	for account := range s.accounts {
		accounts = append(accounts, account)
	}
	s.accountMutex.RUnlock()

	s.logger.Info(ctx, "isolated_tokens_listed",
		observability.F("count", len(accounts)),
	)

	return accounts, nil
}

// GetTokenMetadata returns metadata for a token without exposing the actual token
func (s *IsolatedTokenService) GetTokenMetadata(ctx context.Context, accountAlias string) (*IsolatedAccountToken, error) {
	s.accountMutex.RLock()
	isolatedToken, exists := s.accounts[accountAlias]
	s.accountMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no token found for account: %s", accountAlias)
	}

	// Return a copy without the encrypted token
	return &IsolatedAccountToken{
		AccountAlias:    isolatedToken.AccountAlias,
		Username:        isolatedToken.Username,
		TokenType:       isolatedToken.TokenType,
		Scopes:          append([]string{}, isolatedToken.Scopes...),
		CreatedAt:       isolatedToken.CreatedAt,
		LastUsed:        isolatedToken.LastUsed,
		LastValidated:   isolatedToken.LastValidated,
		ExpiresAt:       isolatedToken.ExpiresAt,
		IsValid:         isolatedToken.IsValid,
		ValidationCount: isolatedToken.ValidationCount,
		Metadata: func() map[string]string {
			metadata := make(map[string]string)
			for k, v := range isolatedToken.Metadata {
				metadata[k] = v
			}
			return metadata
		}(),
	}, nil
}

// initializeEncryption sets up encryption for token storage
func (s *IsolatedTokenService) initializeEncryption() error {
	keyPath := filepath.Join(s.storageDir, ".isolation_key")

	// Try to load existing key
	if _, err := os.Stat(keyPath); err == nil {
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read encryption key: %w", err)
		}

		decoded, err := base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			return fmt.Errorf("failed to decode encryption key: %w", err)
		}

		s.encKey = decoded
		return nil
	}

	// Generate new key
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Save key to file with strict permissions
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to save encryption key: %w", err)
	}

	s.encKey = key
	return nil
}

// encryptToken encrypts a token using AES-GCM
func (s *IsolatedTokenService) encryptToken(token string) ([]byte, []byte, error) {
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(token), nil)
	return ciphertext, nonce, nil
}

// decryptToken decrypts a token using AES-GCM
func (s *IsolatedTokenService) decryptToken(encryptedData, nonceData string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceData)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// detectTokenType identifies the type of GitHub token
func (s *IsolatedTokenService) detectTokenType(token string) string {
	if len(token) < 4 {
		return "unknown"
	}

	switch token[:4] {
	case "ghp_":
		return "personal_access_token"
	case "gho_":
		return "oauth_token"
	case "ghu_":
		return "user_token"
	case "ghs_":
		return "server_token"
	case "ghr_":
		return "refresh_token"
	default:
		return "legacy_token"
	}
}

// persistToken saves token to disk
func (s *IsolatedTokenService) persistToken(ctx context.Context, accountAlias string, token *IsolatedAccountToken) error {
	tokenPath := filepath.Join(s.storageDir, accountAlias+".json")

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// loadTokenFromDisk loads a token from disk storage
func (s *IsolatedTokenService) loadTokenFromDisk(ctx context.Context, accountAlias string) error {
	tokenPath := filepath.Join(s.storageDir, accountAlias+".json")

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var token IsolatedAccountToken
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	s.accountMutex.Lock()
	s.accounts[accountAlias] = &token
	s.accountMutex.Unlock()

	return nil
}

// loadExistingTokens loads all existing tokens from disk
func (s *IsolatedTokenService) loadExistingTokens(ctx context.Context) error {
	files, err := os.ReadDir(s.storageDir)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		accountAlias := strings.TrimSuffix(file.Name(), ".json")
		if err := s.loadTokenFromDisk(ctx, accountAlias); err != nil {
			s.logger.Warn(ctx, "failed_to_load_token_from_disk",
				observability.F("account", accountAlias),
				observability.F("error", err.Error()),
			)
		}
	}

	return nil
}
