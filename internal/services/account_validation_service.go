package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// AccountValidationService provides comprehensive account validation with isolation support
type AccountValidationService struct {
	logger              observability.Logger
	tokenService        *IsolatedTokenService
	directGitHubService *DirectGitHubAPIService
	sshManager          *IsolatedSSHManager
	validationCache     map[string]*ValidationCacheEntry
}

// ValidationCacheEntry caches validation results to avoid repeated expensive operations
type ValidationCacheEntry struct {
	Result    *ComprehensiveValidationResult `json:"result"`
	ExpiresAt time.Time                      `json:"expires_at"`
}

// ComprehensiveValidationResult contains detailed validation results
type ComprehensiveValidationResult struct {
	AccountAlias        string    `json:"account_alias"`
	OverallValid        bool      `json:"overall_valid"`
	ValidationTimestamp time.Time `json:"validation_timestamp"`

	// Individual validation results
	BasicValidation     *BasicValidationResult     `json:"basic_validation"`
	TokenValidation     *TokenValidationResult     `json:"token_validation"`
	SSHValidation       *SSHValidationResult       `json:"ssh_validation"`
	GitHubValidation    *GitHubValidationResult    `json:"github_validation"`
	IsolationValidation *IsolationValidationResult `json:"isolation_validation"`

	// Summary
	ValidationErrors []string  `json:"validation_errors"`
	Warnings         []string  `json:"warnings"`
	Recommendations  []string  `json:"recommendations"`
	NextValidation   time.Time `json:"next_validation"`
}

// BasicValidationResult validates basic account configuration
type BasicValidationResult struct {
	Valid               bool     `json:"valid"`
	HasName             bool     `json:"has_name"`
	HasEmail            bool     `json:"has_email"`
	HasGitHubUsername   bool     `json:"has_github_username"`
	HasSSHKeyPath       bool     `json:"has_ssh_key_path"`
	EmailFormatValid    bool     `json:"email_format_valid"`
	UsernameFormatValid bool     `json:"username_format_valid"`
	Issues              []string `json:"issues"`
}

// TokenValidationResult validates GitHub token
type TokenValidationResult struct {
	Valid           bool      `json:"valid"`
	TokenExists     bool      `json:"token_exists"`
	TokenEncrypted  bool      `json:"token_encrypted"`
	UsernameMatches bool      `json:"username_matches"`
	TokenActive     bool      `json:"token_active"`
	LastValidated   time.Time `json:"last_validated"`
	TokenType       string    `json:"token_type"`
	Scopes          []string  `json:"scopes"`
	Issues          []string  `json:"issues"`
}

// SSHValidationResult validates SSH configuration
type SSHValidationResult struct {
	Valid                bool      `json:"valid"`
	KeyExists            bool      `json:"key_exists"`
	KeyReadable          bool      `json:"key_readable"`
	KeyFormatValid       bool      `json:"key_format_valid"`
	PublicKeyExists      bool      `json:"public_key_exists"`
	PermissionsCorrect   bool      `json:"permissions_correct"`
	AgentIsolated        bool      `json:"agent_isolated"`
	ConnectivityTested   bool      `json:"connectivity_tested"`
	ConnectivityWorking  bool      `json:"connectivity_working"`
	LastConnectivityTest time.Time `json:"last_connectivity_test"`
	Issues               []string  `json:"issues"`
}

// GitHubValidationResult validates GitHub API integration
type GitHubValidationResult struct {
	Valid            bool      `json:"valid"`
	APIAccessible    bool      `json:"api_accessible"`
	UserInfoMatches  bool      `json:"user_info_matches"`
	SSHKeyRegistered bool      `json:"ssh_key_registered"`
	CorrectUserAuth  bool      `json:"correct_user_auth"`
	RateLimitOK      bool      `json:"rate_limit_ok"`
	LastAPICheck     time.Time `json:"last_api_check"`
	ActualUsername   string    `json:"actual_username"`
	ActualUserID     string    `json:"actual_user_id"`
	Issues           []string  `json:"issues"`
}

