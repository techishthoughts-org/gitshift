package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/techishthoughts/GitPersona/internal"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"gopkg.in/yaml.v3"
)

func TestMigrationIntegration(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.cleanup()

	t.Run("LegacyConfigDetection", func(t *testing.T) {
		testLegacyConfigDetection(t, suite)
	})

	t.Run("LegacyToV2Migration", func(t *testing.T) {
		testLegacyToV2Migration(t, suite)
	})

	t.Run("V1ToV2Migration", func(t *testing.T) {
		testV1ToV2Migration(t, suite)
	})

	t.Run("BackupCreation", func(t *testing.T) {
		testBackupCreation(t, suite)
	})

	t.Run("AccountDiscovery", func(t *testing.T) {
		testAccountDiscovery(t, suite)
	})
}

func testLegacyConfigDetection(t *testing.T, suite *IntegrationTestSuite) {
	logger := observability.NewLogger(observability.LogLevelInfo)
	migrationManager := internal.NewMigrationManager(logger)

	// Test with no config - should detect no migration needed
	needsMigration, version, err := migrationManager.DetectMigrationNeeded(suite.ctx)
	assert.NoError(t, err)
	assert.False(t, needsMigration)
	assert.Empty(t, version)

	// Create legacy config file
	legacyPath := filepath.Join(suite.testHomeDir, ".git-persona.yaml")
	legacyConfig := map[string]interface{}{
		"current_account": "work",
		"accounts": map[string]interface{}{
			"work": map[string]interface{}{
				"name":            "John Doe",
				"email":           "john@work.com",
				"ssh_key_path":    "~/.ssh/id_ed25519_work",
				"github_username": "johndoe",
			},
		},
		"global_git_mode": true,
	}

	data, err := yaml.Marshal(legacyConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(legacyPath, data, 0644))

	// Should now detect migration needed
	needsMigration, version, err = migrationManager.DetectMigrationNeeded(suite.ctx)
	assert.NoError(t, err)
	assert.True(t, needsMigration)
	assert.Equal(t, "legacy", version)
}

func testLegacyToV2Migration(t *testing.T, suite *IntegrationTestSuite) {
	logger := observability.NewLogger(observability.LogLevelInfo)
	migrationManager := internal.NewMigrationManager(logger)

	// Create SSH directory with test keys
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	// Create fake SSH keys for discovery
	testKeys := []string{"id_ed25519_work", "id_ed25519_personal", "id_rsa"}
	for _, keyName := range testKeys {
		keyPath := filepath.Join(sshDir, keyName)
		require.NoError(t, os.WriteFile(keyPath, []byte("fake-ssh-key"), 0600))
	}

	// Run migration from legacy
	result, err := migrationManager.RunMigration(suite.ctx, "legacy", "2.0.0")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "legacy", result.FromVersion)
	assert.Equal(t, "2.0.0", result.ToVersion)
	assert.NotEmpty(t, result.BackupPath)
	assert.Greater(t, result.MigratedAccounts, 0)

	// Verify migrated configuration exists
	configDir := filepath.Join(suite.testHomeDir, ".gitpersona")
	configFile := filepath.Join(configDir, "config.yaml")
	assert.FileExists(t, configFile)

	// Load and verify config structure
	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	assert.Equal(t, "2.0.0", config["config_version"])
	assert.NotNil(t, config["accounts"])

	accounts, ok := config["accounts"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, accounts)

	// Check if accounts have isolation metadata
	for accountName, accountData := range accounts {
		accountMap, ok := accountData.(map[string]interface{})
		assert.True(t, ok, "Account %s should be a map", accountName)

		// Check for v2 features
		assert.Contains(t, accountMap, "isolation_level")
		assert.Contains(t, accountMap, "isolation_metadata")

		isolationMetadata, ok := accountMap["isolation_metadata"].(map[string]interface{})
		if ok {
			assert.Contains(t, isolationMetadata, "ssh_isolation")
			assert.Contains(t, isolationMetadata, "token_isolation")
			assert.Contains(t, isolationMetadata, "git_isolation")
		}
	}
}

func testV1ToV2Migration(t *testing.T, suite *IntegrationTestSuite) {
	logger := observability.NewLogger(observability.LogLevelInfo)
	migrationManager := internal.NewMigrationManager(logger)

	// Create v1 config
	configDir := filepath.Join(suite.testHomeDir, ".gitpersona")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	v1Config := map[string]interface{}{
		"config_version":    "1.5.0",
		"current_account":   "test",
		"global_git_config": true,
		"auto_detect":       false,
		"accounts": map[string]interface{}{
			"test": map[string]interface{}{
				"alias":           "test",
				"name":            "Test User",
				"email":           "test@example.com",
				"ssh_key_path":    "~/.ssh/id_ed25519",
				"github_username": "testuser",
				"last_used":       "2024-01-15T10:30:00Z",
			},
		},
	}

	configFile := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(v1Config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFile, data, 0644))

	// Run migration
	result, err := migrationManager.RunMigration(suite.ctx, "1.5.0", "2.0.0")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "1.5.0", result.FromVersion)
	assert.Equal(t, "2.0.0", result.ToVersion)

	// Verify updated config
	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var updatedConfig map[string]interface{}
	err = yaml.Unmarshal(configData, &updatedConfig)
	require.NoError(t, err)

	assert.Equal(t, "2.0.0", updatedConfig["config_version"])

	accounts, ok := updatedConfig["accounts"].(map[string]interface{})
	require.True(t, ok)

	testAccount, ok := accounts["test"].(map[string]interface{})
	require.True(t, ok)

	// Verify v2 features were added
	assert.Contains(t, testAccount, "isolation_level")
	assert.Contains(t, testAccount, "isolation_metadata")
	assert.Contains(t, testAccount, "account_metadata")

	isolationLevel, ok := testAccount["isolation_level"].(string)
	assert.True(t, ok)
	assert.Equal(t, "standard", isolationLevel)

	isolationMetadata, ok := testAccount["isolation_metadata"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, isolationMetadata, "ssh_isolation")
	assert.Contains(t, isolationMetadata, "token_isolation")
	assert.Contains(t, isolationMetadata, "git_isolation")
	assert.Contains(t, isolationMetadata, "environment_isolation")
}

