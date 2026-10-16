package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

type commandRunner interface {
	Run(name string, args []string, env []string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(name string, args []string, env []string) ([]byte, error) {
	return execCommandRunner{}.RunContext(context.Background(), name, args, env)
}

func (execCommandRunner) RunContext(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = sanitizedGHEnvironment(env)
	return cmd.Output()
}

type contextCommandRunner interface {
	RunContext(context.Context, string, []string, []string) ([]byte, error)
}

func runWithContext(ctx context.Context, runner commandRunner, name string, args, env []string) ([]byte, error) {
	if contextual, ok := runner.(contextCommandRunner); ok {
		return contextual.RunContext(ctx, name, args, env)
	}
	return runner.Run(name, args, env)
}

func sanitizedGHEnvironment(explicit []string) []string {
	blocked := map[string]bool{
		"GH_TOKEN":                true,
		"GITHUB_TOKEN":            true,
		"GH_ENTERPRISE_TOKEN":     true,
		"GITHUB_ENTERPRISE_TOKEN": true,
		"GH_HOST":                 true,
	}
	env := make([]string, 0, len(os.Environ())+len(explicit))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !blocked[strings.ToUpper(key)] {
			env = append(env, entry)
		}
	}
	return append(env, explicit...)
}

// GitHubCLIScanner discovers every account known to GitHub CLI without
// changing which account is active on a host.
type GitHubCLIScanner struct {
	runner commandRunner
	output io.Writer
}

func NewGitHubCLIScanner() *GitHubCLIScanner {
	return &GitHubCLIScanner{runner: execCommandRunner{}, output: os.Stdout}
}

type ghAuthStatus struct {
	Hosts map[string][]ghAuthAccount `json:"hosts"`
}

type ghAuthAccount struct {
	Host   string `json:"host"`
	Login  string `json:"login"`
	State  string `json:"state"`
	Active bool   `json:"active"`
}

type ghUserProfile struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *GitHubCLIScanner) ScanAccounts() ([]*DiscoveredAccount, error) {
	return s.ScanAccountsContext(context.Background())
}

// ScanAccountsContext is the cancelable form used by interactive clients.
func (s *GitHubCLIScanner) ScanAccountsContext(ctx context.Context) ([]*DiscoveredAccount, error) {
	if s.output == nil {
		s.output = io.Discard
	}
	output, err := runWithContext(ctx, s.runner, "gh", []string{"auth", "status", "--json", "hosts"}, nil)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if errors.Is(err, exec.ErrNotFound) {
			fmt.Fprintln(s.output, "⚠️  GitHub CLI not installed, skipping GitHub CLI discovery")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query GitHub CLI authentication status")
	}

	var status ghAuthStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub CLI authentication status: %w", err)
	}

	type identity struct {
		host   string
		login  string
		state  string
		active bool
	}
	var identities []identity
	for host, accounts := range status.Hosts {
		for _, account := range accounts {
			if !strings.EqualFold(strings.TrimSpace(account.State), "success") {
				continue
			}
			resolvedHost := strings.ToLower(strings.TrimSpace(account.Host))
			if resolvedHost == "" {
				resolvedHost = strings.ToLower(strings.TrimSpace(host))
			}
			login := strings.TrimSpace(account.Login)
			if resolvedHost == "" || login == "" {
				continue
			}
			identities = append(identities, identity{resolvedHost, login, account.State, account.Active})
		}
	}
	sort.Slice(identities, func(i, j int) bool {
		left := canonicalProviderIdentity("github", identities[i].host, identities[i].login)
		right := canonicalProviderIdentity("github", identities[j].host, identities[j].login)
		return left < right
	})

	discovered := make([]*DiscoveredAccount, 0, len(identities))
	for _, identity := range identities {
		profile := ghUserProfile{Login: identity.login}
		confidence := 8
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		enriched, ok := s.profileContext(ctx, identity.host, identity.login)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ok {
			profile.Name = enriched.Name
			profile.Email = enriched.Email
			confidence = 10
		}

		key := canonicalProviderIdentity("github", identity.host, identity.login)
		account := &models.Account{
			Alias:          sanitizeAlias(identity.login),
			Name:           profile.Name,
			Email:          profile.Email,
			Username:       identity.login,
			GitHubUsername: identity.login,
			Platform:       "github",
			Domain:         identity.host,
			Description:    "Discovered from GitHub CLI authentication",
			AccountMetadata: map[string]string{
				"discovery_identity": key,
			},
		}
		discovered = append(discovered, &DiscoveredAccount{
			Account:     account,
			Source:      "github-cli",
			Confidence:  confidence,
			IdentityKey: key,
		})
	}

	fmt.Fprintf(s.output, "✅ Found %d GitHub CLI account(s)\n", len(discovered))
	return discovered, nil
}

// profile retrieves a token for one explicit identity and passes it only in
// the environment of a single gh api process. It never switches the active
// GitHub CLI account and never returns or logs the token.
func (s *GitHubCLIScanner) profile(host, login string) (ghUserProfile, bool) {
	return s.profileContext(context.Background(), host, login)
}

func (s *GitHubCLIScanner) profileContext(ctx context.Context, host, login string) (ghUserProfile, bool) {
	tokenBytes, err := runWithContext(ctx, s.runner, "gh", []string{"auth", "token", "--hostname", host, "--user", login}, nil)
	if err != nil {
		return ghUserProfile{}, false
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return ghUserProfile{}, false
	}
	// gh uses GH_TOKEN for github.com/GHE Cloud and GH_ENTERPRISE_TOKEN for
	// other enterprise hosts. Set both only for this child so either host type
	// uses the credential selected by --user above.
	output, err := runWithContext(ctx, s.runner, "gh", []string{"api", "--hostname", host, "user"}, []string{
		"GH_TOKEN=" + token,
		"GH_ENTERPRISE_TOKEN=" + token,
		"GH_HOST=" + host,
	})
	if err != nil {
		return ghUserProfile{}, false
	}
	var profile ghUserProfile
	if err := json.Unmarshal(output, &profile); err != nil {
		return ghUserProfile{}, false
	}
	return profile, strings.EqualFold(strings.TrimSpace(profile.Login), login)
}

func canonicalProviderIdentity(provider, hostname, username string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "|" +
		strings.ToLower(strings.TrimSpace(hostname)) + "|" +
		strings.ToLower(strings.TrimSpace(username))
}

func sanitizeAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	alias := strings.Trim(b.String(), ".-_")
	if alias == "" {
		alias = "account"
	}
	if len(alias) > 64 {
		alias = alias[:64]
		alias = strings.TrimRight(alias, ".-_")
	}
	return alias
}
