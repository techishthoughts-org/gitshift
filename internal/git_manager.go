package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealGitManager implements the GitManager interface
type RealGitManager struct {
	logger observability.Logger
}

// NewGitManager creates a new Git manager
func NewGitManager(logger observability.Logger) GitManager {
	return &RealGitManager{
		logger: logger,
	}
}

// SetGlobalConfig sets global Git configuration for an account
func (gm *RealGitManager) SetGlobalConfig(ctx context.Context, account *Account) error {
	gm.logger.Info(ctx, "setting_global_git_config",
		observability.F("account", account.Alias),
		observability.F("name", account.Name),
		observability.F("email", account.Email),
	)

	// Set user.name
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name", account.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		gm.logger.Error(ctx, "failed_to_set_git_name",
			observability.F("account", account.Alias),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	// Set user.email
	cmd = exec.CommandContext(ctx, "git", "config", "--global", "user.email", account.Email)
	if output, err := cmd.CombinedOutput(); err != nil {
		gm.logger.Error(ctx, "failed_to_set_git_email",
			observability.F("account", account.Alias),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	// Set signing key if available
	if account.SSHKeyPath != "" {
		pubKeyPath := account.SSHKeyPath + ".pub"
		if _, err := os.Stat(pubKeyPath); err == nil {
			cmd = exec.CommandContext(ctx, "git", "config", "--global", "user.signingkey", pubKeyPath)
			if err := cmd.Run(); err != nil {
				gm.logger.Warn(ctx, "failed_to_set_signing_key",
					observability.F("account", account.Alias),
					observability.F("error", err.Error()),
				)
			} else {
				// Enable commit signing
				cmd = exec.CommandContext(ctx, "git", "config", "--global", "commit.gpgsign", "true")
				cmd.Run() // Best effort
			}
		}
	}

	gm.logger.Info(ctx, "global_git_config_set",
		observability.F("account", account.Alias),
	)

	return nil
}

// SetLocalConfig sets local Git configuration for an account in a specific repository
func (gm *RealGitManager) SetLocalConfig(ctx context.Context, account *Account, repoPath string) error {
	gm.logger.Info(ctx, "setting_local_git_config",
		observability.F("account", account.Alias),
		observability.F("repo_path", repoPath),
	)

	// Change to repository directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(repoPath); err != nil {
		return fmt.Errorf("failed to change to repository directory: %w", err)
	}

	defer func() {
		os.Chdir(originalDir)
	}()

	// Verify this is a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Set user.name locally
	cmd := exec.CommandContext(ctx, "git", "config", "user.name", account.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set local git user.name: %w", err)
	}

	// Set user.email locally
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", account.Email)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set local git user.email: %w", err)
	}

	// Set SSH command for this repository if key is available
	if account.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", account.SSHKeyPath)
		cmd = exec.CommandContext(ctx, "git", "config", "core.sshCommand", sshCmd)
		if err := cmd.Run(); err != nil {
			gm.logger.Warn(ctx, "failed_to_set_ssh_command",
				observability.F("account", account.Alias),
				observability.F("error", err.Error()),
			)
		}
	}

	gm.logger.Info(ctx, "local_git_config_set",
		observability.F("account", account.Alias),
		observability.F("repo_path", repoPath),
	)

	return nil
}

