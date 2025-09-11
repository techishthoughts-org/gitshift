package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealSSHAgentService implements SSHAgentService
type RealSSHAgentService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewRealSSHAgentService creates a new SSH agent service
func NewRealSSHAgentService(logger observability.Logger, runner execrunner.CmdRunner) *RealSSHAgentService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealSSHAgentService{
		logger: logger,
		runner: runner,
	}
}

// IsAgentRunning checks if the SSH agent is running
func (s *RealSSHAgentService) IsAgentRunning(ctx context.Context) (bool, error) {
	s.logger.Info(ctx, "checking_ssh_agent_status")

	cmd := exec.Command("ssh-add", "-l")
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	if err != nil {
		// SSH agent not running or no keys loaded
		s.logger.Info(ctx, "ssh_agent_not_running_or_no_keys")
		return false, nil
	}

	// Check if output indicates agent is running
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "The agent has no identities." {
		s.logger.Info(ctx, "ssh_agent_running_no_keys")
		return true, nil
	}

	s.logger.Info(ctx, "ssh_agent_running_with_keys",
		observability.F("output", outputStr),
	)
	return true, nil
}

// StartAgent starts the SSH agent
func (s *RealSSHAgentService) StartAgent(ctx context.Context) error {
	s.logger.Info(ctx, "starting_ssh_agent")

	// Check if agent is already running
	if running, _ := s.IsAgentRunning(ctx); running {
		s.logger.Info(ctx, "ssh_agent_already_running")
		return nil
	}

	// Start SSH agent
	cmd := exec.Command("ssh-agent", "-s")
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		s.logger.Error(ctx, "failed_to_start_ssh_agent",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to start SSH agent: %w", err)
	}

	// Parse agent output to get environment variables
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "SSH_AUTH_SOCK=") {
			s.logger.Info(ctx, "ssh_agent_started_successfully",
				observability.F("socket", line),
			)
			break
		}
	}

	return nil
}

// StopAgent stops the SSH agent
func (s *RealSSHAgentService) StopAgent(ctx context.Context) error {
	s.logger.Info(ctx, "stopping_ssh_agent")

	// Clear all keys first
	if err := s.ClearAllKeys(ctx); err != nil {
		s.logger.Warn(ctx, "failed_to_clear_keys_before_stop",
			observability.F("error", err.Error()),
		)
	}

	// Kill SSH agent
	cmd := exec.Command("ssh-agent", "-k")
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		s.logger.Error(ctx, "failed_to_stop_ssh_agent",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to stop SSH agent: %w", err)
	}

	s.logger.Info(ctx, "ssh_agent_stopped_successfully")
	return nil
}

// LoadKey loads a specific SSH key into the agent
func (s *RealSSHAgentService) LoadKey(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "loading_ssh_key",
		observability.F("key_path", keyPath),
	)

	// Ensure agent is running
	if err := s.StartAgent(ctx); err != nil {
		return fmt.Errorf("failed to start SSH agent: %w", err)
	}

	// Add key to agent
	cmd := exec.Command("ssh-add", keyPath)
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		s.logger.Error(ctx, "failed_to_load_ssh_key",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to load SSH key: %w", err)
	}

	s.logger.Info(ctx, "ssh_key_loaded_successfully",
		observability.F("key_path", keyPath),
	)
	return nil
}

// UnloadKey removes a specific SSH key from the agent
func (s *RealSSHAgentService) UnloadKey(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "unloading_ssh_key",
		observability.F("key_path", keyPath),
	)

	// Remove key from agent
	cmd := exec.Command("ssh-add", "-d", keyPath)
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		s.logger.Warn(ctx, "failed_to_unload_ssh_key",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
		// Don't return error as key might not be loaded
	}

	s.logger.Info(ctx, "ssh_key_unloaded_successfully",
		observability.F("key_path", keyPath),
	)
	return nil
}

// ClearAllKeys removes all keys from the SSH agent
func (s *RealSSHAgentService) ClearAllKeys(ctx context.Context) error {
	s.logger.Info(ctx, "clearing_all_ssh_keys")

	cmd := exec.Command("ssh-add", "-D")
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		s.logger.Error(ctx, "failed_to_clear_ssh_keys",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to clear SSH keys: %w", err)
	}

	s.logger.Info(ctx, "all_ssh_keys_cleared_successfully")
	return nil
}

