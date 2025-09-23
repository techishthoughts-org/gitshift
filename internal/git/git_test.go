package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techishthoughts/GitPersona/internal/models"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestIsGitRepo(t *testing.T) {
	manager := NewManager()

	// Test with a temporary directory that is not a git repo
	tempDir := t.TempDir()
	if manager.IsGitRepo(tempDir) {
		t.Error("Expected non-git directory to return false")
	}

	// Test with current directory (should be a git repo)
	if !manager.IsGitRepo(".") {
		t.Error("Expected current directory to be a git repo")
	}

	// Test with a non-existent directory
	if manager.IsGitRepo("/non/existent/path") {
		t.Error("Expected non-existent directory to return false")
	}
}

func TestSetLocalConfig(t *testing.T) {
	manager := NewManager()

	// Create a test account
	account := &models.Account{
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Test setting local config (this will fail if not in a git repo)
	err := manager.SetLocalConfig(account)
	if err != nil {
		// This is expected if we're not in a git repo
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestSetGlobalConfig(t *testing.T) {
	manager := NewManager()

	// Create a test account
	account := &models.Account{
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Test setting global config
	err := manager.SetGlobalConfig(account)
	if err != nil {
		// This is expected if git is not available
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestGetCurrentConfig(t *testing.T) {
	manager := NewManager()

	// Test getting current config
	name, email, err := manager.GetCurrentConfig()
	if err != nil {
		// This is expected if git is not available or no config is set
		if !strings.Contains(err.Error(), "failed to get user.name") &&
			!strings.Contains(err.Error(), "failed to get user.email") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If successful, verify the values are not empty
		if name == "" {
			t.Error("Expected name to be non-empty")
		}
		if email == "" {
			t.Error("Expected email to be non-empty")
		}
	}
}

func TestGenerateSSHCommand(t *testing.T) {
	manager := NewManager()

	// Test with empty SSH key path
	cmd := manager.GenerateSSHCommand("")
	if cmd != "" {
		t.Errorf("Expected empty command for empty SSH key path, got: %s", cmd)
	}

	// Test with non-existent SSH key
	cmd = manager.GenerateSSHCommand("/non/existent/key")
	if cmd != "" {
		t.Errorf("Expected empty command for non-existent SSH key, got: %s", cmd)
	}

	// Test with home directory expansion
	_, err := os.UserHomeDir()
	if err == nil {
		cmd = manager.GenerateSSHCommand("~/non/existent/key")
		if cmd != "" {
			t.Errorf("Expected empty command for non-existent SSH key with ~ expansion, got: %s", cmd)
		}
	}

	// Test with a temporary file
	tempFile := filepath.Join(t.TempDir(), "test_key")
	err = os.WriteFile(tempFile, []byte("test key content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	cmd = manager.GenerateSSHCommand(tempFile)
	expected := "ssh -i " + tempFile + " -o IdentitiesOnly=yes"
	if cmd != expected {
		t.Errorf("Expected command '%s', got: %s", expected, cmd)
	}
}

func TestValidateSSHKey(t *testing.T) {
	manager := NewManager()

	// Test with empty SSH key path (should be valid)
	err := manager.ValidateSSHKey("")
	if err != nil {
		t.Errorf("Expected no error for empty SSH key path, got: %v", err)
	}

	// Test with non-existent SSH key
	err = manager.ValidateSSHKey("/non/existent/key")
	if err != models.ErrSSHKeyNotFound {
		t.Errorf("Expected ErrSSHKeyNotFound, got: %v", err)
	}

	// Test with home directory expansion
	_, err = os.UserHomeDir()
	if err == nil {
		err = manager.ValidateSSHKey("~/non/existent/key")
		if err != models.ErrSSHKeyNotFound {
			t.Errorf("Expected ErrSSHKeyNotFound for ~ expansion, got: %v", err)
		}
	}

	// Test with a valid temporary file
	tempFile := filepath.Join(t.TempDir(), "test_key")
	err = os.WriteFile(tempFile, []byte("test key content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	err = manager.ValidateSSHKey(tempFile)
	if err != nil {
		t.Errorf("Expected no error for valid SSH key, got: %v", err)
	}

	// Test with a file that exists but is not readable
	tempFile = filepath.Join(t.TempDir(), "unreadable_key")
	err = os.WriteFile(tempFile, []byte("test key content"), 0000) // No permissions
	if err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	err = manager.ValidateSSHKey(tempFile)
	if err == nil {
		t.Error("Expected error for unreadable SSH key")
	}
}

func TestGetGitVersion(t *testing.T) {
	manager := NewManager()

	version, err := manager.GetGitVersion()
	if err != nil {
		// This is expected if git is not available
		if err != models.ErrGitNotFound {
			t.Errorf("Expected ErrGitNotFound, got: %v", err)
		}
	} else {
		// If successful, verify the version string
		if !strings.HasPrefix(version, "git version") {
			t.Errorf("Expected version to start with 'git version', got: %s", version)
		}
	}
}

func TestGetRemoteURL(t *testing.T) {
	manager := NewManager()

	// Test with default remote
	url, err := manager.GetRemoteURL("")
	if err != nil {
		// This is expected if not in a git repo or no remote configured
		if !strings.Contains(err.Error(), "failed to get remote URL") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If successful, verify the URL is not empty
		if url == "" {
			t.Error("Expected non-empty remote URL")
		}
	}

	// Test with specific remote
	url, err = manager.GetRemoteURL("origin")
	if err != nil {
		// This is expected if not in a git repo or no remote configured
		if !strings.Contains(err.Error(), "failed to get remote URL") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If successful, verify the URL is not empty
		if url == "" {
			t.Error("Expected non-empty remote URL")
		}
	}
}

func TestGetCurrentBranch(t *testing.T) {
	manager := NewManager()

	branch, err := manager.GetCurrentBranch()
	if err != nil {
		// This is expected if not in a git repo
		if !strings.Contains(err.Error(), "failed to get current branch") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If successful, verify the branch name is not empty
		if branch == "" {
			t.Error("Expected non-empty branch name")
		}
	}
}

func TestSetUserName(t *testing.T) {
	manager := NewManager()

	// Test setting local user name
	err := manager.setUserName("Test User", false)
	if err != nil {
		// This is expected if not in a git repo
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Test setting global user name
	err = manager.setUserName("Test User", true)
	if err != nil {
		// This is expected if git is not available
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestSetUserEmail(t *testing.T) {
	manager := NewManager()

	// Test setting local user email
	err := manager.setUserEmail("test@example.com", false)
	if err != nil {
		// This is expected if not in a git repo
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Test setting global user email
	err = manager.setUserEmail("test@example.com", true)
	if err != nil {
		// This is expected if git is not available
		if !strings.Contains(err.Error(), "git config failed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestGetConfigValue(t *testing.T) {
	manager := NewManager()

	// Test getting a config value
	value, err := manager.getConfigValue("user.name")
	if err != nil {
		// This is expected if git is not available or no config is set
		if !strings.Contains(err.Error(), "exit status") {
			t.Errorf("Unexpected error: %v", err)
		}
	} else {
		// If successful, verify the value is trimmed
		if strings.Contains(value, "\n") {
			t.Error("Expected config value to be trimmed")
		}
	}
}

func TestManagerIntegration(t *testing.T) {
	manager := NewManager()

	// Test that all methods can be called without panicking
	_ = manager.IsGitRepo(".")
	_ = manager.GenerateSSHCommand("")
	_ = manager.ValidateSSHKey("")
	_, _ = manager.GetGitVersion()
	_, _ = manager.GetRemoteURL("")
	_, _ = manager.GetCurrentBranch()
	_, _, _ = manager.GetCurrentConfig()

	// Test with a test account
	account := &models.Account{
		Name:  "Test User",
		Email: "test@example.com",
	}
	_ = manager.SetLocalConfig(account)
	_ = manager.SetGlobalConfig(account)
}

func TestSSHKeyPathExpansion(t *testing.T) {
	manager := NewManager()

	// Test home directory expansion in GenerateSSHCommand
	homeDir, err := os.UserHomeDir()
	if err == nil {
		// Create a test key in home directory
		testKeyPath := filepath.Join(homeDir, "test_key")
		err = os.WriteFile(testKeyPath, []byte("test key"), 0600)
		if err == nil {
			defer func() {
				if err := os.Remove(testKeyPath); err != nil {
					t.Logf("Failed to remove test key: %v", err)
				}
			}()

			cmd := manager.GenerateSSHCommand("~/test_key")
			expected := "ssh -i " + testKeyPath + " -o IdentitiesOnly=yes"
			if cmd != expected {
				t.Errorf("Expected command '%s', got: %s", expected, cmd)
			}
		}
	}

	// Test home directory expansion in ValidateSSHKey
	if err == nil {
		// Create a test key in home directory
		testKeyPath := filepath.Join(homeDir, "test_key_validate")
		err = os.WriteFile(testKeyPath, []byte("test key"), 0600)
		if err == nil {
			defer func() {
				if err := os.Remove(testKeyPath); err != nil {
					t.Logf("Failed to remove test key: %v", err)
				}
			}()

			err = manager.ValidateSSHKey("~/test_key_validate")
			if err != nil {
				t.Errorf("Expected no error for valid SSH key with ~ expansion, got: %v", err)
			}
		}
	}
}

func TestGitCommandExecution(t *testing.T) {
	manager := NewManager()

	// Test that git commands can be executed (if git is available)
	_, err := manager.GetGitVersion()
	if err == nil {
		// Git is available, test other commands
		_, _ = manager.GetCurrentBranch()
		_, _ = manager.GetRemoteURL("")
		_, _, _ = manager.GetCurrentConfig()
	} else {
		// Git is not available, verify we get the expected error
		if err != models.ErrGitNotFound {
			t.Errorf("Expected ErrGitNotFound when git is not available, got: %v", err)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	manager := NewManager()

	// Test error handling for various scenarios
	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "SetLocalConfig with nil account",
			test: func() error {
				return manager.SetLocalConfig(nil)
			},
		},
		{
			name: "SetGlobalConfig with nil account",
			test: func() error {
				return manager.SetGlobalConfig(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.test()
			if err == nil {
				t.Error("Expected error for nil account")
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()

	// Test concurrent access to manager methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Test various methods concurrently
			_ = manager.IsGitRepo(".")
			_ = manager.GenerateSSHCommand("")
			_ = manager.ValidateSSHKey("")
			_, _ = manager.GetGitVersion()
			_, _ = manager.GetRemoteURL("")
			_, _ = manager.GetCurrentBranch()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
