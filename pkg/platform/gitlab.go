package platform

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// GitLabPlatform implements the Platform interface for GitLab
type GitLabPlatform struct {
	domain      string
	apiEndpoint string
	token       string
}

// NewGitLabPlatform creates a new GitLab platform instance
func NewGitLabPlatform() *GitLabPlatform {
	return &GitLabPlatform{
		domain:      "gitlab.com",
		apiEndpoint: "https://gitlab.com/api/v4",
	}
}

// NewGitLabSelfHostedPlatform creates a new self-hosted GitLab platform instance
func NewGitLabSelfHostedPlatform(domain, apiEndpoint string) *GitLabPlatform {
	if apiEndpoint == "" {
		apiEndpoint = fmt.Sprintf("https://%s/api/v4", domain)
	}
	return &GitLabPlatform{
		domain:      domain,
		apiEndpoint: apiEndpoint,
	}
}

// GetType returns the platform type
func (p *GitLabPlatform) GetType() Type {
	return TypeGitLab
}

// GetDomain returns the platform's domain
func (p *GitLabPlatform) GetDomain() string {
	return p.domain
}

// GetSSHHost returns the SSH host for the platform
func (p *GitLabPlatform) GetSSHHost() string {
	return p.domain
}

// GetSSHUser returns the SSH user for the platform
func (p *GitLabPlatform) GetSSHUser() string {
	return "git"
}

// FormatSSHURL formats a repository path as an SSH URL
func (p *GitLabPlatform) FormatSSHURL(owner, repo string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", p.domain, owner, repo)
}

// FormatHTTPSURL formats a repository path as an HTTPS URL
func (p *GitLabPlatform) FormatHTTPSURL(owner, repo string) string {
	return fmt.Sprintf("https://%s/%s/%s.git", p.domain, owner, repo)
}

// ParseRepositoryURL parses a repository URL and extracts owner and repo name
func (p *GitLabPlatform) ParseRepositoryURL(repoURL string) (owner, repo string, err error) {
	// Handle SSH URLs (git@gitlab.com:owner/repo.git)
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
		}

		// Verify domain matches
		hostPart := strings.TrimPrefix(parts[0], "git@")
		if hostPart != p.domain {
			return "", "", fmt.Errorf("domain mismatch: expected %s, got %s", p.domain, hostPart)
		}

		repoPath := strings.TrimSuffix(parts[1], ".git")

		// GitLab supports nested groups (e.g., group/subgroup/repo)
		// We need to handle this differently than GitHub
		repoParts := strings.Split(repoPath, "/")
		if len(repoParts) < 2 {
			return "", "", fmt.Errorf("invalid repository path in URL")
		}

		// For now, treat everything except the last part as owner
		// In GitLab, this could be group/subgroup
		repoName := repoParts[len(repoParts)-1]
		ownerPath := strings.Join(repoParts[:len(repoParts)-1], "/")

		return ownerPath, repoName, nil
	}

	// Handle HTTPS URLs (https://gitlab.com/owner/repo.git)
	if strings.HasPrefix(repoURL, "http") {
		parsedURL, err := url.Parse(repoURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		// Verify domain matches
		if parsedURL.Host != p.domain {
			return "", "", fmt.Errorf("domain mismatch: expected %s, got %s", p.domain, parsedURL.Host)
		}

		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid repository path in URL")
		}

		// Handle nested groups
		repoName := strings.TrimSuffix(pathParts[len(pathParts)-1], ".git")
		ownerPath := strings.Join(pathParts[:len(pathParts)-1], "/")

		return ownerPath, repoName, nil
	}

	// Handle shorthand notation (owner/repo or group/subgroup/repo)
	parts := strings.Split(repoURL, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository format, expected 'owner/repo'")
	}

	// Handle nested groups
	repoName := parts[len(parts)-1]
	ownerPath := strings.Join(parts[:len(parts)-1], "/")

	return ownerPath, repoName, nil
}

