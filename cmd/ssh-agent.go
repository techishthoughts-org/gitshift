package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
	"github.com/techishthoughts/GitPersona/internal/errors"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHAgentCommand handles SSH agent management
type SSHAgentCommand struct {
	*commands.BaseCommand

	// Command-specific flags
	status  bool
	clear   bool
	load    string
	cleanup bool
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
			"gitpersona ssh-agent cleanup",
		).WithFlags(
			commands.Flag{Name: "status", Short: "s", Type: "bool", Default: false, Description: "Show SSH agent status"},
			commands.Flag{Name: "clear", Short: "c", Type: "bool", Default: false, Description: "Clear all keys from SSH agent"},
			commands.Flag{Name: "load", Short: "l", Type: "string", Default: "", Description: "Load a specific SSH key"},
			commands.Flag{Name: "cleanup", Short: "k", Type: "bool", Default: false, Description: "Clean up SSH sockets to prevent authentication conflicts"},
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
	cmd.Flags().BoolVarP(&c.cleanup, "cleanup", "k", false, "Clean up SSH sockets to prevent authentication conflicts")

	// Override the RunE to use our command structure
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Get flag values
		c.status = c.GetFlagBool(cmd, "status")
		c.clear = c.GetFlagBool(cmd, "clear")
		c.load = c.GetFlagString(cmd, "load")
		c.cleanup = c.GetFlagBool(cmd, "cleanup")

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

	if c.cleanup {
		fmt.Printf("DEBUG: Cleanup flag is set, calling cleanupSockets\n")
		return c.cleanupSockets(ctx, sshAgentService)
	}

	fmt.Printf("DEBUG: Cleanup flag is NOT set, cleanup=%v\n", c.cleanup)

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

// cleanupSockets cleans up SSH sockets to prevent authentication conflicts
func (c *SSHAgentCommand) cleanupSockets(ctx context.Context, sshAgentService interface{}) error {
	c.PrintInfo(ctx, "Cleaning up SSH sockets...")

	fmt.Printf("DEBUG: SSH agent service type: %T\n", sshAgentService)

	if service, ok := sshAgentService.(interface {
		CleanupSSHSockets(ctx context.Context) error
	}); ok {
		fmt.Printf("DEBUG: Service implements CleanupSSHSockets interface\n")
		if err := service.CleanupSSHSockets(ctx); err != nil {
			return fmt.Errorf("failed to cleanup SSH sockets: %w", err)
		}
	} else {
		fmt.Printf("DEBUG: Service does NOT implement CleanupSSHSockets interface\n")
	}

	c.PrintSuccess(ctx, "SSH sockets cleaned up successfully")
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
are loaded in the SSH agent and cleaning up SSH sockets.

Examples:
  gitpersona ssh-agent --status            # Show current agent status
  gitpersona ssh-agent --clear             # Clear all keys from agent
  gitpersona ssh-agent --load ~/.ssh/id_rsa # Load a specific key
  gitpersona ssh-agent --cleanup           # Clean up SSH sockets`,
		Args: cobra.NoArgs,
		RunE: runSSHAgent,
	}
)

func init() {
	// Add flags to the command
	sshAgentCmd.Flags().BoolP("status", "s", false, "Show SSH agent status")
	sshAgentCmd.Flags().BoolP("clear", "c", false, "Clear all keys from SSH agent")
	sshAgentCmd.Flags().StringP("load", "l", "", "Load a specific SSH key")
	sshAgentCmd.Flags().BoolP("cleanup", "k", false, "Clean up SSH sockets to prevent authentication conflicts")

	rootCmd.AddCommand(sshAgentCmd)
}

// runSSHAgent runs the SSH agent command
func runSSHAgent(cmd *cobra.Command, args []string) error {
	// Get flag values
	status, _ := cmd.Flags().GetBool("status")
	clear, _ := cmd.Flags().GetBool("clear")
	load, _ := cmd.Flags().GetString("load")
	cleanup, _ := cmd.Flags().GetBool("cleanup")

	// Handle cleanup flag directly
	if cleanup {
		return cleanupSSHSocketsDirect()
	}

	// Handle other flags
	if status {
		return showSSHAgentStatus()
	}

	if clear {
		return clearSSHAgentKeys()
	}

	if load != "" {
		return loadSSHKey(load)
	}

	// Default: show status
	return showSSHAgentStatus()
}

// cleanupSSHSocketsDirect cleans up SSH sockets directly
func cleanupSSHSocketsDirect() error {
	fmt.Println("🔧 Cleaning up SSH sockets...")

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Common SSH socket locations
	socketPaths := []string{
		filepath.Join(homeDir, ".ssh", "socket"),
		filepath.Join(homeDir, ".ssh", "sockets"),
		filepath.Join(homeDir, ".ssh", "control"),
	}

	cleanedCount := 0
	for _, socketPath := range socketPaths {
		if _, err := os.Stat(socketPath); err == nil {
			fmt.Printf("  🗑️  Removing: %s\n", socketPath)
			if err := os.RemoveAll(socketPath); err == nil {
				cleanedCount++
			}
		}
	}

	// Handle glob patterns for /tmp/ssh-*
	matches, _ := filepath.Glob("/tmp/ssh-*")
	for _, match := range matches {
		fmt.Printf("  🗑️  Removing: %s\n", match)
		if err := os.RemoveAll(match); err == nil {
			cleanedCount++
		}
	}

	// Try to close existing SSH connections
	fmt.Println("  🔌 Attempting to close existing SSH connections...")
	_ = exec.Command("ssh", "-O", "exit", "github.com").Run()
	_ = exec.Command("ssh", "-O", "exit", "gitlab.com").Run()
	_ = exec.Command("ssh", "-O", "exit", "bitbucket.org").Run()

	// Ensure socket directories exist after cleanup
	fmt.Println("  📁 Ensuring socket directories exist...")
	for _, socketPath := range socketPaths {
		if err := os.MkdirAll(socketPath, 0700); err != nil {
			fmt.Printf("  ⚠️  Failed to create socket directory %s: %v\n", socketPath, err)
		} else {
			fmt.Printf("  ✅ Socket directory ensured: %s\n", socketPath)
		}
	}

	fmt.Printf("✅ SSH socket cleanup completed (cleaned %d items)\n", cleanedCount)
	return nil
}

// showSSHAgentStatus shows the current SSH agent status
func showSSHAgentStatus() error {
	fmt.Println("🔑 SSH Agent Status:")
	fmt.Println("===================")

	// Check if agent is running
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("❌ SSH agent is not running or no keys loaded")
		return nil
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "The agent has no identities." {
		fmt.Println("ℹ️  SSH agent is running but no keys loaded")
	} else {
		fmt.Println("✅ SSH agent is running with keys:")
		lines := strings.Split(outputStr, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("  %d. %s\n", i+1, line)
			}
		}
	}

	return nil
}

// clearSSHAgentKeys clears all keys from the SSH agent
func clearSSHAgentKeys() error {
	fmt.Println("🧹 Clearing all SSH keys from agent...")

	cmd := exec.Command("ssh-add", "-D")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clear SSH keys: %w", err)
	}

	fmt.Println("✅ All SSH keys cleared from agent")
	return nil
}

// loadSSHKey loads a specific SSH key into the agent
func loadSSHKey(keyPath string) error {
	fmt.Printf("🔑 Loading SSH key: %s\n", keyPath)

	// Check if key exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key file does not exist: %s", keyPath)
	}

	cmd := exec.Command("ssh-add", keyPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load SSH key: %w", err)
	}

	fmt.Printf("✅ SSH key loaded successfully: %s\n", keyPath)
	return nil
}
