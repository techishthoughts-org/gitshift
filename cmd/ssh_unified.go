package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// sshUnifiedCmd is the new unified SSH command that replaces all SSH-related commands
var sshUnifiedCmd = &cobra.Command{
	Use:   "ssh [action]",
	Short: "🔐 Unified SSH management and troubleshooting",
	Long: `Complete SSH management system with all operations unified:

🔑 KEY MANAGEMENT:
  keys list           - List all SSH keys
  keys generate       - Generate new SSH key
  keys delete         - Delete SSH key
  keys validate       - Validate SSH key

⚙️  CONFIG MANAGEMENT:
  config show         - Show SSH configuration
  config generate     - Generate SSH config entries
  config install      - Install SSH config

🔧 TROUBLESHOOTING:
  test [account]      - Test SSH connectivity
  diagnose            - Comprehensive diagnostics
  fix                 - Auto-fix SSH issues

🚀 AGENT MANAGEMENT:
  agent status        - Check SSH agent status
  agent start         - Start SSH agent
  agent stop          - Stop SSH agent
  agent load [key]    - Load key into agent

Examples:
  gitpersona ssh keys list
  gitpersona ssh test work
  gitpersona ssh diagnose --verbose
  gitpersona ssh config generate --account work
  gitpersona ssh fix --auto

Use 'gitpersona ssh [action] --help' for detailed information about each action.`,
	Args: cobra.MinimumNArgs(1),
	RunE: sshUnifiedHandler,
}

// SSH action subcommands
var (
	sshUnifiedKeysCmd = &cobra.Command{
		Use:   "keys [action]",
		Short: "SSH key management operations",
		Long: `Manage SSH keys with the following actions:
  list      - List all SSH keys
  generate  - Generate new SSH key
  delete    - Delete SSH key
  validate  - Validate SSH key`,
	}

	sshUnifiedConfigCmd = &cobra.Command{
		Use:   "config [action]",
		Short: "SSH configuration management",
		Long: `Manage SSH configuration with the following actions:
  show      - Show current SSH configuration
  generate  - Generate SSH config entries
  install   - Install SSH configuration`,
	}

	sshUnifiedTestCmd = &cobra.Command{
		Use:   "test [account]",
		Short: "Test SSH connectivity",
		Long:  `Test SSH connectivity for the specified account or current account.`,
		Args:  cobra.RangeArgs(0, 1),
		RunE:  sshTestHandler,
	}

	sshUnifiedDiagnoseCmd = &cobra.Command{
		Use:   "diagnose",
		Short: "Run comprehensive SSH diagnostics",
		Long:  `Perform comprehensive SSH diagnostics and report issues.`,
		RunE:  sshDiagnoseHandler,
	}

	sshUnifiedFixCmd = &cobra.Command{
		Use:   "fix",
		Short: "Auto-fix SSH issues",
		Long:  `Automatically fix common SSH configuration issues.`,
		RunE:  sshFixHandler,
	}

	sshUnifiedAgentCmd = &cobra.Command{
		Use:   "agent [action]",
		Short: "SSH agent management",
		Long: `Manage SSH agent with the following actions:
  status    - Check SSH agent status
  start     - Start SSH agent
  stop      - Stop SSH agent
  load      - Load key into agent`,
	}
)

