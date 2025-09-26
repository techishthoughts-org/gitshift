package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// TestIsolatedAccountSwitching tests complete account isolation
func TestIsolatedAccountSwitching(t *testing.T) {
	testDir := setupIsolatedTestEnvironment(t)
	defer cleanupIsolatedTestEnvironment(testDir)

	// Create test accounts with isolation metadata
	account1 := createIsolatedTestAccount("test-work", "Work User", "work@example.com", "workuser")
	account2 := createIsolatedTestAccount("test-personal", "Personal User", "personal@example.com", "personaluser")
	account3 := createIsolatedTestAccount("test-client", "Client User", "client@example.com", "clientuser")

	// Create test SSH keys and tokens
	createTestSSHKeysWithIsolation(t, testDir, account1, account2, account3)

	logger := observability.NewDefaultLogger()
	ctx := context.Background()

	t.Run("IsolatedTokenManagement", func(t *testing.T) {
		// Test isolated token service
		tokenConfig := &services.TokenIsolationConfig{
			StrictIsolation:    true,
			AutoValidation:     true,
			ValidationInterval: time.Minute,
			EncryptionEnabled:  true,
			BackupEnabled:      true,
		}

		tokenService, err := services.NewIsolatedTokenService(logger, tokenConfig)
		require.NoError(t, err)

		// Store tokens for each account
		testTokens := map[string]string{
			account1.Alias: "ghp_test_work_token_123456789",
			account2.Alias: "ghp_test_personal_token_987654321",
			account3.Alias: "ghp_test_client_token_456789123",
		}

		// Store tokens with usernames
		for alias, token := range testTokens {
			var username string
			switch alias {
			case account1.Alias:
				username = account1.GitHubUsername
			case account2.Alias:
				username = account2.GitHubUsername
			case account3.Alias:
				username = account3.GitHubUsername
			}

			err := tokenService.StoreToken(ctx, alias, token, username)
			assert.NoError(t, err, "Failed to store token for %s", alias)
		}

		// Test token retrieval and isolation
		for alias, expectedToken := range testTokens {
			retrievedToken, err := tokenService.GetToken(ctx, alias)
			assert.NoError(t, err, "Failed to retrieve token for %s", alias)
			assert.Equal(t, expectedToken, retrievedToken, "Token mismatch for %s", alias)
		}

		// Test strict isolation - account1 should not get account2's token
		_, err = tokenService.GetToken(ctx, "nonexistent-account")
		assert.Error(t, err, "Should fail to get token for nonexistent account")

		// Test token validation with username mismatch (should fail)
		err = tokenService.ValidateTokenIsolation(ctx, account1.Alias, "wrong-username")
		assert.Error(t, err, "Should fail validation with wrong username")

		// Test successful validation
		err = tokenService.ValidateTokenIsolation(ctx, account1.Alias, account1.GitHubUsername)
		assert.NoError(t, err, "Should pass validation with correct username")
	})

	t.Run("IsolatedSSHManagement", func(t *testing.T) {
		// Test isolated SSH manager
		sshConfig := &services.SSHIsolationConfig{
			StrictIsolation:     true,
			AutoCleanup:         true,
			SocketTimeout:       30 * time.Second,
			KeyLoadTimeout:      10 * time.Second,
			MaxIdleTime:         time.Hour,
			ForceIdentitiesOnly: true,
		}

		sshManager, err := services.NewIsolatedSSHManager(logger, sshConfig)
		require.NoError(t, err)

		// Test SSH switching for each account
		for _, account := range []*models.Account{account1, account2, account3} {
			t.Run(fmt.Sprintf("SSH_Isolation_%s", account.Alias), func(t *testing.T) {
				err := sshManager.SwitchToAccount(ctx, account.Alias, account.SSHKeyPath)
				assert.NoError(t, err, "Failed to switch SSH for %s", account.Alias)

				// Verify agent was created
				agent, err := sshManager.GetAccountAgent(account.Alias)
				assert.NoError(t, err, "Failed to get SSH agent for %s", account.Alias)
				assert.NotNil(t, agent)
				assert.Equal(t, account.Alias, agent.AccountAlias)
				assert.True(t, agent.IsRunning)
				assert.NotEmpty(t, agent.SocketPath)

				// Verify socket file exists
				_, err = os.Stat(agent.SocketPath)
				assert.NoError(t, err, "SSH socket file should exist for %s", account.Alias)
			})
		}

		// Test concurrent SSH isolation
		t.Run("ConcurrentSSHIsolation", func(t *testing.T) {
			var wg sync.WaitGroup
			accounts := []*models.Account{account1, account2, account3}

			for _, account := range accounts {
				wg.Add(1)
				go func(acc *models.Account) {
					defer wg.Done()

					err := sshManager.SwitchToAccount(ctx, acc.Alias, acc.SSHKeyPath)
					assert.NoError(t, err, "Concurrent SSH switch failed for %s", acc.Alias)

					agent, err := sshManager.GetAccountAgent(acc.Alias)
					assert.NoError(t, err, "Failed to get agent for %s", acc.Alias)
					assert.True(t, agent.IsRunning, "Agent should be running for %s", acc.Alias)
				}(account)
			}

			wg.Wait()

			// Verify all agents are running independently
			activeAgents, err := sshManager.ListActiveAgents(ctx)
			assert.NoError(t, err)
			assert.Len(t, activeAgents, 3, "Should have 3 active agents")

			// Verify each agent has unique socket
			socketPaths := make(map[string]bool)
			for _, agent := range activeAgents {
				assert.False(t, socketPaths[agent.SocketPath], "Socket paths should be unique")
				socketPaths[agent.SocketPath] = true
			}
		})

		// Cleanup
		err = sshManager.CleanupAllAgents(ctx)
		assert.NoError(t, err, "Failed to cleanup SSH agents")
	})

	t.Run("AtomicAccountSwitching", func(t *testing.T) {
		// Test complete atomic account switching with transaction
		tokenConfig := &services.TokenIsolationConfig{
			StrictIsolation:    true,
			AutoValidation:     true,
			ValidationInterval: time.Hour,
			EncryptionEnabled:  true,
			BackupEnabled:      true,
		}

		tokenService, err := services.NewIsolatedTokenService(logger, tokenConfig)
		require.NoError(t, err)

		sshConfig := &services.SSHIsolationConfig{
			StrictIsolation:     true,
			AutoCleanup:         true,
			SocketTimeout:       30 * time.Second,
			KeyLoadTimeout:      10 * time.Second,
			MaxIdleTime:         time.Hour,
			ForceIdentitiesOnly: true,
		}

		sshManager, err := services.NewIsolatedSSHManager(logger, sshConfig)
		require.NoError(t, err)

		// Store test tokens
		testTokens := map[string]string{
			account1.Alias: "ghp_test_work_token_123456789",
			account2.Alias: "ghp_test_personal_token_987654321",
		}

		for alias, token := range testTokens {
			var username string
			if alias == account1.Alias {
				username = account1.GitHubUsername
			} else {
				username = account2.GitHubUsername
			}
			err := tokenService.StoreToken(ctx, alias, token, username)
			require.NoError(t, err)
		}

		// Test successful atomic switch
		t.Run("SuccessfulAtomicSwitch", func(t *testing.T) {
			options := &services.TransactionOptions{
				StrictValidation:     true,
				RollbackOnFailure:    true,
				ValidateBeforeSwitch: true,
				ValidateAfterSwitch:  false, // Skip post-validation for test
				Timeout:              2 * time.Minute,
				ConcurrentSteps:      false,
				SkipSSHValidation:    true, // Skip SSH connectivity for test
				SkipTokenValidation:  false,
			}

			transaction := services.NewAccountSwitchTransaction(
				ctx, logger, nil, account1, tokenService, sshManager, options,
			)

			// Add switch steps
			transaction.AddStep(services.NewTokenIsolationStep(logger))
			transaction.AddStep(services.NewSSHIsolationStep(logger))
			transaction.AddStep(services.NewGitConfigurationStep(logger))
			transaction.AddStep(services.NewEnvironmentStep(logger))

			// Execute transaction
			result, err := transaction.Execute()
			assert.NoError(t, err, "Transaction should succeed")
			assert.True(t, result.Success, "Transaction should be successful")
			assert.Equal(t, services.TransactionStateCompleted, result.FinalState)
			assert.Len(t, result.CompletedSteps, 4, "Should complete all 4 steps")
			assert.Nil(t, result.FailedStep, "No step should fail")
			assert.Empty(t, result.RollbackSteps, "No rollback should be needed")
		})

		// Test rollback on failure
		t.Run("RollbackOnFailure", func(t *testing.T) {
			// Create account with missing SSH key to trigger failure
			failAccount := createIsolatedTestAccount("test-fail", "Fail User", "fail@example.com", "failuser")
			failAccount.SSHKeyPath = "/nonexistent/key/path"

			options := &services.TransactionOptions{
				StrictValidation:     true,
				RollbackOnFailure:    true,
				ValidateBeforeSwitch: true,
				ValidateAfterSwitch:  false,
				Timeout:              1 * time.Minute,
				ConcurrentSteps:      false,
				SkipSSHValidation:    false,
				SkipTokenValidation:  true, // Skip token validation since no token stored
			}

			transaction := services.NewAccountSwitchTransaction(
				ctx, logger, account1, failAccount, tokenService, sshManager, options,
			)

			transaction.AddStep(services.NewSSHIsolationStep(logger))
			transaction.AddStep(services.NewGitConfigurationStep(logger))

			// Execute transaction - should fail and rollback
			result, err := transaction.Execute()
			assert.Error(t, err, "Transaction should fail")
			assert.False(t, result.Success, "Transaction should not be successful")
			assert.Equal(t, services.TransactionStateRolledBack, result.FinalState)
			assert.NotNil(t, result.FailedStep, "Should have a failed step")
			assert.NotEmpty(t, result.RollbackSteps, "Should have rollback steps")
		})

		// Cleanup
		err = sshManager.CleanupAllAgents(ctx)
		assert.NoError(t, err)
	})

	t.Run("ConcurrentAccountSwitching", func(t *testing.T) {
		// Test concurrent account switching with isolation
		tokenService, err := services.NewIsolatedTokenService(logger, &services.TokenIsolationConfig{
			StrictIsolation:    true,
			AutoValidation:     true,
			ValidationInterval: time.Hour,
			EncryptionEnabled:  true,
			BackupEnabled:      true,
		})
		require.NoError(t, err)

		sshManager, err := services.NewIsolatedSSHManager(logger, &services.SSHIsolationConfig{
			StrictIsolation:     true,
			AutoCleanup:         true,
			SocketTimeout:       30 * time.Second,
			KeyLoadTimeout:      10 * time.Second,
			MaxIdleTime:         time.Hour,
			ForceIdentitiesOnly: true,
		})
		require.NoError(t, err)

		// Store tokens for all accounts
		testTokens := map[string]struct {
			token    string
			username string
		}{
			account1.Alias: {"ghp_test_work_token_123456789", account1.GitHubUsername},
			account2.Alias: {"ghp_test_personal_token_987654321", account2.GitHubUsername},
			account3.Alias: {"ghp_test_client_token_456789123", account3.GitHubUsername},
		}

		for alias, data := range testTokens {
			err := tokenService.StoreToken(ctx, alias, data.token, data.username)
			require.NoError(t, err)
		}

		// Perform concurrent switches
		var wg sync.WaitGroup
		results := make([]bool, 3)
		accounts := []*models.Account{account1, account2, account3}

		for i, account := range accounts {
			wg.Add(1)
			go func(idx int, acc *models.Account) {
				defer wg.Done()

				options := &services.TransactionOptions{
					StrictValidation:     false, // Relaxed for concurrent test
					RollbackOnFailure:    true,
					ValidateBeforeSwitch: true,
					ValidateAfterSwitch:  false,
					Timeout:              1 * time.Minute,
					ConcurrentSteps:      false,
					SkipSSHValidation:    true,
					SkipTokenValidation:  false,
				}

				transaction := services.NewAccountSwitchTransaction(
					ctx, logger, nil, acc, tokenService, sshManager, options,
				)

				transaction.AddStep(services.NewTokenIsolationStep(logger))
				transaction.AddStep(services.NewSSHIsolationStep(logger))

				result, err := transaction.Execute()
				results[idx] = (err == nil && result.Success)
			}(i, account)
		}

		wg.Wait()

		// Verify all switches succeeded
		for i, success := range results {
			assert.True(t, success, "Concurrent switch %d should succeed", i+1)
		}

		// Verify all accounts have isolated SSH agents
		activeAgents, err := sshManager.ListActiveAgents(ctx)
		assert.NoError(t, err)
		assert.Len(t, activeAgents, 3, "Should have 3 active isolated agents")

		// Cleanup
		err = sshManager.CleanupAllAgents(ctx)
		assert.NoError(t, err)
	})
}

