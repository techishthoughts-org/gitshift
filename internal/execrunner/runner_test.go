package execrunner

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockCmdRunner is a mock implementation for testing
type MockCmdRunner struct {
	CombinedOutputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
	RunFunc            func(ctx context.Context, name string, args ...string) error
}

func (m *MockCmdRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.CombinedOutputFunc != nil {
		return m.CombinedOutputFunc(ctx, name, args...)
	}
	return []byte("mock output"), nil
}

func (m *MockCmdRunner) Run(ctx context.Context, name string, args ...string) error {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, name, args...)
	}
	return nil
}

func TestRealCmdRunner_CombinedOutput(t *testing.T) {
	runner := &RealCmdRunner{}
	ctx := context.Background()

	// Test successful command
	output, err := runner.CombinedOutput(ctx, "echo", "hello world")
	if err != nil {
		t.Errorf("CombinedOutput should not return error for echo command: %v", err)
	}
	if string(output) != "hello world\n" {
		t.Errorf("Expected 'hello world\\n', got %q", string(output))
	}

	// Test command with multiple args
	output, err = runner.CombinedOutput(ctx, "echo", "-n", "test")
	if err != nil {
		t.Errorf("CombinedOutput should not return error for echo -n: %v", err)
	}
	if string(output) != "test" {
		t.Errorf("Expected 'test', got %q", string(output))
	}

	// Test non-existent command
	_, err = runner.CombinedOutput(ctx, "nonexistentcommand12345")
	if err == nil {
		t.Error("CombinedOutput should return error for non-existent command")
	}
}

func TestRealCmdRunner_Run(t *testing.T) {
	runner := &RealCmdRunner{}
	ctx := context.Background()

	// Test successful command
	err := runner.Run(ctx, "echo", "hello")
	if err != nil {
		t.Errorf("Run should not return error for echo command: %v", err)
	}

	// Test command that returns non-zero exit code
	err = runner.Run(ctx, "false")
	if err == nil {
		t.Error("Run should return error for false command")
	}

	// Test non-existent command
	err = runner.Run(ctx, "nonexistentcommand12345")
	if err == nil {
		t.Error("Run should return error for non-existent command")
	}
}

func TestRealCmdRunner_ContextCancellation(t *testing.T) {
	runner := &RealCmdRunner{}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// This should fail due to context cancellation
	_, err := runner.CombinedOutput(ctx, "sleep", "10")
	if err == nil {
		t.Error("CombinedOutput should return error when context is cancelled")
	}

	err = runner.Run(ctx, "sleep", "10")
	if err == nil {
		t.Error("Run should return error when context is cancelled")
	}
}

func TestRealCmdRunner_ContextTimeout(t *testing.T) {
	runner := &RealCmdRunner{}

	// Test context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should timeout
	_, err := runner.CombinedOutput(ctx, "sleep", "1")
	if err == nil {
		t.Error("CombinedOutput should return error when context times out")
	}

	err = runner.Run(ctx, "sleep", "1")
	if err == nil {
		t.Error("Run should return error when context times out")
	}
}

func TestMockCmdRunner_CombinedOutput(t *testing.T) {
	mock := &MockCmdRunner{
		CombinedOutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("mocked output"), nil
		},
	}

	output, err := mock.CombinedOutput(context.Background(), "test", "arg1", "arg2")
	if err != nil {
		t.Errorf("Mock CombinedOutput should not return error: %v", err)
	}
	if string(output) != "mocked output" {
		t.Errorf("Expected 'mocked output', got %q", string(output))
	}
}

func TestMockCmdRunner_Run(t *testing.T) {
	mock := &MockCmdRunner{
		RunFunc: func(ctx context.Context, name string, args ...string) error {
			return errors.New("mocked error")
		},
	}

	err := mock.Run(context.Background(), "test", "arg1", "arg2")
	if err == nil {
		t.Error("Mock Run should return error")
	}
	if err.Error() != "mocked error" {
		t.Errorf("Expected 'mocked error', got %q", err.Error())
	}
}

