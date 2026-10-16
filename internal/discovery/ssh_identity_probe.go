package discovery

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	githubGreeting   = regexp.MustCompile(`(?m)Hi ([^!\r\n]+)! You've successfully authenticated`)
	gitlabGreeting   = regexp.MustCompile(`(?m)Welcome to GitLab, @?([^!\s\r\n]+)!`)
	probeSSHIdentity = probeSSHIdentityWithOpenSSH
)

// enrichSSHAccountsWithProviderIdentity attaches a key only when an isolated
// SSH authentication result exactly matches one provider identity on that
// host. It deliberately ignores the user's ~/.ssh/config so a currently active
// key cannot masquerade as the key being inspected.
func enrichSSHAccountsWithProviderIdentity(ctx context.Context, providers, keys []*DiscoveredAccount) {
	type providerIdentity struct {
		platform string
		host     string
		username string
	}
	byHost := make(map[string][]providerIdentity)
	for _, provider := range providers {
		if provider == nil || provider.Account == nil {
			continue
		}
		host := strings.ToLower(strings.TrimSpace(provider.GetDomain()))
		username := strings.TrimSpace(provider.GetUsername())
		if host == "" || username == "" {
			continue
		}
		byHost[host] = append(byHost[host], providerIdentity{
			platform: provider.GetPlatform(), host: host, username: username,
		})
	}

	for _, key := range keys {
		if key == nil || key.Account == nil || key.SSHKeyPath == "" {
			continue
		}
		var matches []providerIdentity
		for host, identities := range byHost {
			username, ok := probeSSHIdentity(ctx, host, key.SSHKeyPath)
			if !ok {
				continue
			}
			for _, identity := range identities {
				if strings.EqualFold(identity.username, username) {
					matches = append(matches, identity)
				}
			}
		}
		if len(matches) != 1 {
			continue
		}
		match := matches[0]
		key.Username = match.username
		key.Platform = match.platform
		key.Domain = match.host
		if match.platform == "github" {
			key.GitHubUsername = match.username
		}
		key.Alias = sanitizeAlias(match.username)
		key.Confidence = 10
	}
}

func probeSSHIdentityWithOpenSSH(ctx context.Context, host, keyPath string) (string, bool) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	args := []string{
		"-F", os.DevNull,
		"-T",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "IdentityAgent=none",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"git@" + host,
	}
	output, _ := exec.CommandContext(probeCtx, "ssh", args...).CombinedOutput()
	return parseSSHAuthenticatedUsername(string(output))
}

func parseSSHAuthenticatedUsername(output string) (string, bool) {
	for _, pattern := range []*regexp.Regexp{githubGreeting, gitlabGreeting} {
		if match := pattern.FindStringSubmatch(output); len(match) == 2 {
			username := strings.TrimSpace(match[1])
			return username, username != ""
		}
	}
	return "", false
}