// TestPerformanceWithIsolation tests performance of isolated switching
func TestPerformanceWithIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testDir := setupIsolatedTestEnvironment(t)
	defer cleanupIsolatedTestEnvironment(testDir)

	logger := observability.NewDefaultLogger()
	ctx := context.Background()

	// Create 10 test accounts
	accounts := make([]*models.Account, 10)
	for i := 0; i < 10; i++ {
		accounts[i] = createIsolatedTestAccount(
			fmt.Sprintf("perf-test-%d", i),
			fmt.Sprintf("Perf User %d", i),
			fmt.Sprintf("perf%d@example.com", i),
			fmt.Sprintf("perfuser%d", i),
		)
	}

	createTestSSHKeysWithIsolation(t, testDir, accounts...)

	tokenService, err := services.NewIsolatedTokenService(logger, &services.TokenIsolationConfig{
		StrictIsolation:    true,
		AutoValidation:     true,
		ValidationInterval: time.Hour,
		EncryptionEnabled:  true,
		BackupEnabled:      true,
	})
	require.NoError(t, err)

	sshManager, err := services.NewIsolatedSSHManager(logger, &services.SSHIsolationConfig{
		StrictIsolation:     true,
		AutoCleanup:         true,
		SocketTimeout:       30 * time.Second,
		KeyLoadTimeout:      10 * time.Second,
		MaxIdleTime:         time.Hour,
		ForceIdentitiesOnly: true,
	})
	require.NoError(t, err)

	// Store tokens for all accounts
	for i, account := range accounts {
		token := fmt.Sprintf("ghp_perf_test_token_%d_123456789", i)
		err := tokenService.StoreToken(ctx, account.Alias, token, account.GitHubUsername)
		require.NoError(t, err)
	}

	// Performance test: sequential switching
	t.Run("SequentialSwitchingPerformance", func(t *testing.T) {
		startTime := time.Now()

		for _, account := range accounts {
			options := &services.TransactionOptions{
				StrictValidation:     false,
				RollbackOnFailure:    true,
				ValidateBeforeSwitch: true,
				ValidateAfterSwitch:  false,
				Timeout:              30 * time.Second,
				ConcurrentSteps:      false,
				SkipSSHValidation:    true,
				SkipTokenValidation:  false,
			}

			transaction := services.NewAccountSwitchTransaction(
				ctx, logger, nil, account, tokenService, sshManager, options,
			)

			transaction.AddStep(services.NewTokenIsolationStep(logger))
			transaction.AddStep(services.NewSSHIsolationStep(logger))

			result, err := transaction.Execute()
			assert.NoError(t, err, "Switch should succeed for %s", account.Alias)
			assert.True(t, result.Success, "Transaction should be successful for %s", account.Alias)
		}

		totalDuration := time.Since(startTime)
		avgDuration := totalDuration / time.Duration(len(accounts))

		t.Logf("Sequential switching performance:")
		t.Logf("  Total accounts: %d", len(accounts))
		t.Logf("  Total duration: %v", totalDuration)
		t.Logf("  Average per account: %v", avgDuration)

		// Performance expectations (adjust based on requirements)
		maxTotalDuration := 2 * time.Minute
		maxAvgDuration := 12 * time.Second

		assert.Less(t, totalDuration, maxTotalDuration,
			"Total switching time should be less than %v", maxTotalDuration)
		assert.Less(t, avgDuration, maxAvgDuration,
			"Average switching time should be less than %v", maxAvgDuration)
	})

	// Cleanup
	err = sshManager.CleanupAllAgents(ctx)
	assert.NoError(t, err)
}

