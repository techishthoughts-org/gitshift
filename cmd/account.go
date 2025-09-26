package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// accountCmd is the unified account management command
var accountCmd = &cobra.Command{
	Use:   "account [action]",
	Short: "👤 Unified account management",
	Long: `Complete account management system with hierarchical operations:

📋 ACCOUNT OPERATIONS:
  list                - List all accounts
  show [alias]        - Show detailed account information
  current             - Show current active account

➕ ACCOUNT CREATION:
  add [alias]         - Add new account interactively
  create [alias]      - Create account with all options
  import [file]       - Import accounts from file

🔄 ACCOUNT SWITCHING:
  switch [alias]      - Switch to account
  use [alias]         - Alias for switch (shorter command)

✏️  ACCOUNT MODIFICATION:
  update [alias]      - Update account settings
  rename [old] [new]  - Rename account alias
  delete [alias]      - Delete account

🔍 ACCOUNT VALIDATION:
  validate [alias]    - Validate single account
  validate-all        - Validate all accounts
  discover            - Discover accounts from system

🚀 QUICK ACTIONS:
  setup               - Interactive account setup wizard
  clone [url]         - Auto-detect and switch for cloning

Examples:
  gitpersona account list
  gitpersona account add work
  gitpersona account switch personal
  gitpersona account validate work
  gitpersona account setup

Use 'gitpersona account [action] --help' for detailed information about each action.`,
	Args: cobra.MinimumNArgs(1),
	RunE: accountHandler,
}

// Account action subcommands
var (
	accountListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all configured accounts",
		Long:  `Display a comprehensive list of all configured accounts with status information.`,
		RunE:  accountListHandler,
	}

	accountShowCmd = &cobra.Command{
		Use:   "show [alias]",
		Short: "Show detailed account information",
		Long:  `Display detailed information about a specific account or current account.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE:  accountShowHandler,
	}

	accountCurrentCmd = &cobra.Command{
		Use:     "current",
		Aliases: []string{"active"},
		Short:   "Show current active account",
		Long:    `Display information about the currently active account.`,
		RunE:    accountCurrentHandler,
	}

	accountAddCmd = &cobra.Command{
		Use:   "add [alias]",
		Short: "Add new account interactively",
		Long: `Add a new account with interactive prompts for all required information.
This is the user-friendly way to create accounts with guided setup.`,
		Args: cobra.RangeArgs(0, 1),
		RunE: accountAddHandler,
	}

	accountCreateCmd = &cobra.Command{
		Use:   "create [alias]",
		Short: "Create account with all options",
		Long: `Create a new account with command-line flags for all options.
This is the power-user way to create accounts with full control.`,
		Args: cobra.ExactArgs(1),
		RunE: accountCreateHandler,
	}

	accountImportCmd = &cobra.Command{
		Use:   "import [file]",
		Short: "Import accounts from configuration file",
		Long:  `Import multiple accounts from a YAML or JSON configuration file.`,
		Args:  cobra.ExactArgs(1),
		RunE:  accountImportHandler,
	}

	accountSwitchCmd = &cobra.Command{
		Use:     "switch [alias]",
		Aliases: []string{"use", "activate"},
		Short:   "Switch to specified account",
		Long:    `Switch the active account to the specified alias.`,
		Args:    cobra.ExactArgs(1),
		RunE:    accountSwitchHandler,
	}

	accountUpdateCmd = &cobra.Command{
		Use:   "update [alias]",
		Short: "Update account settings",
		Long:  `Update settings for an existing account.`,
		Args:  cobra.ExactArgs(1),
		RunE:  accountUpdateHandler,
	}

	accountRenameCmd = &cobra.Command{
		Use:   "rename [old-alias] [new-alias]",
		Short: "Rename account alias",
		Long:  `Rename an existing account alias to a new name.`,
		Args:  cobra.ExactArgs(2),
		RunE:  accountRenameHandler,
	}

	accountDeleteCmd = &cobra.Command{
		Use:     "delete [alias]",
		Aliases: []string{"remove", "rm"},
		Short:   "Delete account",
		Long:    `Delete an existing account configuration.`,
		Args:    cobra.ExactArgs(1),
		RunE:    accountDeleteHandler,
	}

	accountValidateCmd = &cobra.Command{
		Use:   "validate [alias]",
		Short: "Validate single account",
		Long:  `Validate the configuration and connectivity of a single account.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE:  accountValidateHandler,
	}

	accountValidateAllCmd = &cobra.Command{
		Use:   "validate-all",
		Short: "Validate all accounts",
		Long:  `Validate the configuration and connectivity of all accounts.`,
		RunE:  accountValidateAllHandler,
	}

	accountDiscoverCmd = &cobra.Command{
		Use:   "discover",
		Short: "Discover accounts from system",
		Long:  `Automatically discover potential accounts from SSH keys, Git config, and other sources.`,
		RunE:  accountDiscoverHandler,
	}

	accountSetupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Interactive account setup wizard",
		Long:  `Launch the interactive setup wizard to configure your first account or add additional accounts.`,
		RunE:  accountSetupHandler,
	}

	accountCloneCmd = &cobra.Command{
		Use:   "clone [repository-url]",
		Short: "Auto-detect account and clone repository",
		Long:  `Automatically detect the appropriate account for a repository and clone it.`,
		Args:  cobra.ExactArgs(1),
		RunE:  accountCloneHandler,
	}
)

