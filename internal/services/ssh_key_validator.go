package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHKeyValidator provides advanced SSH key validation and conflict detection
type SSHKeyValidator struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewSSHKeyValidator creates a new SSH key validator
func NewSSHKeyValidator(logger observability.Logger, runner execrunner.CmdRunner) *SSHKeyValidator {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &SSHKeyValidator{
		logger: logger,
		runner: runner,
	}
}

// SSHKeyValidationResult represents comprehensive SSH key validation results
type SSHKeyValidationResult struct {
	KeyPath          string               `json:"key_path"`
	Valid            bool                 `json:"valid"`
	Exists           bool                 `json:"exists"`
	Readable         bool                 `json:"readable"`
	GitHubAccount    string               `json:"github_account"`
	ExpectedAccount  string               `json:"expected_account"`
	AuthenticationOK bool                 `json:"authentication_ok"`
	Conflicts        []SSHKeyConflict     `json:"conflicts"`
	Issues           []SSHValidationIssue `json:"issues"`
	Recommendations  []string             `json:"recommendations"`
}

// SSHKeyConflict represents a detected conflict between SSH keys
type SSHKeyConflict struct {
	Type               string `json:"type"`
	Description        string `json:"description"`
	ConflictingKeyPath string `json:"conflicting_key_path"`
	ConflictingAccount string `json:"conflicting_account"`
	Severity           string `json:"severity"`
	Resolution         string `json:"resolution"`
}

// SSHValidationIssue represents an SSH validation issue
type SSHValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Automated   bool   `json:"automated"` // Can be fixed automatically
}

// ValidateSSHKey performs comprehensive validation of an SSH key
func (v *SSHKeyValidator) ValidateSSHKey(ctx context.Context, keyPath, expectedAccount string) (*SSHKeyValidationResult, error) {
	v.logger.Info(ctx, "validating_ssh_key",
		observability.F("key_path", keyPath),
		observability.F("expected_account", expectedAccount),
	)

	result := &SSHKeyValidationResult{
		KeyPath:         keyPath,
		ExpectedAccount: expectedAccount,
		Conflicts:       []SSHKeyConflict{},
		Issues:          []SSHValidationIssue{},
		Recommendations: []string{},
	}

	// Check if key exists and is readable
	if err := v.checkKeyExistence(ctx, result); err != nil {
		return result, err
	}

	if !result.Exists {
		return result, nil
	}

	// Test GitHub authentication
	if err := v.testGitHubAuthentication(ctx, result); err != nil {
		v.logger.Warn(ctx, "github_authentication_test_failed",
			observability.F("key_path", keyPath),
			observability.F("error", err.Error()),
		)
	}

	// Check for account mismatch
	v.checkAccountMismatch(ctx, result)

	// Check for conflicts with other keys
	if err := v.detectKeyConflicts(ctx, result); err != nil {
		v.logger.Warn(ctx, "conflict_detection_failed",
			observability.F("error", err.Error()),
		)
	}

	// Generate recommendations
	v.generateRecommendations(ctx, result)

	// Determine overall validity
	result.Valid = result.Exists && result.Readable && result.AuthenticationOK && len(result.Issues) == 0

	v.logger.Info(ctx, "ssh_key_validation_complete",
		observability.F("key_path", keyPath),
		observability.F("valid", result.Valid),
		observability.F("conflicts", len(result.Conflicts)),
		observability.F("issues", len(result.Issues)),
	)

	return result, nil
}

// checkKeyExistence checks if the SSH key exists and is readable
func (v *SSHKeyValidator) checkKeyExistence(ctx context.Context, result *SSHKeyValidationResult) error {
	// Check if private key exists
	if _, err := v.runner.CombinedOutput(ctx, "test", "-f", result.KeyPath); err != nil {
		result.Exists = false
		result.Issues = append(result.Issues, SSHValidationIssue{
			Type:        "missing_private_key",
			Severity:    "critical",
			Description: fmt.Sprintf("Private key file does not exist: %s", result.KeyPath),
			Fix:         "Generate a new SSH key or verify the correct path",
			Automated:   false,
		})
		return nil
	}

	result.Exists = true

	// Check if private key is readable
	if _, err := v.runner.CombinedOutput(ctx, "test", "-r", result.KeyPath); err != nil {
		result.Readable = false
		result.Issues = append(result.Issues, SSHValidationIssue{
			Type:        "private_key_unreadable",
			Severity:    "high",
			Description: fmt.Sprintf("Private key file is not readable: %s", result.KeyPath),
			Fix:         "Fix file permissions: chmod 600 " + result.KeyPath,
			Automated:   true,
		})
	} else {
		result.Readable = true
	}

	// Check if public key exists
	pubKeyPath := result.KeyPath + ".pub"
	if _, err := v.runner.CombinedOutput(ctx, "test", "-f", pubKeyPath); err != nil {
		result.Issues = append(result.Issues, SSHValidationIssue{
			Type:        "missing_public_key",
			Severity:    "medium",
			Description: fmt.Sprintf("Public key file does not exist: %s", pubKeyPath),
			Fix:         "Generate public key: ssh-keygen -y -f " + result.KeyPath + " > " + pubKeyPath,
			Automated:   true,
		})
	}

	return nil
}

