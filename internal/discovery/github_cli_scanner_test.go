package discovery

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type recordedCall struct {
	name string
	args []string
	env  []string
}

type fakeGHRunner struct {
	calls []recordedCall
}

func (f *fakeGHRunner) Run(name string, args []string, env []string) ([]byte, error) {
	f.calls = append(f.calls, recordedCall{name, append([]string(nil), args...), append([]string(nil), env...)})
	joined := strings.Join(args, " ")
	switch {
	case joined == "auth status --json hosts":
		return []byte(`{"hosts":{"github.com":[{"host":"github.com","login":"work","state":"success","active":true},{"host":"github.com","login":"personal","state":"success","active":false}],"git.example.com":[{"host":"git.example.com","login":"work","state":"success","active":true}]}}`), nil
	case strings.HasPrefix(joined, "auth token "):
		return []byte("secret-for-" + args[len(args)-1]), nil
	case joined == "api --hostname github.com user":
		for _, item := range env {
			if item == "GH_TOKEN=secret-for-personal" {
				return []byte(`{"login":"personal","name":"Personal User","email":"same@example.com"}`), nil
			}
			if item == "GH_TOKEN=secret-for-work" {
				return []byte(`{"login":"work","name":"Work User","email":"work@example.com"}`), nil
			}
		}
	case joined == "api --hostname git.example.com user":
		return []byte(`{"login":"work","name":"Enterprise User","email":"enterprise@example.com"}`), nil
	}
	return nil, fmt.Errorf("unexpected call: %s", joined)
}

func TestGitHubCLIScannerDiscoversAllHostAccountsWithoutSwitching(t *testing.T) {
	runner := &fakeGHRunner{}
	scanner := &GitHubCLIScanner{runner: runner}
	accounts, err := scanner.ScanAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 3 {
		t.Fatalf("accounts = %d, want 3", len(accounts))
	}
	wantKeys := []string{
		"github|git.example.com|work",
		"github|github.com|personal",
		"github|github.com|work",
	}
	var gotKeys []string
	for _, account := range accounts {
		gotKeys = append(gotKeys, account.IdentityKey)
		if account.AccountMetadata["discovery_identity"] != account.IdentityKey {
			t.Fatalf("identity metadata = %#v", account.AccountMetadata)
		}
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		t.Fatalf("identity keys = %v, want %v", gotKeys, wantKeys)
	}

	for _, call := range runner.calls {
		joined := strings.Join(call.args, " ")
		if strings.Contains(joined, "switch") || strings.Contains(joined, "secret-") {
			t.Fatalf("unsafe gh arguments: %q", joined)
		}
		if strings.HasPrefix(joined, "auth token") && len(call.env) != 0 {
			t.Fatalf("token lookup inherited custom credentials: %v", call.env)
		}
	}
}

type failingRunner struct{}

func (failingRunner) Run(string, []string, []string) ([]byte, error) {
	return nil, errors.New("boom")
}

func TestGitHubCLIScannerStatusFailureDoesNotExposeCommandDetails(t *testing.T) {
	_, err := (&GitHubCLIScanner{runner: failingRunner{}}).ScanAccounts()
	if err == nil || err.Error() != "failed to query GitHub CLI authentication status" {
		t.Fatalf("error = %v", err)
	}
}

func TestCanonicalProviderIdentityNormalizesCase(t *testing.T) {
	if got := canonicalProviderIdentity(" GitHub ", "GitHub.COM", "SomeUser"); got != "github|github.com|someuser" {
		t.Fatalf("canonical identity = %q", got)
	}
}

func TestSanitizedGHEnvironmentRemovesAmbientAuthentication(t *testing.T) {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN", "GH_HOST"} {
		t.Setenv(key, "ambient-secret")
	}
	t.Setenv("GITSHIFT_DISCOVERY_TEST", "preserved")
	env := sanitizedGHEnvironment([]string{"GH_TOKEN=explicit-secret", "GH_HOST=explicit.example"})
	values := make(map[string]string)
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		values[strings.ToUpper(key)] = value
	}
	if values["GH_TOKEN"] != "explicit-secret" || values["GH_HOST"] != "explicit.example" {
		t.Fatalf("explicit environment missing: %#v", values)
	}
	for _, key := range []string{"GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		if _, exists := values[key]; exists {
			t.Fatalf("ambient %s was retained", key)
		}
	}
	if values["GITSHIFT_DISCOVERY_TEST"] != "preserved" {
		t.Fatal("unrelated environment was not retained")
	}
}

type functionRunner func(name string, args []string, env []string) ([]byte, error)

func (f functionRunner) Run(name string, args []string, env []string) ([]byte, error) {
	return f(name, args, env)
}

func TestGitHubCLIScannerSkipsFailedAuthAndRejectsMismatchedProfile(t *testing.T) {
	var tokenUsers []string
	runner := functionRunner(func(_ string, args []string, _ []string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case joined == "auth status --json hosts":
			return []byte(`{"hosts":{"github.com":[{"login":"CaseUser","state":"success"},{"login":"broken","state":"failed"}]}}`), nil
		case strings.HasPrefix(joined, "auth token"):
			tokenUsers = append(tokenUsers, args[len(args)-1])
			return []byte("selected-token"), nil
		case joined == "api --hostname github.com user":
			return []byte(`{"login":"different-user","name":"Wrong Profile","email":"wrong@example.com"}`), nil
		default:
			return nil, fmt.Errorf("unexpected call: %s", joined)
		}
	})

	accounts, err := (&GitHubCLIScanner{runner: runner}).ScanAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 {
		t.Fatalf("accounts = %d, want only successful auth entry", len(accounts))
	}
	account := accounts[0]
	if account.Username != "CaseUser" || account.GitHubUsername != "CaseUser" || account.IdentityKey != "github|github.com|caseuser" {
		t.Fatalf("auth identity was rewritten: %#v", account)
	}
	if account.Name != "" || account.Email != "" || account.Confidence != 8 {
		t.Fatalf("mismatched profile was accepted: %#v", account)
	}
	if !reflect.DeepEqual(tokenUsers, []string{"CaseUser"}) {
		t.Fatalf("token requested for failed account: %v", tokenUsers)
	}
}

func TestGitHubCLIScannerAcceptsCaseInsensitiveMatchingProfileWithoutRewritingLogin(t *testing.T) {
	runner := functionRunner(func(_ string, args []string, _ []string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "auth status --json hosts":
			return []byte(`{"hosts":{"github.com":[{"login":"CaseUser","state":"success"}]}}`), nil
		case "auth token --hostname github.com --user CaseUser":
			return []byte("selected-token"), nil
		case "api --hostname github.com user":
			return []byte(`{"login":"caseuser","name":"Case User","email":"case@example.com"}`), nil
		default:
			return nil, errors.New("unexpected command")
		}
	})
	accounts, err := (&GitHubCLIScanner{runner: runner}).ScanAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].Username != "CaseUser" || accounts[0].Name != "Case User" || accounts[0].Confidence != 10 {
		t.Fatalf("account = %#v", accounts)
	}
}
