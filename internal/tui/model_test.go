package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/techishthoughts/gitshift/internal/models"
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
	m, cmd = step(t, m, msg)
	if cmd == nil || !m.busy {
		t.Fatal("post-mutation reload did not start")
	}
	m, _ = step(t, m, cmd())
	if f.switches != 1 || m.errText != "" {
		t.Fatalf("switch calls=%d err=%s", f.switches, m.errText)
	}
	if m.status != "Switched to personal" {
		t.Fatalf("success status overwritten: %q", m.status)
	}
}

func TestDiscoverySelectionAndImportAreExplicit(t *testing.T) {
	f := &fakeService{found: []Discovery{
		{Identity: "github|github.com|one", Alias: "one", Username: "one", Platform: "github", Domain: "github.com"},
		{Identity: "github|github.com|two", Alias: "two", Username: "two", Platform: "github", Domain: "github.com"},
	}}
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

func TestInitialLoadIsBusyAndIgnoresNavigation(t *testing.T) {
	m := New(&fakeService{}, true)
	if !m.busy {
		t.Fatal("initial load must start busy")
	}
	m, cmd := step(t, m, press('1'))
	if cmd != nil || m.screen != dashboard {
		t.Fatal("navigation was accepted during initial load")
	}
}

func TestStaleAsyncCompletionsAreIgnored(t *testing.T) {
	m := initialize(t, New(&fakeService{}, true))
	oldID := m.beginOperation("old")
	newID := m.beginOperation("new")
	m, _ = step(t, m, discoveredMsg{id: oldID, results: []Discovery{{Alias: "stale"}}})
	if !m.busy || len(m.discovered) != 0 || m.status != "new" {
		t.Fatalf("stale result changed model: %+v", m)
	}
	m, _ = step(t, m, discoveredMsg{id: newID, results: []Discovery{{Alias: "fresh"}}})
	if m.busy || len(m.discovered) != 1 || m.discovered[0].Alias != "fresh" {
		t.Fatalf("current result not applied: %+v", m.discovered)
	}
}

type blockingService struct {
	*fakeService
	started   chan struct{}
	release   chan struct{}
	completed chan struct{}
}

func (b *blockingService) SwitchAccount(context.Context, string) error {
	close(b.started)
	<-b.release
	b.switches++
	close(b.completed)
	return nil
}

func TestQuitWaitsForMutationAndPostMutationReload(t *testing.T) {
	for name, quitKey := range map[string]tea.KeyPressMsg{"ctrl-c": {Code: 'c', Mod: tea.ModCtrl}, "q": press('q')} {
		t.Run(name, func(t *testing.T) {
			base := &fakeService{snapshot: Snapshot{Accounts: []Account{{Alias: "personal"}}}}
			service := &blockingService{fakeService: base, started: make(chan struct{}), release: make(chan struct{}), completed: make(chan struct{})}
			m := initialize(t, New(service, true))
			m, _ = step(t, m, press('1'))
			m, _ = step(t, m, press('s'))
			m, mutationCmd := step(t, m, press('y'))
			result := make(chan tea.Msg, 1)
			go func() { result <- mutationCmd() }()
			select {
			case <-service.started:
			case <-time.After(time.Second):
				t.Fatal("mutation did not start")
			}
			m, quitCmd := step(t, m, quitKey)
			if quitCmd != nil || !m.pendingQuit || !m.busy {
				t.Fatal("quit exited while mutation was active")
			}
			select {
			case <-service.completed:
				t.Fatal("mutation completed before release")
			default:
			}
			close(service.release)
			msg := <-result
			m, reloadCmd := step(t, m, msg)
			if !m.busy || reloadCmd == nil {
				t.Fatal("model did not remain busy for post-mutation reload")
			}
			select {
			case <-service.completed:
			default:
				t.Fatal("mutation had not completed before reload")
			}
			m, quitCmd = step(t, m, reloadCmd())
			if quitCmd == nil || m.busy {
				t.Fatal("deferred quit was not emitted after reload")
			}
		})
	}
}

func TestSignalQuitRequestWaitsForMutationAndPostMutationReload(t *testing.T) {
	base := &fakeService{snapshot: Snapshot{Accounts: []Account{{Alias: "personal"}}}}
	service := &blockingService{fakeService: base, started: make(chan struct{}), release: make(chan struct{}), completed: make(chan struct{})}
	m := initialize(t, New(service, true))
	m, _ = step(t, m, press('1'))
	m, _ = step(t, m, press('s'))
	m, mutationCmd := step(t, m, press('y'))
	result := make(chan tea.Msg, 1)
	go func() { result <- mutationCmd() }()
	select {
	case <-service.started:
	case <-time.After(time.Second):
		t.Fatal("mutation did not start")
	}

	m, quitCmd := step(t, m, QuitRequestedMsg{})
	if quitCmd != nil || !m.pendingQuit || !m.busy {
		t.Fatal("signal request exited while mutation was active")
	}
	close(service.release)
	m, reloadCmd := step(t, m, <-result)
	if !m.busy || reloadCmd == nil {
		t.Fatal("model did not refresh after mutation")
	}
	m, quitCmd = step(t, m, reloadCmd())
	if quitCmd == nil || m.busy {
		t.Fatal("signal request did not quit after the safe boundary")
	}
}

func TestNoColorPresenceAndRenderSanitization(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	f := &fakeService{snapshot: Snapshot{Current: "\x1b]52;c;secret\aevil\nnext", Accounts: []Account{{Alias: "\x1b[31mwork\x1b[0m", Username: "user\rname", Domain: "github.com"}}}}
	m := initialize(t, New(f, true))
	if !m.noColor {
		t.Fatal("empty NO_COLOR must disable color")
	}
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 48, Height: 16})
	m, _ = step(t, m, press('1'))
	view := m.View().Content
	if strings.Contains(view, "\x1b") || strings.Contains(view, "evil\nnext") {
		t.Fatalf("unsanitized terminal content: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > 46 {
			t.Fatalf("line width=%d: %q", width, line)
		}
	}
}

