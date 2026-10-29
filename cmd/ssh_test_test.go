package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHKeyPermissionCheckIsReadOnly(t *testing.T) {
	key := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(key, []byte("key"), 0644); err != nil {
		t.Fatal(err)
	}
	tester := &SSHTester{}
	if tester.testKeyPermissions(key) {
		t.Fatal("insecure private key permissions should fail")
	}
	info, err := os.Stat(key)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("permission check mutated file mode to %o", info.Mode().Perm())
	}
}

func TestSSHKeyPermissionCheckAcceptsOwnerOnlyModes(t *testing.T) {
	tests := []struct {
		mode os.FileMode
		want bool
	}{
		{0600, true},
		{0400, true},
		{0640, false},
		{0644, false},
	}
	for _, tt := range tests {
		key := filepath.Join(t.TempDir(), "id_test")
		if err := os.WriteFile(key, []byte("key"), tt.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(key, tt.mode); err != nil {
			t.Fatal(err)
		}
		if got := (&SSHTester{}).testKeyPermissions(key); got != tt.want {
			t.Fatalf("mode %o: got %t, want %t", tt.mode, got, tt.want)
		}
	}
}

func TestKnownHostsContainsReadsHashedEntries(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen is not installed")
	}
	dir := t.TempDir()
	hostKey := filepath.Join(dir, "hostkey")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", hostKey).CombinedOutput(); err != nil {
		t.Fatalf("generate host key: %v: %s", err, out)
	}
	pub, err := os.ReadFile(hostKey + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(pub))
	knownHosts := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(knownHosts, []byte("gitlab.example.com "+fields[0]+" "+fields[1]+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !knownHostsContains(knownHosts, "gitlab.example.com") {
		t.Fatal("plain entry not found")
	}
	if out, err := exec.Command("ssh-keygen", "-H", "-f", knownHosts).CombinedOutput(); err != nil {
		t.Fatalf("hash known_hosts: %v: %s", err, out)
	}
	hashed, _ := os.ReadFile(knownHosts)
	if strings.Contains(string(hashed), "gitlab.example.com") {
		t.Fatalf("known_hosts was not hashed: %s", hashed)
	}
	if !knownHostsContains(knownHosts, "gitlab.example.com") {
		t.Fatal("hashed entry not found")
	}
	if knownHostsContains(knownHosts, "github.com") {
		t.Fatal("absent host reported as known")
	}
}
