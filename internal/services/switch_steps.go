package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHIsolationStep handles SSH isolation during account switching
type SSHIsolationStep struct {
	logger observability.Logger
}

func NewSSHIsolationStep(logger observability.Logger) *SSHIsolationStep {
	return &SSHIsolationStep{logger: logger}
}

func (s *SSHIsolationStep) GetName() string {
	return "ssh_isolation"
}

func (s *SSHIsolationStep) GetDescription() string {
	return "Isolate SSH agent and load account-specific SSH key"
}

func (s *SSHIsolationStep) GetDependencies() []string {
	return []string{} // No dependencies
}

func (s *SSHIsolationStep) CanRollback() bool {
	return true
}

func (s *SSHIsolationStep) Validate(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()

	// Check if SSH key exists
	if targetAccount.SSHKeyPath == "" {
		return fmt.Errorf("no SSH key path configured for account: %s", targetAccount.Alias)
	}

	if _, err := os.Stat(targetAccount.SSHKeyPath); err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", targetAccount.SSHKeyPath, err)
	}

	return nil
}

func (s *SSHIsolationStep) Execute(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()
	sshManager := transaction.GetSSHManager()

	s.logger.Info(ctx, "executing_ssh_isolation_step",
		observability.F("account", targetAccount.Alias),
		observability.F("key_path", targetAccount.SSHKeyPath),
	)

	// Switch to isolated SSH for this account
	return sshManager.SwitchToAccount(ctx, targetAccount.Alias, targetAccount.SSHKeyPath)
}

func (s *SSHIsolationStep) Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error {
	sourceAccount := transaction.GetSourceAccount()
	sshManager := transaction.GetSSHManager()

	s.logger.Info(ctx, "rolling_back_ssh_isolation_step")

	if sourceAccount != nil && sourceAccount.SSHKeyPath != "" {
		// Restore previous SSH configuration
		return sshManager.SwitchToAccount(ctx, sourceAccount.Alias, sourceAccount.SSHKeyPath)
	}

	// Clean up current SSH agent if no source account
	return sshManager.CleanupAllAgents(ctx)
}

// TokenIsolationStep handles token isolation during account switching
type TokenIsolationStep struct {
	logger observability.Logger
}

func NewTokenIsolationStep(logger observability.Logger) *TokenIsolationStep {
	return &TokenIsolationStep{logger: logger}
}

func (s *TokenIsolationStep) GetName() string {
	return "token_isolation"
}

func (s *TokenIsolationStep) GetDescription() string {
	return "Ensure account has isolated GitHub token"
}

func (s *TokenIsolationStep) GetDependencies() []string {
	return []string{} // No dependencies
}

func (s *TokenIsolationStep) CanRollback() bool {
	return false // Token validation doesn't change state
}

func (s *TokenIsolationStep) Validate(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()
	tokenService := transaction.GetTokenService()

	if tokenService == nil {
		return fmt.Errorf("token service not available")
	}

	// Check if account has a token
	_, err := tokenService.GetToken(ctx, targetAccount.Alias)
	if err != nil {
		return fmt.Errorf("no token found for account '%s': %w", targetAccount.Alias, err)
	}

	// Validate token belongs to correct user
	return tokenService.ValidateTokenIsolation(ctx, targetAccount.Alias, targetAccount.GitHubUsername)
}

func (s *TokenIsolationStep) Execute(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()
	tokenService := transaction.GetTokenService()

	s.logger.Info(ctx, "executing_token_isolation_step",
		observability.F("account", targetAccount.Alias),
		observability.F("username", targetAccount.GitHubUsername),
	)

	// Validate token isolation
	return tokenService.ValidateTokenIsolation(ctx, targetAccount.Alias, targetAccount.GitHubUsername)
}

func (s *TokenIsolationStep) Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error {
	// Token validation is stateless, nothing to rollback
	return nil
}

// GitConfigurationStep handles Git configuration during account switching
type GitConfigurationStep struct {
	logger observability.Logger
}

func NewGitConfigurationStep(logger observability.Logger) *GitConfigurationStep {
	return &GitConfigurationStep{logger: logger}
}

func (s *GitConfigurationStep) GetName() string {
	return "git_configuration"
}

func (s *GitConfigurationStep) GetDescription() string {
	return "Update Git user configuration"
}

func (s *GitConfigurationStep) GetDependencies() []string {
	return []string{"ssh_isolation"} // Depends on SSH being configured first
}

func (s *GitConfigurationStep) CanRollback() bool {
	return true
}

func (s *GitConfigurationStep) Validate(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()

	if targetAccount.Name == "" {
		return fmt.Errorf("no name configured for account: %s", targetAccount.Alias)
	}

	if targetAccount.Email == "" {
		return fmt.Errorf("no email configured for account: %s", targetAccount.Alias)
	}

	return nil
}

