package models

import (
	"time"
)

// RefactoredCoreAccount contains the essential account data
type RefactoredCoreAccount struct {
	Alias          string `json:"alias" yaml:"alias" validate:"required"`
	Name           string `json:"name" yaml:"name" validate:"required"`
	Email          string `json:"email" yaml:"email" validate:"required,email"`
	GitHubUsername string `json:"github_username" yaml:"github_username" validate:"required"`
}

// RefactoredAccountSSHConfig handles SSH-specific configuration
type RefactoredAccountSSHConfig struct {
	KeyPath    string `json:"key_path,omitempty" yaml:"key_path,omitempty"`
	KeyType    string `json:"key_type,omitempty" yaml:"key_type,omitempty"`
	KeyExists  bool   `json:"key_exists" yaml:"key_exists"`
	IsVerified bool   `json:"is_verified" yaml:"is_verified"`
}

// RefactoredAccountMetadata contains metadata about the account
type RefactoredAccountMetadata struct {
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at" yaml:"created_at"`
	LastUsed    *time.Time `json:"last_used,omitempty" yaml:"last_used,omitempty"`
	UsageCount  int        `json:"usage_count" yaml:"usage_count"`
	Source      string     `json:"source,omitempty" yaml:"source,omitempty"`
	IsDefault   bool       `json:"is_default" yaml:"is_default"`
}

// RefactoredAccountState represents the current state of an account
type RefactoredAccountState struct {
	Status        RefactoredStatusType `json:"status" yaml:"status"`
	IsActive      bool                 `json:"is_active" yaml:"is_active"`
	LastValidated *time.Time           `json:"last_validated,omitempty" yaml:"last_validated,omitempty"`
	Issues        []string             `json:"issues,omitempty" yaml:"issues,omitempty"`
}

// RefactoredStatusType defines possible account states
type RefactoredStatusType string

const (
	RefactoredStatusActive   RefactoredStatusType = "active"
	RefactoredStatusPending  RefactoredStatusType = "pending"
	RefactoredStatusDisabled RefactoredStatusType = "disabled"
	RefactoredStatusUnknown  RefactoredStatusType = "unknown"
)

// RefactoredAccount represents a complete GitHub account configuration
// This is the composed type that brings everything together
// Note: This is a proposed refactoring - not currently used to avoid conflicts
type RefactoredAccount struct {
	Core     RefactoredCoreAccount      `json:"core" yaml:"core"`
	SSH      RefactoredAccountSSHConfig `json:"ssh" yaml:"ssh"`
	Metadata RefactoredAccountMetadata  `json:"metadata" yaml:"metadata"`
	State    RefactoredAccountState     `json:"state" yaml:"state"`
}

// NewRefactoredAccount creates a new account with sensible defaults
func NewRefactoredAccount(alias, name, email, githubUsername string) *RefactoredAccount {
	now := time.Now()
	return &RefactoredAccount{
		Core: RefactoredCoreAccount{
			Alias:          alias,
			Name:           name,
			Email:          email,
			GitHubUsername: githubUsername,
		},
		SSH: RefactoredAccountSSHConfig{
			KeyExists:  false,
			IsVerified: false,
		},
		Metadata: RefactoredAccountMetadata{
			CreatedAt:  now,
			UsageCount: 0,
			IsDefault:  false,
		},
		State: RefactoredAccountState{
			Status:   RefactoredStatusActive,
			IsActive: true,
		},
	}
}

// Getters for backward compatibility
func (a *RefactoredAccount) GetAlias() string          { return a.Core.Alias }
func (a *RefactoredAccount) GetName() string           { return a.Core.Name }
func (a *RefactoredAccount) GetEmail() string          { return a.Core.Email }
func (a *RefactoredAccount) GetGitHubUsername() string { return a.Core.GitHubUsername }
func (a *RefactoredAccount) GetSSHKeyPath() string     { return a.SSH.KeyPath }
func (a *RefactoredAccount) IsAccountActive() bool     { return a.State.IsActive }

// MarkAsUsedRefactored updates usage statistics
func (a *RefactoredAccount) MarkAsUsedRefactored() {
	now := time.Now()
	a.Metadata.LastUsed = &now
	a.Metadata.UsageCount++
}
