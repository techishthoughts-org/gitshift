package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// DirectGitHubAPIService provides GitHub API access without GitHub CLI dependency
type DirectGitHubAPIService struct {
	logger     observability.Logger
	httpClient *http.Client
	baseURL    string
	accounts   map[string]*AccountToken
	mutex      sync.RWMutex
}

// AccountToken stores token and metadata for a specific account
type AccountToken struct {
	Token     string            `json:"token"`
	Username  string            `json:"username"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	Scopes    []string          `json:"scopes"`
	Metadata  map[string]string `json:"metadata"`
	LastUsed  time.Time         `json:"last_used"`
	Valid     bool              `json:"valid"`
}

// GitHubAPIResponse represents a standard GitHub API response
type GitHubAPIResponse struct {
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data"`
	RateLimit *RateLimitInfo         `json:"rate_limit,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// NewDirectGitHubAPIService creates a new direct GitHub API service
func NewDirectGitHubAPIService(logger observability.Logger) *DirectGitHubAPIService {
	return &DirectGitHubAPIService{
		logger:   logger,
		baseURL:  "https://api.github.com",
		accounts: make(map[string]*AccountToken),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// AuthenticateAccount validates and stores a token for a specific account
func (s *DirectGitHubAPIService) AuthenticateAccount(ctx context.Context, accountAlias, token string) error {
	s.logger.Info(ctx, "authenticating_account_with_direct_api",
		observability.F("account", accountAlias),
		observability.F("token_prefix", token[:minInt(8, len(token))]),
	)

	// Validate token by getting user info
	user, err := s.getAuthenticatedUser(ctx, token)
	if err != nil {
		s.logger.Error(ctx, "account_authentication_failed",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to authenticate account '%s': %w", accountAlias, err)
	}

	// Get token scopes
	scopes, err := s.getTokenScopes(ctx, token)
	if err != nil {
		s.logger.Warn(ctx, "failed_to_get_token_scopes",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
		scopes = []string{} // Continue without scopes info
	}

	// Store account token
	s.mutex.Lock()
	s.accounts[accountAlias] = &AccountToken{
		Token:    token,
		Username: user.Login,
		Scopes:   scopes,
		Metadata: map[string]string{
			"user_id":    fmt.Sprintf("%d", user.ID),
			"user_name":  user.Name,
			"user_email": user.Email,
			"avatar_url": user.AvatarURL,
		},
		LastUsed: time.Now(),
		Valid:    true,
	}
	s.mutex.Unlock()

	s.logger.Info(ctx, "account_authenticated_successfully",
		observability.F("account", accountAlias),
		observability.F("username", user.Login),
		observability.F("scopes", len(scopes)),
	)

	return nil
}

// GetTokenForAccount retrieves the token for a specific account
func (s *DirectGitHubAPIService) GetTokenForAccount(ctx context.Context, accountAlias string) (string, error) {
	s.mutex.RLock()
	accountToken, exists := s.accounts[accountAlias]
	s.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("no token found for account: %s", accountAlias)
	}

	if !accountToken.Valid {
		return "", fmt.Errorf("token for account '%s' is invalid", accountAlias)
	}

	// Update last used time
	s.mutex.Lock()
	accountToken.LastUsed = time.Now()
	s.mutex.Unlock()

	s.logger.Info(ctx, "token_retrieved_for_account",
		observability.F("account", accountAlias),
		observability.F("username", accountToken.Username),
	)

	return accountToken.Token, nil
}

// ValidateAccountToken validates that a token belongs to the expected username
func (s *DirectGitHubAPIService) ValidateAccountToken(ctx context.Context, accountAlias, expectedUsername string) error {
	s.mutex.RLock()
	accountToken, exists := s.accounts[accountAlias]
	s.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no token found for account: %s", accountAlias)
	}

	if accountToken.Username != expectedUsername {
		return fmt.Errorf("token username mismatch for account '%s': expected '%s', got '%s'",
			accountAlias, expectedUsername, accountToken.Username)
	}

	// Re-validate token by making API call
	user, err := s.getAuthenticatedUser(ctx, accountToken.Token)
	if err != nil {
		s.mutex.Lock()
		accountToken.Valid = false
		s.mutex.Unlock()
		return fmt.Errorf("token validation failed for account '%s': %w", accountAlias, err)
	}

	if user.Login != expectedUsername {
		s.mutex.Lock()
		accountToken.Valid = false
		s.mutex.Unlock()
		return fmt.Errorf("token belongs to different user: expected '%s', got '%s'",
			expectedUsername, user.Login)
	}

	s.logger.Info(ctx, "account_token_validated_successfully",
		observability.F("account", accountAlias),
		observability.F("username", expectedUsername),
	)

	return nil
}

// GetAccountInfo retrieves account information
func (s *DirectGitHubAPIService) GetAccountInfo(ctx context.Context, accountAlias string) (*AccountToken, error) {
	s.mutex.RLock()
	accountToken, exists := s.accounts[accountAlias]
	s.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no account found: %s", accountAlias)
	}

	// Return a copy to prevent external modification
	return &AccountToken{
		Token:    "", // Don't return the actual token
		Username: accountToken.Username,
		Scopes:   append([]string{}, accountToken.Scopes...),
		Metadata: func() map[string]string {
			metadata := make(map[string]string)
			for k, v := range accountToken.Metadata {
				metadata[k] = v
			}
			return metadata
		}(),
		LastUsed: accountToken.LastUsed,
		Valid:    accountToken.Valid,
	}, nil
}

// ListAccounts returns all configured accounts
func (s *DirectGitHubAPIService) ListAccounts(ctx context.Context) []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	accounts := make([]string, 0, len(s.accounts))
	for account := range s.accounts {
		accounts = append(accounts, account)
	}

	s.logger.Info(ctx, "accounts_listed",
		observability.F("count", len(accounts)),
	)

	return accounts
}

// RemoveAccount removes an account and its token
func (s *DirectGitHubAPIService) RemoveAccount(ctx context.Context, accountAlias string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.accounts[accountAlias]; !exists {
		return fmt.Errorf("account not found: %s", accountAlias)
	}

	delete(s.accounts, accountAlias)

	s.logger.Info(ctx, "account_removed",
		observability.F("account", accountAlias),
	)

	return nil
}

// getAuthenticatedUser makes a direct API call to get user information
func (s *DirectGitHubAPIService) getAuthenticatedUser(ctx context.Context, token string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "GitPersona/1.0.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}

// getTokenScopes retrieves the scopes for a token
func (s *DirectGitHubAPIService) getTokenScopes(ctx context.Context, token string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "GitPersona/1.0.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	// Get scopes from X-OAuth-Scopes header
	scopesHeader := resp.Header.Get("X-OAuth-Scopes")
	if scopesHeader == "" {
		return []string{}, nil
	}

	scopes := strings.Split(scopesHeader, ", ")
	for i, scope := range scopes {
		scopes[i] = strings.TrimSpace(scope)
	}

	return scopes, nil
}

// TestAPIAccess tests API access for an account
func (s *DirectGitHubAPIService) TestAPIAccess(ctx context.Context, accountAlias string) error {
	token, err := s.GetTokenForAccount(ctx, accountAlias)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	_, err = s.getAuthenticatedUser(ctx, token)
	if err != nil {
		return fmt.Errorf("API access test failed: %w", err)
	}

	s.logger.Info(ctx, "api_access_test_successful",
		observability.F("account", accountAlias),
	)

	return nil
}

// RefreshAccountToken refreshes token information for an account
func (s *DirectGitHubAPIService) RefreshAccountToken(ctx context.Context, accountAlias string) error {
	s.mutex.RLock()
	accountToken, exists := s.accounts[accountAlias]
	s.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("account not found: %s", accountAlias)
	}

	// Re-authenticate with existing token
	return s.AuthenticateAccount(ctx, accountAlias, accountToken.Token)
}

// minInt returns the minimum of two integers
func minIntGitHub(a, b int) int {
	if a < b {
		return a
	}
	return b
}
