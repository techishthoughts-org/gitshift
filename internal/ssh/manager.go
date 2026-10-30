package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/techishthoughts/gitshift/internal/models"
)

// Manager handles SSH configuration and key management
type Manager struct {
	homeDir    string
	configPath string
}

// NewManager creates a new SSH manager
func NewManager() *Manager {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		resolvedHome, err := os.UserHomeDir()
		if err != nil {
			resolvedHome = "."
		}
		homeDir = resolvedHome
	}

	return &Manager{
		homeDir:    homeDir,
		configPath: filepath.Join(homeDir, ".ssh", "config"),
	}
}

// SwitchToAccount switches SSH configuration to use the specified account with improved isolation
// CheckPrivateKeyFile reports a missing private key or one that group or other
// users can access. OpenSSH accepts 0600 and 0400; it rejects anything wider.
func CheckPrivateKeyFile(keyPath string) error {
	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm&0o077 != 0 {
		return fmt.Errorf("SSH key %s has permissions %o; run 'chmod 600 %s'", keyPath, perm, keyPath)
	}
	return nil
}

func (m *Manager) SwitchToAccount(accountAlias, keyPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.SwitchToAccountContext(ctx, accountAlias, keyPath, "github.com", true)
}

// SwitchToAccountContext updates the SSH host entry without mutating shell startup
// files or clearing the caller's shared SSH agent. Validation is bounded by ctx.
func (m *Manager) SwitchToAccountContext(ctx context.Context, accountAlias, keyPath, domain string, validate bool) error {
	var err error
	keyPath, err = models.NormalizeSSHKeyPath(keyPath)
	if err != nil {
		return err
	}
	if strings.IndexFunc(accountAlias+domain, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 || strings.ContainsAny(domain, " /\\") {
		return fmt.Errorf("invalid SSH account alias or domain")
	}
	// Refuse insecure permissions; switching must not silently mutate key files.
	if err := CheckPrivateKeyFile(keyPath); err != nil {
		return err
	}

	if validate {
		if err := m.TestConnectionToPlatformContext(ctx, domain, keyPath); err != nil {
			return err
		}
	}
	if err := m.UpdateSSHConfig(accountAlias, keyPath, domain); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}
	return nil
}

// GetLoadedKeys returns the list of currently loaded SSH keys
func (m *Manager) GetLoadedKeys() ([]string, error) {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "The agent has no identities") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var keys []string
	for _, line := range lines {
		if parts := strings.Fields(line); len(parts) >= 3 {
			keys = append(keys, strings.TrimSpace(parts[2]))
		}
	}
	return keys, nil
}

// detectShell detects the user's shell and returns shell type and config file path
func (m *Manager) detectShell() (shellType, configPath string, err error) {
	// On Windows, we don't update shell config (not reliable)
	if runtime.GOOS == "windows" {
		return "", "", fmt.Errorf("shell config update not supported on Windows - please set GIT_SSH_COMMAND environment variable manually:\n" +
			"  PowerShell: $env:GIT_SSH_COMMAND = 'ssh -i <path> -o IdentitiesOnly=yes'\n" +
			"  CMD:        set GIT_SSH_COMMAND=ssh -i <path> -o IdentitiesOnly=yes")
	}

	// Try to detect shell from SHELL environment variable
	shellEnv := os.Getenv("SHELL")

	// Determine shell type and config file
	switch {
	case strings.Contains(shellEnv, "zsh"):
		// ZSH: check for .zshrc, then .zsh_secrets
		zshrc := filepath.Join(m.homeDir, ".zshrc")
		if _, err := os.Stat(zshrc); err == nil {
			return "zsh", zshrc, nil
		}
		// Fall back to .zsh_secrets for backward compatibility
		return "zsh", filepath.Join(m.homeDir, ".zsh_secrets"), nil

	case strings.Contains(shellEnv, "bash"):
		// Bash: check for .bashrc on Linux, .bash_profile on macOS
		if runtime.GOOS == "darwin" {
			bashProfile := filepath.Join(m.homeDir, ".bash_profile")
			if _, err := os.Stat(bashProfile); err == nil {
				return "bash", bashProfile, nil
			}
		}
		bashrc := filepath.Join(m.homeDir, ".bashrc")
		return "bash", bashrc, nil

	case strings.Contains(shellEnv, "fish"):
		// Fish shell uses a different directory structure
		fishConfig := filepath.Join(m.homeDir, ".config", "fish", "config.fish")
		return "fish", fishConfig, nil

	case strings.Contains(shellEnv, "ksh"):
		// Korn shell
		return "ksh", filepath.Join(m.homeDir, ".kshrc"), nil

	default:
		// Default fallback: try .profile (POSIX standard)
		profile := filepath.Join(m.homeDir, ".profile")
		return "posix", profile, nil
	}
}

