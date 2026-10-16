package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fakeService struct {
	snapshot                         Snapshot
	found                            []Discovery
	loads, writes, switches, imports int
	err                              error
}

func (f *fakeService) Load(context.Context) (Snapshot, error)          { f.loads++; return f.snapshot, f.err }
func (f *fakeService) SaveAccount(context.Context, AccountInput) error { f.writes++; return f.err }
func (f *fakeService) RemoveAccount(context.Context, string) error     { f.writes++; return f.err }
func (f *fakeService) SwitchAccount(context.Context, string) error     { f.switches++; return f.err }
func (f *fakeService) AddRule(context.Context, Rule) error             { f.writes++; return f.err }
func (f *fakeService) RemoveRule(context.Context, string) error        { f.writes++; return f.err }
func (f *fakeService) Discover(context.Context) ([]Discovery, error)   { return f.found, f.err }
func (f *fakeService) Import(context.Context, []Discovery) error       { f.imports++; return f.err }

func press(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func step(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("model type %T", next)
	}
	return got, cmd
}

func initialize(t *testing.T, m Model) Model {
	t.Helper()
	msg := m.Init()()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		var cmd tea.Cmd
		m, cmd = step(t, m, msg)
		_ = cmd
		return m
	}
	for _, cmd := range batch {
		if loaded, ok := cmd().(loadedMsg); ok {
			m, _ = step(t, m, loaded)
		}
	}
	return m
}

func TestOpenAndQuitDoNotWrite(t *testing.T) {
	f := &fakeService{snapshot: Snapshot{Current: "work"}}
	m := New(f, false)
	m = initialize(t, m)
	if f.loads != 1 || f.writes != 0 || f.switches != 0 || f.imports != 0 {
		t.Fatalf("unexpected calls: %+v", f)
	}
	_, cmd := step(t, m, press('q'))
	if cmd == nil {
		t.Fatal("quit command missing")
	}
	if f.writes != 0 {
		t.Fatal("quit wrote state")
	}
}

func TestSwitchRequiresExplicitConfirmation(t *testing.T) {
	f := &fakeService{snapshot: Snapshot{Accounts: []Account{{Alias: "personal", Username: "me", Domain: "github.com"}}}}
	m := New(f, true)
	m = initialize(t, m)
	m, _ = step(t, m, press('1'))
	m, _ = step(t, m, press('s'))
	if f.switches != 0 || m.screen != confirm {
		t.Fatalf("switch ran before confirmation")
	}
	m, cmd := step(t, m, press('y'))
	if cmd == nil {
		t.Fatal("switch command missing")
	}
	msg := cmd()
	m, _ = step(t, m, msg)
	if f.switches != 1 || m.errText != "" {
		t.Fatalf("switch calls=%d err=%s", f.switches, m.errText)
	}
}

func TestDiscoverySelectionAndImportAreExplicit(t *testing.T) {
	f := &fakeService{found: []Discovery{{Identity: "github|github.com|one", Alias: "one"}, {Identity: "github|github.com|two", Alias: "two"}}}
	m := New(f, true)
	m = initialize(t, m)
	m, cmd := step(t, m, press('3'))
	m, _ = step(t, m, cmd())
	if f.imports != 0 {
		t.Fatal("discovery imported automatically")
	}
	m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = step(t, m, press('i'))
	if m.screen != confirm {
		t.Fatal("import did not request confirmation")
	}
	m, cmd = step(t, m, press('y'))
	m, _ = step(t, m, cmd())
	if f.imports != 1 {
		t.Fatalf("imports=%d", f.imports)
	}
}

func TestErrorsAreShownWithoutLeavingUI(t *testing.T) {
	f := &fakeService{err: errors.New("broken config")}
	m := New(f, true)
	m = initialize(t, m)
	if !strings.Contains(m.View().Content, "broken config") {
		t.Fatalf("view=%q", m.View().Content)
	}
}

func TestViewDeclaresScreenModeAndNoSecrets(t *testing.T) {
	f := &fakeService{}
	m := New(f, false)
	if !m.View().AltScreen {
		t.Fatal("expected alternate screen")
	}
	m.inline = true
	if m.View().AltScreen {
		t.Fatal("inline must disable alternate screen")
	}
	if strings.Contains(m.View().Content, "token") {
		t.Fatal("view should not expose token data")
	}
}
