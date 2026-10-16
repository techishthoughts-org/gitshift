package config

import (
	"path/filepath"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func importAccount(alias string) *models.Account {
	a := models.NewAccount(alias, "User", "user@example.com", "")
	a.Platform = "github"
	a.Domain = "github.com"
	a.SetUsername("user")
	return a
}

func TestAddAccountsIsAtomicOnValidationFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := NewManagerWithPath(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	bad := importAccount("bad alias")
	if err := m.AddAccounts([]*models.Account{importAccount("good"), bad}); err == nil {
		t.Fatal("expected validation failure")
	}
	reloaded := NewManagerWithPath(path)
	if err := reloaded.LoadReadOnly(); err != nil {
		t.Fatal(err)
	}
	if len(reloaded.ListAccounts()) != 0 {
		t.Fatal("partial batch was persisted")
	}
}

func TestAddAccountsRejectsCaseInsensitiveCollisionWithoutWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := NewManagerWithPath(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if err := m.AddAccount(importAccount("Work")); err != nil {
		t.Fatal(err)
	}
	if err := m.AddAccounts([]*models.Account{importAccount("work"), importAccount("personal")}); err == nil {
		t.Fatal("expected collision")
	}
	reloaded := NewManagerWithPath(path)
	if err := reloaded.LoadReadOnly(); err != nil {
		t.Fatal(err)
	}
	if got := len(reloaded.ListAccounts()); got != 1 {
		t.Fatalf("accounts=%d", got)
	}
}
