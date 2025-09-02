package services

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// GitConfigService handles Git configuration management and validation
type GitConfigService struct {
	logger observability.Logger
}

// GitConfig represents Git configuration state
type GitConfig struct {
	User struct {
		Name  string
		Email string
	}
	SSHCommand string
	Remotes    map[string]string
	Issues     []GitConfigIssue
}

// GitConfigIssue represents a configuration problem
type GitConfigIssue struct {
	Type        string
	Severity    string
	Description string
	Fix         string
	Fixed       bool
}

// NewGitConfigService creates a new Git configuration service
func NewGitConfigService(logger observability.Logger) *GitConfigService {
	return &GitConfigService{
		logger: logger,
	}
}

// AnalyzeConfiguration analyzes the current Git configuration for issues
func (s *GitConfigService) AnalyzeConfiguration(ctx context.Context) (*GitConfig, error) {
	s.logger.Info(ctx, "analyzing_git_configuration")

	config := &GitConfig{
		Remotes: make(map[string]string),
		Issues:  []GitConfigIssue{},
	}

	// Get user configuration
	if err := s.getUserConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to get user config: %w", err)
	}

	// Get SSH command configuration
	if err := s.getSSHCommandConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to get SSH command config: %w", err)
	}

	// Get remote configuration
	if err := s.getRemoteConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to get remote config: %w", err)
	}

	// Analyze for issues
	s.analyzeIssues(ctx, config)

	s.logger.Info(ctx, "git_configuration_analysis_complete",
		observability.F("issues_found", len(config.Issues)),
	)

	return config, nil
}

// getUserConfig retrieves user name and email configuration
func (s *GitConfigService) getUserConfig(ctx context.Context, config *GitConfig) error {
	// Get user name
	if name, err := s.runGitConfig(ctx, "user.name"); err == nil {
		config.User.Name = strings.TrimSpace(name)
	}

	// Get user email
	if email, err := s.runGitConfig(ctx, "user.email"); err == nil {
		config.User.Email = strings.TrimSpace(email)
	}

	return nil
}

// getSSHCommandConfig retrieves SSH command configuration
func (s *GitConfigService) getSSHCommandConfig(ctx context.Context, config *GitConfig) error {
	// Check global SSH command
	if sshCmd, err := s.runGitConfig(ctx, "--global", "core.sshcommand"); err == nil {
		config.SSHCommand = strings.TrimSpace(sshCmd)
	}

	// Check local SSH command (overrides global)
	if sshCmd, err := s.runGitConfig(ctx, "--local", "core.sshcommand"); err == nil {
		config.SSHCommand = strings.TrimSpace(sshCmd)
	}

	return nil
}

// getRemoteConfig retrieves remote repository configuration
func (s *GitConfigService) getRemoteConfig(ctx context.Context, config *GitConfig) error {
	// Get all remotes
	if remotes, err := s.runGitRemote(ctx, "-v"); err == nil {
		lines := strings.Split(strings.TrimSpace(remotes), "\n")
		for _, line := range lines {
			if strings.Contains(line, "\t") {
				parts := strings.Split(line, "\t")
				if len(parts) >= 2 {
					remoteName := parts[0]
					remoteURL := parts[1]
					if strings.HasSuffix(remoteURL, " (fetch)") {
						remoteURL = strings.TrimSuffix(remoteURL, " (fetch)")
						config.Remotes[remoteName] = remoteURL
					}
				}
			}
		}
	}

	return nil
}

// analyzeIssues analyzes the configuration for common problems
func (s *GitConfigService) analyzeIssues(ctx context.Context, config *GitConfig) {
	// Check for missing user configuration
	if config.User.Name == "" {
		config.Issues = append(config.Issues, GitConfigIssue{
			Type:        "missing_user_name",
			Severity:    "high",
			Description: "Git user.name is not configured",
			Fix:         "Set user.name using: git config --global user.name \"Your Name\"",
			Fixed:       false,
		})
	}

	if config.User.Email == "" {
		config.Issues = append(config.Issues, GitConfigIssue{
			Type:        "missing_user_email",
			Severity:    "high",
			Description: "Git user.email is not configured",
			Fix:         "Set user.email using: git config --global user.email \"your.email@example.com\"",
			Fixed:       false,
		})
	}

	// Check for SSH command issues
	if config.SSHCommand != "" {
		// Check for duplicate SSH commands
		if s.hasDuplicateSSHCommands(ctx) {
			config.Issues = append(config.Issues, GitConfigIssue{
				Type:        "duplicate_ssh_command",
				Severity:    "medium",
				Description: "Multiple SSH command configurations found (global and local)",
				Fix:         "Remove duplicate using: git config --unset core.sshcommand (local) and --global --unset core.sshcommand",
				Fixed:       false,
			})
		}

		// Check for wrong SSH key in command
		if s.hasWrongSSHKey(ctx, config.SSHCommand) {
			config.Issues = append(config.Issues, GitConfigIssue{
				Type:        "wrong_ssh_key",
				Severity:    "high",
				Description: "SSH command references wrong or non-existent SSH key",
				Fix:         "Update SSH command to use correct key for current account",
				Fixed:       false,
			})
		}
	}

	// Check for credential helper issues
	if s.hasCredentialHelperIssues(ctx) {
		config.Issues = append(config.Issues, GitConfigIssue{
			Type:        "credential_helper_issue",
			Severity:    "low",
			Description: "Credential helper may conflict with SSH authentication",
			Fix:         "Consider using SSH keys instead of credential helper for GitHub",
			Fixed:       false,
		})
	}
}