// GetSSHKnownHosts returns the SSH known_hosts entries for GitLab
func (p *GitLabPlatform) GetSSHKnownHosts() []string {
	// For gitlab.com, return the official known hosts
	if p.domain == "gitlab.com" {
		return []string{
			"gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf",
			"gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4Yt0aGROY5qX2UKSeOvuP4D6TPqKF1onrSzH9bx9XUf2lEdWT/ia1NEKjunUqu1xOB/StKDHMoX4/OKyIzuS0q/T1zOATthvasJFoPrAjkohTyaDUz2LN5JoH839hViyEG82yB+MjcFV5MU3N1l1QL3cVUCh93xSaua1N85qivl+siMkPGbO5xR/En4iEY6K2XPASUEMaieWVNTRCtJ4S8H+9",
			"gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=",
		}
	}

	// For self-hosted GitLab, we can't provide known hosts upfront
	return []string{}
}

// TestSSHConnection tests the SSH connection to GitLab
func (p *GitLabPlatform) TestSSHConnection(keyPath string) error {
	args := []string{"-T", fmt.Sprintf("git@%s", p.domain)}

	if keyPath != "" {
		args = append([]string{"-i", keyPath, "-o", "IdentitiesOnly=yes"}, args...)
	}

	testCmd := exec.Command("ssh", args...)
	output, err := testCmd.CombinedOutput()
	outputStr := string(output)

	// GitLab returns exit code 1 but with a welcome message when authentication succeeds
	// Example: "Welcome to GitLab, @username!"
	if err == nil || strings.Contains(outputStr, "Welcome to GitLab") {
		return nil
	}

	return fmt.Errorf("SSH connection test failed: %w\nOutput: %s", err, outputStr)
}

// GetAPIClient returns a GitLab API client
func (p *GitLabPlatform) GetAPIClient() (APIClient, error) {
	// For now, return a placeholder implementation
	// TODO: Implement a full GitLab API client similar to gh.Client
	return &GitLabAPIClient{
		platform: p,
	}, nil
}

// GitLabAPIClient implements the APIClient interface for GitLab
type GitLabAPIClient struct {
	platform *GitLabPlatform
}

// IsAuthenticated checks if the client is properly authenticated
func (c *GitLabAPIClient) IsAuthenticated() (bool, error) {
	// TODO: Implement GitLab authentication check
	// For now, return false to indicate not implemented
	return false, fmt.Errorf("GitLab API authentication not yet implemented")
}

// GetAuthenticatedUser returns the username of the authenticated user
func (c *GitLabAPIClient) GetAuthenticatedUser(ctx context.Context) (string, error) {
	// TODO: Implement GitLab user API call
	return "", fmt.Errorf("GitLab API not yet implemented")
}

// CheckRepoAccess checks if the authenticated user has access to a repository
func (c *GitLabAPIClient) CheckRepoAccess(owner, repo string) (*Repository, error) {
	// TODO: Implement GitLab repository access check
	return nil, fmt.Errorf("GitLab API not yet implemented")
}

// GetDefaultBranch gets the default branch for a repository
func (c *GitLabAPIClient) GetDefaultBranch(owner, repo string) (string, error) {
	// TODO: Implement GitLab default branch API call
	// For now, return common defaults
	return "main", nil
}

// VerifySSHKey verifies if an SSH key is added to the user's account
func (c *GitLabAPIClient) VerifySSHKey(ctx context.Context, publicKey string) (bool, error) {
	// TODO: Implement GitLab SSH key verification
	return false, fmt.Errorf("GitLab API not yet implemented")
}

// HasWriteAccess checks if the authenticated user has write access to a repository
func (c *GitLabAPIClient) HasWriteAccess(owner, repo string) (bool, error) {
	// TODO: Implement GitLab write access check
	return false, fmt.Errorf("GitLab API not yet implemented")
}
