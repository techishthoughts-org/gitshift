package container

import (
	"context"
	"testing"

	"github.com/techishthoughts/GitPersona/internal/observability"
	"github.com/techishthoughts/GitPersona/internal/validation"
)

// testFakeGitMgr implements the minimal GitConfigManager interface for tests
type testFakeGitMgr struct{}

func (f *testFakeGitMgr) SetUserConfiguration(ctx context.Context, name, email string) error {
	return nil
}
func (f *testFakeGitMgr) SetSSHCommand(ctx context.Context, sshCommand string) error { return nil }

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

// Package-level fakes used across tests
type fakeConfigSvc struct{}

func (f *fakeConfigSvc) Load(ctx context.Context) error { return nil }
func (f *fakeConfigSvc) Save(ctx context.Context) error { return nil }
func (f *fakeConfigSvc) GetAccounts(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{}
}
func (f *fakeConfigSvc) GetCurrentAccount(ctx context.Context) string              { return "" }
func (f *fakeConfigSvc) SetCurrentAccount(ctx context.Context, alias string) error { return nil }

type fakeSSH struct{}

func (f *fakeSSH) ValidateSSHConfiguration() (*validation.ValidationResult, error) {
	return &validation.ValidationResult{}, nil
}

func TestSimpleContainerServiceManagement(t *testing.T) {
	container := NewSimpleContainer()

	// Test service setting and getting using minimal fakes
	cfg := &fakeConfigSvc{}
	container.SetConfigService(cfg)

	retrievedService := container.GetConfigService()
	if retrievedService == nil {
		t.Errorf("Expected non-nil config service, got nil")
	}

	// Test account service (nil is acceptable for now)
	container.SetAccountService(nil)
	accountService := container.GetAccountService()
	if accountService != nil {
		t.Errorf("Expected nil account service, got %v", accountService)
	}

	// Test SSH service (use a fake matching the SSHValidator interface)
	ssh := &fakeSSH{}
	container.SetSSHService(ssh)
	sshService := container.GetSSHService()
	if sshService == nil {
		t.Errorf("Expected non-nil ssh service, got nil")
	}

	// Test Git service (use a fake implementation matching the interface)
	var testGitService testFakeGitMgr
	container.SetGitService(&testGitService)
	gitService := container.GetGitService()
	if gitService == nil {
		t.Errorf("Expected non-nil git service, got nil")
	}

	// Test GitHub, health and validation services (nil is acceptable placeholders)
	container.SetGitHubService(nil)
	githubService := container.GetGitHubService()
	if githubService != nil {
		t.Errorf("Expected nil github service, got %v", githubService)
	}

	container.SetHealthService(nil)
	healthService := container.GetHealthService()
	if healthService != nil {
		t.Errorf("Expected nil health service, got %v", healthService)
	}

	container.SetValidationService(nil)
	validationService := container.GetValidationService()
	if validationService != nil {
		t.Errorf("Expected nil validation service, got %v", validationService)
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

	// Test concurrent access to services using a concrete fake
	cfg := &fakeConfigSvc{}
	for i := 0; i < 10; i++ {
		go func() {
			container.SetConfigService(cfg)
			container.GetConfigService()
			container.SetAccountService(nil)
			container.GetAccountService()
			done <- true
		}()
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

	// account service may be nil in this test scenario
	_ = container.GetAccountService()
}
