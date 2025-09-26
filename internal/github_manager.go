package internal

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealGitHubManager implements the GitHubManager interface with secure token management
type RealGitHubManager struct {
	logger     observability.Logger
	httpClient *http.Client
	baseURL    string
}

// NewGitHubManager creates a new GitHub manager
func NewGitHubManager(logger observability.Logger) GitHubManager {
	return &RealGitHubManager{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// SetToken securely stores a GitHub token for an account
func (ghm *RealGitHubManager) SetToken(ctx context.Context, account *Account, token string) error {
	ghm.logger.Info(ctx, "setting_github_token",
		observability.F("account", account.Alias),
	)

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Create secure token storage directory
	tokenDir, err := ghm.getTokenDirectory()
	if err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Encrypt and store the token
	encryptedToken, err := ghm.encryptToken(token, account.Alias)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	tokenPath := filepath.Join(tokenDir, fmt.Sprintf("%s.token", account.Alias))
	if err := os.WriteFile(tokenPath, encryptedToken, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	ghm.logger.Info(ctx, "github_token_stored_securely",
		observability.F("account", account.Alias),
		observability.F("token_path", tokenPath),
	)

	return nil
}

// GetToken retrieves and decrypts a GitHub token for an account
func (ghm *RealGitHubManager) GetToken(ctx context.Context, account *Account) (string, error) {
	ghm.logger.Info(ctx, "getting_github_token",
		observability.F("account", account.Alias),
	)

	tokenDir, err := ghm.getTokenDirectory()
	if err != nil {
		return "", fmt.Errorf("failed to get token directory: %w", err)
	}

	tokenPath := filepath.Join(tokenDir, fmt.Sprintf("%s.token", account.Alias))
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return "", fmt.Errorf("no token found for account '%s'", account.Alias)
	}

	encryptedToken, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	token, err := ghm.decryptToken(encryptedToken, account.Alias)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}

// ValidateToken validates a GitHub token and returns information about it
func (ghm *RealGitHubManager) ValidateToken(ctx context.Context, account *Account) (*TokenValidation, error) {
	ghm.logger.Info(ctx, "validating_github_token",
		observability.F("account", account.Alias),
	)

	token, err := ghm.GetToken(ctx, account)
	if err != nil {
		return &TokenValidation{
			Valid:   false,
			Message: fmt.Sprintf("Failed to retrieve token: %v", err),
		}, nil
	}

	// Get user information to validate token
	user, err := ghm.GetUserInfo(ctx, account)
	if err != nil {
		return &TokenValidation{
			Valid:   false,
			Message: fmt.Sprintf("Token validation failed: %v", err),
		}, nil
	}

	// Get token scopes
	req, err := http.NewRequestWithContext(ctx, "GET", ghm.baseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := ghm.httpClient.Do(req)
	if err != nil {
		return &TokenValidation{
			Valid:   false,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	validation := &TokenValidation{
		Valid:    resp.StatusCode == 200,
		Username: user.Username,
	}

	// Parse scopes from headers
	if scopes := resp.Header.Get("X-OAuth-Scopes"); scopes != "" {
		validation.Scopes = strings.Split(strings.ReplaceAll(scopes, " ", ""), ",")
	}

	// Check for token expiration (if available in headers)
	if expiresAt := resp.Header.Get("GitHub-Authentication-Token-Expiration"); expiresAt != "" {
		validation.ExpiresAt = &expiresAt
	}

	if validation.Valid {
		validation.Message = "Token is valid and active"
	} else {
		validation.Message = "Token is invalid or expired"
	}

	ghm.logger.Info(ctx, "github_token_validated",
		observability.F("account", account.Alias),
		observability.F("valid", validation.Valid),
		observability.F("username", validation.Username),
		observability.F("scopes_count", len(validation.Scopes)),
	)

	return validation, nil
}

// RefreshToken refreshes a GitHub token (placeholder for future OAuth implementation)
func (ghm *RealGitHubManager) RefreshToken(ctx context.Context, account *Account) error {
	ghm.logger.Info(ctx, "refreshing_github_token",
		observability.F("account", account.Alias),
	)

	// GitHub personal access tokens don't have a refresh mechanism
	// This would be implemented for OAuth Apps in the future
	return fmt.Errorf("token refresh not supported for personal access tokens")
}

// GetUserInfo retrieves GitHub user information
func (ghm *RealGitHubManager) GetUserInfo(ctx context.Context, account *Account) (*GitHubUser, error) {
	ghm.logger.Info(ctx, "getting_github_user_info",
		observability.F("account", account.Alias),
	)

	token, err := ghm.GetToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ghm.baseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := ghm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	ghm.logger.Info(ctx, "github_user_info_retrieved",
		observability.F("account", account.Alias),
		observability.F("username", user.Username),
		observability.F("user_id", user.ID),
	)

	return &user, nil
}

// ListRepositories lists repositories for the authenticated user
func (ghm *RealGitHubManager) ListRepositories(ctx context.Context, account *Account) ([]*GitHubRepository, error) {
	ghm.logger.Info(ctx, "listing_github_repositories",
		observability.F("account", account.Alias),
	)

	token, err := ghm.GetToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Get user's repositories
	req, err := http.NewRequestWithContext(ctx, "GET", ghm.baseURL+"/user/repos?per_page=100&sort=updated", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := ghm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	var repos []*GitHubRepository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode repositories response: %w", err)
	}

	ghm.logger.Info(ctx, "github_repositories_listed",
		observability.F("account", account.Alias),
		observability.F("repo_count", len(repos)),
	)

	return repos, nil
}

// UploadSSHKey uploads an SSH key to GitHub
func (ghm *RealGitHubManager) UploadSSHKey(ctx context.Context, account *Account, keyContent, title string) error {
	ghm.logger.Info(ctx, "uploading_ssh_key_to_github",
		observability.F("account", account.Alias),
		observability.F("title", title),
	)

	token, err := ghm.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Prepare key upload payload
	keyData := map[string]string{
		"title": title,
		"key":   strings.TrimSpace(keyContent),
	}

	jsonData, err := json.Marshal(keyData)
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ghm.baseURL+"/user/keys", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ghm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload SSH key (status %d): %s", resp.StatusCode, string(body))
	}

	ghm.logger.Info(ctx, "ssh_key_uploaded_to_github",
		observability.F("account", account.Alias),
		observability.F("title", title),
	)

	return nil
}

// TestAPIAccess tests GitHub API access
func (ghm *RealGitHubManager) TestAPIAccess(ctx context.Context, account *Account) error {
	ghm.logger.Info(ctx, "testing_github_api_access",
		observability.F("account", account.Alias),
	)

	// Simply try to get user info
	_, err := ghm.GetUserInfo(ctx, account)
	if err != nil {
		return fmt.Errorf("GitHub API access test failed: %w", err)
	}

	ghm.logger.Info(ctx, "github_api_access_successful",
		observability.F("account", account.Alias),
	)

	return nil
}

// ValidateSSHAccess validates SSH access to GitHub
func (ghm *RealGitHubManager) ValidateSSHAccess(ctx context.Context, account *Account) error {
	ghm.logger.Info(ctx, "validating_github_ssh_access",
		observability.F("account", account.Alias),
	)

	// This would typically use SSH manager to test connectivity
	// For now, we'll assume it's handled elsewhere
	return fmt.Errorf("SSH access validation not implemented - use SSH manager")
}

// Helper methods for secure token storage

func (ghm *RealGitHubManager) getTokenDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	tokenDir := filepath.Join(homeDir, ".gitpersona", "tokens")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return "", err
	}

	return tokenDir, nil
}

func (ghm *RealGitHubManager) encryptToken(token, accountAlias string) ([]byte, error) {
	// Create encryption key from account alias and system info
	key := ghm.generateEncryptionKey(accountAlias)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the token
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	// Encode as base64 for storage
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return []byte(encoded), nil
}

func (ghm *RealGitHubManager) decryptToken(encryptedToken []byte, accountAlias string) (string, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(string(encryptedToken))
	if err != nil {
		return "", err
	}

	// Create encryption key
	key := ghm.generateEncryptionKey(accountAlias)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (ghm *RealGitHubManager) generateEncryptionKey(accountAlias string) []byte {
	// Generate a deterministic but secure key from account alias and system info
	// In production, this should use a more sophisticated key derivation
	h := sha256.New()
	h.Write([]byte(accountAlias))

	// Add some system-specific entropy
	if hostname, err := os.Hostname(); err == nil {
		h.Write([]byte(hostname))
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		h.Write([]byte(filepath.Base(homeDir)))
	}

	return h.Sum(nil)
}