// ListLoadedKeys returns a list of currently loaded SSH keys
func (s *RealSSHAgentService) ListLoadedKeys(ctx context.Context) ([]string, error) {
	s.logger.Info(ctx, "listing_loaded_ssh_keys")

	cmd := exec.Command("ssh-add", "-l")
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		s.logger.Info(ctx, "no_ssh_keys_loaded")
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	keys := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "The agent has no identities") {
			// Extract key path from fingerprint line
			// Format: "256 SHA256:abc123... user@host (ED25519)"
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				keys = append(keys, line)
			}
		}
	}

	s.logger.Info(ctx, "loaded_ssh_keys_listed",
		observability.F("count", len(keys)),
	)
	return keys, nil
}

// SwitchToAccount switches to a specific account by loading only its key
func (s *RealSSHAgentService) SwitchToAccount(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "switching_to_account",
		observability.F("key_path", keyPath),
	)

	// Clear all existing keys
	if err := s.ClearAllKeys(ctx); err != nil {
		s.logger.Warn(ctx, "failed_to_clear_keys_during_switch",
			observability.F("error", err.Error()),
		)
	}

	// Load the target key
	if err := s.LoadKey(ctx, keyPath); err != nil {
		return fmt.Errorf("failed to load account key: %w", err)
	}

	// Verify only the target key is loaded
	loadedKeys, err := s.ListLoadedKeys(ctx)
	if err != nil {
		s.logger.Warn(ctx, "failed_to_verify_key_switch",
			observability.F("error", err.Error()),
		)
	} else if len(loadedKeys) != 1 {
		s.logger.Warn(ctx, "multiple_keys_loaded_after_switch",
			observability.F("key_count", len(loadedKeys)),
		)
	}

	s.logger.Info(ctx, "account_switch_completed",
		observability.F("key_path", keyPath),
	)
	return nil
}

// IsolateAccount ensures only the specified key is loaded
func (s *RealSSHAgentService) IsolateAccount(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "isolating_account",
		observability.F("key_path", keyPath),
	)

	// Get currently loaded keys
	loadedKeys, err := s.ListLoadedKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list loaded keys: %w", err)
	}

	// Check if target key is already loaded
	targetLoaded := false
	for _, key := range loadedKeys {
		if strings.Contains(key, keyPath) {
			targetLoaded = true
			break
		}
	}

	// If target key is not loaded or other keys are present, switch
	if !targetLoaded || len(loadedKeys) > 1 {
		return s.SwitchToAccount(ctx, keyPath)
	}

	s.logger.Info(ctx, "account_already_isolated",
		observability.F("key_path", keyPath),
	)
	return nil
}

// GetAgentStatus returns the current status of the SSH agent
func (s *RealSSHAgentService) GetAgentStatus(ctx context.Context) (*SSHAgentStatus, error) {
	s.logger.Info(ctx, "getting_ssh_agent_status")

	status := &SSHAgentStatus{
		LastUpdated: time.Now(),
	}

	// Check if agent is running
	running, err := s.IsAgentRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent status: %w", err)
	}
	status.Running = running

	if running {
		// Get loaded keys
		keys, err := s.ListLoadedKeys(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list loaded keys: %w", err)
		}
		status.LoadedKeys = keys
		status.KeyCount = len(keys)
	}

	s.logger.Info(ctx, "ssh_agent_status_retrieved",
		observability.F("running", status.Running),
		observability.F("key_count", status.KeyCount),
	)
	return status, nil
}

// DiagnoseAgentIssues diagnoses common SSH agent issues
func (s *RealSSHAgentService) DiagnoseAgentIssues(ctx context.Context) ([]string, error) {
	s.logger.Info(ctx, "diagnosing_ssh_agent_issues")

	issues := []string{}

	// Check if agent is running
	running, err := s.IsAgentRunning(ctx)
	if err != nil {
		issues = append(issues, "Failed to check SSH agent status")
		return issues, nil
	}

	if !running {
		issues = append(issues, "SSH agent is not running")
		return issues, nil
	}

	// Check loaded keys
	keys, err := s.ListLoadedKeys(ctx)
	if err != nil {
		issues = append(issues, "Failed to list loaded SSH keys")
		return issues, nil
	}

	if len(keys) == 0 {
		issues = append(issues, "No SSH keys loaded in agent")
	} else if len(keys) > 1 {
		issues = append(issues, fmt.Sprintf("Multiple SSH keys loaded (%d), may cause authentication conflicts", len(keys)))
	}

	// Check for common issues
	for _, key := range keys {
		if strings.Contains(key, "Permission denied") {
			issues = append(issues, "SSH key has permission issues")
		}
	}

	s.logger.Info(ctx, "ssh_agent_diagnosis_completed",
		observability.F("issues_count", len(issues)),
	)
	return issues, nil
}

