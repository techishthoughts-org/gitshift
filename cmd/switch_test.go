package cmd

import (
	"context"
	"testing"

	"github.com/techishthoughts/GitPersona/internal/container"
	"github.com/techishthoughts/GitPersona/internal/models"
)

type fakeGitService struct {
	setName  string
	setEmail string
	setSSH   string
}

func (f *fakeGitService) SetUserConfiguration(ctx context.Context, name, email string) error {
	f.setName = name
	f.setEmail = email
	return nil
}

func (f *fakeGitService) SetSSHCommand(ctx context.Context, sshCommand string) error {
	f.setSSH = sshCommand
	return nil
}

func TestUpdateGitConfig_usesGitService(t *testing.T) {
	// Prepare fake service and inject into global container
	fake := &fakeGitService{}
	c := container.GetGlobalSimpleContainer()
	c.SetGitService(fake)

	acct := &models.Account{
		Alias:      "personal",
		Name:       "Alice",
		Email:      "alice@example.com",
		SSHKeyPath: "/home/alice/.ssh/id_rsa",
	}

	// Create a switch command and test the updateGitConfig method
	switchCmd := NewSwitchCommand()
	ctx := context.Background()

	if err := switchCmd.updateGitConfig(ctx, acct); err != nil {
		t.Fatalf("updateGitConfig failed: %v", err)
	}

	if fake.setName != "Alice" || fake.setEmail != "alice@example.com" {
		t.Fatalf("expected name/email set, got %s/%s", fake.setName, fake.setEmail)
	}

	if fake.setSSH == "" {
		t.Fatalf("expected ssh command set, got empty")
	}
}
