package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
)

func aliasTestResult(alias, identity string) *discovery.DiscoveredAccount {
	return &discovery.DiscoveredAccount{
		Account: &models.Account{
			Alias:          alias,
			Name:           "Test User",
			Email:          alias + "@example.com",
			Username:       alias,
			GitHubUsername: alias,
			Platform:       "github",
			Domain:         "github.com",
		},
		IdentityKey: identity,
		Confidence:  10,
	}
}

func aliasPlanCommand(input string) (*cobra.Command, *bytes.Buffer) {
	command := &cobra.Command{}
	command.SetIn(strings.NewReader(input))
	output := &bytes.Buffer{}
	command.SetOut(output)
	return command, output
}

func TestPlanDiscoveryAliasesUsesIdentityBeforeSuggestedAlias(t *testing.T) {
	discovered := []*discovery.DiscoveredAccount{
		aliasTestResult("other", "shared"),
		aliasTestResult("shared", "github|github.com|bob"),
	}
	command, _ := aliasPlanCommand("")

	got, err := planDiscoveryAliases(command, discovered, []string{"shared=personal"}, false, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "personal" || got[1] != "shared" {
		t.Fatalf("aliases = %#v", got)
	}
	if discovered[0].Alias != "other" {
		t.Fatal("planner mutated a discovery result")
	}
}

func TestPlanDiscoveryAliasesRejectsInvalidSelectionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name       string
		discovered []*discovery.DiscoveredAccount
		specs      []string
		want       string
	}{
		{"malformed", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"id-a"}, "expected SELECTOR=ALIAS"},
		{"empty selector", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"=mine"}, "expected SELECTOR=ALIAS"},
		{"empty alias", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"id-a="}, "expected SELECTOR=ALIAS"},
		{"unknown", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"missing=mine"}, "unknown discovery selector"},
		{"ambiguous", []*discovery.DiscoveredAccount{aliasTestResult("same", "id-a"), aliasTestResult("same", "id-b")}, []string{"same=mine"}, "ambiguous"},
		{"ambiguous identity", []*discovery.DiscoveredAccount{aliasTestResult("first", "id-a"), aliasTestResult("second", "id-a")}, []string{"id-a=mine"}, "identity selector"},
		{"repeated selector", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"id-a=one", "id-a=two"}, "repeated"},
		{"duplicate target", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"id-a=one", "alice=two"}, "selected more than once"},
		{"invalid alias", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}, []string{"id-a=has space"}, "invalid alias"},
		{"case collision", []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a"), aliasTestResult("bob", "id-b")}, []string{"id-a=Team", "id-b=team"}, "case-insensitively"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, _ := aliasPlanCommand("")
			before := make([]string, len(test.discovered))
			for i := range test.discovered {
				before[i] = test.discovered[i].Alias
			}
			_, err := planDiscoveryAliases(command, test.discovered, test.specs, false, true, nil, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
			for i := range test.discovered {
				if test.discovered[i].Alias != before[i] {
					t.Fatal("planner mutated discovery results on error")
				}
			}
		})
	}
}

func TestPlanDiscoveryAliasesPromptsOnlyUnselectedAndAcceptsEmpty(t *testing.T) {
	discovered := []*discovery.DiscoveredAccount{
		aliasTestResult("alice", "id-a"),
		aliasTestResult("bob", "id-b"),
		aliasTestResult("carol", "id-c"),
	}
	command, output := aliasPlanCommand("\nchosen\n")
	got, err := planDiscoveryAliases(command, discovered, []string{"id-a=explicit"}, true, true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "id-a") || !strings.Contains(output.String(), "id-b") || !strings.Contains(output.String(), "id-c") {
		t.Fatalf("prompts = %q", output.String())
	}
	if strings.Join(got, ",") != "explicit,bob,chosen" {
		t.Fatalf("aliases = %#v", got)
	}
}

func TestPlanDiscoveryAliasesPromptReadFailureAborts(t *testing.T) {
	discovered := []*discovery.DiscoveredAccount{aliasTestResult("alice", "id-a")}
	command, _ := aliasPlanCommand("")
	_, err := planDiscoveryAliases(command, discovered, nil, true, true, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlanDiscoveryAliasesChecksExistingAliasesUnlessOverwrite(t *testing.T) {
	discovered := []*discovery.DiscoveredAccount{aliasTestResult("new", "id-new")}
	existing := []*models.Account{{Alias: "WORK"}}
	pending := []*models.PendingAccount{{Alias: "later"}}
	command, _ := aliasPlanCommand("")
	if _, err := planDiscoveryAliases(command, discovered, []string{"id-new=work"}, false, false, existing, pending); err == nil {
		t.Fatal("expected active-account collision")
	}
	if _, err := planDiscoveryAliases(command, discovered, []string{"id-new=LATER"}, false, false, existing, pending); err == nil {
		t.Fatal("expected pending-account collision")
	}
	if got, err := planDiscoveryAliases(command, discovered, []string{"id-new=work"}, false, true, existing, pending); err != nil || got[0] != "work" {
		t.Fatalf("overwrite plan = %#v, %v", got, err)
	}
}

func TestPlanDiscoveryAliasesRejectsExistingActiveCaseCollision(t *testing.T) {
	discovered := []*discovery.DiscoveredAccount{aliasTestResult("new", "id-new")}
	existing := []*models.Account{{Alias: "work"}, {Alias: "WORK"}}
	command, _ := aliasPlanCommand("")
	_, err := planDiscoveryAliases(command, discovered, nil, false, false, existing, nil)
	if err == nil || !strings.Contains(err.Error(), "existing active aliases") {
		t.Fatalf("error = %v", err)
	}
}
