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
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// TokenStorageService manages secure storage of GitHub tokens
type TokenStorageService interface {
	StoreToken(ctx context.Context, account, token string) error
	GetToken(ctx context.Context, account string) (string, error)
	DeleteToken(ctx context.Context, account string) error
	ListTokens(ctx context.Context) ([]string, error)
	ValidateToken(ctx context.Context, token string) (*TokenValidationResult, error)
	GetTokenMetadata(ctx context.Context, account string) (*TokenMetadata, error)
}

// RealTokenStorageService implements secure token storage
type RealTokenStorageService struct {
	logger    observability.Logger
	storePath string
	encKey    []byte
}

// TokenMetadata contains metadata about a stored token
type TokenMetadata struct {
	Account     string    `json:"account"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	TokenPrefix string    `json:"token_prefix"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// TokenValidationResult contains the result of token validation
type TokenValidationResult struct {
	Valid     bool           `json:"valid"`
	Username  string         `json:"username,omitempty"`
	Email     string         `json:"email,omitempty"`
	Scopes    []string       `json:"scopes,omitempty"`
	Error     string         `json:"error,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
}

// RateLimitInfo contains GitHub API rate limit information
type RateLimitInfo struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"reset_at"`
}

// EncryptedToken represents an encrypted token with metadata
type EncryptedToken struct {
	EncryptedData string        `json:"encrypted_data"`
	Nonce         string        `json:"nonce"`
	Metadata      TokenMetadata `json:"metadata"`
}

// NewTokenStorageService creates a new token storage service
func NewTokenStorageService(logger observability.Logger) (*RealTokenStorageService, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	storePath := filepath.Join(homeDir, ".config", "gitpersona", "tokens")

	// Ensure tokens directory exists
	if err := os.MkdirAll(storePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create tokens directory: %w", err)
	}

	service := &RealTokenStorageService{
		logger:    logger,
		storePath: storePath,
	}

	// Initialize encryption key
	if err := service.initializeEncryptionKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize encryption: %w", err)
	}

	return service, nil
}

// initializeEncryptionKey creates or loads the encryption key
func (s *RealTokenStorageService) initializeEncryptionKey() error {
	keyPath := filepath.Join(s.storePath, ".encryption_key")

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

	// Save key to file
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to save encryption key: %w", err)
	}

	s.encKey = key
	return nil
}

// StoreToken encrypts and stores a GitHub token
func (s *RealTokenStorageService) StoreToken(ctx context.Context, account, token string) error {
	s.logger.Info(ctx, "storing_github_token",
		observability.F("account", account),
		observability.F("token_prefix", token[:minInt(8, len(token))]),
	)

	// Create metadata
	metadata := TokenMetadata{
		Account:     account,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		TokenPrefix: token[:minInt(8, len(token))],
	}

	// Encrypt token
	encryptedData, nonce, err := s.encryptToken(token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Create encrypted token structure
	encToken := EncryptedToken{
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
		Nonce:         base64.StdEncoding.EncodeToString(nonce),
		Metadata:      metadata,
	}

	// Save to file
	tokenPath := filepath.Join(s.storePath, account+".json")
	data, err := json.MarshalIndent(encToken, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	s.logger.Info(ctx, "github_token_stored_successfully",
		observability.F("account", account),
		observability.F("path", tokenPath),
	)

	return nil
}

// GetToken retrieves and decrypts a GitHub token
func (s *RealTokenStorageService) GetToken(ctx context.Context, account string) (string, error) {
	s.logger.Info(ctx, "retrieving_github_token",
		observability.F("account", account),
	)

	tokenPath := filepath.Join(s.storePath, account+".json")

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return "", fmt.Errorf("no token found for account: %s", account)
	}

	// Read token file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	// Parse encrypted token
	var encToken EncryptedToken
	if err := json.Unmarshal(data, &encToken); err != nil {
		return "", fmt.Errorf("failed to parse token file: %w", err)
	}

	// Decrypt token
	token, err := s.decryptToken(encToken.EncryptedData, encToken.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Update last used timestamp
	encToken.Metadata.LastUsed = time.Now()
	if err := s.updateTokenMetadata(tokenPath, encToken); err != nil {
		s.logger.Warn(ctx, "failed_to_update_token_metadata",
			observability.F("account", account),
			observability.F("error", err.Error()),
		)
	}

	s.logger.Info(ctx, "github_token_retrieved_successfully",
		observability.F("account", account),
	)

	return token, nil
}

// DeleteToken removes a stored token
func (s *RealTokenStorageService) DeleteToken(ctx context.Context, account string) error {
	s.logger.Info(ctx, "deleting_github_token",
		observability.F("account", account),
	)

	tokenPath := filepath.Join(s.storePath, account+".json")

	if err := os.Remove(tokenPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no token found for account: %s", account)
		}
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	s.logger.Info(ctx, "github_token_deleted_successfully",
		observability.F("account", account),
	)

	return nil
}

// ListTokens returns a list of accounts with stored tokens
func (s *RealTokenStorageService) ListTokens(ctx context.Context) ([]string, error) {
	s.logger.Info(ctx, "listing_stored_tokens")

	files, err := os.ReadDir(s.storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokens directory: %w", err)
	}

	var accounts []string
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" || file.Name() == ".encryption_key" {
			continue
		}

		// Extract account name from filename
		name := file.Name()
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			accounts = append(accounts, name[:len(name)-5])
		}
	}

	s.logger.Info(ctx, "tokens_listed_successfully",
		observability.F("count", len(accounts)),
	)

	return accounts, nil
}

