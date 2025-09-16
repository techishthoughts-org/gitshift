package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealGitService implements Git operations
type RealGitService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewRealGitService creates a new Git service
func NewRealGitService(logger observability.Logger, runner execrunner.CmdRunner) *RealGitService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealGitService{
		logger: logger,
		runner: runner,
	}
}

// ValidateRepositoryExists validates if a repository exists and is accessible
func (s *RealGitService) ValidateRepositoryExists(ctx context.Context, repoURL string) error {
	s.logger.Info(ctx, "validating_repository_exists",
		observability.F("repo_url", repoURL),
	)

	// Try SSH first if it's an SSH URL
	if strings.HasPrefix(repoURL, "git@") {
		if err := s.testSSHRepositoryAccess(ctx, repoURL); err == nil {
			s.logger.Info(ctx, "ssh_repository_access_successful",
				observability.F("repo_url", repoURL),
			)
			return nil
		} else {
			s.logger.Warn(ctx, "ssh_repository_access_failed",
				observability.F("repo_url", repoURL),
				observability.F("error", err.Error()),
			)
		}
	}

	// Try HTTP as fallback
	if strings.HasPrefix(repoURL, "https://") {
		return s.testHTTPRepositoryAccess(ctx, repoURL)
	}

	// Convert SSH URL to HTTP for testing
	if strings.HasPrefix(repoURL, "git@") {
		httpURL := s.convertSSHToHTTP(repoURL)
		s.logger.Info(ctx, "trying_http_fallback",
			observability.F("ssh_url", repoURL),
			observability.F("http_url", httpURL),
		)
		return s.testHTTPRepositoryAccess(ctx, httpURL)
	}

	return fmt.Errorf("unable to validate repository access for URL: %s", repoURL)
}

// testSSHRepositoryAccess tests SSH access to a repository
func (s *RealGitService) testSSHRepositoryAccess(ctx context.Context, repoURL string) error {
	s.logger.Info(ctx, "testing_ssh_repository_access",
		observability.F("repo_url", repoURL),
	)

	// Extract host from SSH URL (e.g., git@github.com:owner/repo.git -> github.com)
	parts := strings.Split(repoURL, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid SSH URL format: %s", repoURL)
	}

	hostPart := strings.Split(parts[1], ":")[0]
	host := hostPart

	// Test SSH connection to the host
	cmd := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=10", fmt.Sprintf("git@%s", host))
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	if err != nil {
		s.logger.Warn(ctx, "ssh_host_connection_failed",
			observability.F("host", host),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("SSH connection to %s failed: %w", host, err)
	}

	// Check if the output indicates successful authentication
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Hi ") {
		s.logger.Info(ctx, "ssh_host_connection_successful",
			observability.F("host", host),
			observability.F("output", outputStr),
		)
		return nil
	}

	s.logger.Warn(ctx, "ssh_host_connection_unexpected_output",
		observability.F("host", host),
		observability.F("output", outputStr),
	)
	return fmt.Errorf("unexpected SSH connection output from %s: %s", host, outputStr)
}

// testHTTPRepositoryAccess tests HTTP access to a repository
func (s *RealGitService) testHTTPRepositoryAccess(ctx context.Context, repoURL string) error {
	s.logger.Info(ctx, "testing_http_repository_access",
		observability.F("repo_url", repoURL),
	)

	// Use git ls-remote to test repository access
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	if err != nil {
		s.logger.Warn(ctx, "http_repository_access_failed",
			observability.F("repo_url", repoURL),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("HTTP repository access failed: %w", err)
	}

	// Check if we got valid output (should contain commit hashes and refs)
	outputStr := strings.TrimSpace(string(output))
	if len(outputStr) > 0 && strings.Contains(outputStr, "refs/") {
		s.logger.Info(ctx, "http_repository_access_successful",
			observability.F("repo_url", repoURL),
		)
		return nil
	}

	s.logger.Warn(ctx, "http_repository_access_unexpected_output",
		observability.F("repo_url", repoURL),
		observability.F("output", outputStr),
	)
	return fmt.Errorf("unexpected HTTP repository access output: %s", outputStr)
}

// convertSSHToHTTP converts SSH URL to HTTP URL
func (s *RealGitService) convertSSHToHTTP(sshURL string) string {
	// Convert git@github.com:owner/repo.git to https://github.com/owner/repo.git
	if strings.HasPrefix(sshURL, "git@") {
		// Remove git@ prefix
		url := strings.TrimPrefix(sshURL, "git@")
		// Replace first : with /
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			return "https://" + parts[0] + "/" + parts[1]
		}
	}
	return sshURL
}

// GetRemoteURL gets the remote URL for the current repository
func (s *RealGitService) GetRemoteURL(ctx context.Context, remote string) (string, error) {
	s.logger.Info(ctx, "getting_remote_url",
		observability.F("remote", remote),
	)

	cmd := exec.Command("git", "remote", "get-url", remote)
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	if err != nil {
		s.logger.Warn(ctx, "failed_to_get_remote_url",
			observability.F("remote", remote),
			observability.F("error", err.Error()),
		)
		return "", fmt.Errorf("failed to get remote URL for %s: %w", remote, err)
	}

	url := strings.TrimSpace(string(output))
	s.logger.Info(ctx, "remote_url_retrieved",
		observability.F("remote", remote),
		observability.F("url", url),
	)
	return url, nil
}

// SetRemoteURL sets the remote URL for the current repository
func (s *RealGitService) SetRemoteURL(ctx context.Context, remote, url string) error {
	s.logger.Info(ctx, "setting_remote_url",
		observability.F("remote", remote),
		observability.F("url", url),
	)

	cmd := exec.Command("git", "remote", "set-url", remote, url)
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	if err != nil {
		s.logger.Warn(ctx, "failed_to_set_remote_url",
			observability.F("remote", remote),
			observability.F("url", url),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("failed to set remote URL for %s: %w", remote, err)
	}

	s.logger.Info(ctx, "remote_url_set_successfully",
		observability.F("remote", remote),
		observability.F("url", url),
	)
	return nil
}
