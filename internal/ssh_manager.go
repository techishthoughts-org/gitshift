package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealSSHManager implements the SSHManager interface with unified SSH operations
type RealSSHManager struct {
	logger    observability.Logger
	configMux sync.RWMutex // Protects SSH config file operations
}

// NewSSHManager creates a new SSH manager
func NewSSHManager(logger observability.Logger) SSHManager {
	return &RealSSHManager{
		logger: logger,
	}
}

// GenerateKey generates a new SSH key with enhanced security
func (sm *RealSSHManager) GenerateKey(ctx context.Context, req GenerateKeyRequest) (*SSHKey, error) {
	sm.logger.Info(ctx, "generating_ssh_key",
		observability.F("type", req.Type),
		observability.F("email", req.Email),
		observability.F("key_path", req.KeyPath),
	)

	// Validate key type
	if req.Type != "ed25519" && req.Type != "rsa" {
		return nil, fmt.Errorf("unsupported key type: %s (supported: ed25519, rsa)", req.Type)
	}

	// Check if key already exists
	if _, err := os.Stat(req.KeyPath); err == nil && !req.Overwrite {
		return nil, fmt.Errorf("SSH key already exists at %s (use --overwrite to replace)", req.KeyPath)
	}

	// Create directory if needed
	keyDir := filepath.Dir(req.KeyPath)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate the key
	args := []string{
		"-t", req.Type,
		"-C", req.Email,
		"-f", req.KeyPath,
		"-N", "", // No passphrase for now
	}

	// Add key size for RSA
	if req.Type == "rsa" {
		args = append(args, "-b", "4096")
	}

	// Add Ed25519 security options
	if req.Type == "ed25519" {
		args = append(args, "-o") // Use OpenSSH format
	}

	cmd := exec.CommandContext(ctx, "ssh-keygen", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		sm.logger.Error(ctx, "failed_to_generate_ssh_key",
			observability.F("type", req.Type),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return nil, fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// Set correct permissions
	if err := os.Chmod(req.KeyPath, 0600); err != nil {
		return nil, fmt.Errorf("failed to set private key permissions: %w", err)
	}

	if err := os.Chmod(req.KeyPath+".pub", 0644); err != nil {
		return nil, fmt.Errorf("failed to set public key permissions: %w", err)
	}

	// Get key information
	keyInfo, err := sm.ValidateKey(ctx, req.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to validate generated key: %w", err)
	}

	// Read public key
	publicKeyContent, err := os.ReadFile(req.KeyPath + ".pub")
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	key := &SSHKey{
		Path:        req.KeyPath,
		Type:        req.Type,
		Fingerprint: keyInfo.Fingerprint,
		PublicKey:   strings.TrimSpace(string(publicKeyContent)),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	sm.logger.Info(ctx, "ssh_key_generated_successfully",
		observability.F("type", req.Type),
		observability.F("fingerprint", key.Fingerprint),
	)

	return key, nil
}

// ListKeys lists all SSH keys in the SSH directory
func (sm *RealSSHManager) ListKeys(ctx context.Context) ([]*SSHKeyInfo, error) {
	sm.logger.Info(ctx, "listing_ssh_keys")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	keys := []*SSHKeyInfo{}

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".pub") || entry.Name() == "known_hosts" || entry.Name() == "config" {
			continue
		}

		// Skip files that don't look like SSH keys
		if !strings.HasPrefix(entry.Name(), "id_") {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())
		keyInfo, err := sm.ValidateKey(ctx, keyPath)
		if err != nil {
			sm.logger.Warn(ctx, "invalid_ssh_key_found",
				observability.F("key_path", keyPath),
				observability.F("error", err.Error()),
			)
			// Include invalid keys with error information
			keyInfo = &SSHKeyInfo{
				Path:     keyPath,
				Valid:    false,
				Exists:   true,
				Readable: false,
			}
		}

		keys = append(keys, keyInfo)
	}

	sm.logger.Info(ctx, "ssh_keys_listed",
		observability.F("count", len(keys)),
	)

	return keys, nil
}

// ValidateKey validates an SSH key and returns detailed information
func (sm *RealSSHManager) ValidateKey(ctx context.Context, keyPath string) (*SSHKeyInfo, error) {
	sm.logger.Info(ctx, "validating_ssh_key",
		observability.F("key_path", keyPath),
	)

	keyInfo := &SSHKeyInfo{
		Path:     keyPath,
		Valid:    false,
		Exists:   false,
		Readable: false,
	}

	// Check if private key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return keyInfo, fmt.Errorf("SSH key file does not exist: %s", keyPath)
	}
	keyInfo.Exists = true

	// Check if private key is readable
	if _, err := os.Open(keyPath); err != nil {
		return keyInfo, fmt.Errorf("SSH key file is not readable: %w", err)
	}
	keyInfo.Readable = true

	// Check public key exists
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		return keyInfo, fmt.Errorf("public key file does not exist: %s", pubKeyPath)
	}

	// Get key fingerprint and type using ssh-keygen
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-l", "-f", pubKeyPath)
	output, err := cmd.Output()
	if err != nil {
		return keyInfo, fmt.Errorf("failed to get key information: %w", err)
	}

	// Parse output: "2048 SHA256:abc123... user@host (RSA)"
	parts := strings.Fields(string(output))
	if len(parts) >= 4 {
		keyInfo.Size = 0
		fmt.Sscanf(parts[0], "%d", &keyInfo.Size)
		keyInfo.Fingerprint = parts[1]
		keyInfo.Type = strings.Trim(parts[len(parts)-1], "()")
	}

	// Extract email from public key
	pubKeyContent, err := os.ReadFile(pubKeyPath)
	if err == nil {
		pubKeyParts := strings.Fields(string(pubKeyContent))
		if len(pubKeyParts) >= 3 {
			keyInfo.Email = pubKeyParts[2]
		}
	}

	// Validate key permissions
	if info, err := os.Stat(keyPath); err == nil {
		if info.Mode().Perm() != 0600 {
			sm.logger.Warn(ctx, "insecure_ssh_key_permissions",
				observability.F("key_path", keyPath),
				observability.F("permissions", fmt.Sprintf("%o", info.Mode().Perm())),
			)
		}
	}

	keyInfo.Valid = true

	sm.logger.Info(ctx, "ssh_key_validated",
		observability.F("key_path", keyPath),
		observability.F("type", keyInfo.Type),
		observability.F("fingerprint", keyInfo.Fingerprint),
	)

	return keyInfo, nil
}

