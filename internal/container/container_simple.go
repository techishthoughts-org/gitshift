package container

import (
	"context"
	"sync"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/services"
)

// SimpleContainer manages all service dependencies without complex type assertions
type SimpleContainer struct {
	mu sync.RWMutex

	// Core services typed to useful interfaces for safer access
	configService      services.ConfigurationService
	accountService     services.AccountService
	sshService         services.SSHService
	sshAgentService    services.SSHAgentService
	gitService         services.GitConfigManager
	githubService      services.GitHubService
	githubTokenService services.GitHubTokenService
	healthService      services.HealthService
	validationService  services.ValidationService
	zshSecretsService  services.ZshSecretsService
	zshrcService       services.ZshrcService

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
func (c *SimpleContainer) GetConfigService() services.ConfigurationService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configService
}

// SetConfigService sets the config service instance
func (c *SimpleContainer) SetConfigService(service services.ConfigurationService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configService = service
}

// GetAccountService returns the account service instance
func (c *SimpleContainer) GetAccountService() services.AccountService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accountService
}

// SetAccountService sets the account service instance
func (c *SimpleContainer) SetAccountService(service services.AccountService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accountService = service
}

// GetSSHService returns the SSH service instance
func (c *SimpleContainer) GetSSHService() services.SSHService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshService
}

// SetSSHService sets the SSH service instance
func (c *SimpleContainer) SetSSHService(service services.SSHService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sshService = service
}

// GetSSHAgentService returns the SSH agent service instance
func (c *SimpleContainer) GetSSHAgentService() services.SSHAgentService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sshAgentService
}

// SetSSHAgentService sets the SSH agent service instance
func (c *SimpleContainer) SetSSHAgentService(service services.SSHAgentService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sshAgentService = service
}

// GetGitService returns the Git service instance
func (c *SimpleContainer) GetGitService() services.GitConfigManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gitService
}

// SetGitService sets the Git service instance
func (c *SimpleContainer) SetGitService(service services.GitConfigManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gitService = service
}

// GetGitHubService returns the GitHub service instance
func (c *SimpleContainer) GetGitHubService() services.GitHubService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.githubService
}

// SetGitHubService sets the GitHub service instance
func (c *SimpleContainer) SetGitHubService(service services.GitHubService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.githubService = service
}

// GetGitHubTokenService returns the GitHub token service instance
func (c *SimpleContainer) GetGitHubTokenService() services.GitHubTokenService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.githubTokenService
}

// SetGitHubTokenService sets the GitHub token service instance
func (c *SimpleContainer) SetGitHubTokenService(service services.GitHubTokenService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.githubTokenService = service
}

// GetHealthService returns the health service instance
func (c *SimpleContainer) GetHealthService() services.HealthService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthService
}

// SetHealthService sets the health service instance
func (c *SimpleContainer) SetHealthService(service services.HealthService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthService = service
}

// GetValidationService returns the validation service instance
func (c *SimpleContainer) GetValidationService() services.ValidationService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.validationService
}

// SetValidationService sets the validation service instance
func (c *SimpleContainer) SetValidationService(service services.ValidationService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validationService = service
}

// GetZshSecretsService returns the zsh secrets service instance
func (c *SimpleContainer) GetZshSecretsService() services.ZshSecretsService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.zshSecretsService
}

// SetZshSecretsService sets the zsh secrets service instance
func (c *SimpleContainer) SetZshSecretsService(service services.ZshSecretsService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.zshSecretsService = service
}

// GetZshrcService returns the zshrc service instance
func (c *SimpleContainer) GetZshrcService() services.ZshrcService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.zshrcService
}

// SetZshrcService sets the zshrc service instance
func (c *SimpleContainer) SetZshrcService(service services.ZshrcService) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.zshrcService = service
}

// Initialize initializes all services
func (c *SimpleContainer) Initialize(ctx context.Context) error {
	logger := c.GetLogger()
	logger.Info(ctx, "initializing_simple_service_container")

	// Initialize the config service
	configPath := "/Users/arthurcosta/.config/gitpersona"
	configService := services.NewRealConfigService(configPath, logger)
	c.SetConfigService(configService)

	// Initialize the Git config service with a real command runner
	runner := execrunner.RealCmdRunner{}
	gitConfigService := services.NewGitConfigService(logger, &runner)
	c.SetGitService(gitConfigService)

	// Initialize the account service
	accountService := services.NewRealAccountService(configService, logger)
	c.SetAccountService(accountService)

	// Initialize the SSH service
	sshService := services.NewRealSSHService(logger, &runner)
	c.SetSSHService(sshService)

	// Initialize the SSH agent service
	sshAgentService := services.NewRealSSHAgentService(logger, &runner)
	c.SetSSHAgentService(sshAgentService)

	// Initialize the zsh secrets service
	zshSecretsService := services.NewZshSecretsService(logger, &runner)
	c.SetZshSecretsService(zshSecretsService)

	// Initialize the zshrc service
	zshrcService := services.NewZshrcService(logger, &runner)
	c.SetZshrcService(zshrcService)

	// Initialize the GitHub token service
	githubTokenService := services.NewGitHubTokenService(logger, &runner)
	c.SetGitHubTokenService(githubTokenService)

	// TODO: Initialize GitHub, Health, and Validation services as they are implemented
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
