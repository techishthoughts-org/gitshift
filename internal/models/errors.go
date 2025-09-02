package models

import "errors"

// Common errors for the application
var (
	// Account validation errors
	ErrInvalidAlias                = errors.New("account alias cannot be empty")
	ErrInvalidName                 = errors.New("account name cannot be empty")
	ErrInvalidEmail                = errors.New("account email cannot be empty")
	ErrInvalidGitHubUsername       = errors.New("GitHub username cannot be empty")
	ErrInvalidEmailFormat          = errors.New("invalid email format")
	ErrInvalidGitHubUsernameFormat = errors.New("invalid GitHub username format")

	// Account management errors
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountExists    = errors.New("account already exists")
	ErrNoAccountsFound  = errors.New("no accounts configured")
	ErrNoDefaultAccount = errors.New("no default account set")

	// Configuration errors
	ErrConfigNotFound  = errors.New("configuration file not found")
	ErrConfigCorrupted = errors.New("configuration file is corrupted")
	ErrInvalidConfig   = errors.New("invalid configuration format")

	// Git errors
	ErrGitNotFound     = errors.New("git command not found")
	ErrGitConfigFailed = errors.New("failed to configure git")
	ErrNotGitRepo      = errors.New("not a git repository")

	// SSH errors
	ErrSSHKeyNotFound = errors.New("SSH key file not found")
	ErrSSHKeyInvalid  = errors.New("SSH key file is invalid")

	// Project errors
	ErrNotInProject         = errors.New("not in a project directory")
	ErrProjectConfigInvalid = errors.New("invalid project configuration")
)
