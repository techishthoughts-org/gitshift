package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealGitHubTokenService handles GitHub token retrieval and management
type RealGitHubTokenService struct {
	logger       observability.Logger
	runner       execrunner.CmdRunner
	tokenStorage TokenStorageService
}

// NewGitHubTokenService creates a new GitHub token service
func NewGitHubTokenService(logger observability.Logger, runner execrunner.CmdRunner) GitHubTokenService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	// Initialize token storage
	tokenStorage, err := NewTokenStorageService(logger)
	if err != nil {
		logger.Error(context.Background(), "failed_to_initialize_token_storage",
			observability.F("error", err.Error()),
		)
		// Fall back to CLI-based service
		tokenStorage = nil
	}

	return &RealGitHubTokenService{
		logger:       logger,
		runner:       runner,
		tokenStorage: tokenStorage,
	}
}

// GetCurrentGitHubToken retrieves the current GitHub token
func (s *RealGitHubTokenService) GetCurrentGitHubToken(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "retrieving_current_github_token")

	// Try to get token from storage first
	if s.tokenStorage != nil {
		// Get current account from config
		// This is a simplified implementation - you'd need to inject config manager
		// For now, try to get the "default" token
		token, err := s.tokenStorage.GetToken(ctx, "default")
		if err == nil {
			s.logger.Info(ctx, "retrieved_token_from_storage")
			return token, nil
		}
		s.logger.Info(ctx, "no_token_in_storage_falling_back_to_cli")
	}

	// Fallback to GitHub CLI
	return s.getTokenFromCLI(ctx)
}

// getTokenFromCLI retrieves token from GitHub CLI (fallback method)
func (s *RealGitHubTokenService) getTokenFromCLI(ctx context.Context) (string, error) {
	// Check if GitHub CLI is available
	if err := s.checkGitHubCLIAvailable(ctx); err != nil {
		return "", fmt.Errorf("GitHub CLI not available: %w", err)
	}

	// Check if user is authenticated
	if err := s.checkGitHubAuthentication(ctx); err != nil {
		return "", fmt.Errorf("GitHub authentication required: %w", err)
	}

	// Get the token
	output, err := s.runner.CombinedOutput(ctx, "gh", "auth", "token")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("received empty token from GitHub CLI")
	}

	s.logger.Info(ctx, "successfully_retrieved_github_token_from_cli")
	return token, nil
}

// GetTokenForAccount retrieves a GitHub token for a specific account
func (s *RealGitHubTokenService) GetTokenForAccount(ctx context.Context, accountAlias string) (string, error) {
	s.logger.Info(ctx, "retrieving_github_token_for_account",
		observability.F("account", accountAlias),
	)

	// Try to get token from storage
	if s.tokenStorage != nil {
		token, err := s.tokenStorage.GetToken(ctx, accountAlias)
		if err == nil {
			s.logger.Info(ctx, "retrieved_account_token_from_storage",
				observability.F("account", accountAlias),
			)
			return token, nil
		}
	}

	// Fallback to current token if account-specific token not found
	s.logger.Info(ctx, "account_token_not_found_using_current_token",
		observability.F("account", accountAlias),
	)
	return s.GetCurrentGitHubToken(ctx)
}

// ValidateToken validates that a GitHub token is working
func (s *RealGitHubTokenService) ValidateToken(ctx context.Context, token string) error {
	s.logger.Info(ctx, "validating_github_token")

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Test the token by making a GitHub API call
	cmd := exec.Command("gh", "api", "user")
	cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error(ctx, "github_token_validation_failed",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Parse response to ensure it's valid
	outputStr := string(output)
	if !strings.Contains(outputStr, "login") {
		s.logger.Error(ctx, "github_token_validation_invalid_response",
			observability.F("response", outputStr),
		)
		return fmt.Errorf("invalid token response: missing user login")
	}

	// Extract and validate user info
	user, err := parseGitHubUserJSON(outputStr)
	if err != nil {
		return fmt.Errorf("failed to parse user data during validation: %w", err)
	}

	if user.Login == "" {
		return fmt.Errorf("token validation failed: empty login")
	}

	s.logger.Info(ctx, "github_token_validation_successful",
		observability.F("user_login", user.Login),
	)
	return nil
}

// ValidateTokenWithRetry validates a GitHub token with retry mechanism
func (s *RealGitHubTokenService) ValidateTokenWithRetry(ctx context.Context, token string, maxRetries int) error {
	s.logger.Info(ctx, "validating_github_token_with_retry",
		observability.F("max_retries", maxRetries),
	)

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := s.ValidateToken(ctx, token); err == nil {
			s.logger.Info(ctx, "github_token_validation_successful_after_retry",
				observability.F("attempt", i+1),
			)
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				s.logger.Warn(ctx, "github_token_validation_failed_retrying",
					observability.F("attempt", i+1),
					observability.F("error", err.Error()),
				)
				time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
			}
		}
	}

	return fmt.Errorf("token validation failed after %d attempts: %w", maxRetries, lastErr)
}

