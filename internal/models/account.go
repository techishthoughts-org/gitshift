package models

import (
	"regexp"
	"strings"
	"time"
)

// Account represents a GitHub account configuration with complete isolation support.
// It provides multi-account management with SSH key isolation, token management,
// and comprehensive validation capabilities.
type Account struct {
	// Alias is a unique identifier for the account (e.g., "work", "personal")
	Alias string `json:"alias" yaml:"alias" mapstructure:"alias" validate:"required"`

	// Name is the Git user.name
	Name string `json:"name" yaml:"name" mapstructure:"name" validate:"required"`

	// Email is the Git user.email
	Email string `json:"email" yaml:"email" mapstructure:"email" validate:"required,email"`

	// SSHKeyPath is the path to the SSH private key file
	SSHKeyPath string `json:"ssh_key_path" yaml:"ssh_key_path" mapstructure:"ssh_key_path"`

	// GitHubUsername is the GitHub username (required for GitHub platform)
	// Deprecated: Use Username instead with Platform field
	GitHubUsername string `json:"github_username" yaml:"github_username" mapstructure:"github_username"`

	// Username is the platform-specific username (e.g., GitHub, GitLab username)
	Username string `json:"username,omitempty" yaml:"username,omitempty" mapstructure:"username"`

	// Platform is the Git hosting platform type (github, gitlab, bitbucket, custom)
	Platform string `json:"platform,omitempty" yaml:"platform,omitempty" mapstructure:"platform"`

	// Domain is the platform domain (e.g., "github.com", "gitlab.com", "custom-gitlab.company.com")
	// If empty, defaults to the standard domain for the platform
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty" mapstructure:"domain"`

	// APIEndpoint is the API endpoint URL for custom installations (optional)
	APIEndpoint string `json:"api_endpoint,omitempty" yaml:"api_endpoint,omitempty" mapstructure:"api_endpoint"`

	// Description is an optional description of the account
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`

	// IsDefault indicates if this is the default account
	IsDefault bool `json:"is_default" yaml:"is_default" mapstructure:"is_default"`

	// CreatedAt tracks when the account was added
	CreatedAt time.Time `json:"created_at" yaml:"created_at" mapstructure:"created_at"`

	// LastUsed tracks when the account was last used
	LastUsed *time.Time `json:"last_used,omitempty" yaml:"last_used,omitempty" mapstructure:"last_used"`

	// Status indicates the account status (active, pending, disabled)
	Status AccountStatus `json:"status" yaml:"status" mapstructure:"status"`

	// MissingFields tracks which required fields are missing for pending accounts
	MissingFields []string `json:"missing_fields,omitempty" yaml:"missing_fields,omitempty" mapstructure:"missing_fields"`

	// === ISOLATION ENHANCEMENTS ===

	// TokenPath is the path where the account's GitHub token is stored
	TokenPath string `json:"token_path,omitempty" yaml:"token_path,omitempty" mapstructure:"token_path"`

	// SSHSocketPath is the path to the isolated SSH agent socket for this account
	SSHSocketPath string `json:"ssh_socket_path,omitempty" yaml:"ssh_socket_path,omitempty" mapstructure:"ssh_socket_path"`

	// IsolationLevel defines the level of isolation for this account
	IsolationLevel IsolationLevel `json:"isolation_level" yaml:"isolation_level" mapstructure:"isolation_level"`

	// LastValidation tracks when the account was last validated
	LastValidation *time.Time `json:"last_validation,omitempty" yaml:"last_validation,omitempty" mapstructure:"last_validation"`

	// ValidationErrors tracks any validation errors encountered
	ValidationErrors []string `json:"validation_errors,omitempty" yaml:"validation_errors,omitempty" mapstructure:"validation_errors"`

	// IsolationMetadata contains account-specific isolation settings
	IsolationMetadata *IsolationMetadata `json:"isolation_metadata,omitempty" yaml:"isolation_metadata,omitempty" mapstructure:"isolation_metadata"`

	// AccountMetadata stores additional account-specific data
	AccountMetadata map[string]string `json:"account_metadata,omitempty" yaml:"account_metadata,omitempty" mapstructure:"account_metadata"`
}

// AccountStatus represents the status of an account
type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusPending  AccountStatus = "pending"
	AccountStatusDisabled AccountStatus = "disabled"
	AccountStatusIsolated AccountStatus = "isolated" // New status for accounts with full isolation
)

// IsolationLevel represents the level of isolation for an account
type IsolationLevel string

const (
	IsolationLevelNone     IsolationLevel = "none"     // No isolation (legacy mode)
	IsolationLevelBasic    IsolationLevel = "basic"    // Basic SSH key isolation
	IsolationLevelStandard IsolationLevel = "standard" // SSH + Token isolation
	IsolationLevelStrict   IsolationLevel = "strict"   // Full isolation with validation
	IsolationLevelComplete IsolationLevel = "complete" // Maximum isolation with containers
)

// IsolationMetadata contains isolation-specific configuration
type IsolationMetadata struct {
	// SSH isolation settings
	SSHIsolation *SSHIsolationSettings `json:"ssh_isolation,omitempty" yaml:"ssh_isolation,omitempty"`

	// Token isolation settings
	TokenIsolation *TokenIsolationSettings `json:"token_isolation,omitempty" yaml:"token_isolation,omitempty"`

	// Git isolation settings
	GitIsolation *GitIsolationSettings `json:"git_isolation,omitempty" yaml:"git_isolation,omitempty"`

	// Environment isolation settings
	EnvironmentIsolation *EnvironmentIsolationSettings `json:"environment_isolation,omitempty" yaml:"environment_isolation,omitempty"`
}

// SSHIsolationSettings defines SSH-specific isolation parameters
type SSHIsolationSettings struct {
	UseIsolatedAgent    bool   `json:"use_isolated_agent" yaml:"use_isolated_agent"`
	SocketPath          string `json:"socket_path,omitempty" yaml:"socket_path,omitempty"`
	ForceIdentitiesOnly bool   `json:"force_identities_only" yaml:"force_identities_only"`
	AgentTimeout        int    `json:"agent_timeout" yaml:"agent_timeout"` // in seconds
	AutoCleanup         bool   `json:"auto_cleanup" yaml:"auto_cleanup"`
}

// TokenIsolationSettings defines token-specific isolation parameters
type TokenIsolationSettings struct {
	UseEncryptedStorage bool   `json:"use_encrypted_storage" yaml:"use_encrypted_storage"`
	StoragePath         string `json:"storage_path,omitempty" yaml:"storage_path,omitempty"`
	AutoValidation      bool   `json:"auto_validation" yaml:"auto_validation"`
	ValidationInterval  int    `json:"validation_interval" yaml:"validation_interval"` // in minutes
	StrictValidation    bool   `json:"strict_validation" yaml:"strict_validation"`
}

// GitIsolationSettings defines Git-specific isolation parameters
type GitIsolationSettings struct {
	UseLocalConfig      bool              `json:"use_local_config" yaml:"use_local_config"`
	CustomGitConfig     map[string]string `json:"custom_git_config,omitempty" yaml:"custom_git_config,omitempty"`
	IsolateSSHCommand   bool              `json:"isolate_ssh_command" yaml:"isolate_ssh_command"`
	UseCustomSSHCommand string            `json:"use_custom_ssh_command,omitempty" yaml:"use_custom_ssh_command,omitempty"`
}

// EnvironmentIsolationSettings defines environment-specific isolation parameters
type EnvironmentIsolationSettings struct {
	IsolateEnvironment  bool              `json:"isolate_environment" yaml:"isolate_environment"`
	CustomEnvironment   map[string]string `json:"custom_environment,omitempty" yaml:"custom_environment,omitempty"`
	ClearEnvironment    bool              `json:"clear_environment" yaml:"clear_environment"`
	PreserveEnvironment []string          `json:"preserve_environment,omitempty" yaml:"preserve_environment,omitempty"`
}

// PendingAccount represents an account that needs manual completion
type PendingAccount struct {
	// Alias is a unique identifier for the account
	Alias string `json:"alias" yaml:"alias" mapstructure:"alias"`

	// GitHubUsername is the GitHub username (if available)
	GitHubUsername string `json:"github_username,omitempty" yaml:"github_username,omitempty" mapstructure:"github_username"`

	// Partial data that was discovered
	PartialData map[string]string `json:"partial_data,omitempty" yaml:"partial_data,omitempty" mapstructure:"partial_data"`

	// MissingFields lists what needs to be completed
	MissingFields []string `json:"missing_fields" yaml:"missing_fields" mapstructure:"missing_fields"`

	// Source indicates where this account was discovered
	Source string `json:"source" yaml:"source" mapstructure:"source"`

	// Confidence level from discovery
	Confidence int `json:"confidence" yaml:"confidence" mapstructure:"confidence"`

	// CreatedAt tracks when the pending account was created
	CreatedAt time.Time `json:"created_at" yaml:"created_at" mapstructure:"created_at"`
}

// Config represents the entire application configuration including
// all account definitions, pending accounts, and global settings.
type Config struct {
	// Accounts is a map of alias to account configurations
	Accounts map[string]*Account `json:"accounts" yaml:"accounts" mapstructure:"accounts"`

	// PendingAccounts stores accounts that need manual completion
	PendingAccounts map[string]*PendingAccount `json:"pending_accounts,omitempty" yaml:"pending_accounts,omitempty" mapstructure:"pending_accounts"`

	// CurrentAccount is the alias of the currently active account
	CurrentAccount string `json:"current_account,omitempty" yaml:"current_account,omitempty" mapstructure:"current_account"`

	// GlobalGitConfig determines whether to set global git config or local only
	GlobalGitConfig bool `json:"global_git_config" yaml:"global_git_config" mapstructure:"global_git_config"`

	// AutoDetect enables automatic account detection based on folder configuration
	AutoDetect bool `json:"auto_detect" yaml:"auto_detect" mapstructure:"auto_detect"`

	// ConfigVersion for future migrations
	ConfigVersion string `json:"config_version" yaml:"config_version" mapstructure:"config_version"`
}

// ProjectConfig represents the project-specific configuration
type ProjectConfig struct {
	// Account is the alias of the account to use for this project
	Account string `json:"account" yaml:"account" validate:"required"`

	// CreatedAt tracks when the project config was created
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// NewAccount creates a new account with the current timestamp and default isolation settings.
// It initializes standard isolation metadata for SSH, token, Git, and environment isolation.
func NewAccount(alias, name, email, sshKeyPath string) *Account {
	return &Account{
		Alias:           alias,
		Name:            name,
		Email:           email,
		SSHKeyPath:      sshKeyPath,
		IsDefault:       false,
		Status:          AccountStatusActive,
		CreatedAt:       time.Now(),
		IsolationLevel:  IsolationLevelStandard, // Default to standard isolation
		AccountMetadata: make(map[string]string),
		IsolationMetadata: &IsolationMetadata{
			SSHIsolation: &SSHIsolationSettings{
				UseIsolatedAgent:    true,
				ForceIdentitiesOnly: true,
				AgentTimeout:        3600, // 1 hour
				AutoCleanup:         true,
			},
			TokenIsolation: &TokenIsolationSettings{
				UseEncryptedStorage: true,
				AutoValidation:      true,
				ValidationInterval:  60, // 1 hour
				StrictValidation:    true,
			},
			GitIsolation: &GitIsolationSettings{
				UseLocalConfig:    true,
				IsolateSSHCommand: true,
				CustomGitConfig:   make(map[string]string),
			},
			EnvironmentIsolation: &EnvironmentIsolationSettings{
				IsolateEnvironment:  false, // Default to non-isolated environment
				CustomEnvironment:   make(map[string]string),
				ClearEnvironment:    false,
				PreserveEnvironment: []string{"HOME", "USER", "PATH"},
			},
		},
	}
}

// NewIsolatedAccount creates a new account with complete isolation
func NewIsolatedAccount(alias, name, email, sshKeyPath, githubUsername string) *Account {
	account := NewAccount(alias, name, email, sshKeyPath)
	account.GitHubUsername = githubUsername
	account.IsolationLevel = IsolationLevelComplete
	account.Status = AccountStatusIsolated

	// Enhanced isolation settings
	account.IsolationMetadata.SSHIsolation.UseIsolatedAgent = true
	account.IsolationMetadata.TokenIsolation.StrictValidation = true
	account.IsolationMetadata.EnvironmentIsolation.IsolateEnvironment = true
	account.IsolationMetadata.EnvironmentIsolation.ClearEnvironment = true

	return account
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Accounts:        make(map[string]*Account),
		PendingAccounts: make(map[string]*PendingAccount),
		GlobalGitConfig: true, // Always use global Git config by default
		AutoDetect:      true,
		ConfigVersion:   "1.0.0",
	}
}

// NewProjectConfig creates a new project configuration
func NewProjectConfig(accountAlias string) *ProjectConfig {
	return &ProjectConfig{
		Account:   accountAlias,
		CreatedAt: time.Now(),
	}
}

// Validate checks if the account configuration is valid by verifying
// required fields, email format, and GitHub username format.
func (a *Account) Validate() error {
	if a.Alias == "" {
		return ErrInvalidAlias
	}

	// For discovered accounts, we need at least one of: Name+Email OR GitHubUsername
	hasBasicInfo := a.Name != "" && a.Email != ""
	hasGitHubInfo := a.GitHubUsername != ""

	if !hasBasicInfo && !hasGitHubInfo {
		return ErrInvalidConfig // Need at least basic info or GitHub info
	}

	// If we have email, validate its format
	if a.Email != "" && !isValidEmail(a.Email) {
		return ErrInvalidEmailFormat
	}

	// If we have GitHub username, validate its format
	if a.GitHubUsername != "" && !isValidGitHubUsername(a.GitHubUsername) {
		return ErrInvalidGitHubUsernameFormat
	}

	return nil
}

// MarkAsUsed updates the LastUsed timestamp
func (a *Account) MarkAsUsed() {
	now := time.Now()
	a.LastUsed = &now
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isValidGitHubUsername validates GitHub username format
func isValidGitHubUsername(username string) bool {
	if username == "" {
		return false
	}

	// GitHub username rules:
	// - Can only contain alphanumeric characters and hyphens
	// - Cannot start or end with a hyphen
	// - Cannot have consecutive hyphens
	// - Maximum 39 characters

	if len(username) > 39 || len(username) < 1 {
		return false
	}

	if strings.HasPrefix(username, "-") || strings.HasSuffix(username, "-") {
		return false
	}

	if strings.Contains(username, "--") {
		return false
	}

	// Must contain only alphanumeric characters and hyphens
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	return usernameRegex.MatchString(username)
}

// NewPendingAccount creates a new pending account
func NewPendingAccount(alias, githubUsername, source string, confidence int, missingFields []string, partialData map[string]string) *PendingAccount {
	return &PendingAccount{
		Alias:          alias,
		GitHubUsername: githubUsername,
		Source:         source,
		Confidence:     confidence,
		MissingFields:  missingFields,
		PartialData:    partialData,
		CreatedAt:      time.Now(),
	}
}

// IsPending returns true if the account is in pending status
func (a *Account) IsPending() bool {
	return a.Status == AccountStatusPending
}

// IsIsolated returns true if the account has isolation enabled
func (a *Account) IsIsolated() bool {
	return a.Status == AccountStatusIsolated || a.IsolationLevel != IsolationLevelNone
}

// RequiresSSHIsolation returns true if the account requires SSH isolation
func (a *Account) RequiresSSHIsolation() bool {
	return a.IsolationMetadata != nil &&
		a.IsolationMetadata.SSHIsolation != nil &&
		a.IsolationMetadata.SSHIsolation.UseIsolatedAgent
}

// RequiresTokenIsolation returns true if the account requires token isolation
func (a *Account) RequiresTokenIsolation() bool {
	return a.IsolationMetadata != nil &&
		a.IsolationMetadata.TokenIsolation != nil &&
		a.IsolationMetadata.TokenIsolation.UseEncryptedStorage
}

// GetSSHSocketPath returns the SSH socket path for this account
func (a *Account) GetSSHSocketPath() string {
	if a.SSHSocketPath != "" {
		return a.SSHSocketPath
	}
	if a.IsolationMetadata != nil &&
		a.IsolationMetadata.SSHIsolation != nil &&
		a.IsolationMetadata.SSHIsolation.SocketPath != "" {
		return a.IsolationMetadata.SSHIsolation.SocketPath
	}
	return ""
}

// SetLastValidation updates the last validation timestamp
func (a *Account) SetLastValidation(validationTime time.Time, errors []string) {
	a.LastValidation = &validationTime
	a.ValidationErrors = errors
	if len(errors) == 0 {
		// Clear previous validation errors if validation passed
		a.ValidationErrors = nil
	}
}

// IsValidationRequired checks if account validation is required
func (a *Account) IsValidationRequired() bool {
	if a.LastValidation == nil {
		return true
	}

	if a.IsolationMetadata != nil &&
		a.IsolationMetadata.TokenIsolation != nil &&
		a.IsolationMetadata.TokenIsolation.AutoValidation {

		validationInterval := time.Duration(a.IsolationMetadata.TokenIsolation.ValidationInterval) * time.Minute
		return time.Since(*a.LastValidation) > validationInterval
	}

	// Default validation interval: 24 hours
	return time.Since(*a.LastValidation) > 24*time.Hour
}

// GetIsolationLevel returns the isolation level with fallback to default
func (a *Account) GetIsolationLevel() IsolationLevel {
	if a.IsolationLevel == "" {
		return IsolationLevelNone // Legacy accounts
	}
	return a.IsolationLevel
}

// UpdateAccountMetadata updates or adds account metadata
func (a *Account) UpdateAccountMetadata(key, value string) {
	if a.AccountMetadata == nil {
		a.AccountMetadata = make(map[string]string)
	}
	a.AccountMetadata[key] = value
}

// GetAccountMetadata retrieves account metadata
func (a *Account) GetAccountMetadata(key string) (string, bool) {
	if a.AccountMetadata == nil {
		return "", false
	}
	value, exists := a.AccountMetadata[key]
	return value, exists
}

// GetMissingFields returns the list of missing fields for pending accounts
func (a *Account) GetMissingFields() []string {
	if a.Status != AccountStatusPending {
		return nil
	}
	return a.MissingFields
}

// GetPlatform returns the platform type, defaulting to GitHub for backward compatibility
func (a *Account) GetPlatform() string {
	if a.Platform == "" {
		return "github" // Default to GitHub for legacy accounts
	}
	return a.Platform
}

// GetDomain returns the platform domain, defaulting to platform-specific domains
func (a *Account) GetDomain() string {
	if a.Domain != "" {
		return a.Domain
	}

	// Return default domain based on platform
	switch a.GetPlatform() {
	case "github":
		return "github.com"
	case "gitlab":
		return "gitlab.com"
	case "bitbucket":
		return "bitbucket.org"
	default:
		return ""
	}
}

// GetUsername returns the username, falling back to GitHubUsername for backward compatibility
func (a *Account) GetUsername() string {
	if a.Username != "" {
		return a.Username
	}
	// Fallback to GitHubUsername for backward compatibility
	return a.GitHubUsername
}

// SetUsername sets the username and updates GitHubUsername for backward compatibility
func (a *Account) SetUsername(username string) {
	a.Username = username
	// Keep GitHubUsername in sync for backward compatibility
	if a.GetPlatform() == "github" {
		a.GitHubUsername = username
	}
}

// IsPlatformSupported checks if the account's platform is supported
func (a *Account) IsPlatformSupported() bool {
	platform := a.GetPlatform()
	return platform == "github" || platform == "gitlab" || platform == "bitbucket"
}
