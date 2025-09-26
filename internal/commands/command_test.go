package commands

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// MockCommand implements the Command interface for testing
type MockCommand struct {
	*BaseCommand
	runFunc      func(ctx context.Context, args []string) error
	validateFunc func(args []string) error
}

func NewMockCommand(name, description, usage string) *MockCommand {
	return &MockCommand{
		BaseCommand: NewBaseCommand(name, description, usage),
	}
}

func (m *MockCommand) Run(ctx context.Context, args []string) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, args)
	}
	// Default implementation that doesn't fail
	return nil
}

func (m *MockCommand) Execute(ctx context.Context, args []string) error {
	m.startTime = time.Now()
	m.ctx = ctx

	// Log command execution
	m.logger.Info(ctx, "executing_command",
		observability.F("command", m.name),
		observability.F("args", args),
	)

	// Validate arguments
	if err := m.Validate(args); err != nil {
		return m.wrapError(err, "validation_failed")
	}

	// Execute the command
	err := m.Run(ctx, args)

	// Log completion
	duration := time.Since(m.startTime)
	if err != nil {
		observability.LogCommandError(ctx, m.logger, m.name, err, duration)
		return m.wrapError(err, "execution_failed")
	}

	observability.LogCommandSuccess(ctx, m.logger, m.name, duration)
	return nil
}

func (m *MockCommand) Validate(args []string) error {
	if m.validateFunc != nil {
		return m.validateFunc(args)
	}
	return m.BaseCommand.Validate(args)
}

func TestNewBaseCommand(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	if cmd == nil {
		t.Fatal("NewBaseCommand should return non-nil command")
	}

	if cmd.name != "test" {
		t.Errorf("Expected name 'test', got '%s'", cmd.name)
	}

	if cmd.description != "Test command" {
		t.Errorf("Expected description 'Test command', got '%s'", cmd.description)
	}

	if cmd.usage != "test [args]" {
		t.Errorf("Expected usage 'test [args]', got '%s'", cmd.usage)
	}

	if cmd.container == nil {
		t.Error("Container should be initialized")
	}

	if cmd.logger == nil {
		t.Error("Logger should be initialized")
	}
}

func TestWithExamples(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	examples := []string{"test arg1", "test arg2 --flag"}

	cmd = cmd.WithExamples(examples...)

	if len(cmd.examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(cmd.examples))
	}

	if cmd.examples[0] != "test arg1" {
		t.Errorf("Expected first example 'test arg1', got '%s'", cmd.examples[0])
	}
}

func TestWithFlags(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	flags := []Flag{
		{Name: "verbose", Short: "v", Type: "bool", Default: false, Description: "Verbose output"},
		{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	}

	cmd = cmd.WithFlags(flags...)

	if len(cmd.flags) != 2 {
		t.Errorf("Expected 2 flags, got %d", len(cmd.flags))
	}

	if cmd.flags[0].Name != "verbose" {
		t.Errorf("Expected first flag name 'verbose', got '%s'", cmd.flags[0].Name)
	}
}

func TestGetContainer(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	container := cmd.GetContainer()

	if container == nil {
		t.Error("GetContainer should return non-nil container")
	}
}

func TestGetLogger(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	logger := cmd.GetLogger()

	if logger == nil {
		t.Error("GetLogger should return non-nil logger")
	}
}

func TestGetContext(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	// Test default context
	ctx := cmd.GetContext()
	if ctx == nil {
		t.Error("GetContext should return non-nil context")
	}

	// Test custom context
	type testContextKey string
	customCtx := context.WithValue(context.Background(), testContextKey("test"), "value")
	cmd.SetContext(customCtx)

	retrievedCtx := cmd.GetContext()
	if retrievedCtx.Value(testContextKey("test")) != "value" {
		t.Error("GetContext should return the set context")
	}
}

func TestSetContext(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	type testContextKey string
	customCtx := context.WithValue(context.Background(), testContextKey("test"), "value")

	cmd.SetContext(customCtx)

	if cmd.ctx != customCtx {
		t.Error("SetContext should set the context")
	}
}

func TestName(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	if cmd.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", cmd.Name())
	}
}

