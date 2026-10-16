// Package tui contains the testable Bubble Tea application for GitShift.
// It deliberately knows nothing about Cobra, files, subprocesses, or secrets.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

type Account struct {
	Alias, Name, Email, Username, Platform, Domain, SSHKeyPath, GPGKeyID, Description string
	Current, Default                                                                  bool
}

type Rule struct{ Path, Account string }

type Discovery struct {
	Identity, Alias, Name, Email, Username, Platform, Domain, SSHKeyPath, Source string
	GPGKeyID, GPGKeyFingerprint, GPGKeyType                                      string
	GPGKeySize, Confidence                                                       int
	GPGEnabled                                                                   bool
}

type Snapshot struct {
	Accounts []Account
	Rules    []Rule
	Current  string
}

type AccountInput struct {
	OriginalAlias, Alias, Name, Email, Username, Platform, Domain, SSHKeyPath, Description string
}

func (in AccountInput) Validate() error {
	account := models.NewAccount(strings.TrimSpace(in.Alias), strings.TrimSpace(in.Name), strings.TrimSpace(in.Email), strings.TrimSpace(in.SSHKeyPath))
	account.Platform = strings.ToLower(strings.TrimSpace(in.Platform))
	account.Domain = strings.TrimSpace(in.Domain)
	account.SetUsername(strings.TrimPrefix(strings.TrimSpace(in.Username), "@"))
	account.Description = strings.TrimSpace(in.Description)
	if err := account.Validate(); err != nil {
		return err
	}
	if in.OriginalAlias != "" && in.Alias != in.OriginalAlias {
		return fmt.Errorf("account aliases cannot be renamed; create a new account instead")
	}
	return nil
}

// Service is the only side-effect boundary used by the UI. Implementations
// must return copied DTOs, never mutable configuration pointers.
type Service interface {
	Load(context.Context) (Snapshot, error)
	SaveAccount(context.Context, AccountInput) error
	RemoveAccount(context.Context, string) error
	SwitchAccount(context.Context, string) error
	AddRule(context.Context, Rule) error
	RemoveRule(context.Context, string) error
	Discover(context.Context) ([]Discovery, error)
	Import(context.Context, []Discovery) error
}