// DeleteKey deletes an SSH key pair
func (sm *RealSSHManager) DeleteKey(ctx context.Context, keyPath string) error {
	sm.logger.Info(ctx, "deleting_ssh_key",
		observability.F("key_path", keyPath),
	)

	// Validate key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key does not exist: %s", keyPath)
	}

	// Delete private key
	if err := os.Remove(keyPath); err != nil {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	// Delete public key
	pubKeyPath := keyPath + ".pub"
	if err := os.Remove(pubKeyPath); err != nil && !os.IsNotExist(err) {
		sm.logger.Warn(ctx, "failed_to_delete_public_key",
			observability.F("pub_key_path", pubKeyPath),
			observability.F("error", err.Error()),
		)
	}

	sm.logger.Info(ctx, "ssh_key_deleted",
		observability.F("key_path", keyPath),
	)

	return nil
}

// GenerateConfig generates SSH configuration for accounts
func (sm *RealSSHManager) GenerateConfig(ctx context.Context, accounts []*Account) (string, error) {
	sm.logger.Info(ctx, "generating_ssh_config",
		observability.F("account_count", len(accounts)),
	)

	var configBuilder strings.Builder

	// Header
	configBuilder.WriteString("# GitPersona SSH Configuration\n")
	configBuilder.WriteString("# Generated automatically - do not edit manually\n")
	configBuilder.WriteString(fmt.Sprintf("# Generated at: %s\n\n", time.Now().Format(time.RFC3339)))

	// Global SSH settings
	configBuilder.WriteString("# Global SSH settings\n")
	configBuilder.WriteString("Host *\n")
	configBuilder.WriteString("    ServerAliveInterval 60\n")
	configBuilder.WriteString("    ServerAliveCountMax 3\n")
	configBuilder.WriteString("    AddKeysToAgent yes\n")
	configBuilder.WriteString("    UseKeychain yes\n\n")

	// Account-specific configurations
	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		configBuilder.WriteString(fmt.Sprintf("# Account: %s (%s)\n", account.Alias, account.Name))

		// Primary host entry
		configBuilder.WriteString(fmt.Sprintf("Host github.com-%s\n", account.Alias))
		configBuilder.WriteString("    HostName github.com\n")
		configBuilder.WriteString("    User git\n")
		configBuilder.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
		configBuilder.WriteString("    IdentitiesOnly yes\n")
		configBuilder.WriteString("    PreferredAuthentications publickey\n")

		// Alternative host entry with username
		if account.GitHubUsername != "" {
			configBuilder.WriteString(fmt.Sprintf("Host %s.github.com\n", account.GitHubUsername))
			configBuilder.WriteString("    HostName github.com\n")
			configBuilder.WriteString("    User git\n")
			configBuilder.WriteString(fmt.Sprintf("    IdentityFile %s\n", account.SSHKeyPath))
			configBuilder.WriteString("    IdentitiesOnly yes\n")
		}

		configBuilder.WriteString("\n")
	}

	config := configBuilder.String()

	sm.logger.Info(ctx, "ssh_config_generated",
		observability.F("account_count", len(accounts)),
		observability.F("config_size", len(config)),
	)

	return config, nil
}