func TestDescription(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	if cmd.Description() != "Test command" {
		t.Errorf("Expected description 'Test command', got '%s'", cmd.Description())
	}
}

func TestUsage(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	if cmd.Usage() != "test [args]" {
		t.Errorf("Expected usage 'test [args]', got '%s'", cmd.Usage())
	}
}

func TestExamples(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	examples := []string{"test arg1", "test arg2"}
	cmd = cmd.WithExamples(examples...)

	retrievedExamples := cmd.Examples()
	if len(retrievedExamples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(retrievedExamples))
	}
}

func TestExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.runFunc = func(ctx context.Context, args []string) error {
			return nil
		}

		err := cmd.Execute(ctx, []string{"arg1", "arg2"})
		if err != nil {
			t.Errorf("Execute should not return error: %v", err)
		}

		if cmd.startTime.IsZero() {
			t.Error("Start time should be set")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.validateFunc = func(args []string) error {
			return errors.New("validation failed")
		}

		err := cmd.Execute(ctx, []string{"arg1"})
		if err == nil {
			t.Error("Execute should return error on validation failure")
		}
	})

	t.Run("execution failure", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.runFunc = func(ctx context.Context, args []string) error {
			return errors.New("execution failed")
		}

		err := cmd.Execute(ctx, []string{"arg1"})
		if err == nil {
			t.Error("Execute should return error on execution failure")
		}
	})
}

func TestValidate(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	// Default validation should pass
	err := cmd.Validate([]string{"arg1", "arg2"})
	if err != nil {
		t.Errorf("Default validation should pass: %v", err)
	}
}

func TestRun(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	err := cmd.Run(context.Background(), []string{"arg1"})
	if err == nil {
		t.Error("Base Run should return error")
	}

	expectedErr := "run method not implemented"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestWrapError(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	type argsContextKey string
	cmd.SetContext(context.WithValue(context.Background(), argsContextKey("args"), []string{"arg1"}))

	t.Run("nil error", func(t *testing.T) {
		err := cmd.wrapError(nil, "test_code")
		if err != nil {
			t.Error("wrapError should return nil for nil input")
		}
	})

	t.Run("regular error", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrappedErr := cmd.wrapError(originalErr, "test_code")

		if wrappedErr == nil {
			t.Fatal("wrapError should return non-nil error")
		}

		// Check that error message contains expected content
		errorMsg := wrappedErr.Error()
		if !strings.Contains(errorMsg, "test_code") {
			t.Errorf("Expected error message to contain 'test_code', got '%s'", errorMsg)
		}
	})

	t.Run("GitPersonaError", func(t *testing.T) {
		originalErr := errors.New("config error")
		wrappedErr := cmd.wrapError(originalErr, "test_code")

		if wrappedErr == nil {
			t.Fatal("wrapError should return non-nil error")
		}

		// Check that error message contains expected content
		errorMsg := wrappedErr.Error()
		if !strings.Contains(errorMsg, "test_code") {
			t.Errorf("Expected error message to contain 'test_code', got '%s'", errorMsg)
		}
	})
}

func TestCreateCobraCommand(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cmd = cmd.WithExamples("test arg1", "test arg2 --flag")
	cmd = cmd.WithFlags(
		Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	)

	cobraCmd := cmd.CreateCobraCommand()

	if cobraCmd == nil {
		t.Fatal("CreateCobraCommand should return non-nil command")
	}

	if cobraCmd.Use != "test [args]" {
		t.Errorf("Expected Use 'test [args]', got '%s'", cobraCmd.Use)
	}

	if cobraCmd.Short != "Test command" {
		t.Errorf("Expected Short 'Test command', got '%s'", cobraCmd.Short)
	}

	// Check that flags were added
	verboseFlag := cobraCmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag should be added")
	}

	outputFlag := cobraCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("output flag should be added")
	}

	// Check common flags
	jsonFlag := cobraCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("json flag should be added")
	}
}

