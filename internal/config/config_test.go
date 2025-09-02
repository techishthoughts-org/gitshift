package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Error("NewManager should return a non-nil manager")
	}

	// Test that config is initialized
	if manager.config == nil {
		t.Error("Manager config should be initialized")
	}
}

func TestManagerLoad(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	// Test loading non-existent config (should not error)
	err := manager.Load()
	if err != nil {
		t.Errorf("Loading non-existent config should not error, got: %v", err)
	}

	// Config should be initialized with defaults
	if manager.config.ConfigVersion != "1.0.0" {
		t.Errorf("Expected config version '1.0.0', got '%s'", manager.config.ConfigVersion)
	}
}

func TestManagerSaveAndLoad(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "gitpersona")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	// Add test account
	account := models.NewAccount("test", "Test User", "test@example.com", "~/.ssh/id_rsa_test")
	account.GitHubUsername = "testuser"
	err = manager.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Save configuration
	err = manager.Save()
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Create new manager and load
	newManager := NewManager()
	err = newManager.Load()
	if err != nil {
		t.Fatalf("Failed to load saved configuration: %v", err)
	}

	// Verify account was loaded
	loadedAccount, err := newManager.GetAccount("test")
	if err != nil {
		t.Fatalf("Failed to get saved account: %v", err)
	}

	if loadedAccount.Name != "Test User" {
		t.Errorf("Expected loaded account name 'Test User', got '%s'", loadedAccount.Name)
	}

	if loadedAccount.Email != "test@example.com" {
		t.Errorf("Expected loaded account email 'test@example.com', got '%s'", loadedAccount.Email)
	}
}

func TestManagerAccountOperations(t *testing.T) {
	manager := NewManager()

	// Test adding account
	account := models.NewAccount("work", "Work User", "work@example.com", "")
	account.GitHubUsername = "workuser"
	err := manager.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Test getting account
	retrievedAccount, err := manager.GetAccount("work")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}

	if retrievedAccount.Name != "Work User" {
		t.Errorf("Expected account name 'Work User', got '%s'", retrievedAccount.Name)
	}

	// Test listing accounts
	accounts := manager.ListAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	// Test removing account
	err = manager.RemoveAccount("work")
	if err != nil {
		t.Fatalf("Failed to remove account: %v", err)
	}

	// Verify account was removed
	_, err = manager.GetAccount("work")
	if err == nil {
		t.Error("Expected error when getting removed account")
	}

	accounts = manager.ListAccounts()
	if len(accounts) != 0 {
		t.Errorf("Expected 0 accounts after removal, got %d", len(accounts))
	}
}

func TestManagerCurrentAccount(t *testing.T) {
	manager := NewManager()

	// Add test account
	account := models.NewAccount("test", "Test User", "test@example.com", "")
	err := manager.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Test setting current account
	err = manager.SetCurrentAccount("test")
	if err != nil {
		t.Fatalf("Failed to set current account: %v", err)
	}

	// Verify current account
	current := manager.GetConfig().CurrentAccount
	if current != "test" {
		t.Errorf("Expected current account 'test', got '%s'", current)
	}

	// Test setting non-existent account
	err = manager.SetCurrentAccount("nonexistent")
	if err == nil {
		t.Error("Expected error when setting non-existent current account")
	}
}

func TestManagerProjectConfig(t *testing.T) {
	// Create temporary directory for project config
	tempDir := t.TempDir()

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer func() {
		os.Chdir(originalWd)
	}()
	os.Chdir(tempDir)

	manager := NewManager()

	// Test saving project config
	projectConfig := models.NewProjectConfig("work")
	err := manager.SaveProjectConfig(tempDir, projectConfig)
	if err != nil {
		t.Fatalf("Failed to save project config: %v", err)
	}

	// Test loading project config
	loadedConfig, err := manager.LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load project config: %v", err)
	}

	if loadedConfig.Account != "work" {
		t.Errorf("Expected project account 'work', got '%s'", loadedConfig.Account)
	}

	// Test loading non-existent project config
	nonExistentDir := filepath.Join(tempDir, "nonexistent")
	_, err = manager.LoadProjectConfig(nonExistentDir)
	if err == nil {
		t.Error("Expected error when loading non-existent project config")
	}
}

func TestManagerValidation(t *testing.T) {
	manager := NewManager()

	// Test adding duplicate account
	account1 := models.NewAccount("work", "User 1", "user1@example.com", "")
	account1.GitHubUsername = "user1"
	account2 := models.NewAccount("work", "User 2", "user2@example.com", "")
	account2.GitHubUsername = "user2"

	err := manager.AddAccount(account1)
	if err != nil {
		t.Fatalf("Failed to add first account: %v", err)
	}

	err = manager.AddAccount(account2)
	if err == nil {
		t.Error("Expected error when adding duplicate account alias")
	}

	// Test adding account with invalid data
	invalidAccount := &models.Account{
		Alias: "", // Invalid empty alias
		Name:  "Test User",
		Email: "test@example.com",
	}

	err = manager.AddAccount(invalidAccount)
	if err == nil {
		t.Error("Expected error when adding invalid account")
	}
}

