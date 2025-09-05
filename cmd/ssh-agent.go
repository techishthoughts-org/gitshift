package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHAgentCommand handles SSH agent management
type SSHAgentCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	status bool
	clear  bool
	load   string
}

// NewSSHAgentCommand creates a new SSH agent command
func NewSSHAgentCommand() *SSHAgentCommand {
	cmd := &SSHAgentCommand{
		BaseCommand: commands.NewBaseCommand(
			"ssh-agent",
			"🔑 Manage SSH agent and key loading",
			"ssh-agent [command]",
		).WithExamples(
			"gitpersona ssh-agent status",
			"gitpersona ssh-agent clear",
			"gitpersona ssh-agent load ~/.ssh/id_ed25519_thukabjj",
		).WithFlags(
			commands.Flag{Name: "status", Short: "s", Type: "bool", Default: false, Description: "Show SSH agent status"},
			commands.Flag{Name: "clear", Short: "c", Type: "bool", Default: false, Description: "Clear all keys from SSH agent"},
			commands.Flag{Name: "load", Short: "l", Type: "string", Default: "", Description: "Load a specific SSH key"},
		),
	}

	return cmd
}

// CreateCobraCommand creates the Cobra command
func (c *SSHAgentCommand) CreateCobraCommand() *cobra.Command {
	cmd := c.BaseCommand.CreateCobraCommand()

	// Add specific flags
	cmd.Flags().BoolVarP(&c.status, "status", "s", false, "Show SSH agent status")
	cmd.Flags().BoolVarP(&c.clear, "clear", "c", false, "Clear all keys from SSH agent")
	cmd.Flags().StringVarP(&c.load, "load", "l", "", "Load a specific SSH key")

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return c.Execute(ctx, args)
	}

	return cmd
}

// Validate validates the command arguments
func (c *SSHAgentCommand) Validate(args []string) error {
	// No validation needed for this command
	return nil
}

// Run executes the SSH agent command logic
func (c *SSHAgentCommand) Run(ctx context.Context, args []string) error {
	container := c.GetContainer()

	// Get SSH agent service
	sshAgentService := container.GetSSHAgentService()
	if sshAgentService == nil {
		return errors.New(errors.ErrCodeInternal, "SSH agent service not available")
	}

	// Handle different operations
	if c.status {
		return c.showStatus(ctx, sshAgentService)
	}

	if c.clear {
		return c.clearKeys(ctx, sshAgentService)
	}

	if c.load != "" {
		return c.loadKey(ctx, sshAgentService, c.load)
	}

	// Default: show status
	return c.showStatus(ctx, sshAgentService)
}

// showStatus shows the current SSH agent status
func (c *SSHAgentCommand) showStatus(ctx context.Context, sshAgentService interface{}) error {
	c.PrintInfo(ctx, "Checking SSH agent status...")

	// Check if agent is running
	if service, ok := sshAgentService.(interface {
		IsAgentRunning(ctx context.Context) (bool, error)
	}); ok {
		running, err := service.IsAgentRunning(ctx)
		if err != nil {
			return fmt.Errorf("failed to check agent status: %w", err)
		}

		if !running {
			c.PrintWarning(ctx, "SSH agent is not running")
			return nil
		}
	}

	// Get agent status
	if service, ok := sshAgentService.(interface {
		GetAgentStatus(ctx context.Context) (interface{}, error)
	}); ok {
		status, err := service.GetAgentStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get agent status: %w", err)
		}

		c.PrintSuccess(ctx, "SSH agent is running",
			observability.F("status", fmt.Sprintf("%+v", status)),
		)
	}

	// List loaded keys
	if service, ok := sshAgentService.(interface {
		ListLoadedKeys(ctx context.Context) ([]string, error)
	}); ok {
		keys, err := service.ListLoadedKeys(ctx)
		if err != nil {
			return fmt.Errorf("failed to list loaded keys: %w", err)
		}

		if len(keys) == 0 {
			c.PrintInfo(ctx, "No SSH keys loaded in agent")
		} else {
			c.PrintInfo(ctx, fmt.Sprintf("Loaded SSH keys (%d):", len(keys)))
			for i, key := range keys {
				c.PrintInfo(ctx, fmt.Sprintf("  %d. %s", i+1, key))
			}
		}
	}

	return nil
}

// clearKeys clears all keys from the SSH agent
func (c *SSHAgentCommand) clearKeys(ctx context.Context, sshAgentService interface{}) error {
	c.PrintInfo(ctx, "Clearing all SSH keys from agent...")

	if service, ok := sshAgentService.(interface {
		ClearAllKeys(ctx context.Context) error
	}); ok {
		if err := service.ClearAllKeys(ctx); err != nil {
			return fmt.Errorf("failed to clear SSH keys: %w", err)
		}
	}

	c.PrintSuccess(ctx, "All SSH keys cleared from agent")
	return nil
}

// loadKey loads a specific SSH key into the agent
func (c *SSHAgentCommand) loadKey(ctx context.Context, sshAgentService interface{}, keyPath string) error {
	c.PrintInfo(ctx, "Loading SSH key into agent...",
		observability.F("key_path", keyPath),
	)

	if service, ok := sshAgentService.(interface {
		LoadKey(ctx context.Context, keyPath string) error
	}); ok {
		if err := service.LoadKey(ctx, keyPath); err != nil {
			return fmt.Errorf("failed to load SSH key: %w", err)
		}
	}

	c.PrintSuccess(ctx, "SSH key loaded into agent",
		observability.F("key_path", keyPath),
	)
	return nil
}

// SSH agent command for integration
var (
	sshAgentCmd = &cobra.Command{
		Use:     "ssh-agent [command]",
		Aliases: []string{"agent"},
		Short:   "🔑 Manage SSH agent and key loading",
		Long: `🔑 Manage SSH Agent and Key Loading

This command helps you manage your SSH agent and the keys loaded in it.
It can help resolve SSH authentication conflicts by managing which keys
are loaded in the SSH agent.

Examples:
  gitpersona ssh-agent --status            # Show current agent status
  gitpersona ssh-agent --clear             # Clear all keys from agent
  gitpersona ssh-agent --load ~/.ssh/id_rsa # Load a specific key`,
		Args: cobra.NoArgs,
		RunE: runSSHAgent,
	}
)

func init() {
	// Add flags to the command
	sshAgentCmd.Flags().BoolP("status", "s", false, "Show SSH agent status")
	sshAgentCmd.Flags().BoolP("clear", "c", false, "Clear all keys from SSH agent")
	sshAgentCmd.Flags().StringP("load", "l", "", "Load a specific SSH key")

	rootCmd.AddCommand(sshAgentCmd)
}

// runSSHAgent runs the SSH agent command
func runSSHAgent(cmd *cobra.Command, args []string) error {
	// Get flag values
	status, _ := cmd.Flags().GetBool("status")
	clear, _ := cmd.Flags().GetBool("clear")
	load, _ := cmd.Flags().GetString("load")

	// Create and run the SSH agent command
	sshAgentCmd := NewSSHAgentCommand()
	sshAgentCmd.status = status
	sshAgentCmd.clear = clear
	sshAgentCmd.load = load

	ctx := context.Background()
	return sshAgentCmd.Execute(ctx, args)
}