func TestAddFlags(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cmd = cmd.WithFlags(
		Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
		Flag{Name: "count", Short: "n", Type: "int", Default: 0, Description: "Count"},
	)

	cobraCmd := &cobra.Command{}
	cmd.addFlags(cobraCmd)

	// Check bool flag
	debugFlag := cobraCmd.Flags().Lookup("debug")
	if debugFlag == nil {
		t.Error("debug flag should be added")
	}

	// Check string flag
	outputFlag := cobraCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("output flag should be added")
	}

	// Check int flag
	countFlag := cobraCmd.Flags().Lookup("count")
	if countFlag == nil {
		t.Error("count flag should be added")
	}

	// Check that common flags were added
	verboseFlag := cobraCmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag should be added as common flag")
	}

	jsonFlag := cobraCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("json flag should be added as common flag")
	}

	configFlag := cobraCmd.Flags().Lookup("config")
	if configFlag == nil {
		t.Error("config flag should be added as common flag")
	}
}

func TestFormatExamples(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	t.Run("no examples", func(t *testing.T) {
		formatted := cmd.formatExamples()
		if formatted != "" {
			t.Errorf("Expected empty string for no examples, got '%s'", formatted)
		}
	})

	t.Run("with examples", func(t *testing.T) {
		cmd = cmd.WithExamples("test arg1", "test arg2 --flag")
		formatted := cmd.formatExamples()

		expected := "  test arg1\n  test arg2 --flag"
		if formatted != expected {
			t.Errorf("Expected '%s', got '%s'", expected, formatted)
		}
	})
}

func TestGetFlagBool(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().BoolP("debug", "d", false, "Debug output")

	// Test default value
	value := cmd.GetFlagBool(cobraCmd, "debug")
	if value != false {
		t.Errorf("Expected false, got %v", value)
	}

	// Test set value
	if err := cobraCmd.Flags().Set("debug", "true"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}
	value = cmd.GetFlagBool(cobraCmd, "debug")
	if value != true {
		t.Errorf("Expected true, got %v", value)
	}
}

func TestGetFlagString(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().StringP("output", "o", "", "Output file")

	// Test default value
	value := cmd.GetFlagString(cobraCmd, "output")
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}

	// Test set value
	if err := cobraCmd.Flags().Set("output", "test.txt"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}
	value = cmd.GetFlagString(cobraCmd, "output")
	if value != "test.txt" {
		t.Errorf("Expected 'test.txt', got '%s'", value)
	}
}

func TestGetFlagInt(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cobraCmd := &cobra.Command{}
	cobraCmd.Flags().IntP("count", "c", 0, "Count")

	// Test default value
	value := cmd.GetFlagInt(cobraCmd, "count")
	if value != 0 {
		t.Errorf("Expected 0, got %d", value)
	}

	// Test set value
	if err := cobraCmd.Flags().Set("count", "5"); err != nil {
		t.Fatalf("Failed to set flag: %v", err)
	}
	value = cmd.GetFlagInt(cobraCmd, "count")
	if value != 5 {
		t.Errorf("Expected 5, got %d", value)
	}
}

func TestRequireArgs(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")

	t.Run("sufficient arguments", func(t *testing.T) {
		cobraCmd := &cobra.Command{}
		if err := cobraCmd.Flags().Parse([]string{"arg1", "arg2"}); err != nil {
			t.Fatalf("Failed to parse flags: %v", err)
		}

		err := cmd.RequireArgs(cobraCmd, 1, 2)
		if err != nil {
			t.Errorf("RequireArgs should not return error: %v", err)
		}
	})

	t.Run("insufficient arguments", func(t *testing.T) {
		cobraCmd := &cobra.Command{}
		if err := cobraCmd.Flags().Parse([]string{"arg1"}); err != nil {
			t.Fatalf("Failed to parse flags: %v", err)
		}

		err := cmd.RequireArgs(cobraCmd, 2, 0)
		if err == nil {
			t.Error("RequireArgs should return error for insufficient arguments")
		}
	})

	t.Run("too many arguments", func(t *testing.T) {
		cobraCmd := &cobra.Command{}
		if err := cobraCmd.Flags().Parse([]string{"arg1", "arg2", "arg3"}); err != nil {
			t.Fatalf("Failed to parse flags: %v", err)
		}

		err := cmd.RequireArgs(cobraCmd, 1, 2)
		if err == nil {
			t.Error("RequireArgs should return error for too many arguments")
		}
	})
}

