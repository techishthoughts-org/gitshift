package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/techishthoughts/GitPersona/internal/git"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// handleKeyPress handles key press events based on current view
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.currentView == MainView {
			m.quitting = true
			return m, tea.Quit
		} else {
			// Return to main view from other views
			m.currentView = MainView
			return m, nil
		}

	case "esc":
		// Go back to previous view or main view
		if m.currentView != MainView {
			m.currentView = m.previousView
			if m.currentView == MainView {
				m.currentView = MainView
			}
		}
		return m, nil
	}

	// Handle view-specific key presses
	switch m.currentView {
	case MainView:
		return m.handleMainViewKeys(msg)
	case AccountListView:
		return m.handleAccountListKeys(msg)
	case AccountDetailView:
		return m.handleAccountDetailKeys(msg)
	case AddAccountView:
		return m.handleAddAccountKeys(msg)

	case HelpView:
		return m.handleHelpViewKeys(msg)

	case ConfirmationView:
		return m.handleConfirmationKeys(msg)
	}

	return m, nil
}

// handleMainViewKeys handles key presses in the main view
func (m *Model) handleMainViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "l", "L":
		// List accounts
		m.previousView = m.currentView
		m.currentView = AccountListView
		return m, nil

	case "a", "A":
		// Add account
		m.previousView = m.currentView
		m.currentView = AddAccountView
		m.addAccount.Reset()
		return m, nil

	case "s", "S":
		// Quick switch - show account list
		if len(m.accounts) > 0 {
			m.previousView = m.currentView
			m.currentView = AccountListView
			return m, nil
		} else {
			m.errorMessage = "No accounts configured. Press 'a' to add an account."
			return m, nil
		}

	case "c", "C":
		// Show current account details
		if m.currentAccount != "" {
			if account, err := m.configManager.GetAccount(m.currentAccount); err == nil {
				m.previousView = m.currentView
				m.currentView = AccountDetailView
				m.accountDetail.SetAccount(account)
				return m, nil
			}
		} else {
			m.errorMessage = "No account currently active."
			return m, nil
		}

	case "h", "H":
		// Show help menu
		m.previousView = m.currentView
		m.currentView = HelpView
		return m, nil
	}

	return m, nil
}

// handleHelpViewKeys handles key presses in the help view
func (m *Model) handleHelpViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "h", "H", "esc", "b", "B":
		// Back to previous view
		m.currentView = m.previousView
		return m, nil

	case "q", "Q":
		// Quit
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// handleAccountListKeys handles key presses in the account list view
func (m *Model) handleAccountListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		// Switch to selected account
		selectedAccount := m.accountList.GetSelectedAccount()
		if selectedAccount != nil {
			return m.handleAccountSelection(selectedAccount)
		}

	case "d", "D":
		// Delete selected account
		selectedAccount := m.accountList.GetSelectedAccount()
		if selectedAccount != nil {
			m.previousView = m.currentView
			m.currentView = ConfirmationView
			m.confirmation.SetConfirmation(
				fmt.Sprintf("Delete account '%s'?", selectedAccount.Alias),
				fmt.Sprintf("This will permanently remove the account '%s' (%s - %s) from your configuration.",
					selectedAccount.Alias, selectedAccount.Name, selectedAccount.Email),
				"delete_account",
				selectedAccount,
			)
			return m, nil
		}

	case "i", "I":
		// Show account info/details
		selectedAccount := m.accountList.GetSelectedAccount()
		if selectedAccount != nil {
			m.previousView = m.currentView
			m.currentView = AccountDetailView
			m.accountDetail.SetAccount(selectedAccount)
			return m, nil
		}

	case "a", "A":
		// Add new account
		m.previousView = m.currentView
		m.currentView = AddAccountView
		m.addAccount.Reset()
		return m, nil

	case "b", "B":
		// Back to main view
		m.currentView = MainView
		return m, nil
	}

	return m, nil
}

