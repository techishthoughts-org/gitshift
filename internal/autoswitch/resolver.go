// Package autoswitch resolves canonical filesystem paths to configured accounts.
package autoswitch

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

// CanonicalDirectory resolves path to an absolute, physical directory.
func CanonicalDirectory(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("directory path cannot be empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("make directory absolute: %w", err)
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("inspect directory %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path %q is not a directory", abs)
	}
	physical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", abs, err)
	}
	return filepath.Clean(physical), nil
}

// Resolve returns the deepest component-aware rule matching path.
func Resolve(rules []models.AutoSwitchRule, path string) (*models.AutoSwitchRule, error) {
	canonical, err := CanonicalDirectory(path)
	if err != nil {
		return nil, err
	}
	var match *models.AutoSwitchRule
	for i := range rules {
		rule := rules[i]
		if !contains(rule.Path, canonical) {
			continue
		}
		if match == nil || depth(rule.Path) > depth(match.Path) {
			copy := rule
			match = &copy
		}
	}
	return match, nil
}

// SortRules orders rules by path for deterministic human and JSON output.
func SortRules(rules []models.AutoSwitchRule) {
	sort.Slice(rules, func(i, j int) bool { return rules[i].Path < rules[j].Path })
}

func contains(parent, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func depth(path string) int {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	trimmed := strings.TrimPrefix(clean, volume)
	trimmed = strings.Trim(trimmed, string(filepath.Separator))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, string(filepath.Separator)))
}