// RefreshToken refreshes the current GitHub token
func (s *RealGitHubTokenService) RefreshToken(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "refreshing_github_token")

	// GitHub CLI doesn't support token refresh directly
	// This is a placeholder for future enhancement
	return s.GetCurrentGitHubToken(ctx)
}

// CacheToken caches a token for an account
func (s *RealGitHubTokenService) CacheToken(ctx context.Context, account, token string) error {
	s.logger.Info(ctx, "caching_github_token",
		observability.F("account", account),
	)

	if s.tokenStorage == nil {
		s.logger.Warn(ctx, "no_token_storage_available_for_caching")
		return fmt.Errorf("token storage not available")
	}

	return s.tokenStorage.StoreToken(ctx, account, token)
}

// GetCachedToken retrieves a cached token for an account
func (s *RealGitHubTokenService) GetCachedToken(ctx context.Context, account string) (string, error) {
	s.logger.Info(ctx, "retrieving_cached_github_token",
		observability.F("account", account),
	)

	if s.tokenStorage == nil {
		s.logger.Warn(ctx, "no_token_storage_available_for_retrieval")
		return s.GetCurrentGitHubToken(ctx)
	}

	return s.tokenStorage.GetToken(ctx, account)
}

// checkGitHubCLIAvailable checks if GitHub CLI is installed and available
func (s *RealGitHubTokenService) checkGitHubCLIAvailable(ctx context.Context) error {
	_, err := s.runner.CombinedOutput(ctx, "gh", "--version")
	if err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed or not in PATH")
	}
	return nil
}

// checkGitHubAuthentication checks if the user is authenticated with GitHub CLI
func (s *RealGitHubTokenService) checkGitHubAuthentication(ctx context.Context) error {
	_, err := s.runner.CombinedOutput(ctx, "gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("not authenticated with GitHub CLI - run 'gh auth login' first")
	}
	return nil
}

// GetAuthenticatedUser gets the currently authenticated user information
func (s *RealGitHubTokenService) GetAuthenticatedUser(ctx context.Context) (*GitHubUser, error) {
	s.logger.Info(ctx, "getting_authenticated_user_info")

	output, err := s.runner.CombinedOutput(ctx, "gh", "api", "user")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	// Parse the JSON response
	user, err := parseGitHubUserJSON(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	s.logger.Info(ctx, "retrieved_authenticated_user",
		observability.F("login", user.Login),
		observability.F("name", user.Name),
	)

	return user, nil
}

// parseGitHubUserJSON parses GitHub user JSON response
func parseGitHubUserJSON(jsonData string) (*GitHubUser, error) {
	// Simple JSON parsing - in a real implementation, you'd use encoding/json
	// For now, we'll extract basic fields using string operations

	user := &GitHubUser{}

	// Extract login
	if login := extractJSONField(jsonData, "login"); login != "" {
		user.Login = login
	}

	// Extract name
	if name := extractJSONField(jsonData, "name"); name != "" {
		user.Name = name
	}

	// Extract email
	if email := extractJSONField(jsonData, "email"); email != "" {
		user.Email = email
	}

	// Extract avatar_url
	if avatarURL := extractJSONField(jsonData, "avatar_url"); avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	return user, nil
}

// extractJSONField extracts a field value from JSON using simple string operations
func extractJSONField(jsonData, fieldName string) string {
	// Look for the field pattern: "fieldName": "value"
	pattern := fmt.Sprintf(`"%s":\s*"([^"]*)"`, fieldName)

	// Use regex to find the field value
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(jsonData)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}
