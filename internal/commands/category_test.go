package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestNewCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	if registry == nil {
		t.Fatal("NewCommandRegistry should return non-nil registry")
	}

	if registry.categories == nil {
		t.Error("categories map should be initialized")
	}

	if len(registry.categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(registry.categories))
	}
}

func TestRegisterCategory(t *testing.T) {
	registry := NewCommandRegistry()

	category := registry.RegisterCategory("test", "Test category")

	if category == nil {
		t.Fatal("RegisterCategory should return non-nil category")
	}

	if category.Name != "test" {
		t.Errorf("Expected category name 'test', got '%s'", category.Name)
	}

	if category.Description != "Test category" {
		t.Errorf("Expected category description 'Test category', got '%s'", category.Description)
	}

	if category.Commands == nil {
		t.Error("Commands slice should be initialized")
	}

	if len(category.Commands) != 0 {
		t.Errorf("Expected 0 commands, got %d", len(category.Commands))
	}

	// Check that category was added to registry
	if len(registry.categories) != 1 {
		t.Errorf("Expected 1 category in registry, got %d", len(registry.categories))
	}

	if registry.categories["test"] != category {
		t.Error("Category should be stored in registry")
	}
}

func TestAddCommand(t *testing.T) {
	registry := NewCommandRegistry()
	category := registry.RegisterCategory("test", "Test category")

	cmd := NewMockCommand("testcmd", "Test command", "testcmd [args]")

	registry.AddCommand("test", cmd)

	if len(category.Commands) != 1 {
		t.Errorf("Expected 1 command in category, got %d", len(category.Commands))
	}

	if category.Commands[0] != cmd {
		t.Error("Command should be added to category")
	}
}

func TestAddCommandToNonExistentCategory(t *testing.T) {
	registry := NewCommandRegistry()

	cmd := NewMockCommand("testcmd", "Test command", "testcmd [args]")

	// This should not panic or error
	registry.AddCommand("nonexistent", cmd)

	if len(registry.categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(registry.categories))
	}
}

func TestGetCategory(t *testing.T) {
	registry := NewCommandRegistry()
	category := registry.RegisterCategory("test", "Test category")

	retrieved := registry.GetCategory("test")
	if retrieved != category {
		t.Error("GetCategory should return the registered category")
	}

	// Test non-existent category
	retrieved = registry.GetCategory("nonexistent")
	if retrieved != nil {
		t.Error("GetCategory should return nil for non-existent category")
	}
}

func TestCreateCobraCommands(t *testing.T) {
	registry := NewCommandRegistry()
	rootCmd := &cobra.Command{Use: "root"}

	// Test with single command category
	registry.RegisterCategory("single", "Single command category")
	cmd1 := NewMockCommand("singlecmd", "Single command", "singlecmd [args]")
	registry.AddCommand("single", cmd1)

	// Test with multiple commands category
	registry.RegisterCategory("multiple", "Multiple commands category")
	cmd2 := NewMockCommand("cmd1", "Command 1", "cmd1 [args]")
	cmd3 := NewMockCommand("cmd2", "Command 2", "cmd2 [args]")
	registry.AddCommand("multiple", cmd2)
	registry.AddCommand("multiple", cmd3)

	registry.CreateCobraCommands(rootCmd)

	// Check that commands were added to root
	// Single command category adds 1 command directly to root
	// Multiple command category adds 1 parent command to root
	// So we expect 2 commands total (1 single + 1 parent)
	if len(rootCmd.Commands()) != 2 {
		t.Errorf("Expected 2 commands in root, got %d", len(rootCmd.Commands()))
	}
}

func TestCreateCobraCommandsWithEmptyCategory(t *testing.T) {
	registry := NewCommandRegistry()
	rootCmd := &cobra.Command{Use: "root"}

	// Register empty category
	registry.RegisterCategory("empty", "Empty category")

	registry.CreateCobraCommands(rootCmd)

	// Should not add any commands
	if len(rootCmd.Commands()) != 0 {
		t.Errorf("Expected 0 commands in root, got %d", len(rootCmd.Commands()))
	}
}

