package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealGitHubService implements GitHubService using GitHub CLI
type RealGitHubService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewRealGitHubService creates a new real GitHub service
func NewRealGitHubService(logger observability.Logger, runner execrunner.CmdRunner) *RealGitHubService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &RealGitHubService{
		logger: logger,
		runner: runner,
	}
}

// Authenticate authenticates with GitHub
func (s *RealGitHubService) Authenticate(ctx context.Context, token string) error {
	s.logger.Info(ctx, "authenticating_with_github")

	// Test authentication
	_, err := s.runner.CombinedOutput(ctx, "gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	s.logger.Info(ctx, "github_authentication_successful")
	return nil
}

// GetAuthenticatedUser gets the currently authenticated user
func (s *RealGitHubService) GetAuthenticatedUser(ctx context.Context) (*GitHubUser, error) {
	s.logger.Info(ctx, "getting_authenticated_user")

	out, err := s.runner.CombinedOutput(ctx, "gh", "api", "user")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	var user GitHubUser
	if err := json.Unmarshal(out, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	s.logger.Info(ctx, "authenticated_user_retrieved",
		observability.F("login", user.Login),
		observability.F("name", user.Name),
	)

	return &user, nil
}

// TestSSHKey tests SSH key authentication with GitHub
func (s *RealGitHubService) TestSSHKey(ctx context.Context, keyPath string) error {
	s.logger.Info(ctx, "testing_ssh_key", observability.F("key_path", keyPath))

	out, err := s.runner.CombinedOutput(ctx, "ssh", "-T", "git@github.com", "-i", keyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=no")

	// SSH returns exit code 1 for successful authentication with GitHub
	if err != nil && !strings.Contains(string(out), "successfully authenticated") {
		return fmt.Errorf("SSH key test failed: %s", string(out))
	}

	s.logger.Info(ctx, "ssh_key_test_successful",
		observability.F("key_path", keyPath),
		observability.F("response", string(out)),
	)

	return nil
}

// ListRepositories lists repositories for the authenticated user
func (s *RealGitHubService) ListRepositories(ctx context.Context) ([]*GitHubRepository, error) {
	s.logger.Info(ctx, "listing_repositories")

	out, err := s.runner.CombinedOutput(ctx, "gh", "repo", "list", "--json", "id,name,fullName,description,isPrivate,url,sshUrl,owner")
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var repos []*GitHubRepository
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repository data: %w", err)
	}

	s.logger.Info(ctx, "repositories_listed",
		observability.F("count", len(repos)),
	)

	return repos, nil
}

// GetRepository gets a specific repository
func (s *RealGitHubService) GetRepository(ctx context.Context, owner, name string) (*GitHubRepository, error) {
	s.logger.Info(ctx, "getting_repository",
		observability.F("owner", owner),
		observability.F("name", name),
	)

	out, err := s.runner.CombinedOutput(ctx, "gh", "repo", "view", fmt.Sprintf("%s/%s", owner, name), "--json", "id,name,fullName,description,isPrivate,url,sshUrl,owner")
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var repo GitHubRepository
	if err := json.Unmarshal(out, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse repository data: %w", err)
	}

	s.logger.Info(ctx, "repository_retrieved",
		observability.F("owner", owner),
		observability.F("name", name),
	)

	return &repo, nil
}

// CreateRepository creates a new repository
func (s *RealGitHubService) CreateRepository(ctx context.Context, name string, private bool) (*GitHubRepository, error) {
	s.logger.Info(ctx, "creating_repository",
		observability.F("name", name),
		observability.F("private", private),
	)

	args := []string{"repo", "create", name, "--json", "id,name,fullName,description,isPrivate,url,sshUrl,owner"}
	if private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}

	out, err := s.runner.CombinedOutput(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	var repo GitHubRepository
	if err := json.Unmarshal(out, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse repository data: %w", err)
	}

	s.logger.Info(ctx, "repository_created",
		observability.F("name", name),
		observability.F("url", repo.URL),
	)

	return &repo, nil
}

// DeleteRepository deletes a repository
func (s *RealGitHubService) DeleteRepository(ctx context.Context, owner, name string) error {
	s.logger.Info(ctx, "deleting_repository",
		observability.F("owner", owner),
		observability.F("name", name),
	)

	_, err := s.runner.CombinedOutput(ctx, "gh", "repo", "delete", fmt.Sprintf("%s/%s", owner, name), "--yes")
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	s.logger.Info(ctx, "repository_deleted",
		observability.F("owner", owner),
		observability.F("name", name),
	)

	return nil
}

// ListSSHKeys lists SSH keys for the authenticated user
func (s *RealGitHubService) ListSSHKeys(ctx context.Context) ([]*GitHubSSHKey, error) {
	s.logger.Info(ctx, "listing_ssh_keys")

	out, err := s.runner.CombinedOutput(ctx, "gh", "ssh-key", "list")
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	keys := s.parseSSHKeyList(string(out))

	s.logger.Info(ctx, "ssh_keys_listed",
		observability.F("count", len(keys)),
	)

	return keys, nil
}

// AddSSHKey adds an SSH key to the authenticated user's account
func (s *RealGitHubService) AddSSHKey(ctx context.Context, title, keyPath string) (*GitHubSSHKey, error) {
	s.logger.Info(ctx, "adding_ssh_key",
		observability.F("title", title),
		observability.F("key_path", keyPath),
	)

	// Read the public key
	pubKeyPath := keyPath
	if !strings.HasSuffix(keyPath, ".pub") {
		pubKeyPath = keyPath + ".pub"
	}

	_, err := s.runner.CombinedOutput(ctx, "cat", pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Add the key using gh CLI
	_, err = s.runner.CombinedOutput(ctx, "gh", "ssh-key", "add", pubKeyPath, "--title", title)
	if err != nil {
		return nil, fmt.Errorf("failed to add SSH key: %w", err)
	}

	// Return the created key (we'll need to parse it from the list)
	keys, err := s.ListSSHKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve added key: %w", err)
	}

	// Find the key we just added by title
	for _, key := range keys {
		if key.Title == title {
			s.logger.Info(ctx, "ssh_key_added_successfully",
				observability.F("title", title),
				observability.F("key_id", key.ID),
			)
			return key, nil
		}
	}

	return nil, fmt.Errorf("added SSH key not found in list")
}

// DeleteSSHKey deletes an SSH key by ID
func (s *RealGitHubService) DeleteSSHKey(ctx context.Context, keyID int) error {
	s.logger.Info(ctx, "deleting_ssh_key",
		observability.F("key_id", keyID),
	)

	_, err := s.runner.CombinedOutput(ctx, "gh", "ssh-key", "delete", strconv.Itoa(keyID), "--yes")
	if err != nil {
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	s.logger.Info(ctx, "ssh_key_deleted_successfully",
		observability.F("key_id", keyID),
	)

	return nil
}

// ListOrganizations lists organizations for the authenticated user
func (s *RealGitHubService) ListOrganizations(ctx context.Context) ([]*GitHubOrganization, error) {
	s.logger.Info(ctx, "listing_organizations")

	out, err := s.runner.CombinedOutput(ctx, "gh", "api", "user/orgs")
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	var orgs []*GitHubOrganization
	if err := json.Unmarshal(out, &orgs); err != nil {
		return nil, fmt.Errorf("failed to parse organization data: %w", err)
	}

	s.logger.Info(ctx, "organizations_listed",
		observability.F("count", len(orgs)),
	)

	return orgs, nil
}

// GetOrganization gets a specific organization
func (s *RealGitHubService) GetOrganization(ctx context.Context, name string) (*GitHubOrganization, error) {
	s.logger.Info(ctx, "getting_organization",
		observability.F("name", name),
	)

	out, err := s.runner.CombinedOutput(ctx, "gh", "api", fmt.Sprintf("orgs/%s", name))
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	var org GitHubOrganization
	if err := json.Unmarshal(out, &org); err != nil {
		return nil, fmt.Errorf("failed to parse organization data: %w", err)
	}

	s.logger.Info(ctx, "organization_retrieved",
		observability.F("name", name),
	)

	return &org, nil
}

// parseSSHKeyList parses the output of `gh ssh-key list` into structured data
func (s *RealGitHubService) parseSSHKeyList(output string) []*GitHubSSHKey {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var keys []*GitHubSSHKey

	// Regular expression to parse the output format
	// Example: "My Key    ssh-ed25519 AAAAC3... 2023-09-05T10:30:00Z 12345 authentication"
	re := regexp.MustCompile(`^(.+?)\s+(ssh-\S+)\s+(\S+)\s+(\S+)\s+(\d+)\s+(\S+)`)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) >= 6 {
			keyID, _ := strconv.Atoi(matches[5])
			key := &GitHubSSHKey{
				ID:    keyID,
				Title: strings.TrimSpace(matches[1]),
				Key:   fmt.Sprintf("%s %s", matches[2], matches[3]),
			}
			keys = append(keys, key)
		}
	}

	return keys
}

// ValidateSSHKeyOwnership validates that an SSH key belongs to the expected GitHub account
func (s *RealGitHubService) ValidateSSHKeyOwnership(ctx context.Context, keyPath, expectedUsername string) (*SSHKeyOwnershipResult, error) {
	s.logger.Info(ctx, "validating_ssh_key_ownership",
		observability.F("key_path", keyPath),
		observability.F("expected_username", expectedUsername),
	)

	result := &SSHKeyOwnershipResult{
		KeyPath:          keyPath,
		ExpectedUsername: expectedUsername,
		Valid:            false,
	}

	// Test SSH authentication
	out, err := s.runner.CombinedOutput(ctx, "ssh", "-T", "git@github.com", "-i", keyPath, "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=no")

	if err != nil && !strings.Contains(string(out), "successfully authenticated") {
		result.Error = fmt.Sprintf("SSH authentication failed: %s", string(out))
		return result, nil
	}

	// Extract username from SSH response
	output := string(out)
	if strings.Contains(output, "Hi ") && strings.Contains(output, "!") {
		re := regexp.MustCompile(`Hi\s+([^!]+)!`)
		matches := re.FindStringSubmatch(output)
		if len(matches) > 1 {
			result.ActualUsername = strings.TrimSpace(matches[1])
			result.Valid = (result.ActualUsername == expectedUsername)

			if !result.Valid {
				result.Error = fmt.Sprintf("SSH key belongs to '%s' but expected '%s'", result.ActualUsername, expectedUsername)
			}
		}
	}

	s.logger.Info(ctx, "ssh_key_ownership_validation_complete",
		observability.F("key_path", keyPath),
		observability.F("valid", result.Valid),
		observability.F("actual_username", result.ActualUsername),
	)

	return result, nil
}

// AutoFixSSHKeyConflicts automatically resolves SSH key conflicts
func (s *RealGitHubService) AutoFixSSHKeyConflicts(ctx context.Context, conflicts []SSHKeyConflict) (*SSHKeyFixResult, error) {
	s.logger.Info(ctx, "auto_fixing_ssh_key_conflicts",
		observability.F("conflicts", len(conflicts)),
	)

	result := &SSHKeyFixResult{
		FixedConflicts:   []string{},
		FailedConflicts:  []string{},
		ActionsPerformed: []string{},
	}

	for _, conflict := range conflicts {
		switch conflict.Type {
		case "duplicate_account_authentication":
			// For duplicate keys, we'll remove the older or less descriptive one
			if err := s.handleDuplicateKeyConflict(ctx, conflict, result); err != nil {
				result.FailedConflicts = append(result.FailedConflicts,
					fmt.Sprintf("Failed to resolve conflict for %s: %v", conflict.ConflictingKeyPath, err))
			} else {
				result.FixedConflicts = append(result.FixedConflicts, conflict.ConflictingKeyPath)
			}

		case "wrong_account_key":
			// For keys on wrong accounts, we'll remove them and suggest re-adding
			if err := s.handleWrongAccountKey(ctx, conflict, result); err != nil {
				result.FailedConflicts = append(result.FailedConflicts,
					fmt.Sprintf("Failed to fix wrong account key %s: %v", conflict.ConflictingKeyPath, err))
			} else {
				result.FixedConflicts = append(result.FixedConflicts, conflict.ConflictingKeyPath)
			}
		}
	}

	s.logger.Info(ctx, "ssh_key_conflicts_fix_complete",
		observability.F("fixed", len(result.FixedConflicts)),
		observability.F("failed", len(result.FailedConflicts)),
	)

	return result, nil
}

// handleDuplicateKeyConflict handles duplicate key conflicts
func (s *RealGitHubService) handleDuplicateKeyConflict(ctx context.Context, conflict SSHKeyConflict, result *SSHKeyFixResult) error {
	// List SSH keys to find which one to remove
	keys, err := s.ListSSHKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}

	// Find keys that might be duplicates (simplified logic)
	for _, key := range keys {
		if strings.Contains(key.Title, "old") || strings.Contains(key.Title, "duplicate") || strings.Contains(key.Title, "backup") {
			if err := s.DeleteSSHKey(ctx, key.ID); err != nil {
				return fmt.Errorf("failed to delete duplicate key %s: %w", key.Title, err)
			}
			result.ActionsPerformed = append(result.ActionsPerformed,
				fmt.Sprintf("Removed duplicate SSH key: %s", key.Title))
			return nil
		}
	}

	return fmt.Errorf("could not identify which key to remove")
}

