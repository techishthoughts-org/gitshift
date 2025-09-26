package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// CompatibilityManager handles backward compatibility for legacy commands
type CompatibilityManager struct {
	logger   observability.Logger
	services *CoreServices
	aliases  map[string]CommandAlias
}

// CommandAlias represents a legacy command mapping to new command structure
type CommandAlias struct {
	LegacyCommand   string            `json:"legacy_command"`
	NewCommand      string            `json:"new_command"`
	ArgumentMapping map[string]string `json:"argument_mapping"`
	Description     string            `json:"description"`
	DeprecationNote string            `json:"deprecation_note"`
	RemovalVersion  string            `json:"removal_version"`
}

// CompatibilityWarning represents a deprecation warning
type CompatibilityWarning struct {
	LegacyCommand  string `json:"legacy_command"`
	NewCommand     string `json:"new_command"`
	Message        string `json:"message"`
	RemovalVersion string `json:"removal_version"`
	ShowMigration  bool   `json:"show_migration"`
}

// NewCompatibilityManager creates a new compatibility manager
func NewCompatibilityManager(logger observability.Logger, services *CoreServices) *CompatibilityManager {
	cm := &CompatibilityManager{
		logger:   logger,
		services: services,
		aliases:  make(map[string]CommandAlias),
	}

	cm.initializeAliases()
	return cm
}

// CreateLegacyCommand creates a cobra command for legacy compatibility
func (cm *CompatibilityManager) CreateLegacyCommand(legacyCmd string) *cobra.Command {
	alias, exists := cm.aliases[legacyCmd]
	if !exists {
		return nil
	}

	cmd := &cobra.Command{
		Use:        alias.LegacyCommand,
		Short:      alias.Description,
		Deprecated: alias.DeprecationNote,
		Hidden:     true, // Hide from help but still functional
		RunE: func(cmd *cobra.Command, args []string) error {
			return cm.handleLegacyCommand(cmd.Context(), legacyCmd, args)
		},
	}

	// Add common flags that might be used
	cmd.Flags().StringP("output", "o", "text", "Output format (text, json)")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	cmd.Flags().BoolP("quiet", "q", false, "Quiet output")
	cmd.Flags().Bool("json", false, "JSON output (legacy)")

	return cmd
}

// GetAllLegacyCommands returns all available legacy command aliases
func (cm *CompatibilityManager) GetAllLegacyCommands() map[string]CommandAlias {
	return cm.aliases
}

// ShowDeprecationWarning displays a deprecation warning for legacy commands
func (cm *CompatibilityManager) ShowDeprecationWarning(ctx context.Context, legacyCmd string) {
	alias, exists := cm.aliases[legacyCmd]
	if !exists {
		return
	}

	warning := &CompatibilityWarning{
		LegacyCommand:  legacyCmd,
		NewCommand:     alias.NewCommand,
		Message:        fmt.Sprintf("Command '%s' is deprecated", legacyCmd),
		RemovalVersion: alias.RemovalVersion,
		ShowMigration:  true,
	}

	cm.displayWarning(warning)

	cm.logger.Warn(ctx, "legacy_command_used",
		observability.F("legacy_command", legacyCmd),
		observability.F("new_command", alias.NewCommand),
		observability.F("removal_version", alias.RemovalVersion),
	)
}

// Private methods

