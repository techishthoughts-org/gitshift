//go:build !windows

package config

import (
	"fmt"
	"os"
)

func syncConfigDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config directory for sync: %w", err)
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return fmt.Errorf("failed to sync config directory: %w", err)
	}
	if err := dir.Close(); err != nil {
		return fmt.Errorf("failed to close config directory: %w", err)
	}
	return nil
}
