package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealAccountManager implements the AccountManager interface
type RealAccountManager struct {
	logger        observability.Logger
	configManager *config.Manager
	sshManager    SSHManager
	gitManager    GitManager
	githubManager GitHubManager
}

// NewAccountManager creates a new account manager
func NewAccountManager(logger observability.Logger) AccountManager {
	return &RealAccountManager{
		logger:        logger,
		configManager: config.NewManager(),
	}
}

// SetDependencies injects other service dependencies
func (am *RealAccountManager) SetDependencies(ssh SSHManager, git GitManager, github GitHubManager) {
	am.sshManager = ssh
	am.gitManager = git
	am.githubManager = github
}

// CreateAccount creates a new account with validation and setup
func (am *RealAccountManager) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	am.logger.Info(ctx, "creating_account",
		observability.F("alias", req.Alias),
		observability.F("github_username", req.GitHubUsername),
	)

	// Load current configuration
	if err := am.configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate request
	if err := am.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid account request: %w", err)
	}

	// Check if account already exists
	if _, err := am.configManager.GetAccount(req.Alias); err == nil {
		return nil, fmt.Errorf("account '%s' already exists", req.Alias)
	}

	// Create the account model
	account := models.NewAccount(req.Alias, req.Name, req.Email, req.SSHKeyPath)
	account.GitHubUsername = req.GitHubUsername
	account.Description = req.Description

	// Generate SSH key if needed
	if req.SSHKeyPath == "" {
		keyPath := am.generateSSHKeyPath(req.Alias)
		if am.sshManager != nil {
			keyReq := GenerateKeyRequest{
				Type:    "ed25519",
				Email:   req.Email,
				KeyPath: keyPath,
			}
			sshKey, err := am.sshManager.GenerateKey(ctx, keyReq)
			if err != nil {
				am.logger.Error(ctx, "failed_to_generate_ssh_key",
					observability.F("alias", req.Alias),
					observability.F("error", err.Error()),
				)
				return nil, fmt.Errorf("failed to generate SSH key: %w", err)
			}
			account.SSHKeyPath = sshKey.Path
		}
	}

	// Save the account
	if err := am.configManager.AddAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	if err := am.configManager.Save(); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	// Convert to return type
	result := &Account{
		Alias:          account.Alias,
		Name:           account.Name,
		Email:          account.Email,
		GitHubUsername: account.GitHubUsername,
		SSHKeyPath:     account.SSHKeyPath,
		Description:    account.Description,
		IsActive:       account.Status == models.AccountStatusActive,
		CreatedAt:      account.CreatedAt.Format(time.RFC3339),
	}

	if account.LastUsed != nil {
		lastUsed := account.LastUsed.Format(time.RFC3339)
		result.LastUsed = &lastUsed
	}

	am.logger.Info(ctx, "account_created_successfully",
		observability.F("alias", req.Alias),
		observability.F("ssh_key_generated", req.SSHKeyPath == ""),
	)

	return result, nil
}

