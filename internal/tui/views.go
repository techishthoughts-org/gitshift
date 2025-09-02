package tui

import (
	"fmt"

	"github.com/thukabjj/GitPersona/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AccountDetailModel handles the account detail view
type AccountDetailModel struct {
	account *models.Account
}

// NewAccountDetailModel creates a new account detail model
func NewAccountDetailModel() AccountDetailModel {
	return AccountDetailModel{}
}

// SetAccount sets the account to display
func (m *AccountDetailModel) SetAccount(account *models.Account) {
	m.account = account
}

// GetAccount returns the current account
func (m AccountDetailModel) GetAccount() *models.Account {
	return m.account
}

// Init initializes the account detail model
func (m AccountDetailModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the account detail view
func (m AccountDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the account detail view
func (m AccountDetailModel) View() string {
	if m.account == nil {
		return "No account selected"
	}

	var content []string

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		Render(fmt.Sprintf("Account Details: %s", m.account.Alias))
	content = append(content, header)
	content = append(content, "")

	// Account information
	content = append(content, fmt.Sprintf("Alias: %s", m.account.Alias))
	content = append(content, fmt.Sprintf("Name: %s", m.account.Name))
	content = append(content, fmt.Sprintf("Email: %s", m.account.Email))

	if m.account.GitHubUsername != "" {
		content = append(content, fmt.Sprintf("GitHub Username: @%s", m.account.GitHubUsername))
	}

	if m.account.SSHKeyPath != "" {
		content = append(content, fmt.Sprintf("SSH Key Path: %s", m.account.SSHKeyPath))
	}

	if m.account.Description != "" {
		content = append(content, fmt.Sprintf("Description: %s", m.account.Description))
	}

	content = append(content, fmt.Sprintf("Default Account: %t", m.account.IsDefault))
	content = append(content, fmt.Sprintf("Created: %s", formatTime(m.account.CreatedAt)))

	if m.account.LastUsed != nil {
		content = append(content, fmt.Sprintf("Last Used: %s", formatTime(*m.account.LastUsed)))
	}

	content = append(content, "")
	content = append(content, "Press 's' to switch to this account, 'd' to delete, 'b' to go back")

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// AddAccountModel handles the add account form
type AddAccountModel struct {
	inputs []string
	labels []string
	cursor int
	values map[string]string
}

// NewAddAccountModel creates a new add account model
func NewAddAccountModel() AddAccountModel {
	labels := []string{
		"Alias (required)",
		"Full Name (required)",
		"Email (required)",
		"GitHub Username (optional)",
		"SSH Key Path (optional)",
		"Description (optional)",
	}

	return AddAccountModel{
		labels: labels,
		cursor: 0,
		values: make(map[string]string),
	}
}

// Reset resets the form
func (m *AddAccountModel) Reset() {
	m.cursor = 0
	m.values = make(map[string]string)
}

// Init initializes the add account model
func (m AddAccountModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the add account form
func (m AddAccountModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "shift+tab":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "tab":
			if m.cursor < len(m.labels)-1 {
				m.cursor++
			}
		case "backspace":
			key := m.getKeyForIndex(m.cursor)
			if len(m.values[key]) > 0 {
				m.values[key] = m.values[key][:len(m.values[key])-1]
			}
		default:
			// Add character to current field
			if len(msg.String()) == 1 && msg.String() != " " && msg.String() != "\t" && msg.String() != "\n" {
				key := m.getKeyForIndex(m.cursor)
				m.values[key] += msg.String()
			} else if msg.String() == " " {
				key := m.getKeyForIndex(m.cursor)
				m.values[key] += " "
			}
		}
	}

	return m, nil
}

// View renders the add account form
func (m AddAccountModel) View() string {
	var content []string

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		Render("Add New GitHub Account")
	content = append(content, header)
	content = append(content, "")

	// Form fields
	for i, label := range m.labels {
		key := m.getKeyForIndex(i)
		value := m.values[key]

		var fieldStyle lipgloss.Style
		if i == m.cursor {
			fieldStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00ff87"))
		} else {
			fieldStyle = lipgloss.NewStyle()
		}

		fieldLine := fmt.Sprintf("%s: %s", label, value)
		if i == m.cursor {
			fieldLine += "█" // Cursor
		}

		content = append(content, fieldStyle.Render(fieldLine))
	}

	content = append(content, "")
	content = append(content, "Use Tab/Shift+Tab to navigate fields")
	content = append(content, "Press Ctrl+S to save, Esc to cancel")

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// GetAccount returns the account from the form data
func (m AddAccountModel) GetAccount() *models.Account {
	alias := m.values["alias"]
	name := m.values["name"]
	email := m.values["email"]

	if alias == "" || name == "" || email == "" {
		return nil
	}

	account := models.NewAccount(alias, name, email, m.values["ssh_key"])
	account.GitHubUsername = m.values["github_username"]
	account.Description = m.values["description"]

	return account
}

// getKeyForIndex returns the key name for a given field index
func (m AddAccountModel) getKeyForIndex(index int) string {
	keys := []string{"alias", "name", "email", "github_username", "ssh_key", "description"}
	if index >= 0 && index < len(keys) {
		return keys[index]
	}
	return ""
}

// ConfirmationModel handles confirmation dialogs
type ConfirmationModel struct {
	title       string
	description string
	action      string
	data        interface{}
}

// NewConfirmationModel creates a new confirmation model
func NewConfirmationModel() ConfirmationModel {
	return ConfirmationModel{}
}

// SetConfirmation sets up the confirmation dialog
func (m *ConfirmationModel) SetConfirmation(title, description, action string, data interface{}) {
	m.title = title
	m.description = description
	m.action = action
	m.data = data
}

// GetAction returns the current action
func (m ConfirmationModel) GetAction() string {
	return m.action
}

// GetData returns the current data
func (m ConfirmationModel) GetData() interface{} {
	return m.data
}

// Init initializes the confirmation model
func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the confirmation dialog
func (m ConfirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the confirmation dialog
func (m ConfirmationModel) View() string {
	var content []string

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff5555")).
		Render(m.title)
	content = append(content, title)
	content = append(content, "")

	// Description
	if m.description != "" {
		content = append(content, m.description)
		content = append(content, "")
	}

	// Confirmation prompt
	content = append(content, "Are you sure you want to proceed?")
	content = append(content, "")
	content = append(content, "Press 'y' to confirm, 'n' to cancel")

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}
