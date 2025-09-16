package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealConfigService implements the config service with actual functionality
type RealConfigService struct {
	configPath string
	manager    *config.Manager
	logger     observability.Logger
}

// NewRealConfigService creates a new real config service
func NewRealConfigService(configPath string, logger observability.Logger) *RealConfigService {
	return &RealConfigService{
		configPath: configPath,
		manager:    config.NewManager(),
		logger:     logger,
	}
}

func (s *RealConfigService) Load(ctx context.Context) error {
	s.logger.Info(ctx, "loading_configuration",
		observability.F("config_path", s.configPath),
	)

	if err := s.manager.Load(); err != nil {
		s.logger.Error(ctx, "failed_to_load_configuration",
			observability.F("error", err.Error()),
		)
		return err
	}

	s.logger.Info(ctx, "configuration_loaded_successfully")
	return nil
}

func (s *RealConfigService) Save(ctx context.Context) error {
	s.logger.Info(ctx, "saving_configuration",
		observability.F("config_path", s.configPath),
	)

	if err := s.manager.Save(); err != nil {
		s.logger.Error(ctx, "failed_to_save_configuration",
			observability.F("error", err.Error()),
		)
		return err
	}

	s.logger.Info(ctx, "configuration_saved_successfully")
	return nil
}

func (s *RealConfigService) Reload(ctx context.Context) error {
	s.logger.Info(ctx, "reloading_configuration",
		observability.F("config_path", s.configPath),
	)

	if err := s.manager.Load(); err != nil {
		s.logger.Error(ctx, "failed_to_reload_configuration",
			observability.F("error", err.Error()),
		)
		return err
	}

	s.logger.Info(ctx, "configuration_reloaded_successfully")
	return nil
}

func (s *RealConfigService) Validate(ctx context.Context) error {
	s.logger.Info(ctx, "validating_configuration")

	// Basic validation - check if config file exists and is readable
	if _, err := os.Stat(filepath.Join(s.configPath, "config.yaml")); os.IsNotExist(err) {
		s.logger.Warn(ctx, "config_file_not_found")
		return nil // Not an error, just means no config yet
	}

	s.logger.Info(ctx, "configuration_validation_passed")
	return nil
}

func (s *RealConfigService) Get(ctx context.Context, key string) interface{} {
	s.logger.Info(ctx, "getting_config_value",
		observability.F("key", key),
	)

	// TODO: Implement key-based config access
	return nil
}

func (s *RealConfigService) Set(ctx context.Context, key string, value interface{}) error {
	s.logger.Info(ctx, "setting_config_value",
		observability.F("key", key),
		observability.F("value", value),
	)

	// TODO: Implement key-based config setting
	return nil
}

func (s *RealConfigService) GetString(ctx context.Context, key string) string {
	s.logger.Info(ctx, "getting_config_string",
		observability.F("key", key),
	)

	// TODO: Implement key-based config string access
	return ""
}

func (s *RealConfigService) GetBool(ctx context.Context, key string) bool {
	s.logger.Info(ctx, "getting_config_bool",
		observability.F("key", key),
	)

	// TODO: Implement key-based config bool access
	return false
}

func (s *RealConfigService) GetInt(ctx context.Context, key string) int {
	s.logger.Info(ctx, "getting_config_int",
		observability.F("key", key),
	)

	// TODO: Implement key-based config int access
	return 0
}

// GetAccount retrieves an account by alias
func (s *RealConfigService) GetAccount(ctx context.Context, alias string) (*models.Account, error) {
	s.logger.Info(ctx, "getting_account",
		observability.F("alias", alias),
	)

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get account from manager
	account, err := s.manager.GetAccount(alias)
	if err != nil {
		s.logger.Error(ctx, "failed_to_get_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get account '%s': %w", alias, err)
	}

	return account, nil
}