func TestStandardCommandCategories(t *testing.T) {
	registry := StandardCommandCategories()

	if registry == nil {
		t.Fatal("StandardCommandCategories should return non-nil registry")
	}

	expectedCategories := []string{"account", "ssh", "git", "system", "project"}

	for _, expected := range expectedCategories {
		category := registry.GetCategory(expected)
		if category == nil {
			t.Errorf("Expected category '%s' to be registered", expected)
		}
	}

	if len(registry.categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(registry.categories))
	}
}

func TestGroupedCommand(t *testing.T) {
	cmd := NewGroupedCommand("test", "group1", "testcmd", "Test command", "testcmd [args]")

	if cmd == nil {
		t.Fatal("NewGroupedCommand should return non-nil command")
	}

	if cmd.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", cmd.Category)
	}

	if cmd.Group != "group1" {
		t.Errorf("Expected group 'group1', got '%s'", cmd.Group)
	}

	if cmd.Name() != "testcmd" {
		t.Errorf("Expected name 'testcmd', got '%s'", cmd.Name())
	}

	if cmd.Description() != "Test command" {
		t.Errorf("Expected description 'Test command', got '%s'", cmd.Description())
	}
}

func TestDefaultCommandExecutor(t *testing.T) {
	executor := &DefaultCommandExecutor{}
	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.runFunc = func(ctx context.Context, args []string) error {
			return nil
		}

		err := executor.ExecuteWithContext(ctx, cmd, []string{"arg1"})
		if err != nil {
			t.Errorf("ExecuteWithContext should not return error: %v", err)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.validateFunc = func(args []string) error {
			return errors.New("validation failed")
		}

		err := executor.ExecuteWithContext(ctx, cmd, []string{"arg1"})
		if err == nil {
			t.Error("ExecuteWithContext should return error on validation failure")
		}
	})

	t.Run("execution failure", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.runFunc = func(ctx context.Context, args []string) error {
			return errors.New("execution failed")
		}

		err := executor.ExecuteWithContext(ctx, cmd, []string{"arg1"})
		if err == nil {
			t.Error("ExecuteWithContext should return error on execution failure")
		}
	})
}

func TestValidateCommand(t *testing.T) {
	executor := &DefaultCommandExecutor{}

	t.Run("successful validation", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")

		err := executor.ValidateCommand(cmd, []string{"arg1"})
		if err != nil {
			t.Errorf("ValidateCommand should not return error: %v", err)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		cmd := NewMockCommand("test", "Test command", "test [args]")
		cmd.validateFunc = func(args []string) error {
			return errors.New("validation failed")
		}

		err := executor.ValidateCommand(cmd, []string{"arg1"})
		if err == nil {
			t.Error("ValidateCommand should return error on validation failure")
		}
	})
}

func TestCommandRegistryWithMultipleCategories(t *testing.T) {
	registry := NewCommandRegistry()
	rootCmd := &cobra.Command{Use: "root"}

	// Add multiple categories with different numbers of commands
	registry.RegisterCategory("single", "Single command category")
	cmd1 := NewMockCommand("singlecmd", "Single command", "singlecmd [args]")
	registry.AddCommand("single", cmd1)

	registry.RegisterCategory("double", "Double command category")
	cmd2 := NewMockCommand("cmd1", "Command 1", "cmd1 [args]")
	cmd3 := NewMockCommand("cmd2", "Command 2", "cmd2 [args]")
	registry.AddCommand("double", cmd2)
	registry.AddCommand("double", cmd3)

	registry.RegisterCategory("triple", "Triple command category")
	cmd4 := NewMockCommand("cmd1", "Command 1", "cmd1 [args]")
	cmd5 := NewMockCommand("cmd2", "Command 2", "cmd2 [args]")
	cmd6 := NewMockCommand("cmd3", "Command 3", "cmd3 [args]")
	registry.AddCommand("triple", cmd4)
	registry.AddCommand("triple", cmd5)
	registry.AddCommand("triple", cmd6)

	registry.CreateCobraCommands(rootCmd)

	// Should have 3 commands total (1 single + 1 double parent + 1 triple parent)
	if len(rootCmd.Commands()) != 3 {
		t.Errorf("Expected 3 commands in root, got %d", len(rootCmd.Commands()))
	}
}