// handleWrongAccountKey handles keys on the wrong account
func (s *RealGitHubService) handleWrongAccountKey(ctx context.Context, conflict SSHKeyConflict, result *SSHKeyFixResult) error {
	// This is more complex and typically requires manual intervention
	result.ActionsPerformed = append(result.ActionsPerformed,
		fmt.Sprintf("Identified wrong account key: %s (manual intervention required)", conflict.ConflictingKeyPath))

	// We could add logic here to:
	// 1. Remove the key from the wrong account
	// 2. Generate a new key
	// 3. Add the new key to the correct account
	// But this is risky without user confirmation

	return nil
}

// Data structures for enhanced GitHub operations

// SSHKeyOwnershipResult represents the result of SSH key ownership validation
type SSHKeyOwnershipResult struct {
	KeyPath          string `json:"key_path"`
	ExpectedUsername string `json:"expected_username"`
	ActualUsername   string `json:"actual_username"`
	Valid            bool   `json:"valid"`
	Error            string `json:"error,omitempty"`
}

// SSHKeyFixResult represents the result of SSH key conflict resolution
type SSHKeyFixResult struct {
	FixedConflicts   []string `json:"fixed_conflicts"`
	FailedConflicts  []string `json:"failed_conflicts"`
	ActionsPerformed []string `json:"actions_performed"`
}