// testGitHubAuthentication tests GitHub authentication with the SSH key
func (v *SSHKeyValidator) testGitHubAuthentication(ctx context.Context, result *SSHKeyValidationResult) error {
	// Ensure SSH socket directories exist before testing
	if err := v.ensureSSHSocketDirectories(ctx); err != nil {
		v.logger.Warn(ctx, "failed_to_ensure_ssh_socket_directories",
			observability.F("error", err.Error()),
		)
		// Continue with the test even if socket directory creation fails
	}

	// Test authentication with the specific key
	out, err := v.runner.CombinedOutput(ctx, "ssh", "-T", "git@github.com", "-i", result.KeyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=no")

	if err != nil {
		result.AuthenticationOK = false
		result.Issues = append(result.Issues, SSHValidationIssue{
			Type:        "github_authentication_failed",
			Severity:    "high",
			Description: fmt.Sprintf("GitHub authentication failed: %s", string(out)),
			Fix:         "Ensure the SSH key is added to the correct GitHub account",
			Automated:   false,
		})
		return err
	}

	result.AuthenticationOK = true

	// Extract the authenticated account from the output
	output := string(out)
	if strings.Contains(output, "Hi ") && strings.Contains(output, "!") {
		// Extract username from "Hi username! You've successfully authenticated..."
		re := regexp.MustCompile(`Hi\s+([^!]+)!`)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			result.GitHubAccount = strings.TrimSpace(matches[1])
		}
	}

	return nil
}

// checkAccountMismatch checks if the key authenticates as the expected account
func (v *SSHKeyValidator) checkAccountMismatch(ctx context.Context, result *SSHKeyValidationResult) {
	if result.ExpectedAccount == "" || result.GitHubAccount == "" {
		return
	}

	if result.GitHubAccount != result.ExpectedAccount {
		result.Issues = append(result.Issues, SSHValidationIssue{
			Type:        "account_mismatch",
			Severity:    "critical",
			Description: "SSH key authenticates as '" + result.GitHubAccount + "' but expected '" + result.ExpectedAccount + "'",
			Fix:         "Remove this key from the wrong GitHub account and add it to the correct one",
			Automated:   false,
		})
	}
}

// detectKeyConflicts detects conflicts between SSH keys
func (v *SSHKeyValidator) detectKeyConflicts(ctx context.Context, result *SSHKeyValidationResult) error {
	if result.GitHubAccount == "" {
		return nil
	}

	// Get SSH directory path dynamically
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	// Get all SSH keys in the .ssh directory
	out, err := v.runner.CombinedOutput(ctx, "find", sshDir, "-name", "id_*", "-type", "f", "!", "-name", "*.pub")
	if err != nil {
		return err
	}

	keyPaths := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, keyPath := range keyPaths {
		if keyPath == "" || keyPath == result.KeyPath {
			continue
		}

		// Test this key
		out, err := v.runner.CombinedOutput(ctx, "ssh", "-T", "git@github.com", "-i", keyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=no")
		if err != nil {
			continue // Skip keys that don't work
		}

		// Extract account
		output := string(out)
		if strings.Contains(output, "Hi ") && strings.Contains(output, "!") {
			re := regexp.MustCompile(`Hi\s+([^!]+)!`)
			matches := re.FindStringSubmatch(output)
			if len(matches) > 1 {
				otherAccount := strings.TrimSpace(matches[1])
				if otherAccount == result.GitHubAccount {
					result.Conflicts = append(result.Conflicts, SSHKeyConflict{
						Type:               "duplicate_account_authentication",
						Description:        fmt.Sprintf("Multiple SSH keys authenticate as the same GitHub account '%s'", otherAccount),
						ConflictingKeyPath: keyPath,
						ConflictingAccount: otherAccount,
						Severity:           "high",
						Resolution:         "Remove one of the conflicting keys from GitHub or use different GitHub accounts",
					})
				}
			}
		}
	}

	// Check for SSH agent conflicts
	v.detectSSHAgentConflicts(ctx, result)

	return nil
}

// detectSSHAgentConflicts detects conflicts in the SSH agent
func (v *SSHKeyValidator) detectSSHAgentConflicts(ctx context.Context, result *SSHKeyValidationResult) {
	// Check SSH agent status
	out, err := v.runner.CombinedOutput(ctx, "ssh-add", "-l")
	if err != nil {
		// SSH agent not running or no keys loaded
		return
	}

	output := string(out)
	if strings.Contains(output, "The agent has no identities") {
		return
	}

	// Count loaded keys
	lines := strings.Split(output, "\n")
	keyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "The agent has no identities") {
			keyCount++
		}
	}

	if keyCount > 1 {
		result.Conflicts = append(result.Conflicts, SSHKeyConflict{
			Type:               "multiple_keys_in_agent",
			Description:        fmt.Sprintf("SSH agent has %d keys loaded, which can cause authentication conflicts", keyCount),
			ConflictingKeyPath: "",
			ConflictingAccount: "",
			Severity:           "medium",
			Resolution:         "Clear SSH agent and load only the required key for the current account",
		})
	}
}