// CleanupSSHSockets cleans up SSH control sockets to prevent authentication conflicts
func (s *RealSSHAgentService) CleanupSSHSockets(ctx context.Context) error {
	s.logger.Info(ctx, "cleaning_up_ssh_sockets")

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.logger.Warn(ctx, "failed_to_get_home_directory",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Ensure SSH directory exists
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		s.logger.Warn(ctx, "failed_to_create_ssh_directory",
			observability.F("path", sshDir),
			observability.F("error", err.Error()),
		)
	}

	// Common SSH socket locations
	socketPaths := []string{
		filepath.Join(homeDir, ".ssh", "socket"),
		filepath.Join(homeDir, ".ssh", "sockets"),
		filepath.Join(homeDir, ".ssh", "control"),
		"/tmp/ssh-*",
	}

	// Ensure socket directories exist after cleanup
	socketDirs := []string{
		filepath.Join(homeDir, ".ssh", "socket"),
		filepath.Join(homeDir, ".ssh", "sockets"),
		filepath.Join(homeDir, ".ssh", "control"),
	}

	cleanedCount := 0
	for _, socketPath := range socketPaths {
		if strings.Contains(socketPath, "*") {
			// Handle glob patterns
			matches, err := filepath.Glob(socketPath)
			if err != nil {
				s.logger.Warn(ctx, "failed_to_glob_socket_path",
					observability.F("path", socketPath),
					observability.F("error", err.Error()),
				)
				continue
			}
			for _, match := range matches {
				if err := s.cleanupSocketFile(ctx, match); err == nil {
					cleanedCount++
				}
			}
		} else {
			// Handle specific paths
			if err := s.cleanupSocketFile(ctx, socketPath); err == nil {
				cleanedCount++
			}
		}
	}

	// Also try to close existing SSH connections
	if err := s.closeExistingSSHConnections(ctx); err != nil {
		s.logger.Warn(ctx, "failed_to_close_ssh_connections",
			observability.F("error", err.Error()),
		)
	}

	// Ensure socket directories exist after cleanup
	s.logger.Info(ctx, "creating_socket_directories",
		observability.F("socket_dirs", socketDirs),
	)
	for _, socketDir := range socketDirs {
		s.logger.Info(ctx, "creating_socket_directory",
			observability.F("path", socketDir),
		)
		if err := os.MkdirAll(socketDir, 0700); err != nil {
			s.logger.Warn(ctx, "failed_to_create_socket_directory",
				observability.F("path", socketDir),
				observability.F("error", err.Error()),
			)
		} else {
			s.logger.Info(ctx, "socket_directory_ensured",
				observability.F("path", socketDir),
			)
		}
	}

	s.logger.Info(ctx, "ssh_socket_cleanup_completed",
		observability.F("cleaned_count", cleanedCount),
	)
	return nil
}

// cleanupSocketFile removes a specific socket file or directory
func (s *RealSSHAgentService) cleanupSocketFile(ctx context.Context, socketPath string) error {
	// Check if path exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil // Path doesn't exist, nothing to clean
	}

	// Try to remove the socket file/directory
	if err := os.RemoveAll(socketPath); err != nil {
		s.logger.Warn(ctx, "failed_to_remove_socket",
			observability.F("path", socketPath),
			observability.F("error", err.Error()),
		)
		return err
	}

	s.logger.Info(ctx, "socket_cleaned",
		observability.F("path", socketPath),
	)
	return nil
}

// closeExistingSSHConnections attempts to close existing SSH connections
func (s *RealSSHAgentService) closeExistingSSHConnections(ctx context.Context) error {
	// Try to close SSH connections using ssh -O exit
	// This is a best-effort attempt and may not work for all connections
	cmd := exec.Command("ssh", "-O", "exit", "github.com")
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		// This is expected to fail if no connection exists, so we don't log as error
		s.logger.Debug(ctx, "no_ssh_connection_to_close")
	}

	// Try to close connections to common Git hosts
	hosts := []string{"github.com", "gitlab.com", "bitbucket.org"}
	for _, host := range hosts {
		cmd := exec.Command("ssh", "-O", "exit", host)
		if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
			s.logger.Debug(ctx, "no_ssh_connection_to_close",
				observability.F("host", host),
			)
		}
	}

	return nil
}

