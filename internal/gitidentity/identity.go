// Package gitidentity computes the Git settings that make up an account's
// identity and keeps per-directory includeIf entries in the global Git config.
package gitidentity

import (
	"fmt"
	"strings"

	"github.com/techishthoughts/gitshift/internal/models"
)

// Setting is one Git config key and value.
type Setting struct {
	Key, Value string
}

// SSHCommand pins Git to keyPath while keeping the user's SSH config (ports,
// ProxyJump, agents holding this key). IdentitiesOnly stops ssh from offering
// unrelated agent keys first.
func SSHCommand(keyPath string) string {
	return fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", ShellQuote(keyPath))
}

// ShellQuote quotes value for a POSIX shell.
func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// SigningSettings returns the gpg.format, user.signingkey, commit.gpgsign and
// tag.gpgsign values for the account, and the keys to unset when it signs with
// nothing and has no GPG key.
func SigningSettings(account *models.Account) (set []Setting, unset []string, err error) {
	format, key := account.CommitSigning()
	switch format {
	case models.SigningFormatSSH:
		keyPath, err := models.NormalizeSSHKeyPath(key)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH signing key: %w", err)
		}
		return signing("ssh", keyPath, "true"), nil, nil
	case models.SigningFormatGPG:
		return signing("openpgp", key, "true"), nil, nil
	}
	if account.HasGPGKey() {
		// A GPG key whose signing is turned off stays selected.
		return signing("openpgp", account.GPGKeyID, "false"), nil, nil
	}
	return []Setting{{"commit.gpgsign", "false"}, {"tag.gpgsign", "false"}}, []string{"user.signingkey", "gpg.format"}, nil
}

func signing(format, key, enabled string) []Setting {
	return []Setting{
		{"gpg.format", format},
		{"user.signingkey", key},
		{"commit.gpgsign", enabled},
		{"tag.gpgsign", enabled},
	}
}

// IdentitySettings returns every value an account's include file sets: name,
// email, Git's SSH command and signing. Keys that switching would unset are
// left out; commit.gpgsign=false already disables signing.
func IdentitySettings(account *models.Account) ([]Setting, error) {
	var settings []Setting
	if account.Name != "" {
		settings = append(settings, Setting{"user.name", account.Name})
	}
	if account.Email != "" {
		settings = append(settings, Setting{"user.email", account.Email})
	}
	if account.SSHKeyPath != "" {
		keyPath, err := models.NormalizeSSHKeyPath(account.SSHKeyPath)
		if err != nil {
			return nil, err
		}
		settings = append(settings, Setting{"core.sshCommand", SSHCommand(keyPath)})
	}
	signingSettings, _, err := SigningSettings(account)
	if err != nil {
		return nil, err
	}
	return append(settings, signingSettings...), nil
}
