package account

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
)

func TestCreateAccountCommands(t *testing.T) {
	cmd := CreateAccountCommands()

	if cmd == nil {
		t.Fatal("CreateAccountCommands should return non-nil command")
	}

	if cmd.Use != "account" {
		t.Errorf("Expected Use 'account', got '%s'", cmd.Use)
	}

	if cmd.Short != "Account management commands" {
		t.Errorf("Expected Short 'Account management commands', got '%s'", cmd.Short)
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}

	// Check that the command has the expected structure
	if cmd.Use != "account" {
		t.Errorf("Expected command use 'account', got '%s'", cmd.Use)
	}
}

func TestNewAccountCommand(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	if cmd == nil {
		t.Fatal("NewAccountCommand should return non-nil command")
	}

	if cmd.BaseCommand == nil {
		t.Error("BaseCommand should be initialized")
	}

	if cmd.category != "account" {
		t.Errorf("Expected category 'account', got '%s'", cmd.category)
	}

	if cmd.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", cmd.Name())
	}

	if cmd.Description() != "Test command" {
		t.Errorf("Expected description 'Test command', got '%s'", cmd.Description())
	}

	if cmd.Usage() != "test [args]" {
		t.Errorf("Expected usage 'test [args]', got '%s'", cmd.Usage())
	}
}

func TestGetCategory(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	category := cmd.GetCategory()
	if category != "account" {
		t.Errorf("Expected category 'account', got '%s'", category)
	}
}

func TestAccountCommandInheritance(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	// Test that AccountCommand inherits from BaseCommand
	if cmd.BaseCommand == nil {
		t.Error("AccountCommand should have BaseCommand")
	}

	// Test that methods from BaseCommand work
	if cmd.GetContainer() == nil {
		t.Error("GetContainer should work on AccountCommand")
	}

	if cmd.GetLogger() == nil {
		t.Error("GetLogger should work on AccountCommand")
	}

	// Test that the command can create a Cobra command
	cobraCmd := cmd.CreateCobraCommand()
	if cobraCmd == nil {
		t.Error("CreateCobraCommand should work on AccountCommand")
		return
	}

	if cobraCmd.Use != "test [args]" {
		t.Errorf("Expected Cobra command use 'test [args]', got '%s'", cobraCmd.Use)
	}
}

func TestAccountCommandWithExamples(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cmd.BaseCommand = cmd.WithExamples("test arg1", "test arg2 --flag")

	examples := cmd.Examples()
	if len(examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(examples))
	}

	if examples[0] != "test arg1" {
		t.Errorf("Expected first example 'test arg1', got '%s'", examples[0])
	}
}

func TestAccountCommandWithFlags(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cmd.BaseCommand = cmd.BaseCommand.WithFlags(
		commands.Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		commands.Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	)

	cobraCmd := cmd.CreateCobraCommand()

	// Check that flags were added
	debugFlag := cobraCmd.Flags().Lookup("debug")
	if debugFlag == nil {
		t.Error("debug flag should be added")
	}

	outputFlag := cobraCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Error("output flag should be added")
	}
}

func TestAccountCommandExecution(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	// Test that the command can be executed (though it will fail with default implementation)
	ctx := context.Background()
	err := cmd.Execute(ctx, []string{"arg1"})

	// The base implementation should return an error since Run is not implemented
	if err == nil {
		t.Error("Execute should return error for unimplemented Run method")
	}
}

func TestAccountCommandValidation(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	// Test default validation (should pass)
	err := cmd.Validate([]string{"arg1", "arg2"})
	if err != nil {
		t.Errorf("Default validation should pass: %v", err)
	}
}

func TestAccountCommandContext(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	// Test context handling
	ctx := context.Background()
	cmd.SetContext(ctx)

	retrievedCtx := cmd.GetContext()
	if retrievedCtx != ctx {
		t.Error("GetContext should return the set context")
	}
}

func TestAccountCommandCobraIntegration(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cobraCmd := cmd.CreateCobraCommand()

	// Test that the Cobra command is properly configured
	if cobraCmd.Use != "test [args]" {
		t.Errorf("Expected Use 'test [args]', got '%s'", cobraCmd.Use)
	}

	if cobraCmd.Short != "Test command" {
		t.Errorf("Expected Short 'Test command', got '%s'", cobraCmd.Short)
	}

	// Test that the command can be added to a parent command
	parentCmd := &cobra.Command{Use: "parent"}
	parentCmd.AddCommand(cobraCmd)

	if len(parentCmd.Commands()) != 1 {
		t.Errorf("Expected 1 subcommand, got %d", len(parentCmd.Commands()))
	}
}

