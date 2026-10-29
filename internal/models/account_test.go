package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeSSHKeyPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got, err := NormalizeSSHKeyPath("~/.ssh/id_work")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(os.Getenv("HOME"), ".ssh", "id_work")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := NormalizeSSHKeyPath("/tmp/key\nHost attacker"); err == nil {
		t.Fatal("expected control-character path to be rejected")
	}
}

func TestAccountValidatePlatformFields(t *testing.T) {
	a := NewAccount("work", "Work User", "work@example.com", "")
	a.Platform, a.Domain, a.Username = "gitlab", "gitlab.example.com", "work_user"
	if err := a.Validate(); err != nil {
		t.Fatalf("valid GitLab account rejected: %v", err)
	}
	a.Domain = "gitlab.example.com\nHost attacker"
	if err := a.Validate(); err == nil {
		t.Fatal("expected injected domain to be rejected")
	}
}

func TestAccountValidateRejectsUnsafeAliases(t *testing.T) {
	for _, alias := range []string{"../outside", "nested/key", `nested\key`, ".", "..", "has space", "-option"} {
		t.Run(alias, func(t *testing.T) {
			account := NewAccount(alias, "Test User", "test@example.com", "")
			if err := account.Validate(); err != ErrInvalidAlias {
				t.Fatalf("Validate() error = %v, want ErrInvalidAlias", err)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	for _, alias := range []string{"personal", "work.github", "team_2", "A-1"} {
		if err := ValidateAlias(alias); err != nil {
			t.Errorf("ValidateAlias(%q) error = %v", alias, err)
		}
	}
	for _, alias := range []string{"", ".", "..", "has space", "-option", "nested/key"} {
		if err := ValidateAlias(alias); err != ErrInvalidAlias {
			t.Errorf("ValidateAlias(%q) error = %v, want ErrInvalidAlias", alias, err)
		}
	}
}

func TestAccountValidateAppliesGitHubUsernameRulesToCanonicalUsername(t *testing.T) {
	account := NewAccount("work", "Work User", "work@example.com", "")
	account.Platform = "github"
	account.Username = "invalid_name"
	if err := account.Validate(); err != ErrInvalidGitHubUsernameFormat {
		t.Fatalf("Validate() error = %v, want ErrInvalidGitHubUsernameFormat", err)
	}
	account.Platform = "gitlab"
	if err := account.Validate(); err != nil {
		t.Fatalf("GitLab username was incorrectly rejected: %v", err)
	}
}

func TestNewAccountDoesNotAdvertiseTokenStorage(t *testing.T) {
	for _, account := range []*Account{
		NewAccount("work", "Work", "w@example.com", ""),
		NewIsolatedAccount("iso", "Iso", "i@example.com", "", "iso"),
	} {
		if account.IsolationMetadata != nil || account.RequiresTokenIsolation() {
			t.Fatalf("%s writes unused isolation metadata: %#v", account.Alias, account.IsolationMetadata)
		}
	}
}
