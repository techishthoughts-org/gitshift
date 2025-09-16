package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		message  string
		expected string
	}{
		{
			name:     "config load failed",
			code:     ErrCodeConfigLoadFailed,
			message:  "failed to load config",
			expected: "failed to load config",
		},
		{
			name:     "account not found",
			code:     ErrCodeAccountNotFound,
			message:  "account not found",
			expected: "account not found",
		},
		{
			name:     "ssh validation failed",
			code:     ErrCodeSSHValidationFailed,
			message:  "SSH key validation failed",
			expected: "SSH key validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message)
			if err == nil {
				t.Fatal("New() returned nil error")
			}

			gitPersonaErr := err

			if gitPersonaErr.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, gitPersonaErr.Code)
			}

			if gitPersonaErr.Message != tt.expected {
				t.Errorf("Expected message %s, got %s", tt.expected, gitPersonaErr.Message)
			}

			if gitPersonaErr.Timestamp.IsZero() {
				t.Error("Expected timestamp to be set")
			}

			if gitPersonaErr.Location == "" {
				t.Error("Expected location to be set")
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeConfigLoadFailed
	message := "wrapped error"

	err := Wrap(originalErr, code, message)
	if err == nil {
		t.Fatal("Wrap() returned nil error")
	}

	gitPersonaErr := err

	if gitPersonaErr.Code != code {
		t.Errorf("Expected code %s, got %s", code, gitPersonaErr.Code)
	}

	if gitPersonaErr.Message != message {
		t.Errorf("Expected message %s, got %s", message, gitPersonaErr.Message)
	}

	if gitPersonaErr.Cause != originalErr {
		t.Errorf("Expected original error %v, got %v", originalErr, gitPersonaErr.Cause)
	}
}

func TestWithContext(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeAccountNotFound
	message := "account not found"

	err := Wrap(originalErr, code, message)
	err = err.WithContext("account_id", "test-account")
	err = err.WithContext("user_id", "test-user")
	if err == nil {
		t.Fatal("WithContext() returned nil error")
	}

	gitPersonaErr := err

	if gitPersonaErr.Code != code {
		t.Errorf("Expected code %s, got %s", code, gitPersonaErr.Code)
	}

	if gitPersonaErr.Message != message {
		t.Errorf("Expected message %s, got %s", message, gitPersonaErr.Message)
	}

	if gitPersonaErr.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if gitPersonaErr.Context["account_id"] != "test-account" {
		t.Errorf("Expected account_id 'test-account', got %v", gitPersonaErr.Context["account_id"])
	}
}

func TestWithContextMap(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeSSHValidationFailed
	message := "SSH validation failed"
	context := map[string]interface{}{
		"ssh_key_path": "/path/to/key",
		"host":         "github.com",
	}

	err := Wrap(originalErr, code, message)
	err = err.WithContextMap(context)
	if err == nil {
		t.Fatal("WithContextMap() returned nil error")
	}

	gitPersonaErr := err

	if gitPersonaErr.Code != code {
		t.Errorf("Expected code %s, got %s", code, gitPersonaErr.Code)
	}

	if gitPersonaErr.Message != message {
		t.Errorf("Expected message %s, got %s", message, gitPersonaErr.Message)
	}

	if gitPersonaErr.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if gitPersonaErr.Context["ssh_key_path"] != "/path/to/key" {
		t.Errorf("Expected ssh_key_path '/path/to/key', got %v", gitPersonaErr.Context["ssh_key_path"])
	}
}

func TestError(t *testing.T) {
	code := ErrCodeConfigLoadFailed
	message := "failed to load config"
	err := New(code, message)

	errorString := err.Error()
	expected := "[config_load_failed] failed to load config"
	if errorString != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, errorString)
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeAccountNotFound
	message := "wrapped error"

	err := Wrap(originalErr, code, message)
	unwrapped := err.Unwrap()

	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error %v, got %v", originalErr, unwrapped)
	}
}

func TestIs(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeConfigLoadFailed
	message := "wrapped error"

	err := Wrap(originalErr, code, message)

	// Test that it matches the original error
	if !errors.Is(err, originalErr) {
		t.Error("Expected error to match original error")
	}

	// Test that it doesn't match a different error
	differentErr := errors.New("different error")
	if errors.Is(err, differentErr) {
		t.Error("Expected error to not match different error")
	}
}

func TestGetCallerLocation(t *testing.T) {
	err := New(ErrCodeConfigLoadFailed, "test error")
	gitPersonaErr := err

	caller := gitPersonaErr.Location
	if caller == "" {
		t.Error("Expected caller location to be set")
	}

	// Should contain the test file name
	if !strings.Contains(caller, "errors_test.go") {
		t.Errorf("Expected caller to contain test file name, got %s", caller)
	}
}