// InstallConfig installs SSH configuration
func (sm *RealSSHManager) InstallConfig(ctx context.Context, configContent string) error {
	// Lock to prevent concurrent SSH config modifications
	sm.configMux.Lock()
	defer sm.configMux.Unlock()

	sm.logger.Info(ctx, "installing_ssh_config")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	configPath := filepath.Join(sshDir, "config")

	// Backup existing config
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + ".gitpersona-backup-" + time.Now().Format("20060102-150405")
		if err := os.Rename(configPath, backupPath); err != nil {
			sm.logger.Warn(ctx, "failed_to_backup_ssh_config",
				observability.F("error", err.Error()),
			)
		} else {
			sm.logger.Info(ctx, "ssh_config_backed_up",
				observability.F("backup_path", backupPath),
			)
		}
	}

	// Write new config
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	sm.logger.Info(ctx, "ssh_config_installed",
		observability.F("config_path", configPath),
	)

	return nil
}

// ValidateConfig validates SSH configuration
func (sm *RealSSHManager) ValidateConfig(ctx context.Context) (*SSHConfigValidation, error) {
	sm.logger.Info(ctx, "validating_ssh_config")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	validation := &SSHConfigValidation{
		Valid:  true,
		Issues: []string{},
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	configPath := filepath.Join(sshDir, "config")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		validation.Valid = false
		validation.Issues = append(validation.Issues, "SSH config file does not exist")
		return validation, nil
	}

	// Check config permissions
	if info, err := os.Stat(configPath); err == nil {
		if info.Mode().Perm() != 0600 {
			validation.Issues = append(validation.Issues, fmt.Sprintf("SSH config has incorrect permissions: %o (should be 600)", info.Mode().Perm()))
		}
	}

	// Test config syntax
	cmd := exec.CommandContext(ctx, "ssh", "-F", configPath, "-T", "git@github.com", "-o", "ConnectTimeout=1")
	if err := cmd.Run(); err != nil {
		// This might fail for connectivity reasons, so we don't mark as invalid
		sm.logger.Debug(ctx, "ssh_config_connectivity_test_failed",
			observability.F("error", err.Error()),
		)
	}

	if len(validation.Issues) > 0 {
		validation.Valid = false
	}

	sm.logger.Info(ctx, "ssh_config_validated",
		observability.F("valid", validation.Valid),
		observability.F("issues_count", len(validation.Issues)),
	)

	return validation, nil
}