func init() {
	rootCmd.AddCommand(accountCmd)

	// Add all subcommands
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountCurrentCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountCreateCmd)
	accountCmd.AddCommand(accountImportCmd)
	accountCmd.AddCommand(accountSwitchCmd)
	accountCmd.AddCommand(accountUpdateCmd)
	accountCmd.AddCommand(accountRenameCmd)
	accountCmd.AddCommand(accountDeleteCmd)
	accountCmd.AddCommand(accountValidateCmd)
	accountCmd.AddCommand(accountValidateAllCmd)
	accountCmd.AddCommand(accountDiscoverCmd)
	accountCmd.AddCommand(accountSetupCmd)
	accountCmd.AddCommand(accountCloneCmd)

	// Global account flags
	accountCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	accountCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	accountCmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")

	// List command flags
	accountListCmd.Flags().Bool("status", false, "Show account status information")
	accountListCmd.Flags().Bool("detailed", false, "Show detailed account information")
	accountListCmd.Flags().String("filter", "", "Filter accounts by status (active, pending, disabled)")

	// Create command flags (for power users)
	accountCreateCmd.Flags().StringP("name", "n", "", "Full name for Git commits")
	accountCreateCmd.Flags().StringP("email", "e", "", "Email address for Git commits")
	accountCreateCmd.Flags().StringP("github-username", "g", "", "GitHub username")
	accountCreateCmd.Flags().StringP("ssh-key", "k", "", "Path to SSH private key")
	accountCreateCmd.Flags().StringP("description", "d", "", "Account description")
	accountCreateCmd.Flags().Bool("default", false, "Set as default account")
	accountCreateCmd.Flags().Bool("generate-key", false, "Generate new SSH key")

	// Update command flags
	accountUpdateCmd.Flags().StringP("name", "n", "", "Update full name")
	accountUpdateCmd.Flags().StringP("email", "e", "", "Update email address")
	accountUpdateCmd.Flags().StringP("github-username", "g", "", "Update GitHub username")
	accountUpdateCmd.Flags().StringP("ssh-key", "k", "", "Update SSH key path")
	accountUpdateCmd.Flags().StringP("description", "d", "", "Update description")

	// Validation flags
	accountValidateCmd.Flags().Bool("fix", false, "Attempt to fix validation issues")
	accountValidateCmd.Flags().Bool("ssh", true, "Validate SSH connectivity")
	accountValidateCmd.Flags().Bool("github", true, "Validate GitHub access")
	accountValidateAllCmd.Flags().Bool("fix", false, "Attempt to fix validation issues")
	accountValidateAllCmd.Flags().Bool("parallel", true, "Run validations in parallel")

	// Discovery flags
	accountDiscoverCmd.Flags().Bool("interactive", true, "Prompt for each discovered account")
	accountDiscoverCmd.Flags().Bool("auto-add", false, "Automatically add all discovered accounts")
	accountDiscoverCmd.Flags().StringSlice("sources", []string{"ssh", "git", "github"}, "Discovery sources")

	// Delete command flags
	accountDeleteCmd.Flags().Bool("force", false, "Force deletion without confirmation")
	accountDeleteCmd.Flags().Bool("keep-keys", false, "Keep SSH keys when deleting account")

	// Clone command flags
	accountCloneCmd.Flags().StringP("directory", "d", "", "Directory to clone into")
	accountCloneCmd.Flags().StringP("account", "a", "", "Force specific account (skip auto-detection)")
}

