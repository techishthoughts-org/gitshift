package discovery

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

type AccountDiscovery struct{ output io.Writer }

func NewAccountDiscovery() *AccountDiscovery { return &AccountDiscovery{output: os.Stdout} }

// NewSilentAccountDiscovery returns a scanner suitable for full-screen UIs.
func NewSilentAccountDiscovery() *AccountDiscovery { return &AccountDiscovery{output: io.Discard} }

// DiscoveredAccount represents an account found during discovery. IdentityKey
// is populated for provider-authenticated identities and has the canonical
// provider|hostname|username form.
type DiscoveredAccount struct {
	*models.Account
	Source      string
	Confidence  int
	Conflicting bool
	IdentityKey string
}

func (d *AccountDiscovery) ScanExistingAccounts() ([]*DiscoveredAccount, error) {
	fmt.Fprintln(d.output, "🔍 Starting account discovery...")
	fmt.Fprintln(d.output)

	githubScanner := NewGitHubCLIScanner()
	githubScanner.output = d.output
	githubAccounts, err := githubScanner.ScanAccounts()
	if err != nil {
		return nil, fmt.Errorf("GitHub CLI scan failed: %w", err)
	}
	fmt.Fprintln(d.output)
	sshScanner := NewSSHOnlyScanner()
	sshScanner.output = d.output
	sshAccounts, err := sshScanner.ScanSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("SSH scan failed: %w", err)
	}
	fmt.Fprintln(d.output)
	gpgScanner := NewGPGScanner()
	gpgScanner.output = d.output
	gpgAccounts, err := gpgScanner.ScanGPGKeys()
	if err != nil {
		return nil, fmt.Errorf("GPG scan failed: %w", err)
	}
	fmt.Fprintln(d.output)

	merged := d.mergeAccounts(githubAccounts, sshAccounts, gpgAccounts)
	fmt.Fprintf(d.output, "🎯 Discovery complete: %d account(s) found\n", len(merged))
	return merged, nil
}

// mergeAccounts combines evidence only when an email maps one-to-one. An
// email shared by multiple provider identities or multiple keys is ambiguous,
// so those records remain separate rather than being attached arbitrarily.
func (d *AccountDiscovery) mergeAccounts(groups ...[]*DiscoveredAccount) []*DiscoveredAccount {
	if len(groups) == 2 { // compatibility for older package tests: SSH, GPG
		groups = append([][]*DiscoveredAccount{nil}, groups...)
	}
	for len(groups) < 3 {
		groups = append(groups, nil)
	}
	providers, sshAccounts, gpgAccounts := groups[0], groups[1], groups[2]
	result := append([]*DiscoveredAccount(nil), providers...)
	usedSSH := make(map[int]bool)
	usedGPG := make(map[int]bool)

	providerByEmail := indexesByEmail(providers)
	sshByEmail := indexesByEmail(sshAccounts)
	gpgByEmail := indexesByEmail(gpgAccounts)
	providerByUsername := indexesByUsername(providers)
	sshByUsername := indexesByUsername(sshAccounts)

	// An SSH key filename can provide stronger account evidence than an email,
	// but only when that username identifies exactly one provider identity (and
	// therefore one host) and exactly one SSH key.
	for username, providerIndexes := range providerByUsername {
		sshIndexes := sshByUsername[username]
		if len(providerIndexes) != 1 || len(sshIndexes) != 1 {
			continue
		}
		pi, si := providerIndexes[0], sshIndexes[0]
		result[pi] = mergeDiscovered(result[pi], sshAccounts[si])
		usedSSH[si] = true
	}

	for email, providerIndexes := range providerByEmail {
		if len(providerIndexes) != 1 {
			continue
		}
		pi := providerIndexes[0]
		if indexes := unusedIndexes(sshByEmail[email], usedSSH); len(indexes) == 1 && result[pi].SSHKeyPath == "" {
			result[pi] = mergeDiscovered(result[pi], sshAccounts[indexes[0]])
			usedSSH[indexes[0]] = true
		}
		if indexes := gpgByEmail[email]; len(indexes) == 1 {
			result[pi] = mergeDiscovered(result[pi], gpgAccounts[indexes[0]])
			usedGPG[indexes[0]] = true
		}
	}

	for email, sshIndexes := range sshByEmail {
		gpgIndexes := gpgByEmail[email]
		if len(sshIndexes) != 1 || len(gpgIndexes) != 1 || usedSSH[sshIndexes[0]] || usedGPG[gpgIndexes[0]] {
			continue
		}
		result = append(result, mergeDiscovered(sshAccounts[sshIndexes[0]], gpgAccounts[gpgIndexes[0]]))
		usedSSH[sshIndexes[0]] = true
		usedGPG[gpgIndexes[0]] = true
	}
	for i, account := range sshAccounts {
		if !usedSSH[i] {
			result = append(result, account)
		}
	}
	for i, account := range gpgAccounts {
		if !usedGPG[i] {
			result = append(result, account)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].IdentityKey + "|" + result[i].Source + "|" + result[i].Alias
		right := result[j].IdentityKey + "|" + result[j].Source + "|" + result[j].Alias
		return left < right
	})
	ensureUniqueAliases(result)
	return result
}