// StartAgent starts SSH agent for an account
func (sm *RealSSHManager) StartAgent(ctx context.Context, account *Account) error {
	sm.logger.Info(ctx, "starting_ssh_agent",
		observability.F("account", account.Alias),
	)

	// Check if agent is already running
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		sm.logger.Info(ctx, "ssh_agent_already_running")
		return sm.LoadKey(ctx, account.SSHKeyPath)
	}

	// Start new agent
	cmd := exec.CommandContext(ctx, "ssh-agent", "-s")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start SSH agent: %w", err)
	}

	// Parse agent output to set environment variables
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "SSH_AUTH_SOCK=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv("SSH_AUTH_SOCK", strings.Trim(parts[1], ";"))
			}
		}
		if strings.HasPrefix(line, "SSH_AGENT_PID=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv("SSH_AGENT_PID", strings.Trim(parts[1], ";"))
			}
		}
	}

	// Load the key
	return sm.LoadKey(ctx, account.SSHKeyPath)
}

// StopAgent stops SSH agent
func (sm *RealSSHManager) StopAgent(ctx context.Context, account *Account) error {
	sm.logger.Info(ctx, "stopping_ssh_agent",
		observability.F("account", account.Alias),
	)

	agentPID := os.Getenv("SSH_AGENT_PID")
	if agentPID == "" {
		return fmt.Errorf("no SSH agent running")
	}

	cmd := exec.CommandContext(ctx, "kill", agentPID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop SSH agent: %w", err)
	}

	// Clear environment variables
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")

	sm.logger.Info(ctx, "ssh_agent_stopped")
	return nil
}

// LoadKey loads an SSH key into the agent
func (sm *RealSSHManager) LoadKey(ctx context.Context, keyPath string) error {
	sm.logger.Info(ctx, "loading_ssh_key",
		observability.F("key_path", keyPath),
	)

	if keyPath == "" {
		return fmt.Errorf("no key path provided")
	}

	// Validate key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key does not exist: %s", keyPath)
	}

	// Add key to agent
	cmd := exec.CommandContext(ctx, "ssh-add", keyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		sm.logger.Error(ctx, "failed_to_load_ssh_key",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("failed to load SSH key: %w", err)
	}

	sm.logger.Info(ctx, "ssh_key_loaded",
		observability.F("key_path", keyPath),
	)

	return nil
}

// TestConnectivity tests SSH connectivity for an account
func (sm *RealSSHManager) TestConnectivity(ctx context.Context, account *Account) (*ConnectivityResult, error) {
	sm.logger.Info(ctx, "testing_ssh_connectivity",
		observability.F("account", account.Alias),
	)

	start := time.Now()

	result := &ConnectivityResult{
		Account: account.Alias,
		Success: false,
		Details: make(map[string]interface{}),
	}

	if account.SSHKeyPath == "" {
		result.Message = "No SSH key configured"
		return result, nil
	}

	// Test SSH connection to GitHub
	args := []string{
		"-T",
		"git@github.com",
		"-i", account.SSHKeyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ConnectTimeout=10",
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()

	result.Latency = time.Since(start).Milliseconds()
	outputStr := string(output)

	// SSH to GitHub returns exit code 1 on successful auth
	if strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Hi ") {
		result.Success = true
		result.Message = "SSH connectivity successful"

		// Extract username from output
		if idx := strings.Index(outputStr, "Hi "); idx >= 0 {
			line := outputStr[idx:]
			if parts := strings.Fields(line); len(parts) >= 2 {
				result.Details["github_username"] = strings.Trim(parts[1], "!")
			}
		}
	} else {
		result.Message = fmt.Sprintf("SSH connectivity failed: %s", outputStr)
		if err != nil {
			result.Details["error"] = err.Error()
		}
	}

	result.Details["output"] = outputStr
	result.Details["latency_ms"] = result.Latency

	sm.logger.Info(ctx, "ssh_connectivity_tested",
		observability.F("account", account.Alias),
		observability.F("success", result.Success),
		observability.F("latency_ms", result.Latency),
	)

	return result, nil
}

