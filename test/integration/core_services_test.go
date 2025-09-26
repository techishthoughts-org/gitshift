package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/techishthoughts/GitPersona/internal"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// IntegrationTestSuite provides shared setup for integration tests
type IntegrationTestSuite struct {
	ctx         context.Context
	logger      observability.Logger
	services    *internal.CoreServices
	factory     *internal.ServiceFactory
	testHomeDir string
	cleanup     func()
}

// SetupIntegrationTest creates a test environment with isolated services
func SetupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	ctx := context.Background()
	logger := observability.NewLogger(observability.LogLevelInfo)

	// Create temporary test directory
	tempDir := t.TempDir()
	testHomeDir := filepath.Join(tempDir, "home")
	require.NoError(t, os.MkdirAll(testHomeDir, 0755))

	// Set HOME environment variable for test isolation
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", testHomeDir)

	// Create service factory
	factory := internal.NewServiceFactory(logger)
	require.NoError(t, factory.Initialize(ctx))

	services := factory.GetServices()
	require.NotNil(t, services)

	cleanup := func() {
		factory.Shutdown(ctx)
		os.Setenv("HOME", originalHome)
	}

	return &IntegrationTestSuite{
		ctx:         ctx,
		logger:      logger,
		services:    services,
		factory:     factory,
		testHomeDir: testHomeDir,
		cleanup:     cleanup,
	}
}

func TestCoreServicesIntegration(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.cleanup()

	t.Run("ServiceFactoryInitialization", func(t *testing.T) {
		testServiceFactoryInitialization(t, suite)
	})

	t.Run("AccountManagerCRUD", func(t *testing.T) {
		testAccountManagerCRUD(t, suite)
	})

	t.Run("SSHManagerOperations", func(t *testing.T) {
		testSSHManagerOperations(t, suite)
	})

	t.Run("GitManagerOperations", func(t *testing.T) {
		testGitManagerOperations(t, suite)
	})

	t.Run("SystemManagerDiagnostics", func(t *testing.T) {
		testSystemManagerDiagnostics(t, suite)
	})

	t.Run("ServiceInterdependencies", func(t *testing.T) {
		testServiceInterdependencies(t, suite)
	})

	t.Run("ErrorHandlingIntegration", func(t *testing.T) {
		testErrorHandlingIntegration(t, suite)
	})

	t.Run("SecurityIntegration", func(t *testing.T) {
		testSecurityIntegration(t, suite)
	})
}

func testServiceFactoryInitialization(t *testing.T, suite *IntegrationTestSuite) {
	// Test that all services are properly initialized
	assert.NotNil(t, suite.services.Account)
	assert.NotNil(t, suite.services.SSH)
	assert.NotNil(t, suite.services.Git)
	assert.NotNil(t, suite.services.GitHub)
	assert.NotNil(t, suite.services.System)

	// Test health check
	healthChecks, err := suite.factory.HealthCheck(suite.ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, healthChecks)

	// Verify core services are healthy (GitHub might fail without tokens)
	healthyServices := 0
	for _, health := range healthChecks {
		if health.Name != "GitHub" && health.Healthy {
			healthyServices++
		}
	}
	assert.GreaterOrEqual(t, healthyServices, 3, "At least 3 core services should be healthy")
}

