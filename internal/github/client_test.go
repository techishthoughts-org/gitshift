package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
)

func TestNewClient(t *testing.T) {
	// Test with token
	client := NewClient("test-token")
	if client == nil {
		t.Fatal("NewClient should return non-nil client")
	}
	if client.client == nil {
		t.Error("Client should have non-nil github client")
	}
	if client.ctx == nil {
		t.Error("Client should have non-nil context")
	}

	// Test without token
	client = NewClient("")
	if client == nil {
		t.Fatal("NewClient should return non-nil client even without token")
	}
	if client.client == nil {
		t.Error("Client should have non-nil github client even without token")
	}
}

func TestClient_isAuthenticated(t *testing.T) {
	client := NewClient("")

	// Test unauthenticated client
	authenticated := client.isAuthenticated()
	// This will likely be false since we don't have a real token
	if authenticated {
		t.Log("Client appears to be authenticated (unexpected in test environment)")
	}
}

func TestClient_FetchUserInfo(t *testing.T) {
	client := NewClient("")

	// Test with invalid username
	_, err := client.FetchUserInfo("")
	if err == nil {
		t.Error("FetchUserInfo should return error for empty username")
	}

	// Test with non-existent user
	_, err = client.FetchUserInfo("nonexistentuser12345")
	if err == nil {
		t.Error("FetchUserInfo should return error for non-existent user")
	}
}

func TestClient_generateAlias(t *testing.T) {
	client := NewClient("")

	tests := []struct {
		name     string
		userInfo *UserInfo
		expected string
	}{
		{
			name: "with login",
			userInfo: &UserInfo{
				Login: "testuser",
			},
			expected: "testuser",
		},
		{
			name: "with company",
			userInfo: &UserInfo{
				Login:   "testuser",
				Company: "Test Company",
			},
			expected: "testuser", // The actual implementation returns the first candidate (login)
		},
		{
			name: "with name",
			userInfo: &UserInfo{
				Login: "testuser",
				Name:  "John Doe",
			},
			expected: "testuser", // The actual implementation returns the first candidate (login)
		},
		{
			name: "with short login",
			userInfo: &UserInfo{
				Login: "ab",
			},
			expected: "ab",
		},
		{
			name: "with user login",
			userInfo: &UserInfo{
				Login: "user",
			},
			expected: "user",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := client.generateAlias(test.userInfo)
			if result != test.expected {
				t.Errorf("generateAlias() = %q, expected %q", result, test.expected)
			}
		})
	}
}

func TestClient_promptForEmail(t *testing.T) {
	client := NewClient("")

	email := client.promptForEmail("testuser")
	expected := "testuser@users.noreply.github.com"
	if email != expected {
		t.Errorf("promptForEmail() = %q, expected %q", email, expected)
	}
}

func TestClient_generateSSHKey(t *testing.T) {
	client := NewClient("")

	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()
	_ = os.Setenv("HOME", tempDir)

	// Test SSH key generation
	keyPath, err := client.generateSSHKey("test", "test@example.com")
	if err != nil {
		t.Errorf("generateSSHKey should not return error: %v", err)
	}
	if keyPath == "" {
		t.Error("generateSSHKey should return non-empty key path")
	}

	// Check that key files were created
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Private key file should be created")
	}

	publicKeyPath := keyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file should be created")
	}

	// Test with existing key
	keyPath2, err := client.generateSSHKey("test", "test@example.com")
	if err != nil {
		t.Errorf("generateSSHKey should not return error for existing key: %v", err)
	}
	if keyPath != keyPath2 {
		t.Error("generateSSHKey should return same path for existing key")
	}
}

func TestClient_showSSHPublicKey(t *testing.T) {
	client := NewClient("")

	// Create temporary directory and key file
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key")
	publicKeyPath := keyPath + ".pub"

	// Create a mock public key file
	publicKeyData := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test@example.com"
	if err := os.WriteFile(publicKeyPath, []byte(publicKeyData), 0644); err != nil {
		t.Fatalf("Failed to create mock public key file: %v", err)
	}

	// Test showing public key
	err := client.showSSHPublicKey(keyPath)
	if err != nil {
		t.Errorf("showSSHPublicKey should not return error: %v", err)
	}

	// Test with non-existent key
	err = client.showSSHPublicKey("/nonexistent/key")
	if err == nil {
		t.Error("showSSHPublicKey should return error for non-existent key")
	}
}

