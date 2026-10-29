package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/techishthoughts/gitshift/internal/autoswitch"
	"github.com/techishthoughts/gitshift/internal/models"
	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName     = ".config/gitshift"
	ConfigFileName    = "config"
	ProjectConfigName = ".gitshift.yaml"
)

// ErrConfigChanged prevents a stale manager from overwriting a newer file.
var ErrConfigChanged = errors.New("configuration changed on disk; reload and retry")

type Manager struct {
	configPath string
	configFile string
	config     *models.Config
	mu         sync.RWMutex
	loadedHash [sha256.Size]byte
	loaded     bool
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	if configuredFile := viper.ConfigFileUsed(); configuredFile != "" {
		return NewManagerWithPath(configuredFile)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	configFile := filepath.Join(homeDir, ConfigDirName, ConfigFileName+".yaml")
	return NewManagerWithPath(configFile)
}

// NewManagerWithPath creates a manager for the exact configuration file path.
func NewManagerWithPath(configFile string) *Manager {
	if configFile == "" {
		return NewManager()
	}

	configFile = filepath.Clean(configFile)
	return &Manager{
		configPath: filepath.Dir(configFile),
		configFile: configFile,
		config:     models.NewConfig(),
	}
}

// Load loads the configuration from file
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if config file exists
	if _, err := os.Stat(m.configFile); os.IsNotExist(err) {
		// Create a default config file
		return m.saveLocked()
	} else if err != nil {
		return fmt.Errorf("failed to inspect config file: %w", err)
	}

	// Read the file directly and use yaml.v3 to unmarshal
	// This avoids Viper's issue with dots in map keys
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into a zero value so an omitted version can be distinguished
	// from the current version supplied by NewConfig.
	loaded := &models.Config{}
	if err := yaml.Unmarshal(data, loaded); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	needsSave, err := configVersionMigration(loaded.ConfigVersion)
	if err != nil {
		return err
	}
	if needsSave {
		loaded.ConfigVersion = "1.1.0"
	}
	if err := os.Chmod(m.configFile, 0600); err != nil {
		return fmt.Errorf("failed to secure config file permissions: %w", err)
	}
	m.config = loaded
	m.loadedHash = sha256.Sum256(data)
	m.loaded = true

	// Initialize accounts map if nil
	if m.config.Accounts == nil {
		m.config.Accounts = make(map[string]*models.Account)
	}

	// Initialize pending accounts map if nil
	if m.config.PendingAccounts == nil {
		m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	}

	// Fix accounts with zero CreatedAt values (migration fix)
	for _, account := range m.config.Accounts {
		if account.CreatedAt.IsZero() {
			account.CreatedAt = time.Now()
			needsSave = true
		}
	}

	// Fix pending accounts with zero CreatedAt values
	for _, pending := range m.config.PendingAccounts {
		if pending.CreatedAt.IsZero() {
			pending.CreatedAt = time.Now()
			needsSave = true
		}
	}

	// Save the config if we made any fixes
	if needsSave {
		return m.saveLocked()
	}

	return nil
}

// LoadReadOnly loads configuration without creating directories or files,
// changing permissions, or persisting migrations. A missing file is treated as
// an empty configuration. This is intended for report-only commands.
func (m *Manager) LoadReadOnly() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			m.config = models.NewConfig()
			m.loaded = false
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	loaded := &models.Config{}
	if err := yaml.Unmarshal(data, loaded); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	needsMigration, err := configVersionMigration(loaded.ConfigVersion)
	if err != nil {
		return err
	}
	if needsMigration {
		loaded.ConfigVersion = "1.1.0"
	}
	if loaded.Accounts == nil {
		loaded.Accounts = make(map[string]*models.Account)
	}
	if loaded.PendingAccounts == nil {
		loaded.PendingAccounts = make(map[string]*models.PendingAccount)
	}
	m.config = loaded
	m.loadedHash = sha256.Sum256(data)
	m.loaded = true
	return nil
}

