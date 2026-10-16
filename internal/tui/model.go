package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type screen uint8

const (
	dashboard screen = iota
	accounts
	detail
	accountForm
	rules
	ruleForm
	discover
	discoveryAlias
	help
	confirm
)

type action uint8

const (
	noAction action = iota
	switchAction
	removeAccountAction
	removeRuleAction
	importAction
)

type loadedMsg struct {
	id       uint64
	snapshot Snapshot
	err      error
}
type discoveredMsg struct {
	id      uint64
	results []Discovery
	err     error
}
type mutationMsg struct {
	id    uint64
	label string
	err   error
}

// QuitRequestedMsg asks the UI to exit at the next safe boundary. The Cobra
// adapter uses it for SIGINT and SIGTERM so operating-system signals cannot
// bypass an in-flight mutation and its rollback/refresh path.
type QuitRequestedMsg struct{}

type Model struct {
	service          Service
	ctx              context.Context
	inline, noColor  bool
	dark             bool
	width, height    int
	screen, previous screen
	cursor           int
	snapshot         Snapshot
	discovered       []Discovery
	selected         map[int]bool
	busy             bool
	nextOp, activeOp uint64
	pendingQuit      bool
	mutationStatus   string
	mutationError    string
	status, errText  string
	inputs           []textinput.Model
	labels           []string
	focus            int
	formOriginal     string
	accountEdit      bool
	confirmAction    action
	confirmValue     string
}

func New(service Service, inline bool) Model {
	return NewWithContext(context.Background(), service, inline)
}

