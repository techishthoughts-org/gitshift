package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// IsolatedSSHManager provides complete SSH isolation per account
type IsolatedSSHManager struct {
	logger        observability.Logger
	socketDir     string
	agentSockets  map[string]*SSHAgentProcess
	mutex         sync.RWMutex
	defaultSSHDir string
}

// SSHAgentProcess represents an isolated SSH agent process
type SSHAgentProcess struct {
	AccountAlias string    `json:"account_alias"`
	PID          int       `json:"pid"`
	SocketPath   string    `json:"socket_path"`
	KeyPath      string    `json:"key_path"`
	StartTime    time.Time `json:"start_time"`
	LastUsed     time.Time `json:"last_used"`
	IsRunning    bool      `json:"is_running"`
	LoadedKeys   []string  `json:"loaded_keys"`
}

// SSHIsolationConfig defines SSH isolation parameters
type SSHIsolationConfig struct {
	StrictIsolation     bool          `json:"strict_isolation"`
	AutoCleanup         bool          `json:"auto_cleanup"`
	SocketTimeout       time.Duration `json:"socket_timeout"`
	KeyLoadTimeout      time.Duration `json:"key_load_timeout"`
	MaxIdleTime         time.Duration `json:"max_idle_time"`
	ForceIdentitiesOnly bool          `json:"force_identities_only"`
}

// NewIsolatedSSHManager creates a new SSH manager with complete account isolation
func NewIsolatedSSHManager(logger observability.Logger, config *SSHIsolationConfig) (*IsolatedSSHManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	socketDir := filepath.Join(homeDir, ".config", "gitpersona", "ssh-sockets")

	// Ensure socket directory exists with strict permissions
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	manager := &IsolatedSSHManager{
		logger:        logger,
		socketDir:     socketDir,
		agentSockets:  make(map[string]*SSHAgentProcess),
		defaultSSHDir: filepath.Join(homeDir, ".ssh"),
	}

	// Clean up any leftover sockets from previous sessions
	if config.AutoCleanup {
		if err := manager.cleanupOrphanedSockets(context.Background()); err != nil {
			logger.Warn(context.Background(), "failed_to_cleanup_orphaned_sockets",
				observability.F("error", err.Error()),
			)
		}
	}

	return manager, nil
}

// SwitchToAccount switches SSH to use a specific account with complete isolation
func (m *IsolatedSSHManager) SwitchToAccount(ctx context.Context, accountAlias, keyPath string) error {
	m.logger.Info(ctx, "switching_ssh_to_isolated_account",
		observability.F("account", accountAlias),
		observability.F("key_path", keyPath),
	)

	// Validate key exists
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}

	// Stop any existing agent for this account
	if err := m.stopAccountAgent(ctx, accountAlias); err != nil {
		m.logger.Warn(ctx, "failed_to_stop_existing_agent",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
	}

	// Start new isolated agent
	agentProcess, err := m.startIsolatedAgent(ctx, accountAlias, keyPath)
	if err != nil {
		return fmt.Errorf("failed to start isolated SSH agent: %w", err)
	}

	// Load the account key
	if err := m.loadKeyIntoAgent(ctx, agentProcess, keyPath); err != nil {
		// Clean up on failure
		m.stopAccountAgent(ctx, accountAlias)
		return fmt.Errorf("failed to load key into agent: %w", err)
	}

	// Set environment for this session
	if err := m.setSSHEnvironment(ctx, agentProcess); err != nil {
		m.stopAccountAgent(ctx, accountAlias)
		return fmt.Errorf("failed to set SSH environment: %w", err)
	}

	// Generate SSH config for this account
	if err := m.generateSSHConfig(ctx, accountAlias, keyPath); err != nil {
		m.logger.Warn(ctx, "failed_to_generate_ssh_config",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
	}

	// Test SSH connectivity
	if err := m.testSSHConnectivity(ctx, accountAlias); err != nil {
		m.logger.Warn(ctx, "ssh_connectivity_test_failed",
			observability.F("account", accountAlias),
			observability.F("error", err.Error()),
		)
		// Don't fail the switch for connectivity issues
	}

	m.logger.Info(ctx, "ssh_switched_to_isolated_account_successfully",
		observability.F("account", accountAlias),
		observability.F("socket_path", agentProcess.SocketPath),
		observability.F("pid", agentProcess.PID),
	)

	return nil
}

// startIsolatedAgent starts a new SSH agent for an account
func (m *IsolatedSSHManager) startIsolatedAgent(ctx context.Context, accountAlias, keyPath string) (*SSHAgentProcess, error) {
	// Create unique socket path for this account
	socketPath := filepath.Join(m.socketDir, fmt.Sprintf("ssh-agent-%s-%d", accountAlias, time.Now().Unix()))

	// Start SSH agent with isolated socket
	cmd := exec.CommandContext(ctx, "ssh-agent", "-a", socketPath, "-t", "3600") // 1 hour timeout
	cmd.Env = []string{}                                                         // Start with clean environment

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to start SSH agent: %w", err)
	}

	// Parse SSH agent output to get PID
	lines := strings.Split(string(output), "\n")
	var pid int
	for _, line := range lines {
		if strings.HasPrefix(line, "SSH_AGENT_PID=") {
			pidStr := strings.TrimPrefix(line, "SSH_AGENT_PID=")
			pidStr = strings.TrimSuffix(pidStr, ";")
			if p, err := strconv.Atoi(pidStr); err == nil {
				pid = p
				break
			}
		}
	}

	if pid == 0 {
		return nil, fmt.Errorf("failed to get SSH agent PID")
	}

	agentProcess := &SSHAgentProcess{
		AccountAlias: accountAlias,
		PID:          pid,
		SocketPath:   socketPath,
		KeyPath:      keyPath,
		StartTime:    time.Now(),
		LastUsed:     time.Now(),
		IsRunning:    true,
		LoadedKeys:   []string{},
	}

	// Store agent process
	m.mutex.Lock()
	m.agentSockets[accountAlias] = agentProcess
	m.mutex.Unlock()

	m.logger.Info(ctx, "isolated_ssh_agent_started",
		observability.F("account", accountAlias),
		observability.F("pid", pid),
		observability.F("socket_path", socketPath),
	)

	return agentProcess, nil
}

