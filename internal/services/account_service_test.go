package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// MockConfigurationService for testing
type MockConfigurationService struct {
	accounts          map[string]*models.Account
	currentAccount    string
	getAccountErr     error
	setAccountErr     error
	deleteAccountErr  error
	listAccountsErr   error
	setCurrentErr     error
	loadErr           error
	saveErr           error
	reloadErr         error
	validateErr       error
	validateConfigErr error
	checkConflictsErr error
}

func NewMockConfigurationService() *MockConfigurationService {
	return &MockConfigurationService{
		accounts: make(map[string]*models.Account),
	}
}

func (m *MockConfigurationService) Load(ctx context.Context) error {
	return m.loadErr
}

func (m *MockConfigurationService) Save(ctx context.Context) error {
	return m.saveErr
}

func (m *MockConfigurationService) Reload(ctx context.Context) error {
	return m.reloadErr
}

func (m *MockConfigurationService) Validate(ctx context.Context) error {
	return m.validateErr
}

func (m *MockConfigurationService) GetAccount(ctx context.Context, alias string) (*models.Account, error) {
	if m.getAccountErr != nil {
		return nil, m.getAccountErr
	}
	if account, exists := m.accounts[alias]; exists {
		return account, nil
	}
	return nil, errors.New("account not found")
}

func (m *MockConfigurationService) SetAccount(ctx context.Context, account *models.Account) error {
	if m.setAccountErr != nil {
		return m.setAccountErr
	}
	m.accounts[account.Alias] = account
	return nil
}

func (m *MockConfigurationService) DeleteAccount(ctx context.Context, alias string) error {
	if m.deleteAccountErr != nil {
		return m.deleteAccountErr
	}
	delete(m.accounts, alias)
	return nil
}