// handleAccountDetailKeys handles key presses in the account detail view
func (m *Model) handleAccountDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "b", "B":
		// Back to previous view
		m.currentView = m.previousView
		return m, nil

	case "s", "S":
		// Switch to this account
		account := m.accountDetail.GetAccount()
		if account != nil {
			return m.handleAccountSelection(account)
		}

	case "d", "D":
		// Delete this account
		account := m.accountDetail.GetAccount()
		if account != nil {
			m.previousView = m.currentView
			m.currentView = ConfirmationView
			m.confirmation.SetConfirmation(
				fmt.Sprintf("Delete account '%s'?", account.Alias),
				fmt.Sprintf("This will permanently remove the account '%s' (%s - %s) from your configuration.",
					account.Alias, account.Name, account.Email),
				"delete_account",
				account,
			)
			return m, nil
		}
	}

	return m, nil
}

// handleAddAccountKeys handles key presses in the add account view
func (m *Model) handleAddAccountKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		// Save the account
		account := m.addAccount.GetAccount()
		if account != nil {
			if err := account.Validate(); err != nil {
				m.errorMessage = fmt.Sprintf("Validation failed: %v", err)
				return m, nil
			}

			// Validate SSH key if provided
			if account.SSHKeyPath != "" {
				gitManager := git.NewManager()
				if err := gitManager.ValidateSSHKey(account.SSHKeyPath); err != nil {
					m.errorMessage = fmt.Sprintf("SSH key validation failed: %v", err)
					return m, nil
				}
			}

			// Add the account
			if err := m.configManager.AddAccount(account); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to add account: %v", err)
				return m, nil
			}

			// Refresh accounts list
			m.accounts = m.configManager.ListAccounts()
			m.accountList.UpdateAccounts(m.accounts, m.currentAccount)

			// Show success message and return to main view
			m.message = fmt.Sprintf("Successfully added account '%s'", account.Alias)
			m.currentView = MainView
			return m, nil
		} else {
			m.errorMessage = "Please fill in all required fields"
			return m, nil
		}

	case "esc":
		// Cancel and go back
		m.currentView = m.previousView
		return m, nil
	}

	return m, nil
}

// handleConfirmationKeys handles key presses in the confirmation view
func (m *Model) handleConfirmationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		// Confirm
		return m, func() tea.Msg {
			return ConfirmationResultMsg{
				Confirmed: true,
				Action:    m.confirmation.GetAction(),
				Data:      m.confirmation.GetData(),
			}
		}

	case "n", "N", "esc":
		// Cancel
		m.currentView = m.previousView
		return m, nil
	}

	return m, nil
}

// handleAccountSelection handles account selection and switching
func (m *Model) handleAccountSelection(account *models.Account) (tea.Model, tea.Cmd) {
	// Set loading state
	m.loading = true

	gitManager := git.NewManager()

	// Determine if we should set global or local config
	useGlobal := m.config.GlobalGitConfig

	var err error
	if useGlobal {
		err = gitManager.SetGlobalConfig(account)
	} else {
		// Check if we're in a git repository
		currentDir := "."
		if !gitManager.IsGitRepo(currentDir) {
			m.errorMessage = "Not in a git repository. Enable global config in settings or switch to a git repository."
			return m, nil
		}
		err = gitManager.SetLocalConfig(account)
	}

	if err != nil {
		m.errorMessage = fmt.Sprintf("Failed to set git config: %v", err)
		return m, nil
	}

	// Update current account in config
	if err := m.configManager.SetCurrentAccount(account.Alias); err != nil {
		m.errorMessage = fmt.Sprintf("Failed to update current account: %v", err)
		return m, nil
	}

	// Return success message
	return m, func() tea.Msg {
		return AccountSwitchedMsg{AccountAlias: account.Alias}
	}
}

// handleConfirmationResult handles the result of confirmation dialogs
func (m *Model) handleConfirmationResult(msg ConfirmationResultMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case "delete_account":
		if account, ok := msg.Data.(*models.Account); ok {
			if err := m.configManager.RemoveAccount(account.Alias); err != nil {
				m.errorMessage = fmt.Sprintf("Failed to remove account: %v", err)
				m.currentView = m.previousView
				return m, nil
			}

			// Refresh accounts list
			m.accounts = m.configManager.ListAccounts()
			m.currentAccount = m.configManager.GetConfig().CurrentAccount
			m.accountList.UpdateAccounts(m.accounts, m.currentAccount)

			// Show success message
			m.message = fmt.Sprintf("Successfully removed account '%s'", account.Alias)
			m.currentView = MainView
			return m, nil
		}
	}

	m.currentView = m.previousView
	return m, nil
}