func TestAccountCommandErrorHandling(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")

	// Test error handling in Execute
	ctx := context.Background()
	err := cmd.Execute(ctx, []string{"arg1"})

	if err == nil {
		t.Error("Execute should return error for unimplemented Run method")
	}

	// Check that error is properly wrapped
	if err.Error() == "" {
		t.Error("Error should have a message")
	}
}

func TestAccountCommandPerformance(t *testing.T) {
	// Test performance of creating many account commands
	numCommands := 100

	start := time.Now()

	for i := 0; i < numCommands; i++ {
		cmd := NewAccountCommand("test"+string(rune(i)), "Test command", "test [args]")
		if cmd == nil {
			t.Fatalf("Failed to create command %d", i)
		}
	}

	duration := time.Since(start)

	// Performance requirement: should create 100 commands in < 100ms
	if duration > 100*time.Millisecond {
		t.Errorf("Creating %d commands took too long: %v", numCommands, duration)
	}
}

func TestAccountCommandConcurrency(t *testing.T) {
	// Test concurrent creation of account commands
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer func() { done <- true }()

			cmd := NewAccountCommand("test"+string(rune(i)), "Test command", "test [args]")
			if cmd == nil {
				t.Errorf("Failed to create command in goroutine %d", i)
			}

			// Test that the command works
			if cmd.GetCategory() != "account" {
				t.Errorf("Command in goroutine %d has wrong category", i)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestAccountCommandIntegration(t *testing.T) {
	// Test integration with the command registry
	registry := commands.NewCommandRegistry()
	registry.RegisterCategory("account", "Account management")

	cmd := NewAccountCommand("test", "Test command", "test [args]")
	registry.AddCommand("account", cmd)

	// Verify command was added
	retrievedCategory := registry.GetCategory("account")
	if retrievedCategory == nil {
		t.Fatal("Category should exist")
	}

	if len(retrievedCategory.Commands) != 1 {
		t.Errorf("Expected 1 command in category, got %d", len(retrievedCategory.Commands))
	}

	if retrievedCategory.Commands[0] != cmd {
		t.Error("Command should be the same instance")
	}
}

func TestAccountCommandExamples(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cmd.BaseCommand = cmd.BaseCommand.WithExamples(
		"test arg1",
		"test arg2 --flag",
		"test arg3 --verbose",
	)

	examples := cmd.Examples()
	if len(examples) != 3 {
		t.Errorf("Expected 3 examples, got %d", len(examples))
	}

	// Test that examples are properly formatted in Cobra command
	cobraCmd := cmd.CreateCobraCommand()
	if cobraCmd.Example == "" {
		t.Error("Cobra command should have examples")
	}

	// Check that examples contain the expected content
	expectedContent := "test arg1"
	if !strings.Contains(cobraCmd.Example, expectedContent) {
		t.Errorf("Cobra command example should contain '%s'", expectedContent)
	}
}

func TestAccountCommandFlags(t *testing.T) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cmd.BaseCommand = cmd.BaseCommand.WithFlags(
		commands.Flag{Name: "debug", Short: "d", Type: "bool", Default: false, Description: "Debug output"},
		commands.Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
		commands.Flag{Name: "count", Short: "n", Type: "int", Default: 0, Description: "Count"},
	)

	cobraCmd := cmd.CreateCobraCommand()

	// Test flag parsing
	err := cobraCmd.Flags().Parse([]string{"--debug", "--output", "test.txt", "--count", "5"})
	if err != nil {
		t.Errorf("Flag parsing should not return error: %v", err)
	}

	// Test flag retrieval
	debug := cmd.GetFlagBool(cobraCmd, "debug")
	if !debug {
		t.Error("debug flag should be true")
	}

	output := cmd.GetFlagString(cobraCmd, "output")
	if output != "test.txt" {
		t.Errorf("Expected output 'test.txt', got '%s'", output)
	}

	count := cmd.GetFlagInt(cobraCmd, "count")
	if count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

// Benchmark tests
func BenchmarkNewAccountCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewAccountCommand("test", "Test command", "test [args]")
	}
}

func BenchmarkAccountCommandCreateCobraCommand(b *testing.B) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	cmd.BaseCommand = cmd.BaseCommand.WithExamples("test arg1", "test arg2")
	cmd.BaseCommand = cmd.BaseCommand.WithFlags(
		commands.Flag{Name: "verbose", Short: "v", Type: "bool", Default: false, Description: "Verbose output"},
		commands.Flag{Name: "output", Short: "o", Type: "string", Default: "", Description: "Output file"},
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.CreateCobraCommand()
	}
}

func BenchmarkAccountCommandExecute(b *testing.B) {
	cmd := NewAccountCommand("test", "Test command", "test [args]")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Execute(ctx, []string{"arg1"})
	}
}
