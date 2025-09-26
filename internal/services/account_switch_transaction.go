package services

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// AccountSwitchTransaction provides atomic account switching with rollback capability
type AccountSwitchTransaction struct {
	logger         observability.Logger
	transactionID  string
	sourceAccount  *models.Account
	targetAccount  *models.Account
	steps          []SwitchStep
	completedSteps []SwitchStep
	rollbackSteps  []SwitchStep
	startTime      time.Time
	state          TransactionState
	mutex          sync.RWMutex
	context        context.Context

	// Service dependencies
	tokenService      *IsolatedTokenService
	sshManager        *IsolatedSSHManager
	gitService        GitService
	configService     ConfigurationService
	validationService ValidationService

	// Transaction options
	options *TransactionOptions
}

// TransactionState represents the current state of the transaction
type TransactionState string

const (
	TransactionStateInitialized TransactionState = "initialized"
	TransactionStateInProgress  TransactionState = "in_progress"
	TransactionStateCompleted   TransactionState = "completed"
	TransactionStateFailed      TransactionState = "failed"
	TransactionStateRolledBack  TransactionState = "rolled_back"
)

// TransactionOptions configures transaction behavior
type TransactionOptions struct {
	StrictValidation     bool          `json:"strict_validation"`
	RollbackOnFailure    bool          `json:"rollback_on_failure"`
	ValidateBeforeSwitch bool          `json:"validate_before_switch"`
	ValidateAfterSwitch  bool          `json:"validate_after_switch"`
	Timeout              time.Duration `json:"timeout"`
	ConcurrentSteps      bool          `json:"concurrent_steps"`
	SkipSSHValidation    bool          `json:"skip_ssh_validation"`
	SkipTokenValidation  bool          `json:"skip_token_validation"`
}

// SwitchStep represents a single step in the account switching process
type SwitchStep interface {
	GetName() string
	GetDescription() string
	Execute(ctx context.Context, transaction *AccountSwitchTransaction) error
	Rollback(ctx context.Context, transaction *AccountSwitchTransaction) error
	CanRollback() bool
	GetDependencies() []string
	Validate(ctx context.Context, transaction *AccountSwitchTransaction) error
}

