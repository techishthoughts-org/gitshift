package tui

import (
	"time"

	"github.com/thukabjj/GitPersona/internal/models"
)

// Messages for the TUI application

// AccountSelectedMsg is sent when an account is selected
type AccountSelectedMsg struct {
	Account interface{}
}

// AccountSwitchedMsg is sent when successfully switching accounts
type AccountSwitchedMsg struct {
	AccountAlias string
}

// AccountAddedMsg is sent when an account is successfully added
type AccountAddedMsg struct {
	Account *models.Account
}

// AccountRemovedMsg is sent when an account is successfully removed
type AccountRemovedMsg struct {
	AccountAlias string
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Error string
}

// ConfirmationResultMsg is sent with the result of a confirmation dialog
type ConfirmationResultMsg struct {
	Confirmed bool
	Action    string
	Data      interface{}
}

// TickMsg is sent for animation updates
type TickMsg time.Time