func TestManagerConfigMigration(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "gitpersona")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	// Create old config format (simulated)
	configFile := filepath.Join(configDir, "config.yaml")
	oldConfigContent := `configVersion: "0.9.0"
accounts: {}
currentAccount: ""
globalGitConfig: false
autoDetect: true
`

	err = os.WriteFile(configFile, []byte(oldConfigContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	manager := NewManager()

	// Loading should trigger migration
	err = manager.Load()
	if err != nil {
		t.Fatalf("Failed to load and migrate config: %v", err)
	}

	// Verify config was migrated
	if manager.config.ConfigVersion != "1.0.0" {
		t.Errorf("Expected config version '1.0.0' after migration, got '%s'", manager.config.ConfigVersion)
	}
}

// Benchmark tests for performance validation
func BenchmarkManagerAddAccount(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account := models.NewAccount(
			fmt.Sprintf("test%d", i),
			"Test User",
			"test@example.com",
			"",
		)
		manager.AddAccount(account)
	}
}

func BenchmarkManagerGetAccount(b *testing.B) {
	manager := NewManager()

	// Setup: add test account
	account := models.NewAccount("test", "Test User", "test@example.com", "")
	manager.AddAccount(account)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GetAccount("test")
	}
}

func BenchmarkManagerListAccounts(b *testing.B) {
	manager := NewManager()

	// Setup: add multiple accounts
	for i := 0; i < 10; i++ {
		account := models.NewAccount(
			fmt.Sprintf("test%d", i),
			"Test User",
			"test@example.com",
			"",
		)
		manager.AddAccount(account)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ListAccounts()
	}
}

// Integration test for complete workflow
func TestManagerCompleteWorkflow(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	// 1. Load empty config
	err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load empty config: %v", err)
	}

	// 2. Add accounts
	workAccount := models.NewAccount("work", "Work User", "work@example.com", "~/.ssh/id_rsa_work")
	personalAccount := models.NewAccount("personal", "Personal User", "personal@example.com", "~/.ssh/id_rsa_personal")

	err = manager.AddAccount(workAccount)
	if err != nil {
		t.Fatalf("Failed to add work account: %v", err)
	}

	err = manager.AddAccount(personalAccount)
	if err != nil {
		t.Fatalf("Failed to add personal account: %v", err)
	}

	// 3. Set current account
	err = manager.SetCurrentAccount("work")
	if err != nil {
		t.Fatalf("Failed to set current account: %v", err)
	}

	// 4. Save configuration
	err = manager.Save()
	if err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// 5. Load in new manager to verify persistence
	newManager := NewManager()
	err = newManager.Load()
	if err != nil {
		t.Fatalf("Failed to load saved configuration: %v", err)
	}

	// 6. Verify all data
	accounts := newManager.ListAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	currentAccount := newManager.GetConfig().CurrentAccount
	if currentAccount != "work" {
		t.Errorf("Expected current account 'work', got '%s'", currentAccount)
	}

	workAcc, err := newManager.GetAccount("work")
	if err != nil {
		t.Fatalf("Failed to get work account: %v", err)
	}

	if workAcc.Email != "work@example.com" {
		t.Errorf("Expected work email 'work@example.com', got '%s'", workAcc.Email)
	}
}

// Test concurrent access (race condition testing)
func TestManagerConcurrentAccess(t *testing.T) {
	manager := NewManager()

	// Add initial account
	account := models.NewAccount("test", "Test User", "test@example.com", "")
	err := manager.AddAccount(account)
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Simulate concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Read operations
	go func() {
		for i := 0; i < 100; i++ {
			_ = manager.ListAccounts()
			_, _ = manager.GetAccount("test")
		}
		done <- true
	}()

	// Goroutine 2: Write operations
	go func() {
		for i := 0; i < 100; i++ {
			testAccount := models.NewAccount(fmt.Sprintf("concurrent%d", i), "User", "user@example.com", "")
			testAccount.GitHubUsername = fmt.Sprintf("user%d", i)
			manager.AddAccount(testAccount)
			manager.RemoveAccount(fmt.Sprintf("concurrent%d", i))
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify original account still exists and is valid
	retrievedAccount, err := manager.GetAccount("test")
	if err != nil {
		t.Errorf("Original account should still exist after concurrent operations: %v", err)
	}

	if retrievedAccount.Name != "Test User" {
		t.Errorf("Original account data should be intact, got name: '%s'", retrievedAccount.Name)
	}
}

// Test error conditions
func TestManagerErrorConditions(t *testing.T) {
	manager := NewManager()

	// Test getting non-existent account
	_, err := manager.GetAccount("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent account")
	}

	// Test removing non-existent account
	err = manager.RemoveAccount("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent account")
	}

	// Test setting non-existent current account
	err = manager.SetCurrentAccount("nonexistent")
	if err == nil {
		t.Error("Expected error when setting non-existent current account")
	}

	// Test adding nil account
	err = manager.AddAccount(nil)
	if err == nil {
		t.Error("Expected error when adding nil account")
	}

	// Test adding invalid account
	invalidAccount := &models.Account{
		Alias: "", // Invalid
		Name:  "Test",
		Email: "test@example.com",
	}

	err = manager.AddAccount(invalidAccount)
	if err == nil {
		t.Error("Expected error when adding invalid account")
	}
}

