package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents a specific error type in the system
type ErrorCode string

const (
	// Configuration errors
	ErrCodeConfigLoadFailed ErrorCode = "config_load_failed"
	ErrCodeConfigSaveFailed ErrorCode = "config_save_failed"
	ErrCodeConfigInvalid    ErrorCode = "config_invalid"
	ErrCodeConfigNotFound   ErrorCode = "config_not_found"

	// Account management errors
	ErrCodeAccountNotFound     ErrorCode = "account_not_found"
	ErrCodeAccountInvalid      ErrorCode = "account_invalid"
	ErrCodeAccountExists       ErrorCode = "account_exists"
	ErrCodeAccountSwitchFailed ErrorCode = "account_switch_failed"

	// SSH errors
	ErrCodeSSHValidationFailed ErrorCode = "ssh_validation_failed"
	ErrCodeSSHKeyNotFound      ErrorCode = "ssh_key_not_found"
	ErrCodeSSHKeyInvalid       ErrorCode = "ssh_key_invalid"
	ErrCodeSSHConnectionFailed ErrorCode = "ssh_connection_failed"
	ErrCodeSSHPermissionDenied ErrorCode = "ssh_permission_denied"

	// GitHub API errors
	ErrCodeGitHubAuthFailed  ErrorCode = "github_auth_failed"
	ErrCodeGitHubAPIFailed   ErrorCode = "github_api_failed"
	ErrCodeGitHubRateLimited ErrorCode = "github_rate_limited"
	ErrCodeGitHubNotFound    ErrorCode = "github_not_found"

	// Git errors
	ErrCodeGitConfigFailed    ErrorCode = "git_config_failed"
	ErrCodeGitRepositoryError ErrorCode = "git_repository_error"
	ErrCodeGitCommandFailed   ErrorCode = "git_command_failed"

	// Validation errors
	ErrCodeValidationFailed ErrorCode = "validation_failed"
	ErrCodeInvalidInput     ErrorCode = "invalid_input"
	ErrCodeMissingRequired  ErrorCode = "missing_required"

	// System errors
	ErrCodePermissionDenied ErrorCode = "permission_denied"
	ErrCodeFileNotFound     ErrorCode = "file_not_found"
	ErrCodeNetworkError     ErrorCode = "network_error"
	ErrCodeTimeout          ErrorCode = "timeout"
	ErrCodeInternal         ErrorCode = "internal_error"
)

// GitPersonaError represents a structured error in the system
type GitPersonaError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Cause     error                  `json:"cause,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Location  string                 `json:"location,omitempty"`
	Stack     []string               `json:"stack,omitempty"`
}

// New creates a new GitPersonaError
func New(code ErrorCode, message string) *GitPersonaError {
	return &GitPersonaError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Location:  getCallerLocation(),
		Stack:     getStackTrace(),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *GitPersonaError {
	if err == nil {
		return New(code, message)
	}

	// If it's already a GitPersonaError, enhance it
	if gpe, ok := err.(*GitPersonaError); ok {
		gpe.Message = message
		if gpe.Code == "" {
			gpe.Code = code
		}
		return gpe
	}

	return &GitPersonaError{
		Code:      code,
		Message:   message,
		Cause:     err,
		Timestamp: time.Now(),
		Location:  getCallerLocation(),
		Stack:     getStackTrace(),
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *GitPersonaError) WithContext(key string, value interface{}) *GitPersonaError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values to the error
func (e *GitPersonaError) WithContextMap(context map[string]interface{}) *GitPersonaError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// Error implements the error interface
func (e *GitPersonaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *GitPersonaError) Unwrap() error {
	return e.Cause
}

// Is checks if the error has a specific error code
func (e *GitPersonaError) Is(code ErrorCode) bool {
	return e.Code == code
}

// getCallerLocation returns the file and line number where the error was created
func getCallerLocation() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// getStackTrace returns a simplified stack trace
func getStackTrace() []string {
	var stack []string
	for i := 2; i < 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stack = append(stack, fmt.Sprintf("%s:%d", file, line))
	}
	return stack
}

// Helper functions for common error patterns
func ConfigLoadFailed(err error, context map[string]interface{}) *GitPersonaError {
	return Wrap(err, ErrCodeConfigLoadFailed, "failed to load configuration").WithContextMap(context)
}

func AccountNotFound(alias string, context map[string]interface{}) *GitPersonaError {
	return New(ErrCodeAccountNotFound, fmt.Sprintf("account '%s' not found", alias)).WithContextMap(context)
}

func SSHValidationFailed(err error, context map[string]interface{}) *GitPersonaError {
	return Wrap(err, ErrCodeSSHValidationFailed, "SSH validation failed").WithContextMap(context)
}

func ValidationFailed(err error, context map[string]interface{}) *GitPersonaError {
	return Wrap(err, ErrCodeValidationFailed, "validation failed").WithContextMap(context)
}

func InvalidInput(field string, value interface{}, context map[string]interface{}) *GitPersonaError {
	ctx := map[string]interface{}{
		"field": field,
		"value": value,
	}
	for k, v := range context {
		ctx[k] = v
	}
	return New(ErrCodeInvalidInput, fmt.Sprintf("invalid input for field '%s'", field)).WithContextMap(ctx)
}

func MissingRequired(field string, context map[string]interface{}) *GitPersonaError {
	ctx := map[string]interface{}{
		"field": field,
	}
	for k, v := range context {
		ctx[k] = v
	}
	return New(ErrCodeMissingRequired, fmt.Sprintf("required field '%s' is missing", field)).WithContextMap(ctx)
}

// IsGitPersonaError checks if an error is a GitPersonaError
func IsGitPersonaError(err error) bool {
	_, ok := err.(*GitPersonaError)
	return ok
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	if gpe, ok := err.(*GitPersonaError); ok {
		return gpe.Code
	}
	return ""
}

// GetErrorContext extracts the context from an error
func GetErrorContext(err error) map[string]interface{} {
	if gpe, ok := err.(*GitPersonaError); ok {
		return gpe.Context
	}
	return nil
}