// generateRecommendations generates recommendations based on validation results
func (v *SSHKeyValidator) generateRecommendations(ctx context.Context, result *SSHKeyValidationResult) {
	if !result.Exists {
		result.Recommendations = append(result.Recommendations, "Generate a new SSH key for this account")
		return
	}

	if !result.Readable {
		result.Recommendations = append(result.Recommendations, "Fix SSH key file permissions")
	}

	if !result.AuthenticationOK {
		result.Recommendations = append(result.Recommendations, "Add SSH key to GitHub account or check connectivity")
	}

	if result.GitHubAccount != result.ExpectedAccount && result.ExpectedAccount != "" {
		result.Recommendations = append(result.Recommendations, "Move SSH key from '"+result.GitHubAccount+"' to '"+result.ExpectedAccount+"' GitHub account")
	}

	if len(result.Conflicts) > 0 {
		result.Recommendations = append(result.Recommendations, "Resolve SSH key conflicts by using unique keys per GitHub account")
	}

	if len(result.Issues) == 0 && result.Valid {
		result.Recommendations = append(result.Recommendations, "SSH key configuration is optimal")
	}
}

// FixSSHKeyIssues automatically fixes issues that can be resolved programmatically
func (v *SSHKeyValidator) FixSSHKeyIssues(ctx context.Context, result *SSHKeyValidationResult) error {
	v.logger.Info(ctx, "fixing_ssh_key_issues",
		observability.F("key_path", result.KeyPath),
		observability.F("fixable_issues", len(result.Issues)),
	)

	fixedCount := 0

	for i, issue := range result.Issues {
		if !issue.Automated {
			continue
		}

		switch issue.Type {
		case "private_key_unreadable":
			if err := v.fixKeyPermissions(ctx, result.KeyPath); err != nil {
				v.logger.Error(ctx, "failed_to_fix_key_permissions",
					observability.F("key_path", result.KeyPath),
					observability.F("error", err.Error()),
				)
			} else {
				result.Issues[i].Fix = "Fixed automatically"
				fixedCount++
			}

		case "missing_public_key":
			if err := v.generatePublicKey(ctx, result.KeyPath); err != nil {
				v.logger.Error(ctx, "failed_to_generate_public_key",
					observability.F("key_path", result.KeyPath),
					observability.F("error", err.Error()),
				)
			} else {
				result.Issues[i].Fix = "Fixed automatically"
				fixedCount++
			}
		}
	}

	v.logger.Info(ctx, "ssh_key_issues_fix_complete",
		observability.F("key_path", result.KeyPath),
		observability.F("fixed_count", fixedCount),
	)

	return nil
}

// fixKeyPermissions fixes SSH key file permissions
func (v *SSHKeyValidator) fixKeyPermissions(ctx context.Context, keyPath string) error {
	return v.runner.Run(ctx, "chmod", "600", keyPath)
}

// generatePublicKey generates the public key from the private key
func (v *SSHKeyValidator) generatePublicKey(ctx context.Context, keyPath string) error {
	pubKeyPath := keyPath + ".pub"
	return v.runner.Run(ctx, "ssh-keygen", "-y", "-f", keyPath, ">", pubKeyPath)
}

// ValidateAllAccountKeys validates SSH keys for all configured accounts
func (v *SSHKeyValidator) ValidateAllAccountKeys(ctx context.Context, accounts map[string]interface{}) (map[string]*SSHKeyValidationResult, error) {
	results := make(map[string]*SSHKeyValidationResult)

	for alias, accountData := range accounts {
		if accountMap, ok := accountData.(map[string]interface{}); ok {
			var keyPath, githubUsername string

			if kp, exists := accountMap["ssh_key_path"]; exists {
				keyPath = fmt.Sprintf("%v", kp)
			}

			if gu, exists := accountMap["github_username"]; exists {
				githubUsername = fmt.Sprintf("%v", gu)
			}

			if keyPath != "" {
				result, err := v.ValidateSSHKey(ctx, keyPath, githubUsername)
				if err != nil {
					v.logger.Error(ctx, "failed_to_validate_account_key",
						observability.F("account", alias),
						observability.F("key_path", keyPath),
						observability.F("error", err.Error()),
					)
				}
				results[alias] = result
			}
		}
	}

	return results, nil
}

// ensureSSHSocketDirectories ensures that SSH socket directories exist
func (v *SSHKeyValidator) ensureSSHSocketDirectories(ctx context.Context) error {
	v.logger.Info(ctx, "ensuring_ssh_socket_directories")

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
			v.logger.Warn(ctx, "failed_to_create_socket_directory",
				observability.F("path", socketDir),
				observability.F("error", err.Error()),
			)
			return fmt.Errorf("failed to create socket directory %s: %w", socketDir, err)
		}
		v.logger.Info(ctx, "socket_directory_ensured",
			observability.F("path", socketDir),
		)
	}

	v.logger.Info(ctx, "ssh_socket_directories_ensured")
	return nil
}
