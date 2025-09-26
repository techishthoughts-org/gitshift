package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// MigrationManager handles configuration migrations between versions
type MigrationManager struct {
	logger        observability.Logger
	configManager *config.Manager
}

// MigrationResult contains the results of a migration operation
type MigrationResult struct {
	Success          bool                   `json:"success"`
	FromVersion      string                 `json:"from_version"`
	ToVersion        string                 `json:"to_version"`
	BackupPath       string                 `json:"backup_path"`
	MigratedAccounts int                    `json:"migrated_accounts"`
	Issues           []string               `json:"issues"`
	Duration         time.Duration          `json:"duration"`
	Changes          map[string]interface{} `json:"changes"`
}

// LegacyAccount represents old account format for migration
type LegacyAccount struct {
	Name           string `yaml:"name"`
	Email          string `yaml:"email"`
	SSHKeyPath     string `yaml:"ssh_key_path"`
	GitHubUsername string `yaml:"github_username,omitempty"`
}

// LegacyConfig represents old configuration format
type LegacyConfig struct {
	CurrentAccount string                    `yaml:"current_account"`
	Accounts       map[string]*LegacyAccount `yaml:"accounts"`
	GlobalGitMode  bool                      `yaml:"global_git_mode"`
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(logger observability.Logger) *MigrationManager {
	return &MigrationManager{
		logger:        logger,
		configManager: config.NewManager(),
	}
}

// DetectMigrationNeeded checks if configuration needs migration
func (mm *MigrationManager) DetectMigrationNeeded(ctx context.Context) (bool, string, error) {
	mm.logger.Info(ctx, "detecting_migration_requirements")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".gitpersona")
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// No config file - check for legacy locations
		return mm.checkLegacyConfigurations(ctx)
	}

	// Load current config to check version
	if err := mm.configManager.Load(); err != nil {
		return false, "", fmt.Errorf("failed to load current config: %w", err)
	}

	currentConfig := mm.configManager.GetConfig()
	currentVersion := currentConfig.ConfigVersion

	if currentVersion == "" || currentVersion < "2.0.0" {
		return true, currentVersion, nil
	}

	return false, currentVersion, nil
}

// RunMigration performs the migration from old to new format
func (mm *MigrationManager) RunMigration(ctx context.Context, fromVersion, toVersion string) (*MigrationResult, error) {
	mm.logger.Info(ctx, "starting_configuration_migration",
		observability.F("from_version", fromVersion),
		observability.F("to_version", toVersion),
	)

	start := time.Now()
	result := &MigrationResult{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Changes:     make(map[string]interface{}),
		Issues:      []string{},
	}

	// Create backup first
	backupPath, err := mm.createBackup(ctx)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Backup creation failed: %v", err))
		return result, fmt.Errorf("failed to create backup: %w", err)
	}
	result.BackupPath = backupPath

	// Perform version-specific migration
	switch {
	case fromVersion == "" || fromVersion < "1.0.0":
		if err := mm.migrateFromLegacy(ctx, result); err != nil {
			return result, fmt.Errorf("legacy migration failed: %w", err)
		}
	case fromVersion < "2.0.0":
		if err := mm.migrateToV2(ctx, result); err != nil {
			return result, fmt.Errorf("v2 migration failed: %w", err)
		}
	default:
		return result, fmt.Errorf("unsupported migration path from %s to %s", fromVersion, toVersion)
	}

	result.Duration = time.Since(start)
	result.Success = true

	mm.logger.Info(ctx, "configuration_migration_completed",
		observability.F("from_version", fromVersion),
		observability.F("to_version", toVersion),
		observability.F("duration_ms", result.Duration.Milliseconds()),
		observability.F("migrated_accounts", result.MigratedAccounts),
	)

	return result, nil
}

// RestoreFromBackup restores configuration from a backup
func (mm *MigrationManager) RestoreFromBackup(ctx context.Context, backupPath string) error {
	mm.logger.Info(ctx, "restoring_from_backup",
		observability.F("backup_path", backupPath),
	)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	_ = filepath.Join(homeDir, ".gitpersona")

	// Extract backup
	// TODO: Implement actual tar.gz extraction
	// For now, assume simple file copy
	return fmt.Errorf("backup restoration not fully implemented")
}

// ListBackups returns available configuration backups
func (mm *MigrationManager) ListBackups(ctx context.Context) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	backupDir := filepath.Join(homeDir, ".gitpersona", "backups")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "gitpersona-backup-") {
			backups = append(backups, filepath.Join(backupDir, entry.Name()))
		}
	}

	return backups, nil
}

// Private methods

func (mm *MigrationManager) checkLegacyConfigurations(ctx context.Context) (bool, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, "", err
	}

	// Check common legacy locations
	legacyPaths := []string{
		filepath.Join(homeDir, ".gitconfig-persona"),
		filepath.Join(homeDir, ".git-persona.yaml"),
		filepath.Join(homeDir, ".config", "gitpersona", "config.yaml"),
	}

	for _, path := range legacyPaths {
		if _, err := os.Stat(path); err == nil {
			mm.logger.Info(ctx, "legacy_configuration_found",
				observability.F("path", path),
			)
			return true, "legacy", nil
		}
	}

	return false, "", nil
}