func TestEditAliasIsReadOnlyAndFocusedFieldStaysVisible(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	f := &fakeService{snapshot: Snapshot{Accounts: []Account{{Alias: "immutable", Name: "Name", Email: "n@example.com", Username: "name", Platform: "github", Domain: "github.com"}}}}
	m := initialize(t, New(f, true))
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 48, Height: 16})
	m, _ = step(t, m, press('1'))
	m, _ = step(t, m, press('e'))
	if !m.accountEdit || m.labels[0] != "Name" || m.focus != 0 {
		t.Fatalf("edit form state: labels=%v focus=%d", m.labels, m.focus)
	}
	for range len(m.inputs) - 1 {
		m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	}
	view := m.View().Content
	if !strings.Contains(view, "Alias (read-only): immutable") || !strings.Contains(view, "Description") {
		t.Fatalf("focused field/read-only alias not visible: %q", view)
	}
	if strings.Contains(m.footer(), "q quit") {
		t.Fatalf("form footer advertises q: %q", m.footer())
	}
}

func TestSelectedDiscoveryAliasCannotReplaceExistingAccount(t *testing.T) {
	m := Model{snapshot: Snapshot{Accounts: []Account{{Alias: "work"}}}, discovered: []Discovery{{Alias: "WORK"}}, selected: map[int]bool{0: true}}
	if err := m.validateSelectedImport(); err == nil {
		t.Fatal("expected configured alias collision")
	}
}

func TestSelectedDiscoveryMustBeReadyBeforeConfirmation(t *testing.T) {
	m := Model{discovered: []Discovery{{Alias: "incomplete", Platform: "github", Domain: "github.com"}}, selected: map[int]bool{0: true}}
	if err := m.validateSelectedImport(); err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("error = %v, want actionable readiness failure", err)
	}
}

type cancelDiscoveryService struct{ *fakeService }

