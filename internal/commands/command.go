package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/container"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// ContextKey represents a custom type for context keys
type ContextKey string

// Command represents a unified command interface
type Command interface {
	Name() string
	Description() string
	Usage() string
	Examples() []string
	Execute(ctx context.Context, args []string) error
	Validate(args []string) error
	CreateCobraCommand() *cobra.Command
}

// Flag represents a command flag
type Flag struct {
	Name        string
	Short       string
	Type        string // "bool", "string", "int"
	Default     interface{}
	Description string
}

// BaseCommand provides common functionality for all commands
type BaseCommand struct {
	// Command metadata
	name        string
	description string
	usage       string
	examples    []string
	flags       []Flag

	// Dependencies
	container *container.SimpleContainer
	logger    observability.Logger

	// Command state
	startTime time.Time
	ctx       context.Context
}

// NewBaseCommand creates a new base command
func NewBaseCommand(name, description, usage string) *BaseCommand {
	return &BaseCommand{
		name:        name,
		description: description,
		usage:       usage,
		container:   container.GetGlobalSimpleContainer(),
		logger:      observability.NewDefaultLogger(),
	}
}

// WithExamples adds examples to the command
func (c *BaseCommand) WithExamples(examples ...string) *BaseCommand {
	c.examples = examples
	return c
}

// WithFlags adds flags to the command
func (c *BaseCommand) WithFlags(flags ...Flag) *BaseCommand {
	c.flags = append(c.flags, flags...)
	return c
}

// GetContainer returns the service container
func (c *BaseCommand) GetContainer() *container.SimpleContainer {
	return c.container
}

// GetLogger returns the logger instance
func (c *BaseCommand) GetLogger() observability.Logger {
	return c.logger
}

// GetContext returns the command context
func (c *BaseCommand) GetContext() context.Context {
	if c.ctx == nil {
		c.ctx = context.Background()
	}
	return c.ctx
}

// SetContext sets the command context
func (c *BaseCommand) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// Name returns the command name
func (c *BaseCommand) Name() string {
	return c.name
}

// Description returns the command description
func (c *BaseCommand) Description() string {
	return c.description
}

// Usage returns the command usage
func (c *BaseCommand) Usage() string {
	return c.usage
}

// Examples returns the command examples
func (c *BaseCommand) Examples() []string {
	return c.examples
}

// Execute runs the command with proper error handling and logging
func (c *BaseCommand) Execute(ctx context.Context, args []string) error {
	c.startTime = time.Now()
	c.ctx = ctx

	// Log command execution
	c.logger.Info(ctx, "executing_command",
		observability.F("command", c.name),
		observability.F("args", args),
	)

	// Validate arguments
	if err := c.Validate(args); err != nil {
		return c.wrapError(err, "validation_failed")
	}

	// Execute the command
	err := c.Run(ctx, args)

	// Log completion
	duration := time.Since(c.startTime)
	if err != nil {
		observability.LogCommandError(ctx, c.logger, c.name, err, duration)
		return c.wrapError(err, "execution_failed")
	}

	observability.LogCommandSuccess(ctx, c.logger, c.name, duration)
	return nil
}

// Validate validates command arguments
func (c *BaseCommand) Validate(args []string) error {
	// Default validation - can be overridden by subcommands
	return nil
}

// Run executes the command logic
func (c *BaseCommand) Run(ctx context.Context, args []string) error {
	// This should be implemented by subcommands
	return fmt.Errorf("run method not implemented")
}

// wrapError wraps an error with command context
func (c *BaseCommand) wrapError(err error, code string) error {
	if err == nil {
		return nil
	}

	// If it's already a GitPersonaError, enhance it
	if gpe, ok := err.(*errors.GitPersonaError); ok {
		return gpe.WithContext("command", c.name)
	}

	// Create a new error with context
	return errors.Wrap(err, errors.ErrorCode(code), fmt.Sprintf("command '%s' failed", c.name)).
		WithContext("command", c.name).
		WithContext("args", c.ctx.Value("args"))
}

// CreateCobraCommand creates a Cobra command from this base command
func (c *BaseCommand) CreateCobraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     c.usage,
		Short:   c.description,
		Long:    c.description,
		Example: c.formatExamples(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Store args in context for error reporting
			ctx := context.WithValue(c.ctx, ContextKey("args"), args)
			return c.Execute(ctx, args)
		},
	}

	// Add flags
	c.addFlags(cmd)

	return cmd
}

// addFlags adds flags to the command
func (c *BaseCommand) addFlags(cmd *cobra.Command) {
	for _, flag := range c.flags {
		switch flag.Type {
		case "bool":
			cmd.Flags().BoolP(flag.Name, flag.Short, flag.Default.(bool), flag.Description)
		case "string":
			cmd.Flags().StringP(flag.Name, flag.Short, flag.Default.(string), flag.Description)
		case "int":
			cmd.Flags().IntP(flag.Name, flag.Short, flag.Default.(int), flag.Description)
		}
	}

	// Add common flags
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	cmd.Flags().StringP("config", "c", "", "Configuration file path")
}

// formatExamples formats the examples for display
func (c *BaseCommand) formatExamples() string {
	if len(c.examples) == 0 {
		return ""
	}

	result := ""
	for i, example := range c.examples {
		if i > 0 {
			result += "\n"
		}
		result += fmt.Sprintf("  %s", example)
	}
	return result
}

// GetFlagBool gets a boolean flag value
func (c *BaseCommand) GetFlagBool(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}

// GetFlagString gets a string flag value
func (c *BaseCommand) GetFlagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

// GetFlagInt gets an integer flag value
func (c *BaseCommand) GetFlagInt(cmd *cobra.Command, name string) int {
	value, _ := cmd.Flags().GetInt(name)
	return value
}

// RequireArgs ensures the command has the required number of arguments
func (c *BaseCommand) RequireArgs(cmd *cobra.Command, min, max int) error {
	args := cmd.Flags().Args()
	if len(args) < min {
		return errors.New(errors.ErrCodeMissingRequired, fmt.Sprintf("at least %d arguments required", min))
	}
	if max > 0 && len(args) > max {
		return errors.New(errors.ErrCodeInvalidInput, fmt.Sprintf("at most %d arguments allowed", max))
	}
	return nil
}

// PrintInfo prints an info message
func (c *BaseCommand) PrintInfo(ctx context.Context, message string, fields ...observability.Field) {
	c.logger.Info(ctx, message, fields...)
}

// PrintSuccess prints a success message
func (c *BaseCommand) PrintSuccess(ctx context.Context, message string, fields ...observability.Field) {
	c.logger.Info(ctx, "success: "+message, fields...)
}

// PrintWarning prints a warning message
func (c *BaseCommand) PrintWarning(ctx context.Context, message string, fields ...observability.Field) {
	c.logger.Warn(ctx, "warning: "+message, fields...)
}

// PrintError prints an error message
func (c *BaseCommand) PrintError(ctx context.Context, message string, fields ...observability.Field) {
	c.logger.Error(ctx, "error: "+message, fields...)
}
