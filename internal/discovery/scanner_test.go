package discovery

import (
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func discovered(alias, email, source string) *DiscoveredAccount {
	return &DiscoveredAccount{
		Account:    &models.Account{Alias: alias, Email: email, Name: alias, Platform: "github"},
		Source:     source,
		Confidence: 8,
	}
}

func TestMergeAccountsJoinsOnlyUnambiguousEvidence(t *testing.T) {
	d := NewAccountDiscovery()
	provider := discovered("person", "one@example.com", "github-cli")
	provider.IdentityKey = "github|github.com|person"
	ssh := discovered("ssh", "one@example.com", "ssh")
	ssh.SSHKeyPath = "/keys/one"
	gpg := discovered("gpg", "one@example.com", "gpg")
	gpg.GPGKeyID = "ABC"

	got := d.mergeAccounts([]*DiscoveredAccount{provider}, []*DiscoveredAccount{ssh}, []*DiscoveredAccount{gpg})
	if len(got) != 1 || got[0].SSHKeyPath != "/keys/one" || got[0].GPGKeyID != "ABC" {
		t.Fatalf("unambiguous result = %#v", got)
	}
	if got[0].Source != "github-cli+ssh+gpg" {
		t.Fatalf("source = %q", got[0].Source)
	}
}

func TestMergeAccountsKeepsSharedEmailProviderIdentitiesSeparate(t *testing.T) {
	d := NewAccountDiscovery()
	first := discovered("same", "shared@example.com", "github-cli")
	first.IdentityKey = "github|github.com|first"
	second := discovered("same", "shared@example.com", "github-cli")
	second.IdentityKey = "github|github.com|second"
	ssh := discovered("same", "shared@example.com", "ssh")
	ssh.SSHKeyPath = "/keys/shared"

	got := d.mergeAccounts([]*DiscoveredAccount{first, second}, []*DiscoveredAccount{ssh}, nil)
	if len(got) != 3 {
		t.Fatalf("ambiguous result count = %d, want 3", len(got))
	}
	aliases := map[string]bool{}
	for _, account := range got {
		if aliases[account.Alias] {
			t.Fatalf("duplicate alias %q", account.Alias)
		}
		aliases[account.Alias] = true
		if account.Source == "github-cli" && account.SSHKeyPath != "" {
			t.Fatalf("ambiguous SSH key attached to %s", account.IdentityKey)
		}
	}
}

func TestCorporateEmailDoesNotImplyGitLab(t *testing.T) {
	if got := detectPlatform("person@company.example"); got != "github" {
		t.Fatalf("platform = %q, want github default", got)
	}
	if got := detectPlatform("person@gitlab.example"); got != "gitlab" {
		t.Fatalf("explicit GitLab domain = %q", got)
	}
}

func TestMergeAccountsUsesUniqueUsernameBeforeEmail(t *testing.T) {
	provider := discovered("alice", "", "github-cli")
	provider.Username = "Alice"
	provider.GitHubUsername = "Alice"
	provider.Domain = "github.com"
	provider.IdentityKey = "github|github.com|alice"
	ssh := discovered("alice-key", "unrelated@example.com", "ssh")
	ssh.GitHubUsername = "alice"
	ssh.SSHKeyPath = "/keys/alice"

	got := NewAccountDiscovery().mergeAccounts([]*DiscoveredAccount{provider}, []*DiscoveredAccount{ssh}, nil)
	if len(got) != 1 || got[0].SSHKeyPath != "/keys/alice" || got[0].IdentityKey != provider.IdentityKey {
		t.Fatalf("username merge = %#v", got)
	}
}

func TestMergeAccountsDoesNotUseUsernameAcrossMultipleHosts(t *testing.T) {
	first := discovered("alice", "", "github-cli")
	first.Username, first.GitHubUsername, first.Domain = "alice", "alice", "github.com"
	first.IdentityKey = "github|github.com|alice"
	second := discovered("alice", "", "github-cli")
	second.Username, second.GitHubUsername, second.Domain = "alice", "alice", "git.example.com"
	second.IdentityKey = "github|git.example.com|alice"
	ssh := discovered("alice-key", "", "ssh")
	ssh.GitHubUsername = "alice"
	ssh.SSHKeyPath = "/keys/alice"

	got := NewAccountDiscovery().mergeAccounts([]*DiscoveredAccount{first, second}, []*DiscoveredAccount{ssh}, nil)
	if len(got) != 3 {
		t.Fatalf("ambiguous-host result count = %d", len(got))
	}
	providerCount, sshOnlyCount := 0, 0
	for _, account := range got {
		if account.Source == "github-cli" {
			providerCount++
			if account.SSHKeyPath != "" {
				t.Fatalf("SSH evidence attached across host ambiguity: %#v", account)
			}
		}
		if account.Source == "ssh" && account.SSHKeyPath == "/keys/alice" {
			sshOnlyCount++
		}
	}
	if providerCount != 2 || sshOnlyCount != 1 {
		t.Fatalf("provider=%d ssh-only=%d", providerCount, sshOnlyCount)
	}
}
