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

// RealSSHService implements the SSHService interface
type RealSSHService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewRealSSHService creates a new real SSH service
func NewRealSSHService(logger observability.Logger, runner execrunner.CmdRunner) *RealSSHService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealSSHService{
		logger: logger,
		runner: runner,
	}
}

// GenerateKey generates a new SSH key
func (s *RealSSHService) GenerateKey(ctx context.Context, keyType string, email string, keyPath string) (*SSHKey, error) {
	s.logger.Info(ctx, "generating_ssh_key",
		observability.F("key_type", keyType),
		observability.F("email", email),
		observability.F("key_path", keyPath),
	)

	// Validate key type
	if keyType != "ed25519" && keyType != "rsa" {
		return nil, fmt.Errorf("unsupported key type: %s (supported: ed25519, rsa)", keyType)
	}

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return nil, fmt.Errorf("SSH key already exists at %s", keyPath)
	}

	// Create directory if it doesn't exist
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate the key
	args := []string{"-t", keyType, "-C", email, "-f", keyPath, "-N", ""}
	if keyType == "rsa" {
		args = append(args, "-b", "4096")
	}

	cmd := exec.Command("ssh-keygen", args...)
	if err := s.runner.Run(ctx, cmd.Path, cmd.Args[1:]...); err != nil {
		s.logger.Error(ctx, "failed_to_generate_ssh_key",
			observability.F("key_type", keyType),
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// Get key information
	keyInfo, err := s.ValidateKey(ctx, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to validate generated key: %w", err)
	}

	key := &SSHKey{
		Path:        keyPath,
		Type:        keyType,
		Size:        keyInfo.Size,
		Fingerprint: keyInfo.Fingerprint,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// Read public key
	if publicKey, err := os.ReadFile(keyPath + ".pub"); err == nil {
		key.PublicKey = strings.TrimSpace(string(publicKey))
	}

	s.logger.Info(ctx, "ssh_key_generated_successfully",
		observability.F("key_type", keyType),
		observability.F("key_path", keyPath),
		observability.F("fingerprint", key.Fingerprint),
	)

	return key, nil
}

// ValidateKey validates an SSH key
func (s *RealSSHService) ValidateKey(ctx context.Context, keyPath string) (*SSHKeyInfo, error) {
	s.logger.Info(ctx, "validating_ssh_key",
		observability.F("key_path", keyPath),
	)

	keyInfo := &SSHKeyInfo{
		Path: keyPath,
	}

	// Check if file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		keyInfo.Exists = false
		return keyInfo, fmt.Errorf("SSH key file does not exist: %s", keyPath)
	}
	keyInfo.Exists = true

	// Check if file is readable
	if _, err := os.Open(keyPath); err != nil {
		keyInfo.Readable = false
		return keyInfo, fmt.Errorf("SSH key file is not readable: %s", keyPath)
	}
	keyInfo.Readable = true

	// Get key information using ssh-keygen
	cmd := exec.Command("ssh-keygen", "-l", "-f", keyPath)
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		keyInfo.Valid = false
		return keyInfo, fmt.Errorf("failed to get key information: %w", err)
	}

	// Parse key information
	// Output format: "2048 SHA256:abc123... user@host (RSA)"
	parts := strings.Fields(string(output))
	if len(parts) >= 4 {
		keyInfo.Fingerprint = parts[1]
		keyInfo.Type = strings.Trim(parts[3], "()")
		_, _ = fmt.Sscanf(parts[0], "%d", &keyInfo.Size)
	}

	// Extract email from public key
	if publicKeyPath := keyPath + ".pub"; publicKeyPath != "" {
		if publicKey, err := os.ReadFile(publicKeyPath); err == nil {
			keyInfo.Email = s.extractEmailFromKey(string(publicKey))
		}
	}

	keyInfo.Valid = true

	s.logger.Info(ctx, "ssh_key_validated_successfully",
		observability.F("key_path", keyPath),
		observability.F("type", keyInfo.Type),
		observability.F("size", keyInfo.Size),
		observability.F("fingerprint", keyInfo.Fingerprint),
	)

	return keyInfo, nil
}

