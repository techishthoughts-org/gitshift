package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestUserError_Error(t *testing.T) {
	tests := []struct {
		name        string
		userError   *UserError
		expected    []string
		notExpected []string
	}{
		{
			name: "SSH error with suggestions",
			userError: &UserError{
				Title:       "SSH Connection Failed",
				Message:     "Cannot connect to GitHub via SSH",
				Category:    CategorySSH,
				Suggestions: []string{"Check SSH key", "Verify permissions"},
				Debug:       "Connection timeout",
			},
			expected: []string{
				"🔑 SSH Connection Failed",
				"Cannot connect to GitHub via SSH",
				"💡 Try these solutions:",
				"1. Check SSH key",
				"2. Verify permissions",
				"🔍 Debug details: Connection timeout",
			},
		},
		{
			name: "Config error without debug",
			userError: &UserError{
				Title:       "Config Invalid",
				Message:     "Configuration file is malformed",
				Category:    CategoryConfig,
				Suggestions: []string{"Check config syntax"},
			},
			expected: []string{
				"⚙️ Config Invalid",
				"Configuration file is malformed",
				"💡 Try these solutions:",
				"1. Check config syntax",
			},
			notExpected: []string{"🔍 Debug details:"},
		},
		{
			name: "GitHub error",
			userError: &UserError{
				Title:    "API Rate Limited",
				Message:  "GitHub API rate limit exceeded",
				Category: CategoryGitHub,
			},
			expected: []string{
				"🐙 API Rate Limited",
				"GitHub API rate limit exceeded",
			},
		},
		{
			name: "Network error",
			userError: &UserError{
				Title:    "Connection Failed",
				Message:  "Cannot reach GitHub servers",
				Category: CategoryNetwork,
			},
			expected: []string{
				"🌐 Connection Failed",
				"Cannot reach GitHub servers",
			},
		},
		{
			name: "System error",
			userError: &UserError{
				Title:    "Permission Denied",
				Message:  "Insufficient system permissions",
				Category: CategorySystem,
			},
			expected: []string{
				"🖥️ Permission Denied",
				"Insufficient system permissions",
			},
		},
		{
			name: "Default category",
			userError: &UserError{
				Title:    "Unknown Error",
				Message:  "Something went wrong",
				Category: "unknown",
			},
			expected: []string{
				"❌ Unknown Error",
				"Something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.userError.Error()

			// Check expected content
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected error message to contain '%s', got:\n%s", expected, result)
				}
			}

			// Check not expected content
			for _, notExpected := range tt.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected error message to not contain '%s', got:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestGetCategoryEmoji(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{CategorySSH, "🔑"},
		{CategoryNetwork, "🌐"},
		{CategoryConfig, "⚙️"},
		{CategoryGitHub, "🐙"},
		{CategorySystem, "🖥️"},
		{"unknown", "❌"},
		{"", "❌"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			userError := &UserError{Category: tt.category}
			result := userError.getCategoryEmoji()

			if result != tt.expected {
				t.Errorf("Expected emoji '%s' for category '%s', got '%s'", tt.expected, tt.category, result)
			}
		})
	}
}

func TestNewSSHError(t *testing.T) {
	title := "SSH Connection Failed"
	message := "Cannot connect to GitHub"
	debug := "Connection timeout after 30s"

	err := NewSSHError(title, message, debug)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategorySSH {
		t.Errorf("Expected category '%s', got '%s'", CategorySSH, err.Category)
	}

	if err.Debug != debug {
		t.Errorf("Expected debug '%s', got '%s'", debug, err.Debug)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "gitpersona ssh test") {
		t.Error("Expected SSH suggestions to contain 'gitpersona ssh test'")
	}
}

func TestNewConfigError(t *testing.T) {
	title := "Config Invalid"
	message := "Configuration file is malformed"
	debug := "YAML parsing error at line 5"

	err := NewConfigError(title, message, debug)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategoryConfig {
		t.Errorf("Expected category '%s', got '%s'", CategoryConfig, err.Category)
	}

	if err.Debug != debug {
		t.Errorf("Expected debug '%s', got '%s'", debug, err.Debug)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "config.yaml") {
		t.Error("Expected config suggestions to contain 'config.yaml'")
	}
}

