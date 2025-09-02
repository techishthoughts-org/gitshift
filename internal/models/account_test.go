package models

import (
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
			account:   &Account{Alias: "test", Name: "Test User", Email: "test@example.com"},
			expectErr: nil,
		},
		{
			name:      "Empty alias",
			account:   &Account{Alias: "", Name: "Test User", Email: "test@example.com"},
			expectErr: ErrInvalidAlias,
		},
		{
			name:      "Empty name",
			account:   &Account{Alias: "test", Name: "", Email: "test@example.com"},
			expectErr: ErrInvalidName,
		},
		{
			name:      "Empty email",
			account:   &Account{Alias: "test", Name: "Test User", Email: ""},
			expectErr: ErrInvalidEmail,
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

	if config.GlobalGitConfig != false {
		t.Error("Expected GlobalGitConfig to be false by default")
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