// updateShellConfig updates the shell configuration to set GIT_SSH_COMMAND
func (m *Manager) updateShellConfig(accountAlias, keyPath string) error {
	// Detect shell and get config file path
	shellType, configPath, err := m.detectShell()
	if err != nil {
		return err
	}

	// Read existing content
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read shell config: %w", err)
	}

	// Create backup
	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Remove existing GIT_SSH_COMMAND if it exists
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Remove both export and set variants (for fish shell)
		if !strings.HasPrefix(trimmed, "export GIT_SSH_COMMAND=") &&
			!strings.HasPrefix(trimmed, "set -x GIT_SSH_COMMAND") {
			newLines = append(newLines, line)
		}
	}

	// Add new GIT_SSH_COMMAND based on shell type
	var commandLine string
	if shellType == "fish" {
		// Fish shell uses 'set -x' for environment variables
		commandLine = fmt.Sprintf(`set -x GIT_SSH_COMMAND "ssh -i %s -o IdentitiesOnly=yes"`, keyPath)
	} else {
		// Bash, ZSH, KSH, and POSIX shells use 'export'
		commandLine = fmt.Sprintf(`export GIT_SSH_COMMAND="ssh -i %s -o IdentitiesOnly=yes"`, keyPath)
	}

	// Add comment before the command for clarity
	newLines = append(newLines, "")
	newLines = append(newLines, "# gitshift: SSH key configuration")
	newLines = append(newLines, commandLine)

	// Add a newline at the end
	newLines = append(newLines, "")

	// Write back to file
	if err := os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		// Try to restore backup if write fails
		_ = os.WriteFile(configPath, content, 0600)
		return fmt.Errorf("failed to write shell config: %w", err)
	}

	// Print helpful message about what was done
	fmt.Printf("📝 Updated %s config: %s\n", shellType, configPath)
	fmt.Printf("💡 Run 'source %s' or restart your terminal to apply changes\n", configPath)

	return nil
}

