package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealZshSecretsService manages the zsh_secrets file for environment variables
type RealZshSecretsService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewZshSecretsService creates a new zsh secrets service
func NewZshSecretsService(logger observability.Logger, runner execrunner.CmdRunner) *RealZshSecretsService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealZshSecretsService{
		logger: logger,
		runner: runner,
	}
}

// GetZshSecretsPath returns the path to the zsh_secrets file
func (s *RealZshSecretsService) GetZshSecretsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Common locations for zsh_secrets file
	possiblePaths := []string{
		filepath.Join(homeDir, ".zsh_secrets"),
		filepath.Join(homeDir, ".config", "zsh_secrets"),
		filepath.Join(homeDir, ".secrets", "zsh_secrets"),
		filepath.Join(homeDir, ".zsh", "secrets"),
	}

	// Check which path exists
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			s.logger.Info(context.Background(), "found_zsh_secrets_file",
				observability.F("path", path),
			)
			return path, nil
		}
	}

	// If none exist, use the default location
	defaultPath := filepath.Join(homeDir, ".zsh_secrets")
	s.logger.Info(context.Background(), "using_default_zsh_secrets_path",
		observability.F("path", defaultPath),
	)
	return defaultPath, nil
}

// UpdateGitHubToken updates the GITHUB_TOKEN in the zsh_secrets file
func (s *RealZshSecretsService) UpdateGitHubToken(ctx context.Context, token string) error {
	s.logger.Info(ctx, "updating_github_token_in_zsh_secrets")

	secretsPath, err := s.GetZshSecretsPath()
	if err != nil {
		return fmt.Errorf("failed to get zsh_secrets path: %w", err)
	}

	// Read current file content
	content, err := s.readSecretsFile(secretsPath)
	if err != nil {
		return fmt.Errorf("failed to read zsh_secrets file: %w", err)
	}

	// Update or add GITHUB_TOKEN
	updatedContent := s.updateTokenInContent(content, token)

	// Write back to file
	if err := s.writeSecretsFile(secretsPath, updatedContent); err != nil {
		return fmt.Errorf("failed to write zsh_secrets file: %w", err)
	}

	s.logger.Info(ctx, "github_token_updated_in_zsh_secrets",
		observability.F("path", secretsPath),
	)

	return nil
}

// GetCurrentGitHubToken retrieves the current GITHUB_TOKEN from zsh_secrets
func (s *RealZshSecretsService) GetCurrentGitHubToken(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "getting_current_github_token_from_zsh_secrets")

	secretsPath, err := s.GetZshSecretsPath()
	if err != nil {
		return "", fmt.Errorf("failed to get zsh_secrets path: %w", err)
	}

	content, err := s.readSecretsFile(secretsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read zsh_secrets file: %w", err)
	}

	// Extract GITHUB_TOKEN from content
	token := s.extractTokenFromContent(content)
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN not found in zsh_secrets file")
	}

	s.logger.Info(ctx, "retrieved_github_token_from_zsh_secrets")
	return token, nil
}

// readSecretsFile reads the content of the zsh_secrets file
func (s *RealZshSecretsService) readSecretsFile(path string) (string, error) {
	// If file doesn't exist, return empty content
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// writeSecretsFile writes content to the zsh_secrets file
func (s *RealZshSecretsService) writeSecretsFile(path string, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with secure permissions (600)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updateTokenInContent updates or adds the GITHUB_TOKEN in the file content
func (s *RealZshSecretsService) updateTokenInContent(content, token string) string {
	// Pattern to match GITHUB_TOKEN export line (including GitPersona managed ones)
	pattern := regexp.MustCompile(`(?m)^\s*export\s+GITHUB_TOKEN\s*=\s*["']?[^"'\n]*["']?\s*$`)

	// New token lines with GitPersona integration
	newLines := fmt.Sprintf(`# GitPersona managed GitHub token - do not edit manually
export GITHUB_TOKEN="%s"
export GITHUB_PERSONAL_ACCESS_TOKEN="%s"`, token, token)

	// Check if GITHUB_TOKEN already exists
	if pattern.MatchString(content) {
		// Replace existing line with GitPersona managed block
		updated := pattern.ReplaceAllString(content, newLines)
		return updated
	}

	// Add new lines at the end
	if content == "" {
		return newLines + "\n"
	}

	// Ensure content ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content + newLines + "\n"
}

// extractTokenFromContent extracts the GITHUB_TOKEN value from file content
func (s *RealZshSecretsService) extractTokenFromContent(content string) string {
	// Pattern to match and capture GITHUB_TOKEN value
	pattern := regexp.MustCompile(`(?m)^\s*export\s+GITHUB_TOKEN\s*=\s*["']?([^"'\n]*)["']?\s*$`)

	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// ReloadZshSecrets reloads the zsh_secrets file in the current shell
func (s *RealZshSecretsService) ReloadZshSecrets(ctx context.Context) error {
	s.logger.Info(ctx, "reloading_zsh_secrets")

	secretsPath, err := s.GetZshSecretsPath()
	if err != nil {
		return fmt.Errorf("failed to get zsh_secrets path: %w", err)
	}

	// Source the zsh_secrets file
	cmd := fmt.Sprintf("source %s", secretsPath)
	_, err = s.runner.CombinedOutput(ctx, "zsh", "-c", cmd)
	if err != nil {
		s.logger.Info(ctx, "failed_to_reload_zsh_secrets",
			observability.F("error", err.Error()),
		)
		// Don't fail the entire operation if reload fails
		return nil
	}

	s.logger.Info(ctx, "zsh_secrets_reloaded_successfully")
	return nil
}

// ValidateZshSecretsFile validates that the zsh_secrets file is properly formatted
func (s *RealZshSecretsService) ValidateZshSecretsFile(ctx context.Context) error {
	s.logger.Info(ctx, "validating_zsh_secrets_file")

	secretsPath, err := s.GetZshSecretsPath()
	if err != nil {
		return fmt.Errorf("failed to get zsh_secrets path: %w", err)
	}

	content, err := s.readSecretsFile(secretsPath)
	if err != nil {
		return fmt.Errorf("failed to read zsh_secrets file: %w", err)
	}

	// Basic validation - check for proper export syntax
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line looks like an export statement
		if strings.HasPrefix(line, "export ") {
			// Basic syntax check
			if !regexp.MustCompile(`^export\s+\w+\s*=\s*["']?[^"'\n]*["']?\s*$`).MatchString(line) {
				return fmt.Errorf("invalid export syntax on line %d: %s", i+1, line)
			}
		}
	}

	s.logger.Info(ctx, "zsh_secrets_file_validation_passed")
	return nil
}