// GetAccount retrieves a specific account
func (am *RealAccountManager) GetAccount(ctx context.Context, alias string) (*Account, error) {
	am.logger.Info(ctx, "getting_account",
		observability.F("alias", alias),
	)

	if err := am.configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	account, err := am.configManager.GetAccount(alias)
	if err != nil {
		return nil, fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	result := &Account{
		Alias:          account.Alias,
		Name:           account.Name,
		Email:          account.Email,
		GitHubUsername: account.GitHubUsername,
		SSHKeyPath:     account.SSHKeyPath,
		Description:    account.Description,
		IsActive:       account.Status == models.AccountStatusActive,
		CreatedAt:      account.CreatedAt.Format(time.RFC3339),
	}

	if account.LastUsed != nil {
		lastUsed := account.LastUsed.Format(time.RFC3339)
		result.LastUsed = &lastUsed
	}

	return result, nil
}

// ListAccounts returns all configured accounts
func (am *RealAccountManager) ListAccounts(ctx context.Context) ([]*Account, error) {
	am.logger.Info(ctx, "listing_accounts")

	if err := am.configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	accounts := am.configManager.ListAccounts()
	results := make([]*Account, 0, len(accounts))

	for _, account := range accounts {
		result := &Account{
			Alias:          account.Alias,
			Name:           account.Name,
			Email:          account.Email,
			GitHubUsername: account.GitHubUsername,
			SSHKeyPath:     account.SSHKeyPath,
			Description:    account.Description,
			IsActive:       account.Status == models.AccountStatusActive,
			CreatedAt:      account.CreatedAt.Format(time.RFC3339),
		}

		if account.LastUsed != nil {
			lastUsed := account.LastUsed.Format(time.RFC3339)
			result.LastUsed = &lastUsed
		}

		results = append(results, result)
	}

	// Sort by last used (most recent first), then by alias
	sort.Slice(results, func(i, j int) bool {
		if results[i].LastUsed != nil && results[j].LastUsed != nil {
			return *results[i].LastUsed > *results[j].LastUsed
		}
		if results[i].LastUsed != nil {
			return true
		}
		if results[j].LastUsed != nil {
			return false
		}
		return results[i].Alias < results[j].Alias
	})

	am.logger.Info(ctx, "accounts_listed_successfully",
		observability.F("count", len(results)),
	)

	return results, nil
}

// UpdateAccount updates an existing account
func (am *RealAccountManager) UpdateAccount(ctx context.Context, alias string, updates AccountUpdates) error {
	am.logger.Info(ctx, "updating_account",
		observability.F("alias", alias),
	)

	if err := am.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	account, err := am.configManager.GetAccount(alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	// Apply updates
	if updates.Name != nil {
		account.Name = *updates.Name
	}
	if updates.Email != nil {
		account.Email = *updates.Email
	}
	if updates.GitHubUsername != nil {
		account.GitHubUsername = *updates.GitHubUsername
	}
	if updates.SSHKeyPath != nil {
		account.SSHKeyPath = *updates.SSHKeyPath
	}
	if updates.Description != nil {
		account.Description = *updates.Description
	}

	// Validate updated account
	if err := account.Validate(); err != nil {
		return fmt.Errorf("invalid account after updates: %w", err)
	}

	// Save changes
	if err := am.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	am.logger.Info(ctx, "account_updated_successfully",
		observability.F("alias", alias),
	)

	return nil
}

// DeleteAccount deletes an account
func (am *RealAccountManager) DeleteAccount(ctx context.Context, alias string) error {
	am.logger.Info(ctx, "deleting_account",
		observability.F("alias", alias),
	)

	if err := am.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if account exists
	account, err := am.configManager.GetAccount(alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	// Check if it's the current account
	config := am.configManager.GetConfig()
	if config.CurrentAccount == alias {
		return fmt.Errorf("cannot delete active account '%s'. Switch to another account first", alias)
	}

	// Remove the account
	if err := am.configManager.RemoveAccount(alias); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	if err := am.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	am.logger.Info(ctx, "account_deleted_successfully",
		observability.F("alias", alias),
		observability.F("ssh_key_path", account.SSHKeyPath),
	)

	return nil
}

// SwitchAccount switches to the specified account
func (am *RealAccountManager) SwitchAccount(ctx context.Context, alias string) error {
	am.logger.Info(ctx, "switching_account",
		observability.F("alias", alias),
	)

	if err := am.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate account exists
	account, err := am.configManager.GetAccount(alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	// Update current account
	config := am.configManager.GetConfig()
	previousAccount := config.CurrentAccount
	config.CurrentAccount = alias

	// Mark as used
	account.MarkAsUsed()

	// Save configuration
	if err := am.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update Git configuration if Git manager is available
	if am.gitManager != nil {
		if err := am.gitManager.SetGlobalConfig(ctx, &Account{
			Alias:          account.Alias,
			Name:           account.Name,
			Email:          account.Email,
			GitHubUsername: account.GitHubUsername,
			SSHKeyPath:     account.SSHKeyPath,
		}); err != nil {
			am.logger.Warn(ctx, "failed_to_update_git_config",
				observability.F("alias", alias),
				observability.F("error", err.Error()),
			)
		}
	}

	am.logger.Info(ctx, "account_switched_successfully",
		observability.F("from", previousAccount),
		observability.F("to", alias),
	)

	return nil
}

// GetCurrentAccount returns the currently active account
func (am *RealAccountManager) GetCurrentAccount(ctx context.Context) (*Account, error) {
	am.logger.Info(ctx, "getting_current_account")

	if err := am.configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	config := am.configManager.GetConfig()
	if config.CurrentAccount == "" {
		return nil, fmt.Errorf("no account is currently active")
	}

	return am.GetAccount(ctx, config.CurrentAccount)
}

// ValidateAccount validates a single account
func (am *RealAccountManager) ValidateAccount(ctx context.Context, alias string) (*ValidationResult, error) {
	am.logger.Info(ctx, "validating_account",
		observability.F("alias", alias),
	)

	account, err := am.GetAccount(ctx, alias)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Account: alias,
		Valid:   true,
		Issues:  []string{},
		Fixed:   []string{},
	}

	// Validate SSH key
	if account.SSHKeyPath != "" {
		if am.sshManager != nil {
			if _, err := am.sshManager.ValidateKey(ctx, account.SSHKeyPath); err != nil {
				result.Valid = false
				result.Issues = append(result.Issues, fmt.Sprintf("SSH key validation failed: %v", err))
			}
		}
	} else {
		result.Issues = append(result.Issues, "No SSH key configured")
	}

	// Validate GitHub access
	if account.GitHubUsername != "" && am.githubManager != nil {
		if err := am.githubManager.TestAPIAccess(ctx, account); err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("GitHub access failed: %v", err))
		}
	}

	if len(result.Issues) > 0 {
		result.Valid = false
	}

	am.logger.Info(ctx, "account_validation_completed",
		observability.F("alias", alias),
		observability.F("valid", result.Valid),
		observability.F("issues_count", len(result.Issues)),
	)

	return result, nil
}

// ValidateAllAccounts validates all accounts
func (am *RealAccountManager) ValidateAllAccounts(ctx context.Context) ([]*ValidationResult, error) {
	am.logger.Info(ctx, "validating_all_accounts")

	accounts, err := am.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*ValidationResult, 0, len(accounts))

	for _, account := range accounts {
		result, err := am.ValidateAccount(ctx, account.Alias)
		if err != nil {
			result = &ValidationResult{
				Account: account.Alias,
				Valid:   false,
				Issues:  []string{fmt.Sprintf("Validation error: %v", err)},
				Fixed:   []string{},
			}
		}
		results = append(results, result)
	}

	am.logger.Info(ctx, "all_accounts_validation_completed",
		observability.F("total", len(results)),
	)

	return results, nil
}