func configVersionMigration(version string) (bool, error) {
	switch version {
	case "1.1.0":
		return false, nil
	case "", "1.0.0":
		return true, nil
	default:
		return false, fmt.Errorf("unsupported configuration version %q; supported version is 1.1.0", version)
	}
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) ensureConfigDirectory() error {
	info, err := os.Stat(m.configPath)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("configuration parent %q is not a directory", m.configPath)
		}
		// A custom --config path may live under a shared directory such as
		// /tmp. Never change permissions on a directory GitShift did not create.
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(m.configPath, 0700); err != nil {
		return err
	}
	return os.Chmod(m.configPath, 0700)
}

func (m *Manager) saveLocked() error {
	if err := m.ensureConfigDirectory(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	release, err := m.acquireSaveLock()
	if err != nil {
		return err
	}
	defer release()

	if m.loaded {
		current, err := os.ReadFile(m.configFile)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrConfigChanged
			}
			return fmt.Errorf("failed to verify config before saving: %w", err)
		}
		if sha256.Sum256(current) != m.loadedHash {
			return ErrConfigChanged
		}
	} else if _, err := os.Stat(m.configFile); err == nil {
		return ErrConfigChanged
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to verify config before saving: %w", err)
	}

	// Use direct YAML marshaling instead of Viper to properly handle map keys with dots
	// Viper has issues with dots in map keys, treating them as path separators
	data, err := marshalConfigToYAML(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tempFile, err := os.CreateTemp(m.configPath, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tempName := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()

	if err := tempFile.Chmod(0600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to secure temporary config file: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("failed to sync temporary config file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary config file: %w", err)
	}
	if err := replaceConfigFile(tempName, m.configFile); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	removeTemp = false
	m.loadedHash = sha256.Sum256(data)
	m.loaded = true

	if err := syncConfigDirectory(m.configPath); err != nil {
		return err
	}

	return nil
}

func (m *Manager) acquireSaveLock() (func(), error) {
	return m.acquireFileLock(m.configFile+".lock", "config")
}

// AcquireSwitchLock serializes the complete global-state switch transaction
// across gitshift processes. Callers must release it after all external and
// configuration mutations finish.
func (m *Manager) AcquireSwitchLock() (func(), error) {
	if err := m.ensureConfigDirectory(); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	return m.acquireFileLock(m.configFile+".switch.lock", "switch")
}

func (m *Manager) acquireFileLock(lockPath, kind string) (func(), error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to create %s lock token: %w", kind, err)
	}
	token := []byte(hex.EncodeToString(tokenBytes))
	deadline := time.Now().Add(2 * time.Second)
	for {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			if _, writeErr := lock.Write(token); writeErr != nil {
				_ = lock.Close()
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("failed to write %s lock: %w", kind, writeErr)
			}
			if closeErr := lock.Close(); closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("failed to close %s lock: %w", kind, closeErr)
			}
			return func() {
				currentToken, readErr := os.ReadFile(lockPath)
				if readErr == nil && string(currentToken) == string(token) {
					_ = os.Remove(lockPath)
				}
			}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("failed to create %s lock: %w", kind, err)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for %s lock %q; remove it only if no gitshift process is running", kind, lockPath)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// marshalConfigToYAML marshals the config to YAML format
// This uses gopkg.in/yaml.v3 directly to properly handle map keys with dots
func marshalConfigToYAML(config *models.Config) ([]byte, error) {
	// Using yaml.v3 directly instead of Viper to properly handle map keys with dots
	// yaml.v3 will automatically quote keys that contain special characters like dots

	// Marshal the entire config struct directly
	// The yaml.v3 encoder will handle the map keys with dots correctly
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	return data, nil
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

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.config.Accounts[account.Alias]; exists {
		return models.ErrAccountExists
	}

	// The first account, or an explicitly default account, becomes the default.
	// It only becomes current once a switch applies it.
	if len(m.config.Accounts) == 0 || account.IsDefault {
		for _, existing := range m.config.Accounts {
			existing.IsDefault = false
		}
		account.IsDefault = true
	}

	m.config.Accounts[account.Alias] = account
	return m.saveLocked()
}

// AddAccounts atomically adds a validated batch of accounts. No in-memory or
// on-disk state is changed when any account is invalid or collides with an
// existing alias (comparisons are case-insensitive).
func (m *Manager) AddAccounts(accounts []*models.Account) error {
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts selected for import")
	}
	prepared := make([]*models.Account, len(accounts))
	for i, account := range accounts {
		if account == nil {
			return fmt.Errorf("account %d is nil", i+1)
		}
		copyAccount := *account
		if err := copyAccount.Validate(); err != nil {
			return fmt.Errorf("invalid account %q: %w", copyAccount.Alias, err)
		}
		prepared[i] = &copyAccount
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	originalHash, originalLoaded := m.loadedHash, m.loaded
	original := *m.config
	original.Accounts = make(map[string]*models.Account, len(m.config.Accounts))
	for alias, account := range m.config.Accounts {
		original.Accounts[alias] = account
	}
	original.PendingAccounts = make(map[string]*models.PendingAccount, len(m.config.PendingAccounts))
	for alias, account := range m.config.PendingAccounts {
		original.PendingAccounts[alias] = account
	}
	original.AutoSwitch.Rules = append([]models.AutoSwitchRule(nil), m.config.AutoSwitch.Rules...)
	aliases := make(map[string]string, len(m.config.Accounts)+len(m.config.PendingAccounts)+len(prepared))
	for alias := range m.config.Accounts {
		aliases[strings.ToLower(alias)] = alias
	}
	for alias := range m.config.PendingAccounts {
		aliases[strings.ToLower(alias)] = alias
	}
	for _, account := range prepared {
		key := strings.ToLower(account.Alias)
		if existing, ok := aliases[key]; ok {
			return fmt.Errorf("account alias %q conflicts with %q", account.Alias, existing)
		}
		aliases[key] = account.Alias
	}

	first := len(m.config.Accounts) == 0
	for _, account := range prepared {
		if first {
			account.IsDefault = true
			first = false
		}
		m.config.Accounts[account.Alias] = account
	}
	if err := m.saveLocked(); err != nil {
		m.config = &original
		m.loadedHash, m.loaded = originalHash, originalLoaded
		return err
	}
	return nil
}

// UpdateAccount validates and replaces an existing account in one save. Callers
// should start from a complete copy of the existing account so fields that are
// not being edited remain unchanged.
func (m *Manager) UpdateAccount(account *models.Account) error {
	if account == nil {
		return fmt.Errorf("cannot update nil account")
	}
	if err := account.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	previous, exists := m.config.Accounts[account.Alias]
	if !exists {
		return models.ErrAccountNotFound
	}

	if account.IsDefault {
		for _, existing := range m.config.Accounts {
			existing.IsDefault = false
		}
	} else if previous.IsDefault {
		account.IsDefault = true
	}

	m.config.Accounts[account.Alias] = account
	return m.saveLocked()
}

// ReplaceAccounts atomically replaces active and pending account state. The
// rest of the application configuration is retained.
func (m *Manager) ReplaceAccounts(accounts []*models.Account, pendingAccounts []*models.PendingAccount) error {
	active := make(map[string]*models.Account, len(accounts))
	pending := make(map[string]*models.PendingAccount, len(pendingAccounts))

	var defaultAlias string
	for _, account := range accounts {
		if account == nil {
			return fmt.Errorf("cannot replace accounts with a nil account")
		}
		if err := account.Validate(); err != nil {
			return fmt.Errorf("invalid account %q: %w", account.Alias, err)
		}
		if _, exists := active[account.Alias]; exists {
			return fmt.Errorf("duplicate account alias %q", account.Alias)
		}
		if defaultAlias == "" && account.IsDefault {
			defaultAlias = account.Alias
		}
		active[account.Alias] = account
	}
	for _, account := range pendingAccounts {
		if account == nil || account.Alias == "" {
			return fmt.Errorf("pending account must have an alias")
		}
		if _, exists := pending[account.Alias]; exists {
			return fmt.Errorf("duplicate pending account alias %q", account.Alias)
		}
		pending[account.Alias] = account
	}

	if defaultAlias == "" && len(accounts) > 0 {
		defaultAlias = accounts[0].Alias
	}
	for _, account := range active {
		account.IsDefault = account.Alias == defaultAlias
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Accounts = active
	m.config.PendingAccounts = pending
	keptRules := m.config.AutoSwitch.Rules[:0]
	for _, rule := range m.config.AutoSwitch.Rules {
		if _, exists := active[rule.Account]; exists {
			keptRules = append(keptRules, rule)
		}
	}
	m.config.AutoSwitch.Rules = keptRules
	if _, exists := active[m.config.AutoSwitch.DefaultAccount]; !exists {
		m.config.AutoSwitch.DefaultAccount = ""
	}
	// The applied identity is unchanged; only forget it if its account is gone.
	if _, exists := active[m.config.CurrentAccount]; !exists {
		m.config.CurrentAccount = ""
	}
	return m.saveLocked()
}

// RemoveAccount removes an account from the configuration
func (m *Manager) RemoveAccount(alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	account, exists := m.config.Accounts[alias]
	if !exists {
		return models.ErrAccountNotFound
	}

	delete(m.config.Accounts, alias)
	keptRules := m.config.AutoSwitch.Rules[:0]
	for _, rule := range m.config.AutoSwitch.Rules {
		if rule.Account != alias {
			keptRules = append(keptRules, rule)
		}
	}
	m.config.AutoSwitch.Rules = keptRules
	if m.config.AutoSwitch.DefaultAccount == alias {
		m.config.AutoSwitch.DefaultAccount = ""
	}

	// If we removed the current account, clear it
	if m.config.CurrentAccount == alias {
		m.config.CurrentAccount = ""
	}

	// If we removed the default account, set a new one
	if account.IsDefault && len(m.config.Accounts) > 0 {
		// Set the first remaining account as default
		for _, acc := range m.config.Accounts {
			acc.IsDefault = true
			break
		}
	}

	return m.saveLocked()
}

// GetAccount returns an account by alias
func (m *Manager) GetAccount(alias string) (*models.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	account, exists := m.config.Accounts[alias]
	if !exists {
		return nil, models.ErrAccountNotFound
	}
	return account, nil
}

// ListAccounts returns all accounts
func (m *Manager) ListAccounts() []*models.Account {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]*models.Account, 0, len(m.config.Accounts))
	for _, account := range m.config.Accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// ClearAllAccounts removes all accounts from the configuration
func (m *Manager) ClearAllAccounts() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Accounts = make(map[string]*models.Account)
	m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	m.config.CurrentAccount = ""
	m.config.AutoSwitch.Rules = nil
	m.config.AutoSwitch.DefaultAccount = ""
	return m.saveLocked()
}