// IsolationValidationResult validates isolation configuration
type IsolationValidationResult struct {
	Valid                    bool                  `json:"valid"`
	IsolationEnabled         bool                  `json:"isolation_enabled"`
	IsolationLevel           models.IsolationLevel `json:"isolation_level"`
	SSHIsolationConfigured   bool                  `json:"ssh_isolation_configured"`
	TokenIsolationConfigured bool                  `json:"token_isolation_configured"`
	EnvironmentIsolated      bool                  `json:"environment_isolated"`
	CrossAccountLeakage      bool                  `json:"cross_account_leakage"`
	Issues                   []string              `json:"issues"`
}

// NewAccountValidationService creates a new account validation service
func NewAccountValidationService(
	logger observability.Logger,
	tokenService *IsolatedTokenService,
	directGitHubService *DirectGitHubAPIService,
	sshManager *IsolatedSSHManager,
) *AccountValidationService {
	return &AccountValidationService{
		logger:              logger,
		tokenService:        tokenService,
		directGitHubService: directGitHubService,
		sshManager:          sshManager,
		validationCache:     make(map[string]*ValidationCacheEntry),
	}
}

// ValidateAccount performs comprehensive account validation
func (s *AccountValidationService) ValidateAccount(ctx context.Context, account *models.Account) (*ComprehensiveValidationResult, error) {
	s.logger.Info(ctx, "starting_comprehensive_account_validation",
		observability.F("account", account.Alias),
		observability.F("isolation_level", account.GetIsolationLevel()),
	)

	// Check cache first
	if cached := s.getFromCache(account.Alias); cached != nil {
		s.logger.Info(ctx, "returning_cached_validation_result",
			observability.F("account", account.Alias),
		)
		return cached, nil
	}

	startTime := time.Now()
	result := &ComprehensiveValidationResult{
		AccountAlias:        account.Alias,
		ValidationTimestamp: startTime,
		ValidationErrors:    []string{},
		Warnings:            []string{},
		Recommendations:     []string{},
	}

	// Perform individual validations
	result.BasicValidation = s.validateBasicConfiguration(ctx, account)
	result.TokenValidation = s.validateTokenConfiguration(ctx, account)
	result.SSHValidation = s.validateSSHConfiguration(ctx, account)
	result.GitHubValidation = s.validateGitHubIntegration(ctx, account)
	result.IsolationValidation = s.validateIsolationConfiguration(ctx, account)

	// Determine overall validity
	result.OverallValid = result.BasicValidation.Valid &&
		result.TokenValidation.Valid &&
		result.SSHValidation.Valid &&
		result.GitHubValidation.Valid &&
		result.IsolationValidation.Valid

	// Collect all issues
	result.ValidationErrors = append(result.ValidationErrors, result.BasicValidation.Issues...)
	result.ValidationErrors = append(result.ValidationErrors, result.TokenValidation.Issues...)
	result.ValidationErrors = append(result.ValidationErrors, result.SSHValidation.Issues...)
	result.ValidationErrors = append(result.ValidationErrors, result.GitHubValidation.Issues...)
	result.ValidationErrors = append(result.ValidationErrors, result.IsolationValidation.Issues...)

	// Generate recommendations
	s.generateRecommendations(result, account)

	// Set next validation time based on account settings
	result.NextValidation = s.calculateNextValidation(account)

	// Cache result
	s.cacheResult(account.Alias, result)

	s.logger.Info(ctx, "comprehensive_account_validation_completed",
		observability.F("account", account.Alias),
		observability.F("overall_valid", result.OverallValid),
		observability.F("duration", time.Since(startTime)),
		observability.F("issues", len(result.ValidationErrors)),
	)

	return result, nil
}