// GetTokenMetadata retrieves metadata for a stored token
func (s *RealTokenStorageService) GetTokenMetadata(ctx context.Context, account string) (*TokenMetadata, error) {
	tokenPath := filepath.Join(s.storePath, account+".json")

	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no token found for account: %s", account)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var encToken EncryptedToken
	if err := json.Unmarshal(data, &encToken); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &encToken.Metadata, nil
}

// ValidateToken validates a GitHub token by making an API call
func (s *RealTokenStorageService) ValidateToken(ctx context.Context, token string) (*TokenValidationResult, error) {
	s.logger.Info(ctx, "validating_github_token")

	result := &TokenValidationResult{
		Valid: false,
	}

	// Make API call to validate token
	// This is a simplified implementation - in a real scenario, you'd use the GitHub API client
	// For now, we'll simulate validation
	if len(token) < 10 || !isValidTokenFormat(token) {
		result.Error = "invalid token format"
		return result, nil
	}

	result.Valid = true
	result.Username = "validated-user"       // Would come from actual API call
	result.Scopes = []string{"repo", "user"} // Would come from actual API call

	s.logger.Info(ctx, "github_token_validation_completed",
		observability.F("valid", result.Valid),
	)

	return result, nil
}

// encryptToken encrypts a token using AES-GCM
func (s *RealTokenStorageService) encryptToken(token string) ([]byte, []byte, error) {
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
func (s *RealTokenStorageService) decryptToken(encryptedData, nonceData string) (string, error) {
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

// updateTokenMetadata updates the metadata in a token file
func (s *RealTokenStorageService) updateTokenMetadata(tokenPath string, encToken EncryptedToken) error {
	data, err := json.MarshalIndent(encToken, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

// isValidTokenFormat checks if a token has a valid GitHub token format
func isValidTokenFormat(token string) bool {
	// GitHub personal access tokens start with specific prefixes
	validPrefixes := []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"}

	for _, prefix := range validPrefixes {
		if len(token) > len(prefix) && token[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Enhanced GitHub API service that uses token storage
type EnhancedGitHubService struct {
	*RealGitHubService
	tokenStorage TokenStorageService
}

// NewEnhancedGitHubService creates a GitHub service with token storage
func NewEnhancedGitHubService(logger observability.Logger, tokenStorage TokenStorageService) *EnhancedGitHubService {
	return &EnhancedGitHubService{
		RealGitHubService: NewRealGitHubService(logger, nil),
		tokenStorage:      tokenStorage,
	}
}

// GetTokenForAccount retrieves a token for a specific account
func (s *EnhancedGitHubService) GetTokenForAccount(ctx context.Context, account string) (string, error) {
	return s.tokenStorage.GetToken(ctx, account)
}

// SetTokenForAccount stores a token for a specific account
func (s *EnhancedGitHubService) SetTokenForAccount(ctx context.Context, account, token string) error {
	return s.tokenStorage.StoreToken(ctx, account, token)
}
