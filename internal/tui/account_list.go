package tui

import (
	"fmt"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// AccountListModel handles the account list view
type AccountListModel struct {
	accounts       []*models.Account
	currentAccount string
	selectedIndex  int
	viewOffset     int
	height         int
}

// NewAccountListModel creates a new account list model
func NewAccountListModel(accounts []*models.Account, currentAccount string) AccountListModel {
	// Sort accounts by alias
	sorted := make([]*models.Account, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Alias < sorted[j].Alias
	})

	return AccountListModel{
		accounts:       sorted,
		currentAccount: currentAccount,
		selectedIndex:  0,
		viewOffset:     0,
		height:         20, // Default height
	}
}

// Init initializes the account list model
func (m AccountListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the account list
func (m AccountListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				// Adjust view offset if necessary
				if m.selectedIndex < m.viewOffset {
					m.viewOffset = m.selectedIndex
				}
			}

		case "down", "j":
			if m.selectedIndex < len(m.accounts)-1 {
				m.selectedIndex++
				// Adjust view offset if necessary
				maxVisible := m.height - 4 // Account for header and footer
				if m.selectedIndex >= m.viewOffset+maxVisible {
					m.viewOffset = m.selectedIndex - maxVisible + 1
				}
			}

		case "home", "g":
			m.selectedIndex = 0
			m.viewOffset = 0

		case "end", "G":
			m.selectedIndex = len(m.accounts) - 1
			maxVisible := m.height - 4
			if len(m.accounts) > maxVisible {
				m.viewOffset = len(m.accounts) - maxVisible
			}
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
	}

	return m, nil
}

// View renders the account list
func (m AccountListModel) View() string {
	if len(m.accounts) == 0 {
		return RenderCard(
			"No Accounts Found",
			RenderStatus("No accounts configured", "warning")+"\n\n"+
				InfoStyle.Render("Press 'a' to add your first account!"),
			false,
		)
	}

	// Calculate visible range
	maxVisible := m.height - 6 // Account for header, footer, and padding
	if maxVisible < 1 {
		maxVisible = 1
	}

	startIndex := m.viewOffset
	endIndex := startIndex + maxVisible
	if endIndex > len(m.accounts) {
		endIndex = len(m.accounts)
	}

	var accountCards []string

	// Render visible accounts as beautiful cards
	for i := startIndex; i < endIndex; i++ {
		account := m.accounts[i]

		// Determine styling
		isSelected := i == m.selectedIndex
		isCurrent := account.Alias == m.currentAccount

		// Build account info
		var accountDetails []string

		// Main line with alias and name
		mainLine := fmt.Sprintf("📧 %s", account.Alias)
		if account.Name != "" {
			mainLine += fmt.Sprintf(" (%s)", account.Name)
		}

		if isCurrent {
			mainLine = "⭐ " + mainLine + " (ACTIVE)"
			accountDetails = append(accountDetails, SuccessStyle.Render(mainLine))
		} else {
			accountDetails = append(accountDetails, mainLine)
		}

		// Email line
		if account.Email != "" {
			accountDetails = append(accountDetails,
				lipgloss.NewStyle().Foreground(MutedColor).Render("📧 "+account.Email))
		}

		// Description or GitHub username
		if account.Description != "" {
			accountDetails = append(accountDetails,
				lipgloss.NewStyle().Foreground(InfoColor).Render("💬 "+account.Description))
		} else if account.GitHubUsername != "" {
			accountDetails = append(accountDetails,
				lipgloss.NewStyle().Foreground(InfoColor).Render("🐙 @"+account.GitHubUsername))
		}

		// SSH key info
		if account.SSHKeyPath != "" {
			accountDetails = append(accountDetails,
				lipgloss.NewStyle().Foreground(MutedColor).Render("🔑 "+account.SSHKeyPath))
		}

		// Last used info
		if account.LastUsed != nil {
			accountDetails = append(accountDetails,
				lipgloss.NewStyle().Foreground(MutedColor).Render("🕒 Last used: "+formatTime(*account.LastUsed)))
		}

		cardContent := lipgloss.JoinVertical(lipgloss.Left, accountDetails...)

		// Create card with selection highlighting
		if isSelected {
			card := GlowStyle.Render(cardContent)
			selectionIndicator := lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true).
				Render("▶ SELECTED")
			accountCards = append(accountCards, selectionIndicator+"\n"+card)
		} else {
			card := CardStyle.Render(cardContent)
			accountCards = append(accountCards, card)
		}
	}

	// Show scrolling indicator if needed
	var scrollInfo string
	if len(m.accounts) > maxVisible {
		scrollInfo = RenderProgressBar(endIndex, len(m.accounts)) + "\n" +
			lipgloss.NewStyle().
				Foreground(MutedColor).
				Render(fmt.Sprintf("Showing %d-%d of %d accounts",
					startIndex+1, endIndex, len(m.accounts)))
	}

	// Build final content
	var content []string

	// Title
	title := RenderTitle("Select Account")
	content = append(content, title)

	// Account cards
	content = append(content, lipgloss.JoinVertical(lipgloss.Left, accountCards...))

	// Scroll info
	if scrollInfo != "" {
		content = append(content, scrollInfo)
	}

	// Summary info
	totalAccounts := len(m.accounts)
	currentAccountFound := false
	for _, account := range m.accounts {
		if account.Alias == m.currentAccount {
			currentAccountFound = true
			break
		}
	}

	var statusMessages []string
	statusMessages = append(statusMessages,
		InfoStyle.Render(fmt.Sprintf("📊 %d total accounts", totalAccounts)))

	if currentAccountFound {
		statusMessages = append(statusMessages,
			SuccessStyle.Render("⭐ Current account highlighted"))
	}

	summary := PanelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, statusMessages...))
	content = append(content, summary)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// GetSelectedAccount returns the currently selected account
func (m AccountListModel) GetSelectedAccount() *models.Account {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.accounts) {
		return m.accounts[m.selectedIndex]
	}
	return nil
}

// UpdateAccounts updates the accounts list and refreshes the view
func (m *AccountListModel) UpdateAccounts(accounts []*models.Account, currentAccount string) {
	// Sort accounts by alias
	sorted := make([]*models.Account, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Alias < sorted[j].Alias
	})

	m.accounts = sorted
	m.currentAccount = currentAccount

	// Adjust selected index if necessary
	if m.selectedIndex >= len(m.accounts) {
		m.selectedIndex = len(m.accounts) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}

	// Adjust view offset if necessary
	maxVisible := m.height - 4
	if m.viewOffset > len(m.accounts)-maxVisible {
		m.viewOffset = len(m.accounts) - maxVisible
	}
	if m.viewOffset < 0 {
		m.viewOffset = 0
	}
}

// formatTime formats a timestamp for display
func formatTime(t time.Time) string {
	now := time.Now()

	if t.After(now.Add(-24 * time.Hour)) {
		return t.Format("Today 15:04")
	}

	if t.After(now.Add(-7 * 24 * time.Hour)) {
		return t.Format("Mon 15:04")
	}

	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}

	return t.Format("Jan 02, 2006")
}
