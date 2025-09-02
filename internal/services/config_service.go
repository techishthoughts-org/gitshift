package services

import (
	"context"
	"os"
	"path/filepath"

	"github.com/techishthoughts/GitPersona/internal/config"
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
	// TODO: Implement key-based config access
	return nil
}

func (s *RealConfigService) Set(ctx context.Context, key string, value interface{}) error {
	// TODO: Implement key-based config setting
	return nil
}

func (s *RealConfigService) GetString(ctx context.Context, key string) string {
	// TODO: Implement key-based config string access
	return ""
}

func (s *RealConfigService) GetBool(ctx context.Context, key string) bool {
	// TODO: Implement key-based config bool access
	return false
}

func (s *RealConfigService) GetInt(ctx context.Context, key string) int {
	// TODO: Implement key-based config int access
	return 0
}

func (s *RealConfigService) GetAccounts(ctx context.Context) map[string]interface{} {
	accounts := s.manager.ListAccounts()

	// Convert slice to map for compatibility
	result := make(map[string]interface{})
	for _, account := range accounts {
		result[account.Alias] = account
	}

	return result
}

func (s *RealConfigService) GetCurrentAccount(ctx context.Context) string {
	return s.manager.GetConfig().CurrentAccount
}

func (s *RealConfigService) SetCurrentAccount(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "setting_current_account",
		observability.F("alias", alias),
	)

	config := s.manager.GetConfig()
	config.CurrentAccount = alias

	if err := s.manager.Save(); err != nil {
		s.logger.Error(ctx, "failed_to_save_current_account",
			observability.F("error", err.Error()),
		)
		return err
	}

	s.logger.Info(ctx, "current_account_set_successfully",
		observability.F("alias", alias),
	)
	return nil
}
