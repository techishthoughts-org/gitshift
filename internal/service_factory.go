package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// ServiceFactory provides centralized service creation and lifecycle management
type ServiceFactory struct {
	logger    observability.Logger
	container *ServiceContainer
	mu        sync.RWMutex
	started   bool
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name      string    `json:"name"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
	Error     string    `json:"error,omitempty"`
	Uptime    int64     `json:"uptime_seconds"`
}

// ServiceMetrics contains service performance metrics
type ServiceMetrics struct {
	Name            string        `json:"name"`
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastRequestTime time.Time     `json:"last_request_time"`
}

// GlobalServiceFactory is the singleton instance
var (
	globalFactory     *ServiceFactory
	globalFactoryOnce sync.Once
)

// GetGlobalServiceFactory returns the singleton service factory
func GetGlobalServiceFactory() *ServiceFactory {
	globalFactoryOnce.Do(func() {
		logger := observability.NewLogger(observability.LogLevelInfo)
		globalFactory = NewServiceFactory(logger)
	})
	return globalFactory
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(logger observability.Logger) *ServiceFactory {
	return &ServiceFactory{
		logger:  logger,
		started: false,
	}
}

// Initialize creates and configures all services with proper dependency injection
func (sf *ServiceFactory) Initialize(ctx context.Context) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.started {
		return fmt.Errorf("service factory already initialized")
	}

	sf.logger.Info(ctx, "initializing_service_factory")

	// Create service container with dependency injection
	sf.container = NewServiceContainer(sf.logger)

	// Validate all services are properly initialized
	if err := sf.validateServices(ctx); err != nil {
		return fmt.Errorf("service validation failed: %w", err)
	}

	// Perform initial health check
	if err := sf.container.HealthCheck(ctx); err != nil {
		sf.logger.Warn(ctx, "initial_health_check_failed",
			observability.F("error", err.Error()),
		)
		// Don't fail initialization on health check failure
	}

	sf.started = true
	sf.logger.Info(ctx, "service_factory_initialized_successfully")

	return nil
}

// GetServices returns the core services container
func (sf *ServiceFactory) GetServices() *CoreServices {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if !sf.started || sf.container == nil {
		// Auto-initialize if not done
		ctx := context.Background()
		if err := sf.Initialize(ctx); err != nil {
			sf.logger.Error(ctx, "auto_initialization_failed",
				observability.F("error", err.Error()),
			)
			return nil
		}
	}

	return sf.container.GetServices()
}

// HealthCheck performs comprehensive health check on all services
func (sf *ServiceFactory) HealthCheck(ctx context.Context) ([]*ServiceHealth, error) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if !sf.started {
		return nil, fmt.Errorf("service factory not initialized")
	}

	sf.logger.Info(ctx, "performing_service_health_check")

	healthChecks := []*ServiceHealth{
		sf.checkServiceHealth(ctx, "Account", func() error {
			_, err := sf.container.services.Account.ListAccounts(ctx)
			return err
		}),
		sf.checkServiceHealth(ctx, "SSH", func() error {
			_, err := sf.container.services.SSH.ListKeys(ctx)
			return err
		}),
		sf.checkServiceHealth(ctx, "Git", func() error {
			_, err := sf.container.services.Git.GetCurrentConfig(ctx)
			return err
		}),
		sf.checkServiceHealth(ctx, "System", func() error {
			return sf.container.services.System.PerformHealthCheck(ctx)
		}),
	}

	// GitHub health check is more complex due to authentication
	githubHealth := sf.checkGitHubHealth(ctx)
	healthChecks = append(healthChecks, githubHealth)

	sf.logger.Info(ctx, "service_health_check_completed",
		observability.F("services_checked", len(healthChecks)),
	)

	return healthChecks, nil
}

// GetMetrics returns performance metrics for all services
func (sf *ServiceFactory) GetMetrics(ctx context.Context) ([]*ServiceMetrics, error) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if !sf.started {
		return nil, fmt.Errorf("service factory not initialized")
	}

	// TODO: Implement actual metrics collection
	// For now, return placeholder metrics
	metrics := []*ServiceMetrics{
		{
			Name:            "Account",
			RequestCount:    0,
			ErrorCount:      0,
			AverageLatency:  0,
			LastRequestTime: time.Time{},
		},
		{
			Name:            "SSH",
			RequestCount:    0,
			ErrorCount:      0,
			AverageLatency:  0,
			LastRequestTime: time.Time{},
		},
		{
			Name:            "Git",
			RequestCount:    0,
			ErrorCount:      0,
			AverageLatency:  0,
			LastRequestTime: time.Time{},
		},
		{
			Name:            "GitHub",
			RequestCount:    0,
			ErrorCount:      0,
			AverageLatency:  0,
			LastRequestTime: time.Time{},
		},
		{
			Name:            "System",
			RequestCount:    0,
			ErrorCount:      0,
			AverageLatency:  0,
			LastRequestTime: time.Time{},
		},
	}

	return metrics, nil
}

// Shutdown gracefully shuts down all services
func (sf *ServiceFactory) Shutdown(ctx context.Context) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.started {
		return nil
	}

	sf.logger.Info(ctx, "shutting_down_service_factory")

	// Perform cleanup operations
	// TODO: Add service-specific cleanup if needed

	sf.started = false
	sf.container = nil

	sf.logger.Info(ctx, "service_factory_shutdown_completed")
	return nil
}

// Reset recreates all services (useful for testing)
func (sf *ServiceFactory) Reset(ctx context.Context) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.logger.Info(ctx, "resetting_service_factory")

	if sf.started {
		if err := sf.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown before reset: %w", err)
		}
	}

	return sf.Initialize(ctx)
}

// IsHealthy returns true if all critical services are healthy
func (sf *ServiceFactory) IsHealthy(ctx context.Context) bool {
	healthChecks, err := sf.HealthCheck(ctx)
	if err != nil {
		return false
	}

	for _, health := range healthChecks {
		// GitHub is optional, so don't fail on GitHub issues
		if health.Name == "GitHub" {
			continue
		}
		if !health.Healthy {
			return false
		}
	}

	return true
}

// Helper methods

func (sf *ServiceFactory) validateServices(ctx context.Context) error {
	services := sf.container.GetServices()

	if services.Account == nil {
		return fmt.Errorf("AccountManager not initialized")
	}
	if services.SSH == nil {
		return fmt.Errorf("SSHManager not initialized")
	}
	if services.Git == nil {
		return fmt.Errorf("GitManager not initialized")
	}
	if services.GitHub == nil {
		return fmt.Errorf("GitHubManager not initialized")
	}
	if services.System == nil {
		return fmt.Errorf("SystemManager not initialized")
	}

	sf.logger.Info(ctx, "all_services_validated_successfully")
	return nil
}

func (sf *ServiceFactory) checkServiceHealth(ctx context.Context, name string, healthFunc func() error) *ServiceHealth {
	start := time.Now()

	health := &ServiceHealth{
		Name:      name,
		LastCheck: start,
		Uptime:    0, // TODO: Track actual uptime
	}

	if err := healthFunc(); err != nil {
		health.Healthy = false
		health.Error = err.Error()
	} else {
		health.Healthy = true
	}

	return health
}

func (sf *ServiceFactory) checkGitHubHealth(ctx context.Context) *ServiceHealth {
	health := &ServiceHealth{
		Name:      "GitHub",
		LastCheck: time.Now(),
		Healthy:   true, // Assume healthy unless we can test
		Uptime:    0,
	}

	// Try to get current account and test GitHub access
	if accounts, err := sf.container.services.Account.ListAccounts(ctx); err == nil && len(accounts) > 0 {
		// Test with first account that might have GitHub access
		for _, account := range accounts {
			if err := sf.container.services.GitHub.TestAPIAccess(ctx, account); err != nil {
				// GitHub access failed, but this might be expected (no token, etc.)
				health.Error = fmt.Sprintf("GitHub access test failed: %v", err)
				// Don't mark as unhealthy since GitHub access is optional
				break
			} else {
				// At least one account has working GitHub access
				health.Healthy = true
				health.Error = ""
				break
			}
		}
	}

	return health
}

// ServiceInitializer provides a simple interface for command initialization
type ServiceInitializer struct {
	factory *ServiceFactory
}

// NewServiceInitializer creates a service initializer
func NewServiceInitializer() *ServiceInitializer {
	return &ServiceInitializer{
		factory: GetGlobalServiceFactory(),
	}
}

// EnsureInitialized ensures services are initialized before use
func (si *ServiceInitializer) EnsureInitialized(ctx context.Context) (*CoreServices, error) {
	services := si.factory.GetServices()
	if services == nil {
		return nil, fmt.Errorf("failed to initialize services")
	}
	return services, nil
}

// MustInitialize initializes services and panics on failure (for testing)
func (si *ServiceInitializer) MustInitialize(ctx context.Context) *CoreServices {
	services, err := si.EnsureInitialized(ctx)
	if err != nil {
		panic(fmt.Sprintf("service initialization failed: %v", err))
	}
	return services
}