func indexesByUsername(accounts []*DiscoveredAccount) map[string][]int {
	result := make(map[string][]int)
	for i, account := range accounts {
		if account == nil || account.Account == nil {
			continue
		}
		username := strings.ToLower(strings.TrimSpace(account.GetUsername()))
		if username != "" {
			result[username] = append(result[username], i)
		}
	}
	return result
}

func unusedIndexes(indexes []int, used map[int]bool) []int {
	result := make([]int, 0, len(indexes))
	for _, index := range indexes {
		if !used[index] {
			result = append(result, index)
		}
	}
	return result
}

func indexesByEmail(accounts []*DiscoveredAccount) map[string][]int {
	result := make(map[string][]int)
	for i, account := range accounts {
		if account == nil || account.Account == nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(account.Email))
		if email != "" {
			result[email] = append(result[email], i)
		}
	}
	return result
}

func mergeDiscovered(primary, evidence *DiscoveredAccount) *DiscoveredAccount {
	accountCopy := *primary.Account
	merged := &DiscoveredAccount{
		Account:     &accountCopy,
		Source:      primary.Source + "+" + evidence.Source,
		Confidence:  min(10, max(primary.Confidence, evidence.Confidence)+1),
		IdentityKey: primary.IdentityKey,
	}
	if merged.Name == "" || len(evidence.Name) > len(merged.Name) {
		merged.Name = evidence.Name
	}
	if merged.Email == "" {
		merged.Email = evidence.Email
	}
	if merged.SSHKeyPath == "" {
		merged.SSHKeyPath = evidence.SSHKeyPath
	}
	if merged.GPGKeyID == "" {
		merged.GPGKeyID = evidence.GPGKeyID
		merged.GPGKeyFingerprint = evidence.GPGKeyFingerprint
		merged.GPGKeyType = evidence.GPGKeyType
		merged.GPGKeySize = evidence.GPGKeySize
		merged.GPGKeyExpiry = evidence.GPGKeyExpiry
		merged.GPGEnabled = evidence.GPGEnabled
	}
	merged.Description = "Discovered from " + strings.ReplaceAll(merged.Source, "+", " and ")
	return merged
}

func ensureUniqueAliases(accounts []*DiscoveredAccount) {
	used := make(map[string]bool)
	for _, account := range accounts {
		base := sanitizeAlias(account.Alias)
		alias := base
		for suffix := 2; used[strings.ToLower(alias)]; suffix++ {
			suffixText := fmt.Sprintf("-%d", suffix)
			trimmed := base
			if len(trimmed)+len(suffixText) > 64 {
				trimmed = strings.TrimRight(trimmed[:64-len(suffixText)], ".-_")
			}
			alias = trimmed + suffixText
		}
		account.Alias = alias
		used[strings.ToLower(alias)] = true
	}
}

func detectPlatform(email string) string {
	domain := emailDomain(email)
	if strings.Contains(domain, "gitlab") {
		return "gitlab"
	}
	return "github"
}

func emailDomain(email string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