// GetCurrentConfig retrieves current Git configuration
func (gm *RealGitManager) GetCurrentConfig(ctx context.Context) (*GitConfig, error) {
	gm.logger.Info(ctx, "getting_current_git_config")

	config := &GitConfig{
		Scope: "global",
	}

	// Get global name
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name")
	if output, err := cmd.Output(); err == nil {
		config.Name = strings.TrimSpace(string(output))
	}

	// Get global email
	cmd = exec.CommandContext(ctx, "git", "config", "--global", "user.email")
	if output, err := cmd.Output(); err == nil {
		config.Email = strings.TrimSpace(string(output))
	}

	// Check if we're in a repository and get local config
	if _, err := os.Stat(".git"); err == nil {
		config.Scope = "local"

		// Get local name (falls back to global if not set)
		cmd = exec.CommandContext(ctx, "git", "config", "user.name")
		if output, err := cmd.Output(); err == nil {
			localName := strings.TrimSpace(string(output))
			if localName != "" {
				config.Name = localName
			}
		}

		// Get local email (falls back to global if not set)
		cmd = exec.CommandContext(ctx, "git", "config", "user.email")
		if output, err := cmd.Output(); err == nil {
			localEmail := strings.TrimSpace(string(output))
			if localEmail != "" {
				config.Email = localEmail
			}
		}
	}

	gm.logger.Info(ctx, "current_git_config_retrieved",
		observability.F("scope", config.Scope),
		observability.F("name", config.Name),
		observability.F("email", config.Email),
	)

	return config, nil
}

// DetectRepository detects repository information from the current directory
func (gm *RealGitManager) DetectRepository(ctx context.Context, path string) (*RepositoryInfo, error) {
	gm.logger.Info(ctx, "detecting_repository",
		observability.F("path", path),
	)

	// Change to the specified directory
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(path); err != nil {
		return nil, fmt.Errorf("failed to change to directory: %w", err)
	}

	defer func() {
		os.Chdir(originalDir)
	}()

	// Check if this is a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository: %s", path)
	}

	repoInfo := &RepositoryInfo{
		Path: path,
	}

	// Get remote URL
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	if output, err := cmd.Output(); err == nil {
		remoteURL := strings.TrimSpace(string(output))
		repoInfo.RemoteURL = remoteURL

		// Parse GitHub information
		if gm.isGitHubURL(remoteURL) {
			repoInfo.IsGitHub = true
			repoInfo.SSHRemote = strings.HasPrefix(remoteURL, "git@github.com:")

			// Extract organization/user from URL
			org := gm.extractOrganizationFromURL(remoteURL)
			if org != "" {
				repoInfo.Organization = org
			}
		}
	}

	// Get repository name from directory or remote
	if repoName := gm.extractRepositoryName(repoInfo.RemoteURL, path); repoName != "" {
		// Repository name can be extracted from path or remote URL
	}

	gm.logger.Info(ctx, "repository_detected",
		observability.F("path", path),
		observability.F("is_github", repoInfo.IsGitHub),
		observability.F("organization", repoInfo.Organization),
		observability.F("ssh_remote", repoInfo.SSHRemote),
	)

	return repoInfo, nil
}

// SuggestAccount suggests the appropriate account for a repository
func (gm *RealGitManager) SuggestAccount(ctx context.Context, repoInfo *RepositoryInfo) (*Account, error) {
	gm.logger.Info(ctx, "suggesting_account",
		observability.F("organization", repoInfo.Organization),
		observability.F("is_github", repoInfo.IsGitHub),
	)

	// TODO: This would integrate with the AccountManager to suggest accounts
	// based on repository organization, URL patterns, etc.

	// For now, return a simple suggestion based on organization
	if repoInfo.Organization != "" {
		// Map known organizations to account types
		suggestions := map[string]string{
			"fanduel":    "work",
			"company":    "work",
			"enterprise": "work",
			"personal":   "personal",
		}

		if alias, exists := suggestions[strings.ToLower(repoInfo.Organization)]; exists {
			// This would normally call AccountManager.GetAccount()
			return &Account{
				Alias: alias,
			}, nil
		}
	}

	return nil, fmt.Errorf("no account suggestion available for repository")
}

