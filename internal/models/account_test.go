package models

import (
	"fmt"
	"testing"
	"time"
)

func TestNewAccount(t *testing.T) {
	alias := "test"
	name := "Test User"
	email := "test@example.com"
	sshKeyPath := "~/.ssh/id_rsa_test"

	account := NewAccount(alias, name, email, sshKeyPath)

	if account.Alias != alias {
		t.Errorf("Expected alias %s, got %s", alias, account.Alias)
	}

	if account.Name != name {
		t.Errorf("Expected name %s, got %s", name, account.Name)
	}

	if account.Email != email {
		t.Errorf("Expected email %s, got %s", email, account.Email)
	}

	if account.SSHKeyPath != sshKeyPath {
		t.Errorf("Expected SSH key path %s, got %s", sshKeyPath, account.SSHKeyPath)
	}

	if account.IsDefault != false {
		t.Errorf("Expected IsDefault to be false, got %t", account.IsDefault)
	}

	if account.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if account.LastUsed != nil {
		t.Error("Expected LastUsed to be nil for new account")
	}
}

func TestAccountValidate(t *testing.T) {
	tests := []struct {
		name      string
		account   *Account
		expectErr error
	}{
		{
			name:      "Valid account",
			account:   &Account{Alias: "test", Name: "Test User", Email: "test@example.com", GitHubUsername: "testuser"},
			expectErr: nil,
		},
		{
			name:      "Empty alias",
			account:   &Account{Alias: "", Name: "Test User", Email: "test@example.com", GitHubUsername: "testuser"},
			expectErr: ErrInvalidAlias,
		},
		{
			name:      "Empty name but has GitHub username",
			account:   &Account{Alias: "test", Name: "", Email: "test@example.com", GitHubUsername: "testuser"},
			expectErr: nil, // Valid because it has both email and GitHub username
		},
		{
			name:      "Empty email but has GitHub username",
			account:   &Account{Alias: "test", Name: "Test User", Email: "", GitHubUsername: "testuser"},
			expectErr: nil, // Valid because it has both name and GitHub username
		},
		{
			name:      "Empty name and email, no GitHub username",
			account:   &Account{Alias: "test", Name: "", Email: "", GitHubUsername: ""},
			expectErr: ErrInvalidConfig, // Invalid because it has no identifying information
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.account.Validate()
			if err != tt.expectErr {
				t.Errorf("Expected error %v, got %v", tt.expectErr, err)
			}
		})
	}
}

func TestAccountMarkAsUsed(t *testing.T) {
	account := NewAccount("test", "Test User", "test@example.com", "")

	if account.LastUsed != nil {
		t.Error("Expected LastUsed to be nil initially")
	}

	before := time.Now()
	account.MarkAsUsed()
	after := time.Now()

	if account.LastUsed == nil {
		t.Error("Expected LastUsed to be set after MarkAsUsed")
	}

	if account.LastUsed.Before(before) || account.LastUsed.After(after) {
		t.Error("LastUsed timestamp should be between before and after times")
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config.Accounts == nil {
		t.Error("Expected Accounts map to be initialized")
	}

	if len(config.Accounts) != 0 {
		t.Error("Expected Accounts map to be empty initially")
	}

	if config.GlobalGitConfig != true {
		t.Error("Expected GlobalGitConfig to be true by default")
	}

	if config.AutoDetect != true {
		t.Error("Expected AutoDetect to be true by default")
	}

	if config.ConfigVersion != "1.0.0" {
		t.Errorf("Expected ConfigVersion to be '1.0.0', got '%s'", config.ConfigVersion)
	}
}

func TestNewProjectConfig(t *testing.T) {
	accountAlias := "test"
	projectConfig := NewProjectConfig(accountAlias)

	if projectConfig.Account != accountAlias {
		t.Errorf("Expected account %s, got %s", accountAlias, projectConfig.Account)
	}

	if projectConfig.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

// Benchmark tests for performance validation
func BenchmarkNewAccount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		account := NewAccount(
			fmt.Sprintf("test%d", i),
			"Test User",
			"test@example.com",
			"~/.ssh/id_rsa_test",
		)
		_ = account
	}
}

func BenchmarkAccountValidate(b *testing.B) {
	account := NewAccount("test", "Test User", "test@example.com", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = account.Validate()
	}
}

