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
