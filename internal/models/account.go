package models

import (
	"regexp"
	"strings"
	"time"
)

// Account represents a GitHub account configuration
type Account struct {
	// Alias is a unique identifier for the account (e.g., "work", "personal")
	Alias string `json:"alias" yaml:"alias" mapstructure:"alias" validate:"required"`

	// Name is the Git user.name
	Name string `json:"name" yaml:"name" mapstructure:"name" validate:"required"`

	// Email is the Git user.email
	Email string `json:"email" yaml:"email" mapstructure:"email" validate:"required,email"`

	// SSHKeyPath is the path to the SSH private key file
	SSHKeyPath string `json:"ssh_key_path" yaml:"ssh_key_path" mapstructure:"ssh_key_path"`

	// GitHubUsername is the GitHub username (required)
	GitHubUsername string `json:"github_username" yaml:"github_username" mapstructure:"github_username" validate:"required"`

	// Description is an optional description of the account
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`

	// IsDefault indicates if this is the default account
	IsDefault bool `json:"is_default" yaml:"is_default" mapstructure:"is_default"`

	// CreatedAt tracks when the account was added
	CreatedAt time.Time `json:"created_at" yaml:"created_at" mapstructure:"created_at"`

	// LastUsed tracks when the account was last used
	LastUsed *time.Time `json:"last_used,omitempty" yaml:"last_used,omitempty" mapstructure:"last_used"`
}

// Config represents the entire application configuration
type Config struct {
	// Accounts is a map of alias to account configurations
	Accounts map[string]*Account `json:"accounts" yaml:"accounts" mapstructure:"accounts"`

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

// NewAccount creates a new account with the current timestamp
func NewAccount(alias, name, email, sshKeyPath string) *Account {
	return &Account{
		Alias:      alias,
		Name:       name,
		Email:      email,
		SSHKeyPath: sshKeyPath,
		IsDefault:  false,
		CreatedAt:  time.Now(),
	}
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Accounts:        make(map[string]*Account),
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

// Validate checks if the account configuration is valid
func (a *Account) Validate() error {
	if a.Alias == "" {
		return ErrInvalidAlias
	}
	if a.Name == "" {
		return ErrInvalidName
	}
	if a.Email == "" {
		return ErrInvalidEmail
	}
	if a.GitHubUsername == "" {
		return ErrInvalidGitHubUsername
	}

	// Validate email format
	if !isValidEmail(a.Email) {
		return ErrInvalidEmailFormat
	}

	// Validate GitHub username format
	if !isValidGitHubUsername(a.GitHubUsername) {
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