func TestCommandCategoryCommands(t *testing.T) {
	category := &CommandCategory{
		Name:        "test",
		Description: "Test category",
		Commands:    make([]Command, 0),
	}

	cmd1 := NewMockCommand("cmd1", "Command 1", "cmd1 [args]")
	cmd2 := NewMockCommand("cmd2", "Command 2", "cmd2 [args]")

	category.Commands = append(category.Commands, cmd1, cmd2)

	if len(category.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(category.Commands))
	}

	if category.Commands[0] != cmd1 {
		t.Error("First command should be cmd1")
	}

	if category.Commands[1] != cmd2 {
		t.Error("Second command should be cmd2")
	}
}

func TestCommandRegistryConcurrency(t *testing.T) {
	registry := NewCommandRegistry()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()

			categoryName := "category" + string(rune(i))
			category := registry.RegisterCategory(categoryName, "Test category")

			cmd := NewMockCommand("cmd"+string(rune(i)), "Test command", "cmd [args]")
			registry.AddCommand(categoryName, cmd)

			// Verify category was created
			retrieved := registry.GetCategory(categoryName)
			if retrieved != category {
				t.Errorf("Category should be retrievable")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all categories were created
	if len(registry.categories) != 10 {
		t.Errorf("Expected 10 categories, got %d", len(registry.categories))
	}
}

func TestCommandRegistryPerformance(t *testing.T) {
	registry := NewCommandRegistry()

	// Test performance with many categories and commands
	numCategories := 100
	numCommandsPerCategory := 10

	start := time.Now()

	for i := 0; i < numCategories; i++ {
		categoryName := "category" + string(rune(i))
		registry.RegisterCategory(categoryName, "Test category")

		for j := 0; j < numCommandsPerCategory; j++ {
			cmdName := "cmd" + string(rune(j))
			cmd := NewMockCommand(cmdName, "Test command", "cmd [args]")
			registry.AddCommand(categoryName, cmd)
		}
	}

	duration := time.Since(start)

	// Performance requirement: should create 1000 commands in < 1s
	if duration > time.Second {
		t.Errorf("Creating %d categories with %d commands each took too long: %v",
			numCategories, numCommandsPerCategory, duration)
	}

	// Verify all categories were created
	if len(registry.categories) != numCategories {
		t.Errorf("Expected %d categories, got %d", numCategories, len(registry.categories))
	}

	// Verify commands were added
	totalCommands := 0
	for _, category := range registry.categories {
		totalCommands += len(category.Commands)
	}

	expectedCommands := numCategories * numCommandsPerCategory
	if totalCommands != expectedCommands {
		t.Errorf("Expected %d total commands, got %d", expectedCommands, totalCommands)
	}
}

// Benchmark tests
func BenchmarkNewCommandRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewCommandRegistry()
	}
}

func BenchmarkRegisterCategory(b *testing.B) {
	registry := NewCommandRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.RegisterCategory("test"+string(rune(i)), "Test category")
	}
}

func BenchmarkAddCommand(b *testing.B) {
	registry := NewCommandRegistry()
	registry.RegisterCategory("test", "Test category")
	cmd := NewMockCommand("testcmd", "Test command", "testcmd [args]")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.AddCommand("test", cmd)
	}
}

func BenchmarkGetCategory(b *testing.B) {
	registry := NewCommandRegistry()
	registry.RegisterCategory("test", "Test category")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetCategory("test")
	}
}

func BenchmarkCreateCobraCommands(b *testing.B) {
	registry := NewCommandRegistry()
	registry.RegisterCategory("test", "Test category")
	cmd := NewMockCommand("testcmd", "Test command", "testcmd [args]")
	registry.AddCommand("test", cmd)

	rootCmd := &cobra.Command{Use: "root"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.CreateCobraCommands(rootCmd)
	}
}
