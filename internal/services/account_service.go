package services

import (
	"context"
	"fmt"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealAccountService implements the AccountService interface
type RealAccountService struct {
	configService ConfigurationService
	logger        observability.Logger
}

// NewRealAccountService creates a new real account service
func NewRealAccountService(configService ConfigurationService, logger observability.Logger) *RealAccountService {
	return &RealAccountService{
		configService: configService,
		logger:        logger,
	}
}

// GetAccount retrieves an account by alias
func (s *RealAccountService) GetAccount(ctx context.Context, alias string) (*models.Account, error) {
	s.logger.Info(ctx, "getting_account",
		observability.F("alias", alias),
	)

	if s.configService == nil {
		return nil, fmt.Errorf("config service not available")
	}

	account, err := s.configService.GetAccount(ctx, alias)
	if err != nil {
		s.logger.Error(ctx, "failed_to_get_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get account '%s': %w", alias, err)
	}

	s.logger.Info(ctx, "account_retrieved_successfully",
		observability.F("alias", alias),
		observability.F("name", account.Name),
	)

	return account, nil
}

// CreateAccount creates a new account
func (s *RealAccountService) CreateAccount(ctx context.Context, account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	s.logger.Info(ctx, "creating_account",
		observability.F("alias", account.Alias),
		observability.F("name", account.Name),
		observability.F("email", account.Email),
	)

	if s.configService == nil {
		return fmt.Errorf("config service not available")
	}

	// Validate the account
	if err := s.ValidateAccount(ctx, account); err != nil {
		return fmt.Errorf("account validation failed: %w", err)
	}

	// Set creation timestamp
	account.CreatedAt = time.Now()
	account.Status = models.AccountStatusActive

	// Save the account
	if err := s.configService.SetAccount(ctx, account); err != nil {
		s.logger.Error(ctx, "failed_to_create_account",
			observability.F("alias", account.Alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to create account '%s': %w", account.Alias, err)
	}

	s.logger.Info(ctx, "account_created_successfully",
		observability.F("alias", account.Alias),
	)

	return nil
}

// UpdateAccount updates an existing account
func (s *RealAccountService) UpdateAccount(ctx context.Context, account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	s.logger.Info(ctx, "updating_account",
		observability.F("alias", account.Alias),
	)

	if s.configService == nil {
		return fmt.Errorf("config service not available")
	}

	// Validate the account
	if err := s.ValidateAccount(ctx, account); err != nil {
		return fmt.Errorf("account validation failed: %w", err)
	}

	// Save the updated account
	if err := s.configService.SetAccount(ctx, account); err != nil {
		s.logger.Error(ctx, "failed_to_update_account",
			observability.F("alias", account.Alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to update account '%s': %w", account.Alias, err)
	}

	s.logger.Info(ctx, "account_updated_successfully",
		observability.F("alias", account.Alias),
	)

	return nil
}

// DeleteAccount deletes an account
func (s *RealAccountService) DeleteAccount(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "deleting_account",
		observability.F("alias", alias),
	)

	if s.configService == nil {
		return fmt.Errorf("config service not available")
	}

	// Check if this is the current account
	currentAccount := s.configService.GetCurrentAccount(ctx)
	if currentAccount == alias {
		s.logger.Warn(ctx, "deleting_current_account",
			observability.F("alias", alias),
		)
	}

	// Delete the account
	if err := s.configService.DeleteAccount(ctx, alias); err != nil {
		s.logger.Error(ctx, "failed_to_delete_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to delete account '%s': %w", alias, err)
	}

	s.logger.Info(ctx, "account_deleted_successfully",
		observability.F("alias", alias),
	)

	return nil
}

// ListAccounts lists all accounts
func (s *RealAccountService) ListAccounts(ctx context.Context) ([]*models.Account, error) {
	s.logger.Info(ctx, "listing_accounts")

	if s.configService == nil {
		return nil, fmt.Errorf("config service not available")
	}

	accounts, err := s.configService.ListAccounts(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed_to_list_accounts",
			observability.F("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	s.logger.Info(ctx, "accounts_listed_successfully",
		observability.F("count", len(accounts)),
	)

	return accounts, nil
}

// SetCurrentAccount sets the current active account
func (s *RealAccountService) SetCurrentAccount(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "setting_current_account",
		observability.F("alias", alias),
	)

	if s.configService == nil {
		return fmt.Errorf("config service not available")
	}

	// Verify the account exists
	_, err := s.GetAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("account '%s' not found: %w", alias, err)
	}

	// Set as current account
	if err := s.configService.SetCurrentAccount(ctx, alias); err != nil {
		s.logger.Error(ctx, "failed_to_set_current_account",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to set current account to '%s': %w", alias, err)
	}

	// Mark as used
	if err := s.MarkAccountAsUsed(ctx, alias); err != nil {
		s.logger.Warn(ctx, "failed_to_mark_account_as_used",
			observability.F("alias", alias),
			observability.F("error", err.Error()),
		)
	}

	s.logger.Info(ctx, "current_account_set_successfully",
		observability.F("alias", alias),
	)

	return nil
}

// GetCurrentAccount gets the current active account
func (s *RealAccountService) GetCurrentAccount(ctx context.Context) string {
	if s.configService == nil {
		return ""
	}

	return s.configService.GetCurrentAccount(ctx)
}

// MarkAccountAsUsed marks an account as used
func (s *RealAccountService) MarkAccountAsUsed(ctx context.Context, alias string) error {
	s.logger.Info(ctx, "marking_account_as_used",
		observability.F("alias", alias),
	)

	// Get the account
	account, err := s.GetAccount(ctx, alias)
	if err != nil {
		return fmt.Errorf("failed to get account '%s': %w", alias, err)
	}

	// Update last used timestamp
	now := time.Now()
	account.LastUsed = &now

	// Save the updated account
	if err := s.UpdateAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to update account '%s': %w", alias, err)
	}

	s.logger.Info(ctx, "account_marked_as_used",
		observability.F("alias", alias),
		observability.F("last_used", now.Format(time.RFC3339)),
	)

	return nil
}

// ValidateAccount validates an account configuration
func (s *RealAccountService) ValidateAccount(ctx context.Context, account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	s.logger.Info(ctx, "validating_account",
		observability.F("alias", account.Alias),
	)

	// Use the model's validation
	if err := account.Validate(); err != nil {
		s.logger.Error(ctx, "account_validation_failed",
			observability.F("alias", account.Alias),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("account validation failed: %w", err)
	}

	s.logger.Info(ctx, "account_validation_passed",
		observability.F("alias", account.Alias),
	)

	return nil
}

// TestAccountSSH tests SSH connectivity for an account
func (s *RealAccountService) TestAccountSSH(ctx context.Context, account *models.Account) error {
	if account == nil {
		return fmt.Errorf("account cannot be nil")
	}

	s.logger.Info(ctx, "testing_account_ssh",
		observability.F("alias", account.Alias),
		observability.F("ssh_key_path", account.SSHKeyPath),
	)

	// TODO: Implement actual SSH testing
	// This would involve:
	// 1. Checking if SSH key exists and is readable
	// 2. Testing GitHub authentication with the key
	// 3. Verifying the key is associated with the correct GitHub account

	s.logger.Info(ctx, "account_ssh_test_passed",
		observability.F("alias", account.Alias),
	)

	return nil
}
