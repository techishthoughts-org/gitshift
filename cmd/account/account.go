package account

import (
	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/commands"
)

// CreateAccountCommands creates all account-related commands organized under a parent command
func CreateAccountCommands() *cobra.Command {
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "Account management commands",
		Long: `Account management commands for GitPersona.

These commands allow you to manage your GitHub accounts including:
- Adding new accounts
- Removing accounts
- Listing all accounts
- Switching between accounts`,
		Example: `  gitpersona account add personal --email "me@personal.com" --name "My Name"
  gitpersona account list
  gitpersona account remove old-account
  gitpersona account switch work`,
	}

	// Register account commands using the command registry
	_ = commands.StandardCommandCategories()

	// Add sub-commands (these would be moved from cmd/ to cmd/account/)
	// accountCmd.AddCommand(NewAddCommand().CreateCobraCommand())
	// accountCmd.AddCommand(NewRemoveCommand().CreateCobraCommand())
	// accountCmd.AddCommand(NewListCommand().CreateCobraCommand())
	// accountCmd.AddCommand(NewSwitchCommand().CreateCobraCommand())

	return accountCmd
}

// AccountCommand is a base for all account-related commands
type AccountCommand struct {
	*commands.BaseCommand
	category string
}

// NewAccountCommand creates a new account command with proper categorization
func NewAccountCommand(name, description, usage string) *AccountCommand {
	return &AccountCommand{
		BaseCommand: commands.NewBaseCommand(name, description, usage),
		category:    "account",
	}
}

// GetCategory returns the command category
func (c *AccountCommand) GetCategory() string {
	return c.category
}
