package config

import (
	"errors"
	"os"
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

func TestAddAccountsRestoresMemoryWhenSaveFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := NewManagerWithPath(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if err := m.AddAccount(importAccount("existing")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("# concurrent change\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.AddAccounts([]*models.Account{importAccount("new")}); !errors.Is(err, ErrConfigChanged) {
		t.Fatalf("error=%v", err)
	}
	if got := len(m.ListAccounts()); got != 1 {
		t.Fatalf("in-memory accounts=%d", got)
	}
	if _, err := m.GetAccount("existing"); err != nil {
		t.Fatal("original account was not restored")
	}
	if _, err := m.GetAccount("new"); !errors.Is(err, models.ErrAccountNotFound) {
		t.Fatalf("new account remained in memory: %v", err)
	}
}

func TestCustomConfigParentPermissionsArePreserved(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatal(err)
	}
	m := NewManagerWithPath(filepath.Join(parent, "config.yaml"))
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0755 {
		t.Fatalf("existing parent mode=%#o", got)
	}
}

func TestMissingCustomConfigParentIsCreatedPrivate(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "nested", "gitshift")
	m := NewManagerWithPath(filepath.Join(parent, "config.yaml"))
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0700 {
		t.Fatalf("created parent mode=%#o", got)
	}
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