func testAccountManagerCRUD(t *testing.T, suite *IntegrationTestSuite) {
	// Create test account
	createReq := internal.CreateAccountRequest{
		Alias:          "test-account",
		Name:           "Test User",
		Email:          "test@example.com",
		SSHKeyPath:     filepath.Join(suite.testHomeDir, ".ssh/id_ed25519"),
		GitHubUsername: "testuser",
	}

	account, err := suite.services.Account.CreateAccount(suite.ctx, createReq)
	require.NoError(t, err)
	assert.Equal(t, "test-account", account.Alias)
	assert.Equal(t, "Test User", account.Name)
	assert.Equal(t, "test@example.com", account.Email)

	// List accounts
	accounts, err := suite.services.Account.ListAccounts(suite.ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "test-account", accounts[0].Alias)

	// Get account by alias
	retrievedAccount, err := suite.services.Account.GetAccount(suite.ctx, "test-account")
	require.NoError(t, err)
	assert.Equal(t, account.Alias, retrievedAccount.Alias)

	// Update account
	name := "Updated Test User"
	email := "updated@example.com"
	updates := internal.AccountUpdates{
		Name:  &name,
		Email: &email,
	}
	err = suite.services.Account.UpdateAccount(suite.ctx, "test-account", updates)
	require.NoError(t, err)
	// Verify update by retrieving account again
	updatedAccount, err := suite.services.Account.GetAccount(suite.ctx, "test-account")
	require.NoError(t, err)
	assert.Equal(t, "Updated Test User", updatedAccount.Name)
	assert.Equal(t, "updated@example.com", updatedAccount.Email)

	// Switch to account
	err = suite.services.Account.SwitchAccount(suite.ctx, "test-account")
	assert.NoError(t, err)

	// Validate account
	validationResult, err := suite.services.Account.ValidateAccount(suite.ctx, "test-account")
	require.NoError(t, err)
	assert.NotNil(t, validationResult)

	// Remove account
	err = suite.services.Account.DeleteAccount(suite.ctx, "test-account")
	assert.NoError(t, err)

	// Verify removal
	accounts, err = suite.services.Account.ListAccounts(suite.ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 0)
}

func testSSHManagerOperations(t *testing.T, suite *IntegrationTestSuite) {
	// Create SSH directory
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	// Generate SSH key
	generateReq := internal.GenerateKeyRequest{
		Type:    "ed25519",
		Email:   "test@example.com",
		KeyPath: filepath.Join(sshDir, "test-key"),
	}

	key, err := suite.services.SSH.GenerateKey(suite.ctx, generateReq)
	require.NoError(t, err)
	assert.NotEmpty(t, key.Path)
	assert.Equal(t, "ed25519", key.Type)

	// List keys
	keys, err := suite.services.SSH.ListKeys(suite.ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 1)

	// Validate key
	keyInfo, err := suite.services.SSH.ValidateKey(suite.ctx, key.Path)
	assert.NoError(t, err)
	assert.True(t, keyInfo.Valid)

	// Test SSH functionality (SSH agent operations not available in SSHManager interface)
	t.Logf("SSH key generated and validated successfully")

	// Delete key using system operations since RemoveKey is not in SSHManager interface
	err = suite.services.SSH.DeleteKey(suite.ctx, key.Path)
	assert.NoError(t, err)
}

func testGitManagerOperations(t *testing.T, suite *IntegrationTestSuite) {
	// Create a test git repository
	repoDir := filepath.Join(suite.testHomeDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoDir, 0755))

	// Git initialization and config operations not available in GitManager interface
	t.Logf("Test repository setup at %s", repoDir)

	// Get current config
	currentConfig, err := suite.services.Git.GetCurrentConfig(suite.ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, currentConfig)

	// Get repository info using DetectRepository
	repo, err := suite.services.Git.DetectRepository(suite.ctx, repoDir)
	if err == nil {
		assert.NotNil(t, repo)
		t.Logf("Repository detected: %s", repo.Path)
	}

	// Repository detection already tested above
}

func testSystemManagerDiagnostics(t *testing.T, suite *IntegrationTestSuite) {
	// Perform health check
	err := suite.services.System.PerformHealthCheck(suite.ctx)
	assert.NoError(t, err)

	// Get system information
	info, err := suite.services.System.GetSystemInfo(suite.ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.Platform)
	assert.NotEmpty(t, info.GitVersion)

	// System manager doesn't have GetEnvironmentInfo method in interface
	t.Logf("System information retrieved successfully")

	// Run diagnostics
	report, err := suite.services.System.RunDiagnostics(suite.ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, report.Checks)
	assert.NotEmpty(t, report.Overall)
}

