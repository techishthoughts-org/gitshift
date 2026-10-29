package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func TestAddKeyToGitHubUsesNamedAccountLogin(t *testing.T) {
	logFile := fakeGH(t, "success")
	account := models.NewAccount("work", "Work", "w@example.com", "")
	account.Platform = "github"
	account.Domain = "github.example.com"
	account.SetUsername("alice")

	if err := addKeyToGitHub("/tmp/id_work.pub", "work", account); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(lines) != 2 {
		t.Fatalf("gh calls = %q", lines)
	}
	if !strings.HasPrefix(lines[0], "auth token --hostname github.example.com --user alice|") {
		t.Fatalf("token lookup = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "ssh-key add /tmp/id_work.pub --title gitshift-work-") ||
		!strings.HasSuffix(lines[1], "|GH_TOKEN=token-for-alice|GH_HOST=github.example.com") {
		t.Fatalf("upload did not use alice's token on her host: %q", lines[1])
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "auth switch") {
			t.Fatal("upload must not change the active gh account")
		}
	}
}

func TestAddKeyToGitHubRejectsUnusableAccounts(t *testing.T) {
	fakeGH(t, "success")
	gitlab := models.NewAccount("lab", "Lab", "l@example.com", "")
	gitlab.Platform = "gitlab"
	gitlab.SetUsername("alice")
	notLoggedIn := models.NewAccount("other", "Other", "o@example.com", "")
	notLoggedIn.SetUsername("bob")

	tests := []struct {
		name    string
		account *models.Account
		wantErr string
	}{
		{name: "unknown alias", account: nil, wantErr: "existing GitHub account"},
		{name: "gitlab account", account: gitlab, wantErr: "existing GitHub account"},
		{name: "gh not logged in as user", account: notLoggedIn, wantErr: "not logged in as @bob"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := addKeyToGitHub("/tmp/key.pub", "x", tt.account)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
