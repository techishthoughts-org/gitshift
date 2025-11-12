// Package platform provides an abstraction layer for Git hosting platforms
// (GitHub, GitLab, Bitbucket, etc.)
package platform

import (
	"context"
)

// Type represents a Git hosting platform type
type Type string

const (
	// TypeGitHub represents GitHub platform
	TypeGitHub Type = "github"

	// TypeGitLab represents GitLab platform
	TypeGitLab Type = "gitlab"

	// TypeBitbucket represents Bitbucket platform
	TypeBitbucket Type = "bitbucket"

	// TypeCustom represents a custom Git hosting platform
	TypeCustom Type = "custom"
)

// String returns the string representation of a platform type
func (t Type) String() string {
	return string(t)
}

// IsValid checks if the platform type is valid
func (t Type) IsValid() bool {
	switch t {
	case TypeGitHub, TypeGitLab, TypeBitbucket, TypeCustom:
		return true
	default:
		return false
	}
}

// Platform defines the interface that all Git hosting platforms must implement
type Platform interface {
	// GetType returns the platform type
	GetType() Type

	// GetDomain returns the platform's domain (e.g., "github.com", "gitlab.com")
	GetDomain() string

	// GetSSHHost returns the SSH host for the platform (e.g., "github.com", "gitlab.com")
	GetSSHHost() string

	// GetSSHUser returns the SSH user for the platform (usually "git")
	GetSSHUser() string

	// FormatSSHURL formats a repository path as an SSH URL
	// Example: owner/repo -> git@github.com:owner/repo.git
	FormatSSHURL(owner, repo string) string

	// FormatHTTPSURL formats a repository path as an HTTPS URL
	// Example: owner/repo -> https://github.com/owner/repo.git
	FormatHTTPSURL(owner, repo string) string

	// ParseRepositoryURL parses a repository URL and extracts owner and repo name
	// Supports SSH (git@...), HTTPS (https://...), and shorthand (owner/repo) formats
	ParseRepositoryURL(url string) (owner, repo string, err error)

	// GetSSHKnownHosts returns the SSH known_hosts entries for this platform
	GetSSHKnownHosts() []string

	// TestSSHConnection tests the SSH connection to the platform
	TestSSHConnection(keyPath string) error

	// GetAPIClient returns an API client for this platform
	GetAPIClient() (APIClient, error)
}

// APIClient defines the interface for platform API operations
type APIClient interface {
	// IsAuthenticated checks if the client is properly authenticated
	IsAuthenticated() (bool, error)

	// GetAuthenticatedUser returns the username of the authenticated user
	GetAuthenticatedUser(ctx context.Context) (string, error)

	// CheckRepoAccess checks if the authenticated user has access to a repository
	CheckRepoAccess(owner, repo string) (*Repository, error)

	// GetDefaultBranch gets the default branch for a repository
	GetDefaultBranch(owner, repo string) (string, error)

	// VerifySSHKey verifies if an SSH key is added to the user's account
	VerifySSHKey(ctx context.Context, publicKey string) (bool, error)

	// HasWriteAccess checks if the authenticated user has write access to a repository
	HasWriteAccess(owner, repo string) (bool, error)
}

// Repository represents a repository on a Git hosting platform
type Repository struct {
	// FullName is the full repository name (owner/repo)
	FullName string

	// Owner is the repository owner/organization
	Owner string

	// Name is the repository name
	Name string

	// Description is the repository description
	Description string

	// Private indicates if the repository is private
	Private bool

	// Fork indicates if the repository is a fork
	Fork bool

	// SSHURL is the SSH clone URL
	SSHURL string

	// HTTPSURL is the HTTPS clone URL
	HTTPSURL string

	// DefaultBranch is the default branch name
	DefaultBranch string

	// Permissions contains the user's permissions for this repository
	Permissions *Permissions
}

// Permissions represents user permissions for a repository
type Permissions struct {
	// Admin indicates if the user has admin access
	Admin bool

	// Push indicates if the user has push access
	Push bool

	// Pull indicates if the user has pull access
	Pull bool
}

// Config represents platform-specific configuration
type Config struct {
	// Type is the platform type
	Type Type

	// Domain is the platform domain (e.g., "github.com")
	Domain string

	// APIEndpoint is the API endpoint URL (optional, for custom installations)
	APIEndpoint string

	// SSHHost is the SSH host (optional, defaults to Domain)
	SSHHost string

	// SSHPort is the SSH port (optional, defaults to 22)
	SSHPort int

	// Token is the authentication token (optional)
	Token string
}

// Registry maintains a registry of available platforms
type Registry struct {
	platforms map[Type]Platform
}

// NewRegistry creates a new platform registry
func NewRegistry() *Registry {
	return &Registry{
		platforms: make(map[Type]Platform),
	}
}

// Register registers a platform implementation
func (r *Registry) Register(platform Platform) {
	r.platforms[platform.GetType()] = platform
}

// Get retrieves a platform by type
func (r *Registry) Get(platformType Type) (Platform, bool) {
	platform, exists := r.platforms[platformType]
	return platform, exists
}

// List returns all registered platforms
func (r *Registry) List() []Platform {
	platforms := make([]Platform, 0, len(r.platforms))
	for _, platform := range r.platforms {
		platforms = append(platforms, platform)
	}
	return platforms
}

// DetectPlatform attempts to detect the platform from a repository URL
func DetectPlatform(url string) Type {
	// Quick domain-based detection
	switch {
	case containsAny(url, []string{"github.com", "github"}):
		return TypeGitHub
	case containsAny(url, []string{"gitlab.com", "gitlab"}):
		return TypeGitLab
	case containsAny(url, []string{"bitbucket.org", "bitbucket"}):
		return TypeBitbucket
	default:
		return TypeCustom
	}
}

// containsAny checks if the string contains any of the given substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