// SwitchToAccountWithCleanup switches to a specific account with socket cleanup
func (s *RealSSHAgentService) SwitchToAccountWithCleanup(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "switching_to_account_with_cleanup",
		observability.F("key_path", keyPath),
	)

	// First, clean up SSH sockets to prevent conflicts
	if err := s.CleanupSSHSockets(ctx); err != nil {
		s.logger.Warn(ctx, "failed_to_cleanup_ssh_sockets",
			observability.F("error", err.Error()),
		)
		// Continue with the switch even if cleanup fails
	}

	// Then perform the normal account switch
	return s.SwitchToAccount(ctx, keyPath)
}

// ValidateSSHConnectionWithRetry validates SSH connection with retry mechanism
func (s *RealSSHAgentService) ValidateSSHConnectionWithRetry(ctx context.Context, keyPath string) error {
	maxRetries := 3
	retryDelay := time.Second * 2

	for i := 0; i < maxRetries; i++ {
		if err := s.testSSHConnection(ctx, keyPath); err == nil {
			s.logger.Info(ctx, "ssh_validation_successful",
				observability.F("attempt", i+1),
				observability.F("key_path", keyPath),
			)
			return nil
		}

		if i < maxRetries-1 {
			s.logger.Info(ctx, "ssh_validation_retry",
				observability.F("attempt", i+1),
				observability.F("max_retries", maxRetries),
				observability.F("retry_delay", retryDelay.String()),
			)
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("SSH validation failed after %d attempts", maxRetries)
}

// testSSHConnection tests SSH connection to GitHub
func (s *RealSSHAgentService) testSSHConnection(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "testing_ssh_connection",
		observability.F("key_path", keyPath),
	)

	// Ensure SSH socket directories exist before testing
	if err := s.ensureSSHSocketDirectories(ctx); err != nil {
		s.logger.Warn(ctx, "failed_to_ensure_ssh_socket_directories",
			observability.F("error", err.Error()),
		)
		// Continue with the test even if socket directory creation fails
	}

	// Test SSH connection to GitHub
	cmd := exec.Command("ssh", "-i", keyPath, "-T", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=10", "git@github.com")
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)

	// Check if the output indicates successful authentication first
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Hi ") {
		s.logger.Info(ctx, "ssh_connection_test_successful",
			observability.F("key_path", keyPath),
			observability.F("output", outputStr),
		)
		return nil
	}

	// If we get here, there was either an error or unexpected output
	if err != nil {
		s.logger.Warn(ctx, "ssh_connection_test_failed",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
			observability.F("output", string(output)),
		)
		return fmt.Errorf("SSH connection test failed: %w", err)
	}

	s.logger.Warn(ctx, "ssh_connection_test_unexpected_output",
		observability.F("key_path", keyPath),
		observability.F("output", outputStr),
	)
	return fmt.Errorf("unexpected SSH connection output: %s", outputStr)
}

// ensureSSHSocketDirectories ensures that SSH socket directories exist
func (s *RealSSHAgentService) ensureSSHSocketDirectories(ctx context.Context) error {
	s.logger.Info(ctx, "ensuring_ssh_socket_directories")

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Common SSH socket directories
	socketDirs := []string{
		filepath.Join(homeDir, ".ssh", "socket"),
		filepath.Join(homeDir, ".ssh", "sockets"),
		filepath.Join(homeDir, ".ssh", "control"),
	}

	// Ensure each socket directory exists
	for _, socketDir := range socketDirs {
		if err := os.MkdirAll(socketDir, 0700); err != nil {
			s.logger.Warn(ctx, "failed_to_create_socket_directory",
				observability.F("path", socketDir),
				observability.F("error", err.Error()),
			)
			return fmt.Errorf("failed to create socket directory %s: %w", socketDir, err)
		}
		s.logger.Info(ctx, "socket_directory_ensured",
			observability.F("path", socketDir),
		)
	}

	s.logger.Info(ctx, "ssh_socket_directories_ensured")
	return nil
}
