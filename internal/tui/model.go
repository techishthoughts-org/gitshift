package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/techishthoughts/GitPersona/internal/config"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// ViewState represents the current view in the TUI
type ViewState int

const (
	MainView ViewState = iota
	AccountListView
	AccountDetailView
	AddAccountView
	ConfirmationView
)

// Model represents the main TUI model
type Model struct {
	// Configuration
	configManager *config.Manager
	config        *models.Config

	// UI State
	currentView  ViewState
	previousView ViewState
	width        int
	height       int

	// Data
	accounts       []*models.Account
	selectedIndex  int
	currentAccount string

	// Sub-models
	accountList   AccountListModel
	accountDetail AccountDetailModel
	addAccount    AddAccountModel
	confirmation  ConfirmationModel

	// Messages
	message      string
	errorMessage string

	// Flags
	quitting bool
	loading  bool

	// Animation
	spinnerFrame int
	lastTick     time.Time
}

// NewModel creates a new TUI model
func NewModel() *Model {
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		// Handle error appropriately
		fmt.Printf("Failed to load config: %v\n", err)
	}

	accounts := configManager.ListAccounts()
	currentAccount := configManager.GetConfig().CurrentAccount

	m := &Model{
		configManager:  configManager,
		config:         configManager.GetConfig(),
		currentView:    MainView,
		accounts:       accounts,
		currentAccount: currentAccount,
		selectedIndex:  0,
	}

	// Initialize sub-models
	m.accountList = NewAccountListModel(accounts, currentAccount)
	m.accountDetail = NewAccountDetailModel()
	m.addAccount = NewAddAccountModel()
	m.confirmation = NewConfirmationModel()

	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("GitPersona"),
		m.tickCmd(),
	)
}

// tickCmd creates a command for animation ticks
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*200, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update sub-models with new size
		m.accountList.height = msg.Height - 10 // Reserve space for header/footer
		return m, nil

	case TickMsg:
		// Handle animation updates
		m.spinnerFrame++
		m.lastTick = time.Time(msg)
		return m, m.tickCmd()

	case AccountSelectedMsg:
		if account, ok := msg.Account.(*models.Account); ok {
			return m.handleAccountSelection(account)
		}

	case AccountSwitchedMsg:
		m.message = fmt.Sprintf("Switched to account '%s'", msg.AccountAlias)
		m.currentAccount = msg.AccountAlias
		m.accounts = m.configManager.ListAccounts() // Refresh accounts
		m.accountList.UpdateAccounts(m.accounts, m.currentAccount)
		m.loading = false
		return m, nil

	case ErrorMsg:
		m.errorMessage = msg.Error
		m.loading = false
		return m, nil

	case ConfirmationResultMsg:
		if msg.Confirmed {
			return m.handleConfirmationResult(msg)
		} else {
			m.currentView = m.previousView
			return m, nil
		}
	}

	// Update sub-models based on current view
	switch m.currentView {
	case AccountListView:
		newModel, cmd := m.accountList.Update(msg)
		m.accountList = newModel.(AccountListModel)
		return m, cmd

	case AccountDetailView:
		newModel, cmd := m.accountDetail.Update(msg)
		m.accountDetail = newModel.(AccountDetailModel)
		return m, cmd

	case AddAccountView:
		newModel, cmd := m.addAccount.Update(msg)
		m.addAccount = newModel.(AddAccountModel)
		return m, cmd

	case ConfirmationView:
		newModel, cmd := m.confirmation.Update(msg)
		m.confirmation = newModel.(ConfirmationModel)
		return m, cmd
	}

	return m, cmd
}

// View renders the current view
func (m *Model) View() string {
	if m.quitting {
		return RenderStatus("Thanks for using GitPersona! 👋", "success") + "\n"
	}

	// Calculate available space
	contentHeight := m.height - 8 // Reserve space for header and footer

	var content string

	switch m.currentView {
	case MainView:
		content = m.renderMainView()
	case AccountListView:
		content = m.accountList.View()
	case AccountDetailView:
		content = m.accountDetail.View()
	case AddAccountView:
		content = m.addAccount.View()
	case ConfirmationView:
		content = m.confirmation.View()
	}

	// Add loading overlay if needed
	if m.loading {
		loadingOverlay := m.renderLoadingOverlay()
		content = lipgloss.JoinVertical(lipgloss.Center, content, loadingOverlay)
	}

	// Add header and footer
	header := m.renderHeader()
	footer := m.renderFooter()

	// Create the main layout
	mainContent := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)

	return BaseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"",
			mainContent,
			"",
			footer,
		),
	)
}

// renderHeader renders the application header
func (m *Model) renderHeader() string {
	// Main title with gradient
	title := RenderTitle("GitPersona")

	// Current account status
	var statusInfo string
	if m.currentAccount != "" {
		if account, err := m.configManager.GetAccount(m.currentAccount); err == nil {
			statusInfo = RenderStatus(
				fmt.Sprintf("Active: %s (%s)", account.Alias, account.Name),
				"success",
			)
		}
	} else {
		statusInfo = RenderStatus("No account active", "warning")
	}

	// View indicator
	viewName := m.getViewName()
	viewIndicator := InfoStyle.Render("📍 " + viewName)

	// Join elements based on width
	if m.width > 120 {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			title,
			lipgloss.NewStyle().Width(5).Render(""),
			statusInfo,
			lipgloss.NewStyle().Width(5).Render(""),
			viewIndicator,
		)
	} else {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			lipgloss.JoinHorizontal(lipgloss.Top, statusInfo, "  ", viewIndicator),
		)
	}
}

