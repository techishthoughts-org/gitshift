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

func TestHookExecutablePrefersStablePathEntry(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	versioned := filepath.Join(root, "Caskroom", "gitshift", "2.0.0", "gitshift")
	other := filepath.Join(root, "other", "gitshift")
	for _, path := range []string{versioned, other} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "bin", "gitshift")
	if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(versioned, link); err != nil {
		t.Fatal(err)
	}
	found := func(path string) func(string) (string, error) {
		return func(string) (string, error) { return path, nil }
	}
	missing := func(string) (string, error) { return "", exec.ErrNotFound }

	tests := []struct {
		name       string
		executable string
		lookPath   func(string) (string, error)
		want       string
	}{
		{name: "symlink on PATH runs this binary", executable: versioned, lookPath: found(link), want: link},
		{name: "invoked through the symlink", executable: link, lookPath: found(link), want: link},
		{name: "PATH entry is a different binary", executable: versioned, lookPath: found(other), want: versioned},
		{name: "not on PATH", executable: link, lookPath: missing, want: versioned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hookExecutable(tt.executable, tt.lookPath); got != tt.want {
				t.Fatalf("hookExecutable() = %q, want %q", got, tt.want)
			}
		})
	}
}
