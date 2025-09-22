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

// RealZshrcService manages the .zshrc file for environment variables
type RealZshrcService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewZshrcService creates a new zshrc service
func NewZshrcService(logger observability.Logger, runner execrunner.CmdRunner) *RealZshrcService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealZshrcService{
		logger: logger,
		runner: runner,
	}
}

// GetZshrcPath returns the path to the .zshrc file
func (s *RealZshrcService) GetZshrcPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Common locations for .zshrc file
	possiblePaths := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".config", "zshrc"),
		filepath.Join(homeDir, ".config", ".zshrc"),
	}

	// Check which path exists
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			s.logger.Info(context.Background(), "found_zshrc_file",
				observability.F("path", path),
			)
			return path, nil
		}
	}

	// If none exist, use the default location
	defaultPath := filepath.Join(homeDir, ".zshrc")
	s.logger.Info(context.Background(), "using_default_zshrc_path",
		observability.F("path", defaultPath),
	)
	return defaultPath, nil
}

// UpdateGitHubToken updates the GITHUB_TOKEN in the .zshrc file
func (s *RealZshrcService) UpdateGitHubToken(ctx context.Context, token string) error {
	s.logger.Info(ctx, "updating_github_token_in_zshrc")

	zshrcPath, err := s.GetZshrcPath()
	if err != nil {
		return fmt.Errorf("failed to get .zshrc path: %w", err)
	}

	// Read current file content
	content, err := s.readZshrcFile(zshrcPath)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc file: %w", err)
	}

	// Update or add GITHUB_TOKEN
	updatedContent := s.updateTokenInContent(content, token)

	// Write back to file
	if err := s.writeZshrcFile(zshrcPath, updatedContent); err != nil {
		return fmt.Errorf("failed to write .zshrc file: %w", err)
	}

	s.logger.Info(ctx, "github_token_updated_in_zshrc",
		observability.F("path", zshrcPath),
	)

	return nil
}

// GetCurrentGitHubToken retrieves the current GITHUB_TOKEN from .zshrc
func (s *RealZshrcService) GetCurrentGitHubToken(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "getting_current_github_token_from_zshrc")

	zshrcPath, err := s.GetZshrcPath()
	if err != nil {
		return "", fmt.Errorf("failed to get .zshrc path: %w", err)
	}

	content, err := s.readZshrcFile(zshrcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .zshrc file: %w", err)
	}

	// Extract GITHUB_TOKEN from content
	token := s.extractTokenFromContent(content)
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN not found in .zshrc file")
	}

	s.logger.Info(ctx, "retrieved_github_token_from_zshrc")
	return token, nil
}

// readZshrcFile reads the content of the .zshrc file
func (s *RealZshrcService) readZshrcFile(path string) (string, error) {
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

// writeZshrcFile writes content to the .zshrc file
func (s *RealZshrcService) writeZshrcFile(path string, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with secure permissions (644)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updateTokenInContent updates or adds the GITHUB_TOKEN in the file content
func (s *RealZshrcService) updateTokenInContent(content, token string) string {
	// Pattern to match GITHUB_TOKEN export line
	pattern := regexp.MustCompile(`(?m)^\s*export\s+GITHUB_TOKEN\s*=\s*["']?[^"'\n]*["']?\s*$`)

	// New token line
	newLine := fmt.Sprintf("export GITHUB_TOKEN=\"%s\"", token)

	// Check if GITHUB_TOKEN already exists
	if pattern.MatchString(content) {
		// Replace ALL existing GITHUB_TOKEN lines with a single new line
		updated := pattern.ReplaceAllString(content, "")

		// Clean up any extra newlines that might have been left
		updated = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(updated, "\n\n")

		// Add the new token line
		if !strings.HasSuffix(updated, "\n") {
			updated += "\n"
		}
		updated += newLine + "\n"

		return updated
	}

	// Add new line at the end
	if content == "" {
		return newLine + "\n"
	}

	// Ensure content ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content + newLine + "\n"
}

// extractTokenFromContent extracts the GITHUB_TOKEN value from file content
func (s *RealZshrcService) extractTokenFromContent(content string) string {
	// Pattern to match and capture GITHUB_TOKEN value
	pattern := regexp.MustCompile(`(?m)^\s*export\s+GITHUB_TOKEN\s*=\s*["']?([^"'\n]*)["']?\s*$`)

	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// ReloadZshrc reloads the .zshrc file in the current shell
func (s *RealZshrcService) ReloadZshrc(ctx context.Context) error {
	s.logger.Info(ctx, "reloading_zshrc")

	zshrcPath, err := s.GetZshrcPath()
	if err != nil {
		return fmt.Errorf("failed to get .zshrc path: %w", err)
	}

	// Source the .zshrc file
	cmd := fmt.Sprintf("source %s", zshrcPath)
	_, err = s.runner.CombinedOutput(ctx, "zsh", "-c", cmd)
	if err != nil {
		s.logger.Info(ctx, "failed_to_reload_zshrc",
			observability.F("error", err.Error()),
		)
		// Don't fail the entire operation if reload fails
		return nil
	}

	s.logger.Info(ctx, "zshrc_reloaded_successfully")
	return nil
}

// ValidateZshrcFile validates that the .zshrc file is properly formatted
func (s *RealZshrcService) ValidateZshrcFile(ctx context.Context) error {
	s.logger.Info(ctx, "validating_zshrc_file")

	zshrcPath, err := s.GetZshrcPath()
	if err != nil {
		return fmt.Errorf("failed to get .zshrc path: %w", err)
	}

	content, err := s.readZshrcFile(zshrcPath)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc file: %w", err)
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

	s.logger.Info(ctx, "zshrc_file_validation_passed")
	return nil
}

// AddGitPersonaSection adds a GitPersona section to .zshrc if it doesn't exist
func (s *RealZshrcService) AddGitPersonaSection(ctx context.Context) error {
	s.logger.Info(ctx, "adding_gitpersona_section_to_zshrc")

	zshrcPath, err := s.GetZshrcPath()
	if err != nil {
		return fmt.Errorf("failed to get .zshrc path: %w", err)
	}

	content, err := s.readZshrcFile(zshrcPath)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc file: %w", err)
	}

	// Check if GitPersona section already exists
	if strings.Contains(content, "# GITPERSONA CONFIGURATION") {
		s.logger.Info(ctx, "gitpersona_section_already_exists")
		return nil
	}

	// Add GitPersona section at the end
	gitpersonaSection := `

# =============================================================================
# GITPERSONA CONFIGURATION
# =============================================================================

# GitPersona environment variables
# This section is automatically managed by GitPersona
# Do not modify manually - use 'gitpersona switch' command instead

# GITHUB_TOKEN will be updated automatically when switching accounts
export GITHUB_TOKEN=""

# GitPersona completion and lazy loading
if command -v gitpersona >/dev/null 2>&1; then
    # Lazy load GitPersona completion
    _gitpersona() {
        unfunction _gitpersona
        eval "$(gitpersona init)"
    }
    compdef _gitpersona gitpersona
fi
`

	// Ensure content ends with newline
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	updatedContent := content + gitpersonaSection

	// Write back to file
	if err := s.writeZshrcFile(zshrcPath, updatedContent); err != nil {
		return fmt.Errorf("failed to write .zshrc file: %w", err)
	}

	s.logger.Info(ctx, "gitpersona_section_added_to_zshrc",
		observability.F("path", zshrcPath),
	)

	return nil
}
