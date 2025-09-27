// Package gh provides GitHub API and authentication functionality for gitshift.
package gh

import (
	"fmt"
	"net/url"
	"strings"
)

// Repository represents a GitHub repository with its basic information and permissions.
type Repository struct {
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	SSHURL      string `json:"ssh_url"`
	CloneURL    string `json:"clone_url"`
	Description string `json:"description,omitempty"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Permissions struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
}

// CheckRepoAccess checks if the authenticated user has access to the specified repository.
// It returns the repository information if accessible, or an error if not.
func (c *Client) CheckRepoAccess(owner, repo string) (*Repository, error) {
	// Try to get the repository information
	var repoInfo Repository

	err := c.REST.Get(fmt.Sprintf("repos/%s/%s", owner, repo), &repoInfo)
	if err != nil {
		// If we get a 404, the repository either doesn't exist or we don't have access
		if apiErr, ok := err.(interface{ StatusCode() int }); ok && apiErr.StatusCode() == 404 {
			return nil, fmt.Errorf("repository %s/%s not found or access denied", owner, repo)
		}
		return nil, fmt.Errorf("failed to access repository: %w", err)
	}

	return &repoInfo, nil
}

// CheckRepoAccessWithToken checks repository access using a specific token.
func CheckRepoAccessWithToken(owner, repo, token string) (*Repository, error) {
	client, err := WithToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	return client.CheckRepoAccess(owner, repo)
}

// CheckRepoAccessDefault checks repository access using the default authentication.
func CheckRepoAccessDefault(owner, repo string) (*Repository, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	return client.CheckRepoAccess(owner, repo)
}

// ParseRepoURL parses a GitHub repository URL or shorthand (e.g., "owner/repo") into owner and repo.
func ParseRepoURL(repoURL string) (owner, repo string, err error) {
	// Handle SSH URLs (git@github.com:owner/repo.git)
	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH URL format")
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

// GetDefaultBranch gets the default branch for a repository.
func (c *Client) GetDefaultBranch(owner, repo string) (string, error) {
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}

	err := c.REST.Get(fmt.Sprintf("repos/%s/%s", owner, repo), &repoInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository info: %w", err)
	}

	if repoInfo.DefaultBranch == "" {
		return "main", nil // Default fallback
	}

	return repoInfo.DefaultBranch, nil
}

// GetDefaultBranchDefault gets the default branch using default authentication.
func GetDefaultBranchDefault(owner, repo string) (string, error) {
	client, err := NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub client: %w", err)
	}
	return client.GetDefaultBranch(owner, repo)
}

// HasWriteAccess checks if the authenticated user has write access to the repository.
func (c *Client) HasWriteAccess(owner, repo string) (bool, error) {
	repoInfo, err := c.CheckRepoAccess(owner, repo)
	if err != nil {
		return false, err
	}

	return repoInfo.Permissions.Push, nil
}

// IsRepositoryPrivate checks if a repository is private.
func (c *Client) IsRepositoryPrivate(owner, repo string) (bool, error) {
	repoInfo, err := c.CheckRepoAccess(owner, repo)
	if err != nil {
		return false, err
	}

	return repoInfo.Private, nil
}