// SwitchStepResult represents the result of executing a switch step
type SwitchStepResult struct {
	StepName string                 `json:"step_name"`
	Success  bool                   `json:"success"`
	Duration time.Duration          `json:"duration"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TransactionResult represents the final result of the transaction
type TransactionResult struct {
	TransactionID    string             `json:"transaction_id"`
	Success          bool               `json:"success"`
	TotalDuration    time.Duration      `json:"total_duration"`
	SourceAccount    string             `json:"source_account"`
	TargetAccount    string             `json:"target_account"`
	CompletedSteps   []SwitchStepResult `json:"completed_steps"`
	FailedStep       *SwitchStepResult  `json:"failed_step,omitempty"`
	RollbackSteps    []SwitchStepResult `json:"rollback_steps,omitempty"`
	FinalState       TransactionState   `json:"final_state"`
	ValidationErrors []string           `json:"validation_errors,omitempty"`
}

// NewAccountSwitchTransaction creates a new atomic account switching transaction
func NewAccountSwitchTransaction(
	ctx context.Context,
	logger observability.Logger,
	sourceAccount *models.Account,
	targetAccount *models.Account,
	tokenService *IsolatedTokenService,
	sshManager *IsolatedSSHManager,
	options *TransactionOptions,
) *AccountSwitchTransaction {

	if options == nil {
		options = &TransactionOptions{
			StrictValidation:     true,
			RollbackOnFailure:    true,
			ValidateBeforeSwitch: true,
			ValidateAfterSwitch:  true,
			Timeout:              5 * time.Minute,
			ConcurrentSteps:      false,
			SkipSSHValidation:    false,
			SkipTokenValidation:  false,
		}
	}

	transactionID := fmt.Sprintf("switch-%d", time.Now().UnixNano())

	return &AccountSwitchTransaction{
		logger:         logger,
		transactionID:  transactionID,
		sourceAccount:  sourceAccount,
		targetAccount:  targetAccount,
		steps:          []SwitchStep{},
		completedSteps: []SwitchStep{},
		rollbackSteps:  []SwitchStep{},
		startTime:      time.Now(),
		state:          TransactionStateInitialized,
		context:        ctx,
		tokenService:   tokenService,
		sshManager:     sshManager,
		options:        options,
	}
}

// AddStep adds a switch step to the transaction
func (t *AccountSwitchTransaction) AddStep(step SwitchStep) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.steps = append(t.steps, step)

	t.logger.Info(t.context, "switch_step_added_to_transaction",
		observability.F("transaction_id", t.transactionID),
		observability.F("step_name", step.GetName()),
		observability.F("total_steps", len(t.steps)),
	)
}

// Execute executes the transaction atomically
func (t *AccountSwitchTransaction) Execute() (*TransactionResult, error) {
	t.mutex.Lock()
	if t.state != TransactionStateInitialized {
		t.mutex.Unlock()
		return nil, fmt.Errorf("transaction already executed or in progress")
	}
	t.state = TransactionStateInProgress
	t.mutex.Unlock()

	t.logger.Info(t.context, "starting_account_switch_transaction",
		observability.F("transaction_id", t.transactionID),
		observability.F("source_account", t.getAccountAlias(t.sourceAccount)),
		observability.F("target_account", t.targetAccount.Alias),
		observability.F("total_steps", len(t.steps)),
	)

	result := &TransactionResult{
		TransactionID:  t.transactionID,
		SourceAccount:  t.getAccountAlias(t.sourceAccount),
		TargetAccount:  t.targetAccount.Alias,
		CompletedSteps: []SwitchStepResult{},
		RollbackSteps:  []SwitchStepResult{},
	}

	// Set timeout context
	ctx, cancel := context.WithTimeout(t.context, t.options.Timeout)
	defer cancel()

	// Pre-validation
	if t.options.ValidateBeforeSwitch {
		if err := t.validateBeforeSwitch(ctx); err != nil {
			result.Success = false
			result.FinalState = TransactionStateFailed
			result.ValidationErrors = []string{err.Error()}
			t.setState(TransactionStateFailed)
			return result, fmt.Errorf("pre-validation failed: %w", err)
		}
	}

	// Execute steps
	if t.options.ConcurrentSteps {
		err := t.executeConcurrentSteps(ctx, result)
		if err != nil {
			t.performRollback(ctx, result)
			return result, err
		}
	} else {
		err := t.executeSequentialSteps(ctx, result)
		if err != nil {
			t.performRollback(ctx, result)
			return result, err
		}
	}

	// Post-validation
	if t.options.ValidateAfterSwitch {
		if err := t.validateAfterSwitch(ctx); err != nil {
			t.logger.Error(ctx, "post_validation_failed",
				observability.F("transaction_id", t.transactionID),
				observability.F("error", err.Error()),
			)
			result.ValidationErrors = append(result.ValidationErrors, err.Error())
			t.performRollback(ctx, result)
			return result, fmt.Errorf("post-validation failed: %w", err)
		}
	}

	// Success
	t.setState(TransactionStateCompleted)
	result.Success = true
	result.FinalState = TransactionStateCompleted
	result.TotalDuration = time.Since(t.startTime)

	t.logger.Info(ctx, "account_switch_transaction_completed_successfully",
		observability.F("transaction_id", t.transactionID),
		observability.F("target_account", t.targetAccount.Alias),
		observability.F("duration", result.TotalDuration),
		observability.F("completed_steps", len(result.CompletedSteps)),
	)

	return result, nil
}

// executeSequentialSteps executes steps one by one
func (t *AccountSwitchTransaction) executeSequentialSteps(ctx context.Context, result *TransactionResult) error {
	for i, step := range t.steps {
		stepStartTime := time.Now()

		t.logger.Info(ctx, "executing_switch_step",
			observability.F("transaction_id", t.transactionID),
			observability.F("step_name", step.GetName()),
			observability.F("step_index", i+1),
			observability.F("total_steps", len(t.steps)),
		)

		// Validate step if strict validation is enabled
		if t.options.StrictValidation {
			if err := step.Validate(ctx, t); err != nil {
				stepResult := SwitchStepResult{
					StepName: step.GetName(),
					Success:  false,
					Duration: time.Since(stepStartTime),
					Error:    fmt.Sprintf("validation failed: %v", err),
				}
				result.FailedStep = &stepResult
				t.setState(TransactionStateFailed)
				return fmt.Errorf("step validation failed: %w", err)
			}
		}

		// Execute step
		if err := step.Execute(ctx, t); err != nil {
			stepResult := SwitchStepResult{
				StepName: step.GetName(),
				Success:  false,
				Duration: time.Since(stepStartTime),
				Error:    err.Error(),
			}
			result.FailedStep = &stepResult
			t.setState(TransactionStateFailed)
			return fmt.Errorf("step execution failed: %w", err)
		}

		// Record successful step
		stepResult := SwitchStepResult{
			StepName: step.GetName(),
			Success:  true,
			Duration: time.Since(stepStartTime),
		}
		result.CompletedSteps = append(result.CompletedSteps, stepResult)

		t.mutex.Lock()
		t.completedSteps = append(t.completedSteps, step)
		t.mutex.Unlock()

		t.logger.Info(ctx, "switch_step_completed_successfully",
			observability.F("transaction_id", t.transactionID),
			observability.F("step_name", step.GetName()),
			observability.F("duration", stepResult.Duration),
		)
	}

	return nil
}

// executeConcurrentSteps executes independent steps concurrently
func (t *AccountSwitchTransaction) executeConcurrentSteps(ctx context.Context, result *TransactionResult) error {
	// For now, implement sequential execution
	// TODO: Add dependency analysis and concurrent execution
	return t.executeSequentialSteps(ctx, result)
}

// performRollback performs rollback if enabled
func (t *AccountSwitchTransaction) performRollback(ctx context.Context, result *TransactionResult) {
	if !t.options.RollbackOnFailure {
		t.logger.Info(ctx, "rollback_skipped_by_configuration",
			observability.F("transaction_id", t.transactionID),
		)
		return
	}

	t.logger.Info(ctx, "starting_transaction_rollback",
		observability.F("transaction_id", t.transactionID),
		observability.F("completed_steps_to_rollback", len(t.completedSteps)),
	)

	// Rollback completed steps in reverse order
	t.mutex.RLock()
	stepsToRollback := make([]SwitchStep, len(t.completedSteps))
	copy(stepsToRollback, t.completedSteps)
	t.mutex.RUnlock()

	for i := len(stepsToRollback) - 1; i >= 0; i-- {
		step := stepsToRollback[i]
		rollbackStartTime := time.Now()

		if !step.CanRollback() {
			t.logger.Warn(ctx, "step_cannot_be_rolled_back",
				observability.F("transaction_id", t.transactionID),
				observability.F("step_name", step.GetName()),
			)
			continue
		}

		t.logger.Info(ctx, "rolling_back_step",
			observability.F("transaction_id", t.transactionID),
			observability.F("step_name", step.GetName()),
		)

		if err := step.Rollback(ctx, t); err != nil {
			t.logger.Error(ctx, "step_rollback_failed",
				observability.F("transaction_id", t.transactionID),
				observability.F("step_name", step.GetName()),
				observability.F("error", err.Error()),
			)

			rollbackResult := SwitchStepResult{
				StepName: step.GetName(),
				Success:  false,
				Duration: time.Since(rollbackStartTime),
				Error:    err.Error(),
			}
			result.RollbackSteps = append(result.RollbackSteps, rollbackResult)
		} else {
			rollbackResult := SwitchStepResult{
				StepName: step.GetName(),
				Success:  true,
				Duration: time.Since(rollbackStartTime),
			}
			result.RollbackSteps = append(result.RollbackSteps, rollbackResult)

			t.logger.Info(ctx, "step_rolled_back_successfully",
				observability.F("transaction_id", t.transactionID),
				observability.F("step_name", step.GetName()),
				observability.F("duration", rollbackResult.Duration),
			)
		}
	}

	t.setState(TransactionStateRolledBack)
	result.FinalState = TransactionStateRolledBack

	t.logger.Info(ctx, "transaction_rollback_completed",
		observability.F("transaction_id", t.transactionID),
		observability.F("rolled_back_steps", len(result.RollbackSteps)),
	)
}

// validateBeforeSwitch performs pre-switch validation
func (t *AccountSwitchTransaction) validateBeforeSwitch(ctx context.Context) error {
	t.logger.Info(ctx, "performing_pre_switch_validation",
		observability.F("transaction_id", t.transactionID),
	)

	// Validate target account exists and is complete
	if err := t.targetAccount.Validate(); err != nil {
		return fmt.Errorf("target account validation failed: %w", err)
	}

	// Validate SSH key exists
	if t.targetAccount.SSHKeyPath != "" && !t.options.SkipSSHValidation {
		if _, err := os.Stat(t.targetAccount.SSHKeyPath); err != nil {
			return fmt.Errorf("SSH key validation failed: %w", err)
		}
	}

	// Validate token exists and is valid
	if !t.options.SkipTokenValidation && t.tokenService != nil {
		if err := t.tokenService.ValidateTokenIsolation(ctx, t.targetAccount.Alias, t.targetAccount.GitHubUsername); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}
	}

	return nil
}

// validateAfterSwitch performs post-switch validation
func (t *AccountSwitchTransaction) validateAfterSwitch(ctx context.Context) error {
	t.logger.Info(ctx, "performing_post_switch_validation",
		observability.F("transaction_id", t.transactionID),
	)

	// TODO: Implement comprehensive post-switch validation
	// - Verify Git config is set correctly
	// - Verify SSH agent has correct key loaded
	// - Verify token is accessible
	// - Test GitHub connectivity

	return nil
}

// setState updates the transaction state thread-safely
func (t *AccountSwitchTransaction) setState(state TransactionState) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.state = state
}

// GetState returns the current transaction state
func (t *AccountSwitchTransaction) GetState() TransactionState {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.state
}

// GetTransactionID returns the transaction ID
func (t *AccountSwitchTransaction) GetTransactionID() string {
	return t.transactionID
}

// GetTargetAccount returns the target account
func (t *AccountSwitchTransaction) GetTargetAccount() *models.Account {
	return t.targetAccount
}

// GetSourceAccount returns the source account
func (t *AccountSwitchTransaction) GetSourceAccount() *models.Account {
	return t.sourceAccount
}

// GetTokenService returns the token service
func (t *AccountSwitchTransaction) GetTokenService() *IsolatedTokenService {
	return t.tokenService
}

// GetSSHManager returns the SSH manager
func (t *AccountSwitchTransaction) GetSSHManager() *IsolatedSSHManager {
	return t.sshManager
}

// getAccountAlias returns the alias for an account (handles nil case)
func (t *AccountSwitchTransaction) getAccountAlias(account *models.Account) string {
	if account == nil {
		return "none"
	}
	return account.Alias
}

// GetOptions returns the transaction options
func (t *AccountSwitchTransaction) GetOptions() *TransactionOptions {
	return t.options
}

// IsCompleted returns true if the transaction is completed
func (t *AccountSwitchTransaction) IsCompleted() bool {
	state := t.GetState()
	return state == TransactionStateCompleted || state == TransactionStateFailed || state == TransactionStateRolledBack
}

// GetDuration returns the duration of the transaction
func (t *AccountSwitchTransaction) GetDuration() time.Duration {
	return time.Since(t.startTime)
}
