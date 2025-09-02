package errors

import (
	"fmt"
	"strings"
)

// ErrorCategory represents different types of errors
type ErrorCategory string

const (
	CategoryUser    ErrorCategory = "user"
	CategorySystem  ErrorCategory = "system"
	CategoryNetwork ErrorCategory = "network"
	CategorySSH     ErrorCategory = "ssh"
	CategoryConfig  ErrorCategory = "config"
	CategoryGitHub  ErrorCategory = "github"
)

// UserError represents a user-friendly error with contextual help
type UserError struct {
	Title       string        `json:"title"`
	Message     string        `json:"message"`
	Suggestions []string      `json:"suggestions"`
	Category    ErrorCategory `json:"category"`
	Debug       string        `json:"debug,omitempty"`
	Code        string        `json:"code,omitempty"`
}

// Error implements the error interface
func (e *UserError) Error() string {
	var sb strings.Builder

	// Title with emoji based on category
	emoji := e.getCategoryEmoji()
	sb.WriteString(fmt.Sprintf("%s %s\n", emoji, e.Title))

	// Main message
	if e.Message != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", e.Message))
	}

	// Suggestions
	if len(e.Suggestions) > 0 {
		sb.WriteString("\n💡 Try these solutions:\n")
		for i, suggestion := range e.Suggestions {
			sb.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
		}
	}

	// Debug information (only if available)
	if e.Debug != "" {
		sb.WriteString(fmt.Sprintf("\n🔍 Debug details: %s\n", e.Debug))
	}

	return sb.String()
}

// getCategoryEmoji returns appropriate emoji for error category
func (e *UserError) getCategoryEmoji() string {
	switch e.Category {
	case CategorySSH:
		return "🔑"
	case CategoryNetwork:
		return "🌐"
	case CategoryConfig:
		return "⚙️"
	case CategoryGitHub:
		return "🐙"
	case CategorySystem:
		return "🖥️"
	default:
		return "❌"
	}
}

// Error constructors for common scenarios

// NewSSHError creates an SSH-related error
func NewSSHError(title, message string, debug string) *UserError {
	return &UserError{
		Title:    title,
		Message:  message,
		Category: CategorySSH,
		Debug:    debug,
		Suggestions: []string{
			"Run 'gitpersona ssh test' to diagnose SSH issues",
			"Check if SSH agent is running: ssh-add -l",
			"Verify key permissions: chmod 600 ~/.ssh/id_*",
			"Test GitHub connection: ssh -T git@github.com",
		},
	}
}

// NewConfigError creates a configuration-related error
func NewConfigError(title, message string, debug string) *UserError {
	return &UserError{
		Title:    title,
		Message:  message,
		Category: CategoryConfig,
		Debug:    debug,
		Suggestions: []string{
			"Check configuration file: ~/.config/gitpersona/config.yaml",
			"Validate accounts: gitpersona list",
			"Reset configuration: gitpersona discover --auto-import",
			"Run health check: gitpersona health --detailed",
		},
	}
}

// NewGitHubError creates a GitHub API-related error
func NewGitHubError(title, message string, debug string) *UserError {
	return &UserError{
		Title:    title,
		Message:  message,
		Category: CategoryGitHub,
		Debug:    debug,
		Suggestions: []string{
			"Check GitHub authentication: gh auth status",
			"Re-authenticate if needed: gh auth login",
			"Verify API access: gitpersona repos",
			"Check GitHub status: https://www.githubstatus.com/",
		},
	}
}

// NewNetworkError creates a network-related error
func NewNetworkError(title, message string, debug string) *UserError {
	return &UserError{
		Title:    title,
		Message:  message,
		Category: CategoryNetwork,
		Debug:    debug,
		Suggestions: []string{
			"Check internet connectivity",
			"Verify DNS resolution: nslookup github.com",
			"Test with different network if available",
			"Check firewall and proxy settings",
		},
	}
}

// NewAccountError creates an account-related error
func NewAccountError(title, message string, accountAlias string) *UserError {
	suggestions := []string{
		"List all accounts: gitpersona list",
		"Add new account: gitpersona add-github USERNAME",
		"Switch to valid account: gitpersona switch ACCOUNT",
	}

	if accountAlias != "" {
		suggestions = append(suggestions,
			fmt.Sprintf("Update account: gitpersona add %s --overwrite", accountAlias))
	}

	return &UserError{
		Title:       title,
		Message:     message,
		Category:    CategoryUser,
		Suggestions: suggestions,
	}
}

// NewValidationError creates a validation-related error
func NewValidationError(field, value, requirement string) *UserError {
	return &UserError{
		Title:    fmt.Sprintf("Invalid %s", field),
		Message:  fmt.Sprintf("The value '%s' is not valid. %s", value, requirement),
		Category: CategoryUser,
		Suggestions: []string{
			fmt.Sprintf("Provide a valid %s value", field),
			"Check the command help: gitpersona COMMAND --help",
			"Use interactive mode for guided input",
		},
	}
}

