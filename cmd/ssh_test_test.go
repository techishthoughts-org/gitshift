package cmd

import (
	"os"
	"path/filepath"
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