func (s *cancelDiscoveryService) Discover(ctx context.Context) ([]Discovery, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestQuitCancelsReadOnlyDiscovery(t *testing.T) {
	service := &cancelDiscoveryService{fakeService: &fakeService{}}
	m := initialize(t, New(service, true))
	m, discoverCmd := step(t, m, press('3'))
	result := make(chan tea.Msg, 1)
	go func() { result <- discoverCmd() }()
	m, quitCmd := step(t, m, press('q'))
	if quitCmd != nil || !m.pendingQuit || m.discoveryCancel == nil {
		t.Fatal("quit did not request discovery cancellation")
	}
	select {
	case msg := <-result:
		m, quitCmd = step(t, m, msg)
		if quitCmd == nil || m.busy {
			t.Fatal("UI did not quit after discovery cancellation completed")
		}
	case <-time.After(time.Second):
		t.Fatal("discovery context was not canceled")
	}
}

func TestNarrowHelpRemainsIntelligible(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	m := initialize(t, New(&fakeService{}, true))
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 48, Height: 16})
	m, _ = step(t, m, press('?'))
	view := m.View().Content
	for _, phrase := range []string{"q quits", "require confirmation", "selected, ready", "identities"} {
		if !strings.Contains(view, phrase) {
			t.Fatalf("narrow help lost %q: %q", phrase, view)
		}
	}
}

func TestResponsiveRenderBoundsAtCommonTerminalSizes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	accounts := make([]Account, 12)
	for i := range accounts {
		accounts[i] = Account{Alias: strings.Repeat("wide", 30), Username: "unicode-user", Domain: "enterprise.example.com"}
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 48, Height: 16}, {Width: 80, Height: 24}} {
		m := initialize(t, New(&fakeService{snapshot: Snapshot{Accounts: accounts}}, true))
		m, _ = step(t, m, size)
		m, _ = step(t, m, press('1'))
		for _, line := range strings.Split(m.View().Content, "\n") {
			if got := lipgloss.Width(line); got > size.Width-2 {
				t.Fatalf("%dx%d line width=%d: %q", size.Width, size.Height, got, line)
			}
		}
		if size.Height == 16 {
			m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
			if lines := strings.Count(m.View().Content, "\n") + 1; lines > size.Height {
				t.Fatalf("detail uses %d lines at %dx%d", lines, size.Width, size.Height)
			}
			m, _ = step(t, m, press('e'))
			if lines := strings.Count(m.View().Content, "\n") + 1; lines > size.Height {
				t.Fatalf("form uses %d lines at %dx%d", lines, size.Width, size.Height)
			}
		}
	}
}

func TestAccountFormDomainFollowsPlatform(t *testing.T) {
	tests := []struct {
		name       string
		account    Account
		platform   string
		wantField  string
		wantDomain string
	}{
		{name: "new gitlab account", platform: "gitlab", wantField: "", wantDomain: "gitlab.com"},
		{name: "new github account", platform: "github", wantField: "", wantDomain: "github.com"},
		{name: "edit gitlab default domain", account: Account{Alias: "lab", Platform: "gitlab", Domain: "gitlab.com"}, platform: "gitlab", wantField: "", wantDomain: "gitlab.com"},
		{name: "edit platform change drops old default", account: Account{Alias: "hub", Platform: "github", Domain: "github.com"}, platform: "gitlab", wantField: "", wantDomain: "gitlab.com"},
		{name: "edit keeps self-managed domain", account: Account{Alias: "corp", Platform: "gitlab", Domain: "git.example.com"}, platform: "gitlab", wantField: "git.example.com", wantDomain: "git.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(&fakeService{}, true)
			m.openAccountForm(tt.account)
			domainIdx, platformIdx := -1, -1
			for i, label := range m.labels {
				switch label {
				case "Domain":
					domainIdx = i
				case "Platform":
					platformIdx = i
				}
			}
			if got := m.inputs[domainIdx].Value(); got != tt.wantField {
				t.Fatalf("Domain field = %q, want %q", got, tt.wantField)
			}
			m.inputs[platformIdx].SetValue(tt.platform)
			v := m.values()
			resolved := (&models.Account{Platform: v[platformIdx], Domain: v[domainIdx]}).GetDomain()
			if resolved != tt.wantDomain {
				t.Fatalf("saved account resolves to %q, want %q", resolved, tt.wantDomain)
			}
			in := AccountInput{Alias: "acct", Name: "Name", Email: "n@example.com", Username: "user", Platform: v[platformIdx], Domain: v[domainIdx]}
			if err := in.Validate(); err != nil {
				t.Fatalf("form values do not validate: %v", err)
			}
		})
	}
}
