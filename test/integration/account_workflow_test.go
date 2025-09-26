package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// TestAccountWorkflow tests the complete account management workflow
func TestAccountWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Initialize services
	logger := observability.NewDefaultLogger()
	configService := services.NewRealConfigService(configPath, logger)

	// Test workflow steps
	t.Run("CreateAccount", func(t *testing.T) {
		account := &models.Account{
			Alias:          "testuser",
			Name:           "Test User",
			Email:          "test@example.com",
			GitHubUsername: "testuser",
			SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", "testuser"),
		}

		err := configService.SetAccount(ctx, account)
		if err != nil {
			t.Fatalf("Failed to create account: %v", err)
		}

		// Verify account was created
		retrievedAccount, err := configService.GetAccount(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to retrieve account: %v", err)
		}

		if retrievedAccount.Alias != "testuser" {
			t.Errorf("Expected alias 'testuser', got '%s'", retrievedAccount.Alias)
		}
	})

	t.Run("SwitchAccount", func(t *testing.T) {
		err := configService.SetCurrentAccount(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to switch to account: %v", err)
		}

		currentAccount := configService.GetCurrentAccount(ctx)
		if currentAccount != "testuser" {
			t.Errorf("Expected current account 'testuser', got '%s'", currentAccount)
		}
	})

	t.Run("ValidateAccount", func(t *testing.T) {
		account, err := configService.GetAccount(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get account: %v", err)
		}

		err = account.Validate()
		if err != nil {
			t.Errorf("Account validation failed: %v", err)
		}
	})

	t.Run("ListAccounts", func(t *testing.T) {
		accounts, err := configService.ListAccounts(ctx)
		if err != nil {
			t.Fatalf("Failed to list accounts: %v", err)
		}

		if len(accounts) == 0 {
			t.Error("Expected at least one account")
		}

		found := false
		for _, account := range accounts {
			if account.Alias == "testuser" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find 'testuser' in account list")
		}
	})

	t.Run("SaveAndReload", func(t *testing.T) {
		// Save configuration
		err := configService.Save(ctx)
		if err != nil {
			t.Fatalf("Failed to save configuration: %v", err)
		}

		// Reload configuration
		err = configService.Reload(ctx)
		if err != nil {
			t.Fatalf("Failed to reload configuration: %v", err)
		}

		// Verify account still exists after reload
		account, err := configService.GetAccount(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to get account after reload: %v", err)
		}

		if account.Alias != "testuser" {
			t.Errorf("Expected alias 'testuser' after reload, got '%s'", account.Alias)
		}
	})

	t.Run("DeleteAccount", func(t *testing.T) {
		err := configService.DeleteAccount(ctx, "testuser")
		if err != nil {
			t.Fatalf("Failed to delete account: %v", err)
		}

		// Verify account was deleted
		_, err = configService.GetAccount(ctx, "testuser")
		if err == nil {
			t.Error("Expected error when getting deleted account")
		}
	})
}

// TestMultiAccountWorkflow tests managing multiple accounts
func TestMultiAccountWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	logger := observability.NewDefaultLogger()
	configService := services.NewRealConfigService(configPath, logger)

	// Create multiple accounts
	accounts := []*models.Account{
		{
			Alias:          "personal",
			Name:           "John Doe",
			Email:          "john@personal.com",
			GitHubUsername: "johndoe",
			SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", "personal"),
		},
		{
			Alias:          "work",
			Name:           "John Doe",
			Email:          "john@company.com",
			GitHubUsername: "john-company",
			SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", "work"),
		},
		{
			Alias:          "client",
			Name:           "John Doe",
			Email:          "john@client.com",
			GitHubUsername: "john-client",
			SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", "client"),
		},
	}

	t.Run("CreateMultipleAccounts", func(t *testing.T) {
		for _, account := range accounts {
			err := configService.SetAccount(ctx, account)
			if err != nil {
				t.Fatalf("Failed to create account %s: %v", account.Alias, err)
			}
		}

		// Verify all accounts were created
		allAccounts, err := configService.ListAccounts(ctx)
		if err != nil {
			t.Fatalf("Failed to list accounts: %v", err)
		}

		if len(allAccounts) != len(accounts) {
			t.Errorf("Expected %d accounts, got %d", len(accounts), len(allAccounts))
		}
	})

	t.Run("SwitchBetweenAccounts", func(t *testing.T) {
		for _, account := range accounts {
			err := configService.SetCurrentAccount(ctx, account.Alias)
			if err != nil {
				t.Fatalf("Failed to switch to account %s: %v", account.Alias, err)
			}

			currentAccount := configService.GetCurrentAccount(ctx)
			if currentAccount != account.Alias {
				t.Errorf("Expected current account '%s', got '%s'", account.Alias, currentAccount)
			}

			// Mark account as used
			accountObj, err := configService.GetAccount(ctx, account.Alias)
			if err != nil {
				t.Fatalf("Failed to get account %s: %v", account.Alias, err)
			}

			accountObj.MarkAsUsed()
			err = configService.SetAccount(ctx, accountObj)
			if err != nil {
				t.Fatalf("Failed to update account %s: %v", account.Alias, err)
			}
		}
	})

	t.Run("CheckForConflicts", func(t *testing.T) {
		conflicts, err := configService.CheckForConflicts(ctx)
		if err != nil {
			t.Fatalf("Failed to check for conflicts: %v", err)
		}

		// Should not have conflicts with the test accounts
		if len(conflicts) > 0 {
			t.Errorf("Unexpected conflicts found: %v", conflicts)
		}

		// Create a conflicting account (same email)
		conflictAccount := &models.Account{
			Alias:          "conflict",
			Name:           "Conflict User",
			Email:          "john@personal.com", // Same as personal account
			GitHubUsername: "conflict-user",
			SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", "conflict"),
		}

		err = configService.SetAccount(ctx, conflictAccount)
		if err != nil {
			t.Fatalf("Failed to create conflict account: %v", err)
		}

		conflicts, err = configService.CheckForConflicts(ctx)
		if err != nil {
			t.Fatalf("Failed to check for conflicts: %v", err)
		}

		if len(conflicts) == 0 {
			t.Error("Expected conflicts for duplicate email")
		}
	})

	t.Run("ValidateConfiguration", func(t *testing.T) {
		err := configService.ValidateConfiguration(ctx)
		if err != nil {
			t.Errorf("Configuration validation failed: %v", err)
		}
	})
}