func (s *GitConfigurationStep) Execute(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()

	s.logger.Info(ctx, "executing_git_configuration_step",
		observability.F("account", targetAccount.Alias),
		observability.F("name", targetAccount.Name),
		observability.F("email", targetAccount.Email),
	)

	// Set global Git configuration
	if err := s.setGitConfig("user.name", targetAccount.Name, true); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	if err := s.setGitConfig("user.email", targetAccount.Email, true); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	// Set SSH command for isolation if SSH key is configured
	if targetAccount.SSHKeyPath != "" {
		sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", targetAccount.SSHKeyPath)
		if err := s.setGitConfig("core.sshCommand", sshCommand, true); err != nil {
			return fmt.Errorf("failed to set git core.sshCommand: %w", err)
		}
	}

	// Set local configuration if in a Git repository
	if s.isGitRepo() {
		if err := s.setGitConfig("user.name", targetAccount.Name, false); err != nil {
			s.logger.Warn(ctx, "failed_to_set_local_git_user_name",
				observability.F("error", err.Error()),
			)
		}

		if err := s.setGitConfig("user.email", targetAccount.Email, false); err != nil {
			s.logger.Warn(ctx, "failed_to_set_local_git_user_email",
				observability.F("error", err.Error()),
			)
		}
	}

	return nil
}

func (s *GitConfigurationStep) Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error {
	sourceAccount := transaction.GetSourceAccount()

	s.logger.Info(ctx, "rolling_back_git_configuration_step")

	if sourceAccount != nil {
		// Restore previous Git configuration
		if sourceAccount.Name != "" {
			if err := s.setGitConfig("user.name", sourceAccount.Name, true); err != nil {
				return fmt.Errorf("failed to rollback git user.name: %w", err)
			}
		}

		if sourceAccount.Email != "" {
			if err := s.setGitConfig("user.email", sourceAccount.Email, true); err != nil {
				return fmt.Errorf("failed to rollback git user.email: %w", err)
			}
		}

		if sourceAccount.SSHKeyPath != "" {
			sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", sourceAccount.SSHKeyPath)
			if err := s.setGitConfig("core.sshCommand", sshCommand, true); err != nil {
				return fmt.Errorf("failed to rollback git core.sshCommand: %w", err)
			}
		}
	}

	return nil
}

func (s *GitConfigurationStep) setGitConfig(key, value string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, key, value)

	cmd := exec.Command("git", args...)
	return cmd.Run()
}

func (s *GitConfigurationStep) isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// EnvironmentStep handles environment variable configuration
type EnvironmentStep struct {
	logger      observability.Logger
	previousEnv map[string]string
}

func NewEnvironmentStep(logger observability.Logger) *EnvironmentStep {
	return &EnvironmentStep{
		logger:      logger,
		previousEnv: make(map[string]string),
	}
}

func (s *EnvironmentStep) GetName() string {
	return "environment_configuration"
}

func (s *EnvironmentStep) GetDescription() string {
	return "Set environment variables for account isolation"
}

func (s *EnvironmentStep) GetDependencies() []string {
	return []string{"ssh_isolation", "token_isolation"}
}

func (s *EnvironmentStep) CanRollback() bool {
	return true
}

func (s *EnvironmentStep) Validate(ctx context.Context, transaction *AccountSwitchTransaction) error {
	// No validation needed for environment step
	return nil
}

func (s *EnvironmentStep) Execute(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()
	sshManager := transaction.GetSSHManager()

	s.logger.Info(ctx, "executing_environment_step",
		observability.F("account", targetAccount.Alias),
	)

	// Get SSH agent socket for this account
	if agent, err := sshManager.GetAccountAgent(targetAccount.Alias); err == nil {
		// Store previous SSH_AUTH_SOCK for rollback
		if previousSocket := os.Getenv("SSH_AUTH_SOCK"); previousSocket != "" {
			s.previousEnv["SSH_AUTH_SOCK"] = previousSocket
		}

		// Set SSH_AUTH_SOCK to isolated agent socket
		if err := os.Setenv("SSH_AUTH_SOCK", agent.SocketPath); err != nil {
			return fmt.Errorf("failed to set SSH_AUTH_SOCK: %w", err)
		}

		s.logger.Info(ctx, "ssh_auth_sock_set_for_isolation",
			observability.F("account", targetAccount.Alias),
			observability.F("socket_path", agent.SocketPath),
		)
	}

	// Unset SSH_AGENT_PID to avoid conflicts
	if previousPID := os.Getenv("SSH_AGENT_PID"); previousPID != "" {
		s.previousEnv["SSH_AGENT_PID"] = previousPID
		if err := os.Unsetenv("SSH_AGENT_PID"); err != nil {
			s.logger.Warn(ctx, "failed_to_unset_ssh_agent_pid",
				observability.F("error", err.Error()),
			)
		}
	}

	return nil
}