func (m *MockConfigurationService) ListAccounts(ctx context.Context) ([]*models.Account, error) {
	if m.listAccountsErr != nil {
		return nil, m.listAccountsErr
	}
	accounts := make([]*models.Account, 0, len(m.accounts))
	for _, account := range m.accounts {
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (m *MockConfigurationService) GetCurrentAccount(ctx context.Context) string {
	return m.currentAccount
}

func (m *MockConfigurationService) SetCurrentAccount(ctx context.Context, alias string) error {
	if m.setCurrentErr != nil {
		return m.setCurrentErr
	}
	m.currentAccount = alias
	return nil
}

func (m *MockConfigurationService) ValidateConfiguration(ctx context.Context) error {
	return m.validateConfigErr
}

func (m *MockConfigurationService) CheckForConflicts(ctx context.Context) ([]*ConfigConflict, error) {
	if m.checkConflictsErr != nil {
		return nil, m.checkConflictsErr
	}
	return []*ConfigConflict{}, nil
}

func TestNewRealAccountService(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()

	service := NewRealAccountService(mockConfig, logger)
	if service == nil {
		t.Fatal("NewRealAccountService should return non-nil service")
	}
	if service.configService != mockConfig {
		t.Error("configService should be set correctly")
	}
	if service.logger != logger {
		t.Error("logger should be set correctly")
	}
}

func TestRealAccountService_GetAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful get
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}
	mockConfig.accounts["test"] = account

	retrieved, err := service.GetAccount(ctx, "test")
	if err != nil {
		t.Errorf("GetAccount should not return error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetAccount should return non-nil account")
	}
	if retrieved.Alias != "test" {
		t.Errorf("Expected alias 'test', got %q", retrieved.Alias)
	}

	// Test account not found
	_, err = service.GetAccount(ctx, "nonexistent")
	if err == nil {
		t.Error("GetAccount should return error for nonexistent account")
	}

	// Test config service error
	mockConfig.getAccountErr = errors.New("config error")
	_, err = service.GetAccount(ctx, "test")
	if err == nil {
		t.Error("GetAccount should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	_, err = service.GetAccount(ctx, "test")
	if err == nil {
		t.Error("GetAccount should return error when config service is nil")
	}
}

func TestRealAccountService_CreateAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful creation
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := service.CreateAccount(ctx, account)
	if err != nil {
		t.Errorf("CreateAccount should not return error: %v", err)
	}

	// Verify account was created
	if _, exists := mockConfig.accounts["test"]; !exists {
		t.Error("Account should be created in config service")
	}

	// Verify account has required fields set
	createdAccount := mockConfig.accounts["test"]
	if createdAccount.Status != models.AccountStatusActive {
		t.Errorf("Expected status %v, got %v", models.AccountStatusActive, createdAccount.Status)
	}
	if createdAccount.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	// Test validation failure
	invalidAccount := &models.Account{
		Alias: "", // Invalid: empty alias
		Name:  "Test User",
		Email: "test@example.com",
	}

	err = service.CreateAccount(ctx, invalidAccount)
	if err == nil {
		t.Error("CreateAccount should return error for invalid account")
	}

	// Test config service error
	validAccount := &models.Account{
		Alias: "test2",
		Name:  "Test User 2",
		Email: "test2@example.com",
	}
	mockConfig.setAccountErr = errors.New("config error")

	err = service.CreateAccount(ctx, validAccount)
	if err == nil {
		t.Error("CreateAccount should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	err = service.CreateAccount(ctx, validAccount)
	if err == nil {
		t.Error("CreateAccount should return error when config service is nil")
	}
}

func TestRealAccountService_UpdateAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful update
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}
	mockConfig.accounts["test"] = account

	account.Name = "Updated Name"
	err := service.UpdateAccount(ctx, account)
	if err != nil {
		t.Errorf("UpdateAccount should not return error: %v", err)
	}

	// Verify account was updated
	updatedAccount := mockConfig.accounts["test"]
	if updatedAccount.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %q", updatedAccount.Name)
	}

	// Test validation failure
	invalidAccount := &models.Account{
		Alias: "", // Invalid: empty alias
		Name:  "Test User",
		Email: "test@example.com",
	}

	err = service.UpdateAccount(ctx, invalidAccount)
	if err == nil {
		t.Error("UpdateAccount should return error for invalid account")
	}

	// Test config service error
	mockConfig.setAccountErr = errors.New("config error")
	err = service.UpdateAccount(ctx, account)
	if err == nil {
		t.Error("UpdateAccount should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	err = service.UpdateAccount(ctx, account)
	if err == nil {
		t.Error("UpdateAccount should return error when config service is nil")
	}
}

func TestRealAccountService_DeleteAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful deletion
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}
	mockConfig.accounts["test"] = account

	err := service.DeleteAccount(ctx, "test")
	if err != nil {
		t.Errorf("DeleteAccount should not return error: %v", err)
	}

	// Verify account was deleted
	if _, exists := mockConfig.accounts["test"]; exists {
		t.Error("Account should be deleted from config service")
	}

	// Test deletion of current account
	mockConfig.accounts["test"] = account
	mockConfig.currentAccount = "test"

	err = service.DeleteAccount(ctx, "test")
	if err != nil {
		t.Errorf("DeleteAccount should not return error even for current account: %v", err)
	}

	// Test config service error
	mockConfig.accounts["test"] = account
	mockConfig.deleteAccountErr = errors.New("config error")

	err = service.DeleteAccount(ctx, "test")
	if err == nil {
		t.Error("DeleteAccount should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	err = service.DeleteAccount(ctx, "test")
	if err == nil {
		t.Error("DeleteAccount should return error when config service is nil")
	}
}

func TestRealAccountService_ListAccounts(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful listing
	account1 := &models.Account{
		Alias: "test1",
		Name:  "Test User 1",
		Email: "test1@example.com",
	}
	account2 := &models.Account{
		Alias: "test2",
		Name:  "Test User 2",
		Email: "test2@example.com",
	}
	mockConfig.accounts["test1"] = account1
	mockConfig.accounts["test2"] = account2

	accounts, err := service.ListAccounts(ctx)
	if err != nil {
		t.Errorf("ListAccounts should not return error: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	// Test config service error
	mockConfig.listAccountsErr = errors.New("config error")
	_, err = service.ListAccounts(ctx)
	if err == nil {
		t.Error("ListAccounts should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	_, err = service.ListAccounts(ctx)
	if err == nil {
		t.Error("ListAccounts should return error when config service is nil")
	}
}

func TestRealAccountService_SetCurrentAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful setting
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}
	mockConfig.accounts["test"] = account

	err := service.SetCurrentAccount(ctx, "test")
	if err != nil {
		t.Errorf("SetCurrentAccount should not return error: %v", err)
	}

	// Verify current account was set
	if mockConfig.currentAccount != "test" {
		t.Errorf("Expected current account 'test', got %q", mockConfig.currentAccount)
	}

	// Test setting nonexistent account
	err = service.SetCurrentAccount(ctx, "nonexistent")
	if err == nil {
		t.Error("SetCurrentAccount should return error for nonexistent account")
	}

	// Test config service error
	mockConfig.setCurrentErr = errors.New("config error")
	err = service.SetCurrentAccount(ctx, "test")
	if err == nil {
		t.Error("SetCurrentAccount should return error when config service fails")
	}

	// Test nil config service
	service.configService = nil
	err = service.SetCurrentAccount(ctx, "test")
	if err == nil {
		t.Error("SetCurrentAccount should return error when config service is nil")
	}
}

func TestRealAccountService_GetCurrentAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful get
	mockConfig.currentAccount = "test"

	current := service.GetCurrentAccount(ctx)
	if current != "test" {
		t.Errorf("Expected current account 'test', got %q", current)
	}

	// Test nil config service
	service.configService = nil
	current = service.GetCurrentAccount(ctx)
	if current != "" {
		t.Errorf("Expected empty string for nil config service, got %q", current)
	}
}

func TestRealAccountService_MarkAccountAsUsed(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful marking
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}
	mockConfig.accounts["test"] = account

	err := service.MarkAccountAsUsed(ctx, "test")
	if err != nil {
		t.Errorf("MarkAccountAsUsed should not return error: %v", err)
	}

	// Verify last used was set
	updatedAccount := mockConfig.accounts["test"]
	if updatedAccount.LastUsed == nil {
		t.Error("LastUsed should be set")
	}
	if updatedAccount.LastUsed.IsZero() {
		t.Error("LastUsed should not be zero")
	}

	// Test marking nonexistent account
	err = service.MarkAccountAsUsed(ctx, "nonexistent")
	if err == nil {
		t.Error("MarkAccountAsUsed should return error for nonexistent account")
	}
}

func TestRealAccountService_ValidateAccount(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful validation
	account := &models.Account{
		Alias: "test",
		Name:  "Test User",
		Email: "test@example.com",
	}

	err := service.ValidateAccount(ctx, account)
	if err != nil {
		t.Errorf("ValidateAccount should not return error: %v", err)
	}

	// Test validation failure
	invalidAccount := &models.Account{
		Alias: "", // Invalid: empty alias
		Name:  "Test User",
		Email: "test@example.com",
	}

	err = service.ValidateAccount(ctx, invalidAccount)
	if err == nil {
		t.Error("ValidateAccount should return error for invalid account")
	}
}

func TestRealAccountService_TestAccountSSH(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test successful SSH test
	account := &models.Account{
		Alias:      "test",
		Name:       "Test User",
		Email:      "test@example.com",
		SSHKeyPath: "/path/to/key",
	}

	err := service.TestAccountSSH(ctx, account)
	if err != nil {
		t.Errorf("TestAccountSSH should not return error: %v", err)
	}
}

func TestRealAccountService_EdgeCases(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test with nil account
	err := service.CreateAccount(ctx, nil)
	if err == nil {
		t.Error("CreateAccount should return error for nil account")
	}

	err = service.UpdateAccount(ctx, nil)
	if err == nil {
		t.Error("UpdateAccount should return error for nil account")
	}

	err = service.ValidateAccount(ctx, nil)
	if err == nil {
		t.Error("ValidateAccount should return error for nil account")
	}

	err = service.TestAccountSSH(ctx, nil)
	if err == nil {
		t.Error("TestAccountSSH should return error for nil account")
	}
}

func TestRealAccountService_Concurrency(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test concurrent operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()

			account := &models.Account{
				Alias: "test" + string(rune(i)),
				Name:  "Test User",
				Email: "test@example.com",
			}

			_ = service.CreateAccount(ctx, account)
			_, _ = service.GetAccount(ctx, account.Alias)
			_, _ = service.ListAccounts(ctx)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify accounts were created
	accounts, err := service.ListAccounts(ctx)
	if err != nil {
		t.Errorf("ListAccounts should not return error: %v", err)
	}
	if len(accounts) != 10 {
		t.Errorf("Expected 10 accounts, got %d", len(accounts))
	}
}

func TestRealAccountService_Performance(t *testing.T) {
	logger := observability.NewDefaultLogger()
	mockConfig := NewMockConfigurationService()
	service := NewRealAccountService(mockConfig, logger)

	ctx := context.Background()

	// Test performance with many accounts
	numAccounts := 100
	start := time.Now()

	for i := 0; i < numAccounts; i++ {
		account := &models.Account{
			Alias: "test" + string(rune(i)),
			Name:  "Test User",
			Email: "test@example.com",
		}
		_ = service.CreateAccount(ctx, account)
	}

	createDuration := time.Since(start)

	// Performance requirement: should create 100 accounts in < 1s
	if createDuration > time.Second {
		t.Errorf("Creating %d accounts took too long: %v", numAccounts, createDuration)
	}

	// Test listing performance
	start = time.Now()
	_, err := service.ListAccounts(ctx)
	if err != nil {
		t.Errorf("ListAccounts should not return error: %v", err)
	}

	listDuration := time.Since(start)

	// Performance requirement: should list 100 accounts in < 100ms
	if listDuration > 100*time.Millisecond {
		t.Errorf("Listing %d accounts took too long: %v", numAccounts, listDuration)
	}
}