func NewWithContext(ctx context.Context, service Service, inline bool) Model {
	if ctx == nil {
		ctx = context.Background()
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	return Model{service: service, ctx: ctx, inline: inline, noColor: noColor,
		dark: true, selected: map[int]bool{}, busy: true, nextOp: 1, activeOp: 1,
		status: "Loading configuration..."}
}

func (m Model) Init() tea.Cmd { return tea.Batch(tea.RequestBackgroundColor, m.loadCmd(m.activeOp)) }

func (m Model) loadCmd(id uint64) tea.Cmd {
	return func() tea.Msg { s, err := m.service.Load(m.ctx); return loadedMsg{id: id, snapshot: s, err: err} }
}

func (m *Model) beginOperation(status string) uint64 {
	m.nextOp++
	m.activeOp = m.nextOp
	m.busy = true
	m.status = status
	return m.activeOp
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case QuitRequestedMsg:
		if m.busy {
			m.pendingQuit = true
			m.status = "Finishing the current operation before quitting..."
			return m, nil
		}
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		inputWidth := max(12, min(72, msg.Width-6))
		for i := range m.inputs {
			m.inputs[i].SetWidth(inputWidth)
		}
		return m, nil
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		return m, nil
	case loadedMsg:
		if msg.id != m.activeOp {
			return m, nil
		}
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
		} else {
			m.snapshot = msg.snapshot
			sort.Slice(m.snapshot.Accounts, func(i, j int) bool {
				return strings.ToLower(m.snapshot.Accounts[i].Alias) < strings.ToLower(m.snapshot.Accounts[j].Alias)
			})
			m.clampCursor()
			if m.mutationError != "" {
				m.errText = m.mutationError
				m.status = ""
			} else if m.mutationStatus != "" {
				m.errText = ""
				m.status = m.mutationStatus
			} else {
				m.errText = ""
				m.status = "Configuration loaded"
			}
		}
		m.mutationError, m.mutationStatus = "", ""
		if m.pendingQuit {
			return m, tea.Quit
		}
		return m, nil
	case discoveredMsg:
		if msg.id != m.activeOp {
			return m, nil
		}
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
		} else {
			m.errText = ""
			m.discovered = append([]Discovery(nil), msg.results...)
			m.selected = map[int]bool{}
			m.cursor = 0
			m.status = fmt.Sprintf("Found %d identities; select the accounts to import", len(msg.results))
		}
		if m.pendingQuit {
			return m, tea.Quit
		}
		return m, nil
	case mutationMsg:
		if msg.id != m.activeOp {
			return m, nil
		}
		if msg.err != nil {
			m.mutationError = msg.err.Error()
			m.mutationStatus = ""
		} else {
			m.mutationStatus = msg.label
			m.mutationError = ""
		}
		m.screen, m.cursor = dashboard, 0
		id := m.beginOperation("Refreshing configuration...")
		return m, m.loadCmd(id)
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	ks := key.String()
	if ks == "ctrl+c" || (m.busy && ks == "q") {
		if m.busy {
			m.pendingQuit = true
			m.status = "Finishing the current operation before quitting..."
			return m, nil
		}
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
	}
	if m.screen == accountForm || m.screen == ruleForm || m.screen == discoveryAlias {
		return m.updateForm(key)
	}
	if m.screen == help {
		if ks == "esc" || ks == "?" || ks == "enter" {
			m.screen = m.previous
		} else if ks == "q" {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.screen == confirm {
		switch ks {
		case "y", "enter":
			return m, m.executeConfirmed()
		case "n", "esc":
			m.screen = m.previous
		}
		return m, nil
	}
	if ks == "q" {
		return m, tea.Quit
	}
	if ks == "?" {
		m.previous, m.screen = m.screen, help
		return m, nil
	}
	if ks == "esc" || ks == "backspace" {
		if m.screen == dashboard {
			return m, tea.Quit
		}
		if m.screen == detail {
			m.screen = accounts
		} else {
			m.screen, m.cursor = dashboard, 0
		}
		m.errText = ""
		return m, nil
	}

	switch m.screen {
	case dashboard:
		switch ks {
		case "1", "a":
			m.screen, m.cursor = accounts, 0
		case "2", "r":
			m.screen, m.cursor = rules, 0
		case "3", "d":
			m.screen, m.cursor = discover, 0
			id := m.beginOperation("Discovering accounts...")
			return m, m.discoverCmd(id)
		case "enter":
			m.screen, m.cursor = accounts, 0
		}
	case accounts:
		m.navigate(ks, len(m.snapshot.Accounts))
		switch ks {
		case "enter":
			if len(m.snapshot.Accounts) > 0 {
				m.screen = detail
			}
		case "n":
			m.openAccountForm(Account{})
		case "e":
			if a, ok := m.accountAt(); ok {
				m.openAccountForm(a)
			}
		case "s":
			if a, ok := m.accountAt(); ok && !a.Current {
				m.openConfirm(switchAction, a.Alias)
			}
		case "d":
			if a, ok := m.accountAt(); ok {
				m.openConfirm(removeAccountAction, a.Alias)
			}
		}
	case detail:
		switch ks {
		case "e":
			if a, ok := m.accountAt(); ok {
				m.openAccountForm(a)
			}
		case "s":
			if a, ok := m.accountAt(); ok && !a.Current {
				m.openConfirm(switchAction, a.Alias)
			}
		case "d":
			if a, ok := m.accountAt(); ok {
				m.openConfirm(removeAccountAction, a.Alias)
			}
		case "esc", "backspace":
			m.screen = accounts
		}
	case rules:
		m.navigate(ks, len(m.snapshot.Rules))
		switch ks {
		case "n":
			m.openRuleForm()
		case "d":
			if len(m.snapshot.Rules) > 0 {
				m.openConfirm(removeRuleAction, m.snapshot.Rules[m.cursor].Path)
			}
		}
	case discover:
		m.navigate(ks, len(m.discovered))
		switch ks {
		case "space":
			if len(m.discovered) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "e":
			if len(m.discovered) > 0 {
				m.openAliasForm()
			}
		case "i":
			if m.selectedCount() > 0 {
				if err := m.validateSelectedImport(); err != nil {
					m.errText = err.Error()
					return m, nil
				}
				m.openConfirm(importAction, fmt.Sprintf("%d selected account(s)", m.selectedCount()))
			}
		case "r":
			id := m.beginOperation("Discovering accounts...")
			return m, m.discoverCmd(id)
		}
	}
	return m, nil
}

func (m *Model) navigate(key string, count int) {
	if count == 0 {
		m.cursor = 0
		return
	}
	switch key {
	case "j", "down":
		if m.cursor < count-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	}
}
func (m *Model) clampCursor() {
	max := len(m.snapshot.Accounts) - 1
	if m.cursor > max {
		m.cursor = max
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}
func (m Model) accountAt() (Account, bool) {
	if m.cursor < 0 || m.cursor >= len(m.snapshot.Accounts) {
		return Account{}, false
	}
	return m.snapshot.Accounts[m.cursor], true
}
func (m *Model) openConfirm(a action, value string) {
	m.previous, m.screen, m.confirmAction, m.confirmValue = m.screen, confirm, a, value
}
func (m Model) selectedCount() int {
	n := 0
	for _, v := range m.selected {
		if v {
			n++
		}
	}
	return n
}

func (m Model) accountAliases() []string {
	aliases := make([]string, 0, len(m.snapshot.Accounts))
	for _, account := range m.snapshot.Accounts {
		aliases = append(aliases, account.Alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (m Model) hasAccount(alias string) bool {
	for _, account := range m.snapshot.Accounts {
		if account.Alias == alias {
			return true
		}
	}
	return false
}

func (m Model) discoveryAliasAvailable(alias string, current int) error {
	if err := validateAlias(alias); err != nil {
		return err
	}
	for _, account := range m.snapshot.Accounts {
		if strings.EqualFold(account.Alias, alias) {
			return fmt.Errorf("alias %q already belongs to a configured account", alias)
		}
	}
	for i, result := range m.discovered {
		if i != current && strings.EqualFold(result.Alias, alias) {
			return fmt.Errorf("alias %q is already used by another discovery result", alias)
		}
	}
	return nil
}

func (m Model) validateSelectedImport() error {
	seen := make(map[string]string)
	for i, selected := range m.selected {
		if selected {
			alias := m.discovered[i].Alias
			if err := validateAlias(alias); err != nil {
				return err
			}
			for _, account := range m.snapshot.Accounts {
				if strings.EqualFold(account.Alias, alias) {
					return fmt.Errorf("alias %q already belongs to a configured account", alias)
				}
			}
			key := strings.ToLower(alias)
			if prior, ok := seen[key]; ok {
				return fmt.Errorf("selected aliases %q and %q collide", prior, alias)
			}
			seen[key] = alias
		}
	}
	return nil
}

func (m Model) visibleRange(count int) (int, int) {
	limit := max(3, m.height-12)
	if m.height == 0 || count <= limit {
		return 0, count
	}
	start := m.cursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > count {
		start = count - limit
	}
	return start, start + limit
}

func makeInput(label, value string) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = label
	in.SetValue(value)
	in.CharLimit = 512
	return in
}
func (m *Model) setInputs(labels, values []string) {
	m.labels = labels
	m.inputs = make([]textinput.Model, len(labels))
	for i := range labels {
		m.inputs[i] = makeInput(labels[i], values[i])
		if m.width > 0 {
			m.inputs[i].SetWidth(max(12, min(72, m.width-6)))
		}
	}
	m.focus = 0
	if len(m.inputs) > 0 {
		_ = m.inputs[0].Focus()
	}
}
func (m *Model) openAccountForm(a Account) {
	m.previous, m.screen = m.screen, accountForm
	m.formOriginal = a.Alias
	m.accountEdit = a.Alias != ""
	if m.accountEdit {
		m.setInputs([]string{"Name", "Email", "Username", "Platform", "Domain", "SSH key path", "Description"}, []string{a.Name, a.Email, a.Username, defaultString(a.Platform, "github"), defaultString(a.Domain, "github.com"), a.SSHKeyPath, a.Description})
	} else {
		m.setInputs([]string{"Alias", "Name", "Email", "Username", "Platform", "Domain", "SSH key path", "Description"}, []string{"", a.Name, a.Email, a.Username, defaultString(a.Platform, "github"), defaultString(a.Domain, "github.com"), a.SSHKeyPath, a.Description})
	}
}
func (m *Model) openRuleForm() {
	m.previous, m.screen = m.screen, ruleForm
	alias := ""
	if len(m.snapshot.Accounts) > 0 {
		alias = m.snapshot.Accounts[0].Alias
	}
	m.setInputs([]string{"Directory path", "Account alias"}, []string{"", alias})
}
func (m *Model) openAliasForm() {
	m.previous, m.screen = m.screen, discoveryAlias
	m.setInputs([]string{"Alias"}, []string{m.discovered[m.cursor].Alias})
}
func defaultString(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func (m Model) updateForm(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	ks := key.String()
	if ks == "ctrl+c" {
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
	}
	if ks == "esc" {
		m.screen = m.previous
		m.errText = ""
		return m, nil
	}
	if ks == "tab" || ks == "shift+tab" || ks == "up" || ks == "down" {
		m.inputs[m.focus].Blur()
		if ks == "shift+tab" || ks == "up" {
			m.focus--
		} else {
			m.focus++
		}
		if m.focus < 0 {
			m.focus = len(m.inputs) - 1
		}
		if m.focus >= len(m.inputs) {
			m.focus = 0
		}
		_ = m.inputs[m.focus].Focus()
		return m, nil
	}
	if ks == "enter" {
		switch m.screen {
		case accountForm:
			v := m.values()
			var in AccountInput
			if m.accountEdit {
				in = AccountInput{OriginalAlias: m.formOriginal, Alias: m.formOriginal, Name: v[0], Email: v[1], Username: v[2], Platform: v[3], Domain: v[4], SSHKeyPath: v[5], Description: v[6]}
			} else {
				in = AccountInput{Alias: v[0], Name: v[1], Email: v[2], Username: v[3], Platform: v[4], Domain: v[5], SSHKeyPath: v[6], Description: v[7]}
			}
			if err := in.Validate(); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			id := m.beginOperation("Saving account...")
			return m, m.mutation(id, "Account saved", func(ctx context.Context) error { return m.service.SaveAccount(ctx, in) })
		case ruleForm:
			v := m.values()
			if strings.TrimSpace(v[0]) == "" || strings.TrimSpace(v[1]) == "" {
				m.errText = "path and account are required"
				return m, nil
			}
			if !m.hasAccount(v[1]) {
				m.errText = "unknown account alias; choose one of: " + strings.Join(m.accountAliases(), ", ")
				return m, nil
			}
			id := m.beginOperation("Saving directory rule...")
			return m, m.mutation(id, "Directory rule saved", func(ctx context.Context) error { return m.service.AddRule(ctx, Rule{Path: v[0], Account: v[1]}) })
		case discoveryAlias:
			v := m.values()
			if err := validateAlias(v[0]); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			if err := m.discoveryAliasAvailable(v[0], m.cursor); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			m.discovered[m.cursor].Alias = v[0]
			m.screen = m.previous
			m.errText = ""
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(key)
	return m, cmd
}
func (m Model) values() []string {
	v := make([]string, len(m.inputs))
	for i := range m.inputs {
		v[i] = strings.TrimSpace(m.inputs[i].Value())
	}
	return v
}
func validateAlias(s string) error {
	return (&AccountInput{Alias: s, Name: "x", Email: "x@example.com", Username: "x", Platform: "github", Domain: "github.com"}).Validate()
}

func (m Model) discoverCmd(id uint64) tea.Cmd {
	return func() tea.Msg { r, e := m.service.Discover(m.ctx); return discoveredMsg{id: id, results: r, err: e} }
}
func (m Model) mutation(id uint64, label string, fn func(context.Context) error) tea.Cmd {
	return func() tea.Msg { return mutationMsg{id: id, label: label, err: fn(m.ctx)} }
}
func (m *Model) executeConfirmed() tea.Cmd {
	a, v := m.confirmAction, m.confirmValue
	switch a {
	case switchAction:
		id := m.beginOperation("Switching account...")
		return m.mutation(id, "Switched to "+v, func(ctx context.Context) error { return m.service.SwitchAccount(ctx, v) })
	case removeAccountAction:
		id := m.beginOperation("Removing account...")
		return m.mutation(id, "Removed "+v, func(ctx context.Context) error { return m.service.RemoveAccount(ctx, v) })
	case removeRuleAction:
		id := m.beginOperation("Removing directory rule...")
		return m.mutation(id, "Rule removed", func(ctx context.Context) error { return m.service.RemoveRule(ctx, v) })
	case importAction:
		chosen := make([]Discovery, 0, len(m.selected))
		for i, d := range m.discovered {
			if m.selected[i] {
				chosen = append(chosen, d)
			}
		}
		id := m.beginOperation("Importing selected accounts...")
		return m.mutation(id, "Discovery import complete", func(ctx context.Context) error { return m.service.Import(ctx, chosen) })
	}
	return nil
}

func (m Model) View() tea.View {
	content := m.render()
	v := tea.NewView(content)
	v.AltScreen = !m.inline
	v.WindowTitle = "GitShift"
	return v
}

func safeText(value string) string {
	value = ansi.Strip(value)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
}

func fit(value string, width int) string {
	value = safeText(value)
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 3 {
		return ansi.Truncate(value, width, "")
	}
	return ansi.Truncate(value, width, "...")
}

func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 78
	}
	return max(20, m.width-2)
}

func (m Model) formRange() (int, int) {
	count := len(m.inputs)
	limit := max(1, (m.height-10)/2)
	if m.height == 0 || count <= limit {
		return 0, count
	}
	start := m.focus - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > count {
		start = count - limit
	}
	return start, start + limit
}

func (m Model) discoveryState(d Discovery, index int) string {
	for _, account := range m.snapshot.Accounts {
		if strings.EqualFold(account.Alias, d.Alias) {
			return "blocked: alias already configured"
		}
	}
	for i, other := range m.discovered {
		if i != index && strings.EqualFold(other.Alias, d.Alias) {
			return "blocked: duplicate alias"
		}
	}
	in := AccountInput{Alias: d.Alias, Name: d.Name, Email: d.Email, Username: d.Username, Platform: d.Platform, Domain: d.Domain, SSHKeyPath: d.SSHKeyPath}
	if err := in.Validate(); err == nil {
		return "ready"
	}
	missing := make([]string, 0, 2)
	if d.Username == "" && (d.Name == "" || d.Email == "") {
		missing = append(missing, "username or name+email")
	}
	if d.Platform == "" {
		missing = append(missing, "platform")
	}
	if d.Domain == "" {
		missing = append(missing, "host")
	}
	if len(missing) == 0 {
		return "needs correction"
	}
	return "needs " + strings.Join(missing, ", ")
}

func (m Model) footer() string {
	if m.busy {
		return "q/ctrl+c quit after operation"
	}
	switch m.screen {
	case accountForm, ruleForm, discoveryAlias:
		return "tab next  enter save  esc cancel  ctrl+c quit"
	case confirm:
		return "y/enter confirm  n/esc cancel  ctrl+c quit"
	case help:
		return "?/enter/esc back  q quit"
	default:
		return "? help  q quit"
	}
}

func (m Model) helpText() string {
	common := "Arrows or j/k navigate. Escape goes back. q quits."
	switch m.previous {
	case dashboard:
		return "1/Enter accounts. 2 directory rules. 3 discover.\n" + common
	case accounts:
		return "Enter details. n add. e edit. s switch. d remove.\n" + common
	case detail:
		return "e edit. s switch. d remove.\n" + common
	case rules:
		return "n add rule. d remove rule. Shell hook setup remains required.\n" + common
	case discover:
		return "Space selects. e edits alias. i imports selected. r rescans.\n" + common
	default:
		return common
	}
}

func (m Model) render() string {
	styles := newStyles(m.dark, m.noColor)
	width := m.contentWidth()
	var b strings.Builder
	b.WriteString(styles.title.Render("GitShift"))
	if width >= 48 {
		b.WriteString("  ")
		b.WriteString(styles.dim.Render("multi-account workspace"))
	}
	b.WriteString("\n\n")
	if m.busy {
		b.WriteString("Working...\n\n")
	}
	switch m.screen {
	case dashboard:
		b.WriteString(styles.heading.Render("Dashboard"))
		b.WriteString("\n")
		if m.snapshot.Current == "" {
			b.WriteString("No active account\n")
		} else {
			b.WriteString("Current account: " + styles.good.Render(fit(m.snapshot.Current, max(8, width-17))) + "\n")
		}
		b.WriteString(fmt.Sprintf("%d accounts  |  %d directory rules\n\n", len(m.snapshot.Accounts), len(m.snapshot.Rules)))
		if width < 60 {
			b.WriteString("[1] Accounts\n[2] Directory rules\n[3] Discover accounts\n")
		} else {
			b.WriteString("[1] Accounts   [2] Directory rules   [3] Discover accounts\n")
		}
	case accounts:
		b.WriteString(styles.heading.Render("Accounts"))
		b.WriteString("\n")
		if len(m.snapshot.Accounts) == 0 {
			b.WriteString("No accounts. Press n to add one.\n")
		}
		start, end := m.visibleRange(len(m.snapshot.Accounts))
		for i := start; i < end; i++ {
			a := m.snapshot.Accounts[i]
			mark := " "
			if i == m.cursor {
				mark = ">"
			}
			active := ""
			if a.Current {
				active = " (active)"
			}
			var row string
			if width < 60 {
				row = fmt.Sprintf("%s %s @%s%s", mark, a.Alias, a.Username, active)
			} else {
				row = fmt.Sprintf("%s %-20s @%-18s %s%s", mark, a.Alias, a.Username, a.Domain, active)
			}
			b.WriteString(fit(row, width) + "\n")
		}
		if width < 60 {
			b.WriteString("\nn new  e edit\ns switch  d remove  enter details\n")
		} else {
			b.WriteString("\nn new  e edit  s switch  d remove  enter details\n")
		}
	case detail:
		if a, ok := m.accountAt(); ok {
			b.WriteString(styles.heading.Render(fit(a.Alias, width)))
			b.WriteString("\n")
			lines := []string{"Name: " + a.Name, "Email: " + a.Email, "Username: @" + a.Username, "Provider: " + a.Platform, "Host: " + a.Domain, "SSH key: " + emptyAsDash(a.SSHKeyPath), "GPG key: " + emptyAsDash(a.GPGKeyID), "Description: " + emptyAsDash(a.Description)}
			if m.height > 0 && m.height < 20 {
				lines = []string{"Name: " + a.Name, "Email: " + a.Email, "@" + a.Username + " | " + a.Platform + " | " + a.Domain, "SSH: " + emptyAsDash(a.SSHKeyPath), "GPG: " + emptyAsDash(a.GPGKeyID)}
			}
			for _, line := range lines {
				b.WriteString(fit(line, width) + "\n")
			}
			b.WriteString("\ne edit  s switch  d remove\n")
		}
	case accountForm, ruleForm, discoveryAlias:
		b.WriteString(styles.heading.Render(map[screen]string{accountForm: "Account", ruleForm: "Directory rule", discoveryAlias: "Discovery alias"}[m.screen]))
		b.WriteString("\n")
		if m.screen == accountForm && m.accountEdit {
			b.WriteString(fit("Alias (read-only): "+m.formOriginal, width) + "\n")
		}
		start, end := m.formRange()
		if start > 0 {
			b.WriteString(styles.dim.Render("... fields above ...") + "\n")
		}
		for i := start; i < end; i++ {
			in := m.inputs[i]
			prefix := "  "
			if i == m.focus {
				prefix = "> "
			}
			b.WriteString(prefix + styles.dim.Render(safeText(m.labels[i])) + "\n  " + fit(in.View(), width-2) + "\n")
		}
		if end < len(m.inputs) {
			b.WriteString(styles.dim.Render("... fields below ...") + "\n")
		}
		if m.screen == ruleForm {
			b.WriteString(fit("Available aliases: "+strings.Join(m.accountAliases(), ", "), width) + "\n")
			b.WriteString("Run `gitshift shell-init zsh` setup after adding rules.\n")
		}
	case rules:
		b.WriteString(styles.heading.Render("Directory rules"))
		b.WriteString("\n")
		if len(m.snapshot.Rules) == 0 {
			b.WriteString("No rules configured.\n")
		}
		start, end := m.visibleRange(len(m.snapshot.Rules))
		for i := start; i < end; i++ {
			r := m.snapshot.Rules[i]
			mark := " "
			if i == m.cursor {
				mark = ">"
			}
			b.WriteString(fit(fmt.Sprintf("%s %s -> %s", mark, r.Path, r.Account), width) + "\n")
		}
		b.WriteString("\nn add  d remove\n")
		b.WriteString("Shell auto-switching also requires the opt-in shell-init hook.\n")
	case discover:
		b.WriteString(styles.heading.Render("Discovery (read-only until import)"))
		b.WriteString("\n")
		if !m.busy && len(m.discovered) == 0 {
			b.WriteString("No identities found. Press r to scan again.\n")
		}
		start, end := m.visibleRange(len(m.discovered))
		for i := start; i < end; i++ {
			d := m.discovered[i]
			cur := " "
			if i == m.cursor {
				cur = ">"
			}
			check := " "
			if m.selected[i] {
				check = "x"
			}
			row := fmt.Sprintf("%s [%s] %s @%s %s %d/10", cur, check, d.Alias, d.Username, d.Domain, d.Confidence)
			b.WriteString(fit(row, width) + "\n")
			if i == m.cursor {
				b.WriteString(styles.dim.Render(fit("    "+m.discoveryState(d, i)+" | "+d.Identity+" | "+d.Source, width)) + "\n")
			}
		}
		if width < 60 {
			b.WriteString("\nspace select  e edit alias\ni import selected  r rescan\n")
		} else {
			b.WriteString("\nspace select  e alias  i import selected  r rescan\n")
		}
	case help:
		b.WriteString(styles.heading.Render("Help"))
		b.WriteString("\n" + m.helpText() + "\n\nAccount switching and removal always require confirmation.\nDiscovery never imports until identities are selected and confirmed.\nForce switching, skipped validation, and key generation remain CLI-only.\n")
	case confirm:
		verb := map[action]string{switchAction: "Switch to", removeAccountAction: "Remove account", removeRuleAction: "Remove directory rule", importAction: "Import"}[m.confirmAction]
		b.WriteString(styles.warn.Render(fit(fmt.Sprintf("%s %s?", verb, m.confirmValue), width)))
		b.WriteString("\n")
	}
	if m.errText != "" {
		b.WriteString("\n" + styles.bad.Render(fit("Error: "+m.errText, width)) + "\n")
	} else if m.status != "" {
		b.WriteString("\n" + styles.dim.Render(fit(m.status, width)) + "\n")
	}
	b.WriteString("\n" + styles.dim.Render(m.footer()))
	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = ansi.Truncate(line, width, "")
		}
	}
	return strings.Join(lines, "\n")
}
func emptyAsDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

type styles struct{ title, heading, dim, good, warn, bad lipgloss.Style }

func newStyles(dark, noColor bool) styles {
	if noColor {
		plain := lipgloss.NewStyle()
		return styles{title: plain, heading: plain, dim: plain, good: plain, warn: plain, bad: plain}
	}
	fg, muted, accent, good, warn, bad := "#1F2937", "#64748B", "#7C3AED", "#15803D", "#B45309", "#B91C1C"
	if dark {
		fg, muted, accent, good, warn, bad = "#F8FAFC", "#94A3B8", "#C4B5FD", "#4ADE80", "#FBBF24", "#F87171"
	}
	return styles{title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent)), heading: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(fg)), dim: lipgloss.NewStyle().Foreground(lipgloss.Color(muted)), good: lipgloss.NewStyle().Foreground(lipgloss.Color(good)), warn: lipgloss.NewStyle().Foreground(lipgloss.Color(warn)).Bold(true), bad: lipgloss.NewStyle().Foreground(lipgloss.Color(bad)).Bold(true)}
}
