package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateSSHConfigQuotesIdentityAndDoesNotTouchShell(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	shellFile := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(shellFile, []byte("keep me\n"), 0600); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(home, ".ssh", "key with space")
	if err := os.MkdirAll(filepath.Dir(key), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if err := m.UpdateSSHConfig("work", key, "gitlab.example.com"); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), `IdentityFile "`+key+`"`) {
		t.Fatalf("identity path was not quoted: %s", config)
	}
	shell, _ := os.ReadFile(shellFile)
	if string(shell) != "keep me\n" {
		t.Fatalf("shell startup file changed: %q", shell)
	}
}

func TestUpdateSSHConfigRejectsInjectedDomain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := NewManager().UpdateSSHConfig("work", "/tmp/key", "example.com\nHost attacker")
	if err == nil {
		t.Fatal("expected injected domain to be rejected")
	}
}
