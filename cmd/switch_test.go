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

func TestIsolatedGitSSHCommandPinsKeyAndKeepsSSHConfig(t *testing.T) {
	command := isolatedGitSSHCommand("/tmp/key with space")
	if command != "ssh -i '/tmp/key with space' -o IdentitiesOnly=yes" {
		t.Fatalf("Git SSH command = %s", command)
	}
	for _, forbidden := range []string{"-F", "IdentityAgent"} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("Git SSH command must keep the user's SSH config and agent, found %q: %s", forbidden, command)
		}
	}
}

// fakeGH installs a gh script that answers 'auth status' with state for
// login "alice" and records every invocation with GH_TOKEN/GH_HOST.
func fakeGH(t *testing.T, state string) string {
	t.Helper()
	dir := t.TempDir()
	logFile := filepath.Join(dir, "gh.log")
	script := `#!/bin/sh
printf '%s|GH_TOKEN=%s|GH_HOST=%s\n' "$*" "$GH_TOKEN" "$GH_HOST" >> "$GH_LOG"
case "$1 $2" in
  "auth status") case "$*" in *'"alice"'*) printf '%s' "$GH_STATE" ;; esac ;;
  "auth token") case "$*" in *"--user alice"*) echo token-for-alice ;; *) exit 1 ;; esac ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_LOG", logFile)
	t.Setenv("GH_STATE", state)
	t.Setenv("GH_TOKEN", "")
	return logFile
}

func TestCheckGitHubCLIAccount(t *testing.T) {
	tests := []struct {
		name, state, user, wantErr string
	}{
		{name: "logged in", state: "success", user: "alice"},
		{name: "not logged in", user: "bob", wantErr: "not logged in as @bob"},
		{name: "expired token", state: "error", user: "alice", wantErr: "is error"},
		{name: "no username", user: "", wantErr: "no username"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeGH(t, tt.state)
			err := checkGitHubCLIAccount("github.com", tt.user)
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

func TestRootCommandPrintsErrorsOnce(t *testing.T) {
	if !rootCmd.SilenceErrors || !rootCmd.SilenceUsage {
		t.Fatal("root command must leave error printing to main and skip usage on errors")
	}
}