// loadKeyIntoAgent loads a key into the isolated agent
func (m *IsolatedSSHManager) loadKeyIntoAgent(ctx context.Context, agent *SSHAgentProcess, keyPath string) error {
	// Set SSH_AUTH_SOCK for this operation
	cmd := exec.CommandContext(ctx, "ssh-add", keyPath)
	cmd.Env = []string{
		"SSH_AUTH_SOCK=" + agent.SocketPath,
		"HOME=" + os.Getenv("HOME"),
	}

	// Handle potential passphrase by using ssh-add with timeout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load key %s: %w", keyPath, err)
	}

	// Update loaded keys
	m.mutex.Lock()
	agent.LoadedKeys = append(agent.LoadedKeys, keyPath)
	agent.LastUsed = time.Now()
	m.mutex.Unlock()

	m.logger.Info(ctx, "ssh_key_loaded_into_isolated_agent",
		observability.F("account", agent.AccountAlias),
		observability.F("key_path", keyPath),
		observability.F("loaded_keys_count", len(agent.LoadedKeys)),
	)

	return nil
}

// setSSHEnvironment sets environment variables for SSH isolation
func (m *IsolatedSSHManager) setSSHEnvironment(ctx context.Context, agent *SSHAgentProcess) error {
	// Set SSH_AUTH_SOCK for current process
	if err := os.Setenv("SSH_AUTH_SOCK", agent.SocketPath); err != nil {
		return fmt.Errorf("failed to set SSH_AUTH_SOCK: %w", err)
	}

	// Unset SSH_AGENT_PID to avoid conflicts
	if err := os.Unsetenv("SSH_AGENT_PID"); err != nil {
		m.logger.Warn(ctx, "failed_to_unset_ssh_agent_pid",
			observability.F("error", err.Error()),
		)
	}

	m.logger.Info(ctx, "ssh_environment_set_for_isolation",
		observability.F("account", agent.AccountAlias),
		observability.F("socket_path", agent.SocketPath),
	)

	return nil
}