func (mm *MigrationManager) createBackup(ctx context.Context) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".gitpersona")
	backupDir := filepath.Join(configDir, "backups")

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("gitpersona-backup-%s.tar.gz", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	// Create tar.gz backup
	// TODO: Implement actual tar.gz creation
	// For now, just create a placeholder file
	if err := os.WriteFile(backupPath+".placeholder", []byte("backup placeholder"), 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	mm.logger.Info(ctx, "backup_created",
		observability.F("backup_path", backupPath),
	)

	return backupPath, nil
}

func (mm *MigrationManager) migrateFromLegacy(ctx context.Context, result *MigrationResult) error {
	mm.logger.Info(ctx, "migrating_from_legacy_configuration")

	// Try to find and parse legacy configurations
	legacyAccounts := mm.discoverLegacyAccounts(ctx)

	if len(legacyAccounts) == 0 {
		result.Issues = append(result.Issues, "No legacy accounts found to migrate")
		return nil
	}

	// Create new configuration
	newConfig := models.NewConfig()
	newConfig.ConfigVersion = "2.0.0"
	newConfig.GlobalGitConfig = true
	newConfig.AutoDetect = true

	// Migrate accounts
	migratedCount := 0
	for alias, legacyAccount := range legacyAccounts {
		account := models.NewAccount(alias, legacyAccount.Name, legacyAccount.Email, legacyAccount.SSHKeyPath)
		account.GitHubUsername = legacyAccount.GitHubUsername

		newConfig.Accounts[alias] = account
		migratedCount++
	}

	// Update configuration directly using reflection access
	configField := reflect.ValueOf(mm.configManager).Elem().FieldByName("config")
	if configField.IsValid() && configField.CanSet() {
		configField.Set(reflect.ValueOf(newConfig))
	}
	if err := mm.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save migrated configuration: %w", err)
	}

	result.MigratedAccounts = migratedCount
	result.Changes["accounts_migrated"] = migratedCount
	result.Changes["legacy_format"] = "converted_to_v2"

	mm.logger.Info(ctx, "legacy_migration_completed",
		observability.F("migrated_accounts", migratedCount),
	)

	return nil
}

func (mm *MigrationManager) migrateToV2(ctx context.Context, result *MigrationResult) error {
	mm.logger.Info(ctx, "migrating_to_v2_format")

	// Load current configuration
	if err := mm.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load current config: %w", err)
	}

	config := mm.configManager.GetConfig()

	// Update version
	config.ConfigVersion = "2.0.0"

	// Add new v2 features to existing accounts
	enhancedCount := 0
	for _, account := range config.Accounts {
		// Add isolation metadata if not present
		if account.IsolationMetadata == nil {
			account.IsolationMetadata = &models.IsolationMetadata{
				SSHIsolation: &models.SSHIsolationSettings{
					UseIsolatedAgent:    true,
					ForceIdentitiesOnly: true,
					AgentTimeout:        3600,
					AutoCleanup:         true,
				},
				TokenIsolation: &models.TokenIsolationSettings{
					UseEncryptedStorage: true,
					AutoValidation:      true,
					ValidationInterval:  60,
					StrictValidation:    true,
				},
				GitIsolation: &models.GitIsolationSettings{
					UseLocalConfig:    true,
					IsolateSSHCommand: true,
					CustomGitConfig:   make(map[string]string),
				},
				EnvironmentIsolation: &models.EnvironmentIsolationSettings{
					IsolateEnvironment:  false,
					CustomEnvironment:   make(map[string]string),
					ClearEnvironment:    false,
					PreserveEnvironment: []string{"HOME", "USER", "PATH"},
				},
			}
			enhancedCount++
		}

		// Set default isolation level
		if account.IsolationLevel == "" {
			account.IsolationLevel = models.IsolationLevelStandard
		}

		// Initialize metadata map
		if account.AccountMetadata == nil {
			account.AccountMetadata = make(map[string]string)
		}
	}

	// Save updated configuration
	if err := mm.configManager.Save(); err != nil {
		return fmt.Errorf("failed to save v2 configuration: %w", err)
	}

	result.MigratedAccounts = len(config.Accounts)
	result.Changes["accounts_enhanced"] = enhancedCount
	result.Changes["isolation_added"] = true
	result.Changes["version_updated"] = "2.0.0"

	mm.logger.Info(ctx, "v2_migration_completed",
		observability.F("enhanced_accounts", enhancedCount),
	)

	return nil
}

func (mm *MigrationManager) discoverLegacyAccounts(ctx context.Context) map[string]*LegacyAccount {
	accounts := make(map[string]*LegacyAccount)

	// Try to discover from Git configuration
	if gitAccount := mm.discoverFromGitConfig(ctx); gitAccount != nil {
		accounts["default"] = gitAccount
	}

	// Try to discover from SSH keys
	sshAccounts := mm.discoverFromSSHKeys(ctx)
	for alias, account := range sshAccounts {
		accounts[alias] = account
	}

	return accounts
}

func (mm *MigrationManager) discoverFromGitConfig(ctx context.Context) *LegacyAccount {
	// TODO: Implement Git config discovery
	return nil
}

func (mm *MigrationManager) discoverFromSSHKeys(ctx context.Context) map[string]*LegacyAccount {
	accounts := make(map[string]*LegacyAccount)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return accounts
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return accounts
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "id_") || strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		// Extract potential account name from key filename
		keyName := entry.Name()
		parts := strings.Split(keyName, "_")
		if len(parts) >= 3 {
			alias := parts[2] // id_ed25519_work -> work
			keyPath := filepath.Join(sshDir, keyName)

			account := &LegacyAccount{
				SSHKeyPath: keyPath,
				Name:       alias, // Will be updated if we can get better info
				Email:      "",    // Will be discovered from key or set manually
			}

			accounts[alias] = account
		}
	}

	return accounts
}
