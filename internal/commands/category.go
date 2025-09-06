package commands

import (
	"context"

	"github.com/spf13/cobra"
)

// CommandCategory groups related commands together
type CommandCategory struct {
	Name        string
	Description string
	Commands    []Command
}

// CommandRegistry manages all command categories
type CommandRegistry struct {
	categories map[string]*CommandCategory
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		categories: make(map[string]*CommandCategory),
	}
}

// RegisterCategory adds a new command category
func (r *CommandRegistry) RegisterCategory(name, description string) *CommandCategory {
	category := &CommandCategory{
		Name:        name,
		Description: description,
		Commands:    make([]Command, 0),
	}
	r.categories[name] = category
	return category
}

// AddCommand adds a command to a category
func (r *CommandRegistry) AddCommand(categoryName string, cmd Command) {
	if category, exists := r.categories[categoryName]; exists {
		category.Commands = append(category.Commands, cmd)
	}
}

// GetCategory returns a command category by name
func (r *CommandRegistry) GetCategory(name string) *CommandCategory {
	return r.categories[name]
}

// CreateCobraCommands creates all Cobra commands organized by category
func (r *CommandRegistry) CreateCobraCommands(rootCmd *cobra.Command) {
	for _, category := range r.categories {
		// Create a parent command for the category if it has multiple commands
		if len(category.Commands) > 1 {
			categoryCmd := &cobra.Command{
				Use:   category.Name,
				Short: category.Description,
			}

			for _, cmd := range category.Commands {
				categoryCmd.AddCommand(cmd.CreateCobraCommand())
			}

			rootCmd.AddCommand(categoryCmd)
		} else if len(category.Commands) == 1 {
			// Single command category - add directly to root
			rootCmd.AddCommand(category.Commands[0].CreateCobraCommand())
		}
	}
}

// StandardCommandCategories defines the standard command organization
func StandardCommandCategories() *CommandRegistry {
	registry := NewCommandRegistry()

	// Account Management
	registry.RegisterCategory("account", "Account management commands")

	// SSH Management
	registry.RegisterCategory("ssh", "SSH key and agent management")

	// Git Management
	registry.RegisterCategory("git", "Git configuration management")

	// System Diagnostics
	registry.RegisterCategory("system", "System diagnostics and health checks")

	// Project Management
	registry.RegisterCategory("project", "Project-specific configuration")

	return registry
}

// GroupedCommand represents a command that belongs to a specific group
type GroupedCommand struct {
	*BaseCommand
	Category string
	Group    string
}

// NewGroupedCommand creates a new grouped command
func NewGroupedCommand(category, group, name, description, usage string) *GroupedCommand {
	return &GroupedCommand{
		BaseCommand: NewBaseCommand(name, description, usage),
		Category:    category,
		Group:       group,
	}
}

// CommandExecutor defines how commands should be executed
type CommandExecutor interface {
	ExecuteWithContext(ctx context.Context, cmd Command, args []string) error
	ValidateCommand(cmd Command, args []string) error
}

// DefaultCommandExecutor provides the standard command execution flow
type DefaultCommandExecutor struct{}

// ExecuteWithContext executes a command with proper error handling
func (e *DefaultCommandExecutor) ExecuteWithContext(ctx context.Context, cmd Command, args []string) error {
	if err := e.ValidateCommand(cmd, args); err != nil {
		return err
	}
	return cmd.Execute(ctx, args)
}

// ValidateCommand validates a command before execution
func (e *DefaultCommandExecutor) ValidateCommand(cmd Command, args []string) error {
	return cmd.Validate(args)
}