// ValidateConfig validates Git configuration
func (gm *RealGitManager) ValidateConfig(ctx context.Context) (*GitValidationResult, error) {
	gm.logger.Info(ctx, "validating_git_config")

	result := &GitValidationResult{
		Valid:  true,
		Issues: []*GitIssue{},
	}

	// Get current config
	config, err := gm.GetCurrentConfig(ctx)
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, &GitIssue{
			Type:        "config_read_error",
			Description: fmt.Sprintf("Failed to read Git configuration: %v", err),
			Fix:         "Check Git installation and permissions",
		})
		return result, nil
	}

	// Validate name is set
	if config.Name == "" {
		result.Valid = false
		result.Issues = append(result.Issues, &GitIssue{
			Type:        "missing_name",
			Description: "Git user.name is not set",
			Fix:         "Set with: git config --global user.name 'Your Name'",
		})
	}

	// Validate email is set and has proper format
	if config.Email == "" {
		result.Valid = false
		result.Issues = append(result.Issues, &GitIssue{
			Type:        "missing_email",
			Description: "Git user.email is not set",
			Fix:         "Set with: git config --global user.email 'your.email@example.com'",
		})
	} else if !gm.isValidEmail(config.Email) {
		result.Valid = false
		result.Issues = append(result.Issues, &GitIssue{
			Type:        "invalid_email",
			Description: "Git user.email has invalid format",
			Fix:         "Set valid email with: git config --global user.email 'your.email@example.com'",
		})
	}

	// Check if we're in a repository with SSH remote but no SSH command configured
	if repoInfo, err := gm.DetectRepository(ctx, "."); err == nil && repoInfo.SSHRemote {
		cmd := exec.CommandContext(ctx, "git", "config", "core.sshCommand")
		if err := cmd.Run(); err != nil {
			// No SSH command configured for SSH remote
			result.Issues = append(result.Issues, &GitIssue{
				Type:        "missing_ssh_command",
				Description: "Repository uses SSH but no SSH command configured",
				Fix:         "Configure SSH with: git config core.sshCommand 'ssh -i /path/to/key'",
			})
		}
	}

	gm.logger.Info(ctx, "git_config_validated",
		observability.F("valid", result.Valid),
		observability.F("issues_count", len(result.Issues)),
	)

	return result, nil
}

// FixConfig attempts to fix Git configuration issues
func (gm *RealGitManager) FixConfig(ctx context.Context, issues []*GitIssue) error {
	gm.logger.Info(ctx, "fixing_git_config",
		observability.F("issues_count", len(issues)),
	)

	// Most Git config issues require user input (name, email),
	// so we can't automatically fix them without account information

	fixedCount := 0
	for _, issue := range issues {
		switch issue.Type {
		case "missing_ssh_command":
			// This could be fixed if we had account information
			gm.logger.Info(ctx, "ssh_command_fix_requires_account_info")
		default:
			gm.logger.Info(ctx, "git_issue_requires_manual_fix",
				observability.F("type", issue.Type),
			)
		}
	}

	gm.logger.Info(ctx, "git_config_fix_completed",
		observability.F("total", len(issues)),
		observability.F("fixed", fixedCount),
	)

	return nil
}

// Helper methods

func (gm *RealGitManager) isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com")
}

func (gm *RealGitManager) extractOrganizationFromURL(url string) string {
	// Handle SSH URLs: git@github.com:org/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	// Handle HTTPS URLs: https://github.com/org/repo.git
	if strings.HasPrefix(url, "https://github.com/") {
		path := strings.TrimPrefix(url, "https://github.com/")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 1 {
			return parts[0]
		}
	}

	return ""
}

func (gm *RealGitManager) extractRepositoryName(url, path string) string {
	// Try to get from URL first
	if url != "" {
		if strings.HasPrefix(url, "git@github.com:") {
			path := strings.TrimPrefix(url, "git@github.com:")
			path = strings.TrimSuffix(path, ".git")
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				return parts[1]
			}
		}

		if strings.HasPrefix(url, "https://github.com/") {
			path := strings.TrimPrefix(url, "https://github.com/")
			path = strings.TrimSuffix(path, ".git")
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	// Fall back to directory name
	return filepath.Base(path)
}

func (gm *RealGitManager) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