// validateBasicConfiguration validates basic account configuration
func (s *AccountValidationService) validateBasicConfiguration(ctx context.Context, account *models.Account) *BasicValidationResult {
	result := &BasicValidationResult{
		Issues: []string{},
	}

	// Check required fields
	result.HasName = account.Name != ""
	result.HasEmail = account.Email != ""
	result.HasGitHubUsername = account.GitHubUsername != ""
	result.HasSSHKeyPath = account.SSHKeyPath != ""

	// Validate formats using the model's validation
	if err := account.Validate(); err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("account validation failed: %v", err))
		result.EmailFormatValid = false
		result.UsernameFormatValid = false
	} else {
		result.EmailFormatValid = true
		result.UsernameFormatValid = true
	}

	// Check individual requirements
	if !result.HasName {
		result.Issues = append(result.Issues, "missing name field")
	}
	if !result.HasEmail {
		result.Issues = append(result.Issues, "missing email field")
	}
	if !result.HasGitHubUsername {
		result.Issues = append(result.Issues, "missing GitHub username")
	}
	if !result.HasSSHKeyPath {
		result.Issues = append(result.Issues, "missing SSH key path")
	}

	result.Valid = len(result.Issues) == 0

	return result
}

// validateTokenConfiguration validates GitHub token configuration
func (s *AccountValidationService) validateTokenConfiguration(ctx context.Context, account *models.Account) *TokenValidationResult {
	result := &TokenValidationResult{
		Issues: []string{},
	}

	if s.tokenService == nil {
		result.Issues = append(result.Issues, "token service not available")
		return result
	}

	// Check if token exists
	_, err := s.tokenService.GetToken(ctx, account.Alias)
	if err != nil {
		result.TokenExists = false
		result.Issues = append(result.Issues, fmt.Sprintf("no token found: %v", err))
		return result
	}

	result.TokenExists = true
	result.TokenEncrypted = true // IsolatedTokenService always encrypts

	// Get token metadata
	metadata, err := s.tokenService.GetTokenMetadata(ctx, account.Alias)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("failed to get token metadata: %v", err))
	} else {
		result.TokenType = metadata.TokenType
		result.Scopes = metadata.Scopes
		result.LastValidated = metadata.LastValidated
		result.TokenActive = metadata.IsValid
		result.UsernameMatches = metadata.Username == account.GitHubUsername

		if !result.UsernameMatches {
			result.Issues = append(result.Issues,
				fmt.Sprintf("token username mismatch: expected '%s', got '%s'",
					account.GitHubUsername, metadata.Username))
		}
	}

	// Validate token isolation
	if err := s.tokenService.ValidateTokenIsolation(ctx, account.Alias, account.GitHubUsername); err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("token isolation validation failed: %v", err))
	}

	// Test token with GitHub API if direct service is available
	if s.directGitHubService != nil {
		if err := s.directGitHubService.TestAPIAccess(ctx, account.Alias); err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("GitHub API access test failed: %v", err))
			result.TokenActive = false
		}
	}

	result.Valid = len(result.Issues) == 0
	return result
}

// validateSSHConfiguration validates SSH configuration
func (s *AccountValidationService) validateSSHConfiguration(ctx context.Context, account *models.Account) *SSHValidationResult {
	result := &SSHValidationResult{
		Issues: []string{},
	}

	if account.SSHKeyPath == "" {
		result.Issues = append(result.Issues, "no SSH key path configured")
		return result
	}

	// Check if SSH key exists and is readable
	keyInfo, err := os.Stat(account.SSHKeyPath)
	if err != nil {
		result.KeyExists = false
		result.Issues = append(result.Issues, fmt.Sprintf("SSH key not found: %v", err))
		return result
	}

	result.KeyExists = true

	// Check permissions (should be 600)
	mode := keyInfo.Mode()
	if mode&0077 != 0 {
		result.PermissionsCorrect = false
		result.Issues = append(result.Issues, "SSH key permissions too permissive (should be 600)")
	} else {
		result.PermissionsCorrect = true
	}

	// Check if key is readable
	if _, err := os.ReadFile(account.SSHKeyPath); err != nil {
		result.KeyReadable = false
		result.Issues = append(result.Issues, fmt.Sprintf("SSH key not readable: %v", err))
	} else {
		result.KeyReadable = true
		result.KeyFormatValid = true // Assume valid if readable
	}

	// Check if public key exists
	pubKeyPath := account.SSHKeyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		result.PublicKeyExists = false
		result.Issues = append(result.Issues, "SSH public key not found")
	} else {
		result.PublicKeyExists = true
	}

	// Test SSH agent isolation if SSH manager is available
	if s.sshManager != nil {
		// Check if account has an isolated agent
		if agent, err := s.sshManager.GetAccountAgent(account.Alias); err == nil {
			result.AgentIsolated = true
			if !agent.IsRunning {
				result.Issues = append(result.Issues, "SSH agent not running")
			}
		}
	}

	// Test SSH connectivity (optional, can be expensive)
	if result.KeyExists && result.KeyReadable {
		result.ConnectivityTested = true
		result.LastConnectivityTest = time.Now()

		if err := s.testSSHConnectivity(ctx, account.SSHKeyPath, account.GitHubUsername); err != nil {
			result.ConnectivityWorking = false
			result.Issues = append(result.Issues, fmt.Sprintf("SSH connectivity failed: %v", err))
		} else {
			result.ConnectivityWorking = true
		}
	}

	result.Valid = len(result.Issues) == 0
	return result
}

