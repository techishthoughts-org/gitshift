package internal

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// ErrorHandler provides centralized error handling with user-friendly messages
type ErrorHandler struct {
	logger observability.Logger
}

// GitPersonaError represents a structured error with context and suggestions
type GitPersonaError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	UserMessage string                 `json:"user_message"`
	Suggestions []string               `json:"suggestions"`
	Context     map[string]interface{} `json:"context"`
	Severity    string                 `json:"severity"`
	Category    string                 `json:"category"`
	Timestamp   time.Time              `json:"timestamp"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Cause       error                  `json:"-"`
}

// Error categories
const (
	ErrorCategoryAccount     = "account"
	ErrorCategorySSH         = "ssh"
	ErrorCategoryGit         = "git"
	ErrorCategoryGitHub      = "github"
	ErrorCategorySystem      = "system"
	ErrorCategoryConfig      = "config"
	ErrorCategoryValidation  = "validation"
	ErrorCategorySecurity    = "security"
	ErrorCategoryNetwork     = "network"
	ErrorCategoryPermissions = "permissions"
)

// Error severities
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// Predefined error codes
const (
	ErrAccountNotFound      = "ACCOUNT_NOT_FOUND"
	ErrAccountAlreadyExists = "ACCOUNT_ALREADY_EXISTS"
	ErrNoActiveAccount      = "NO_ACTIVE_ACCOUNT"
	ErrSSHKeyNotFound       = "SSH_KEY_NOT_FOUND"
	ErrSSHKeyInvalid        = "SSH_KEY_INVALID"
	ErrSSHPermissions       = "SSH_PERMISSIONS"
	ErrSSHConnectivity      = "SSH_CONNECTIVITY"
	ErrGitConfigInvalid     = "GIT_CONFIG_INVALID"
	ErrGitRepoNotFound      = "GIT_REPO_NOT_FOUND"
	ErrGitHubTokenInvalid   = "GITHUB_TOKEN_INVALID"
	ErrGitHubAPIAccess      = "GITHUB_API_ACCESS"
	ErrConfigLoadFailed     = "CONFIG_LOAD_FAILED"
	ErrConfigSaveFailed     = "CONFIG_SAVE_FAILED"
	ErrValidationFailed     = "VALIDATION_FAILED"
	ErrSecurityViolation    = "SECURITY_VIOLATION"
	ErrNetworkTimeout       = "NETWORK_TIMEOUT"
	ErrPermissionDenied     = "PERMISSION_DENIED"
	ErrServiceUnavailable   = "SERVICE_UNAVAILABLE"
)

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger observability.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// Error implements the error interface
func (e *GitPersonaError) Error() string {
	return e.Message
}

// Unwrap returns the underlying cause
func (e *GitPersonaError) Unwrap() error {
	return e.Cause
}

// IsCode checks if the error has a specific code
func (e *GitPersonaError) IsCode(code string) bool {
	return e.Code == code
}

// IsCategory checks if the error belongs to a specific category
func (e *GitPersonaError) IsCategory(category string) bool {
	return e.Category == category
}

// IsSeverity checks if the error has a specific severity
func (e *GitPersonaError) IsSeverity(severity string) bool {
	return e.Severity == severity
}

// NewError creates a new GitPersonaError
func (eh *ErrorHandler) NewError(code, message string) *GitPersonaError {
	return &GitPersonaError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// WrapError wraps an existing error with GitPersona context
func (eh *ErrorHandler) WrapError(err error, code, userMessage string) *GitPersonaError {
	gitPersonaErr := &GitPersonaError{
		Code:        code,
		Message:     err.Error(),
		UserMessage: userMessage,
		Cause:       err,
		Timestamp:   time.Now(),
		Context:     make(map[string]interface{}),
	}

	// Try to extract additional context from the original error
	if existingErr, ok := err.(*GitPersonaError); ok {
		gitPersonaErr.Category = existingErr.Category
		gitPersonaErr.Severity = existingErr.Severity
		for k, v := range existingErr.Context {
			gitPersonaErr.Context[k] = v
		}
	}

	return gitPersonaErr
}

// HandleError processes an error and returns a user-friendly version
func (eh *ErrorHandler) HandleError(ctx context.Context, err error) *GitPersonaError {
	if err == nil {
		return nil
	}

	// If it's already a GitPersonaError, enhance it
	if gitPersonaErr, ok := err.(*GitPersonaError); ok {
		return eh.enhanceError(ctx, gitPersonaErr)
	}

	// Convert regular error to GitPersonaError
	gitPersonaErr := eh.convertError(ctx, err)
	return eh.enhanceError(ctx, gitPersonaErr)
}

// LogError logs an error with appropriate context
func (eh *ErrorHandler) LogError(ctx context.Context, err *GitPersonaError) {
	logLevel := "error"
	if err.Severity == SeverityLow || err.Severity == SeverityInfo {
		logLevel = "warn"
	}

	fields := []observability.Field{
		observability.F("error_code", err.Code),
		observability.F("error_category", err.Category),
		observability.F("error_severity", err.Severity),
		observability.F("error_message", err.Message),
	}

	for key, value := range err.Context {
		fields = append(fields, observability.F("ctx_"+key, value))
	}

	switch logLevel {
	case "warn":
		eh.logger.Warn(ctx, "gitpersona_error", fields...)
	default:
		eh.logger.Error(ctx, "gitpersona_error", fields...)
	}
}

// CreateAccountError creates account-related errors
func (eh *ErrorHandler) CreateAccountError(code string, accountAlias string, cause error) *GitPersonaError {
	errorMap := map[string]struct {
		message     string
		userMessage string
		suggestions []string
		severity    string
	}{
		ErrAccountNotFound: {
			message:     fmt.Sprintf("Account '%s' not found", accountAlias),
			userMessage: fmt.Sprintf("The account '%s' doesn't exist in your configuration.", accountAlias),
			suggestions: []string{
				"List available accounts with: gitpersona account list",
				"Add a new account with: gitpersona account add " + accountAlias,
				"Check for typos in the account name",
			},
			severity: SeverityHigh,
		},
		ErrAccountAlreadyExists: {
			message:     fmt.Sprintf("Account '%s' already exists", accountAlias),
			userMessage: fmt.Sprintf("An account named '%s' is already configured.", accountAlias),
			suggestions: []string{
				"Choose a different account name",
				"Update the existing account with: gitpersona account update " + accountAlias,
				"Remove the existing account first if needed",
			},
			severity: SeverityMedium,
		},
		ErrNoActiveAccount: {
			message:     "No account is currently active",
			userMessage: "You need to select an account before performing this action.",
			suggestions: []string{
				"Switch to an account with: gitpersona account switch [name]",
				"List available accounts with: gitpersona account list",
				"Add your first account with: gitpersona account add [name]",
			},
			severity: SeverityHigh,
		},
	}

	info := errorMap[code]
	gitPersonaErr := &GitPersonaError{
		Code:        code,
		Message:     info.message,
		UserMessage: info.userMessage,
		Suggestions: info.suggestions,
		Severity:    info.severity,
		Category:    ErrorCategoryAccount,
		Timestamp:   time.Now(),
		Context: map[string]interface{}{
			"account_alias": accountAlias,
		},
		Cause: cause,
	}

	return gitPersonaErr
}

// CreateSSHError creates SSH-related errors
func (eh *ErrorHandler) CreateSSHError(code string, keyPath string, cause error) *GitPersonaError {
	errorMap := map[string]struct {
		message     string
		userMessage string
		suggestions []string
		severity    string
	}{
		ErrSSHKeyNotFound: {
			message:     fmt.Sprintf("SSH key not found: %s", keyPath),
			userMessage: fmt.Sprintf("The SSH key file doesn't exist at the specified location."),
			suggestions: []string{
				"Check if the file path is correct",
				"Generate a new SSH key with: gitpersona ssh keys generate",
				"Update the account with the correct key path",
			},
			severity: SeverityHigh,
		},
		ErrSSHKeyInvalid: {
			message:     fmt.Sprintf("SSH key is invalid: %s", keyPath),
			userMessage: "The SSH key file exists but appears to be corrupted or invalid.",
			suggestions: []string{
				"Generate a new SSH key to replace the invalid one",
				"Check file permissions: chmod 600 " + keyPath,
				"Verify the key format is correct",
			},
			severity: SeverityHigh,
		},
		ErrSSHPermissions: {
			message:     fmt.Sprintf("SSH key has incorrect permissions: %s", keyPath),
			userMessage: "The SSH key file permissions are not secure.",
			suggestions: []string{
				"Fix permissions with: chmod 600 " + keyPath,
				"Run security audit: gitpersona diagnose ssh",
				"Use auto-fix: gitpersona ssh fix --auto",
			},
			severity: SeverityMedium,
		},
		ErrSSHConnectivity: {
			message:     "SSH connectivity test failed",
			userMessage: "Unable to connect to GitHub using SSH.",
			suggestions: []string{
				"Check your internet connection",
				"Verify the SSH key is uploaded to GitHub",
				"Test SSH manually: ssh -T git@github.com",
				"Run diagnostics: gitpersona ssh diagnose",
			},
			severity: SeverityHigh,
		},
	}

	info := errorMap[code]
	gitPersonaErr := &GitPersonaError{
		Code:        code,
		Message:     info.message,
		UserMessage: info.userMessage,
		Suggestions: info.suggestions,
		Severity:    info.severity,
		Category:    ErrorCategorySSH,
		Timestamp:   time.Now(),
		Context: map[string]interface{}{
			"key_path": keyPath,
		},
		Cause: cause,
	}

	return gitPersonaErr
}

// CreateGitHubError creates GitHub-related errors
func (eh *ErrorHandler) CreateGitHubError(code string, account string, cause error) *GitPersonaError {
	errorMap := map[string]struct {
		message     string
		userMessage string
		suggestions []string
		severity    string
	}{
		ErrGitHubTokenInvalid: {
			message:     fmt.Sprintf("GitHub token is invalid for account '%s'", account),
			userMessage: "The GitHub token for this account is expired or invalid.",
			suggestions: []string{
				"Generate a new token at: https://github.com/settings/tokens",
				"Update the token with: gitpersona github token set",
				"Check token permissions and scopes",
			},
			severity: SeverityHigh,
		},
		ErrGitHubAPIAccess: {
			message:     fmt.Sprintf("GitHub API access failed for account '%s'", account),
			userMessage: "Unable to access GitHub API. This might be a network or authentication issue.",
			suggestions: []string{
				"Check your internet connection",
				"Verify the GitHub token is valid",
				"Check GitHub status: https://githubstatus.com",
				"Try again in a few minutes",
			},
			severity: SeverityMedium,
		},
	}

	info := errorMap[code]
	gitPersonaErr := &GitPersonaError{
		Code:        code,
		Message:     info.message,
		UserMessage: info.userMessage,
		Suggestions: info.suggestions,
		Severity:    info.severity,
		Category:    ErrorCategoryGitHub,
		Timestamp:   time.Now(),
		Context: map[string]interface{}{
			"account": account,
		},
		Cause: cause,
	}

	return gitPersonaErr
}

// FormatUserError formats an error for display to the user
func (eh *ErrorHandler) FormatUserError(err *GitPersonaError) string {
	var output strings.Builder

	// Error icon and user message
	severityIcon := eh.getSeverityIcon(err.Severity)
	output.WriteString(fmt.Sprintf("%s %s\n", severityIcon, err.UserMessage))

	// Show suggestions if available
	if len(err.Suggestions) > 0 {
		output.WriteString("\n💡 Suggestions:\n")
		for i, suggestion := range err.Suggestions {
			output.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
		}
	}

	// Show error code for debugging (in verbose mode)
	if err.Code != "" {
		output.WriteString(fmt.Sprintf("\n🔍 Error Code: %s\n", err.Code))
	}

	return output.String()
}

// Private helper methods

func (eh *ErrorHandler) convertError(ctx context.Context, err error) *GitPersonaError {
	errStr := err.Error()

	// Try to detect error type from message
	switch {
	case strings.Contains(errStr, "not found"):
		return eh.NewError("RESOURCE_NOT_FOUND", errStr)
	case strings.Contains(errStr, "permission denied"):
		return eh.NewError(ErrPermissionDenied, errStr)
	case strings.Contains(errStr, "timeout"):
		return eh.NewError(ErrNetworkTimeout, errStr)
	case strings.Contains(errStr, "connection refused"):
		return eh.NewError(ErrServiceUnavailable, errStr)
	default:
		return eh.NewError("UNKNOWN_ERROR", errStr)
	}
}

func (eh *ErrorHandler) enhanceError(ctx context.Context, err *GitPersonaError) *GitPersonaError {
	// Set defaults if not already set
	if err.Category == "" {
		err.Category = eh.inferCategory(err.Code)
	}

	if err.Severity == "" {
		err.Severity = eh.inferSeverity(err.Code)
	}

	if err.UserMessage == "" {
		err.UserMessage = eh.generateUserMessage(err)
	}

	if len(err.Suggestions) == 0 {
		err.Suggestions = eh.generateSuggestions(err)
	}

	// Add stack trace in debug mode
	if eh.shouldIncludeStackTrace(err) {
		err.StackTrace = eh.getStackTrace()
	}

	return err
}

func (eh *ErrorHandler) inferCategory(code string) string {
	switch {
	case strings.Contains(code, "ACCOUNT"):
		return ErrorCategoryAccount
	case strings.Contains(code, "SSH"):
		return ErrorCategorySSH
	case strings.Contains(code, "GIT"):
		return ErrorCategoryGit
	case strings.Contains(code, "GITHUB"):
		return ErrorCategoryGitHub
	case strings.Contains(code, "CONFIG"):
		return ErrorCategoryConfig
	case strings.Contains(code, "PERMISSION"):
		return ErrorCategoryPermissions
	case strings.Contains(code, "NETWORK"):
		return ErrorCategoryNetwork
	case strings.Contains(code, "SECURITY"):
		return ErrorCategorySecurity
	default:
		return ErrorCategorySystem
	}
}

func (eh *ErrorHandler) inferSeverity(code string) string {
	criticalErrors := []string{
		ErrSecurityViolation,
		ErrSSHKeyInvalid,
		ErrPermissionDenied,
	}

	highErrors := []string{
		ErrAccountNotFound,
		ErrSSHKeyNotFound,
		ErrSSHConnectivity,
		ErrGitHubTokenInvalid,
		ErrConfigLoadFailed,
	}

	for _, critical := range criticalErrors {
		if code == critical {
			return SeverityCritical
		}
	}

	for _, high := range highErrors {
		if code == high {
			return SeverityHigh
		}
	}

	return SeverityMedium
}

func (eh *ErrorHandler) generateUserMessage(err *GitPersonaError) string {
	// Generate a user-friendly message based on the error
	return fmt.Sprintf("An error occurred: %s", err.Message)
}

func (eh *ErrorHandler) generateSuggestions(err *GitPersonaError) []string {
	// Generate generic suggestions
	suggestions := []string{
		"Check the GitPersona documentation for help",
		"Run 'gitpersona diagnose' to identify issues",
	}

	if err.Category == ErrorCategorySSH {
		suggestions = append(suggestions, "Run 'gitpersona ssh diagnose' for SSH-specific help")
	}

	return suggestions
}

func (eh *ErrorHandler) getSeverityIcon(severity string) string {
	switch severity {
	case SeverityCritical:
		return "🚨"
	case SeverityHigh:
		return "❌"
	case SeverityMedium:
		return "⚠️"
	case SeverityLow:
		return "ℹ️"
	default:
		return "❓"
	}
}

func (eh *ErrorHandler) shouldIncludeStackTrace(err *GitPersonaError) bool {
	// Include stack trace for critical errors or when explicitly requested
	return err.Severity == SeverityCritical
}

func (eh *ErrorHandler) getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var trace strings.Builder
	for {
		frame, more := frames.Next()
		trace.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}

	return trace.String()
}
