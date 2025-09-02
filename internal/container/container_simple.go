package container

import (
	"context"
	"sync"

	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// SimpleContainer manages all service dependencies without complex type assertions
type SimpleContainer struct {
	mu sync.RWMutex

	// Core services as interface{} to avoid type issues
	configService     interface{}
	accountService    interface{}
	sshService        interface{}
	gitService        interface{}
	githubService     interface{}
	healthService     interface{}
	validationService interface{}

	// Infrastructure
	logger observability.Logger
}

// NewSimpleContainer creates a new simple service container
func NewSimpleContainer() *SimpleContainer {
	container := &SimpleContainer{}
	return container
}

// GetLogger returns the logger instance
func (c *SimpleContainer) GetLogger() observability.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logger == nil {
		c.logger = observability.NewDefaultLogger()
	}

	return c.logger
}

// SetLogger sets the logger instance
func (c *SimpleContainer) SetLogger(logger observability.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logger = logger
}

// GetConfigService returns the config service instance
func (c *SimpleContainer) GetConfigService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configService
}

// SetConfigService sets the config service instance
func (c *SimpleContainer) SetConfigService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configService = service
}

// GetAccountService returns the account service instance
func (c *SimpleContainer) GetAccountService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountService
}

// SetAccountService sets the account service instance
func (c *SimpleContainer) SetAccountService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accountService = service
}

// GetSSHService returns the SSH service instance
func (c *SimpleContainer) GetSSHService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshService
}

// SetSSHService sets the SSH service instance
func (c *SimpleContainer) SetSSHService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sshService = service
}

// GetGitService returns the Git service instance
func (c *SimpleContainer) GetGitService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gitService
}

// SetGitService sets the Git service instance
func (c *SimpleContainer) SetGitService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gitService = service
}

// GetGitHubService returns the GitHub service instance
func (c *SimpleContainer) GetGitHubService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.githubService
}

// SetGitHubService sets the GitHub service instance
func (c *SimpleContainer) SetGitHubService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.githubService = service
}

// GetHealthService returns the health service instance
func (c *SimpleContainer) GetHealthService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthService
}

// SetHealthService sets the health service instance
func (c *SimpleContainer) SetHealthService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthService = service
}

// GetValidationService returns the validation service instance
func (c *SimpleContainer) GetValidationService() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.validationService
}

// SetValidationService sets the validation service instance
func (c *SimpleContainer) SetValidationService(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validationService = service
}

// Initialize initializes all services
func (c *SimpleContainer) Initialize(ctx context.Context) error {
	logger := c.GetLogger()
	logger.Info(ctx, "initializing_simple_service_container")

	// Initialize the config service
	configPath := "/Users/arthurcosta/.config/gitpersona"
	configService := services.NewRealConfigService(configPath, logger)
	c.SetConfigService(configService)

	// Initialize the Git config service
	gitConfigService := services.NewGitConfigService(logger)
	c.SetGitService(gitConfigService)

	// TODO: Initialize other services as they are implemented
	logger.Info(ctx, "simple_service_container_initialized")
	return nil
}

// Shutdown gracefully shuts down all services
func (c *SimpleContainer) Shutdown(ctx context.Context) error {
	logger := c.GetLogger()
	logger.Info(ctx, "shutting_down_simple_service_container")
	logger.Info(ctx, "simple_service_container_shutdown_complete")
	return nil
}

// Global simple container instance
var globalSimpleContainer *SimpleContainer
var globalSimpleContainerOnce sync.Once

// GetGlobalSimpleContainer returns the global simple service container
func GetGlobalSimpleContainer() *SimpleContainer {
	globalSimpleContainerOnce.Do(func() {
		globalSimpleContainer = NewSimpleContainer()
	})
	return globalSimpleContainer
}

// InitializeGlobalSimpleContainer initializes the global simple service container
func InitializeGlobalSimpleContainer(ctx context.Context) error {
	return GetGlobalSimpleContainer().Initialize(ctx)
}

// ShutdownGlobalSimpleContainer shuts down the global simple service container
func ShutdownGlobalSimpleContainer(ctx context.Context) error {
	return GetGlobalSimpleContainer().Shutdown(ctx)
}