// validateGitHubIntegration validates GitHub API integration
func (s *AccountValidationService) validateGitHubIntegration(ctx context.Context, account *models.Account) *GitHubValidationResult {
	result := &GitHubValidationResult{
		Issues:       []string{},
		LastAPICheck: time.Now(),
	}

	if s.directGitHubService == nil {
		result.Issues = append(result.Issues, "GitHub API service not available")
		return result
	}

	// Test API accessibility
	if err := s.directGitHubService.TestAPIAccess(ctx, account.Alias); err != nil {
		result.APIAccessible = false
		result.Issues = append(result.Issues, fmt.Sprintf("GitHub API not accessible: %v", err))
		return result
	}

	result.APIAccessible = true

	// Get account info and validate
	accountInfo, err := s.directGitHubService.GetAccountInfo(ctx, account.Alias)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("failed to get account info: %v", err))
		return result
	}

	result.ActualUsername = accountInfo.Username
	if userID, ok := accountInfo.Metadata["user_id"]; ok {
		result.ActualUserID = userID
	}

	// Check if username matches
	result.UserInfoMatches = accountInfo.Username == account.GitHubUsername
	if !result.UserInfoMatches {
		result.Issues = append(result.Issues,
			fmt.Sprintf("GitHub username mismatch: expected '%s', got '%s'",
				account.GitHubUsername, accountInfo.Username))
	}

	result.CorrectUserAuth = result.UserInfoMatches
	result.RateLimitOK = true // Assume OK if we got this far

	result.Valid = len(result.Issues) == 0
	return result
}

// validateIsolationConfiguration validates account isolation
func (s *AccountValidationService) validateIsolationConfiguration(ctx context.Context, account *models.Account) *IsolationValidationResult {
	result := &IsolationValidationResult{
		Issues:         []string{},
		IsolationLevel: account.GetIsolationLevel(),
	}

	result.IsolationEnabled = account.IsIsolated()

	if !result.IsolationEnabled {
		if account.GetIsolationLevel() == models.IsolationLevelNone {
			result.Issues = append(result.Issues, "isolation not enabled for account")
		}
		return result
	}

	// Check SSH isolation configuration
	if account.RequiresSSHIsolation() {
		result.SSHIsolationConfigured = true

		// Verify SSH manager is available and configured
		if s.sshManager != nil {
			if agent, err := s.sshManager.GetAccountAgent(account.Alias); err == nil {
				if agent.SocketPath == "" {
					result.Issues = append(result.Issues, "SSH isolation enabled but no isolated socket")
				}
			}
		} else {
			result.Issues = append(result.Issues, "SSH isolation required but SSH manager not available")
		}
	}

	// Check token isolation configuration
	if account.RequiresTokenIsolation() {
		result.TokenIsolationConfigured = true

		if s.tokenService == nil {
			result.Issues = append(result.Issues, "token isolation required but token service not available")
		}
	}

	// Check for cross-account leakage risks
	result.CrossAccountLeakage = s.checkCrossAccountLeakage(ctx, account)
	if result.CrossAccountLeakage {
		result.Issues = append(result.Issues, "potential cross-account leakage detected")
	}

	result.Valid = len(result.Issues) == 0
	return result
}