func TestPrintMethods(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	ctx := context.Background()

	// These methods should not panic
	cmd.PrintInfo(ctx, "test info", observability.F("key", "value"))
	cmd.PrintSuccess(ctx, "test success", observability.F("key", "value"))
	cmd.PrintWarning(ctx, "test warning", observability.F("key", "value"))
	cmd.PrintError(ctx, "test error", observability.F("key", "value"))
}

func TestCommandExecutionFlow(t *testing.T) {
	ctx := context.Background()

	cmd := NewMockCommand("test", "Test command", "test [args]")
	cmd.runFunc = func(ctx context.Context, args []string) error {
		if len(args) != 1 {
			return errors.New("expected 1 argument")
		}
		return nil
	}

	// Test successful execution
	err := cmd.Execute(ctx, []string{"arg1"})
	if err != nil {
		t.Errorf("Execute should not return error: %v", err)
	}

	// Test execution with wrong number of arguments
	err = cmd.Execute(ctx, []string{"arg1", "arg2"})
	if err == nil {
		t.Error("Execute should return error for wrong number of arguments")
	}
}

func TestCommandWithFlags(t *testing.T) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cmd = cmd.WithFlags(
		Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	)

	cobraCmd := cmd.CreateCobraCommand()

	// Test flag parsing
	err := cobraCmd.Flags().Parse([]string{"--verbose", "--output", "test.txt"})
	if err != nil {
		t.Errorf("Flag parsing should not return error: %v", err)
	}

	verbose := cmd.GetFlagBool(cobraCmd, "verbose")
	if !verbose {
		t.Error("verbose flag should be true")
	}

	output := cmd.GetFlagString(cobraCmd, "output")
	if output != "test.txt" {
		t.Errorf("Expected output 'test.txt', got '%s'", output)
	}
}

func TestCommandErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("validation error", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.validateFunc = func(args []string) error {
			return errors.New("validation failed")
		}

		err := cmd.Execute(ctx, []string{"arg1"})
		if err == nil {
			t.Error("Execute should return error on validation failure")
		}

		// Check that error is wrapped
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "validation_failed") {
			t.Errorf("Expected error message to contain 'validation_failed', got '%s'", errorMsg)
		}
	})

	t.Run("execution error", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.runFunc = func(ctx context.Context, args []string) error {
			return errors.New("execution failed")
		}

		err := cmd.Execute(ctx, []string{"arg1"})
		if err == nil {
			t.Error("Execute should return error on execution failure")
		}

		// Check that error is wrapped
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "execution_failed") {
			t.Errorf("Expected error message to contain 'execution_failed', got '%s'", errorMsg)
		}
	})
}

func TestCommandTiming(t *testing.T) {
	ctx := context.Background()

	cmd := NewMockCommand("test", "Test command", "test [args]")
	cmd.runFunc = func(ctx context.Context, args []string) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	start := time.Now()
	err := cmd.Execute(ctx, []string{"arg1"})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Execute should not return error: %v", err)
	}

	if duration < 10*time.Millisecond {
		t.Error("Execution should take at least 10ms")
	}

	if cmd.startTime.IsZero() {
		t.Error("Start time should be set")
	}
}

// Benchmark tests
func BenchmarkNewBaseCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewBaseCommand("test", "Test command", "test [args]")
	}
}

func BenchmarkCommandExecute(b *testing.B) {
	ctx := context.Background()
	cmd := NewMockCommand("test", "Test command", "test [args]")
	cmd.runFunc = func(ctx context.Context, args []string) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Execute(ctx, []string{"arg1"})
	}
}

func BenchmarkCreateCobraCommand(b *testing.B) {
	cmd := NewBaseCommand("test", "Test command", "test [args]")
	cmd = cmd.WithExamples("test arg1", "test arg2")
	cmd = cmd.WithFlags(
		Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.CreateCobraCommand()
	}
}