// ListKeys lists all SSH keys in the SSH directory
func (s *RealSSHService) ListKeys(ctx context.Context) ([]*SSHKeyInfo, error) {
	s.logger.Info(ctx, "listing_ssh_keys")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	keys := []*SSHKeyInfo{}

	// Read SSH directory
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip public keys and known_hosts
		if strings.HasSuffix(entry.Name(), ".pub") || entry.Name() == "known_hosts" {
			continue
		}

		// Skip files that don't look like SSH keys
		if !strings.HasPrefix(entry.Name(), "id_") {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())
		keyInfo, err := s.ValidateKey(ctx, keyPath)
		if err != nil {
			s.logger.Warn(ctx, "failed_to_validate_key",
				observability.F("key_path", keyPath),
				observability.F("error", err.Error()),
			)
			continue
		}

		keys = append(keys, keyInfo)
	}

	s.logger.Info(ctx, "ssh_keys_listed_successfully",
		observability.F("count", len(keys)),
	)

	return keys, nil
}

// DeleteKey deletes an SSH key
func (s *RealSSHService) DeleteKey(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "deleting_ssh_key",
		observability.F("key_path", keyPath),
	)

	// Delete private key
	if err := os.Remove(keyPath); err != nil {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	// Delete public key
	publicKeyPath := keyPath + ".pub"
	if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	s.logger.Info(ctx, "ssh_key_deleted_successfully",
		observability.F("key_path", keyPath),
	)

	return nil
}

// ValidateConfiguration validates SSH configuration
func (s *RealSSHService) ValidateConfiguration(ctx context.Context) (*SSHValidationResult, error) {
	s.logger.Info(ctx, "validating_ssh_configuration")

	result := &SSHValidationResult{
		Valid:           true,
		Issues:          []*SSHIssue{},
		Recommendations: []string{},
		Keys:            []*SSHKeyInfo{},
	}

	// Check SSH directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, &SSHIssue{
			Type:        "ssh_directory_error",
			Severity:    "high",
			Description: "Failed to get home directory",
			Fix:         "Check system configuration",
		})
		return result, nil
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		result.Valid = false
		result.Issues = append(result.Issues, &SSHIssue{
			Type:        "ssh_directory_missing",
			Severity:    "high",
			Description: "SSH directory does not exist",
			Fix:         "Create SSH directory with proper permissions",
		})
		return result, nil
	}

	// Check SSH directory permissions
	if info, err := os.Stat(sshDir); err == nil {
		if info.Mode().Perm()&077 != 0 {
			result.Issues = append(result.Issues, &SSHIssue{
				Type:        "ssh_directory_permissions",
				Severity:    "medium",
				Description: "SSH directory has incorrect permissions",
				Fix:         "Set SSH directory permissions to 700",
			})
		}
	}

	// List and validate SSH keys
	keys, err := s.ListKeys(ctx)
	if err != nil {
		result.Issues = append(result.Issues, &SSHIssue{
			Type:        "ssh_keys_error",
			Severity:    "medium",
			Description: "Failed to list SSH keys",
			Fix:         "Check SSH directory permissions",
		})
	} else {
		result.Keys = keys
		if len(keys) == 0 {
			result.Issues = append(result.Issues, &SSHIssue{
				Type:        "no_ssh_keys",
				Severity:    "medium",
				Description: "No SSH keys found",
				Fix:         "Generate SSH keys for GitHub authentication",
			})
		}
	}

	// Check for invalid keys
	for _, key := range keys {
		if !key.Valid {
			result.Issues = append(result.Issues, &SSHIssue{
				Type:        "invalid_ssh_key",
				Severity:    "high",
				Description: fmt.Sprintf("Invalid SSH key: %s", key.Path),
				Fix:         "Regenerate or fix the SSH key",
			})
		}
	}

	// Determine overall validity
	if len(result.Issues) > 0 {
		result.Valid = false
	}

	// Generate recommendations
	if len(keys) == 0 {
		result.Recommendations = append(result.Recommendations, "Generate SSH keys for GitHub authentication")
	}
	if result.Valid {
		result.Recommendations = append(result.Recommendations, "SSH configuration looks good!")
	}

	s.logger.Info(ctx, "ssh_configuration_validated",
		observability.F("valid", result.Valid),
		observability.F("issues_count", len(result.Issues)),
		observability.F("keys_count", len(result.Keys)),
	)

	return result, nil
}