func TestNewGitHubError(t *testing.T) {
	title := "API Rate Limited"
	message := "GitHub API rate limit exceeded"
	debug := "Rate limit: 5000/5000 requests per hour"

	err := NewGitHubError(title, message, debug)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategoryGitHub {
		t.Errorf("Expected category '%s', got '%s'", CategoryGitHub, err.Category)
	}

	if err.Debug != debug {
		t.Errorf("Expected debug '%s', got '%s'", debug, err.Debug)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "gh auth status") {
		t.Error("Expected GitHub suggestions to contain 'gh auth status'")
	}
}

func TestNewNetworkError(t *testing.T) {
	title := "Connection Failed"
	message := "Cannot reach GitHub servers"
	debug := "DNS resolution failed"

	err := NewNetworkError(title, message, debug)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategoryNetwork {
		t.Errorf("Expected category '%s', got '%s'", CategoryNetwork, err.Category)
	}

	if err.Debug != debug {
		t.Errorf("Expected debug '%s', got '%s'", debug, err.Debug)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "internet connectivity") {
		t.Error("Expected network suggestions to contain 'internet connectivity'")
	}
}

func TestNewAccountError(t *testing.T) {
	title := "Account Not Found"
	message := "The specified account does not exist"
	accountAlias := "work"

	err := NewAccountError(title, message, accountAlias)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "gitpersona list") {
		t.Error("Expected account suggestions to contain 'gitpersona list'")
	}

	if !strings.Contains(suggestionsText, accountAlias) {
		t.Error("Expected suggestions to contain account alias")
	}
}

func TestNewAccountErrorWithoutAlias(t *testing.T) {
	title := "Account Not Found"
	message := "The specified account does not exist"

	err := NewAccountError(title, message, "")

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions don't contain update command when no alias
	suggestionsText := strings.Join(err.Suggestions, " ")
	if strings.Contains(suggestionsText, "--overwrite") {
		t.Error("Expected suggestions to not contain --overwrite when no alias provided")
	}
}

func TestNewValidationError(t *testing.T) {
	field := "email"
	value := "invalid-email"
	requirement := "Must be a valid email address"

	err := NewValidationError(field, value, requirement)

	expectedTitle := "Invalid " + field
	if err.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, err.Title)
	}

	expectedMessage := "The value '" + value + "' is not valid. " + requirement
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, field) {
		t.Error("Expected suggestions to contain field name")
	}
}

func TestNewSystemError(t *testing.T) {
	title := "Permission Denied"
	message := "Insufficient system permissions"
	debug := "Access denied to /etc/config"

	err := NewSystemError(title, message, debug)

	if err.Title != title {
		t.Errorf("Expected title '%s', got '%s'", title, err.Title)
	}

	if err.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, err.Message)
	}

	if err.Category != CategorySystem {
		t.Errorf("Expected category '%s', got '%s'", CategorySystem, err.Category)
	}

	if err.Debug != debug {
		t.Errorf("Expected debug '%s', got '%s'", debug, err.Debug)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "system permissions") {
		t.Error("Expected system suggestions to contain 'system permissions'")
	}
}

func TestWrapError(t *testing.T) {
	t.Run("wrap standard error", func(t *testing.T) {
		originalErr := errors.New("original error")
		title := "Wrapped Error"
		category := CategorySystem

		wrapped := WrapError(originalErr, title, category)

		if wrapped.Title != title {
			t.Errorf("Expected title '%s', got '%s'", title, wrapped.Title)
		}

		if wrapped.Category != category {
			t.Errorf("Expected category '%s', got '%s'", category, wrapped.Category)
		}

		if wrapped.Debug != originalErr.Error() {
			t.Errorf("Expected debug '%s', got '%s'", originalErr.Error(), wrapped.Debug)
		}

		if len(wrapped.Suggestions) == 0 {
			t.Error("Expected suggestions to be provided")
		}
	})

	t.Run("wrap UserError", func(t *testing.T) {
		originalErr := &UserError{
			Title:    "Original Title",
			Message:  "Original message",
			Category: CategorySSH,
		}
		title := "New Title"
		category := CategoryConfig

		wrapped := WrapError(originalErr, title, category)

		// Should return the original UserError, not create a new one
		if wrapped != originalErr {
			t.Error("Expected to return original UserError")
		}

		// Category should remain the same if already set
		if wrapped.Category != CategorySSH {
			t.Errorf("Expected category to remain '%s', got '%s'", CategorySSH, wrapped.Category)
		}
	})

	t.Run("wrap UserError with empty category", func(t *testing.T) {
		originalErr := &UserError{
			Title:    "Original Title",
			Message:  "Original message",
			Category: "",
		}
		title := "New Title"
		category := CategoryConfig

		wrapped := WrapError(originalErr, title, category)

		// Should return the original UserError with updated category
		if wrapped != originalErr {
			t.Error("Expected to return original UserError")
		}

		// Category should be updated
		if wrapped.Category != category {
			t.Errorf("Expected category to be updated to '%s', got '%s'", category, wrapped.Category)
		}
	})
}