func TestMockCmdRunner_DefaultBehavior(t *testing.T) {
	mock := &MockCmdRunner{}

	// Test default CombinedOutput behavior
	output, err := mock.CombinedOutput(context.Background(), "test")
	if err != nil {
		t.Errorf("Default CombinedOutput should not return error: %v", err)
	}
	if string(output) != "mock output" {
		t.Errorf("Expected 'mock output', got %q", string(output))
	}

	// Test default Run behavior
	err = mock.Run(context.Background(), "test")
	if err != nil {
		t.Errorf("Default Run should not return error: %v", err)
	}
}

func TestCmdRunner_Interface(t *testing.T) {
	// Test that RealCmdRunner implements CmdRunner interface
	var runner CmdRunner = &RealCmdRunner{}
	_ = runner // Just check interface compliance

	// Test that MockCmdRunner implements CmdRunner interface
	var mockRunner CmdRunner = &MockCmdRunner{}
	_ = mockRunner // Just check interface compliance
}

func TestRealCmdRunner_EdgeCases(t *testing.T) {
	runner := &RealCmdRunner{}
	ctx := context.Background()

	// Test with empty command name
	_, err := runner.CombinedOutput(ctx, "")
	if err == nil {
		t.Error("CombinedOutput should return error for empty command name")
	}

	err = runner.Run(ctx, "")
	if err == nil {
		t.Error("Run should return error for empty command name")
	}

	// Test with nil context - this will panic, so we need to recover
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("CombinedOutput should panic for nil context")
			}
		}()
		_, _ = runner.CombinedOutput(context.TODO(), "echo", "test")
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Run should panic for nil context")
			}
		}()
		_ = runner.Run(context.TODO(), "echo", "test")
	}()
}

func TestRealCmdRunner_Concurrency(t *testing.T) {
	runner := &RealCmdRunner{}
	ctx := context.Background()

	// Test concurrent execution
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			output, err := runner.CombinedOutput(ctx, "echo", "concurrent test")
			if err != nil {
				t.Errorf("Concurrent CombinedOutput failed: %v", err)
			}
			if string(output) != "concurrent test\n" {
				t.Errorf("Concurrent CombinedOutput unexpected output: %q", string(output))
			}

			err = runner.Run(ctx, "echo", "concurrent run")
			if err != nil {
				t.Errorf("Concurrent Run failed: %v", err)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRealCmdRunner_Performance(t *testing.T) {
	runner := &RealCmdRunner{}
	ctx := context.Background()

	// Test that commands complete in reasonable time
	start := time.Now()
	_, err := runner.CombinedOutput(ctx, "echo", "performance test")
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Performance test CombinedOutput failed: %v", err)
	}
	if duration > 1*time.Second {
		t.Errorf("CombinedOutput took too long: %v", duration)
	}

	start = time.Now()
	err = runner.Run(ctx, "echo", "performance test")
	duration = time.Since(start)

	if err != nil {
		t.Errorf("Performance test Run failed: %v", err)
	}
	if duration > 1*time.Second {
		t.Errorf("Run took too long: %v", duration)
	}
}

func TestMockCmdRunner_ContextHandling(t *testing.T) {
	mock := &MockCmdRunner{
		CombinedOutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return []byte("context ok"), nil
			}
		},
		RunFunc: func(ctx context.Context, name string, args ...string) error {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}

	// Test with valid context
	ctx := context.Background()
	output, err := mock.CombinedOutput(ctx, "test")
	if err != nil {
		t.Errorf("Mock CombinedOutput with valid context should not error: %v", err)
	}
	if string(output) != "context ok" {
		t.Errorf("Expected 'context ok', got %q", string(output))
	}

	err = mock.Run(ctx, "test")
	if err != nil {
		t.Errorf("Mock Run with valid context should not error: %v", err)
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = mock.CombinedOutput(cancelledCtx, "test")
	if err == nil {
		t.Error("Mock CombinedOutput with cancelled context should error")
	}

	err = mock.Run(cancelledCtx, "test")
	if err == nil {
		t.Error("Mock Run with cancelled context should error")
	}
}
