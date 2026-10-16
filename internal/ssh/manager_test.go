package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectionWithKeyIgnoresConfiguredIdentitiesAndAgent(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	sshPath := filepath.Join(dir, "ssh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARGS_FILE\"\necho \"Hi expected! You've successfully authenticated, but GitHub does not provide shell access.\" >&2\nexit 1\n"
	if err := os.WriteFile(sshPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARGS_FILE", argsFile)
	t.Setenv("PATH", dir)
	key := filepath.Join(dir, "key")
	if err := NewManager().TestConnectionToPlatformContext(context.Background(), "github.com", key); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(args)
	for _, required := range []string{"-F\n" + os.DevNull + "\n", "IdentityAgent=none", "IdentitiesOnly=yes", key} {
		if !strings.Contains(text, required) {
			t.Fatalf("SSH arguments do not contain %q:\n%s", required, text)
		}
	}
}

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

func TestSuccessfulSSHOutputRejectsNegativeMessages(t *testing.T) {
	if isSuccessfulSSHOutput("user is not authenticated") || isSuccessfulSSHOutput("not logged in as anyone") {
		t.Fatal("negative authentication output was accepted")
	}
	if !isSuccessfulSSHOutput("Hi user! You've successfully authenticated, but shell access is disabled") {
		t.Fatal("GitHub success output was rejected")
	}
	if !isSuccessfulSSHOutput("Welcome to GitLab, @user!") {
		t.Fatal("GitLab success output was rejected")
	}
}

func TestUpdateSSHConfigPreservesUnrelatedBlocksExactly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(sshDir, "id_work")
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	original := "Host notgithub.com github.com-alias\n  User deploy\n\nMatch host *.example.com\n  ForwardAgent no\n"
	configPath := filepath.Join(sshDir, "config")
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	manager := NewManager()
	if err := manager.UpdateSSHConfig("work", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), original) {
		t.Fatalf("unrelated SSH config changed:\n%s", first)
	}

	if err := manager.UpdateSSHConfig("personal", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(second), "# >>> gitshift managed: github.com") != 1 || !strings.Contains(string(second), original) {
		t.Fatalf("managed block was not replaced cleanly:\n%s", second)
	}
}

func TestUpdateSSHConfigRemovesConflictingExactHostIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(sshDir, "id_personal")
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	original := "Host github.com other.example.com\n  IdentityFile /old/shared-key\n\nHost github.com\n  IdentityFile /old/github-key\n\nHost keep.example.com\n  User deploy\n"
	configPath := filepath.Join(sshDir, "config")
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NewManager().UpdateSSHConfig("personal", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if strings.Contains(text, "/old/github-key") || strings.Contains(text, "Host github.com other.example.com") {
		t.Fatalf("conflicting GitHub identity was retained:\n%s", text)
	}
	if !strings.Contains(text, "Host other.example.com\n  IdentityFile /old/shared-key") || !strings.Contains(text, "Host keep.example.com") {
		t.Fatalf("unrelated host settings were lost:\n%s", text)
	}
}

func TestUpdateSSHConfigMigratesLegacyManagedHost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(sshDir, "id_new")
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	legacy := "# gitshift Managed Config - DO NOT EDIT MANUALLY\n# This file is automatically generated by gitshift\n\nHost notgithub.com\n  User deploy\n\nHost github.com\n  IdentityFile /old/key\n  AddKeysToAgent yes\n\nMatch host *.example.com\n  ForwardAgent no\n"
	configPath := filepath.Join(sshDir, "config")
	if err := os.WriteFile(configPath, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NewManager().UpdateSSHConfig("work", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if strings.Contains(text, "/old/key") || strings.Contains(text, "AddKeysToAgent yes") {
		t.Fatalf("legacy managed host was retained:\n%s", text)
	}
	if !strings.Contains(text, "Host notgithub.com") || !strings.Contains(text, "Match host *.example.com") {
		t.Fatalf("unrelated legacy config was lost:\n%s", text)
	}
}
