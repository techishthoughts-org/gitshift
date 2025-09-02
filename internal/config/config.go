package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"github.com/techishthoughts/GitPersona/internal/models"
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

	// Create a new viper instance for reading
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into our config struct
	if err := v.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize accounts map if nil
	if m.config.Accounts == nil {
		m.config.Accounts = make(map[string]*models.Account)
	}

	// Initialize pending accounts map if nil
	if m.config.PendingAccounts == nil {
		m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	}

	return nil
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	configFile := filepath.Join(m.configPath, ConfigFileName+".yaml")

	// Create a new viper instance for writing
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Set the config values
	v.Set("accounts", m.config.Accounts)
	v.Set("pending_accounts", m.config.PendingAccounts)
	v.Set("current_account", m.config.CurrentAccount)
	v.Set("global_git_config", m.config.GlobalGitConfig)
	v.Set("auto_detect", m.config.AutoDetect)
	v.Set("config_version", m.config.ConfigVersion)

	if err := v.WriteConfig(); err != nil {
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

// ClearAllAccounts removes all accounts from the configuration
func (m *Manager) ClearAllAccounts() error {
	m.config.Accounts = make(map[string]*models.Account)
	m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	m.config.CurrentAccount = ""
	return m.Save()
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

// AddPendingAccount adds a pending account that needs manual completion
func (m *Manager) AddPendingAccount(pending *models.PendingAccount) error {
	if pending == nil {
		return fmt.Errorf("cannot add nil pending account")
	}

	if pending.Alias == "" {
		return fmt.Errorf("pending account must have an alias")
	}

	m.config.PendingAccounts[pending.Alias] = pending
	return m.Save()
}

// GetPendingAccount returns a pending account by alias
func (m *Manager) GetPendingAccount(alias string) (*models.PendingAccount, error) {
	pending, exists := m.config.PendingAccounts[alias]
	if !exists {
		return nil, models.ErrAccountNotFound
	}
	return pending, nil
}

// ListPendingAccounts returns all pending accounts
func (m *Manager) ListPendingAccounts() []*models.PendingAccount {
	pending := make([]*models.PendingAccount, 0, len(m.config.PendingAccounts))
	for _, account := range m.config.PendingAccounts {
		pending = append(pending, account)
	}
	return pending
}

// CompletePendingAccount converts a pending account to an active account
func (m *Manager) CompletePendingAccount(alias string, name, email string) (*models.Account, error) {
	pending, exists := m.config.PendingAccounts[alias]
	if !exists {
		return nil, models.ErrAccountNotFound
	}

	// Create the completed account
	account := &models.Account{
		Alias:          alias,
		Name:           name,
		Email:          email,
		GitHubUsername: pending.GitHubUsername,
		SSHKeyPath:     pending.PartialData["ssh_key_path"],
		Description:    fmt.Sprintf("Completed from pending account (source: %s)", pending.Source),
		Status:         models.AccountStatusActive,
		IsDefault:      false,
		CreatedAt:      time.Now(),
	}

	// Validate the completed account
	if err := account.Validate(); err != nil {
		return nil, fmt.Errorf("completed account validation failed: %w", err)
	}

	// Add to active accounts
	m.config.Accounts[alias] = account

	// Remove from pending accounts
	delete(m.config.PendingAccounts, alias)

	// Save configuration
	if err := m.Save(); err != nil {
		return nil, fmt.Errorf("failed to save completed account: %w", err)
	}

	return account, nil
}

// RemovePendingAccount removes a pending account
func (m *Manager) RemovePendingAccount(alias string) error {
	if _, exists := m.config.PendingAccounts[alias]; !exists {
		return models.ErrAccountNotFound
	}

	delete(m.config.PendingAccounts, alias)
	return m.Save()
}

// ClearAllPendingAccounts removes all pending accounts
func (m *Manager) ClearAllPendingAccounts() error {
	m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	return m.Save()
}
