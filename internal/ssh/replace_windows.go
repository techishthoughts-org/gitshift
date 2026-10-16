//go:build windows

package ssh

import "golang.org/x/sys/windows"

func replaceSSHConfigFile(source, destination string) error {
	return windows.Rename(source, destination)
}
