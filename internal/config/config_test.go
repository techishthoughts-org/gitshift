package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/techishthoughts/gitshift/internal/models"
)

func testAccount(alias string) *models.Account {
	return models.NewAccount(alias, "Test User", alias+"@example.com", "/tmp/"+alias)
}

func TestManagerRejectsStaleWriter(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	first := NewManagerWithPath(configFile)
	if err := first.Load(); err != nil {
		t.Fatal(err)
	}
	second := NewManagerWithPath(configFile)
	if err := second.Load(); err != nil {
		t.Fatal(err)
	}

	if err := first.AddAccount(testAccount("first")); err != nil {
		t.Fatal(err)
	}
	if err := second.AddAccount(testAccount("second")); !errors.Is(err, ErrConfigChanged) {
		t.Fatalf("stale AddAccount() error = %v, want ErrConfigChanged", err)
	}

	reloaded := NewManagerWithPath(configFile)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.GetAccount("first"); err != nil {
		t.Fatalf("winning update was lost: %v", err)
	}
	if _, err := reloaded.GetAccount("second"); !errors.Is(err, models.ErrAccountNotFound) {
		t.Fatalf("stale update was persisted: %v", err)
	}
}

func TestConfigLockReleaseChecksOwnership(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	manager := NewManagerWithPath(configFile)
	if err := manager.ensureConfigDirectory(); err != nil {
		t.Fatal(err)
	}
	release, err := manager.acquireSaveLock()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := configFile + ".lock"
	if err := os.WriteFile(lockPath, []byte("different-owner"), 0600); err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("release removed a lock owned by another writer: %v", err)
	}
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

func TestLoadReadOnlyDoesNotCreateOrMutateConfiguration(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "config.yaml")
	manager := NewManagerWithPath(missing)
	if err := manager.LoadReadOnly(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !os.IsNotExist(err) {
		t.Fatalf("read-only load created directory: %v", err)
	}

	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	data := []byte("accounts: {}\npending_accounts: {}\nconfig_version: 1.0.0\n")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	manager = NewManagerWithPath(configFile)
	if err := manager.LoadReadOnly(); err != nil {
		t.Fatal(err)
	}
	if manager.GetConfig().ConfigVersion != "1.1.0" {
		t.Fatalf("legacy version was not migrated in memory: %q", manager.GetConfig().ConfigVersion)
	}
	after, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm() != after.Mode().Perm() || before.ModTime() != after.ModTime() {
		t.Fatalf("read-only load mutated file: before=%v after=%v", before, after)
	}
}

func TestUnsupportedConfigVersionsAreRejectedWithoutMutation(t *testing.T) {
	for _, version := range []string{"2.0.0", "banana", "1.1"} {
		for _, readOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/read-only=%t", version, readOnly), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				data := []byte("accounts: {}\npending_accounts: {}\nconfig_version: \"" + version + "\"\n")
				if err := os.WriteFile(path, data, 0644); err != nil {
					t.Fatal(err)
				}
				fixed := time.Unix(1_700_000_000, 0)
				if err := os.Chtimes(path, fixed, fixed); err != nil {
					t.Fatal(err)
				}
				before, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				manager := NewManagerWithPath(path)
				if readOnly {
					err = manager.LoadReadOnly()
				} else {
					err = manager.Load()
				}
				if err == nil || !strings.Contains(err.Error(), "unsupported configuration version") {
					t.Fatalf("error = %v", err)
				}
				afterData, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				after, statErr := os.Stat(path)
				if statErr != nil {
					t.Fatal(statErr)
				}
				if !bytes.Equal(afterData, data) || after.Mode().Perm() != before.Mode().Perm() || !after.ModTime().Equal(before.ModTime()) {
					t.Fatalf("rejected config mutated: bytes=%t mode=%o/%o mtime=%v/%v", bytes.Equal(afterData, data), before.Mode().Perm(), after.Mode().Perm(), before.ModTime(), after.ModTime())
				}
			})
		}
	}
}

func TestLoadMigratesOnlyKnownLegacyVersions(t *testing.T) {
	for _, versionLine := range []string{"", "config_version: \"1.0.0\"\n"} {
		t.Run(versionLine, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte("accounts: {}\npending_accounts: {}\n"+versionLine), 0600); err != nil {
				t.Fatal(err)
			}
			manager := NewManagerWithPath(path)
			if err := manager.Load(); err != nil {
				t.Fatal(err)
			}
			if manager.GetConfig().ConfigVersion != "1.1.0" {
				t.Fatalf("version = %q", manager.GetConfig().ConfigVersion)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "config_version: 1.1.0") {
				t.Fatalf("migration not persisted: %s", data)
			}
		})
	}
}

func TestLoadAcceptsCurrentConfigVersionWithoutRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("accounts: {}\npending_accounts: {}\nconfig_version: \"1.1.0\"\n")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	fixed := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(path, fixed, fixed); err != nil {
		t.Fatal(err)
	}
	manager := NewManagerWithPath(path)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	afterData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterData, data) || !after.ModTime().Equal(fixed) {
		t.Fatalf("current config version was rewritten")
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

func TestAutoSwitchRulesCanonicalizeReplaceAndCascade(t *testing.T) {
	root := t.TempDir()
	configFile := filepath.Join(root, "config.yaml")
	manager := NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddAccount(testAccount("personal")); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddAccount(testAccount("work")); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "projects")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	rule, err := manager.AddAutoSwitchRule(dir, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(rule.Path) || filepath.Base(rule.Path) != "projects" || rule.Account != "personal" || rule.CreatedAt.IsZero() {
		t.Fatalf("unexpected rule: %#v", rule)
	}
	replaced, err := manager.AddAutoSwitchRule(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	if replaced.CreatedAt != rule.CreatedAt || len(manager.ListAutoSwitchRules()) != 1 {
		t.Fatalf("replace was not stable: %#v", replaced)
	}
	if err := manager.RemoveAccount("work"); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAutoSwitchRules()) != 0 {
		t.Fatal("account removal left dangling rule")
	}
	if _, err := manager.AddAutoSwitchRule(dir, "missing"); !errors.Is(err, models.ErrAccountNotFound) {
		t.Fatalf("missing alias error = %v", err)
	}
	if _, err := manager.AddAutoSwitchRule(dir, "personal"); err != nil {
		t.Fatal(err)
	}
	if err := manager.RemoveAutoSwitchRule(dir); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAutoSwitchRules()) != 0 {
		t.Fatal("explicit rule removal failed")
	}
	staleDir := filepath.Join(root, "stale")
	if err := os.Mkdir(staleDir, 0700); err != nil {
		t.Fatal(err)
	}
	staleRule, err := manager.AddAutoSwitchRule(staleDir, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(staleDir); err != nil {
		t.Fatal(err)
	}
	if err := manager.RemoveAutoSwitchRule(staleRule.Path); err != nil {
		t.Fatalf("remove stale rule: %v", err)
	}
	if _, err := manager.AddAutoSwitchRule(dir, "personal"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ReplaceAccounts([]*models.Account{testAccount("fresh")}, nil); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAutoSwitchRules()) != 0 {
		t.Fatal("account replacement left dangling rule")
	}
}

func TestSwitchLockIsExclusive(t *testing.T) {
	manager := NewManagerWithPath(filepath.Join(t.TempDir(), "config.yaml"))
	release, err := manager.AcquireSwitchLock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := manager.AcquireSwitchLock(); err == nil {
		t.Fatal("second switch lock unexpectedly succeeded")
	}
}
