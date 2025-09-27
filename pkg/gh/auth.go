// Package gh provides GitHub API and authentication functionality for GitPersona.
package gh

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// AuthStatus represents the authentication status of a GitHub user.
type AuthStatus struct {
	Authenticated bool   `json:"-"`
	Username      string `json:"login"`
	TokenSource   string `json:"-"` // e.g., "GITHUB_TOKEN", "gh", "ssh"
}

// CheckAuth checks if the user is authenticated with GitHub and returns their status.
// It tries multiple authentication methods in order of security:
// 1. Environment token (GITHUB_TOKEN)
// 2. GitHub CLI authentication
// 3. SSH key authentication (as fallback)
func CheckAuth(ctx context.Context) (*AuthStatus, error) {
	// First try with environment token if set
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client, err := WithToken(token, WithLogger(slog.Default()))
		if err == nil {
			if username, err := client.GetAuthenticatedUser(ctx); err == nil && username != "" {
				return &AuthStatus{
					Authenticated: true,
					Username:      username,
					TokenSource:   "GITHUB_TOKEN",
				}, nil
			}
		}
	}

	// Try with default client (uses gh CLI authentication)
	client, err := NewClient(WithLogger(slog.Default()))
	if err == nil {
		if username, err := client.GetAuthenticatedUser(ctx); err == nil && username != "" {
			return &AuthStatus{
				Authenticated: true,
				Username:      username,
				TokenSource:   "gh",
			}, nil
		}
	}

	// Fall back to checking gh CLI directly
	return checkGhCliAuth(ctx)
}

// checkGhCliAuth checks authentication status using the gh CLI directly
func checkGhCliAuth(ctx context.Context) (*AuthStatus, error) {
	// This is a fallback method that runs 'gh auth status' and parses the output
	// It's used when the default REST client fails to authenticate
	status := &AuthStatus{Authenticated: false}

	// Run 'gh auth status --show-token' to get detailed auth info
	result, err := execGhCommand(ctx, "auth", "status", "--show-token")
	if err != nil {
		return status, fmt.Errorf("not authenticated with GitHub CLI: %w", err)
	}

	// Parse the output to get the username
	// Example output:
	// github.com
	//   ✓ Logged in to github.com as username (GITHUB_TOKEN)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Logged in to") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "as" && i+1 < len(parts) {
					status.Username = parts[i+1]
					status.Authenticated = true
					status.TokenSource = "gh"
					break
				}
			}
		}
	}

	if !status.Authenticated {
		return status, fmt.Errorf("could not determine authentication status from gh CLI output")
	}

	return status, nil
}

// execGhCommand executes a gh CLI command with context and returns its output
func execGhCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh command failed: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