func init() {
	rootCmd.AddCommand(sshUnifiedCmd)

	// Add action subcommands
	sshUnifiedCmd.AddCommand(sshUnifiedKeysCmd)
	sshUnifiedCmd.AddCommand(sshUnifiedConfigCmd)
	sshUnifiedCmd.AddCommand(sshUnifiedTestCmd)
	sshUnifiedCmd.AddCommand(sshUnifiedDiagnoseCmd)
	sshUnifiedCmd.AddCommand(sshUnifiedFixCmd)
	sshUnifiedCmd.AddCommand(sshUnifiedAgentCmd)

	// Keys subcommands
	sshUnifiedKeysCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all SSH keys",
		RunE:  sshKeysListHandler,
	})
	sshUnifiedKeysCmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate new SSH key",
		RunE:  sshKeysGenerateHandler,
	})
	sshUnifiedKeysCmd.AddCommand(&cobra.Command{
		Use:   "delete [key-path]",
		Short: "Delete SSH key",
		Args:  cobra.ExactArgs(1),
		RunE:  sshKeysDeleteHandler,
	})
	sshUnifiedKeysCmd.AddCommand(&cobra.Command{
		Use:   "validate [key-path]",
		Short: "Validate SSH key",
		Args:  cobra.ExactArgs(1),
		RunE:  sshKeysValidateHandler,
	})

	// Config subcommands
	sshUnifiedConfigCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show SSH configuration",
		RunE:  sshConfigShowHandler,
	})
	sshUnifiedConfigCmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate SSH config entries",
		RunE:  sshConfigGenerateHandler,
	})
	sshUnifiedConfigCmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install SSH configuration",
		RunE:  sshConfigInstallHandler,
	})

	// Agent subcommands
	sshUnifiedAgentCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check SSH agent status",
		RunE:  sshAgentStatusHandler,
	})
	sshUnifiedAgentCmd.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start SSH agent",
		RunE:  sshAgentStartHandler,
	})
	sshUnifiedAgentCmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop SSH agent",
		RunE:  sshAgentStopHandler,
	})
	sshUnifiedAgentCmd.AddCommand(&cobra.Command{
		Use:   "load [key-path]",
		Short: "Load key into SSH agent",
		Args:  cobra.RangeArgs(0, 1),
		RunE:  sshAgentLoadHandler,
	})

	// Global flags
	sshUnifiedCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	sshUnifiedCmd.PersistentFlags().StringP("account", "a", "", "Target account (defaults to current)")
	sshUnifiedCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	sshUnifiedCmd.PersistentFlags().Bool("auto", false, "Enable automatic fixes where possible")

	// Specific flags
	sshUnifiedTestCmd.Flags().Int("timeout", 30, "Connection timeout in seconds")
	sshUnifiedDiagnoseCmd.Flags().Bool("full", false, "Run full diagnostic suite")
	sshUnifiedFixCmd.Flags().Bool("force", false, "Force fixes without confirmation")
	sshUnifiedKeysCmd.PersistentFlags().StringP("type", "t", "ed25519", "Key type (ed25519, rsa)")
	sshUnifiedConfigCmd.PersistentFlags().StringP("output", "o", "", "Output file path")
}

// Main handler that routes to appropriate action
func sshUnifiedHandler(cmd *cobra.Command, args []string) error {
	// This should not be called directly since we have subcommands
	return cmd.Help()
}

// Handler implementations

func sshConfigShowHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("📄 Showing SSH configuration")

	// TODO: Implement SSH config display using CoreServices.SSH.ValidateConfig()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.ValidateConfig()")
}

func sshConfigGenerateHandler(cmd *cobra.Command, args []string) error {
	accountName, _ := cmd.Flags().GetString("account")
	outputFile, _ := cmd.Flags().GetString("output")

	fmt.Printf("⚙️  Generating SSH config for account: %s, output: %s\n", accountName, outputFile)

	// TODO: Implement SSH config generation using CoreServices.SSH.GenerateConfig()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.GenerateConfig()")
}

func sshConfigInstallHandler(cmd *cobra.Command, args []string) error {
	fmt.Println("📦 Installing SSH configuration")

	// TODO: Implement SSH config installation using CoreServices.SSH.InstallConfig()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.InstallConfig()")
}

func sshAgentStatusHandler(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Printf("📊 Checking SSH agent status (verbose: %v)\n", verbose)

	// TODO: Implement SSH agent status check using CoreServices.SSH
	return fmt.Errorf("not implemented yet - will check SSH agent status")
}

func sshAgentStartHandler(cmd *cobra.Command, args []string) error {
	accountName, _ := cmd.Flags().GetString("account")

	fmt.Printf("▶️  Starting SSH agent for account: %s\n", accountName)

	// TODO: Implement SSH agent start using CoreServices.SSH.StartAgent()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.StartAgent()")
}

func sshAgentStopHandler(cmd *cobra.Command, args []string) error {
	accountName, _ := cmd.Flags().GetString("account")

	fmt.Printf("⏹️  Stopping SSH agent for account: %s\n", accountName)

	// TODO: Implement SSH agent stop using CoreServices.SSH.StopAgent()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.StopAgent()")
}

func sshAgentLoadHandler(cmd *cobra.Command, args []string) error {
	var keyPath string
	if len(args) > 0 {
		keyPath = args[0]
	}

	fmt.Printf("📂 Loading SSH key into agent: %s\n", keyPath)

	// TODO: Implement SSH key loading using CoreServices.SSH.LoadKey()
	return fmt.Errorf("not implemented yet - will use CoreServices.SSH.LoadKey()")
}

// Hide the old SSH commands to reduce noise
func init() {
	// Mark old SSH commands as hidden during transition
	if sshCmd != nil {
		sshCmd.Hidden = true
	}
	// Additional old commands to hide would be marked here
}