// SourceShellConfig sources the shell configuration to apply changes
func (m *Manager) SourceShellConfig() error {
	// Detect shell and get config file path
	shellType, configPath, err := m.detectShell()
	if err != nil {
		return err
	}

	var cmd *exec.Cmd

	// Build the appropriate source command for each shell
	switch shellType {
	case "fish":
		// Fish shell uses 'source' command
		cmd = exec.Command("fish", "-c", fmt.Sprintf("source %s", configPath))
	case "bash":
		// Bash uses 'source' or '.'
		cmd = exec.Command("bash", "-c", fmt.Sprintf("source %s", configPath))
	case "zsh":
		// ZSH uses 'source'
		cmd = exec.Command("zsh", "-c", fmt.Sprintf("source %s", configPath))
	case "ksh":
		// KSH uses '.'
		cmd = exec.Command("ksh", "-c", fmt.Sprintf(". %s", configPath))
	default:
		// POSIX shells use '.'
		cmd = exec.Command("sh", "-c", fmt.Sprintf(". %s", configPath))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// clearSSHAgent removes all keys from the SSH agent
func (m *Manager) clearSSHAgent() error {
	cmd := exec.Command("ssh-add", "-D")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// It's OK if there are no keys to remove
		if strings.Contains(string(output), "no identities") {
			return nil
		}
		return fmt.Errorf("ssh-add -D failed: %w", err)
	}
	return nil
}

// addKeyToAgent adds a specific key to the SSH agent
func (m *Manager) addKeyToAgent(keyPath string) error {
	cmd := exec.Command("ssh-add", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add %s failed: %w\nOutput: %s", keyPath, err, string(output))
	}
	return nil
}

// TestConnection tests the SSH connection to GitHub (deprecated, use TestConnectionToPlatform)
func (m *Manager) TestConnection() error {
	return m.TestConnectionToPlatform("github.com")
}

// TestConnectionToPlatform tests the SSH connection to a specific platform domain
func (m *Manager) TestConnectionToPlatform(domain string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.TestConnectionToPlatformContext(ctx, domain, "")
}

// TestConnectionToPlatformContext tests a host using an optional identity file.
func (m *Manager) TestConnectionToPlatformContext(ctx context.Context, domain, keyPath string) error {
	// Test SSH connection to the specified domain
	args := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-T", fmt.Sprintf("git@%s", domain)}
	if keyPath != "" {
		// Ignore ~/.ssh/config and the shared agent while validating one key.
		// IdentityFile directives are additive, so -i plus IdentitiesOnly alone
		// can still authenticate with a different key from the host block.
		args = append([]string{
			"-F", os.DevNull,
			"-i", keyPath,
			"-o", "IdentitiesOnly=yes",
			"-o", "IdentityAgent=none",
		}, args...)
	}
	testCmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := testCmd.CombinedOutput()
	outputStr := string(output)

	if isSuccessfulSSHOutput(outputStr) {
		return nil
	}

	// If no error and output suggests success
	if err == nil {
		return nil
	}

	return fmt.Errorf("SSH connection test to %s failed: %w\nOutput: %s", domain, err, outputStr)
}

func isSuccessfulSSHOutput(output string) bool {
	return strings.Contains(output, "You've successfully authenticated") ||
		strings.Contains(output, "Welcome to GitLab")
}

// updateGitHubSSHConfigV2 updates the SSH config with improved multi-account isolation
// Deprecated: Use UpdateSSHConfig with platform domain instead
func (m *Manager) updateGitHubSSHConfigV2(accountAlias, keyPath string) error {
	return m.UpdateSSHConfig(accountAlias, keyPath, "github.com")
}

// UpdateSSHConfig updates the SSH config for a specific platform domain
func (m *Manager) UpdateSSHConfig(accountAlias, keyPath, domain string) error {
	if strings.IndexFunc(accountAlias+domain, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 || strings.ContainsAny(domain, " /\\\t") {
		return fmt.Errorf("invalid SSH account alias or domain")
	}
	var err error
	keyPath, err = models.NormalizeSSHKeyPath(keyPath)
	if err != nil {
		return err
	}
	// Ensure SSH directory exists
	sshDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Read current SSH config (if exists)
	existingContent := ""
	if content, err := os.ReadFile(m.configPath); err == nil {
		if err := backupSSHConfig(m.configPath, content); err != nil {
			return err
		}
		existingContent = string(content)
	}

	return m.writeSSHConfig(m.buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, domain, existingContent))
}

// RemoveSSHConfig deletes gitshift's managed Host block for domain, keeping
// every other entry. It is a no-op when the block is absent.
func (m *Manager) RemoveSSHConfig(domain string) error {
	content, err := os.ReadFile(m.configPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("read SSH config: %w", err)
	}
	startMarker := fmt.Sprintf("# >>> gitshift managed: %s", domain)
	if !strings.Contains(string(content), startMarker) {
		return nil
	}
	if err := backupSSHConfig(m.configPath, content); err != nil {
		return err
	}
	return m.writeSSHConfig(stripManagedSSHBlock(string(content), startMarker, fmt.Sprintf("# <<< gitshift managed: %s", domain)))
}

// writeSSHConfig atomically replaces the SSH config with newConfig.
func (m *Manager) writeSSHConfig(newConfig string) error {
	sshDir := filepath.Dir(m.configPath)
	tmp, err := os.CreateTemp(sshDir, ".gitshift-ssh-config-*")
	if err != nil {
		return fmt.Errorf("create temporary SSH config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(newConfig); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary SSH config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceSSHConfigFile(tmpName, m.configPath); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// buildIsolatedSSHConfig creates an SSH config with proper multi-account isolation
// Deprecated: Use buildIsolatedSSHConfigForPlatform instead
func (m *Manager) buildIsolatedSSHConfig(accountAlias, keyPath, existingConfig string) string {
	return m.buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, "github.com", existingConfig)
}

// buildIsolatedSSHConfigForPlatform creates an SSH config for a specific platform
func (m *Manager) buildIsolatedSSHConfigForPlatform(accountAlias, keyPath, domain, existingConfig string) string {
	startMarker := fmt.Sprintf("# >>> gitshift managed: %s", domain)
	endMarker := fmt.Sprintf("# <<< gitshift managed: %s", domain)
	existingConfig = stripManagedSSHBlock(existingConfig, startMarker, endMarker)
	existingConfig = stripLegacyManagedSSHHost(existingConfig, domain)
	existingConfig = stripExactSSHHost(existingConfig, domain)

	platformName := "Git hosting"
	switch domain {
	case "github.com":
		platformName = "GitHub"
	case "gitlab.com":
		platformName = "GitLab"
	case "bitbucket.org":
		platformName = "Bitbucket"
	}

	managed := fmt.Sprintf(`%s
# %s account: %s
Host %s
    HostName %s
    User git
    IdentityFile %s
    IdentitiesOnly yes

%s
`, startMarker, platformName, accountAlias, domain, domain, quoteSSHConfigValue(keyPath), endMarker)

	// Options before the first Host/Match line apply to every host, so they must
	// stay ahead of the managed block instead of falling under its Host line.
	preamble, hosts := splitSSHConfigPreamble(existingConfig)
	var b strings.Builder
	if preamble != "" {
		b.WriteString(preamble + "\n\n")
	}
	b.WriteString(managed)
	if hosts != "" {
		b.WriteString("\n" + hosts + "\n")
	}
	return b.String()
}

// splitSSHConfigPreamble returns the trimmed global section (everything before
// the first Host or Match line) and the trimmed remainder.
func splitSSHConfigPreamble(config string) (string, string) {
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		if isSSHBlockStart(line) {
			return strings.TrimSpace(strings.Join(lines[:i], "\n")), strings.TrimSpace(strings.Join(lines[i:], "\n"))
		}
	}
	return strings.TrimSpace(config), ""
}

func isSSHBlockStart(line string) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	return len(fields) > 0 && (strings.EqualFold(fields[0], "Host") || strings.EqualFold(fields[0], "Match"))
}

// backupSSHConfig keeps the pre-gitshift config once in .gitshift-orig and the
// most recent previous config in .backup.
func backupSSHConfig(configPath string, content []byte) error {
	origPath := configPath + ".gitshift-orig"
	if _, err := os.Lstat(origPath); os.IsNotExist(err) {
		if err := os.WriteFile(origPath, content, 0600); err != nil {
			return fmt.Errorf("failed to save original SSH config: %w", err)
		}
	}
	if err := os.WriteFile(configPath+".backup", content, 0600); err != nil {
		return fmt.Errorf("failed to backup SSH config: %w", err)
	}
	return nil
}

func stripManagedSSHBlock(config, startMarker, endMarker string) string {
	lines := strings.Split(config, "\n")
	kept := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != startMarker {
			kept = append(kept, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && strings.TrimSpace(lines[end]) != endMarker {
			end++
		}
		if end == len(lines) {
			kept = append(kept, lines[i:]...)
			break
		}
		i = end
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

func stripLegacyManagedSSHHost(config, domain string) string {
	if !strings.Contains(config, "# gitshift Managed Config - DO NOT EDIT MANUALLY") {
		return config
	}
	lines := strings.Split(config, "\n")
	kept := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "# gitshift Managed Config - DO NOT EDIT MANUALLY" || trimmed == "# This file is automatically generated by gitshift" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 2 && strings.EqualFold(fields[0], "Host") && fields[1] == domain {
			for i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(next, "Host ") || strings.HasPrefix(next, "Match ") {
					break
				}
				i++
			}
			continue
		}
		kept = append(kept, lines[i])
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

// stripExactSSHHost disables the target host in pre-existing Host blocks.
// IdentityFile is additive across every matching block, so retaining a second
// exact Host github.com entry defeats IdentitiesOnly and can offer the previous
// account's key after the managed key. Mixed blocks are narrowed to their
// remaining host patterns; exact single-host blocks are commented out.
func stripExactSSHHost(config, domain string) string {
	lines := strings.Split(config, "\n")
	kept := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		fields := strings.Fields(trimmed)
		if len(fields) < 2 || !strings.EqualFold(fields[0], "Host") {
			kept = append(kept, lines[i])
			continue
		}

		remaining := make([]string, 0, len(fields)-1)
		removed := false
		for _, pattern := range fields[1:] {
			if strings.EqualFold(pattern, domain) {
				removed = true
				continue
			}
			remaining = append(remaining, pattern)
		}
		if !removed {
			kept = append(kept, lines[i])
			continue
		}
		if len(remaining) > 0 {
			indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
			kept = append(kept, indent+"Host "+strings.Join(remaining, " "))
			continue
		}

		kept = append(kept, disabledSSHLinePrefix+lines[i])
		for i+1 < len(lines) && !isSSHBlockStart(lines[i+1]) {
			i++
			if strings.TrimSpace(lines[i]) == "" {
				kept = append(kept, lines[i])
			} else {
				kept = append(kept, disabledSSHLinePrefix+lines[i])
			}
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

// disabledSSHLinePrefix marks user lines gitshift commented out so the user can
// restore them; commented lines are never parsed as Host blocks again.
const disabledSSHLinePrefix = "# gitshift-disabled: "

func quoteSSHConfigValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// preserveNonGitHubConfig extracts and preserves non-GitHub host configurations
// Deprecated: Use preserveNonPlatformConfig instead
func (m *Manager) preserveNonGitHubConfig(existingConfig string) string {
	return m.preserveNonPlatformConfig(existingConfig, "github.com")
}

// preserveNonPlatformConfig extracts and preserves configurations not related to the specified domain
func (m *Manager) preserveNonPlatformConfig(existingConfig, targetDomain string) string {
	if existingConfig == "" {
		return ""
	}

	var preserved []string
	lines := strings.Split(existingConfig, "\n")
	inTargetSection := false
	skipUntilNextHost := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Check if we're entering a target platform host section
		if strings.HasPrefix(line, "Host ") {
			hostLine := strings.TrimSpace(strings.TrimPrefix(line, "Host"))
			hosts := strings.Fields(hostLine)

			// Check if any of the hosts matches the target domain
			isTargetHost := false
			for _, host := range hosts {
				if host == targetDomain || strings.Contains(host, targetDomain) {
					isTargetHost = true
					break
				}
			}

			if isTargetHost {
				inTargetSection = true
				skipUntilNextHost = true
				continue
			}

			// If we were in a target section and found a new host, we're done
			if inTargetSection {
				inTargetSection = false
			}

			// If we were skipping until next host, stop skipping
			if skipUntilNextHost {
				skipUntilNextHost = false
			}
		}

		// Skip lines in target sections or when we're in skip mode
		if inTargetSection || skipUntilNextHost {
			continue
		}

		// Preserve all other lines
		preserved = append(preserved, lines[i])
	}

	if len(preserved) > 0 {
		// Ensure there's a blank line before adding platform config
		if preserved[len(preserved)-1] != "" {
			preserved = append(preserved, "")
		}
		return strings.Join(preserved, "\n") + "\n"
	}

	return ""
}
