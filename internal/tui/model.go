package tui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	snapshot Snapshot
	err      error
}
type discoveredMsg struct {
	results []Discovery
	err     error
}
type mutationMsg struct {
	label string
	err   error
}

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
	status, errText  string
	inputs           []textinput.Model
	labels           []string
	focus            int
	formOriginal     string
	confirmAction    action
	confirmValue     string
}

func New(service Service, inline bool) Model {
	return Model{service: service, ctx: context.Background(), inline: inline,
		noColor: os.Getenv("NO_COLOR") != "", dark: true, selected: map[int]bool{}}
}

func (m Model) Init() tea.Cmd { return tea.Batch(tea.RequestBackgroundColor, m.loadCmd()) }

func (m Model) loadCmd() tea.Cmd {
	return func() tea.Msg { s, err := m.service.Load(m.ctx); return loadedMsg{s, err} }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.snapshot = msg.snapshot
		sort.Slice(m.snapshot.Accounts, func(i, j int) bool {
			return strings.ToLower(m.snapshot.Accounts[i].Alias) < strings.ToLower(m.snapshot.Accounts[j].Alias)
		})
		m.status = "Configuration loaded"
		m.clampCursor()
		return m, nil
	case discoveredMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.discovered = append([]Discovery(nil), msg.results...)
		m.selected = map[int]bool{}
		m.cursor = 0
		m.status = fmt.Sprintf("Found %d identities; select the accounts to import", len(msg.results))
		return m, nil
	case mutationMsg:
		m.busy = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.status, m.errText = msg.label, ""
		m.screen, m.cursor = dashboard, 0
		return m, m.loadCmd()
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	ks := key.String()
	if m.screen == accountForm || m.screen == ruleForm || m.screen == discoveryAlias {
		return m.updateForm(key)
	}
	if ks == "ctrl+c" {
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
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
			m.screen, m.cursor, m.busy = discover, 0, true
			return m, m.discoverCmd()
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
				m.openConfirm(importAction, fmt.Sprintf("%d selected account(s)", m.selectedCount()))
			}
		case "r":
			m.busy = true
			return m, m.discoverCmd()
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
	m.setInputs([]string{"Alias", "Name", "Email", "Username", "Platform", "Domain", "SSH key path", "Description"}, []string{a.Alias, a.Name, a.Email, a.Username, defaultString(a.Platform, "github"), defaultString(a.Domain, "github.com"), a.SSHKeyPath, a.Description})
	if a.Alias != "" {
		m.inputs[0].Blur()
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
			in := AccountInput{OriginalAlias: m.formOriginal, Alias: v[0], Name: v[1], Email: v[2], Username: v[3], Platform: v[4], Domain: v[5], SSHKeyPath: v[6], Description: v[7]}
			if err := in.Validate(); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			m.busy = true
			return m, m.mutation("Account saved", func(ctx context.Context) error { return m.service.SaveAccount(ctx, in) })
		case ruleForm:
			v := m.values()
			if strings.TrimSpace(v[0]) == "" || strings.TrimSpace(v[1]) == "" {
				m.errText = "path and account are required"
				return m, nil
			}
			m.busy = true
			return m, m.mutation("Directory rule saved", func(ctx context.Context) error { return m.service.AddRule(ctx, Rule{Path: v[0], Account: v[1]}) })
		case discoveryAlias:
			v := m.values()
			if err := validateAlias(v[0]); err != nil {
				m.errText = err.Error()
				return m, nil
			}
			for i, d := range m.discovered {
				if i != m.cursor && strings.EqualFold(d.Alias, v[0]) {
					m.errText = "alias must be unique"
					return m, nil
				}
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

func (m Model) discoverCmd() tea.Cmd {
	return func() tea.Msg { r, e := m.service.Discover(m.ctx); return discoveredMsg{r, e} }
}
func (m Model) mutation(label string, fn func(context.Context) error) tea.Cmd {
	return func() tea.Msg { return mutationMsg{label, fn(m.ctx)} }
}
func (m *Model) executeConfirmed() tea.Cmd {
	m.busy = true
	a, v := m.confirmAction, m.confirmValue
	switch a {
	case switchAction:
		return m.mutation("Switched to "+v, func(ctx context.Context) error { return m.service.SwitchAccount(ctx, v) })
	case removeAccountAction:
		return m.mutation("Removed "+v, func(ctx context.Context) error { return m.service.RemoveAccount(ctx, v) })
	case removeRuleAction:
		return m.mutation("Rule removed", func(ctx context.Context) error { return m.service.RemoveRule(ctx, v) })
	case importAction:
		chosen := make([]Discovery, 0, len(m.selected))
		for i, d := range m.discovered {
			if m.selected[i] {
				chosen = append(chosen, d)
			}
		}
		return m.mutation("Discovery import complete", func(ctx context.Context) error { return m.service.Import(ctx, chosen) })
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

func (m Model) render() string {
	styles := newStyles(m.dark, m.noColor)
	var b strings.Builder
	b.WriteString(styles.title.Render("GitShift"))
	b.WriteString("  ")
	b.WriteString(styles.dim.Render("multi-account workspace"))
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
			b.WriteString("Current account: " + styles.good.Render(m.snapshot.Current) + "\n")
		}
		b.WriteString(fmt.Sprintf("%d accounts  |  %d directory rules\n\n", len(m.snapshot.Accounts), len(m.snapshot.Rules)))
		b.WriteString("[1] Accounts   [2] Directory rules   [3] Discover accounts\n")
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
			b.WriteString(fmt.Sprintf("%s %-20s @%-18s %s%s\n", mark, a.Alias, a.Username, a.Domain, active))
		}
		b.WriteString("\nn new  e edit  s switch  d remove  enter details\n")
	case detail:
		if a, ok := m.accountAt(); ok {
			b.WriteString(styles.heading.Render(a.Alias))
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("Name: %s\nEmail: %s\nUsername: @%s\nProvider: %s\nHost: %s\nSSH key: %s\nGPG key: %s\nDescription: %s\n", a.Name, a.Email, a.Username, a.Platform, a.Domain, emptyAsDash(a.SSHKeyPath), emptyAsDash(a.GPGKeyID), emptyAsDash(a.Description)))
			b.WriteString("\ne edit  s switch  d remove\n")
		}
	case accountForm, ruleForm, discoveryAlias:
		b.WriteString(styles.heading.Render(map[screen]string{accountForm: "Account", ruleForm: "Directory rule", discoveryAlias: "Discovery alias"}[m.screen]))
		b.WriteString("\n")
		for i, in := range m.inputs {
			prefix := "  "
			if i == m.focus {
				prefix = "> "
			}
			b.WriteString(prefix + styles.dim.Render(m.labels[i]) + "\n  " + in.View() + "\n")
		}
		b.WriteString("\ntab next  enter save  esc cancel\n")
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
			b.WriteString(fmt.Sprintf("%s %s -> %s\n", mark, r.Path, r.Account))
		}
		b.WriteString("\nn add  d remove\n")
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
			b.WriteString(fmt.Sprintf("%s [%s] %-18s @%-18s %s  %d/10\n", cur, check, d.Alias, d.Username, d.Domain, d.Confidence))
			if i == m.cursor {
				b.WriteString(styles.dim.Render("    "+d.Identity+" | "+d.Source) + "\n")
			}
		}
		b.WriteString("\nspace select  e alias  i import selected  r rescan\n")
	case help:
		b.WriteString(styles.heading.Render("Help"))
		b.WriteString("\narrow keys/j/k navigate  enter open/confirm  esc back\n? help  q quit  ctrl+c quit\n\nAccount switching and removal always require confirmation.\nDiscovery never imports until identities are selected and confirmed.\nForce switching, skipped validation, and key generation remain CLI-only.\n")
	case confirm:
		verb := map[action]string{switchAction: "Switch to", removeAccountAction: "Remove account", removeRuleAction: "Remove directory rule", importAction: "Import"}[m.confirmAction]
		b.WriteString(styles.warn.Render(fmt.Sprintf("%s %s?", verb, m.confirmValue)))
		b.WriteString("\n\ny/enter confirm  n/esc cancel\n")
	}
	if m.errText != "" {
		b.WriteString("\n" + styles.bad.Render("Error: "+m.errText) + "\n")
	} else if m.status != "" {
		b.WriteString("\n" + styles.dim.Render(m.status) + "\n")
	}
	b.WriteString("\n" + styles.dim.Render("? help  q quit"))
	return b.String()
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
		return styles{title: lipgloss.NewStyle().Bold(true), heading: lipgloss.NewStyle().Bold(true), dim: lipgloss.NewStyle(), good: lipgloss.NewStyle(), warn: lipgloss.NewStyle().Bold(true), bad: lipgloss.NewStyle().Bold(true)}
	}
	fg, muted, accent, good, warn, bad := "#1F2937", "#64748B", "#7C3AED", "#15803D", "#B45309", "#B91C1C"
	if dark {
		fg, muted, accent, good, warn, bad = "#F8FAFC", "#94A3B8", "#C4B5FD", "#4ADE80", "#FBBF24", "#F87171"
	}
	return styles{title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent)), heading: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(fg)), dim: lipgloss.NewStyle().Foreground(lipgloss.Color(muted)), good: lipgloss.NewStyle().Foreground(lipgloss.Color(good)), warn: lipgloss.NewStyle().Foreground(lipgloss.Color(warn)).Bold(true), bad: lipgloss.NewStyle().Foreground(lipgloss.Color(bad)).Bold(true)}
}