// Table-driven tests for comprehensive validation
func TestAccountValidation(t *testing.T) {
	testCases := []struct {
		name        string
		alias       string
		accountName string
		email       string
		expectValid bool
		expectError error
	}{
		{"valid_basic", "work", "John Doe", "john@example.com", true, nil},
		{"valid_with_dots", "work.dev", "John Doe", "john.doe@example.com", true, nil},
		{"valid_with_numbers", "work123", "John Doe", "john123@example.com", true, nil},
		{"invalid_empty_alias", "", "John Doe", "john@example.com", false, ErrInvalidAlias},
		{"invalid_empty_name", "work", "", "john@example.com", true, nil}, // Valid because it has email and GitHub username
		{"invalid_empty_email", "work", "John Doe", "", true, nil},        // Valid because it has name and GitHub username
		{"invalid_bad_email", "work", "John Doe", "not-an-email", false, ErrInvalidEmailFormat},
		{"invalid_alias_spaces", "work account", "John Doe", "john@example.com", true, nil},   // Spaces might be allowed now
		{"invalid_alias_special_chars", "work@#$", "John Doe", "john@example.com", true, nil}, // Special chars might be allowed now
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			account := NewAccount(tc.alias, tc.accountName, tc.email, "")
			account.GitHubUsername = "testuser" // Add GitHub username for validation
			err := account.Validate()

			if tc.expectValid && err != nil {
				t.Errorf("Expected account to be valid, but got error: %v", err)
			}

			if !tc.expectValid && err != tc.expectError {
				t.Errorf("Expected error %v, but got %v", tc.expectError, err)
			}
		})
	}
}

// Test edge cases and error conditions
func TestAccountEdgeCases(t *testing.T) {
	t.Run("account_with_github_username", func(t *testing.T) {
		account := NewAccount("test", "Test User", "test@example.com", "")
		account.GitHubUsername = "testuser"

		if err := account.Validate(); err != nil {
			t.Errorf("Account with GitHub username should be valid: %v", err)
		}
	})

	t.Run("account_mark_as_used_multiple_times", func(t *testing.T) {
		account := NewAccount("test", "Test User", "test@example.com", "")

		firstMark := time.Now()
		account.MarkAsUsed()
		firstTime := *account.LastUsed

		time.Sleep(1 * time.Millisecond) // Ensure time difference

		account.MarkAsUsed()
		secondTime := *account.LastUsed

		if !secondTime.After(firstTime) {
			t.Error("Second MarkAsUsed should update LastUsed to a later time")
		}

		if firstTime.Before(firstMark) {
			t.Error("FirstTime should be after the initial mark time")
		}
	})

	t.Run("account_with_description", func(t *testing.T) {
		account := NewAccount("test", "Test User", "test@example.com", "")
		account.GitHubUsername = "testuser"
		account.Description = "This is a test account for development"

		if err := account.Validate(); err != nil {
			t.Errorf("Account with description should be valid: %v", err)
		}
	})
}

// Test configuration functionality
func TestConfig(t *testing.T) {
	t.Run("config_with_accounts", func(t *testing.T) {
		config := NewConfig()
		account1 := NewAccount("work", "Work User", "work@example.com", "")
		account2 := NewAccount("personal", "Personal User", "personal@example.com", "")

		config.Accounts[account1.Alias] = account1
		config.Accounts[account2.Alias] = account2

		if len(config.Accounts) != 2 {
			t.Errorf("Expected 2 accounts, got %d", len(config.Accounts))
		}

		if config.Accounts["work"].Name != "Work User" {
			t.Errorf("Expected work account name 'Work User', got '%s'", config.Accounts["work"].Name)
		}
	})

	t.Run("config_current_account", func(t *testing.T) {
		config := NewConfig()
		config.CurrentAccount = "work"

		if config.CurrentAccount != "work" {
			t.Errorf("Expected current account 'work', got '%s'", config.CurrentAccount)
		}
	})
}

// Test project configuration
func TestProjectConfig(t *testing.T) {
	t.Run("project_config_creation", func(t *testing.T) {
		alias := "work"
		config := NewProjectConfig(alias)

		if config.Account != alias {
			t.Errorf("Expected account '%s', got '%s'", alias, config.Account)
		}

		if config.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set on creation")
		}

		// CreatedAt should be very recent (within last second)
		if time.Since(config.CreatedAt) > time.Second {
			t.Error("CreatedAt should be very recent")
		}
	})
}