func testBackupCreation(t *testing.T, suite *IntegrationTestSuite) {
	logger := observability.NewLogger(observability.LogLevelInfo)
	migrationManager := internal.NewMigrationManager(logger)

	// Create some config to backup
	configDir := filepath.Join(suite.testHomeDir, ".gitpersona")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configFile := filepath.Join(configDir, "config.yaml")
	testConfig := `
config_version: "1.0.0"
accounts:
  test:
    name: "Test User"
    email: "test@example.com"
`
	require.NoError(t, os.WriteFile(configFile, []byte(testConfig), 0644))

	// Run migration which should create backup
	result, err := migrationManager.RunMigration(suite.ctx, "1.0.0", "2.0.0")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.BackupPath)

	// Verify backup was created (even if it's just a placeholder)
	placeholderPath := result.BackupPath + ".placeholder"
	assert.FileExists(t, placeholderPath)

	// Test backup listing
	backups, err := migrationManager.ListBackups(suite.ctx)
	assert.NoError(t, err)
	// Should find at least our placeholder
	found := false
	for _, backup := range backups {
		if filepath.Dir(backup) == filepath.Dir(result.BackupPath) {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find backup in list")
}

func testAccountDiscovery(t *testing.T, suite *IntegrationTestSuite) {
	logger := observability.NewLogger(observability.LogLevelInfo)
	migrationManager := internal.NewMigrationManager(logger)

	// Create SSH keys for discovery
	sshDir := filepath.Join(suite.testHomeDir, ".ssh")
	require.NoError(t, os.MkdirAll(sshDir, 0700))

	// Create keys with different naming patterns
	testKeys := map[string]string{
		"id_ed25519_work":     "work account key",
		"id_ed25519_personal": "personal account key",
		"id_rsa_github":       "github account key",
		"id_ed25519":          "default key",
	}

	for keyName, content := range testKeys {
		keyPath := filepath.Join(sshDir, keyName)
		require.NoError(t, os.WriteFile(keyPath, []byte(content), 0600))
	}

	// Run legacy migration which should discover accounts from SSH keys
	result, err := migrationManager.RunMigration(suite.ctx, "legacy", "2.0.0")
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify accounts were discovered and migrated
	assert.Greater(t, result.MigratedAccounts, 0)

	// Load config and verify discovered accounts
	configFile := filepath.Join(suite.testHomeDir, ".gitpersona", "config.yaml")
	configData, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var config map[string]interface{}
	err = yaml.Unmarshal(configData, &config)
	require.NoError(t, err)

	accounts, ok := config["accounts"].(map[string]interface{})
	require.True(t, ok)

	// Should have discovered accounts based on SSH key naming
	expectedAccounts := []string{"work", "personal", "github"}
	discoveredCount := 0

	for _, expectedAccount := range expectedAccounts {
		if _, exists := accounts[expectedAccount]; exists {
			discoveredCount++

			// Verify account has SSH key path set
			accountData, ok := accounts[expectedAccount].(map[string]interface{})
			require.True(t, ok)

			sshKeyPath, exists := accountData["ssh_key_path"]
			assert.True(t, exists, "Account %s should have SSH key path", expectedAccount)
			assert.NotEmpty(t, sshKeyPath)
		}
	}

	assert.Greater(t, discoveredCount, 0, "Should discover at least one account from SSH keys")
}

// Test migration with concurrent operations
func TestMigrationConcurrency(t *testing.T) {
	suite := SetupIntegrationTest(t)
	defer suite.cleanup()

	logger := observability.NewLogger(observability.LogLevelInfo)

	// Create v1 config
	configDir := filepath.Join(suite.testHomeDir, ".gitpersona")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	v1Config := map[string]interface{}{
		"config_version": "1.0.0",
		"accounts": map[string]interface{}{
			"concurrent-test": map[string]interface{}{
				"name":  "Concurrent Test",
				"email": "concurrent@example.com",
			},
		},
	}

	configFile := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(v1Config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configFile, data, 0644))

	// Run multiple migrations concurrently (should handle conflicts gracefully)
	results := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			migrationManager := internal.NewMigrationManager(logger)
			_, err := migrationManager.RunMigration(context.Background(), "1.0.0", "2.0.0")
			results <- err
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < 3; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
		// It's ok if some fail due to concurrency - at least one should succeed
		t.Logf("Migration attempt %d result: %v", i+1, err)
	}

	assert.GreaterOrEqual(t, successCount, 1, "At least one migration should succeed")

	// Verify final config is valid
	finalConfigData, err := os.ReadFile(configFile)
	require.NoError(t, err)

	var finalConfig map[string]interface{}
	err = yaml.Unmarshal(finalConfigData, &finalConfig)
	require.NoError(t, err)

	// Should have v2 config version
	assert.Equal(t, "2.0.0", finalConfig["config_version"])
}
