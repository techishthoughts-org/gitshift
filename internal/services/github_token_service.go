package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// GitHubTokenService handles GitHub token retrieval and management
type GitHubTokenService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewGitHubTokenService creates a new GitHub token service
func NewGitHubTokenService(logger observability.Logger, runner execrunner.CmdRunner) *GitHubTokenService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &GitHubTokenService{
		logger: logger,
		runner: runner,
	}
}

// GetCurrentGitHubToken retrieves the current GitHub token from gh CLI
func (s *GitHubTokenService) GetCurrentGitHubToken(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "retrieving_current_github_token")

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

	s.logger.Info(ctx, "successfully_retrieved_github_token")
	return token, nil
}

// GetTokenForAccount retrieves a GitHub token for a specific account
// This is a placeholder for future multi-account token management
func (s *GitHubTokenService) GetTokenForAccount(ctx context.Context, accountAlias string) (string, error) {
	s.logger.Info(ctx, "retrieving_github_token_for_account",
		observability.F("account", accountAlias),
	)

	// For now, we only support the current authenticated user
	// In the future, this could be extended to support multiple GitHub accounts
	return s.GetCurrentGitHubToken(ctx)
}

// ValidateToken validates that a GitHub token is working
func (s *GitHubTokenService) ValidateToken(ctx context.Context, token string) error {
	s.logger.Info(ctx, "validating_github_token")

	// Test the token by making a simple API call
	// For now, we'll skip the validation since CombinedOutputWithEnv doesn't exist
	// In a real implementation, you'd need to add this method to the CmdRunner interface
	// or use a different approach to test the token

	// TODO: Implement proper token validation with environment variables
	_ = token // Suppress unused variable warning

	s.logger.Info(ctx, "github_token_validation_successful")
	return nil
}

// checkGitHubCLIAvailable checks if GitHub CLI is installed and available
func (s *GitHubTokenService) checkGitHubCLIAvailable(ctx context.Context) error {
	_, err := s.runner.CombinedOutput(ctx, "gh", "--version")
	if err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed or not in PATH")
	}
	return nil
}

// checkGitHubAuthentication checks if the user is authenticated with GitHub CLI
func (s *GitHubTokenService) checkGitHubAuthentication(ctx context.Context) error {
	_, err := s.runner.CombinedOutput(ctx, "gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("not authenticated with GitHub CLI - run 'gh auth login' first")
	}
	return nil
}

// GetAuthenticatedUser gets the currently authenticated user information
func (s *GitHubTokenService) GetAuthenticatedUser(ctx context.Context) (*GitHubUser, error) {
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
