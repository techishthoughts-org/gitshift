package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/techishthoughts/gitshift/internal/models"
)

func testAccount(alias string) *models.Account {
	return models.NewAccount(alias, "Test User", alias+"@example.com", "/tmp/"+alias)
}

func TestManagerUsesCustomPathAndSecuresPersistence(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "private", "custom.yaml")
	manager := NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(configFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("config directory mode = %o, want 700", got)
	}

	before, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := before.Mode().Perm(); got != 0600 {
		t.Fatalf("config file mode = %o, want 600", got)
	}

	if err := manager.AddAccount(testAccount("work")); err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}
	after, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("Save() wrote in place; expected atomic file replacement")
	}

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(configFile), ".config-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}

	if err := os.Chmod(configFile, 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewManagerWithPath(configFile).Load(); err != nil {
		t.Fatalf("reload error = %v", err)
	}
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("loaded config mode = %o, want 600", got)
	}
}

func TestUpdateAccountPreservesFieldsAndDefault(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	manager := NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}

	validated := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	account := testAccount("work")
	account.APIEndpoint = "https://git.example.com/api"
	account.TokenPath = "/tokens/work"
	account.Status = models.AccountStatusIsolated
	account.IsolationLevel = models.IsolationLevelStrict
	account.LastValidation = &validated
	account.ValidationErrors = []string{"old warning"}
	account.AccountMetadata = map[string]string{"team": "platform"}
	if err := manager.AddAccount(account); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddAccount(testAccount("personal")); err != nil {
		t.Fatal(err)
	}

	updated := *account
	updated.Name = "Updated User"
	if err := manager.UpdateAccount(&updated); err != nil {
		t.Fatalf("UpdateAccount() error = %v", err)
	}

	reloaded := NewManagerWithPath(configFile)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.GetCurrentAccount()
	if err != nil {
		t.Fatal(err)
	}
	if got.Alias != "work" || !got.IsDefault {
		t.Fatalf("current/default account = %#v, want work default", got)
	}
	if got.APIEndpoint != account.APIEndpoint || got.TokenPath != account.TokenPath ||
		got.Status != account.Status || got.IsolationLevel != account.IsolationLevel ||
		got.LastValidation == nil || !got.LastValidation.Equal(validated) ||
		got.AccountMetadata["team"] != "platform" || len(got.ValidationErrors) != 1 {
		t.Fatalf("fields were not preserved: %#v", got)
	}
}

func TestReplaceAccountsSetsOneDefaultAndCurrent(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	manager := NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}

	first := testAccount("first")
	second := testAccount("second")
	pending := models.NewPendingAccount("pending", "user", "ssh", 7, []string{"email"}, nil)
	if err := manager.ReplaceAccounts([]*models.Account{first, second}, []*models.PendingAccount{pending}); err != nil {
		t.Fatalf("ReplaceAccounts() error = %v", err)
	}

	if manager.GetConfig().CurrentAccount != "first" || !first.IsDefault || second.IsDefault {
		t.Fatalf("default invariants not maintained: current=%q first=%t second=%t", manager.GetConfig().CurrentAccount, first.IsDefault, second.IsDefault)
	}
	if len(manager.ListAccounts()) != 2 || len(manager.ListPendingAccounts()) != 1 {
		t.Fatal("replacement account state was not persisted")
	}
}