// DiagnoseIssues diagnoses SSH configuration issues
func (sm *RealSSHManager) DiagnoseIssues(ctx context.Context) ([]*SSHIssue, error) {
	sm.logger.Info(ctx, "diagnosing_ssh_issues")

	issues := []*SSHIssue{}

	// Check SSH agent
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		issues = append(issues, &SSHIssue{
			Type:        "ssh_agent_not_running",
			Severity:    "medium",
			Description: "SSH agent is not running",
			Fix:         "Start SSH agent with: eval $(ssh-agent)",
			AutoFixable: true,
		})
	}

	// Check SSH directory permissions
	homeDir, _ := os.UserHomeDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil {
		if info.Mode().Perm()&077 != 0 {
			issues = append(issues, &SSHIssue{
				Type:        "ssh_directory_permissions",
				Severity:    "high",
				Description: "SSH directory has incorrect permissions",
				Fix:         fmt.Sprintf("Fix with: chmod 700 %s", sshDir),
				AutoFixable: true,
			})
		}
	}

	// Check for keys with bad permissions
	keys, _ := sm.ListKeys(ctx)
	for _, key := range keys {
		if !key.Valid {
			continue
		}

		if info, err := os.Stat(key.Path); err == nil {
			if info.Mode().Perm() != 0600 {
				issues = append(issues, &SSHIssue{
					Type:        "ssh_key_permissions",
					Severity:    "high",
					Description: fmt.Sprintf("SSH key has incorrect permissions: %s", key.Path),
					Fix:         fmt.Sprintf("Fix with: chmod 600 %s", key.Path),
					AutoFixable: true,
				})
			}
		}
	}

	sm.logger.Info(ctx, "ssh_issues_diagnosed",
		observability.F("issues_count", len(issues)),
	)

	return issues, nil
}

// FixIssues automatically fixes SSH issues
func (sm *RealSSHManager) FixIssues(ctx context.Context, issues []*SSHIssue) error {
	sm.logger.Info(ctx, "fixing_ssh_issues",
		observability.F("issues_count", len(issues)),
	)

	homeDir, _ := os.UserHomeDir()

	for _, issue := range issues {
		if !issue.AutoFixable || issue.Fixed {
			continue
		}

		switch issue.Type {
		case "ssh_directory_permissions":
			sshDir := filepath.Join(homeDir, ".ssh")
			if err := os.Chmod(sshDir, 0700); err == nil {
				issue.Fixed = true
				sm.logger.Info(ctx, "fixed_ssh_directory_permissions",
					observability.F("ssh_dir", sshDir),
				)
			}

		case "ssh_key_permissions":
			// Extract key path from description - this is a bit hacky
			if strings.Contains(issue.Description, ":") {
				parts := strings.Split(issue.Description, ":")
				if len(parts) >= 2 {
					keyPath := strings.TrimSpace(parts[1])
					if err := os.Chmod(keyPath, 0600); err == nil {
						issue.Fixed = true
						sm.logger.Info(ctx, "fixed_ssh_key_permissions",
							observability.F("key_path", keyPath),
						)
					}
				}
			}

		case "ssh_agent_not_running":
			// Note: Starting SSH agent requires special handling as it affects environment
			// This should be handled by the calling code
			sm.logger.Info(ctx, "ssh_agent_start_required")
		}
	}

	fixedCount := 0
	for _, issue := range issues {
		if issue.Fixed {
			fixedCount++
		}
	}

	sm.logger.Info(ctx, "ssh_issues_fix_completed",
		observability.F("total", len(issues)),
		observability.F("fixed", fixedCount),
	)

	return nil
}