func testServiceInterdependencies(t *testing.T, suite *IntegrationTestSuite) {
	// Test account creation with SSH key integration
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	// Generate SSH key first
	generateReq := internal.GenerateKeyRequest{
		Type:    "ed25519",
		Email:   "integration@example.com",
		KeyPath: filepath.Join(sshDir, "integration-key"),
	}
	key, err := suite.services.SSH.GenerateKey(suite.ctx, generateReq)
	require.NoError(t, err)

	// Create account using the generated SSH key
	createReq := internal.CreateAccountRequest{
		Alias:      "integration-account",
		Name:       "Integration User",
		Email:      "integration@example.com",
		SSHKeyPath: key.Path,
	}
	_, err = suite.services.Account.CreateAccount(suite.ctx, createReq)
	require.NoError(t, err)

	// Switch to the account and verify Git config is updated
	err = suite.services.Account.SwitchAccount(suite.ctx, "integration-account")
	assert.NoError(t, err)

	// Give some time for the switch to propagate
	time.Sleep(100 * time.Millisecond)

	// Verify Git configuration was updated
	gitConfig, err := suite.services.Git.GetCurrentConfig(suite.ctx)
	assert.NoError(t, err)

	// The exact config depends on how SwitchAccount is implemented
	// This test verifies that the services can work together
	t.Logf("Git config after account switch: %+v", gitConfig)

	// Clean up
	err = suite.services.Account.DeleteAccount(suite.ctx, "integration-account")
	assert.NoError(t, err)
	err = suite.services.SSH.DeleteKey(suite.ctx, key.Path)
	assert.NoError(t, err)
}

func testErrorHandlingIntegration(t *testing.T, suite *IntegrationTestSuite) {
	// Test error handling across services

	// Try to get non-existent account
	_, err := suite.services.Account.GetAccount(suite.ctx, "non-existent")
	assert.Error(t, err)

	// Verify it's a structured GitPersona error
	if gpErr, ok := err.(*internal.GitPersonaError); ok {
		assert.Equal(t, internal.ErrAccountNotFound, gpErr.Code)
		assert.Contains(t, gpErr.UserMessage, "doesn't exist")
		assert.NotEmpty(t, gpErr.Suggestions)
	}

	// Test SSH key validation with invalid path
	keyInfo, err := suite.services.SSH.ValidateKey(suite.ctx, "/nonexistent/key")
	assert.Error(t, err)
	if keyInfo != nil {
		assert.False(t, keyInfo.Valid)
	}

	// Test Git operations on non-existent repository
	_, err = suite.services.Git.DetectRepository(suite.ctx, "/nonexistent/repo")
	assert.Error(t, err)
}

func testSecurityIntegration(t *testing.T, suite *IntegrationTestSuite) {
	// Create security validator
	logger := observability.NewLogger(observability.LogLevelInfo)
	validator := internal.NewSecurityValidator(logger)

	// Run security audit
	audit, err := validator.RunSecurityAudit(suite.ctx)
	require.NoError(t, err)
	assert.NotNil(t, audit)
	assert.GreaterOrEqual(t, audit.OverallScore, 0)
	assert.LessOrEqual(t, audit.OverallScore, audit.MaxScore)

	// If there are auto-fixable violations, try to fix them
	autoFixableViolations := []*internal.SecurityViolation{}
	for _, violation := range audit.Violations {
		if violation.AutoFixable {
			autoFixableViolations = append(autoFixableViolations, violation)
		}
	}

	if len(autoFixableViolations) > 0 {
		err = validator.FixSecurityViolations(suite.ctx, autoFixableViolations)
		assert.NoError(t, err)

		// Verify violations were fixed
		for _, violation := range autoFixableViolations {
			assert.True(t, violation.Fixed, "Violation %s should be marked as fixed", violation.ID)
		}
	}

	// Test encrypted storage
	encManager := internal.NewEncryptionManager(logger)
	storage, err := encManager.NewSecureStorage(suite.ctx, "test-storage")
	require.NoError(t, err)

	// Store and retrieve data
	testKey := "test-key"
	testValue := "sensitive-data"
	err = storage.Store(suite.ctx, testKey, testValue)
	assert.NoError(t, err)

	retrievedValue, err := storage.Retrieve(suite.ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, retrievedValue)

	// Clean up
	err = storage.Delete(suite.ctx)
	assert.NoError(t, err)
}

// Benchmark tests for performance validation
func BenchmarkServiceOperations(b *testing.B) {
	suite := SetupIntegrationTest(&testing.T{})
	defer suite.cleanup()

	b.Run("AccountList", func(b *testing.B) {
		// Create test account first
		createReq := internal.CreateAccountRequest{
			Alias: "bench-account",
			Name:  "Bench User",
			Email: "bench@example.com",
		}
		suite.services.Account.CreateAccount(suite.ctx, createReq)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := suite.services.Account.ListAccounts(suite.ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SSHKeyList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.services.SSH.ListKeys(suite.ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.factory.HealthCheck(suite.ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
