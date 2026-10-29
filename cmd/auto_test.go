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