func (cm *CompatibilityManager) initializeAliases() {
	// SSH command aliases
	cm.aliases["ssh-keys"] = CommandAlias{
		LegacyCommand:   "ssh-keys",
		NewCommand:      "ssh keys list",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) List SSH keys - use 'ssh keys list'",
		DeprecationNote: "Use 'gitpersona ssh keys list' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["ssh-generate"] = CommandAlias{
		LegacyCommand: "ssh-generate",
		NewCommand:    "ssh keys generate",
		ArgumentMapping: map[string]string{
			"--type":  "--type",
			"--name":  "--name",
			"--email": "--email",
		},
		Description:     "(DEPRECATED) Generate SSH key - use 'ssh keys generate'",
		DeprecationNote: "Use 'gitpersona ssh keys generate' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["ssh-upload"] = CommandAlias{
		LegacyCommand: "ssh-upload",
		NewCommand:    "ssh keys upload",
		ArgumentMapping: map[string]string{
			"--key":   "--key",
			"--title": "--title",
		},
		Description:     "(DEPRECATED) Upload SSH key to GitHub - use 'ssh keys upload'",
		DeprecationNote: "Use 'gitpersona ssh keys upload' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["ssh-test"] = CommandAlias{
		LegacyCommand: "ssh-test",
		NewCommand:    "ssh test",
		ArgumentMapping: map[string]string{
			"--key": "--key",
		},
		Description:     "(DEPRECATED) Test SSH connection - use 'ssh test'",
		DeprecationNote: "Use 'gitpersona ssh test' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["ssh-fix"] = CommandAlias{
		LegacyCommand: "ssh-fix",
		NewCommand:    "ssh fix",
		ArgumentMapping: map[string]string{
			"--auto": "--auto",
		},
		Description:     "(DEPRECATED) Fix SSH issues - use 'ssh fix'",
		DeprecationNote: "Use 'gitpersona ssh fix' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["ssh-diagnose"] = CommandAlias{
		LegacyCommand: "ssh-diagnose",
		NewCommand:    "diagnose ssh",
		ArgumentMapping: map[string]string{
			"--verbose": "--detailed",
		},
		Description:     "(DEPRECATED) Diagnose SSH issues - use 'diagnose ssh'",
		DeprecationNote: "Use 'gitpersona diagnose ssh' instead",
		RemovalVersion:  "3.0.0",
	}

	// Account command aliases
	cm.aliases["add-account"] = CommandAlias{
		LegacyCommand: "add-account",
		NewCommand:    "account add",
		ArgumentMapping: map[string]string{
			"--name":    "--name",
			"--email":   "--email",
			"--ssh-key": "--ssh-key",
			"--github":  "--github",
		},
		Description:     "(DEPRECATED) Add account - use 'account add'",
		DeprecationNote: "Use 'gitpersona account add' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["list-accounts"] = CommandAlias{
		LegacyCommand:   "list-accounts",
		NewCommand:      "account list",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) List accounts - use 'account list'",
		DeprecationNote: "Use 'gitpersona account list' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["switch-account"] = CommandAlias{
		LegacyCommand:   "switch-account",
		NewCommand:      "account switch",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Switch account - use 'account switch'",
		DeprecationNote: "Use 'gitpersona account switch' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["remove-account"] = CommandAlias{
		LegacyCommand:   "remove-account",
		NewCommand:      "account remove",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Remove account - use 'account remove'",
		DeprecationNote: "Use 'gitpersona account remove' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["update-account"] = CommandAlias{
		LegacyCommand: "update-account",
		NewCommand:    "account update",
		ArgumentMapping: map[string]string{
			"--name":    "--name",
			"--email":   "--email",
			"--ssh-key": "--ssh-key",
			"--github":  "--github",
		},
		Description:     "(DEPRECATED) Update account - use 'account update'",
		DeprecationNote: "Use 'gitpersona account update' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["validate-account"] = CommandAlias{
		LegacyCommand:   "validate-account",
		NewCommand:      "account validate",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Validate account - use 'account validate'",
		DeprecationNote: "Use 'gitpersona account validate' instead",
		RemovalVersion:  "3.0.0",
	}

	// GitHub command aliases
	cm.aliases["github-token"] = CommandAlias{
		LegacyCommand: "github-token",
		NewCommand:    "github token set",
		ArgumentMapping: map[string]string{
			"--token": "--token",
		},
		Description:     "(DEPRECATED) Set GitHub token - use 'github token set'",
		DeprecationNote: "Use 'gitpersona github token set' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["github-test"] = CommandAlias{
		LegacyCommand:   "github-test",
		NewCommand:      "github test",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Test GitHub access - use 'github test'",
		DeprecationNote: "Use 'gitpersona github test' instead",
		RemovalVersion:  "3.0.0",
	}

	// Git command aliases
	cm.aliases["git-config"] = CommandAlias{
		LegacyCommand:   "git-config",
		NewCommand:      "git config show",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Show Git config - use 'git config show'",
		DeprecationNote: "Use 'gitpersona git config show' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["git-status"] = CommandAlias{
		LegacyCommand:   "git-status",
		NewCommand:      "git status",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Show Git status - use 'git status'",
		DeprecationNote: "Use 'gitpersona git status' instead",
		RemovalVersion:  "3.0.0",
	}

	// System command aliases
	cm.aliases["system-info"] = CommandAlias{
		LegacyCommand:   "system-info",
		NewCommand:      "diagnose system",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Show system info - use 'diagnose system'",
		DeprecationNote: "Use 'gitpersona diagnose system' instead",
		RemovalVersion:  "3.0.0",
	}

	cm.aliases["health-check"] = CommandAlias{
		LegacyCommand:   "health-check",
		NewCommand:      "diagnose health",
		ArgumentMapping: map[string]string{},
		Description:     "(DEPRECATED) Health check - use 'diagnose health'",
		DeprecationNote: "Use 'gitpersona diagnose health' instead",
		RemovalVersion:  "3.0.0",
	}
}

func (cm *CompatibilityManager) handleLegacyCommand(ctx context.Context, legacyCmd string, args []string) error {
	alias, exists := cm.aliases[legacyCmd]
	if !exists {
		return fmt.Errorf("legacy command '%s' not found", legacyCmd)
	}

	// Show deprecation warning
	cm.ShowDeprecationWarning(ctx, legacyCmd)

	// Route to appropriate new command handler based on the command type
	switch {
	case strings.HasPrefix(alias.NewCommand, "ssh"):
		return cm.handleLegacySSHCommand(ctx, alias, args)
	case strings.HasPrefix(alias.NewCommand, "account"):
		return cm.handleLegacyAccountCommand(ctx, alias, args)
	case strings.HasPrefix(alias.NewCommand, "github"):
		return cm.handleLegacyGitHubCommand(ctx, alias, args)
	case strings.HasPrefix(alias.NewCommand, "git"):
		return cm.handleLegacyGitCommand(ctx, alias, args)
	case strings.HasPrefix(alias.NewCommand, "diagnose"):
		return cm.handleLegacyDiagnoseCommand(ctx, alias, args)
	default:
		return fmt.Errorf("unsupported legacy command routing for: %s", alias.NewCommand)
	}
}