func TestClient_copyToClipboard(t *testing.T) {
	client := NewClient("")

	// Test clipboard functionality (will likely not work in test environment)
	client.copyToClipboard("test text")
	// This should not panic or error, even if clipboard is not available
}

func TestCommandExists(t *testing.T) {
	// Test with existing command
	if !commandExists("echo") {
		t.Error("commandExists should return true for 'echo'")
	}

	// Test with non-existent command
	if commandExists("nonexistentcommand12345") {
		t.Error("commandExists should return false for non-existent command")
	}
}

func TestClient_GetGitHubToken(t *testing.T) {
	client := NewClient("")

	// Test getting token (will likely fail in test environment)
	_, err := client.GetGitHubToken()
	if err == nil {
		t.Log("GitHub token retrieved successfully (unexpected in test environment)")
	}
}

func TestClient_FetchUserRepositories(t *testing.T) {
	client := NewClient("")

	// Test with empty username
	_, err := client.FetchUserRepositories("")
	if err == nil {
		t.Error("FetchUserRepositories should return error for empty username")
	}

	// Test with non-existent user
	_, err = client.FetchUserRepositories("nonexistentuser12345")
	if err == nil {
		t.Error("FetchUserRepositories should return error for non-existent user")
	}
}

func TestClient_FetchAuthenticatedUserRepositories(t *testing.T) {
	client := NewClient("")

	// Test without authentication
	_, err := client.FetchAuthenticatedUserRepositories()
	if err == nil {
		t.Error("FetchAuthenticatedUserRepositories should return error when not authenticated")
	}
}

func TestClient_UploadSSHKeyToGitHub(t *testing.T) {
	client := NewClient("")

	// Test without authentication
	err := client.UploadSSHKeyToGitHub("/path/to/key", "test")
	if err == nil {
		t.Error("UploadSSHKeyToGitHub should return error when not authenticated")
	}

	// Test with non-existent key
	err = client.UploadSSHKeyToGitHub("/nonexistent/key", "test")
	if err == nil {
		t.Error("UploadSSHKeyToGitHub should return error for non-existent key")
	}
}

func TestClient_findExistingWorkingSSHKey(t *testing.T) {
	client := NewClient("")

	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()
	_ = os.Setenv("HOME", tempDir)

	// Test with no existing keys
	keyPath := client.findExistingWorkingSSHKey("test", "test@example.com")
	if keyPath != "" {
		t.Error("findExistingWorkingSSHKey should return empty string when no keys exist")
	}
}

func TestClient_testSSHKeyWithGitHub(t *testing.T) {
	client := NewClient("")

	// Test with non-existent key
	result := client.testSSHKeyWithGitHub("/nonexistent/key")
	if result {
		t.Error("testSSHKeyWithGitHub should return false for non-existent key")
	}
}

func TestClient_SetupAccountFromUsername(t *testing.T) {
	client := NewClient("")

	// Test with empty username
	_, err := client.SetupAccountFromUsername("", "", "", "")
	if err == nil {
		t.Error("SetupAccountFromUsername should return error for empty username")
	}

	// Test with non-existent user
	_, err = client.SetupAccountFromUsername("nonexistentuser12345", "", "", "")
	if err == nil {
		t.Error("SetupAccountFromUsername should return error for non-existent user")
	}
}

func TestUserInfo_Structure(t *testing.T) {
	userInfo := &UserInfo{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://example.com/avatar.png",
		Company:   "Test Company",
		Bio:       "Test bio",
		Location:  "Test Location",
	}

	if userInfo.Login == "" {
		t.Error("Login should not be empty")
	}
	if userInfo.Name == "" {
		t.Error("Name should not be empty")
	}
	if userInfo.Email == "" {
		t.Error("Email should not be empty")
	}
}

func TestRepository_Structure(t *testing.T) {
	repo := &Repository{
		Name:        "test-repo",
		FullName:    "user/test-repo",
		Description: "Test repository",
		Private:     false,
		Fork:        false,
		Archived:    false,
		Language:    "Go",
		Stars:       10,
		Forks:       5,
		UpdatedAt:   "2023-01-01",
		HTMLURL:     "https://github.com/user/test-repo",
	}

	if repo.Name == "" {
		t.Error("Name should not be empty")
	}
	if repo.FullName == "" {
		t.Error("FullName should not be empty")
	}
	if repo.HTMLURL == "" {
		t.Error("HTMLURL should not be empty")
	}
}