// getViewName returns a human-readable name for the current view
func (m *Model) getViewName() string {
	switch m.currentView {
	case MainView:
		return "Dashboard"
	case AccountListView:
		return "Account List"
	case AccountDetailView:
		return "Account Details"
	case AddAccountView:
		return "Add Account"
	case ConfirmationView:
		return "Confirmation"
	default:
		return "Unknown"
	}
}

// renderFooter renders the application footer with help text
func (m *Model) renderFooter() string {
	var bindings map[string]string

	switch m.currentView {
	case MainView:
		bindings = map[string]string{
			"l": "list accounts",
			"a": "add account",
			"s": "switch account",
			"c": "current status",
			"q": "quit",
		}
	case AccountListView:
		bindings = map[string]string{
			"↑/↓":   "navigate",
			"enter": "switch",
			"i":     "details",
			"a":     "add",
			"d":     "delete",
			"b":     "back",
			"q":     "quit",
		}
	case AccountDetailView:
		bindings = map[string]string{
			"s": "switch to account",
			"d": "delete account",
			"b": "back",
			"q": "quit",
		}
	case AddAccountView:
		bindings = map[string]string{
			"tab":    "next field",
			"ctrl+s": "save",
			"esc":    "cancel",
		}
	case ConfirmationView:
		bindings = map[string]string{
			"y":   "yes",
			"n":   "no",
			"esc": "cancel",
		}
	default:
		bindings = map[string]string{
			"q": "quit",
		}
	}

	return RenderHelpBar(bindings)
}

// renderLoadingOverlay creates a loading animation overlay
func (m *Model) renderLoadingOverlay() string {
	spinner := RenderSpinner(m.spinnerFrame)
	loadingText := "Loading..."

	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		spinner,
		" ",
		InfoStyle.Render(loadingText),
	)

	return PanelStyle.Render(content)
}

// renderMainView renders the main dashboard view
func (m *Model) renderMainView() string {
	accountCount := len(m.accounts)

	// Create the dashboard layout
	dashboard := RenderDashboard(m.currentAccount, accountCount, m.width)

	var sections []string
	sections = append(sections, dashboard)

	// Quick actions menu
	menuOptions := []string{
		"📋 List Accounts",
		"➕ Add Account",
		"🔄 Switch Account",
		"📊 Current Status",
	}

	if accountCount > 0 {
		quickActions := RenderCard(
			"Quick Actions",
			RenderMenu(menuOptions, -1), // -1 means no selection
			false,
		)
		sections = append(sections, quickActions)
	}

	// Account preview
	if accountCount == 0 {
		emptyState := RenderCard(
			"Getting Started",
			RenderStatus("No GitHub accounts configured yet", "warning")+"\n\n"+
				InfoStyle.Render("Press 'a' to add your first account and get started!"),
			false,
		)
		sections = append(sections, emptyState)
	} else {
		// Show recent accounts
		var recentAccounts []string
		displayCount := 3
		if accountCount < displayCount {
			displayCount = accountCount
		}

		for _, account := range m.accounts[:displayCount] {
			marker := "  "
			style := UnselectedItemStyle

			if account.Alias == m.currentAccount {
				marker = "▶ "
				style = CurrentItemStyle
			}

			accountInfo := fmt.Sprintf("%s%s", marker, account.Alias)
			if account.Name != "" {
				accountInfo += fmt.Sprintf(" (%s)", account.Name)
			}
			if account.Email != "" {
				accountInfo += fmt.Sprintf(" <%s>", account.Email)
			}

			recentAccounts = append(recentAccounts, style.Render(accountInfo))
		}

		if accountCount > displayCount {
			moreText := fmt.Sprintf("... and %d more accounts", accountCount-displayCount)
			recentAccounts = append(recentAccounts, lipgloss.NewStyle().
				Foreground(MutedColor).
				Render(moreText))
		}

		accountsCard := RenderCard(
			"Your Accounts",
			lipgloss.JoinVertical(lipgloss.Left, recentAccounts...),
			false,
		)
		sections = append(sections, accountsCard)
	}

	// Status messages
	var messages []string
	if m.message != "" {
		messages = append(messages, RenderStatus(m.message, "success"))
		// Clear message after showing
		m.message = ""
	}

	if m.errorMessage != "" {
		messages = append(messages, RenderStatus(m.errorMessage, "error"))
		// Clear error after showing
		m.errorMessage = ""
	}

	if len(messages) > 0 {
		messageCard := RenderCard(
			"Status",
			lipgloss.JoinVertical(lipgloss.Left, messages...),
			false,
		)
		sections = append(sections, messageCard)
	}

	// Tips section
	tips := []string{
		"💡 Use 'gitpersona init' in your shell for automatic switching",
		"📁 Set project accounts with 'gitpersona project set <alias>'",
		"🔑 Add SSH keys to accounts for secure authentication",
	}

	tipsCard := RenderCard(
		"💡 Tips & Tricks",
		lipgloss.JoinVertical(lipgloss.Left, tips...),
		false,
	)
	sections = append(sections, tipsCard)

	// Layout sections based on screen width
	if m.width > 120 {
		// Wide layout - arrange in columns
		var leftColumn, rightColumn []string
		for i, section := range sections {
			if i%2 == 0 {
				leftColumn = append(leftColumn, section)
			} else {
				rightColumn = append(rightColumn, section)
			}
		}

		left := lipgloss.JoinVertical(lipgloss.Left, leftColumn...)
		right := lipgloss.JoinVertical(lipgloss.Left, rightColumn...)

		return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	} else {
		// Narrow layout - single column
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
}