// Enhanced account setup with conflict detection
func (s *RealGitHubService) SetupAccountWithValidation(ctx context.Context, username, keyPath string) (*AccountSetupResult, error) {
	s.logger.Info(ctx, "setting_up_account_with_validation",
		observability.F("username", username),
		observability.F("key_path", keyPath),
	)

	result := &AccountSetupResult{
		Username: username,
		KeyPath:  keyPath,
		Success:  false,
		Issues:   []string{},
		Actions:  []string{},
	}

	// Validate SSH key ownership
	ownership, err := s.ValidateSSHKeyOwnership(ctx, keyPath, username)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Ownership validation failed: %v", err))
		return result, nil
	}

	if !ownership.Valid {
		result.Issues = append(result.Issues, ownership.Error)

		// Attempt to fix by generating a new key and adding it
		if err := s.generateAndAddNewKey(ctx, username, keyPath, result); err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("Failed to generate new key: %v", err))
			return result, nil
		}
	}

	// Test final configuration
	if err := s.TestSSHKey(ctx, keyPath); err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Final SSH test failed: %v", err))
		return result, nil
	}

	result.Success = true
	result.Actions = append(result.Actions, "Account setup completed successfully")

	s.logger.Info(ctx, "account_setup_with_validation_complete",
		observability.F("username", username),
		observability.F("success", result.Success),
	)

	return result, nil
}