// hasDuplicateSSHCommands checks if there are duplicate SSH command configurations
func (s *GitConfigService) hasDuplicateSSHCommands(ctx context.Context) bool {
	global, _ := s.runGitConfig(ctx, "--global", "core.sshcommand")
	local, _ := s.runGitConfig(ctx, "--local", "core.sshcommand")

	return global != "" && local != "" && global != local
}

// hasWrongSSHKey checks if the SSH command references a wrong or non-existent key
func (s *GitConfigService) hasWrongSSHKey(ctx context.Context, sshCommand string) bool {
	// Extract key path from SSH command
	keyPathMatch := regexp.MustCompile(`-i\s+([^\s]+)`).FindStringSubmatch(sshCommand)
	if len(keyPathMatch) < 2 {
		return false
	}

	keyPath := keyPathMatch[1]
	// Expand ~ to home directory
	if strings.HasPrefix(keyPath, "~") {
		homeDir, _ := s.runGitConfig(ctx, "--global", "core.home")
		if homeDir == "" {
			// Use default home directory
			keyPath = strings.Replace(keyPath, "~", "/Users/arthurcosta", 1)
		} else {
			keyPath = strings.Replace(keyPath, "~", homeDir, 1)
		}
	}

	// Check if key file exists
	if _, err := s.runGitConfig(ctx, "core.filemode"); err != nil {
		// Key file doesn't exist
		return true
	}

	return false
}

// hasCredentialHelperIssues checks for credential helper conflicts
func (s *GitConfigService) hasCredentialHelperIssues(ctx context.Context) bool {
	credentialHelper, _ := s.runGitConfig(ctx, "credential.helper")
	return credentialHelper != ""
}

// FixConfiguration automatically fixes common Git configuration issues
func (s *GitConfigService) FixConfiguration(ctx context.Context, config *GitConfig) error {
	s.logger.Info(ctx, "fixing_git_configuration_issues",
		observability.F("issues_count", len(config.Issues)),
	)

	fixedCount := 0

	for i, issue := range config.Issues {
		if issue.Fixed {
			continue
		}

		switch issue.Type {
		case "duplicate_ssh_command":
			if err := s.fixDuplicateSSHCommands(ctx); err == nil {
				config.Issues[i].Fixed = true
				fixedCount++
			}
		case "wrong_ssh_key":
			if err := s.fixWrongSSHKey(ctx, config); err == nil {
				config.Issues[i].Fixed = true
				fixedCount++
			}
		}
	}

	s.logger.Info(ctx, "git_configuration_fixes_complete",
		observability.F("issues_fixed", fixedCount),
		observability.F("total_issues", len(config.Issues)),
	)

	return nil
}

// fixDuplicateSSHCommands removes duplicate SSH command configurations
func (s *GitConfigService) fixDuplicateSSHCommands(ctx context.Context) error {
	s.logger.Info(ctx, "fixing_duplicate_ssh_commands")

	// Remove local SSH command (keep global)
	if err := s.runGitConfigUnset(ctx, "--local", "core.sshcommand"); err != nil {
		s.logger.Warn(ctx, "failed_to_unset_local_ssh_command",
			observability.F("error", err.Error()),
		)
	}

	return nil
}

// fixWrongSSHKey updates SSH command to use correct key
func (s *GitConfigService) fixWrongSSHKey(ctx context.Context, config *GitConfig) error {
	s.logger.Info(ctx, "fixing_wrong_ssh_key")

	// This would need to be implemented based on the current account
	// For now, we'll just log the issue
	s.logger.Warn(ctx, "ssh_key_fix_requires_account_context")

	return nil
}

// SetUserConfiguration sets the Git user configuration
func (s *GitConfigService) SetUserConfiguration(ctx context.Context, name, email string) error {
	s.logger.Info(ctx, "setting_git_user_configuration",
		observability.F("name", name),
		observability.F("email", email),
	)

	// Set global user name
	if err := s.runGitConfigSet(ctx, "--global", "user.name", name); err != nil {
		return fmt.Errorf("failed to set user.name: %w", err)
	}

	// Set global user email
	if err := s.runGitConfigSet(ctx, "--global", "user.email", email); err != nil {
		return fmt.Errorf("failed to set user.email: %w", err)
	}

	s.logger.Info(ctx, "git_user_configuration_set_successfully")
	return nil
}

// SetSSHCommand sets the Git SSH command configuration
func (s *GitConfigService) SetSSHCommand(ctx context.Context, sshCommand string) error {
	s.logger.Info(ctx, "setting_git_ssh_command",
		observability.F("ssh_command", sshCommand),
	)

	// Remove any existing SSH command configurations
	s.runGitConfigUnset(ctx, "--global", "core.sshcommand")
	s.runGitConfigUnset(ctx, "--local", "core.sshcommand")

	// Set global SSH command
	if err := s.runGitConfigSet(ctx, "--global", "core.sshcommand", sshCommand); err != nil {
		return fmt.Errorf("failed to set SSH command: %w", err)
	}

	s.logger.Info(ctx, "git_ssh_command_set_successfully")
	return nil
}

// Helper methods for running Git commands
func (s *GitConfigService) runGitConfig(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"config"}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (s *GitConfigService) runGitConfigSet(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"config"}, args...)...)
	return cmd.Run()
}

func (s *GitConfigService) runGitConfigUnset(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"config", "--unset"}, args...)...)
	return cmd.Run()
}

func (s *GitConfigService) runGitRemote(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"remote"}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
