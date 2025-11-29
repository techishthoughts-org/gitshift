package discovery

import (
	"fmt"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

// AccountDiscovery handles automatic detection of existing Git accounts
type AccountDiscovery struct {
	// No fields needed as we're using scanners for all operations
}

// NewAccountDiscovery creates a new account discovery service
func NewAccountDiscovery() *AccountDiscovery {
	return &AccountDiscovery{}
}

// DiscoveredAccount represents an account found during discovery
type DiscoveredAccount struct {
	*models.Account
	Source      string // where it was found ("ssh", "gpg", "ssh+gpg")
	Confidence  int    // confidence level (1-10)
	Conflicting bool   // if there are conflicting accounts
}

// ScanExistingAccounts scans for existing SSH keys and GPG keys
func (d *AccountDiscovery) ScanExistingAccounts() ([]*DiscoveredAccount, error) {
	fmt.Println("🔍 Starting account discovery...")
	fmt.Println()

	// Scan SSH keys
	sshScanner := NewSSHOnlyScanner()
	sshAccounts, err := sshScanner.ScanSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("SSH scan failed: %w", err)
	}
	fmt.Println()

	// Scan GPG keys
	gpgScanner := NewGPGScanner()
	gpgAccounts, err := gpgScanner.ScanGPGKeys()
	if err != nil {
		return nil, fmt.Errorf("GPG scan failed: %w", err)
	}
	fmt.Println()

	// Merge accounts by email
	merged := d.mergeAccounts(sshAccounts, gpgAccounts)

	fmt.Printf("🎯 Discovery complete: %d account(s) found\n", len(merged))
	return merged, nil
}

// mergeAccounts merges SSH and GPG discovered accounts by matching email addresses
func (d *AccountDiscovery) mergeAccounts(sshAccounts, gpgAccounts []*DiscoveredAccount) []*DiscoveredAccount {
	var result []*DiscoveredAccount

	// Create a map of GPG accounts by email for quick lookup
	gpgByEmail := make(map[string]*DiscoveredAccount)
	for _, gpgAcc := range gpgAccounts {
		if gpgAcc.Email != "" {
			gpgByEmail[strings.ToLower(gpgAcc.Email)] = gpgAcc
		}
	}

	// Track which GPG accounts have been merged
	mergedGPG := make(map[string]bool)

	// Process SSH accounts and try to match with GPG
	for _, sshAcc := range sshAccounts {
		emailKey := strings.ToLower(sshAcc.Email)

		if gpgAcc, found := gpgByEmail[emailKey]; found {
			// Merge SSH and GPG into one account
			merged := d.mergeSingleAccount(sshAcc, gpgAcc)
			result = append(result, merged)
			mergedGPG[emailKey] = true

			fmt.Printf("🔗 Matched SSH + GPG for %s (%s)\n", merged.Alias, merged.Email)
		} else {
			// SSH-only account
			result = append(result, sshAcc)
		}
	}

	// Add GPG-only accounts (those not merged with SSH)
	for email, gpgAcc := range gpgByEmail {
		if !mergedGPG[email] {
			result = append(result, gpgAcc)
		}
	}

	return result
}

// mergeSingleAccount merges an SSH account and GPG account into one
func (d *AccountDiscovery) mergeSingleAccount(sshAcc, gpgAcc *DiscoveredAccount) *DiscoveredAccount {
	// Start with SSH account as base
	merged := &DiscoveredAccount{
		Account: &models.Account{
			Alias:              sshAcc.Alias,
			Name:               sshAcc.Name,
			Email:              sshAcc.Email,
			GitHubUsername:     sshAcc.GitHubUsername,
			SSHKeyPath:         sshAcc.SSHKeyPath,
			GPGKeyID:           gpgAcc.GPGKeyID,
			GPGKeyFingerprint:  gpgAcc.GPGKeyFingerprint,
			GPGKeyType:         gpgAcc.GPGKeyType,
			GPGKeySize:         gpgAcc.GPGKeySize,
			GPGKeyExpiry:       gpgAcc.GPGKeyExpiry,
			GPGEnabled:         true,
			Platform:           detectPlatform(sshAcc.Email),
			Description:        "Discovered from SSH key and GPG keyring",
		},
		Source:     "ssh+gpg",
		Confidence: max(sshAcc.Confidence, gpgAcc.Confidence) + 1, // Bonus for having both
	}

	// Use GPG name if it's more complete than SSH name
	if len(gpgAcc.Name) > len(sshAcc.Name) {
		merged.Name = gpgAcc.Name
	}

	return merged
}

// detectPlatform attempts to detect the Git platform from email domain
func detectPlatform(email string) string {
	if email == "" {
		return "github" // Default to GitHub
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "github"
	}
	domain := strings.ToLower(parts[1])

	// Platform detection based on email domain
	if strings.Contains(domain, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(domain, "github") {
		return "github"
	}
	if strings.Contains(domain, "bitbucket") {
		return "bitbucket"
	}

	// For corporate/work emails, try to detect based on common patterns
	// If it's a work email (not a common public email provider), default to GitLab
	// as many companies use GitLab for self-hosted git
	commonProviders := []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "icloud.com"}
	isCommonProvider := false
	for _, provider := range commonProviders {
		if domain == provider {
			isCommonProvider = true
			break
		}
	}

	if !isCommonProvider {
		// Corporate email - could be GitLab or GitHub Enterprise
		// Default to GitLab for now, user can change if needed
		return "gitlab"
	}

	// Default to GitHub for personal email providers
	return "github"
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
