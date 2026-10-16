//go:build windows

package config

// Windows does not support opening a directory handle through os.Open for
// fsync. The atomic rename remains the durability boundary on that platform.
func syncConfigDirectory(string) error { return nil }
