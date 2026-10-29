package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
)

func runGitIncludes(t *testing.T, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	autoGitIncludesCmd.SetOut(&output)
	t.Cleanup(func() { autoGitIncludesCmd.SetOut(nil) })
	if err := autoGitIncludesCmd.RunE(autoGitIncludesCmd, args); err != nil {
		t.Fatalf("git-includes %v: %v", args, err)
	}
	return output.String()
}

func repoValue(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func TestGitIncludesCommandLifecycle(t *testing.T) {
	home := isolateGitConfig(t)
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	workDir := filepath.Join(home, "work", "repo")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q", workDir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	manager := newConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddAutoSwitchRule(filepath.Join(home, "work"), "work"); err != nil {
		t.Fatal(err)
	}

	if out := runGitIncludes(t); !strings.HasPrefix(out, "Git includes: off\n") {
		t.Fatalf("initial status = %q", out)
	}
	out := runGitIncludes(t, "on")
	if !strings.Contains(out, "Git includes: on") || !strings.Contains(out, "-> work") {
		t.Fatalf("on output = %q", out)
	}
	if got := repoValue(t, workDir, "user.email"); got != "work@example.com" {
		t.Fatalf("user.email in rule dir = %q", got)
	}

	// A later rule change is picked up by the save hook without running 'on'.
	personal := models.NewAccount("personal", "Me", "me@example.com", "")
	reloaded := newConfigManager()
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if err := reloaded.AddAccount(personal); err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.AddAutoSwitchRule(filepath.Join(home, "work"), "personal"); err != nil {
		t.Fatal(err)
	}
	if got := repoValue(t, workDir, "user.email"); got != "me@example.com" {
		t.Fatalf("after changing the rule user.email = %q, want the new account", got)
	}

	if out := runGitIncludes(t, "off"); !strings.Contains(out, "removed") {
		t.Fatalf("off output = %q", out)
	}
	if got := repoValue(t, workDir, "user.email"); got == "me@example.com" {
		t.Fatal("include still applies after off")
	}
	final := config.NewManagerWithPath(configFile)
	if err := final.Load(); err != nil {
		t.Fatal(err)
	}
	if final.GetConfig().AutoSwitch.GitIncludes {
		t.Fatal("git_includes still set after off")
	}
}

func TestGitIncludesRejectsUnknownAction(t *testing.T) {
	isolateGitConfig(t)
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	if err := autoGitIncludesCmd.RunE(autoGitIncludesCmd, []string{"maybe"}); err == nil || !strings.Contains(err.Error(), "use on, off or status") {
		t.Fatalf("error = %v", err)
	}
}

func TestPromptAliasFollowsDirectoryWithGitIncludes(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := models.NewConfig()
	cfg.CurrentAccount = "personal"
	cfg.AutoSwitch.Rules = []models.AutoSwitchRule{{Path: workDir, Account: "work"}}
	t.Chdir(workDir)

	if got := promptAlias(cfg); got != "personal\n" {
		t.Fatalf("without includes the prompt shows the applied account, got %q", got)
	}
	cfg.AutoSwitch.GitIncludes = true
	if got := promptAlias(cfg); got != "work\n" {
		t.Fatalf("with includes the prompt shows the directory's account, got %q", got)
	}
	t.Chdir(root)
	if got := promptAlias(cfg); got != "personal\n" {
		t.Fatalf("outside rules the prompt shows the applied account, got %q", got)
	}
	cfg.CurrentAccount = ""
	if got := promptAlias(cfg); got != "" {
		t.Fatalf("nothing applied and no rule: got %q", got)
	}
}

func TestUpdateGitConfigSkipsLocalValuesWithGitIncludes(t *testing.T) {
	home := isolateGitConfig(t)
	repo := filepath.Join(home, "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	t.Chdir(repo)
	account := models.NewAccount("work", "Work", "work@example.com", "")
	if err := updateGitConfig(account, false); err != nil {
		t.Fatal(err)
	}
	if out, _ := exec.Command("git", "config", "--local", "--get", "user.email").Output(); len(out) != 0 {
		t.Fatalf("local user.email written with includes on: %q", out)
	}
	if got := repoValue(t, repo, "user.email"); got != "work@example.com" {
		t.Fatalf("global user.email = %q", got)
	}
}

func TestFailedSwitchRestoresIncludePrecedence(t *testing.T) {
	home := isolateGitConfig(t)
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	repo := filepath.Join(home, "work", "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	manager := newConfigManager()
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddAutoSwitchRule(filepath.Join(home, "work"), "work"); err != nil {
		t.Fatal(err)
	}
	runGitIncludes(t, "on")

	// What a rollback leaves behind: a key re-added in a new section after the includes.
	if out, err := exec.Command("git", "config", "--global", "tag.gpgsign", "true").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := repoValue(t, repo, "tag.gpgsign"); got != "true" {
		t.Fatalf("precondition: the later section should win, got %q", got)
	}

	switching := newConfigManager()
	if err := switching.Load(); err != nil {
		t.Fatal(err)
	}
	// The account's key (/tmp/work) is missing, so the switch fails in preflight.
	if err := switchAccount(switchCmd, switching, "work", false, false, true); err == nil {
		t.Fatal("expected the switch to fail")
	}
	if got := repoValue(t, repo, "tag.gpgsign"); got != "false" {
		t.Fatalf("after a failed switch tag.gpgsign = %q; the include should win again", got)
	}
}
