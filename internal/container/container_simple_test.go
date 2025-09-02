package container

import (
	"context"
	"testing"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

func TestNewSimpleContainer(t *testing.T) {
	container := NewSimpleContainer()

	if container == nil {
		t.Fatal("NewSimpleContainer returned nil")
	}

	// Test logger initialization
	logger := container.GetLogger()
	if logger == nil {
		t.Error("GetLogger returned nil")
	}
}

func TestSimpleContainerServiceManagement(t *testing.T) {
	container := NewSimpleContainer()

	// Test service setting and getting
	testService := "test-service"
	container.SetConfigService(testService)

	retrievedService := container.GetConfigService()
	if retrievedService != testService {
		t.Errorf("Expected %v, got %v", testService, retrievedService)
	}

	// Test account service
	container.SetAccountService(testService)
	accountService := container.GetAccountService()
	if accountService != testService {
		t.Errorf("Expected %v, got %v", testService, accountService)
	}

	// Test SSH service
	container.SetSSHService(testService)
	sshService := container.GetSSHService()
	if sshService != testService {
		t.Errorf("Expected %v, got %v", testService, sshService)
	}

	// Test Git service
	container.SetGitService(testService)
	gitService := container.GetGitService()
	if gitService != testService {
		t.Errorf("Expected %v, got %v", testService, gitService)
	}

	// Test GitHub service
	container.SetGitHubService(testService)
	githubService := container.GetGitHubService()
	if githubService != testService {
		t.Errorf("Expected %v, got %v", testService, githubService)
	}

	// Test health service
	container.SetHealthService(testService)
	healthService := container.GetHealthService()
	if healthService != testService {
		t.Errorf("Expected %v, got %v", testService, healthService)
	}

	// Test validation service
	container.SetValidationService(testService)
	validationService := container.GetValidationService()
	if validationService != testService {
		t.Errorf("Expected %v, got %v", testService, validationService)
	}
}

func TestSimpleContainerInitialization(t *testing.T) {
	container := NewSimpleContainer()
	ctx := context.Background()

	// Test initialization
	err := container.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	// Test shutdown
	err = container.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestGlobalSimpleContainer(t *testing.T) {
	ctx := context.Background()

	// Test global container singleton
	container1 := GetGlobalSimpleContainer()
	container2 := GetGlobalSimpleContainer()

	if container1 != container2 {
		t.Error("Global container is not a singleton")
	}

	// Test initialization
	err := InitializeGlobalSimpleContainer(ctx)
	if err != nil {
		t.Errorf("InitializeGlobalSimpleContainer failed: %v", err)
	}

	// Test shutdown
	err = ShutdownGlobalSimpleContainer(ctx)
	if err != nil {
		t.Errorf("ShutdownGlobalSimpleContainer failed: %v", err)
	}
}

func TestSimpleContainerLogger(t *testing.T) {
	container := NewSimpleContainer()

	// Test default logger
	logger1 := container.GetLogger()
	if logger1 == nil {
		t.Error("Default logger is nil")
	}

	// Test custom logger
	customLogger := observability.NewDefaultLogger()
	container.SetLogger(customLogger)

	logger2 := container.GetLogger()
	if logger2 != customLogger {
		t.Error("Custom logger not set correctly")
	}
}

func TestSimpleContainerConcurrency(t *testing.T) {
	container := NewSimpleContainer()
	done := make(chan bool)

	// Test concurrent access to services
	for i := 0; i < 10; i++ {
		go func(id int) {
			container.SetConfigService(id)
			container.GetConfigService()
			container.SetAccountService(id)
			container.GetAccountService()
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	configService := container.GetConfigService()
	if configService == nil {
		t.Error("Config service should not be nil after concurrent access")
	}

	accountService := container.GetAccountService()
	if accountService == nil {
		t.Error("Account service should not be nil after concurrent access")
	}
}