// AddAutoSwitchRule adds or replaces the rule for a canonical existing directory.
func (m *Manager) AddAutoSwitchRule(path, alias string) (models.AutoSwitchRule, error) {
	canonical, err := autoswitch.CanonicalDirectory(path)
	if err != nil {
		return models.AutoSwitchRule{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.config.Accounts[alias]; !exists {
		return models.AutoSwitchRule{}, models.ErrAccountNotFound
	}
	rule := models.AutoSwitchRule{Path: canonical, Account: alias, CreatedAt: time.Now().UTC()}
	for i := range m.config.AutoSwitch.Rules {
		if m.config.AutoSwitch.Rules[i].Path == canonical {
			rule.CreatedAt = m.config.AutoSwitch.Rules[i].CreatedAt
			m.config.AutoSwitch.Rules[i] = rule
			return rule, m.saveLocked()
		}
	}
	m.config.AutoSwitch.Rules = append(m.config.AutoSwitch.Rules, rule)
	return rule, m.saveLocked()
}

// SetAutoSwitchDefault sets the account used when no rule matches. An empty
// alias clears it.
func (m *Manager) SetAutoSwitchDefault(alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if alias != "" {
		if _, exists := m.config.Accounts[alias]; !exists {
			return models.ErrAccountNotFound
		}
	}
	m.config.AutoSwitch.DefaultAccount = alias
	return m.saveLocked()
}

// RemoveAutoSwitchRule removes a canonical rule, including a stale rule whose
// directory no longer exists when its exact stored absolute path is supplied.
func (m *Manager) RemoveAutoSwitchRule(path string) error {
	canonical, err := autoswitch.CanonicalDirectory(path)
	if err != nil {
		// A configured root may have been deleted. In that case its physical
		// path can no longer be re-resolved, so permit the exact cleaned,
		// absolute path printed by `auto list`.
		canonical, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("make rule path absolute: %w", err)
		}
		canonical = filepath.Clean(canonical)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.config.AutoSwitch.Rules {
		if m.config.AutoSwitch.Rules[i].Path == canonical {
			m.config.AutoSwitch.Rules = append(m.config.AutoSwitch.Rules[:i], m.config.AutoSwitch.Rules[i+1:]...)
			return m.saveLocked()
		}
	}
	return fmt.Errorf("no auto-switch rule exists for %q", canonical)
}

// ListAutoSwitchRules returns a defensive copy of configured rules.
func (m *Manager) ListAutoSwitchRules() []models.AutoSwitchRule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rules := make([]models.AutoSwitchRule, len(m.config.AutoSwitch.Rules))
	copy(rules, m.config.AutoSwitch.Rules)
	autoswitch.SortRules(rules)
	return rules
}

// SetCurrentAccount sets the current active account
func (m *Manager) SetCurrentAccount(alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	account, exists := m.config.Accounts[alias]
	if !exists {
		return models.ErrAccountNotFound
	}

	// Applying an account never changes which account is the default.
	m.config.CurrentAccount = alias
	account.MarkAsUsed()
	return m.saveLocked()
}

// GetCurrentAccount returns the current active account
func (m *Manager) GetCurrentAccount() (*models.Account, error) {
	m.mu.RLock()
	alias := m.config.CurrentAccount
	m.mu.RUnlock()
	if alias == "" {
		return nil, models.ErrNoDefaultAccount
	}

	return m.GetAccount(alias)
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

	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.PendingAccounts[pending.Alias] = pending
	return m.saveLocked()
}

// GetPendingAccount returns a pending account by alias
func (m *Manager) GetPendingAccount(alias string) (*models.PendingAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pending, exists := m.config.PendingAccounts[alias]
	if !exists {
		return nil, models.ErrAccountNotFound
	}
	return pending, nil
}

// ListPendingAccounts returns all pending accounts
func (m *Manager) ListPendingAccounts() []*models.PendingAccount {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pending := make([]*models.PendingAccount, 0, len(m.config.PendingAccounts))
	for _, account := range m.config.PendingAccounts {
		pending = append(pending, account)
	}
	return pending
}

// CompletePendingAccount converts a pending account to an active account
func (m *Manager) CompletePendingAccount(alias string, name, email string) (*models.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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
		Description:    "Completed from pending account (source: " + pending.Source + ")",
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
	if err := m.saveLocked(); err != nil {
		return nil, fmt.Errorf("failed to save completed account: %w", err)
	}

	return account, nil
}

// RemovePendingAccount removes a pending account
func (m *Manager) RemovePendingAccount(alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.config.PendingAccounts[alias]; !exists {
		return models.ErrAccountNotFound
	}

	delete(m.config.PendingAccounts, alias)
	return m.saveLocked()
}

// ClearAllPendingAccounts removes all pending accounts
func (m *Manager) ClearAllPendingAccounts() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.PendingAccounts = make(map[string]*models.PendingAccount)
	return m.saveLocked()
}
