package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/models"
)

func TestAutoListJSONAndResolveNoMatch(t *testing.T) {
	oldConfig := cfgFile
	defer func() { cfgFile = oldConfig }()
	root := t.TempDir()
	cfgFile = filepath.Join(root, "config.yaml")
	manager := config.NewManagerWithPath(cfgFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddAccount(models.NewAccount("personal", "Personal", "p@example.com", "")); err != nil {
		t.Fatal(err)
	}
	projects := filepath.Join(root, "projects")
	other := filepath.Join(root, "other")
	if err := os.Mkdir(projects, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddAutoSwitchRule(projects, "personal"); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	autoListCmd.SetOut(&output)
	if err := autoListCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	if err := autoListCmd.RunE(autoListCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"account": "personal"`) {
		t.Fatalf("unexpected JSON: %s", output.String())
	}

	output.Reset()
	autoResolveCmd.SetOut(&output)
	if err := autoResolveCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	if err := autoResolveCmd.RunE(autoResolveCmd, []string{other}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != "null" {
		t.Fatalf("no-match JSON = %q", output.String())
	}
}

func TestAutoSwitchAlreadyCurrentDoesNotMutateConfigOrExternalState(t *testing.T) {
	oldConfig := cfgFile
	defer func() { cfgFile = oldConfig }()
	root := t.TempDir()
	cfgFile = filepath.Join(root, "config.yaml")
	manager := config.NewManagerWithPath(cfgFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.AddAccount(models.NewAccount("personal", "Personal", "p@example.com", "")); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetCurrentAccount("personal"); err != nil {
		t.Fatal(err)
	}
	projects := filepath.Join(root, "projects")
	if err := os.Mkdir(projects, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddAutoSwitchRule(projects, "personal"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cfgFile, 0644); err != nil {
		t.Fatal(err)
	}
	fixed := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(cfgFile, fixed, fixed); err != nil {
		t.Fatal(err)
	}
	beforeData, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	if err := autoSwitchCmd.RunE(autoSwitchCmd, []string{projects}); err != nil {
		t.Fatal(err)
	}
	afterData, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeData, afterData) || before.Mode().Perm() != after.Mode().Perm() || !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("already-current switch mutated config")
	}
	if _, err := os.Stat(cfgFile + ".switch.lock"); !os.IsNotExist(err) {
		t.Fatalf("switch lock was not cleaned up: %v", err)
	}
}

func TestAutoSwitchNoMatchDoesNotCreateConfiguration(t *testing.T) {
	oldConfig := cfgFile
	defer func() { cfgFile = oldConfig }()
	root := t.TempDir()
	cfgFile = filepath.Join(root, "missing", "config.yaml")
	if err := autoSwitchCmd.RunE(autoSwitchCmd, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(cfgFile)); !os.IsNotExist(err) {
		t.Fatalf("no-match auto-switch created configuration directory: %v", err)
	}
}

func autoDefaultTestConfig(t *testing.T) (string, string, string) {
	t.Helper()
	oldConfig := cfgFile
	t.Cleanup(func() { cfgFile = oldConfig })
	root := t.TempDir()
	cfgFile = filepath.Join(root, "config.yaml")
	manager := config.NewManagerWithPath(cfgFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"personal", "work"} {
		if err := manager.AddAccount(models.NewAccount(alias, alias, alias+"@example.com", "")); err != nil {
			t.Fatal(err)
		}
	}
	projects := filepath.Join(root, "projects")
	nested := filepath.Join(projects, "repo")
	other := filepath.Join(root, "other")
	for _, dir := range []string{nested, other} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := manager.AddAutoSwitchRule(projects, "personal"); err != nil {
		t.Fatal(err)
	}
	return cfgFile, nested, other
}

func TestAutoSwitchTargetPrefersRuleThenDefault(t *testing.T) {
	configFile, nested, other := autoDefaultTestConfig(t)
	tests := []struct {
		name, fallback, path, want string
	}{
		{name: "rule wins over default", fallback: "work", path: nested, want: "personal"},
		{name: "default outside rules", fallback: "work", path: other, want: "work"},
		{name: "no default outside rules", path: other, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := config.NewManagerWithPath(configFile)
			if err := manager.Load(); err != nil {
				t.Fatal(err)
			}
			if err := manager.SetAutoSwitchDefault(tt.fallback); err != nil {
				t.Fatal(err)
			}
			got, err := autoSwitchTarget(manager, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("target = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAutoDefaultCommandSetShowClear(t *testing.T) {
	configFile, _, other := autoDefaultTestConfig(t)
	var output bytes.Buffer
	autoDefaultCmd.SetOut(&output)
	autoResolveCmd.SetOut(&output)
	t.Cleanup(func() {
		autoDefaultCmd.SetOut(nil)
		autoResolveCmd.SetOut(nil)
		_ = autoDefaultCmd.Flags().Set("clear", "false")
		_ = autoResolveCmd.Flags().Set("json", "false")
	})
	_ = autoResolveCmd.Flags().Set("json", "false")

	run := func(args ...string) string {
		t.Helper()
		output.Reset()
		if err := autoDefaultCmd.RunE(autoDefaultCmd, args); err != nil {
			t.Fatalf("auto default %v: %v", args, err)
		}
		return strings.TrimSpace(output.String())
	}
	if got := run(); got != "No auto-switch default account configured." {
		t.Fatalf("empty show = %q", got)
	}
	if err := autoDefaultCmd.RunE(autoDefaultCmd, []string{"missing"}); err == nil {
		t.Fatal("unknown alias was accepted")
	}
	if got := run("work"); got != "Auto-switch default account: work" {
		t.Fatalf("set = %q", got)
	}
	if got := run(); got != "work" {
		t.Fatalf("show = %q", got)
	}
	output.Reset()
	if err := autoResolveCmd.RunE(autoResolveCmd, []string{other}); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != "work (default)" {
		t.Fatalf("resolve outside rules = %q", got)
	}
	if err := autoDefaultCmd.Flags().Set("clear", "true"); err != nil {
		t.Fatal(err)
	}
	if err := autoDefaultCmd.RunE(autoDefaultCmd, []string{"work"}); err == nil {
		t.Fatal("ALIAS with --clear was accepted")
	}
	if got := run(); got != "Auto-switch default account cleared." {
		t.Fatalf("clear = %q", got)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if got := manager.GetConfig().AutoSwitch.DefaultAccount; got != "" {
		t.Fatalf("default after clear = %q", got)
	}
}

func TestAutoSwitchDefaultAlreadyCurrentIsNoOp(t *testing.T) {
	configFile, _, other := autoDefaultTestConfig(t)
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetAutoSwitchDefault("work"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetCurrentAccount("work"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	if err := autoSwitchCmd.RunE(autoSwitchCmd, []string{other}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("already-current default switch mutated config")
	}
	if _, err := os.Stat(configFile + ".switch.lock"); !os.IsNotExist(err) {
		t.Fatalf("switch lock was not cleaned up: %v", err)
	}
}

func TestCurrentShortPrintsOnlyAlias(t *testing.T) {
	configFile, _, _ := autoDefaultTestConfig(t)
	var output bytes.Buffer
	currentCmd.SetOut(&output)
	t.Cleanup(func() {
		currentCmd.SetOut(nil)
		_ = currentCmd.Flags().Set("short", "false")
	})
	if err := currentCmd.Flags().Set("short", "true"); err != nil {
		t.Fatal(err)
	}
	if err := currentCmd.RunE(currentCmd, nil); err != nil {
		t.Fatalf("no current account should not fail: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("output without a current account = %q", output.String())
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetCurrentAccount("work"); err != nil {
		t.Fatal(err)
	}
	if err := currentCmd.RunE(currentCmd, nil); err != nil {
		t.Fatal(err)
	}
	if output.String() != "work\n" {
		t.Fatalf("short output = %q, want %q", output.String(), "work\n")
	}
}
