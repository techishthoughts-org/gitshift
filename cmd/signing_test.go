package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func isolateGitConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "global.gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", dir)
	return dir
}

func globalGitValue(t *testing.T, key string) string {
	t.Helper()
	output, _ := exec.Command("git", "config", "--global", "--get", key).Output()
	return strings.TrimSpace(string(output))
}

func TestApplyCommitSigning(t *testing.T) {
	tests := []struct {
		name    string
		account func(home string) *models.Account
		want    map[string]string
	}{
		{
			name: "ssh signing uses the account key",
			account: func(home string) *models.Account {
				a := models.NewAccount("work", "W", "w@example.com", filepath.Join(home, "id_work"))
				a.SigningFormat = models.SigningFormatSSH
				return a
			},
			want: map[string]string{"gpg.format": "ssh", "user.signingkey": "{home}/id_work", "commit.gpgsign": "true", "tag.gpgsign": "true"},
		},
		{
			name: "ssh signing with a separate key",
			account: func(home string) *models.Account {
				a := models.NewAccount("work", "W", "w@example.com", filepath.Join(home, "id_work"))
				a.SigningFormat, a.SigningKey = models.SigningFormatSSH, filepath.Join(home, "id_signing.pub")
				return a
			},
			want: map[string]string{"gpg.format": "ssh", "user.signingkey": "{home}/id_signing.pub", "commit.gpgsign": "true"},
		},
		{
			name: "legacy gpg_enabled account signs with gpg",
			account: func(string) *models.Account {
				a := models.NewAccount("old", "O", "o@example.com", "")
				a.GPGKeyID, a.GPGEnabled = "ABCDEF12", true
				return a
			},
			want: map[string]string{"gpg.format": "openpgp", "user.signingkey": "ABCDEF12", "commit.gpgsign": "true", "tag.gpgsign": "true"},
		},
		{
			name: "gpg key with signing off stays selected",
			account: func(string) *models.Account {
				a := models.NewAccount("old", "O", "o@example.com", "")
				a.GPGKeyID = "ABCDEF12"
				return a
			},
			want: map[string]string{"gpg.format": "openpgp", "user.signingkey": "ABCDEF12", "commit.gpgsign": "false", "tag.gpgsign": "false"},
		},
		{
			name:    "no signing clears a previous ssh setup",
			account: func(string) *models.Account { return models.NewAccount("me", "M", "m@example.com", "") },
			want:    map[string]string{"gpg.format": "", "user.signingkey": "", "commit.gpgsign": "false", "tag.gpgsign": "false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := isolateGitConfig(t)
			// Start from another account's SSH signing setup.
			for key, value := range map[string]string{"gpg.format": "ssh", "user.signingkey": "/old/key", "commit.gpgsign": "true"} {
				if err := exec.Command("git", "config", "--global", key, value).Run(); err != nil {
					t.Fatal(err)
				}
			}
			if err := applyCommitSigning(tt.account(home)); err != nil {
				t.Fatal(err)
			}
			for key, want := range tt.want {
				want = strings.ReplaceAll(want, "{home}", home)
				if got := globalGitValue(t, key); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestSSHSigningProducesVerifiableCommit(t *testing.T) {
	home := isolateGitConfig(t)
	if err := checkGitSupportsSSHSigning(); err != nil {
		t.Skip(err)
	}
	key := filepath.Join(home, "id_work")
	if output, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "work", "-f", key).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen unavailable: %v: %s", err, output)
	}
	t.Setenv("SSH_AUTH_SOCK", "")
	account := models.NewAccount("work", "Work", "w@example.com", key)
	account.SigningFormat = models.SigningFormatSSH
	if err := checkCommitSigning(account); err != nil {
		t.Fatalf("checkCommitSigning() = %v", err)
	}
	if err := applyCommitSigning(account); err != nil {
		t.Fatal(err)
	}

	repo := filepath.Join(home, "repo")
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
		return string(output)
	}
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	run("init", "-q")
	run("-c", "user.name=Work", "-c", "user.email=w@example.com", "commit", "-q", "--allow-empty", "-m", "signed")
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	allowed := filepath.Join(home, "allowed_signers")
	if err := os.WriteFile(allowed, []byte("w@example.com "+string(pub)), 0o600); err != nil {
		t.Fatal(err)
	}
	if output := run("-c", "gpg.ssh.allowedSignersFile="+allowed, "verify-commit", "-v", "HEAD"); !strings.Contains(output, "Good \"git\" signature") {
		t.Fatalf("commit is not verifiably signed:\n%s", output)
	}
}

func TestCheckCommitSigningRejectsUnusableKeys(t *testing.T) {
	home := isolateGitConfig(t)
	open := filepath.Join(home, "id_open")
	if err := os.WriteFile(open, []byte("key"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, key, wantErr string
	}{
		{name: "missing private key", key: filepath.Join(home, "missing"), wantErr: "not found"},
		{name: "missing public key", key: filepath.Join(home, "missing.pub"), wantErr: "not found"},
		{name: "readable private key", key: open, wantErr: "permissions 644"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := models.NewAccount("work", "W", "w@example.com", "")
			account.SigningFormat, account.SigningKey = models.SigningFormatSSH, tt.key
			if err := checkCommitSigning(account); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
	if err := checkCommitSigning(models.NewAccount("me", "M", "m@example.com", "")); err != nil {
		t.Fatalf("an account without signing needs no checks: %v", err)
	}
}

func TestGitVersionAtLeast(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"git version 2.34.0", true},
		{"git version 2.53.0", true},
		{"git version 3.0.1", true},
		{"git version 2.33.8", false},
		{"git version 2.39.5 (Apple Git-154)", true},
		{"git version 1.9.0", false},
		{"garbage", false},
	}
	for _, tt := range tests {
		if got := gitVersionAtLeast(tt.output, 2, 34); got != tt.want {
			t.Errorf("gitVersionAtLeast(%q) = %t, want %t", tt.output, got, tt.want)
		}
	}
}
