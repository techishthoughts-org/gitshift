package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestShellQuote(t *testing.T) {
	got := shellQuote("/tmp/key with ' quote")
	if strings.Contains(got, "\n") || got != "'/tmp/key with '\\'' quote'" {
		t.Fatalf("unexpected shell quoting: %q", got)
	}
}