// Main handler that routes to appropriate action
func accountHandler(cmd *cobra.Command, args []string) error {
	// This should not be called directly since we have subcommands
	return cmd.Help()
}

// Handler implementations

func accountAddHandler(cmd *cobra.Command, args []string) error {
	var alias string
	if len(args) > 0 {
		alias = args[0]
	}

	fmt.Printf("➕ Adding account interactively: %s\n", alias)

	// TODO: Implement interactive account creation
	// This will prompt for all required information
	return fmt.Errorf("not implemented yet - will provide interactive account creation")
}

func accountImportHandler(cmd *cobra.Command, args []string) error {
	filename := args[0]

	fmt.Printf("📥 Importing accounts from file: %s\n", filename)

	// TODO: Implement account import from YAML/JSON file
	return fmt.Errorf("not implemented yet - will implement file import")
}

func accountRenameHandler(cmd *cobra.Command, args []string) error {
	oldAlias := args[0]
	newAlias := args[1]

	fmt.Printf("📝 Renaming account: %s -> %s\n", oldAlias, newAlias)

	// TODO: Implement account renaming
	return fmt.Errorf("not implemented yet - will implement account renaming")
}

func accountDiscoverHandler(cmd *cobra.Command, args []string) error {
	interactive, _ := cmd.Flags().GetBool("interactive")
	autoAdd, _ := cmd.Flags().GetBool("auto-add")
	sources, _ := cmd.Flags().GetStringSlice("sources")

	fmt.Printf("🔍 Discovering accounts (interactive: %v, auto-add: %v, sources: %v)\n",
		interactive, autoAdd, sources)

	// TODO: Implement using CoreServices.Account.DiscoverAccounts()
	return fmt.Errorf("not implemented yet - will use CoreServices.Account.DiscoverAccounts()")
}

func accountSetupHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Starting interactive account setup wizard")

	// TODO: Implement interactive setup wizard
	return fmt.Errorf("not implemented yet - will implement setup wizard")
}

func accountCloneHandler(cmd *cobra.Command, args []string) error {
	repoURL := args[0]
	directory, _ := cmd.Flags().GetString("directory")
	forcedAccount, _ := cmd.Flags().GetString("account")

	fmt.Printf("📥 Auto-cloning repository: %s (dir: %s, account: %s)\n", repoURL, directory, forcedAccount)

	// TODO: Implement smart account detection and cloning
	return fmt.Errorf("not implemented yet - will implement smart cloning")
}

// Hide old account-related commands to reduce noise during transition
func init() {
	// Mark legacy commands as hidden
	hideCommand("add")
	hideCommand("remove")
	hideCommand("list")
	hideCommand("switch")
	hideCommand("current")
}

func hideCommand(cmdName string) {
	// Helper to hide legacy commands during transition
	// This would find and hide the old standalone commands
	fmt.Printf("TODO: Hide legacy command: %s\n", cmdName)
}
