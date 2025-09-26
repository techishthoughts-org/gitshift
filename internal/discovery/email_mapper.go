package discovery

import (
	"context"
	"os/exec"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// EmailMapper handles mapping GitHub usernames to their correct email addresses
type EmailMapper struct {
	logger observability.Logger
}

// NewEmailMapper creates a new email mapper
func NewEmailMapper(logger observability.Logger) *EmailMapper {
	return &EmailMapper{
		logger: logger,
	}
}

// GetEmailForGitHubUser attempts to discover the correct email for a GitHub username
func (m *EmailMapper) GetEmailForGitHubUser(ctx context.Context, username string) string {
	if m.logger != nil {
		m.logger.Info(ctx, "discovering_email_for_github_user", observability.F("username", username))
	}

	// Strategy 1: Use known mapping patterns for specific usernames (highest priority)
	if email := m.getKnownEmailMapping(ctx, username); email != "" {
		if m.logger != nil {
			m.logger.Info(ctx, "email_found_in_known_mapping",
				observability.F("username", username),
				observability.F("email", email))
		}
		return email
	}

	// Strategy 2: Check git log for commits by this user
	if email := m.findEmailInGitHistory(ctx, username); email != "" {
		if m.logger != nil {
			m.logger.Info(ctx, "email_found_in_git_history",
				observability.F("username", username),
				observability.F("email", email))
		}
		return email
	}

	// Strategy 3: Check GitHub commits API for public email
	if email := m.findEmailFromGitHubCommits(ctx, username); email != "" {
		if m.logger != nil {
			m.logger.Info(ctx, "email_found_in_github_commits",
				observability.F("username", username),
				observability.F("email", email))
		}
		return email
	}

	if m.logger != nil {
		m.logger.Warn(ctx, "no_email_found_for_github_user", observability.F("username", username))
	}
	return ""
}

// findEmailInGitHistory searches local git repositories for commits by the username
func (m *EmailMapper) findEmailInGitHistory(ctx context.Context, username string) string {
	// Search for commits in current directory and common git locations
	gitDirs := []string{
		".",
		"../",
		"../../",
	}

	for _, dir := range gitDirs {
		if email := m.searchGitLogInDir(ctx, dir, username); email != "" {
			return email
		}
	}

	return ""
}

// searchGitLogInDir searches git log in a specific directory
func (m *EmailMapper) searchGitLogInDir(ctx context.Context, dir, username string) string {
	// Try different variations of the username in git log
	searchPatterns := []string{
		"--author=" + username,
		"--committer=" + username,
		"--author=@" + username,
	}

	// Handle nil context
	if ctx == nil {
		ctx = context.Background()
	}

	for _, pattern := range searchPatterns {
		cmd := exec.CommandContext(ctx, "git", "-C", dir, "log", pattern, "--format=%ae", "-n", "1")
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			email := strings.TrimSpace(string(output))
			if m.isValidEmail(email) {
				return email
			}
		}
	}

	return ""
}

// findEmailFromGitHubCommits attempts to find email from GitHub public commits
func (m *EmailMapper) findEmailFromGitHubCommits(ctx context.Context, username string) string {
	// Handle nil context
	if ctx == nil {
		ctx = context.Background()
	}

	// Get recent events for the user which might include commits with emails
	cmd := exec.CommandContext(ctx, "curl", "-s",
		"https://api.github.com/users/"+username+"/events/public")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Look for email patterns in the commit data
	emailRegex := regexp.MustCompile(`"email":\s*"([^"@]+@[^"]+)"`)
	matches := emailRegex.FindAllStringSubmatch(string(output), -1)

	for _, match := range matches {
		if len(match) > 1 {
			email := match[1]
			if email != "noreply@github.com" && m.isValidEmail(email) {
				return email
			}
		}
	}

	return ""
}

// getKnownEmailMapping returns known email mappings for specific usernames
func (m *EmailMapper) getKnownEmailMapping(ctx context.Context, username string) string {
	// Define known mappings based on common patterns or user-specific knowledge
	knownMappings := map[string]string{
		"costaar7": "arthur.costa@fanduel.com",
		"thukabjj": "arthur.alvesdeveloper@gmail.com",
	}

	if email, exists := knownMappings[username]; exists {
		return email
	}

	// For now, we won't auto-guess emails, but this could be enhanced
	// to validate emails or ask the user for confirmation
	// Future: Generate educated guesses based on username patterns

	return ""
}

// isValidEmail performs basic email validation
func (m *EmailMapper) isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