// generateAndAddNewKey generates a new SSH key and adds it to GitHub
func (s *RealGitHubService) generateAndAddNewKey(ctx context.Context, username, keyPath string, result *AccountSetupResult) error {
	// Generate new SSH key
	email := fmt.Sprintf("%s@users.noreply.github.com", username)
	newKeyPath := keyPath + "_fixed"

	cmd := exec.CommandContext(ctx, "ssh-keygen", "-t", "ed25519", "-C", email, "-f", newKeyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	result.Actions = append(result.Actions, fmt.Sprintf("Generated new SSH key: %s", newKeyPath))

	// Add key to GitHub
	title := fmt.Sprintf("GitPersona-%s-auto-fixed", username)
	if _, err := s.AddSSHKey(ctx, title, newKeyPath); err != nil {
		return fmt.Errorf("failed to add SSH key to GitHub: %w", err)
	}

	result.Actions = append(result.Actions, fmt.Sprintf("Added SSH key to GitHub: %s", title))

	// Update the result to reflect the new key path
	result.KeyPath = newKeyPath

	return nil
}

// AccountSetupResult represents the result of account setup
type AccountSetupResult struct {
	Username string   `json:"username"`
	KeyPath  string   `json:"key_path"`
	Success  bool     `json:"success"`
	Issues   []string `json:"issues"`
	Actions  []string `json:"actions"`
}
