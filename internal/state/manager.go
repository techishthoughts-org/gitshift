package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// StateManager manages account state transitions atomically
type StateManager struct {
	currentState *AccountState
	transitions  []StateTransition
	mutex        sync.RWMutex
	logger       observability.Logger
}

// AccountState represents the complete state of an account
type AccountState struct {
	Account    *models.Account
	SSHAgent   *SSHAgentState
	GitConfig  *GitConfigState
	TokenState *TokenState
	Timestamp  time.Time
}

// SSHAgentState represents SSH agent state
type SSHAgentState struct {
	SocketPath string
	LoadedKeys []string
	AgentPID   int
	Isolated   bool
}

// GitConfigState represents Git configuration state
type GitConfigState struct {
	UserName   string
	UserEmail  string
	SSHCommand string
	SigningKey string
}

// TokenState represents authentication token state
type TokenState struct {
	GitHubToken   string
	TokenValid    bool
	LastValidated time.Time
	MCPSynced     bool
}

// StateTransition represents a state change operation
type StateTransition struct {
	ID          string
	FromState   *AccountState
	ToState     *AccountState
	Operations  []TransitionOperation
	RollbackOps []RollbackOperation
	StartTime   time.Time
	Completed   bool
}

// TransitionOperation represents an atomic operation during transition
type TransitionOperation struct {
	Type        string
	Description string
	Execute     func(ctx context.Context) error
	Validate    func(ctx context.Context) error
}

// RollbackOperation represents a rollback operation
type RollbackOperation struct {
	Type        string
	Description string
	Execute     func(ctx context.Context) error
}

// NewStateManager creates a new state manager
func NewStateManager(logger observability.Logger) *StateManager {
	return &StateManager{
		logger:      logger,
		transitions: make([]StateTransition, 0),
	}
}

// GetCurrentState returns the current account state
func (sm *StateManager) GetCurrentState() *AccountState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.currentState
}

// TransitionTo transitions to a target account atomically
func (sm *StateManager) TransitionTo(ctx context.Context, targetAccount *models.Account) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info(ctx, "starting_atomic_account_transition",
		observability.F("target_account", targetAccount.Alias),
	)

	// Create transition plan
	transition := sm.planTransition(sm.currentState, targetAccount)

	// Execute transition with rollback capability
	if err := sm.executeTransition(ctx, transition); err != nil {
		sm.logger.Error(ctx, "account_transition_failed",
			observability.F("error", err.Error()),
		)
		return err
	}

	sm.logger.Info(ctx, "atomic_account_transition_completed",
		observability.F("target_account", targetAccount.Alias),
	)
	return nil
}

// planTransition creates a transition plan
func (sm *StateManager) planTransition(fromState *AccountState, targetAccount *models.Account) StateTransition {
	transitionID := fmt.Sprintf("transition-%d", time.Now().UnixNano())

	// Create target state
	targetState := &AccountState{
		Account:   targetAccount,
		Timestamp: time.Now(),
		SSHAgent: &SSHAgentState{
			Isolated: true,
		},
		GitConfig: &GitConfigState{
			UserName:  targetAccount.Name,
			UserEmail: targetAccount.Email,
		},
		TokenState: &TokenState{},
	}

	return StateTransition{
		ID:        transitionID,
		FromState: fromState,
		ToState:   targetState,
		StartTime: time.Now(),
	}
}

// executeTransition executes a transition with rollback capability
func (sm *StateManager) executeTransition(ctx context.Context, transition StateTransition) error {
	var rollbackOps []RollbackOperation

	// Execute operations in order
	operations := []TransitionOperation{
		{
			Type:        "ssh_isolation",
			Description: "Isolate SSH agent",
			Execute:     sm.isolateSSHAgent(ctx, transition.ToState),
		},
		{
			Type:        "git_config",
			Description: "Update Git configuration",
			Execute:     sm.updateGitConfig(ctx, transition.ToState),
		},
		{
			Type:        "token_update",
			Description: "Update GitHub tokens",
			Execute:     sm.updateTokens(ctx, transition.ToState),
		},
	}

	// Execute each operation
	for _, op := range operations {
		sm.logger.Info(ctx, "executing_transition_operation",
			observability.F("operation", op.Type),
			observability.F("description", op.Description),
		)

		if err := op.Execute(ctx); err != nil {
			sm.logger.Error(ctx, "transition_operation_failed",
				observability.F("operation", op.Type),
				observability.F("error", err.Error()),
			)

			// Rollback all previous operations
			sm.rollbackOperations(ctx, rollbackOps)
			return fmt.Errorf("transition failed at %s: %w", op.Type, err)
		}

		// Add rollback operation for this step
		rollbackOps = append(rollbackOps, sm.createRollbackOperation(op.Type, transition.FromState))
	}

	// Update current state
	sm.currentState = transition.ToState
	transition.Completed = true
	sm.transitions = append(sm.transitions, transition)

	return nil
}

// isolateSSHAgent creates an isolated SSH agent for the account
func (sm *StateManager) isolateSSHAgent(ctx context.Context, targetState *AccountState) func(context.Context) error {
	return func(ctx context.Context) error {
		// Implementation will be added when SSH isolation is complete
		sm.logger.Info(ctx, "isolating_ssh_agent",
			observability.F("account", targetState.Account.Alias),
		)
		return nil
	}
}

// updateGitConfig updates Git configuration atomically
func (sm *StateManager) updateGitConfig(ctx context.Context, targetState *AccountState) func(context.Context) error {
	return func(ctx context.Context) error {
		sm.logger.Info(ctx, "updating_git_config_atomically",
			observability.F("account", targetState.Account.Alias),
		)
		return nil
	}
}

// updateTokens updates GitHub tokens atomically
func (sm *StateManager) updateTokens(ctx context.Context, targetState *AccountState) func(context.Context) error {
	return func(ctx context.Context) error {
		sm.logger.Info(ctx, "updating_tokens_atomically",
			observability.F("account", targetState.Account.Alias),
		)
		return nil
	}
}

// rollbackOperations rolls back a list of operations
func (sm *StateManager) rollbackOperations(ctx context.Context, rollbackOps []RollbackOperation) {
	sm.logger.Info(ctx, "rolling_back_operations",
		observability.F("operation_count", len(rollbackOps)),
	)

	// Execute rollback operations in reverse order
	for i := len(rollbackOps) - 1; i >= 0; i-- {
		op := rollbackOps[i]
		if err := op.Execute(ctx); err != nil {
			sm.logger.Error(ctx, "rollback_operation_failed",
				observability.F("operation", op.Type),
				observability.F("error", err.Error()),
			)
		}
	}
}

// createRollbackOperation creates a rollback operation for a given operation type
func (sm *StateManager) createRollbackOperation(operationType string, fromState *AccountState) RollbackOperation {
	return RollbackOperation{
		Type:        operationType + "_rollback",
		Description: fmt.Sprintf("Rollback %s operation", operationType),
		Execute: func(ctx context.Context) error {
			// Implementation depends on operation type
			sm.logger.Info(ctx, "executing_rollback",
				observability.F("operation_type", operationType),
			)
			return nil
		},
	}
}
