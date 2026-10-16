package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/techishthoughts/gitshift/internal/models"
)

func TestPrintAccountsJSONEscapesUserContent(t *testing.T) {
	created := time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC)
	accounts := []*models.Account{{
		Alias:          "work\"account",
		Name:           "Line one\nLine two",
		Email:          "user\\name@example.com",
		Platform:       "github",
		Domain:         "github.com",
		GitHubUsername: "octo\"cat",
		Description:    "tabs\tand control \u0001",
		CreatedAt:      created,
	}}

	var output bytes.Buffer
	if err := printAccountsJSON(&output, accounts); err != nil {
		t.Fatalf("printAccountsJSON() error = %v", err)
	}

	var decoded map[string][]accountJSON
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, output.String())
	}

	got := decoded["github"][0]
	if got.Alias != accounts[0].Alias || got.Name != accounts[0].Name || got.Description != accounts[0].Description {
		t.Fatalf("round trip mismatch: got %#v", got)
	}
	if !strings.Contains(output.String(), `work\"account`) {
		t.Fatalf("quoted alias was not JSON-escaped: %s", output.String())
	}
	if strings.Contains(output.String(), "Line one\nLine two") {
		t.Fatalf("literal newline leaked into JSON string: %s", output.String())
	}
}

func TestPrintAccountsJSONSortsAccountsByAlias(t *testing.T) {
	accounts := []*models.Account{
		{Alias: "zeta", Platform: "gitlab"},
		{Alias: "alpha", Platform: "gitlab"},
	}

	var output bytes.Buffer
	if err := printAccountsJSON(&output, accounts); err != nil {
		t.Fatalf("printAccountsJSON() error = %v", err)
	}

	var decoded map[string][]accountJSON
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got := decoded["gitlab"]; len(got) != 2 || got[0].Alias != "alpha" || got[1].Alias != "zeta" {
		t.Fatalf("accounts not sorted by alias: %#v", got)
	}
}
