package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// GitPersonaTokenService integrates token storage with GitPersona's environment management
type GitPersonaTokenService struct {
	logger       observability.Logger
	tokenStorage TokenStorageService
	envService   EnvironmentService
}

// NewGitPersonaTokenService creates a new integrated token service
func NewGitPersonaTokenService(logger observability.Logger) (*GitPersonaTokenService, error) {
	tokenStorage, err := NewTokenStorageService(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token storage: %w", err)
	}

	envService := NewEnvironmentService(logger, tokenStorage)

	return &GitPersonaTokenService{
		logger:       logger,
		tokenStorage: tokenStorage,
		envService:   envService,
	}, nil
}

// SetTokenForAccount stores a token and updates environment configuration
func (s *GitPersonaTokenService) SetTokenForAccount(ctx context.Context, account, token string) error {
	s.logger.Info(ctx, "setting_token_for_account",
		observability.F("account", account),
	)

	// Store token
	if err := s.tokenStorage.StoreToken(ctx, account, token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Update environment configuration
	if err := s.envService.UpdateMCPServerConfig(ctx, account, token); err != nil {
		s.logger.Warn(ctx, "failed_to_update_mcp_config",
			observability.F("account", account),
			observability.F("error", err.Error()),
		)
	}

	if err := s.envService.UpdateShellEnvironment(ctx, account, token); err != nil {
		s.logger.Warn(ctx, "failed_to_update_shell_env",
			observability.F("account", account),
			observability.F("error", err.Error()),
		)
	}

	s.logger.Info(ctx, "token_and_environment_updated_successfully",
		observability.F("account", account),
	)

	return nil
}

// GetTokenForAccount retrieves a token for an account
func (s *GitPersonaTokenService) GetTokenForAccount(ctx context.Context, account string) (string, error) {
	return s.tokenStorage.GetToken(ctx, account)
}

// GetCurrentToken retrieves the token for the current account from environment or storage
func (s *GitPersonaTokenService) GetCurrentToken(ctx context.Context) (string, error) {
	// Try environment first
	if token, err := s.envService.GetCurrentEnvironmentToken(ctx); err == nil {
		return token, nil
	}

	// Try default account from storage
	if token, err := s.tokenStorage.GetToken(ctx, "default"); err == nil {
		return token, nil
	}

	// List tokens and try the first one
	accounts, err := s.tokenStorage.ListTokens(ctx)
	if err != nil {
		return "", fmt.Errorf("no tokens available: %w", err)
	}

	if len(accounts) == 0 {
		return "", fmt.Errorf("no tokens stored")
	}

	// Return token for the first account
	return s.tokenStorage.GetToken(ctx, accounts[0])
}

// SyncWithEnvironment ensures environment is synchronized with stored tokens
func (s *GitPersonaTokenService) SyncWithEnvironment(ctx context.Context, account string) error {
	s.logger.Info(ctx, "syncing_token_with_environment",
		observability.F("account", account),
	)

	// Get token from storage
	token, err := s.tokenStorage.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get token for account %s: %w", account, err)
	}

	// Update environment
	if err := s.envService.UpdateMCPServerConfig(ctx, account, token); err != nil {
		return fmt.Errorf("failed to update MCP config: %w", err)
	}

	if err := s.envService.UpdateShellEnvironment(ctx, account, token); err != nil {
		return fmt.Errorf("failed to update shell environment: %w", err)
	}

	s.logger.Info(ctx, "token_environment_sync_completed",
		observability.F("account", account),
	)

	return nil
}

// ValidateTokenAndEnvironment validates both token and environment setup
func (s *GitPersonaTokenService) ValidateTokenAndEnvironment(ctx context.Context, account string) (*TokenEnvironmentValidation, error) {
	s.logger.Info(ctx, "validating_token_and_environment",
		observability.F("account", account),
	)

	result := &TokenEnvironmentValidation{
		Account: account,
		Issues:  []string{},
	}

	// Check if token exists in storage
	token, err := s.tokenStorage.GetToken(ctx, account)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("No token stored for account: %s", account))
		result.TokenValid = false
	} else {
		result.TokenValid = true
		result.TokenPrefix = token[:min(8, len(token))] + "..."

		// Validate token
		if validation, err := s.tokenStorage.ValidateToken(ctx, token); err == nil {
			result.TokenAPIValid = validation.Valid
			if !validation.Valid {
				result.Issues = append(result.Issues, "Token failed API validation")
			}
		}
	}

	// Validate environment
	envValidation, err := s.envService.ValidateEnvironmentSetup(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Environment validation failed: %v", err))
	} else {
		result.MCPConfigured = envValidation.MCPConfigExists
		result.ShellConfigured = envValidation.ShellConfigExists
		result.Issues = append(result.Issues, envValidation.Issues...)
	}

	result.Valid = result.TokenValid && result.TokenAPIValid && result.MCPConfigured

	s.logger.Info(ctx, "token_environment_validation_completed",
		observability.F("account", account),
		observability.F("valid", result.Valid),
	)

	return result, nil
}

// CreateDefaultConfiguration creates a default token configuration
func (s *GitPersonaTokenService) CreateDefaultConfiguration(ctx context.Context) error {
	s.logger.Info(ctx, "creating_default_configuration")

	// Check if we have any tokens
	accounts, err := s.tokenStorage.ListTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tokens: %w", err)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no tokens available to configure")
	}

	// Use the first account as default
	defaultAccount := accounts[0]
	return s.SyncWithEnvironment(ctx, defaultAccount)
}

// ExportTokenToFile exports a token to a shell-compatible file
func (s *GitPersonaTokenService) ExportTokenToFile(ctx context.Context, account, filePath string) error {
	s.logger.Info(ctx, "exporting_token_to_file",
		observability.F("account", account),
		observability.F("file", filePath),
	)

	// Get token
	token, err := s.tokenStorage.GetToken(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Create export content
	content := fmt.Sprintf(`#!/bin/bash
# GitPersona Token Export for Account: %s
# Generated: %s
# DO NOT COMMIT THIS FILE TO VERSION CONTROL

export GITHUB_TOKEN="%s"
export GITHUB_PERSONAL_ACCESS_TOKEN="%s"
export GITPERSONA_CURRENT_ACCOUNT="%s"

echo "Loaded GitHub token for account: %s"
`, account, getCurrentTimestamp(), token, token, account, account)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	s.logger.Info(ctx, "token_exported_successfully",
		observability.F("account", account),
		observability.F("file", filePath),
	)

	return nil
}

// TokenEnvironmentValidation contains validation results
type TokenEnvironmentValidation struct {
	Account         string   `json:"account"`
	Valid           bool     `json:"valid"`
	TokenValid      bool     `json:"token_valid"`
	TokenAPIValid   bool     `json:"token_api_valid"`
	TokenPrefix     string   `json:"token_prefix,omitempty"`
	MCPConfigured   bool     `json:"mcp_configured"`
	ShellConfigured bool     `json:"shell_configured"`
	Issues          []string `json:"issues"`
}