// DiscoverAccounts discovers potential accounts from the system
func (am *RealAccountManager) DiscoverAccounts(ctx context.Context) ([]*DiscoveredAccount, error) {
	am.logger.Info(ctx, "discovering_accounts")

	discovered := []*DiscoveredAccount{}

	// Discover from SSH keys
	if sshAccounts := am.discoverFromSSHKeys(ctx); len(sshAccounts) > 0 {
		discovered = append(discovered, sshAccounts...)
	}

	// Discover from Git config
	if gitAccounts := am.discoverFromGitConfig(ctx); len(gitAccounts) > 0 {
		discovered = append(discovered, gitAccounts...)
	}

	// Discover from repositories
	if repoAccounts := am.discoverFromRepositories(ctx); len(repoAccounts) > 0 {
		discovered = append(discovered, repoAccounts...)
	}

	am.logger.Info(ctx, "accounts_discovery_completed",
		observability.F("discovered_count", len(discovered)),
	)

	return discovered, nil
}

// SetupAccount sets up a discovered account
func (am *RealAccountManager) SetupAccount(ctx context.Context, discovered *DiscoveredAccount) (*Account, error) {
	am.logger.Info(ctx, "setting_up_discovered_account",
		observability.F("source", discovered.Source),
		observability.F("username", discovered.Username),
	)

	// Create account request from discovered data
	req := CreateAccountRequest{
		Alias:          am.generateAccountAlias(discovered.Username),
		Name:           discovered.Username,
		Email:          discovered.Email,
		GitHubUsername: discovered.Username,
		SSHKeyPath:     discovered.SSHKeyPath,
		Description:    fmt.Sprintf("Auto-discovered from %s", discovered.Source),
	}

	return am.CreateAccount(ctx, req)
}

// Helper methods

func (am *RealAccountManager) validateCreateRequest(req CreateAccountRequest) error {
	if req.Alias == "" {
		return fmt.Errorf("alias is required")
	}

	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	// Validate email format
	if !strings.Contains(req.Email, "@") {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func (am *RealAccountManager) generateSSHKeyPath(alias string) string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".ssh", fmt.Sprintf("id_ed25519_%s", alias))
}

func (am *RealAccountManager) generateAccountAlias(username string) string {
	// Simple alias generation - can be made more sophisticated
	return strings.ToLower(username)
}

func (am *RealAccountManager) discoverFromSSHKeys(ctx context.Context) []*DiscoveredAccount {
	discovered := []*DiscoveredAccount{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return discovered
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return discovered
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "id_") || strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())

		// Try to extract info from key
		if parts := strings.Split(entry.Name(), "_"); len(parts) > 1 {
			account := &DiscoveredAccount{
				Source:     "ssh",
				Confidence: 60,
				Username:   parts[len(parts)-1],
				SSHKeyPath: keyPath,
			}
			discovered = append(discovered, account)
		}
	}

	return discovered
}

func (am *RealAccountManager) discoverFromGitConfig(ctx context.Context) []*DiscoveredAccount {
	discovered := []*DiscoveredAccount{}

	if am.gitManager != nil {
		if gitConfig, err := am.gitManager.GetCurrentConfig(ctx); err == nil {
			account := &DiscoveredAccount{
				Source:     "git",
				Confidence: 80,
				Username:   gitConfig.Name,
				Email:      gitConfig.Email,
			}
			discovered = append(discovered, account)
		}
	}

	return discovered
}

func (am *RealAccountManager) discoverFromRepositories(ctx context.Context) []*DiscoveredAccount {
	// TODO: Implement repository-based discovery
	return []*DiscoveredAccount{}
}