// NewSystemError creates a system-related error
func NewSystemError(title, message string, debug string) *UserError {
	return &UserError{
		Title:    title,
		Message:  message,
		Category: CategorySystem,
		Debug:    debug,
		Suggestions: []string{
			"Check system permissions and access",
			"Verify required tools are installed",
			"Run system diagnostics: gitpersona health",
			"Check logs for additional details",
		},
	}
}

// WrapError wraps a standard error with user-friendly context
func WrapError(err error, title string, category ErrorCategory) *UserError {
	if userErr, ok := err.(*UserError); ok {
		// Already a UserError, just update category if needed
		if userErr.Category == "" {
			userErr.Category = category
		}
		return userErr
	}

	// Convert standard error to UserError
	return &UserError{
		Title:    title,
		Message:  "An unexpected error occurred while processing your request.",
		Category: category,
		Debug:    err.Error(),
		Suggestions: []string{
			"Try the operation again",
			"Check the debug details below",
			"Run diagnostics: gitpersona health",
			"Report issue if problem persists",
		},
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	debugMode bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(debugMode bool) *ErrorHandler {
	return &ErrorHandler{
		debugMode: debugMode,
	}
}

// HandleError processes and displays errors appropriately
func (h *ErrorHandler) HandleError(err error) {
	if userErr, ok := err.(*UserError); ok {
		// User-friendly error
		fmt.Print(userErr.Error())

		// Show debug info only in debug mode
		if h.debugMode && userErr.Debug != "" {
			fmt.Printf("\n🔍 Debug: %s\n", userErr.Debug)
		}
	} else {
		// Standard error - wrap it
		wrappedErr := WrapError(err, "Unexpected Error", CategorySystem)
		fmt.Print(wrappedErr.Error())
	}
}

// Specialized error creators for common GitPersona scenarios

// NewAccountNotFoundError creates error for missing account
func NewAccountNotFoundError(alias string, availableAliases []string) *UserError {
	message := fmt.Sprintf("Account '%s' not found in your configuration.", alias)

	suggestions := []string{
		"List available accounts: gitpersona list",
		"Add new account: gitpersona add-github USERNAME",
	}

	if len(availableAliases) > 0 {
		suggestions = append(suggestions,
			fmt.Sprintf("Available accounts: %s", strings.Join(availableAliases, ", ")))
	}

	return &UserError{
		Title:       "Account Not Found",
		Message:     message,
		Category:    CategoryUser,
		Suggestions: suggestions,
	}
}

// NewNoAccountsError creates error for when no accounts are configured
func NewNoAccountsError() *UserError {
	return &UserError{
		Title:    "No Accounts Configured",
		Message:  "You haven't configured any GitHub accounts yet.",
		Category: CategoryUser,
		Suggestions: []string{
			"Add your first account: gitpersona add-github YOUR_USERNAME",
			"Auto-discover existing accounts: gitpersona discover",
			"Manual setup: gitpersona add ALIAS --name 'Name' --email 'email@example.com'",
			"Launch interactive setup: gitpersona",
		},
	}
}

// NewSSHKeyNotFoundError creates error for missing SSH keys
func NewSSHKeyNotFoundError(keyPath, account string) *UserError {
	return &UserError{
		Title:    "SSH Key Not Found",
		Message:  fmt.Sprintf("SSH key file not found: %s", keyPath),
		Category: CategorySSH,
		Suggestions: []string{
			fmt.Sprintf("Generate new SSH key: gitpersona add-github %s --overwrite", account),
			"Check key path configuration",
			"Verify file permissions and location",
			"Run SSH diagnostics: gitpersona ssh doctor",
		},
	}
}

// NewGitHubAuthError creates error for GitHub authentication issues
func NewGitHubAuthError(operation string) *UserError {
	return &UserError{
		Title:    "GitHub Authentication Required",
		Message:  fmt.Sprintf("Cannot perform '%s' - GitHub authentication required.", operation),
		Category: CategoryGitHub,
		Suggestions: []string{
			"Authenticate with GitHub: gh auth login",
			"Grant required permissions when prompted",
			"Verify authentication: gh auth status",
			"Use manual setup if OAuth unavailable",
		},
	}
}

// NewProjectConfigError creates error for project configuration issues
func NewProjectConfigError(projectPath, issue string) *UserError {
	return &UserError{
		Title:    "Project Configuration Issue",
		Message:  fmt.Sprintf("Problem with project configuration in %s: %s", projectPath, issue),
		Category: CategoryConfig,
		Suggestions: []string{
			"Set project account: gitpersona project set ACCOUNT",
			"Check project config: gitpersona project show",
			"Remove invalid config: gitpersona project remove",
			"List available accounts: gitpersona list",
		},
	}
}