func TestNewErrorHandler(t *testing.T) {
	t.Run("debug mode enabled", func(t *testing.T) {
		handler := NewErrorHandler(true)

		if handler == nil {
			t.Fatal("NewErrorHandler should return non-nil handler")
		}

		if !handler.debugMode {
			t.Error("Expected debug mode to be enabled")
		}
	})

	t.Run("debug mode disabled", func(t *testing.T) {
		handler := NewErrorHandler(false)

		if handler == nil {
			t.Fatal("NewErrorHandler should return non-nil handler")
		}

		if handler.debugMode {
			t.Error("Expected debug mode to be disabled")
		}
	})
}

func TestErrorHandler_HandleError(t *testing.T) {
	t.Run("handle UserError in debug mode", func(t *testing.T) {
		handler := NewErrorHandler(true)
		userErr := &UserError{
			Title:    "Test Error",
			Message:  "Test message",
			Category: CategorySSH,
			Debug:    "Debug information",
		}

		// This should not panic
		handler.HandleError(userErr)
	})

	t.Run("handle UserError without debug mode", func(t *testing.T) {
		handler := NewErrorHandler(false)
		userErr := &UserError{
			Title:    "Test Error",
			Message:  "Test message",
			Category: CategorySSH,
			Debug:    "Debug information",
		}

		// This should not panic
		handler.HandleError(userErr)
	})

	t.Run("handle standard error", func(t *testing.T) {
		handler := NewErrorHandler(false)
		standardErr := errors.New("standard error")

		// This should not panic
		handler.HandleError(standardErr)
	})
}

func TestNewAccountNotFoundError(t *testing.T) {
	alias := "work"
	availableAliases := []string{"personal", "company", "test"}

	err := NewAccountNotFoundError(alias, availableAliases)

	if err.Title != "Account Not Found" {
		t.Errorf("Expected title 'Account Not Found', got '%s'", err.Title)
	}

	expectedMessage := "Account '" + alias + "' not found in your configuration."
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain available aliases
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, strings.Join(availableAliases, ", ")) {
		t.Error("Expected suggestions to contain available aliases")
	}
}

func TestNewAccountNotFoundErrorWithoutAvailable(t *testing.T) {
	alias := "work"
	availableAliases := []string{}

	err := NewAccountNotFoundError(alias, availableAliases)

	if err.Title != "Account Not Found" {
		t.Errorf("Expected title 'Account Not Found', got '%s'", err.Title)
	}

	expectedMessage := "Account '" + alias + "' not found in your configuration."
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions don't contain available aliases when none provided
	suggestionsText := strings.Join(err.Suggestions, " ")
	if strings.Contains(suggestionsText, "Available accounts:") {
		t.Error("Expected suggestions to not contain 'Available accounts:' when no aliases provided")
	}
}