func (s *EnvironmentStep) Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error {
	s.logger.Info(ctx, "rolling_back_environment_step")

	// Restore previous environment variables
	for key, value := range s.previousEnv {
		if err := os.Setenv(key, value); err != nil {
			s.logger.Warn(ctx, "failed_to_restore_environment_variable",
				observability.F("key", key),
				observability.F("error", err.Error()),
			)
		}
	}

	return nil
}

// ValidationStep performs comprehensive validation after account switch
type ValidationStep struct {
	logger observability.Logger
}

func NewValidationStep(logger observability.Logger) *ValidationStep {
	return &ValidationStep{logger: logger}
}

func (s *ValidationStep) GetName() string {
	return "comprehensive_validation"
}

func (s *ValidationStep) GetDescription() string {
	return "Validate account switch was successful"
}

func (s *ValidationStep) GetDependencies() []string {
	return []string{"ssh_isolation", "token_isolation", "git_configuration", "environment_configuration"}
}

func (s *ValidationStep) CanRollback() bool {
	return false // Validation doesn't change state
}

func (s *ValidationStep) Validate(ctx context.Context, transaction *AccountSwitchTransaction) error {
	// This is the validation step itself
	return nil
}

func (s *ValidationStep) Execute(ctx context.Context, transaction *AccountSwitchTransaction) error {
	targetAccount := transaction.GetTargetAccount()

	s.logger.Info(ctx, "executing_comprehensive_validation_step",
		observability.F("account", targetAccount.Alias),
	)

	// Validate Git configuration
	if err := s.validateGitConfiguration(ctx, targetAccount); err != nil {
		return fmt.Errorf("git configuration validation failed: %w", err)
	}

	// Validate SSH connectivity (if SSH key is configured)
	if targetAccount.SSHKeyPath != "" {
		if err := s.validateSSHConnectivity(ctx, targetAccount); err != nil {
			return fmt.Errorf("SSH connectivity validation failed: %w", err)
		}
	}

	// Validate environment variables
	if err := s.validateEnvironment(ctx, targetAccount); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	s.logger.Info(ctx, "comprehensive_validation_completed_successfully",
		observability.F("account", targetAccount.Alias),
	)

	return nil
}

func (s *ValidationStep) Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error {
	// Validation is stateless, nothing to rollback
	return nil
}

func (s *ValidationStep) validateGitConfiguration(ctx context.Context, account *models.Account) error {
	// Check git user.name
	nameCmd := exec.CommandContext(ctx, "git", "config", "--global", "user.name")
	nameOutput, err := nameCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.name: %w", err)
	}

	if strings.TrimSpace(string(nameOutput)) != account.Name {
		return fmt.Errorf("git user.name mismatch: expected '%s', got '%s'",
			account.Name, strings.TrimSpace(string(nameOutput)))
	}

	// Check git user.email
	emailCmd := exec.CommandContext(ctx, "git", "config", "--global", "user.email")
	emailOutput, err := emailCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git user.email: %w", err)
	}

	if strings.TrimSpace(string(emailOutput)) != account.Email {
		return fmt.Errorf("git user.email mismatch: expected '%s', got '%s'",
			account.Email, strings.TrimSpace(string(emailOutput)))
	}

	return nil
}

func (s *ValidationStep) validateSSHConnectivity(ctx context.Context, account *models.Account) error {
	// Test SSH connection to GitHub
	cmd := exec.CommandContext(ctx, "ssh", "-T", "git@github.com",
		"-i", account.SSHKeyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// SSH returns exit code 1 for successful authentication with GitHub
	if err != nil && !strings.Contains(outputStr, "successfully authenticated") {
		return fmt.Errorf("SSH connectivity test failed: %s", outputStr)
	}

	// Check if authenticated as the correct user
	if account.GitHubUsername != "" && !strings.Contains(outputStr, account.GitHubUsername) {
		return fmt.Errorf("SSH authenticated as wrong user: expected '%s' in response: %s",
			account.GitHubUsername, outputStr)
	}

	return nil
}

func (s *ValidationStep) validateEnvironment(ctx context.Context, account *models.Account) error {
	// Check SSH_AUTH_SOCK is set
	if sshAuthSock := os.Getenv("SSH_AUTH_SOCK"); sshAuthSock == "" {
		return fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	// Check if SSH agent socket exists
	if sshAuthSock := os.Getenv("SSH_AUTH_SOCK"); sshAuthSock != "" {
		if _, err := os.Stat(sshAuthSock); err != nil {
			return fmt.Errorf("SSH agent socket not found: %s", sshAuthSock)
		}
	}

	return nil
}
