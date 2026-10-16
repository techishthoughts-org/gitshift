package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZshHookIsSafeAndSyntacticallyValid(t *testing.T) {
	hook := zshHook("/tmp/gitshift path/'odd'", " --config '/tmp/config path'")
	for _, required := range []string{"[[ -o interactive ]]", `GITSHIFT_AUTO_SWITCH_DISABLE:-}" != "1"`, "_GITSHIFT_AUTO_SWITCH_RUNNING", "_GITSHIFT_AUTO_SWITCH_LAST_DIR", "add-zsh-hook -d chpwd _gitshift_auto_switch", "add-zsh-hook chpwd", "unset _GITSHIFT_AUTO_SWITCH_LAST_DIR", "auto-switch --quiet -- \"$PWD\""} {
		if !strings.Contains(hook, required) {
			t.Fatalf("hook missing %q", required)
		}
	}
	if strings.Contains(hook, "GITSHIFT_AUTO_SWITCH_DISABLED") {
		t.Fatal("hook contains obsolete plural kill-switch variable")
	}
	path := filepath.Join(t.TempDir(), "hook.zsh")
	if err := os.WriteFile(path, []byte(hook), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	if output, err := exec.Command("zsh", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v: %s", err, output)
	}
}

func TestZshHookKillSwitchAndRepeatedSource(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	root := t.TempDir()
	countFile := filepath.Join(root, "count")
	fake := filepath.Join(root, "gitshift")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf x >> \"$COUNT_FILE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(root, "hook.zsh")
	if err := os.WriteFile(hookPath, []byte(zshHook(fake, "")), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(disable, script string) int {
		t.Helper()
		_ = os.Remove(countFile)
		command := exec.Command("zsh", "-dfi", "-c", script)
		command.Env = append(os.Environ(), "COUNT_FILE="+countFile, "GITSHIFT_AUTO_SWITCH_DISABLE="+disable)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("zsh hook execution: %v: %s", err, output)
		}
		data, err := os.ReadFile(countFile)
		if os.IsNotExist(err) {
			return 0
		}
		if err != nil {
			t.Fatal(err)
		}
		return len(data)
	}
	source := "source " + shellQuote(hookPath)
	if got := run("1", source); got != 0 {
		t.Fatalf("exact kill switch invoked gitshift %d times", got)
	}
	if got := run("true", source); got != 1 {
		t.Fatalf("non-1 kill switch value invoked gitshift %d times, want 1", got)
	}
	if got := run("", source+"; "+source+"; cd "+shellQuote(filepath.Dir(root))); got != 3 {
		t.Fatalf("repeated source/chpwd invoked gitshift %d times, want 3", got)
	}
}
