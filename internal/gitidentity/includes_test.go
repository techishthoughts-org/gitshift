package gitidentity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

type fixture struct {
	root, global, includeDir string
	cfg                      *models.Config
}

// newFixture isolates the global Git config and builds a work account that
// signs with SSH and a personal account that does not, with rules for
// "work dir" (path with a space), "personal" and the nested "personal/client".
func newFixture(t *testing.T) *fixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{root: root, global: filepath.Join(root, "global.gitconfig"), includeDir: filepath.Join(root, "gitshift", "git")}
	t.Setenv("GIT_CONFIG_GLOBAL", f.global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", root)

	work := models.NewAccount("work", "Work Name", "work@example.com", filepath.Join(root, "id_work"))
	work.SigningFormat = models.SigningFormatSSH
	personal := models.NewAccount("personal", "Personal Name", "me@example.com", filepath.Join(root, "id_personal"))
	f.cfg = models.NewConfig()
	f.cfg.Accounts = map[string]*models.Account{"work": work, "personal": personal}
	for dir, alias := range map[string]string{"work dir": "work", "personal": "personal", "personal/client": "work"} {
		path := filepath.Join(root, dir)
		f.mkRepo(t, path)
		f.cfg.AutoSwitch.Rules = append(f.cfg.AutoSwitch.Rules, models.AutoSwitchRule{Path: path, Account: alias})
	}
	f.git(t, "", "config", "--global", "user.email", "global@example.com")
	f.git(t, "", "config", "--global", "commit.gpgsign", "false")
	return f
}

func (f *fixture) mkRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	f.git(t, dir, "init", "-q")
}

func (f *fixture) git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func (f *fixture) value(t *testing.T, dir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func TestSyncGivesEachDirectoryItsIdentity(t *testing.T) {
	f := newFixture(t)
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		dir, email, gpgsign, sshKey string
	}{
		{dir: "work dir", email: "work@example.com", gpgsign: "true", sshKey: "id_work"},
		{dir: "personal", email: "me@example.com", gpgsign: "false", sshKey: "id_personal"},
		{dir: "personal/client", email: "work@example.com", gpgsign: "true", sshKey: "id_work"},
	}
	for _, tt := range tests {
		dir := filepath.Join(f.root, tt.dir)
		if got := f.value(t, dir, "user.email"); got != tt.email {
			t.Errorf("%s: user.email = %q, want %q", tt.dir, got, tt.email)
		}
		if got := f.value(t, dir, "commit.gpgsign"); got != tt.gpgsign {
			t.Errorf("%s: commit.gpgsign = %q, want %q", tt.dir, got, tt.gpgsign)
		}
		if got := f.value(t, dir, "core.sshCommand"); !strings.Contains(got, tt.sshKey) {
			t.Errorf("%s: core.sshCommand = %q, want key %s", tt.dir, got, tt.sshKey)
		}
	}
	if got := f.value(t, filepath.Join(f.root, "work dir"), "gpg.format"); got != "ssh" {
		t.Errorf("work gpg.format = %q, want ssh", got)
	}
	outside := filepath.Join(f.root, "elsewhere")
	f.mkRepo(t, outside)
	if got := f.value(t, outside, "user.email"); got != "global@example.com" {
		t.Errorf("outside rules: user.email = %q, want the global identity", got)
	}
	info, err := os.Stat(filepath.Join(f.includeDir, "work.gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("include file mode = %o, want 600", info.Mode().Perm())
	}
}

func TestSyncIsIdempotentAndStaysLast(t *testing.T) {
	f := newFixture(t)
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(f.global)
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(f.global)
	if string(first) != string(second) {
		t.Fatalf("second sync changed the global config:\n%s\n---\n%s", first, second)
	}

	// A section created after the includes would override them.
	f.git(t, "", "config", "--global", "gpg.format", "openpgp")
	if got := f.value(t, filepath.Join(f.root, "work dir"), "gpg.format"); got != "openpgp" {
		t.Fatalf("precondition: later global section should win, got %q", got)
	}
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}
	if got := f.value(t, filepath.Join(f.root, "work dir"), "gpg.format"); got != "ssh" {
		t.Fatalf("after resync gpg.format = %q, want the include's ssh", got)
	}
	if strings.Count(readFile(t, f.global), "[includeIf") != 3 {
		t.Fatalf("includes duplicated:\n%s", readFile(t, f.global))
	}
}

func TestSyncFollowsRuleChangesAndKeepsUserIncludes(t *testing.T) {
	f := newFixture(t)
	userInclude := filepath.Join(f.root, "mine.gitconfig")
	f.git(t, "", "config", "--global", "includeIf.gitdir:"+f.root+"/other/.path", userInclude)
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}

	// Drop every rule that uses "work".
	f.cfg.AutoSwitch.Rules = []models.AutoSwitchRule{{Path: filepath.Join(f.root, "personal"), Account: "personal"}}
	if err := Sync(f.cfg, f.includeDir); err != nil {
		t.Fatal(err)
	}
	installed, err := Installed(f.includeDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 || installed[0].Account != "personal" {
		t.Fatalf("installed = %+v", installed)
	}
	if _, err := os.Stat(filepath.Join(f.includeDir, "work.gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("unused include file kept: %v", err)
	}
	if got := f.value(t, filepath.Join(f.root, "work dir"), "user.email"); got != "global@example.com" {
		t.Fatalf("removed rule still applies: %q", got)
	}

	if err := Remove(f.includeDir); err != nil {
		t.Fatal(err)
	}
	global := readFile(t, f.global)
	if strings.Contains(global, f.includeDir) || !strings.Contains(global, userInclude) {
		t.Fatalf("Remove must drop only GitShift's includes:\n%s", global)
	}
	if files, _ := filepath.Glob(filepath.Join(f.includeDir, "*.gitconfig")); len(files) != 0 {
		t.Fatalf("include files left: %v", files)
	}
}

func TestSigningSettings(t *testing.T) {
	tests := []struct {
		name      string
		account   models.Account
		want      []Setting
		wantUnset []string
	}{
		{
			name:    "ssh",
			account: models.Account{SigningFormat: models.SigningFormatSSH, SSHKeyPath: "/k"},
			want:    []Setting{{"gpg.format", "ssh"}, {"user.signingkey", "/k"}, {"commit.gpgsign", "true"}, {"tag.gpgsign", "true"}},
		},
		{
			name:    "gpg disabled keeps key",
			account: models.Account{GPGKeyID: "K"},
			want:    []Setting{{"gpg.format", "openpgp"}, {"user.signingkey", "K"}, {"commit.gpgsign", "false"}, {"tag.gpgsign", "false"}},
		},
		{
			name:      "none",
			account:   models.Account{},
			want:      []Setting{{"commit.gpgsign", "false"}, {"tag.gpgsign", "false"}},
			wantUnset: []string{"user.signingkey", "gpg.format"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, unset, err := SigningSettings(&tt.account)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(flatten(got), ",") != strings.Join(flatten(tt.want), ",") || strings.Join(unset, ",") != strings.Join(tt.wantUnset, ",") {
				t.Fatalf("got %v unset %v", got, unset)
			}
		})
	}
}

func flatten(settings []Setting) []string {
	out := make([]string, len(settings))
	for i, s := range settings {
		out[i] = s.Key + "=" + s.Value
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