func (cm *CompatibilityManager) handleLegacySSHCommand(ctx context.Context, alias CommandAlias, args []string) error {
	switch alias.NewCommand {
	case "ssh keys list":
		keys, err := cm.services.SSH.ListKeys(ctx)
		if err != nil {
			return err
		}
		// Display keys in legacy format
		for _, key := range keys {
			fmt.Printf("Key: %s (%s)\n", key.Path, key.Type)
		}
	case "ssh keys generate":
		// Parse legacy arguments and convert to new format
		keyType := "ed25519"        // default
		email := "user@example.com" // default
		keyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519")
		if len(args) > 0 {
			keyPath = filepath.Join(os.Getenv("HOME"), ".ssh", args[0])
		}
		req := GenerateKeyRequest{
			Type:    keyType,
			Email:   email,
			KeyPath: keyPath,
		}
		_, err := cm.services.SSH.GenerateKey(ctx, req)
		return err
	case "ssh test":
		// Test connectivity for current account if available
		accounts, err := cm.services.Account.ListAccounts(ctx)
		if err != nil || len(accounts) == 0 {
			return fmt.Errorf("no accounts configured for SSH testing")
		}
		// Use first account for legacy test
		_, err = cm.services.SSH.TestConnectivity(ctx, accounts[0])
		return err
	default:
		return fmt.Errorf("unsupported SSH legacy command: %s", alias.NewCommand)
	}
	return nil
}

func (cm *CompatibilityManager) handleLegacyAccountCommand(ctx context.Context, alias CommandAlias, args []string) error {
	switch alias.NewCommand {
	case "account list":
		accounts, err := cm.services.Account.ListAccounts(ctx)
		if err != nil {
			return err
		}
		// Display accounts in legacy format
		for _, account := range accounts {
			fmt.Printf("Account: %s (%s <%s>)\n", account.Alias, account.Name, account.Email)
		}
	case "account switch":
		if len(args) == 0 {
			return fmt.Errorf("account name required")
		}
		return cm.services.Account.SwitchAccount(ctx, args[0])
	default:
		return fmt.Errorf("unsupported account legacy command: %s", alias.NewCommand)
	}
	return nil
}

func (cm *CompatibilityManager) handleLegacyGitHubCommand(ctx context.Context, alias CommandAlias, args []string) error {
	switch alias.NewCommand {
	case "github test":
		accounts, err := cm.services.Account.ListAccounts(ctx)
		if err != nil {
			return err
		}
		if len(accounts) > 0 {
			return cm.services.GitHub.TestAPIAccess(ctx, accounts[0])
		}
		return fmt.Errorf("no accounts configured")
	default:
		return fmt.Errorf("unsupported GitHub legacy command: %s", alias.NewCommand)
	}
}

func (cm *CompatibilityManager) handleLegacyGitCommand(ctx context.Context, alias CommandAlias, args []string) error {
	switch alias.NewCommand {
	case "git config show":
		config, err := cm.services.Git.GetCurrentConfig(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Git Config:\n")
		fmt.Printf("  user.name = %s\n", config.Name)
		fmt.Printf("  user.email = %s\n", config.Email)
		fmt.Printf("  scope = %s\n", config.Scope)
	case "git status":
		repo, err := cm.services.Git.DetectRepository(ctx, ".")
		if err != nil {
			return err
		}
		fmt.Printf("Repository: %s\n", repo.Path)
	default:
		return fmt.Errorf("unsupported Git legacy command: %s", alias.NewCommand)
	}
	return nil
}

func (cm *CompatibilityManager) handleLegacyDiagnoseCommand(ctx context.Context, alias CommandAlias, args []string) error {
	switch alias.NewCommand {
	case "diagnose system":
		return cm.services.System.PerformHealthCheck(ctx)
	case "diagnose health":
		return cm.services.System.PerformHealthCheck(ctx)
	default:
		return fmt.Errorf("unsupported diagnose legacy command: %s", alias.NewCommand)
	}
}

func (cm *CompatibilityManager) displayWarning(warning *CompatibilityWarning) {
	fmt.Printf("\n⚠️  DEPRECATION WARNING\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Command '%s' is deprecated and will be removed in version %s.\n",
		warning.LegacyCommand, warning.RemovalVersion)
	fmt.Printf("\nPlease use the new command instead:\n")
	fmt.Printf("  gitpersona %s\n", warning.NewCommand)

	if warning.ShowMigration {
		fmt.Printf("\nFor automatic migration assistance, run:\n")
		fmt.Printf("  gitpersona migrate check\n")
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
}
