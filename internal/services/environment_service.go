package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// EnvironmentService manages environment variables and MCP server configuration
type EnvironmentService interface {
	UpdateMCPServerConfig(ctx context.Context, account, token string) error
	UpdateShellEnvironment(ctx context.Context, account, token string) error
	GetCurrentEnvironmentToken(ctx context.Context) (string, error)
	ValidateEnvironmentSetup(ctx context.Context) (*EnvironmentValidationResult, error)
	CleanupMCPConfig(ctx context.Context, account string) error
}

// RealEnvironmentService implements environment management
type RealEnvironmentService struct {
	logger       observability.Logger
	tokenStorage TokenStorageService
}

// EnvironmentValidationResult contains environment validation results
type EnvironmentValidationResult struct {
	MCPConfigExists   bool     `json:"mcp_config_exists"`
	MCPConfigPaths    []string `json:"mcp_config_paths"`
	ShellConfigExists bool     `json:"shell_config_exists"`
	CurrentToken      string   `json:"current_token,omitempty"`
	TokenSource       string   `json:"token_source"`
	Issues            []string `json:"issues"`
	Recommendations   []string `json:"recommendations"`
}

// NewEnvironmentService creates a new environment service
func NewEnvironmentService(logger observability.Logger, tokenStorage TokenStorageService) *RealEnvironmentService {
	return &RealEnvironmentService{
		logger:       logger,
		tokenStorage: tokenStorage,
	}
}

// UpdateMCPServerConfig updates MCP server configuration with the token
func (s *RealEnvironmentService) UpdateMCPServerConfig(ctx context.Context, account, token string) error {
	s.logger.Info(ctx, "updating_mcp_server_config",
		observability.F("account", account),
	)

	// Common MCP config locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	mcpConfigDirs := []string{
		filepath.Join(homeDir, ".config", "claude-code"),
		filepath.Join(homeDir, ".config", "claude"),
		filepath.Join(homeDir, ".claude"),
		filepath.Join(homeDir, ".config", "gitpersona", "mcp"),
	}

	updated := false
	for _, configDir := range mcpConfigDirs {
		if err := s.updateMCPConfigInDir(ctx, configDir, account, token); err != nil {
			s.logger.Warn(ctx, "failed_to_update_mcp_config_in_dir",
				observability.F("dir", configDir),
				observability.F("error", err.Error()),
			)
		} else {
			updated = true
		}
	}

	// Always create/update our own MCP config
	gitPersonaMCPDir := filepath.Join(homeDir, ".config", "gitpersona", "mcp")
	if err := s.updateMCPConfigInDir(ctx, gitPersonaMCPDir, account, token); err != nil {
		s.logger.Error(ctx, "failed_to_update_gitpersona_mcp_config",
			observability.F("error", err.Error()),
		)
	} else {
		updated = true
	}

	if !updated {
		return fmt.Errorf("failed to update any MCP configuration")
	}

	s.logger.Info(ctx, "mcp_server_config_updated_successfully",
		observability.F("account", account),
	)

	return nil
}

// updateMCPConfigInDir updates MCP configuration in a specific directory
func (s *RealEnvironmentService) updateMCPConfigInDir(ctx context.Context, configDir, account, token string) error {
	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Update environment file for MCP server
	envFile := filepath.Join(configDir, "github-token.env")
	envContent := fmt.Sprintf(`# GitPersona GitHub Token Configuration
# Account: %s
# Generated: %s
# DO NOT EDIT MANUALLY - Managed by GitPersona

export GITHUB_TOKEN="%s"
export GITHUB_PERSONAL_ACCESS_TOKEN="%s"
export GITHUB_TOKEN_GITPERSONA="%s"
export GITPERSONA_GITHUB_ACCOUNT="%s"
`, account, getCurrentTimestamp(), token, token, token, account)

	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	// Create or update the MCP server configuration
	mcpConfigFile := filepath.Join(configDir, "gitpersona-mcp.json")
	mcpConfig := fmt.Sprintf(`{
  "name": "gitpersona-github",
  "description": "GitPersona GitHub MCP Server Configuration",
  "env": {
    "GITHUB_TOKEN": "%s",
    "GITHUB_PERSONAL_ACCESS_TOKEN": "%s",
    "GITPERSONA_ACCOUNT": "%s"
  },
  "updated_at": "%s",
  "managed_by": "gitpersona"
}`, token, token, account, getCurrentTimestamp())

	if err := os.WriteFile(mcpConfigFile, []byte(mcpConfig), 0600); err != nil {
		return fmt.Errorf("failed to write MCP config file: %w", err)
	}

	s.logger.Info(ctx, "updated_mcp_config_in_directory",
		observability.F("dir", configDir),
		observability.F("account", account),
	)

	return nil
}