func TestClient_Integration(t *testing.T) {
	client := NewClient("")

	// Test that client can be created and basic methods don't panic
	if client == nil {
		t.Fatal("Client should be created successfully")
	}

	// Test context is set
	if client.ctx == nil {
		t.Error("Client context should be set")
	}

	// Test that context is not cancelled
	select {
	case <-client.ctx.Done():
		t.Error("Client context should not be cancelled")
	default:
		// Context is not cancelled, which is expected
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	client := NewClient("")

	// Test various error conditions
	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "FetchUserInfo with empty username",
			test: func() error {
				_, err := client.FetchUserInfo("")
				return err
			},
		},
		{
			name: "FetchUserRepositories with empty username",
			test: func() error {
				_, err := client.FetchUserRepositories("")
				return err
			},
		},
		{
			name: "UploadSSHKeyToGitHub without authentication",
			test: func() error {
				return client.UploadSSHKeyToGitHub("/nonexistent/key", "test")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.test()
			if err == nil {
				t.Error("Expected error for " + test.name)
			}
		})
	}
}

func TestClient_Concurrency(t *testing.T) {
	client := NewClient("")

	// Test concurrent access to client methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test various methods concurrently
			_ = client.isAuthenticated()
			_ = client.generateAlias(&UserInfo{Login: "testuser"})
			_ = client.promptForEmail("testuser")
			_ = commandExists("echo")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestClient_EdgeCases(t *testing.T) {
	client := NewClient("")

	// Test with nil user info - this will panic, so we need to recover
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("generateAlias should panic for nil user info")
			}
		}()
		_ = client.generateAlias(nil)
	}()

	// Test with empty user info
	alias := client.generateAlias(&UserInfo{})
	if alias != "" {
		t.Error("generateAlias should handle empty user info gracefully")
	}

	// Test with special characters in username
	email := client.promptForEmail("user@domain.com")
	if !strings.Contains(email, "user@domain.com") {
		t.Error("promptForEmail should handle special characters in username")
	}
}

func TestClient_SSHKeyGeneration(t *testing.T) {
	client := NewClient("")

	// Create temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer func() {
		_ = os.Setenv("HOME", originalHome)
	}()
	_ = os.Setenv("HOME", tempDir)

	// Test SSH key generation with different aliases
	aliases := []string{"test", "user", "work", "personal"}

	for _, alias := range aliases {
		keyPath, err := client.generateSSHKey(alias, alias+"@example.com")
		if err != nil {
			t.Errorf("generateSSHKey failed for alias %s: %v", alias, err)
		}
		if keyPath == "" {
			t.Errorf("generateSSHKey returned empty path for alias %s", alias)
		}

		// Check that key files exist
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			t.Errorf("Private key file not created for alias %s", alias)
		}

		publicKeyPath := keyPath + ".pub"
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Errorf("Public key file not created for alias %s", alias)
		}
	}
}

func TestClient_AccountCreation(t *testing.T) {
	// Test creating account from models
	account := models.NewAccount("test", "Test User", "test@example.com", "/path/to/key")
	if account == nil {
		t.Fatal("NewAccount should return non-nil account")
	}

	if account.Alias != "test" {
		t.Errorf("Expected alias 'test', got %q", account.Alias)
	}
	if account.Name != "Test User" {
		t.Errorf("Expected name 'Test User', got %q", account.Name)
	}
	if account.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %q", account.Email)
	}
	if account.SSHKeyPath != "/path/to/key" {
		t.Errorf("Expected SSH key path '/path/to/key', got %q", account.SSHKeyPath)
	}
}

func TestClient_Performance(t *testing.T) {
	client := NewClient("")

	// Test that basic operations complete in reasonable time
	start := time.Now()
	_ = client.isAuthenticated()
	duration := time.Since(start)

	if duration > 1*time.Second {
		t.Errorf("isAuthenticated took too long: %v", duration)
	}

	start = time.Now()
	_ = client.generateAlias(&UserInfo{Login: "testuser"})
	duration = time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("generateAlias took too long: %v", duration)
	}
}
