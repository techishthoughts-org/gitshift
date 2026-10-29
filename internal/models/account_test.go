package models

import (
	"errors"
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

func TestCommitSigning(t *testing.T) {
	tests := []struct {
		name                string
		account             Account
		wantFormat, wantKey string
		wantValidErr        bool
	}{
		{name: "off by default", account: Account{}},
		{name: "legacy gpg enabled", account: Account{GPGKeyID: "K", GPGEnabled: true}, wantFormat: "gpg", wantKey: "K"},
		{name: "legacy gpg disabled", account: Account{GPGKeyID: "K"}},
		{name: "ssh uses account key", account: Account{SigningFormat: "ssh", SSHKeyPath: "/k"}, wantFormat: "ssh", wantKey: "/k"},
		{name: "ssh with signing key", account: Account{SigningFormat: "ssh", SSHKeyPath: "/k", SigningKey: "/s.pub"}, wantFormat: "ssh", wantKey: "/s.pub"},
		{name: "ssh wins over gpg_enabled", account: Account{SigningFormat: "ssh", SSHKeyPath: "/k", GPGKeyID: "K", GPGEnabled: true}, wantFormat: "ssh", wantKey: "/k"},
		{name: "ssh without any key", account: Account{SigningFormat: "ssh"}, wantFormat: "ssh", wantValidErr: true},
		{name: "gpg without key", account: Account{SigningFormat: "gpg"}, wantFormat: "gpg", wantValidErr: true},
		{name: "unknown format", account: Account{SigningFormat: "x509"}, wantValidErr: true},
		{name: "signing key without ssh", account: Account{SigningKey: "/s"}, wantValidErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, key := tt.account.CommitSigning()
			if format != tt.wantFormat || key != tt.wantKey {
				t.Fatalf("CommitSigning() = %q, %q; want %q, %q", format, key, tt.wantFormat, tt.wantKey)
			}
			if err := tt.account.validateSigning(); (err != nil) != tt.wantValidErr {
				t.Fatalf("validateSigning() error = %v, want error %t", err, tt.wantValidErr)
			}
		})
	}
}

func TestSetCommitSigning(t *testing.T) {
	account := Account{SSHKeyPath: "/k", GPGKeyID: "K", GPGEnabled: true, SigningKey: ""}
	if err := account.SetCommitSigning("SSH"); err != nil {
		t.Fatal(err)
	}
	if format, _ := account.CommitSigning(); format != "ssh" || account.GPGEnabled {
		t.Fatalf("ssh: format=%q gpgEnabled=%t", format, account.GPGEnabled)
	}
	account.SigningKey = "/s"
	if err := account.SetCommitSigning("gpg"); err != nil {
		t.Fatal(err)
	}
	if format, key := account.CommitSigning(); format != "gpg" || key != "K" || account.SigningKey != "" {
		t.Fatalf("gpg: format=%q key=%q signingKey=%q", format, key, account.SigningKey)
	}
	if err := account.SetCommitSigning("off"); err != nil {
		t.Fatal(err)
	}
	if format, _ := account.CommitSigning(); format != "" || account.GPGKeyID != "K" {
		t.Fatalf("off: format=%q, key kept=%t", format, account.GPGKeyID == "K")
	}
	if err := (&Account{}).SetCommitSigning("gpg"); !errors.Is(err, ErrSigningKeyMissing) {
		t.Fatalf("gpg without key: %v", err)
	}
	if err := account.SetCommitSigning("pgp"); err == nil {
		t.Fatal("unknown format accepted")
	}
}