func TestNewNoAccountsError(t *testing.T) {
	err := NewNoAccountsError()

	if err.Title != "No Accounts Configured" {
		t.Errorf("Expected title 'No Accounts Configured', got '%s'", err.Title)
	}

	if err.Message != "You haven't configured any GitHub accounts yet." {
		t.Errorf("Expected specific message, got '%s'", err.Message)
	}

	if err.Category != CategoryUser {
		t.Errorf("Expected category '%s', got '%s'", CategoryUser, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "add-github") {
		t.Error("Expected suggestions to contain 'add-github'")
	}

	if !strings.Contains(suggestionsText, "discover") {
		t.Error("Expected suggestions to contain 'discover'")
	}
}

func TestNewSSHKeyNotFoundError(t *testing.T) {
	keyPath := "/path/to/key"
	account := "work"

	err := NewSSHKeyNotFoundError(keyPath, account)

	if err.Title != "SSH Key Not Found" {
		t.Errorf("Expected title 'SSH Key Not Found', got '%s'", err.Title)
	}

	expectedMessage := "SSH key file not found: " + keyPath
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategorySSH {
		t.Errorf("Expected category '%s', got '%s'", CategorySSH, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, account) {
		t.Error("Expected suggestions to contain account name")
	}

	if !strings.Contains(suggestionsText, "add-github") {
		t.Error("Expected suggestions to contain 'add-github'")
	}
}

func TestNewGitHubAuthError(t *testing.T) {
	operation := "fetch repositories"

	err := NewGitHubAuthError(operation)

	if err.Title != "GitHub Authentication Required" {
		t.Errorf("Expected title 'GitHub Authentication Required', got '%s'", err.Title)
	}

	expectedMessage := "Cannot perform '" + operation + "' - GitHub authentication required."
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategoryGitHub {
		t.Errorf("Expected category '%s', got '%s'", CategoryGitHub, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "gh auth login") {
		t.Error("Expected suggestions to contain 'gh auth login'")
	}

	if !strings.Contains(suggestionsText, "gh auth status") {
		t.Error("Expected suggestions to contain 'gh auth status'")
	}
}

func TestNewProjectConfigError(t *testing.T) {
	projectPath := "/path/to/project"
	issue := "invalid account reference"

	err := NewProjectConfigError(projectPath, issue)

	if err.Title != "Project Configuration Issue" {
		t.Errorf("Expected title 'Project Configuration Issue', got '%s'", err.Title)
	}

	expectedMessage := "Problem with project configuration in " + projectPath + ": " + issue
	if err.Message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Category != CategoryConfig {
		t.Errorf("Expected category '%s', got '%s'", CategoryConfig, err.Category)
	}

	if len(err.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	// Check that suggestions contain expected content
	suggestionsText := strings.Join(err.Suggestions, " ")
	if !strings.Contains(suggestionsText, "project set") {
		t.Error("Expected suggestions to contain 'project set'")
	}

	if !strings.Contains(suggestionsText, "project show") {
		t.Error("Expected suggestions to contain 'project show'")
	}
}

func TestErrorCategoryConstants(t *testing.T) {
	expectedCategories := []ErrorCategory{
		CategoryUser,
		CategorySystem,
		CategoryNetwork,
		CategorySSH,
		CategoryConfig,
		CategoryGitHub,
	}

	for _, category := range expectedCategories {
		if string(category) == "" {
			t.Errorf("Error category should not be empty")
		}
	}
}

func TestUserErrorFields(t *testing.T) {
	userErr := &UserError{
		Title:       "Test Title",
		Message:     "Test message",
		Suggestions: []string{"Suggestion 1", "Suggestion 2"},
		Category:    CategorySSH,
		Debug:       "Debug info",
		Code:        "TEST_CODE",
	}

	if userErr.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", userErr.Title)
	}

	if userErr.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", userErr.Message)
	}

	if len(userErr.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(userErr.Suggestions))
	}

	if userErr.Category != CategorySSH {
		t.Errorf("Expected category '%s', got '%s'", CategorySSH, userErr.Category)
	}

	if userErr.Debug != "Debug info" {
		t.Errorf("Expected debug 'Debug info', got '%s'", userErr.Debug)
	}

	if userErr.Code != "TEST_CODE" {
		t.Errorf("Expected code 'TEST_CODE', got '%s'", userErr.Code)
	}
}

// Benchmark tests
func BenchmarkUserError_Error(b *testing.B) {
	userErr := &UserError{
		Title:       "Test Error",
		Message:     "Test message with some content",
		Suggestions: []string{"Suggestion 1", "Suggestion 2", "Suggestion 3"},
		Category:    CategorySSH,
		Debug:       "Debug information",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = userErr.Error()
	}
}

func BenchmarkNewSSHError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewSSHError("Test Title", "Test message", "Debug info")
	}
}

func BenchmarkWrapError(b *testing.B) {
	originalErr := errors.New("original error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapError(originalErr, "Wrapped Error", CategorySystem)
	}
}

func BenchmarkErrorHandler_HandleError(b *testing.B) {
	handler := NewErrorHandler(false)
	userErr := &UserError{
		Title:    "Test Error",
		Message:  "Test message",
		Category: CategorySSH,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleError(userErr)
	}
}