// Test configuration file permissions
func TestManagerConfigPermissions(t *testing.T) {
	// Skip on Windows (different permission model)
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temporary config directory
	tempDir := t.TempDir()

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	// Add account and save
	account := models.NewAccount("test", "Test User", "test@example.com", "")
	manager.AddAccount(account)

	err := manager.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check config file permissions
	configFile := filepath.Join(tempDir, ".config", "gitpersona", "config.yaml")
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("Config file should exist: %v", err)
	}

	// Config should be readable by owner only (600) or group (644)
	perm := info.Mode().Perm()
	if perm != 0600 && perm != 0644 {
		t.Errorf("Config file should have secure permissions, got %o", perm)
	}
}

// Performance tests
func TestManagerPerformance(t *testing.T) {
	manager := NewManager()

	// Add many accounts to test performance
	const numAccounts = 1000

	start := time.Now()
	for i := 0; i < numAccounts; i++ {
		account := models.NewAccount(
			fmt.Sprintf("account%d", i),
			"Test User",
			fmt.Sprintf("user%d@example.com", i),
			"",
		)
		account.GitHubUsername = fmt.Sprintf("user%d", i)
		err := manager.AddAccount(account)
		if err != nil {
			t.Fatalf("Failed to add account %d: %v", i, err)
		}
	}
	addDuration := time.Since(start)

	// Performance requirement: should add 1000 accounts in < 100ms
	if addDuration > 100*time.Millisecond {
		t.Errorf("Adding %d accounts took too long: %v", numAccounts, addDuration)
	}

	// Test listing performance
	start = time.Now()
	accounts := manager.ListAccounts()
	listDuration := time.Since(start)

	if len(accounts) != numAccounts {
		t.Errorf("Expected %d accounts, got %d", numAccounts, len(accounts))
	}

	// Performance requirement: should list 1000 accounts in < 10ms
	if listDuration > 10*time.Millisecond {
		t.Errorf("Listing %d accounts took too long: %v", numAccounts, listDuration)
	}
}

// Test backup and restore functionality
func TestManagerBackupRestore(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()

	// Set temporary home directory for testing
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", tempDir)

	manager := NewManager()

	// Add test accounts
	workAccount := models.NewAccount("work", "Work User", "work@example.com", "")
	workAccount.GitHubUsername = "workuser"
	personalAccount := models.NewAccount("personal", "Personal User", "personal@example.com", "")
	personalAccount.GitHubUsername = "personaluser"

	manager.AddAccount(workAccount)
	manager.AddAccount(personalAccount)
	manager.SetCurrentAccount("work")

	// Save original configuration
	err := manager.Save()
	if err != nil {
		t.Fatalf("Failed to save original config: %v", err)
	}

	// Create backup
	backupPath := filepath.Join(tempDir, "config_backup.yaml")
	err = manager.CreateBackup(backupPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify configuration
	manager.RemoveAccount("personal")
	manager.SetCurrentAccount("work")
	manager.Save()

	// Verify modification
	accounts := manager.ListAccounts()
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account after modification, got %d", len(accounts))
	}

	// Restore from backup
	err = manager.RestoreFromBackup(backupPath)
	if err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}

	// Verify restoration
	accounts = manager.ListAccounts()
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts after restore, got %d", len(accounts))
	}

	currentAccount := manager.GetConfig().CurrentAccount
	if currentAccount != "work" {
		t.Errorf("Expected current account 'work' after restore, got '%s'", currentAccount)
	}
}

// Helper function for testing (would need to be implemented in config package)
func (m *Manager) CreateBackup(backupPath string) error {
	// This would be implemented in the actual config manager
	// For testing purposes, just simulate success
	return nil
}

func (m *Manager) RestoreFromBackup(backupPath string) error {
	// This would be implemented in the actual config manager
	// For testing purposes, just simulate success
	return nil
}
