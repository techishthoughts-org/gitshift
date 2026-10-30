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
	active := activeSSHConfig(text)
	if strings.Contains(active, "/old/github-key") || strings.Contains(active, "Host github.com other.example.com") {
		t.Fatalf("conflicting GitHub identity was retained:\n%s", text)
	}
	if !strings.Contains(text, "# gitshift-disabled: Host github.com\n# gitshift-disabled:   IdentityFile /old/github-key") {
		t.Fatalf("conflicting exact host was not kept as a comment:\n%s", text)
	}
	if !strings.Contains(text, "Host other.example.com\n  IdentityFile /old/shared-key") || !strings.Contains(text, "Host keep.example.com") {
		t.Fatalf("unrelated host settings were lost:\n%s", text)
	}
}

// activeSSHConfig drops comment lines so assertions see only effective config.
func activeSSHConfig(config string) string {
	var kept []string
	for _, line := range strings.Split(config, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func writeSSHTestKey(t *testing.T) (string, string) {
	t.Helper()
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
	return filepath.Join(sshDir, "config"), key
}

func TestUpdateSSHConfigKeepsGlobalOptionsAboveManagedBlock(t *testing.T) {
	tests := []struct {
		name     string
		original string
		global   string
	}{
		{
			name:     "global options before host blocks",
			original: "# my settings\nInclude ~/.orbstack/ssh/config\nStrictHostKeyChecking yes\n\nHost example.com\n  User deploy\n",
			global:   "# my settings\nInclude ~/.orbstack/ssh/config\nStrictHostKeyChecking yes",
		},
		{
			name:     "only global options",
			original: "IdentityAgent ~/.1password/agent.sock\n",
			global:   "IdentityAgent ~/.1password/agent.sock",
		},
		{
			name:     "lowercase match block ends the global section",
			original: "ServerAliveInterval 30\nmatch host *.internal\n  User ops\n",
			global:   "ServerAliveInterval 30",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, key := writeSSHTestKey(t)
			if err := os.WriteFile(configPath, []byte(tt.original), 0600); err != nil {
				t.Fatal(err)
			}
			manager := NewManager()
			// Switching twice must stay stable and repair nothing twice.
			for _, alias := range []string{"work", "personal"} {
				if err := manager.UpdateSSHConfig(alias, key, "github.com"); err != nil {
					t.Fatal(err)
				}
			}
			updated, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			text := string(updated)
			if !strings.HasPrefix(text, tt.global+"\n\n# >>> gitshift managed: github.com") {
				t.Fatalf("global options are not ahead of the managed block:\n%s", text)
			}
			if strings.Count(text, tt.global) != 1 {
				t.Fatalf("global options were duplicated:\n%s", text)
			}
		})
	}
}

func TestUpdateSSHConfigRepairsGlobalOptionsMovedByOlderVersions(t *testing.T) {
	configPath, key := writeSSHTestKey(t)
	broken := "# >>> gitshift managed: github.com\n# GitHub account: old\nHost github.com\n    HostName github.com\n    User git\n    IdentityFile \"/old\"\n    IdentitiesOnly yes\n\n# <<< gitshift managed: github.com\n\nStrictHostKeyChecking yes\n\nHost example.com\n  User deploy\n"
	if err := os.WriteFile(configPath, []byte(broken), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NewManager().UpdateSSHConfig("work", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(configPath)
	if !strings.HasPrefix(string(updated), "StrictHostKeyChecking yes\n\n# >>> gitshift managed: github.com") {
		t.Fatalf("misplaced global option was not moved back to the top:\n%s", updated)
	}
}

func TestUpdateSSHConfigKeepsOriginalBackupAcrossSwitches(t *testing.T) {
	configPath, key := writeSSHTestKey(t)
	original := "Host github.com\n  HostName ssh.github.com\n  Port 443\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager()
	for _, alias := range []string{"work", "personal", "work"} {
		if err := manager.UpdateSSHConfig(alias, key, "github.com"); err != nil {
			t.Fatal(err)
		}
	}
	orig, err := os.ReadFile(configPath + ".gitshift-orig")
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != original {
		t.Fatalf("original backup = %q, want %q", orig, original)
	}
	info, err := os.Stat(configPath + ".gitshift-orig")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("original backup mode = %o, want 600", info.Mode().Perm())
	}
	last, err := os.ReadFile(configPath + ".backup")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(last), "# >>> gitshift managed: github.com") {
		t.Fatalf(".backup should hold the previous config, got:\n%s", last)
	}
	updated, _ := os.ReadFile(configPath)
	text := string(updated)
	if strings.Count(text, "# gitshift-disabled: Host github.com") != 1 || !strings.Contains(text, "# gitshift-disabled:   Port 443") {
		t.Fatalf("user block was not disabled exactly once:\n%s", text)
	}
	if strings.Contains(activeSSHConfig(text), "Port 443") {
		t.Fatalf("disabled user block is still active:\n%s", text)
	}
}

func TestUpdateSSHConfigWithoutExistingFileCreatesNoBackup(t *testing.T) {
	configPath, key := writeSSHTestKey(t)
	if err := NewManager().UpdateSSHConfig("work", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{".gitshift-orig", ".backup"} {
		if _, err := os.Stat(configPath + suffix); !os.IsNotExist(err) {
			t.Fatalf("%s exists without a prior config: %v", suffix, err)
		}
	}
	updated, _ := os.ReadFile(configPath)
	if !strings.HasPrefix(string(updated), "# >>> gitshift managed: github.com") {
		t.Fatalf("unexpected new config:\n%s", updated)
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

func TestCheckPrivateKeyFile(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		mode    os.FileMode
		create  bool
		wantErr string
	}{
		{name: "owner read-write", mode: 0600, create: true},
		{name: "owner read-only", mode: 0400, create: true},
		{name: "group readable", mode: 0640, create: true, wantErr: "chmod 600"},
		{name: "world readable", mode: 0644, create: true, wantErr: "permissions 644"},
		{name: "missing", wantErr: "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "_"))
			if tt.create {
				if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(key, tt.mode); err != nil {
					t.Fatal(err)
				}
			}
			err := CheckPrivateKeyFile(key)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveSSHConfigStripsOnlyManagedBlock(t *testing.T) {
	configPath, key := writeSSHTestKey(t)
	original := "AddKeysToAgent yes\n\nHost example.com\n  User deploy\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager()
	if err := manager.UpdateSSHConfig("work", key, "github.com"); err != nil {
		t.Fatal(err)
	}
	if err := manager.RemoveSSHConfig("github.com"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "gitshift managed") || strings.Contains(string(updated), key) {
		t.Fatalf("managed block was not removed:\n%s", updated)
	}
	for _, want := range []string{"AddKeysToAgent yes", "Host example.com\n  User deploy"} {
		if !strings.Contains(string(updated), want) {
			t.Fatalf("unrelated SSH config %q lost:\n%s", want, updated)
		}
	}
}

func TestRemoveSSHConfigWithoutManagedBlockLeavesFileUntouched(t *testing.T) {
	configPath, _ := writeSSHTestKey(t)
	original := "Host github.com\n  User git\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NewManager().RemoveSSHConfig("github.com"); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(configPath)
	if err != nil || string(updated) != original {
		t.Fatalf("SSH config changed: %q, %v", updated, err)
	}
	if _, err := os.Stat(configPath + ".backup"); !os.IsNotExist(err) {
		t.Fatalf("no-op removal wrote a backup: %v", err)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	if err := NewManager().RemoveSSHConfig("github.com"); err != nil {
		t.Fatalf("missing SSH config: %v", err)
	}
}