// testSSHConnectivity tests SSH connectivity to GitHub
func (s *AccountValidationService) testSSHConnectivity(ctx context.Context, keyPath, expectedUsername string) error {
	cmd := exec.CommandContext(ctx, "ssh", "-T", "git@github.com",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// SSH returns exit code 1 for successful authentication with GitHub
	if err != nil && !strings.Contains(outputStr, "successfully authenticated") {
		return fmt.Errorf("SSH authentication failed: %s", outputStr)
	}

	// Check if authenticated as expected user
	if expectedUsername != "" && !strings.Contains(outputStr, expectedUsername) {
		return fmt.Errorf("authenticated as wrong user: expected '%s' in output: %s", expectedUsername, outputStr)
	}

	return nil
}

// checkCrossAccountLeakage checks for potential cross-account security issues
func (s *AccountValidationService) checkCrossAccountLeakage(ctx context.Context, account *models.Account) bool {
	// This is a simplified implementation
	// In production, you'd want more sophisticated checks:
	// - Check if SSH keys are shared between accounts
	// - Check if tokens are shared between accounts
	// - Check environment variable pollution
	// - Check Git configuration conflicts

	return false // Assume no leakage for now
}

// generateRecommendations generates actionable recommendations
func (s *AccountValidationService) generateRecommendations(result *ComprehensiveValidationResult, account *models.Account) {
	if !result.BasicValidation.Valid {
		result.Recommendations = append(result.Recommendations,
			"Complete basic account configuration (name, email, GitHub username)")
	}

	if !result.TokenValidation.Valid {
		result.Recommendations = append(result.Recommendations,
			"Set up isolated GitHub token for this account")
	}

	if !result.SSHValidation.Valid {
		result.Recommendations = append(result.Recommendations,
			"Configure SSH key with proper permissions (600)")
	}

	if !result.IsolationValidation.Valid && account.IsIsolated() {
		result.Recommendations = append(result.Recommendations,
			"Complete isolation configuration for enhanced security")
	}

	if result.OverallValid {
		result.Recommendations = append(result.Recommendations,
			"Account is fully configured and ready for isolated switching")
	}
}

// calculateNextValidation calculates when the next validation should occur
func (s *AccountValidationService) calculateNextValidation(account *models.Account) time.Time {
	if account.IsValidationRequired() {
		return time.Now().Add(time.Hour) // Revalidate soon if required
	}

	// Default validation interval based on isolation level
	switch account.GetIsolationLevel() {
	case models.IsolationLevelComplete, models.IsolationLevelStrict:
		return time.Now().Add(6 * time.Hour) // More frequent validation for strict isolation
	case models.IsolationLevelStandard:
		return time.Now().Add(12 * time.Hour)
	case models.IsolationLevelBasic:
		return time.Now().Add(24 * time.Hour)
	default:
		return time.Now().Add(7 * 24 * time.Hour) // Weekly for no isolation
	}
}

// Cache management

func (s *AccountValidationService) getFromCache(accountAlias string) *ComprehensiveValidationResult {
	entry, exists := s.validationCache[accountAlias]
	if !exists || entry.ExpiresAt.Before(time.Now()) {
		return nil
	}
	return entry.Result
}

func (s *AccountValidationService) cacheResult(accountAlias string, result *ComprehensiveValidationResult) {
	// Cache for 10 minutes
	expiry := time.Now().Add(10 * time.Minute)
	s.validationCache[accountAlias] = &ValidationCacheEntry{
		Result:    result,
		ExpiresAt: expiry,
	}
}

// ClearCache clears the validation cache
func (s *AccountValidationService) ClearCache() {
	s.validationCache = make(map[string]*ValidationCacheEntry)
}

// ValidateAllAccounts validates multiple accounts concurrently
func (s *AccountValidationService) ValidateAllAccounts(ctx context.Context, accounts []*models.Account) (map[string]*ComprehensiveValidationResult, error) {
	results := make(map[string]*ComprehensiveValidationResult)

	// For now, validate sequentially
	// TODO: Add concurrent validation with proper rate limiting
	for _, account := range accounts {
		result, err := s.ValidateAccount(ctx, account)
		if err != nil {
			s.logger.Error(ctx, "account_validation_failed",
				observability.F("account", account.Alias),
				observability.F("error", err.Error()),
			)
			continue
		}
		results[account.Alias] = result
	}

	return results, nil
}