// Helper functions

func setupIsolatedTestEnvironment(t *testing.T) string {
	testDir := filepath.Join(os.TempDir(), fmt.Sprintf("gitpersona-isolated-test-%d", time.Now().UnixNano()))

	dirs := []string{
		filepath.Join(testDir, "config"),
		filepath.Join(testDir, "ssh"),
		filepath.Join(testDir, "tokens"),
		filepath.Join(testDir, "sockets"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create test directory: %s", dir)
	}

	return testDir
}

func cleanupIsolatedTestEnvironment(testDir string) {
	_ = os.RemoveAll(testDir)
}

func createIsolatedTestAccount(alias, name, email, githubUsername string) *models.Account {
	account := models.NewIsolatedAccount(alias, name, email, "", githubUsername)

	// Set test-specific SSH key path (will be created by createTestSSHKeysWithIsolation)
	account.SSHKeyPath = filepath.Join(os.TempDir(),
		fmt.Sprintf("gitpersona-isolated-test-%d", time.Now().UnixNano()),
		"ssh", fmt.Sprintf("id_ed25519_%s", alias))

	return account
}

func createTestSSHKeysWithIsolation(t *testing.T, testDir string, accounts ...*models.Account) {
	for _, account := range accounts {
		// Update SSH key path to use test directory
		account.SSHKeyPath = filepath.Join(testDir, "ssh", fmt.Sprintf("id_ed25519_%s", account.Alias))

		// Create dummy SSH key files for testing
		keyContent := fmt.Sprintf("-----BEGIN OPENSSH PRIVATE KEY-----\nTEST KEY FOR %s (%s)\n-----END OPENSSH PRIVATE KEY-----\n",
			account.Alias, account.GitHubUsername)

		err := os.WriteFile(account.SSHKeyPath, []byte(keyContent), 0600)
		require.NoError(t, err, "Failed to create test SSH key for %s", account.Alias)

		// Create corresponding public key
		pubKeyPath := account.SSHKeyPath + ".pub"
		pubKeyContent := fmt.Sprintf("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAATEST%s %s\n",
			account.Alias, account.Email)

		err = os.WriteFile(pubKeyPath, []byte(pubKeyContent), 0644)
		require.NoError(t, err, "Failed to create test SSH public key for %s", account.Alias)
	}
}
