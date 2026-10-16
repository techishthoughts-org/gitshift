package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	gitshiftconfig "github.com/techishthoughts/gitshift/internal/config"
	gitshifttui "github.com/techishthoughts/gitshift/internal/tui"
)

func TestUIRejectsNonTTYBeforeCreatingConfig(t *testing.T) {
	oldCheck, oldCfg := terminalCheck, cfgFile
	defer func() { terminalCheck = oldCheck; cfgFile = oldCfg }()
	terminalCheck = func() bool { return false }
	cfgFile = filepath.Join(t.TempDir(), "nested", "config.yaml")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("inline", false, "")
	err := runUICommand(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "requires a TTY") {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(cfgFile); !os.IsNotExist(statErr) {
		t.Fatalf("config created: %v", statErr)
	}
}

func TestUIServiceUsesConfiguredPathAndInjectedSwitch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom", "accounts.yaml")
	manager := gitshiftconfig.NewManagerWithPath(path)
	switchCalls := 0
	service := &uiService{manager: manager, switchFn: func(_ context.Context, got *gitshiftconfig.Manager, alias string) error {
		switchCalls++
		return got.SetCurrentAccount(alias)
	}}
	input := gitshifttui.AccountInput{Alias: "personal", Name: "Personal User", Email: "me@example.com", Username: "me", Platform: "github", Domain: "github.com"}
	if err := service.SaveAccount(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if err := service.SwitchAccount(context.Background(), "personal"); err != nil {
		t.Fatal(err)
	}
	if switchCalls != 1 {
		t.Fatalf("switch calls=%d", switchCalls)
	}
	snapshot, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Current != "personal" || len(snapshot.Accounts) != 1 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestUIServiceDiscoveryImportIsAdditiveAndAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	service := &uiService{manager: gitshiftconfig.NewManagerWithPath(path)}
	base := gitshifttui.AccountInput{Alias: "existing", Name: "Existing", Email: "existing@example.com", Username: "existing", Platform: "github", Domain: "github.com"}
	if err := service.SaveAccount(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	selected := []gitshifttui.Discovery{
		{Identity: "github|github.com|one", Alias: "one", Name: "One", Email: "one@example.com", Username: "one", Platform: "github", Domain: "github.com"},
		{Identity: "github|github.com|two", Alias: "two", Name: "Two", Email: "two@example.com", Username: "two", Platform: "github", Domain: "github.com"},
	}
	if err := service.Import(context.Background(), selected); err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Accounts) != 3 {
		t.Fatalf("accounts=%d", len(snapshot.Accounts))
	}
	selected[0].Alias = "THREE"
	selected[1].Alias = "three"
	if err := service.Import(context.Background(), selected); err == nil {
		t.Fatal("expected case-insensitive batch collision")
	}
	snapshot, _ = service.Load(context.Background())
	if len(snapshot.Accounts) != 3 {
		t.Fatalf("failed import was partial: %d", len(snapshot.Accounts))
	}
}