func TestGetStackTrace(t *testing.T) {
	err := New(ErrCodeConfigLoadFailed, "test error")
	gitPersonaErr := err

	stackTrace := gitPersonaErr.Stack
	if len(stackTrace) == 0 {
		t.Error("Expected stack trace to be set")
	}

	// Should contain some stack trace information
	if len(stackTrace) < 1 {
		t.Errorf("Expected stack trace to have at least one entry, got %d", len(stackTrace))
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code ErrorCode
	}{
		{"ConfigLoadFailed", ConfigLoadFailed(errors.New("test"), map[string]interface{}{}), ErrCodeConfigLoadFailed},
		{"AccountNotFound", AccountNotFound("test", map[string]interface{}{}), ErrCodeAccountNotFound},
		{"SSHValidationFailed", SSHValidationFailed(errors.New("test"), map[string]interface{}{}), ErrCodeSSHValidationFailed},
		{"ValidationFailed", ValidationFailed(errors.New("test"), map[string]interface{}{}), ErrCodeValidationFailed},
		{"InvalidInput", InvalidInput("test", "value", map[string]interface{}{}), ErrCodeInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("Predefined error returned nil")
			}

			gitPersonaErr, ok := tt.err.(*GitPersonaError)
			if !ok {
				t.Fatalf("Predefined error returned %T, expected *GitPersonaError", tt.err)
			}

			if gitPersonaErr.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, gitPersonaErr.Code)
			}
		})
	}
}

func TestIsGitPersonaError(t *testing.T) {
	// Test with GitPersonaError
	err := New(ErrCodeConfigLoadFailed, "test error")
	if !IsGitPersonaError(err) {
		t.Error("Expected IsGitPersonaError to return true for GitPersonaError")
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	if IsGitPersonaError(regularErr) {
		t.Error("Expected IsGitPersonaError to return false for regular error")
	}

	// Test with nil
	if IsGitPersonaError(nil) {
		t.Error("Expected IsGitPersonaError to return false for nil")
	}
}

func TestGetErrorCode(t *testing.T) {
	code := ErrCodeConfigLoadFailed
	err := New(code, "test error")

	retrievedCode := GetErrorCode(err)
	if retrievedCode != code {
		t.Errorf("Expected error code %s, got %s", code, retrievedCode)
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	retrievedCode = GetErrorCode(regularErr)
	if retrievedCode != "" {
		t.Errorf("Expected empty error code for regular error, got %s", retrievedCode)
	}
}

func TestGetErrorContext(t *testing.T) {
	context := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := Wrap(errors.New("test"), ErrCodeConfigLoadFailed, "test error")
	err = err.WithContextMap(context)

	retrievedContext := GetErrorContext(err)
	if retrievedContext == nil {
		t.Fatal("Expected context to be retrieved")
	}

	if retrievedContext["key1"] != "value1" {
		t.Errorf("Expected key1 'value1', got %v", retrievedContext["key1"])
	}

	// Test with regular error
	regularErr := errors.New("regular error")
	retrievedContext = GetErrorContext(regularErr)
	if retrievedContext != nil {
		t.Errorf("Expected nil context for regular error, got %v", retrievedContext)
	}
}

func TestGitPersonaErrorFields(t *testing.T) {
	originalErr := errors.New("original error")
	code := ErrCodeConfigLoadFailed
	message := "test error"
	context := map[string]interface{}{
		"test_key": "test_value",
	}

	err := Wrap(originalErr, code, message)
	err = err.WithContextMap(context)
	gitPersonaErr := err

	// Test all fields are set correctly
	if gitPersonaErr.Code != code {
		t.Errorf("Expected code %s, got %s", code, gitPersonaErr.Code)
	}

	if gitPersonaErr.Message != message {
		t.Errorf("Expected message %s, got %s", message, gitPersonaErr.Message)
	}

	if gitPersonaErr.Cause != originalErr {
		t.Errorf("Expected original error %v, got %v", originalErr, gitPersonaErr.Cause)
	}

	if gitPersonaErr.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if gitPersonaErr.Context["test_key"] != "test_value" {
		t.Errorf("Expected test_key 'test_value', got %v", gitPersonaErr.Context["test_key"])
	}

	if gitPersonaErr.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	if gitPersonaErr.Location == "" {
		t.Error("Expected location to be set")
	}

	if len(gitPersonaErr.Stack) == 0 {
		t.Error("Expected stack trace to be set")
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Test that all error codes are properly defined
	expectedCodes := []ErrorCode{
		ErrCodeConfigLoadFailed,
		ErrCodeConfigSaveFailed,
		ErrCodeConfigInvalid,
		ErrCodeConfigNotFound,
		ErrCodeAccountNotFound,
		ErrCodeAccountInvalid,
		ErrCodeAccountExists,
		ErrCodeAccountSwitchFailed,
		ErrCodeSSHValidationFailed,
		ErrCodeSSHKeyNotFound,
		ErrCodeSSHKeyInvalid,
		ErrCodeSSHConnectionFailed,
		ErrCodeSSHPermissionDenied,
		ErrCodeGitHubAuthFailed,
		ErrCodeGitHubAPIFailed,
		ErrCodeGitHubRateLimited,
		ErrCodeGitHubNotFound,
		ErrCodeGitConfigFailed,
		ErrCodeGitRepositoryError,
		ErrCodeGitCommandFailed,
		ErrCodeValidationFailed,
		ErrCodeInvalidInput,
		ErrCodeMissingRequired,
		ErrCodePermissionDenied,
		ErrCodeNetworkError,
		ErrCodeTimeout,
		ErrCodeInternal,
	}

	for _, code := range expectedCodes {
		if string(code) == "" {
			t.Errorf("Error code %s is empty", code)
		}
	}
}