// FixPermissions fixes SSH key permissions
func (s *RealSSHService) FixPermissions(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "fixing_ssh_permissions",
		observability.F("key_path", keyPath),
	)

	// Fix private key permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to fix private key permissions: %w", err)
	}

	// Fix public key permissions
	publicKeyPath := keyPath + ".pub"
	if _, err := os.Stat(publicKeyPath); err == nil {
		if err := os.Chmod(publicKeyPath, 0644); err != nil {
			return fmt.Errorf("failed to fix public key permissions: %w", err)
		}
	}

	s.logger.Info(ctx, "ssh_permissions_fixed_successfully",
		observability.F("key_path", keyPath),
	)

	return nil
}

// GenerateSSHConfig generates SSH configuration
func (s *RealSSHService) GenerateSSHConfig(ctx context.Context) (string, error) {
	s.logger.Info(ctx, "generating_ssh_config")

	// Get SSH keys
	keys, err := s.ListKeys(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list SSH keys: %w", err)
	}

	config := "# SSH Configuration for GitPersona\n"
	config += "# Generated automatically - do not edit manually\n\n"

	for _, key := range keys {
		if !key.Valid {
			continue
		}

		// Generate host entry
		hostName := fmt.Sprintf("github-%s", key.Email)
		config += fmt.Sprintf("Host %s\n", hostName)
		config += "  HostName github.com\n"
		config += "  User git\n"
		config += fmt.Sprintf("  IdentityFile %s\n", key.Path)
		config += "  IdentitiesOnly yes\n\n"
	}

	s.logger.Info(ctx, "ssh_config_generated_successfully",
		observability.F("keys_count", len(keys)),
	)

	return config, nil
}

// TestGitHubAuthentication tests GitHub authentication with a key
func (s *RealSSHService) TestGitHubAuthentication(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "testing_github_authentication",
		observability.F("key_path", keyPath),
	)

	// Test SSH connection to GitHub
	cmd := exec.Command("ssh", "-T", "-i", keyPath, "-o", "StrictHostKeyChecking=no", "git@github.com")
	output, err := s.runner.CombinedOutput(ctx, cmd.Path, cmd.Args[1:]...)
	if err != nil {
		s.logger.Error(ctx, "github_authentication_failed",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	// Check if authentication was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "successfully authenticated") || strings.Contains(outputStr, "Hi") {
		s.logger.Info(ctx, "github_authentication_successful",
			observability.F("key_path", keyPath),
		)
		return nil
	}

	return fmt.Errorf("GitHub authentication failed: %s", outputStr)
}

// DiagnoseIssues diagnoses SSH configuration issues
func (s *RealSSHService) DiagnoseIssues(ctx context.Context) ([]*SSHIssue, error) {
	result, err := s.ValidateConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}

	return result.Issues, nil
}

// FixIssues fixes SSH configuration issues
func (s *RealSSHService) FixIssues(ctx context.Context, issues []*SSHIssue) error {
	s.logger.Info(ctx, "fixing_ssh_issues",
		observability.F("issues_count", len(issues)),
	)

	for _, issue := range issues {
		if issue.Fixed {
			continue
		}

		switch issue.Type {
		case "ssh_directory_permissions":
			homeDir, _ := os.UserHomeDir()
			sshDir := filepath.Join(homeDir, ".ssh")
			if err := os.Chmod(sshDir, 0700); err == nil {
				issue.Fixed = true
			}
		case "invalid_ssh_key":
			// Mark as fixed if we can't fix it automatically
			issue.Fixed = false
		}
	}

	s.logger.Info(ctx, "ssh_issues_fix_attempted",
		observability.F("issues_count", len(issues)),
	)

	return nil
}

// extractEmailFromKey extracts email from SSH public key
func (s *RealSSHService) extractEmailFromKey(publicKey string) string {
	// SSH public key format: "ssh-rsa AAAAB3NzaC1yc2E... user@host"
	parts := strings.Fields(publicKey)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