// UpdateShellEnvironment updates shell environment configuration
func (s *RealEnvironmentService) UpdateShellEnvironment(ctx context.Context, account, token string) error {
	s.logger.Info(ctx, "updating_shell_environment",
		observability.F("account", account),
	)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create GitPersona environment file
	gitPersonaEnvDir := filepath.Join(homeDir, ".config", "gitpersona")
	if err := os.MkdirAll(gitPersonaEnvDir, 0755); err != nil {
		return fmt.Errorf("failed to create gitpersona config directory: %w", err)
	}

	envFile := filepath.Join(gitPersonaEnvDir, "environment")
	envContent := fmt.Sprintf(`#!/bin/bash
# GitPersona Environment Configuration
# Current Account: %s
# Generated: %s
# Source this file to load GitPersona environment

export GITHUB_TOKEN="%s"
export GITHUB_PERSONAL_ACCESS_TOKEN="%s"
export GITPERSONA_CURRENT_ACCOUNT="%s"
export GITPERSONA_GITHUB_TOKEN="%s"

# Set up MCP server environment
if [ -f ~/.config/gitpersona/mcp/github-token.env ]; then
    source ~/.config/gitpersona/mcp/github-token.env
fi

# Function to reload GitPersona environment
gitpersona_reload_env() {
    if [ -f ~/.config/gitpersona/environment ]; then
        source ~/.config/gitpersona/environment
        echo "GitPersona environment reloaded for account: $GITPERSONA_CURRENT_ACCOUNT"
    fi
}

# Function to switch GitHub token for current session
gitpersona_set_token() {
    if [ $# -eq 0 ]; then
        echo "Usage: gitpersona_set_token <account>"
        return 1
    fi

    local account="$1"
    local token_file="$HOME/.config/gitpersona/tokens/$account.json"

    if [ -f "$token_file" ]; then
        echo "Switching to GitHub token for account: $account"
        export GITPERSONA_CURRENT_ACCOUNT="$account"
        # Note: Token extraction would require the GitPersona binary
        echo "Run 'gitpersona github-token get $account --export' to get the export command"
    else
        echo "No token found for account: $account"
        return 1
    fi
}
`, account, getCurrentTimestamp(), token, token, account, token)

	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	// Update shell rc files to source GitPersona environment
	if err := s.updateShellRCFiles(ctx, envFile); err != nil {
		s.logger.Warn(ctx, "failed_to_update_shell_rc_files",
			observability.F("error", err.Error()),
		)
	}

	s.logger.Info(ctx, "shell_environment_updated_successfully",
		observability.F("account", account),
		observability.F("env_file", envFile),
	)

	return nil
}

// updateShellRCFiles updates shell RC files to source GitPersona environment
func (s *RealEnvironmentService) updateShellRCFiles(ctx context.Context, envFile string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	shellFiles := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".profile"),
	}

	gitPersonaBlock := fmt.Sprintf(`
# GitPersona Environment - Managed by GitPersona
if [ -f "%s" ]; then
    source "%s"
fi
# End GitPersona Environment
`, envFile, envFile)

	for _, shellFile := range shellFiles {
		if _, err := os.Stat(shellFile); os.IsNotExist(err) {
			continue
		}

		// Read current content
		content, err := os.ReadFile(shellFile)
		if err != nil {
			s.logger.Warn(ctx, "failed_to_read_shell_file",
				observability.F("file", shellFile),
				observability.F("error", err.Error()),
			)
			continue
		}

		contentStr := string(content)

		// Check if GitPersona block already exists
		if strings.Contains(contentStr, "# GitPersona Environment - Managed by GitPersona") {
			// Remove old block and add new one
			lines := strings.Split(contentStr, "\n")
			var newLines []string
			skipUntilEnd := false

			for _, line := range lines {
				if strings.Contains(line, "# GitPersona Environment - Managed by GitPersona") {
					skipUntilEnd = true
					continue
				}
				if skipUntilEnd && strings.Contains(line, "# End GitPersona Environment") {
					skipUntilEnd = false
					continue
				}
				if !skipUntilEnd {
					newLines = append(newLines, line)
				}
			}

			contentStr = strings.Join(newLines, "\n")
		}

		// Add new GitPersona block
		newContent := contentStr + gitPersonaBlock

		// Write back to file
		if err := os.WriteFile(shellFile, []byte(newContent), 0644); err != nil {
			s.logger.Warn(ctx, "failed_to_update_shell_file",
				observability.F("file", shellFile),
				observability.F("error", err.Error()),
			)
			continue
		}

		s.logger.Info(ctx, "updated_shell_file",
			observability.F("file", shellFile),
		)
	}

	return nil
}

