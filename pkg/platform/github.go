package platform

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/techishthoughts/gitshift/pkg/gh"
)

// GitHubPlatform implements the Platform interface for GitHub
type GitHubPlatform struct {
	domain      string
	apiEndpoint string
	client      *gh.Client
}

// NewGitHubPlatform creates a new GitHub platform instance
func NewGitHubPlatform() *GitHubPlatform {
	return &GitHubPlatform{
		domain:      "github.com",
		apiEndpoint: "https://api.github.com",
	}
}

// NewGitHubEnterprisePlatform creates a new GitHub Enterprise platform instance
func NewGitHubEnterprisePlatform(domain, apiEndpoint string) *GitHubPlatform {
	return &GitHubPlatform{
		domain:      domain,
		apiEndpoint: apiEndpoint,
	}
}

// GetType returns the platform type
func (p *GitHubPlatform) GetType() Type {
	return TypeGitHub
}

// GetDomain returns the platform's domain
func (p *GitHubPlatform) GetDomain() string {
	return p.domain
}

// GetSSHHost returns the SSH host for the platform
func (p *GitHubPlatform) GetSSHHost() string {
	return p.domain
}

// GetSSHUser returns the SSH user for the platform
func (p *GitHubPlatform) GetSSHUser() string {
	return "git"
}

// FormatSSHURL formats a repository path as an SSH URL
func (p *GitHubPlatform) FormatSSHURL(owner, repo string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", p.domain, owner, repo)
}

// FormatHTTPSURL formats a repository path as an HTTPS URL
func (p *GitHubPlatform) FormatHTTPSURL(owner, repo string) string {
	return fmt.Sprintf("https://%s/%s/%s.git", p.domain, owner, repo)
}

// ParseRepositoryURL parses a repository URL and extracts owner and repo name
func (p *GitHubPlatform) ParseRepositoryURL(repoURL string) (owner, repo string, err error) {
	// Handle SSH URLs (git@github.com:owner/repo.git)
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
		repoParts := strings.Split(repoPath, "/")
		if len(repoParts) != 2 {
			return "", "", fmt.Errorf("invalid repository path in URL")
		}
		return repoParts[0], repoParts[1], nil
	}

	// Handle HTTPS URLs (https://github.com/owner/repo.git)
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

		repoName := strings.TrimSuffix(pathParts[1], ".git")
		return pathParts[0], repoName, nil
	}

	// Handle shorthand notation (owner/repo)
	parts := strings.Split(repoURL, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format, expected 'owner/repo'")
	}

	return parts[0], parts[1], nil
}

// GetSSHKnownHosts returns the SSH known_hosts entries for GitHub
func (p *GitHubPlatform) GetSSHKnownHosts() []string {
	// For GitHub.com, return the official known hosts
	if p.domain == "github.com" {
		return []string{
			"github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
			"github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=",
			"github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA1sN5N6Qvma0xPP7y1wZD/mJY4qUQcf4rCZA1BH2S0eRzU5O6a8PLxLk5Zme5n+uGZ5WVJwRzFV5rKqNwO3VLflNL8fFaRrKpjjODZo3RH4T1n3Cxj0LL/XzSl1s2L2PjYbwI1FtvRNmWfQPB5DsQXPLBmYUFY9aIk7Zz0K5TjQN2XQvKxh8a7XHlMF6a7cE0tOb8B9N/nVN8xX6F6dMx+vA8DcY0q0vViE4o2e7Xf7c=",
		}
	}

	// For GitHub Enterprise, we can't provide known hosts upfront
	return []string{}
}

// TestSSHConnection tests the SSH connection to GitHub
func (p *GitHubPlatform) TestSSHConnection(keyPath string) error {
	args := []string{"-T", fmt.Sprintf("git@%s", p.domain)}

	if keyPath != "" {
		args = append([]string{"-i", keyPath, "-o", "IdentitiesOnly=yes"}, args...)
	}

	testCmd := exec.Command("ssh", args...)
	output, err := testCmd.CombinedOutput()
	outputStr := string(output)

	// GitHub returns exit code 1 but with a success message when authentication succeeds
	// but shell access is not granted (which is the expected behavior)
	if err == nil || strings.Contains(outputStr, "successfully authenticated") {
		return nil
	}

	return fmt.Errorf("SSH connection test failed: %w\nOutput: %s", err, outputStr)
}

// GetAPIClient returns a GitHub API client
func (p *GitHubPlatform) GetAPIClient() (APIClient, error) {
	if p.client != nil {
		return &GitHubAPIClient{client: p.client}, nil
	}

	// Create a new client using default authentication
	client, err := gh.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	p.client = client
	return &GitHubAPIClient{client: client}, nil
}

// GitHubAPIClient wraps the gh.Client to implement the APIClient interface
type GitHubAPIClient struct {
	client *gh.Client
}

// IsAuthenticated checks if the client is properly authenticated
func (c *GitHubAPIClient) IsAuthenticated() (bool, error) {
	return c.client.IsAuthenticated()
}

// GetAuthenticatedUser returns the username of the authenticated user
func (c *GitHubAPIClient) GetAuthenticatedUser(ctx context.Context) (string, error) {
	return c.client.GetAuthenticatedUser(ctx)
}

// CheckRepoAccess checks if the authenticated user has access to a repository
func (c *GitHubAPIClient) CheckRepoAccess(owner, repo string) (*Repository, error) {
	ghRepo, err := c.client.CheckRepoAccess(owner, repo)
	if err != nil {
		return nil, err
	}

	return &Repository{
		FullName:    ghRepo.FullName,
		Owner:       ghRepo.Owner.Login,
		Name:        repo,
		Description: ghRepo.Description,
		Private:     ghRepo.Private,
		Fork:        ghRepo.Fork,
		SSHURL:      ghRepo.SSHURL,
		HTTPSURL:    ghRepo.CloneURL,
		Permissions: &Permissions{
			Admin: ghRepo.Permissions.Admin,
			Push:  ghRepo.Permissions.Push,
			Pull:  ghRepo.Permissions.Pull,
		},
	}, nil
}

// GetDefaultBranch gets the default branch for a repository
func (c *GitHubAPIClient) GetDefaultBranch(owner, repo string) (string, error) {
	return c.client.GetDefaultBranch(owner, repo)
}

// VerifySSHKey verifies if an SSH key is added to the user's account
func (c *GitHubAPIClient) VerifySSHKey(ctx context.Context, publicKey string) (bool, error) {
	return c.client.VerifySSHKey(ctx, publicKey)
}

// HasWriteAccess checks if the authenticated user has write access to a repository
func (c *GitHubAPIClient) HasWriteAccess(owner, repo string) (bool, error) {
	return c.client.HasWriteAccess(owner, repo)
}