// generateSSHConfig generates SSH config for account isolation
func (m *IsolatedSSHManager) generateSSHConfig(ctx context.Context, accountAlias, keyPath string) error {
	configPath := filepath.Join(m.defaultSSHDir, "config")

	// Create backup of existing config
	if err := m.backupSSHConfig(ctx); err != nil {
		m.logger.Warn(ctx, "failed_to_backup_ssh_config",
			observability.F("error", err.Error()),
		)
	}

	// Generate isolated SSH config section
	configSection := fmt.Sprintf(`
# GitPersona Isolated Config for %s
Host github-%s
    HostName github.com
    User git
    IdentityFile %s
    IdentitiesOnly yes
    AddKeysToAgent no
    UseKeychain no
    StrictHostKeyChecking yes
    UserKnownHostsFile %s

# Default GitHub (using current account: %s)
Host github.com
    HostName github.com
    User git
    IdentityFile %s
    IdentitiesOnly yes
    AddKeysToAgent no
    UseKeychain no
    StrictHostKeyChecking yes

`, accountAlias, accountAlias, keyPath,
		filepath.Join(m.defaultSSHDir, "known_hosts"),
		accountAlias, keyPath)

	// Read existing config
	var existingConfig string
	if data, err := os.ReadFile(configPath); err == nil {
		existingConfig = string(data)
		// Remove any existing GitPersona sections
		existingConfig = m.removeExistingGitPersonaSections(existingConfig)
	}

	// Write new config
	newConfig := configSection + existingConfig
	if err := os.WriteFile(configPath, []byte(newConfig), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	m.logger.Info(ctx, "ssh_config_generated_for_isolation",
		observability.F("account", accountAlias),
		observability.F("config_path", configPath),
	)

	return nil
}

// testSSHConnectivity tests SSH connection with the isolated agent
func (m *IsolatedSSHManager) testSSHConnectivity(ctx context.Context, accountAlias string) error {
	m.mutex.RLock()
	agent, exists := m.agentSockets[accountAlias]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no SSH agent found for account: %s", accountAlias)
	}

	// Test SSH connection
	cmd := exec.CommandContext(ctx, "ssh", "-T", "git@github.com",
		"-i", agent.KeyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10")

	cmd.Env = []string{
		"SSH_AUTH_SOCK=" + agent.SocketPath,
		"HOME=" + os.Getenv("HOME"),
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// SSH returns exit code 1 for successful authentication with GitHub
	if err != nil && !strings.Contains(outputStr, "successfully authenticated") {
		return fmt.Errorf("SSH connectivity test failed: %s", outputStr)
	}

	m.logger.Info(ctx, "ssh_connectivity_test_passed",
		observability.F("account", accountAlias),
		observability.F("response", outputStr),
	)

	return nil
}

// stopAccountAgent stops the SSH agent for a specific account
func (m *IsolatedSSHManager) stopAccountAgent(ctx context.Context, accountAlias string) error {
	m.mutex.Lock()
	agent, exists := m.agentSockets[accountAlias]
	if !exists {
		m.mutex.Unlock()
		return nil // No agent to stop
	}
	delete(m.agentSockets, accountAlias)
	m.mutex.Unlock()

	// Kill the agent process
	if agent.PID > 0 {
		if err := syscall.Kill(agent.PID, syscall.SIGTERM); err != nil {
			// Try SIGKILL if SIGTERM fails
			if killErr := syscall.Kill(agent.PID, syscall.SIGKILL); killErr != nil {
				m.logger.Warn(ctx, "failed_to_kill_ssh_agent",
					observability.F("account", accountAlias),
					observability.F("pid", agent.PID),
					observability.F("error", killErr.Error()),
				)
			}
		}
	}

	// Remove socket file
	if agent.SocketPath != "" {
		if err := os.Remove(agent.SocketPath); err != nil && !os.IsNotExist(err) {
			m.logger.Warn(ctx, "failed_to_remove_socket_file",
				observability.F("account", accountAlias),
				observability.F("socket_path", agent.SocketPath),
				observability.F("error", err.Error()),
			)
		}
	}

	m.logger.Info(ctx, "ssh_agent_stopped_for_account",
		observability.F("account", accountAlias),
		observability.F("pid", agent.PID),
	)

	return nil
}

// GetAccountAgent returns the SSH agent process for an account
func (m *IsolatedSSHManager) GetAccountAgent(accountAlias string) (*SSHAgentProcess, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agent, exists := m.agentSockets[accountAlias]
	if !exists {
		return nil, fmt.Errorf("no SSH agent found for account: %s", accountAlias)
	}

	// Return a copy to prevent external modification
	return &SSHAgentProcess{
		AccountAlias: agent.AccountAlias,
		PID:          agent.PID,
		SocketPath:   agent.SocketPath,
		KeyPath:      agent.KeyPath,
		StartTime:    agent.StartTime,
		LastUsed:     agent.LastUsed,
		IsRunning:    agent.IsRunning,
		LoadedKeys:   append([]string{}, agent.LoadedKeys...),
	}, nil
}

// ListActiveAgents returns all active SSH agents
func (m *IsolatedSSHManager) ListActiveAgents(ctx context.Context) ([]*SSHAgentProcess, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agents := make([]*SSHAgentProcess, 0, len(m.agentSockets))
	for _, agent := range m.agentSockets {
		// Check if process is still running
		if err := syscall.Kill(agent.PID, 0); err != nil {
			agent.IsRunning = false
		}

		agents = append(agents, &SSHAgentProcess{
			AccountAlias: agent.AccountAlias,
			PID:          agent.PID,
			SocketPath:   agent.SocketPath,
			KeyPath:      agent.KeyPath,
			StartTime:    agent.StartTime,
			LastUsed:     agent.LastUsed,
			IsRunning:    agent.IsRunning,
			LoadedKeys:   append([]string{}, agent.LoadedKeys...),
		})
	}

	m.logger.Info(ctx, "active_ssh_agents_listed",
		observability.F("count", len(agents)),
	)

	return agents, nil
}

// CleanupAllAgents stops all SSH agents and cleans up resources
func (m *IsolatedSSHManager) CleanupAllAgents(ctx context.Context) error {
	m.logger.Info(ctx, "cleaning_up_all_ssh_agents")

	m.mutex.Lock()
	accounts := make([]string, 0, len(m.agentSockets))
	for account := range m.agentSockets {
		accounts = append(accounts, account)
	}
	m.mutex.Unlock()

	var lastError error
	for _, account := range accounts {
		if err := m.stopAccountAgent(ctx, account); err != nil {
			lastError = err
			m.logger.Warn(ctx, "failed_to_stop_agent_during_cleanup",
				observability.F("account", account),
				observability.F("error", err.Error()),
			)
		}
	}

	// Clean up orphaned sockets
	if err := m.cleanupOrphanedSockets(ctx); err != nil {
		lastError = err
	}

	m.logger.Info(ctx, "ssh_agents_cleanup_completed",
		observability.F("cleaned_accounts", len(accounts)),
	)

	return lastError
}

// cleanupOrphanedSockets removes socket files without running processes
func (m *IsolatedSSHManager) cleanupOrphanedSockets(ctx context.Context) error {
	files, err := os.ReadDir(m.socketDir)
	if err != nil {
		return fmt.Errorf("failed to read socket directory: %w", err)
	}

	cleaned := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "ssh-agent-") {
			continue
		}

		socketPath := filepath.Join(m.socketDir, file.Name())

		// Try to connect to socket to see if it's alive
		if _, err := os.Stat(socketPath); err == nil {
			// If we can't find a corresponding running process, remove it
			if err := os.Remove(socketPath); err != nil {
				m.logger.Warn(ctx, "failed_to_remove_orphaned_socket",
					observability.F("socket_path", socketPath),
					observability.F("error", err.Error()),
				)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		m.logger.Info(ctx, "orphaned_sockets_cleaned",
			observability.F("count", cleaned),
		)
	}

	return nil
}

// backupSSHConfig creates a backup of the SSH config
func (m *IsolatedSSHManager) backupSSHConfig(ctx context.Context) error {
	configPath := filepath.Join(m.defaultSSHDir, "config")
	backupPath := configPath + ".gitpersona.backup"

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // No config to backup
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write SSH config backup: %w", err)
	}

	return nil
}

// removeExistingGitPersonaSections removes any existing GitPersona sections from SSH config
func (m *IsolatedSSHManager) removeExistingGitPersonaSections(config string) string {
	lines := strings.Split(config, "\n")
	var result []string
	inGitPersonaSection := false

	for _, line := range lines {
		if strings.Contains(line, "# GitPersona Isolated Config") {
			inGitPersonaSection = true
			continue
		}

		if inGitPersonaSection && strings.HasPrefix(line, "# ") &&
			!strings.Contains(line, "GitPersona") {
			inGitPersonaSection = false
		}

		if inGitPersonaSection && (strings.HasPrefix(line, "Host ") ||
			strings.HasPrefix(line, "    ")) {
			continue
		}

		if !inGitPersonaSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
