package autoswitch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/techishthoughts/gitshift/internal/models"
)

func TestResolveUsesDeepestComponentMatch(t *testing.T) {
	root := t.TempDir()
	personal := filepath.Join(root, "dev", "personal")
	nested := filepath.Join(personal, "special")
	work := filepath.Join(root, "dev", "work")
	for _, dir := range []string{nested, work, filepath.Join(root, "dev", "personal-other")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	personal, _ = CanonicalDirectory(personal)
	nested, _ = CanonicalDirectory(nested)
	rules := []models.AutoSwitchRule{{Path: personal, Account: "personal"}, {Path: nested, Account: "special"}}
	got, err := Resolve(rules, nested)
	if err != nil || got == nil || got.Account != "special" {
		t.Fatalf("Resolve() = %#v, %v", got, err)
	}
	got, err = Resolve(rules, filepath.Join(root, "dev", "personal-other"))
	if err != nil || got != nil {
		t.Fatalf("component prefix matched: %#v, %v", got, err)
	}
}

func TestCanonicalDirectoryResolvesSymlinkAndRejectsFiles(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := CanonicalDirectory(link)
	real, canonicalErr := CanonicalDirectory(real)
	if canonicalErr != nil {
		t.Fatal(canonicalErr)
	}
	if err != nil || got != real {
		t.Fatalf("CanonicalDirectory() = %q, %v", got, err)
	}
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalDirectory(file); err == nil {
		t.Fatal("file accepted as directory")
	}
}