// GetCurrentEnvironmentToken retrieves the current GitHub token from environment
func (s *RealEnvironmentService) GetCurrentEnvironmentToken(ctx context.Context) (string, error) {
	// Check various environment variables
	tokenVars := []string{
		"GITHUB_TOKEN",
		"GITHUB_PERSONAL_ACCESS_TOKEN",
		"GITPERSONA_GITHUB_TOKEN",
	}

	for _, varName := range tokenVars {
		if token := os.Getenv(varName); token != "" {
			s.logger.Info(ctx, "found_token_in_environment",
				observability.F("var", varName),
				observability.F("token_prefix", token[:min(8, len(token))]),
			)
			return token, nil
		}
	}

	return "", fmt.Errorf("no GitHub token found in environment variables")
}

// ValidateEnvironmentSetup validates the current environment setup
func (s *RealEnvironmentService) ValidateEnvironmentSetup(ctx context.Context) (*EnvironmentValidationResult, error) {
	s.logger.Info(ctx, "validating_environment_setup")

	result := &EnvironmentValidationResult{
		MCPConfigPaths:  []string{},
		Issues:          []string{},
		Recommendations: []string{},
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Issues = append(result.Issues, "Failed to get user home directory")
		return result, nil
	}

	// Check MCP configuration
	mcpDirs := []string{
		filepath.Join(homeDir, ".config", "claude-code"),
		filepath.Join(homeDir, ".config", "claude"),
		filepath.Join(homeDir, ".claude"),
		filepath.Join(homeDir, ".config", "gitpersona", "mcp"),
	}

	for _, dir := range mcpDirs {
		envFile := filepath.Join(dir, "github-token.env")
		if _, err := os.Stat(envFile); err == nil {
			result.MCPConfigExists = true
			result.MCPConfigPaths = append(result.MCPConfigPaths, envFile)
		}
	}

	// Check shell environment
	gitPersonaEnv := filepath.Join(homeDir, ".config", "gitpersona", "environment")
	if _, err := os.Stat(gitPersonaEnv); err == nil {
		result.ShellConfigExists = true
	}

	// Check current token
	if token, err := s.GetCurrentEnvironmentToken(ctx); err == nil {
		result.CurrentToken = token[:min(8, len(token))] + "..."
		result.TokenSource = "environment"
	} else if s.tokenStorage != nil {
		// Try to get from storage
		if accounts, err := s.tokenStorage.ListTokens(ctx); err == nil && len(accounts) > 0 {
			result.TokenSource = "storage"
		}
	}

	// Generate recommendations
	if !result.MCPConfigExists {
		result.Recommendations = append(result.Recommendations,
			"Run 'gitpersona github-token set' to configure MCP server authentication")
	}

	if !result.ShellConfigExists {
		result.Recommendations = append(result.Recommendations,
			"Run 'gitpersona environment setup' to configure shell environment")
	}

	if result.TokenSource == "" {
		result.Issues = append(result.Issues,
			"No GitHub token found in environment or storage")
		result.Recommendations = append(result.Recommendations,
			"Set up a GitHub token using 'gitpersona github-token set'")
	}

	s.logger.Info(ctx, "environment_validation_completed",
		observability.F("mcp_config_exists", result.MCPConfigExists),
		observability.F("shell_config_exists", result.ShellConfigExists),
		observability.F("token_source", result.TokenSource),
	)

	return result, nil
}

// CleanupMCPConfig removes MCP configuration for an account
func (s *RealEnvironmentService) CleanupMCPConfig(ctx context.Context, account string) error {
	s.logger.Info(ctx, "cleaning_up_mcp_config",
		observability.F("account", account),
	)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Clean up GitPersona MCP config
	mcpDir := filepath.Join(homeDir, ".config", "gitpersona", "mcp")

	filesToRemove := []string{
		filepath.Join(mcpDir, "github-token.env"),
		filepath.Join(mcpDir, "gitpersona-mcp.json"),
	}

	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			s.logger.Warn(ctx, "failed_to_remove_file",
				observability.F("file", file),
				observability.F("error", err.Error()),
			)
		}
	}

	s.logger.Info(ctx, "mcp_config_cleanup_completed",
		observability.F("account", account),
	)

	return nil
}

// Helper function to get current timestamp
func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

// Helper function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
