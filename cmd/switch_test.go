package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
)

func TestSwitchGitHubCLIUsesConfiguredHostAndUsername(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	gh := filepath.Join(dir, "gh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARGS_FILE\"\n"
	if err := os.WriteFile(gh, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARGS_FILE", argsFile)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := switchGitHubCLI("github.example.com", "octocat-work"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := "auth\nswitch\n--hostname\ngithub.example.com\n--user\noctocat-work\n"
	if string(got) != want {
		t.Fatalf("gh args = %q, want %q", got, want)
	}
}

func TestSwitchAccountAlreadyCurrentStillValidatesAndRepairs(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	missingKey := filepath.Join(t.TempDir(), "missing-key")
	account := models.NewAccount("personal", "Personal", "personal@example.com", missingKey)
	if err := manager.AddAccount(account); err != nil {
		t.Fatal(err)
	}
	command := &cobra.Command{}
	err := switchAccount(command, manager, "personal", false, false, true)
	if err == nil || !strings.Contains(err.Error(), "SSH key not found") {
		t.Fatalf("already-current account was not revalidated: %v", err)
	}
}

func TestUpdateGitConfigClearsPreviousSSHIdentityForKeylessAccount(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "global.gitconfig"))
	t.Chdir(dir)
	if err := exec.Command("git", "config", "--global", "core.sshCommand", "ssh -i /old/key").Run(); err != nil {
		t.Fatal(err)
	}
	account := models.NewAccount("keyless", "Keyless User", "keyless@example.com", "")
	if err := updateGitConfig(account); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "config", "--global", "--get", "core.sshCommand").Run(); err == nil {
		t.Fatal("previous core.sshCommand was retained")
	}
}

func TestSwitchStateRestoreRevertsGitAndSSH(t *testing.T) {
	dir := t.TempDir()
	globalConfig := filepath.Join(dir, "global.gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	sshPath := filepath.Join(dir, "ssh", "config")
	if err := os.MkdirAll(filepath.Dir(sshPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshPath, []byte("old ssh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	state := &switchState{
		gitValues:  []configValueSnapshot{{scope: "--global", key: "user.name", values: []string{"Old User"}}},
		sshPath:    sshPath,
		sshData:    []byte("old ssh\n"),
		sshMode:    0600,
		sshExisted: true,
	}
	if err := exec.Command("git", "config", "--global", "user.name", "New User").Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sshPath, []byte("new ssh\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := state.restore(); err != nil {
		t.Fatal(err)
	}
	name, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err != nil || strings.TrimSpace(string(name)) != "Old User" {
		t.Fatalf("git user.name was not restored: %q, %v", name, err)
	}
	sshConfig, err := os.ReadFile(sshPath)
	if err != nil || string(sshConfig) != "old ssh\n" {
		t.Fatalf("SSH config was not restored: %q, %v", sshConfig, err)
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("/tmp/key with ' quote")
	if strings.Contains(got, "\n") || got != "'/tmp/key with '\\'' quote'" {
		t.Fatalf("unexpected shell quoting: %q", got)
	}
}

func TestIsolatedGitSSHCommandIgnoresOtherConfiguredKeys(t *testing.T) {
	command := isolatedGitSSHCommand("/tmp/key with space")
	for _, required := range []string{"-F " + shellQuote(os.DevNull), "-i '/tmp/key with space'", "IdentitiesOnly=yes", "IdentityAgent=none"} {
		if !strings.Contains(command, required) {
			t.Fatalf("isolated Git SSH command does not contain %q: %s", required, command)
		}
	}
}
