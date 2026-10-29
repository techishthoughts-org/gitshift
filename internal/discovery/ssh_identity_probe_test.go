package discovery

import (
	"context"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func TestParseSSHAuthenticatedUsername(t *testing.T) {
	tests := map[string]string{
		"Hi octocat! You've successfully authenticated, but GitHub does not provide shell access.": "octocat",
		"Welcome to GitLab, @gitlab-user!": "gitlab-user",
	}
	for output, want := range tests {
		got, ok := parseSSHAuthenticatedUsername(output)
		if !ok || got != want {
			t.Fatalf("parseSSHAuthenticatedUsername(%q) = %q, %t; want %q", output, got, ok, want)
		}
	}
	if _, ok := parseSSHAuthenticatedUsername("Permission denied (publickey)."); ok {
		t.Fatal("failed authentication was accepted")
	}
}

func TestEnrichSSHAccountsMatchesGenericKeyByAuthenticatedUsername(t *testing.T) {
	previous := probeSSHIdentity
	probeSSHIdentity = func(_ context.Context, host, key string) (string, bool) {
		if host == "github.com" && key == "/keys/id_ed25519" {
			return "work-user", true
		}
		return "", false
	}
	t.Cleanup(func() { probeSSHIdentity = previous })

	provider := &DiscoveredAccount{Account: &models.Account{
		Alias: "work-user", Username: "work-user", GitHubUsername: "work-user", Platform: "github", Domain: "github.com",
	}}
	key := &DiscoveredAccount{Account: &models.Account{
		Alias: "person", Email: "person@example.com", SSHKeyPath: "/keys/id_ed25519", Platform: "github",
	}, Source: "ssh", Confidence: 8}

	enrichSSHAccountsWithProviderIdentity(context.Background(), []*DiscoveredAccount{provider}, []*DiscoveredAccount{key})
	if key.GetUsername() != "work-user" || key.Alias != "work-user" || key.Domain != "github.com" || key.Confidence != 10 {
		t.Fatalf("enriched key = %#v", key)
	}
}
