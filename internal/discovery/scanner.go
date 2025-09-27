package discovery

import (
	"github.com/techishthoughts/GitPersona/internal/models"
)

// AccountDiscovery handles automatic detection of existing Git accounts
type AccountDiscovery struct {
	homeDir string
}

// NewAccountDiscovery creates a new account discovery service
func NewAccountDiscovery() *AccountDiscovery {
	return &AccountDiscovery{}
}

// DiscoveredAccount represents an account found during discovery
type DiscoveredAccount struct {
	*models.Account
	Source      string // where it was found
	Confidence  int    // confidence level (1-10)
	Conflicting bool   // if there are conflicting accounts
}

// ScanExistingAccounts scans for existing SSH keys ONLY (no GitHub API/CLI)
func (d *AccountDiscovery) ScanExistingAccounts() ([]*DiscoveredAccount, error) {
	// Use the new SSH-only scanner
	sshScanner := NewSSHOnlyScanner()
	return sshScanner.ScanSSHKeys()
}