// TestAccountPerformance tests account operations performance
func TestAccountPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	logger := observability.NewDefaultLogger()
	configService := services.NewRealConfigService(configPath, logger)

	numAccounts := 50

	t.Run("BulkAccountCreation", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < numAccounts; i++ {
			account := &models.Account{
				Alias:          fmt.Sprintf("user%d", i),
				Name:           fmt.Sprintf("User %d", i),
				Email:          fmt.Sprintf("user%d@example.com", i),
				GitHubUsername: fmt.Sprintf("user%d", i),
				SSHKeyPath:     filepath.Join(tempDir, "ssh_keys", fmt.Sprintf("user%d", i)),
			}

			err := configService.SetAccount(ctx, account)
			if err != nil {
				t.Fatalf("Failed to create account %s: %v", account.Alias, err)
			}
		}

		duration := time.Since(start)

		// Performance requirement: should create 50 accounts in < 2s
		if duration > 2*time.Second {
			t.Errorf("Creating %d accounts took too long: %v", numAccounts, duration)
		}

		t.Logf("Created %d accounts in %v", numAccounts, duration)
	})

	t.Run("BulkAccountListing", func(t *testing.T) {
		start := time.Now()

		accounts, err := configService.ListAccounts(ctx)
		if err != nil {
			t.Fatalf("Failed to list accounts: %v", err)
		}

		duration := time.Since(start)

		if len(accounts) != numAccounts {
			t.Errorf("Expected %d accounts, got %d", numAccounts, len(accounts))
		}

		// Performance requirement: should list 50 accounts in < 100ms
		if duration > 100*time.Millisecond {
			t.Errorf("Listing %d accounts took too long: %v", numAccounts, duration)
		}

		t.Logf("Listed %d accounts in %v", numAccounts, duration)
	})

	t.Run("BulkAccountSwitching", func(t *testing.T) {
		start := time.Now()

		// Switch between first 10 accounts
		for i := 0; i < 10; i++ {
			alias := fmt.Sprintf("user%d", i)
			err := configService.SetCurrentAccount(ctx, alias)
			if err != nil {
				t.Fatalf("Failed to switch to account %s: %v", alias, err)
			}
		}

		duration := time.Since(start)

		// Performance requirement: should switch between 10 accounts in < 500ms
		if duration > 500*time.Millisecond {
			t.Errorf("Switching between 10 accounts took too long: %v", duration)
		}

		t.Logf("Switched between 10 accounts in %v", duration)
	})
}

// TestSSHKeyIntegration tests SSH key management integration
func TestSSHKeyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, "ssh_keys")

	// Create SSH directory
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create SSH directory: %v", err)
	}

	// Note: SSH service functionality would be tested here
	// For now, we'll focus on file operations and basic validation

	t.Run("CreateSSHKeyDirectory", func(t *testing.T) {
		keyPath := filepath.Join(sshDir, "test_key")

		// Create a dummy key file for testing
		err := os.WriteFile(keyPath, []byte("dummy-ssh-key-content"), 0600)
		if err != nil {
			t.Fatalf("Failed to create dummy SSH key: %v", err)
		}

		// Verify key file exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			t.Error("Key file should exist")
		}

		// Verify permissions
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatalf("Failed to stat key file: %v", err)
		}

		expectedMode := os.FileMode(0600)
		if info.Mode() != expectedMode {
			t.Errorf("Expected key permissions %v, got %v", expectedMode, info.Mode())
		}
	})
}