// SetAccount sets an account
func (s *RealConfigService) SetAccount(ctx context.Context, account *models.Account) error {
	s.logger.Info(ctx, "setting_account",
		observability.F("alias", account.Alias),
	)

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set account in manager
	if err := s.manager.AddAccount(account); err != nil {
		s.logger.Error(ctx, "failed_to_set_account",
			observability.F("alias", account.Alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to set account '%s': %w", account.Alias, err)
	}

	// Save configuration
	if err := s.Save(ctx); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// DeleteAccount deletes an account
func (s *RealConfigService) DeleteAccount(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "deleting_account",
		observability.F("alias", alias),
	)

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Delete account from manager
	if err := s.manager.RemoveAccount(alias); err != nil {
		s.logger.Error(ctx, "failed_to_delete_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to delete account '%s': %w", alias, err)
	}

	// Save configuration
	if err := s.Save(ctx); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// ListAccounts lists all accounts
func (s *RealConfigService) ListAccounts(ctx context.Context) ([]*models.Account, error) {
	s.logger.Info(ctx, "listing_accounts")

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get accounts from manager
	accounts := s.manager.ListAccounts()
	accountList := make([]*models.Account, 0, len(accounts))
	accountList = append(accountList, accounts...)

	return accountList, nil
}

// SetCurrentAccount sets the current account
func (s *RealConfigService) SetCurrentAccount(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "setting_current_account",
		observability.F("alias", alias),
	)

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set current account in manager
	if err := s.manager.SetCurrentAccount(alias); err != nil {
		s.logger.Error(ctx, "failed_to_set_current_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to set current account to '%s': %w", alias, err)
	}

	// Save configuration
	if err := s.Save(ctx); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// GetCurrentAccount gets the current account
func (s *RealConfigService) GetCurrentAccount(ctx context.Context) string {
	// Load configuration first
	if err := s.Load(ctx); err != nil {
		s.logger.Error(ctx, "failed_to_load_configuration_for_current_account",
			observability.F("error", err.Error()),
		)
		return ""
	}

	config := s.manager.GetConfig()
	return config.CurrentAccount
}

// CheckForConflicts checks for configuration conflicts
func (s *RealConfigService) CheckForConflicts(ctx context.Context) ([]*ConfigConflict, error) {
	s.logger.Info(ctx, "checking_for_configuration_conflicts")

	conflicts := []*ConfigConflict{}

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check for duplicate accounts
	accounts := s.manager.ListAccounts()
	seenEmails := make(map[string]string)
	seenSSHKeys := make(map[string]string)

	for _, account := range accounts {
		alias := account.Alias
		// Check for duplicate emails
		if account.Email != "" {
			if existingAlias, exists := seenEmails[account.Email]; exists {
				conflicts = append(conflicts, &ConfigConflict{
					Type:        "duplicate_email",
					Description: "Email '" + account.Email + "' is used by both '" + existingAlias + "' and '" + alias + "'",
					Severity:    "high",
					Resolution:  "Use different email addresses for different accounts",
				})
			} else {
				seenEmails[account.Email] = alias
			}
		}

		// Check for duplicate SSH keys
		if account.SSHKeyPath != "" {
			if existingAlias, exists := seenSSHKeys[account.SSHKeyPath]; exists {
				conflicts = append(conflicts, &ConfigConflict{
					Type:        "duplicate_ssh_key",
					Description: "SSH key '" + account.SSHKeyPath + "' is used by both '" + existingAlias + "' and '" + alias + "'",
					Severity:    "high",
					Resolution:  "Use different SSH keys for different accounts",
				})
			} else {
				seenSSHKeys[account.SSHKeyPath] = alias
			}
		}
	}

	s.logger.Info(ctx, "configuration_conflicts_checked",
		observability.F("conflicts_count", len(conflicts)),
	)

	return conflicts, nil
}

// ValidateConfiguration validates the configuration
func (s *RealConfigService) ValidateConfiguration(ctx context.Context) error {
	s.logger.Info(ctx, "validating_configuration")

	// Load configuration first
	if err := s.Load(ctx); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check for conflicts
	conflicts, err := s.CheckForConflicts(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for conflicts: %w", err)
	}

	// If there are high-severity conflicts, return an error
	for _, conflict := range conflicts {
		if conflict.Severity == "high" {
			return fmt.Errorf("high-severity configuration conflict: %s", conflict.Description)
		}
	}

	s.logger.Info(ctx, "configuration_validation_passed")
	return nil
}
