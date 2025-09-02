package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// Run starts the TUI application
func Run() error {
	model := NewModel()

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// SelectAccount shows a TUI to select an account and returns the selected account
func SelectAccount(accounts []*models.Account, currentAccount string) (*models.Account, error) {
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	// Create a simple selection model
	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: currentAccount,
		selectedIndex:  0,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running account selection: %w", err)
	}

	if selectionModel, ok := finalModel.(*SelectionModel); ok {
		return selectionModel.GetSelectedAccount(), nil
	}

	return nil, fmt.Errorf("failed to get selected account")
}

// SelectionModel is a simple model for account selection
type SelectionModel struct {
	accounts        []*models.Account
	currentAccount  string
	selectedIndex   int
	selectedAccount *models.Account
	quitting        bool
}

// Init initializes the selection model
func (m *SelectionModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the selection model
func (m *SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}

		case "down", "j":
			if m.selectedIndex < len(m.accounts)-1 {
				m.selectedIndex++
			}

		case "enter", " ":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.accounts) {
				m.selectedAccount = m.accounts[m.selectedIndex]
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the selection interface
func (m *SelectionModel) View() string {
	if m.quitting {
		return ""
	}

	var content string
	content += "Select an account:\n\n"

	for i, account := range m.accounts {
		cursor := " "
		if i == m.selectedIndex {
			cursor = ">"
		}

		marker := " "
		if account.Alias == m.currentAccount {
			marker = "*"
		}

		content += fmt.Sprintf("%s%s %s (%s - %s)\n",
			cursor, marker, account.Alias, account.Name, account.Email)
	}

	content += "\nUse ↑/↓ to navigate, Enter to select, q to quit"
	return content
}

// GetSelectedAccount returns the selected account
func (m *SelectionModel) GetSelectedAccount() *models.Account {
	return m.selectedAccount
}
