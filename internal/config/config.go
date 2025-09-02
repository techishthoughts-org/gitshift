package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/spf13/viper"
)

const (
	ConfigDirName     = ".config/gitpersona"
	ConfigFileName    = "config"
	ProjectConfigName = ".gitpersona.yaml"
)

type Manager struct {
	configPath string
	config     *models.Config
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	configPath := filepath.Join(homeDir, ConfigDirName)
	return &Manager{
		configPath: configPath,
		config:     models.NewConfig(),
	}
}

// Load loads the configuration from file
func (m *Manager) Load() error {
	// Ensure config directory exists
	if err := os.MkdirAll(m.configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(m.configPath, ConfigFileName+".yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create a default config file
		return m.Save()
	}

	// Set up viper
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize accounts map if nil
	if m.config.Accounts == nil {
		m.config.Accounts = make(map[string]*models.Account)
	}

	return nil
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	configFile := filepath.Join(m.configPath, ConfigFileName+".yaml")

	// Set up viper for writing
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// Set the config values
	viper.Set("accounts", m.config.Accounts)
	viper.Set("current_account", m.config.CurrentAccount)
	viper.Set("global_git_config", m.config.GlobalGitConfig)
	viper.Set("auto_detect", m.config.AutoDetect)
	viper.Set("config_version", m.config.ConfigVersion)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *models.Config {
	return m.config
}

// AddAccount adds a new account to the configuration
func (m *Manager) AddAccount(account *models.Account) error {
	if account == nil {
		return fmt.Errorf("cannot add nil account")
	}

	if err := account.Validate(); err != nil {
		return err
	}

	if _, exists := m.config.Accounts[account.Alias]; exists {
		return models.ErrAccountExists
	}

	// If this is the first account, make it default
	if len(m.config.Accounts) == 0 {
		account.IsDefault = true
		m.config.CurrentAccount = account.Alias
	}

	m.config.Accounts[account.Alias] = account
	return m.Save()
}

// RemoveAccount removes an account from the configuration
func (m *Manager) RemoveAccount(alias string) error {
	account, exists := m.config.Accounts[alias]
	if !exists {
		return models.ErrAccountNotFound
	}

	delete(m.config.Accounts, alias)

	// If we removed the current account, clear it
	if m.config.CurrentAccount == alias {
		m.config.CurrentAccount = ""
	}

	// If we removed the default account, set a new one
	if account.IsDefault && len(m.config.Accounts) > 0 {
		// Set the first remaining account as default
		for _, acc := range m.config.Accounts {
			acc.IsDefault = true
			m.config.CurrentAccount = acc.Alias
			break
		}
	}

	return m.Save()
}

// GetAccount returns an account by alias
func (m *Manager) GetAccount(alias string) (*models.Account, error) {
	account, exists := m.config.Accounts[alias]
	if !exists {
		return nil, models.ErrAccountNotFound
	}
	return account, nil
}

// ListAccounts returns all accounts
func (m *Manager) ListAccounts() []*models.Account {
	accounts := make([]*models.Account, 0, len(m.config.Accounts))
	for _, account := range m.config.Accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// SetCurrentAccount sets the current active account
func (m *Manager) SetCurrentAccount(alias string) error {
	account, exists := m.config.Accounts[alias]
	if !exists {
		return models.ErrAccountNotFound
	}

	m.config.CurrentAccount = alias
	account.MarkAsUsed()
	return m.Save()
}

// GetCurrentAccount returns the current active account
func (m *Manager) GetCurrentAccount() (*models.Account, error) {
	if m.config.CurrentAccount == "" {
		return nil, models.ErrNoDefaultAccount
	}

	return m.GetAccount(m.config.CurrentAccount)
}

// LoadProjectConfig loads project-specific configuration
func (m *Manager) LoadProjectConfig(projectPath string) (*models.ProjectConfig, error) {
	configFile := filepath.Join(projectPath, ProjectConfigName)

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, models.ErrConfigNotFound
	}

	viper := viper.New()
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	var projectConfig models.ProjectConfig
	if err := viper.Unmarshal(&projectConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project config: %w", err)
	}

	return &projectConfig, nil
}

// SaveProjectConfig saves project-specific configuration
func (m *Manager) SaveProjectConfig(projectPath string, config *models.ProjectConfig) error {
	configFile := filepath.Join(projectPath, ProjectConfigName)

	viper := viper.New()
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.Set("account", config.Account)
	viper.Set("created_at", config.CreatedAt)

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write project config: %w", err)
	}

	return nil
}