// SwitchToAccount switches SSH configuration to use a specific account with isolation
func (sm *RealSSHManager) SwitchToAccount(ctx context.Context, alias, keyPath string) error {
	sm.logger.Info(ctx, "switching_ssh_to_account",
		observability.F("alias", alias),
		observability.F("key_path", keyPath),
	)

	// Validate key exists
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}

	// Update SSH config to use the specific key for GitHub
	if err := sm.updateGitHubSSHConfig(ctx, alias, keyPath); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	// Clear SSH agent to force re-authentication
	if err := sm.clearSSHAgent(ctx); err != nil {
		sm.logger.Error(ctx, "failed_to_clear_ssh_agent",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to clear SSH agent: %w", err)
	}

	// Add only the specific key to agent
	if err := sm.addKeyToAgent(ctx, keyPath); err != nil {
		sm.logger.Error(ctx, "failed_to_add_key_to_agent",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to add SSH key to agent: %w", err)
	}

	sm.logger.Info(ctx, "ssh_switched_to_account_successfully",
		observability.F("alias", alias),
	)

	return nil
}

// updateGitHubSSHConfig updates the SSH config to use the specific key for GitHub
func (sm *RealSSHManager) updateGitHubSSHConfig(ctx context.Context, alias, keyPath string) error {
	// Lock to prevent concurrent SSH config modifications
	sm.configMux.Lock()
	defer sm.configMux.Unlock()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".ssh", "config")

	// Read current SSH config
	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	// Create a completely new SSH config optimized for GitPersona
	newConfig := sm.buildOptimizedSSHConfig(alias, keyPath, string(content))

	// Write the updated config back
	if err := os.WriteFile(configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	sm.logger.Info(ctx, "ssh_config_updated_for_account",
		observability.F("alias", alias),
		observability.F("config_path", configPath),
	)

	return nil
}

// buildOptimizedSSHConfig creates an optimized SSH config for GitPersona account switching
func (sm *RealSSHManager) buildOptimizedSSHConfig(alias, keyPath, existingConfig string) string {
	var result strings.Builder

	// Add GitPersona header
	result.WriteString("# GitPersona SSH Configuration\n")
	result.WriteString("# This configuration prevents SSH key conflicts when using multiple GitHub accounts\n")
	result.WriteString("# Last updated for account: " + alias + "\n\n")

	// Add GitHub host configuration with strict isolation
	result.WriteString("Host github.com\n")
	result.WriteString("    HostName github.com\n")
	result.WriteString("    User git\n")
	result.WriteString(fmt.Sprintf("    IdentityFile %s\n", keyPath))
	result.WriteString("    IdentitiesOnly yes\n")
	result.WriteString("    PreferredAuthentications publickey\n")
	result.WriteString("    AddKeysToAgent no\n") // Prevent automatic key loading
	result.WriteString("    UseKeychain no\n")    // Prevent keychain integration
	result.WriteString("\n")

	// Preserve any non-GitHub host configurations from existing config
	if existingConfig != "" {
		lines := strings.Split(existingConfig, "\n")
		inGitHubSection := false

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)

			// Skip GitPersona header comments and github.com sections
			if strings.HasPrefix(trimmedLine, "# GitPersona") {
				continue
			}

			if strings.HasPrefix(trimmedLine, "Host github.com") || strings.Contains(trimmedLine, "github-") {
				inGitHubSection = true
				continue
			}

			// Check if we're leaving a GitHub section
			if inGitHubSection && strings.HasPrefix(trimmedLine, "Host ") && !strings.Contains(trimmedLine, "github") {
				inGitHubSection = false
			}

			// Skip lines that are part of GitHub sections
			if inGitHubSection {
				continue
			}

			// Preserve other host configurations
			if trimmedLine != "" || !inGitHubSection {
				result.WriteString(line + "\n")
			}
		}
	}

	return result.String()
}

// clearSSHAgent removes all keys from the SSH agent
func (sm *RealSSHManager) clearSSHAgent(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ssh-add", "-D")
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
func (sm *RealSSHManager) addKeyToAgent(ctx context.Context, keyPath string) error {
	cmd := exec.CommandContext(ctx, "ssh-add", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add %s failed: %w\nOutput: %s", keyPath, err, string(output))
	}
	return nil
}
