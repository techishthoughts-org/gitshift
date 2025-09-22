package plugins

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// ExtensionManager manages built-in extensions and third-party plugins
type ExtensionManager struct {
	logger     observability.Logger
	extensions map[string]*Extension
	hooks      map[string][]ExtensionHook
	config     *ExtensionConfig
	mutex      sync.RWMutex
}

// Extension represents a GitPersona extension
type Extension struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Type        ExtensionType          `json:"type"`
	Category    string                 `json:"category"`
	Author      string                 `json:"author"`
	Homepage    string                 `json:"homepage"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	Commands    []ExtensionCommand     `json:"commands"`
	Hooks       []string               `json:"hooks"`
	LoadedAt    time.Time              `json:"loaded_at"`
	Handler     ExtensionHandler       `json:"-"`
}

// ExtensionType defines the type of extension
type ExtensionType string

const (
	ExtensionTypeBuiltIn    ExtensionType = "builtin"
	ExtensionTypeThirdParty ExtensionType = "thirdparty"
	ExtensionTypeScript     ExtensionType = "script"
	ExtensionTypeWebhook    ExtensionType = "webhook"
)

// ExtensionCommand represents a command provided by an extension
type ExtensionCommand struct {
	Name        string                                         `json:"name"`
	Description string                                         `json:"description"`
	Usage       string                                         `json:"usage"`
	Examples    []string                                       `json:"examples"`
	Category    string                                         `json:"category"`
	Aliases     []string                                       `json:"aliases"`
	Hidden      bool                                           `json:"hidden"`
	Handler     func(ctx context.Context, args []string) error `json:"-"`
}

// ExtensionHook represents an extension hook
type ExtensionHook struct {
	Event    string
	Priority int
	Handler  func(ctx context.Context, data interface{}) error
}

// ExtensionHandler defines the interface for extension handlers
type ExtensionHandler interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	Execute(ctx context.Context, command string, args []string) error
	GetCommands() []ExtensionCommand
	GetHooks() []ExtensionHook
	Cleanup(ctx context.Context) error
}

// ExtensionConfig holds configuration for extensions
type ExtensionConfig struct {
	AutoLoad      bool              `json:"auto_load"`
	EnabledTypes  []ExtensionType   `json:"enabled_types"`
	ScriptTimeout time.Duration     `json:"script_timeout"`
	Sandboxing    bool              `json:"sandboxing"`
	WhitelistDirs []string          `json:"whitelist_dirs"`
	Permissions   map[string]string `json:"permissions"`
}

// ScriptExtension handles script-based extensions
type ScriptExtension struct {
	scriptPath  string
	interpreter string
	timeout     time.Duration
	logger      observability.Logger
	environment map[string]string
}

// Initialize implements ExtensionHandler interface
func (se *ScriptExtension) Initialize(ctx context.Context, config map[string]interface{}) error {
	return nil
}

// Execute implements ExtensionHandler interface
func (se *ScriptExtension) Execute(ctx context.Context, command string, args []string) error {
	return fmt.Errorf("script execution not implemented")
}

// GetCommands implements ExtensionHandler interface
func (se *ScriptExtension) GetCommands() []ExtensionCommand {
	return []ExtensionCommand{}
}

// GetHooks implements ExtensionHandler interface
func (se *ScriptExtension) GetHooks() []ExtensionHook {
	return []ExtensionHook{}
}

// Cleanup implements ExtensionHandler interface
func (se *ScriptExtension) Cleanup(ctx context.Context) error {
	return nil
}

// WebhookExtension handles webhook-based extensions
type WebhookExtension struct {
	endpoint string
	secret   string
	timeout  time.Duration
	headers  map[string]string
	logger   observability.Logger
}

// NewExtensionManager creates a new extension manager
func NewExtensionManager(logger observability.Logger) *ExtensionManager {
	return &ExtensionManager{
		logger:     logger,
		extensions: make(map[string]*Extension),
		hooks:      make(map[string][]ExtensionHook),
		config: &ExtensionConfig{
			AutoLoad:      true,
			EnabledTypes:  []ExtensionType{ExtensionTypeBuiltIn, ExtensionTypeScript},
			ScriptTimeout: 30 * time.Second,
			Sandboxing:    true,
			WhitelistDirs: []string{},
			Permissions:   make(map[string]string),
		},
	}
}

// LoadBuiltInExtensions loads all built-in extensions
func (em *ExtensionManager) LoadBuiltInExtensions(ctx context.Context) error {
	em.logger.Info(ctx, "loading_builtin_extensions")

	builtinExtensions := []*Extension{
		em.createSSHManagerExtension(),
		em.createGitHubSyncExtension(),
		em.createBackupExtension(),
		em.createHealthCheckExtension(),
		em.createPerformanceExtension(),
		em.createSecurityExtension(),
	}

	loadedCount := 0
	for _, ext := range builtinExtensions {
		if err := em.registerExtension(ctx, ext); err != nil {
			em.logger.Error(ctx, "failed_to_load_builtin_extension",
				observability.F("extension", ext.Name),
				observability.F("error", err.Error()),
			)
			continue
		}
		loadedCount++
	}

	em.logger.Info(ctx, "builtin_extensions_loaded",
		observability.F("count", loadedCount),
	)

	return nil
}

// createSSHManagerExtension creates the SSH manager extension
func (em *ExtensionManager) createSSHManagerExtension() *Extension {
	return &Extension{
		ID:          "ssh-manager",
		Name:        "SSH Manager",
		Version:     "1.0.0",
		Description: "Advanced SSH key and agent management",
		Type:        ExtensionTypeBuiltIn,
		Category:    "security",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "ssh-diagnose",
				Description: "Diagnose SSH configuration and connectivity issues",
				Usage:       "gitpersona ssh-diagnose [account]",
				Examples:    []string{"gitpersona ssh-diagnose work", "gitpersona ssh-diagnose --all"},
				Category:    "diagnostics",
			},
			{
				Name:        "ssh-repair",
				Description: "Automatically repair SSH configuration issues",
				Usage:       "gitpersona ssh-repair [account]",
				Examples:    []string{"gitpersona ssh-repair personal"},
				Category:    "maintenance",
			},
			{
				Name:        "ssh-backup",
				Description: "Backup SSH keys and configuration",
				Usage:       "gitpersona ssh-backup <destination>",
				Examples:    []string{"gitpersona ssh-backup ~/.ssh-backup"},
				Category:    "backup",
			},
		},
		Hooks:   []string{"pre_account_switch", "post_account_switch", "ssh_error"},
		Handler: &SSHManagerExtension{logger: em.logger},
	}
}

// createGitHubSyncExtension creates the GitHub sync extension
func (em *ExtensionManager) createGitHubSyncExtension() *Extension {
	return &Extension{
		ID:          "github-sync",
		Name:        "GitHub Sync",
		Version:     "1.0.0",
		Description: "Real-time GitHub integration and synchronization",
		Type:        ExtensionTypeBuiltIn,
		Category:    "integration",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "github-status",
				Description: "Check GitHub API connectivity and token status",
				Usage:       "gitpersona github-status [account]",
				Examples:    []string{"gitpersona github-status", "gitpersona github-status work"},
				Category:    "status",
			},
			{
				Name:        "github-sync",
				Description: "Synchronize local configuration with GitHub",
				Usage:       "gitpersona github-sync [--force]",
				Examples:    []string{"gitpersona github-sync", "gitpersona github-sync --force"},
				Category:    "sync",
			},
			{
				Name:        "github-webhook",
				Description: "Manage GitHub webhook integrations",
				Usage:       "gitpersona github-webhook <action> [options]",
				Examples: []string{
					"gitpersona github-webhook create --repo owner/repo",
					"gitpersona github-webhook list",
				},
				Category: "webhook",
			},
		},
		Hooks:   []string{"token_refresh", "account_switch", "git_operation"},
		Handler: &GitHubSyncExtension{logger: em.logger},
	}
}

// createBackupExtension creates the backup extension
func (em *ExtensionManager) createBackupExtension() *Extension {
	return &Extension{
		ID:          "backup",
		Name:        "Backup Manager",
		Version:     "1.0.0",
		Description: "Automated backup and recovery system",
		Type:        ExtensionTypeBuiltIn,
		Category:    "backup",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "backup-create",
				Description: "Create a complete backup of GitPersona configuration",
				Usage:       "gitpersona backup-create [destination]",
				Examples:    []string{"gitpersona backup-create", "gitpersona backup-create ~/backups"},
				Category:    "backup",
			},
			{
				Name:        "backup-restore",
				Description: "Restore GitPersona configuration from backup",
				Usage:       "gitpersona backup-restore <backup-path>",
				Examples:    []string{"gitpersona backup-restore ~/backups/gitpersona-2024-01-01.tar.gz"},
				Category:    "restore",
			},
			{
				Name:        "backup-schedule",
				Description: "Schedule automatic backups",
				Usage:       "gitpersona backup-schedule <frequency>",
				Examples:    []string{"gitpersona backup-schedule daily", "gitpersona backup-schedule weekly"},
				Category:    "automation",
			},
		},
		Hooks:   []string{"pre_config_change", "post_config_change"},
		Handler: &BackupExtension{logger: em.logger},
	}
}

// createHealthCheckExtension creates the health check extension
func (em *ExtensionManager) createHealthCheckExtension() *Extension {
	return &Extension{
		ID:          "health-check",
		Name:        "Health Monitor",
		Version:     "1.0.0",
		Description: "System health monitoring and diagnostics",
		Type:        ExtensionTypeBuiltIn,
		Category:    "monitoring",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "health",
				Description: "Display overall system health",
				Usage:       "gitpersona health [--detailed]",
				Examples:    []string{"gitpersona health", "gitpersona health --detailed"},
				Category:    "monitoring",
			},
			{
				Name:        "doctor",
				Description: "Run comprehensive system diagnostics",
				Usage:       "gitpersona doctor [--fix]",
				Examples:    []string{"gitpersona doctor", "gitpersona doctor --fix"},
				Category:    "diagnostics",
			},
		},
		Hooks:   []string{"system_start", "system_error", "periodic_check"},
		Handler: &HealthCheckExtension{logger: em.logger},
	}
}

// createPerformanceExtension creates the performance extension
func (em *ExtensionManager) createPerformanceExtension() *Extension {
	return &Extension{
		ID:          "performance",
		Name:        "Performance Monitor",
		Version:     "1.0.0",
		Description: "Performance monitoring and optimization",
		Type:        ExtensionTypeBuiltIn,
		Category:    "performance",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "perf",
				Description: "Display performance metrics",
				Usage:       "gitpersona perf [--live]",
				Examples:    []string{"gitpersona perf", "gitpersona perf --live"},
				Category:    "monitoring",
			},
			{
				Name:        "optimize",
				Description: "Run performance optimizations",
				Usage:       "gitpersona optimize [--cache] [--config]",
				Examples:    []string{"gitpersona optimize", "gitpersona optimize --cache"},
				Category:    "optimization",
			},
		},
		Hooks:   []string{"operation_start", "operation_end"},
		Handler: &PerformanceExtension{logger: em.logger},
	}
}

// createSecurityExtension creates the security extension
func (em *ExtensionManager) createSecurityExtension() *Extension {
	return &Extension{
		ID:          "security",
		Name:        "Security Scanner",
		Version:     "1.0.0",
		Description: "Security scanning and hardening",
		Type:        ExtensionTypeBuiltIn,
		Category:    "security",
		Author:      "GitPersona Team",
		Enabled:     true,
		Commands: []ExtensionCommand{
			{
				Name:        "security-scan",
				Description: "Scan for security vulnerabilities",
				Usage:       "gitpersona security-scan [--type=all|ssh|tokens|config]",
				Examples: []string{
					"gitpersona security-scan",
					"gitpersona security-scan --type=ssh",
				},
				Category: "security",
			},
			{
				Name:        "security-harden",
				Description: "Apply security hardening measures",
				Usage:       "gitpersona security-harden [--level=basic|advanced]",
				Examples:    []string{"gitpersona security-harden", "gitpersona security-harden --level=advanced"},
				Category:    "security",
			},
		},
		Hooks:   []string{"security_event", "config_change"},
		Handler: &SecurityExtension{logger: em.logger},
	}
}

// registerExtension registers an extension
func (em *ExtensionManager) registerExtension(ctx context.Context, ext *Extension) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Initialize extension
	if err := ext.Handler.Initialize(ctx, ext.Config); err != nil {
		return fmt.Errorf("failed to initialize extension %s: %w", ext.Name, err)
	}

	// Register hooks
	hooks := ext.Handler.GetHooks()
	for _, hook := range hooks {
		if em.hooks[hook.Event] == nil {
			em.hooks[hook.Event] = make([]ExtensionHook, 0)
		}
		em.hooks[hook.Event] = append(em.hooks[hook.Event], hook)
	}

	// Register extension
	em.extensions[ext.ID] = ext
	ext.LoadedAt = time.Now()

	em.logger.Info(ctx, "extension_registered",
		observability.F("id", ext.ID),
		observability.F("name", ext.Name),
		observability.F("version", ext.Version),
		observability.F("commands", len(ext.Commands)),
	)

	return nil
}

// ExecuteExtensionCommand executes an extension command
func (em *ExtensionManager) ExecuteExtensionCommand(ctx context.Context, commandName string, args []string) error {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	// Find extension that provides this command
	for _, ext := range em.extensions {
		if !ext.Enabled {
			continue
		}

		for _, cmd := range ext.Commands {
			if cmd.Name == commandName || em.isAlias(cmd.Aliases, commandName) {
				// Execute with timeout
				execCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
				defer cancel()

				return ext.Handler.Execute(execCtx, commandName, args)
			}
		}
	}

	return fmt.Errorf("command not found: %s", commandName)
}

// ExecuteHook executes all hooks for a given event
func (em *ExtensionManager) ExecuteHook(ctx context.Context, event string, data interface{}) error {
	em.mutex.RLock()
	hooks, exists := em.hooks[event]
	em.mutex.RUnlock()

	if !exists {
		return nil
	}

	em.logger.Debug(ctx, "executing_hooks",
		observability.F("event", event),
		observability.F("hook_count", len(hooks)),
	)

	// Execute hooks in priority order
	for _, hook := range hooks {
		if err := hook.Handler(ctx, data); err != nil {
			em.logger.Error(ctx, "hook_execution_failed",
				observability.F("event", event),
				observability.F("error", err.Error()),
			)
			return err
		}
	}

	return nil
}

// GetAllCommands returns all available extension commands
func (em *ExtensionManager) GetAllCommands() map[string][]ExtensionCommand {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	commands := make(map[string][]ExtensionCommand)
	for id, ext := range em.extensions {
		if ext.Enabled {
			commands[id] = ext.Commands
		}
	}

	return commands
}

// LoadScriptExtension loads a script-based extension
func (em *ExtensionManager) LoadScriptExtension(ctx context.Context, scriptPath string) error {
	if !em.isTypeEnabled(ExtensionTypeScript) {
		return fmt.Errorf("script extensions are disabled")
	}

	// Validate script path
	if !em.isPathWhitelisted(scriptPath) {
		return fmt.Errorf("script path not whitelisted: %s", scriptPath)
	}

	// Create script extension
	scriptExt := &ScriptExtension{
		scriptPath:  scriptPath,
		interpreter: em.detectInterpreter(scriptPath),
		timeout:     em.config.ScriptTimeout,
		logger:      em.logger,
		environment: make(map[string]string),
	}

	// Read script metadata
	metadata, err := em.parseScriptMetadata(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to parse script metadata: %w", err)
	}

	ext := &Extension{
		ID:          metadata.ID,
		Name:        metadata.Name,
		Version:     metadata.Version,
		Description: metadata.Description,
		Type:        ExtensionTypeScript,
		Category:    metadata.Category,
		Author:      metadata.Author,
		Enabled:     true,
		Handler:     scriptExt,
	}

	return em.registerExtension(ctx, ext)
}

// isAlias checks if a command name is an alias
func (em *ExtensionManager) isAlias(aliases []string, name string) bool {
	for _, alias := range aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// isTypeEnabled checks if an extension type is enabled
func (em *ExtensionManager) isTypeEnabled(extType ExtensionType) bool {
	for _, enabled := range em.config.EnabledTypes {
		if enabled == extType {
			return true
		}
	}
	return false
}

// isPathWhitelisted checks if a path is whitelisted
func (em *ExtensionManager) isPathWhitelisted(path string) bool {
	if len(em.config.WhitelistDirs) == 0 {
		return true // No whitelist restrictions
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, whitelistDir := range em.config.WhitelistDirs {
		absWhitelistDir, err := filepath.Abs(whitelistDir)
		if err != nil {
			continue
		}

		if filepath.HasPrefix(absPath, absWhitelistDir) {
			return true
		}
	}

	return false
}

// detectInterpreter detects the interpreter for a script
func (em *ExtensionManager) detectInterpreter(scriptPath string) string {
	ext := filepath.Ext(scriptPath)
	switch ext {
	case ".py":
		return "python3"
	case ".sh":
		return "bash"
	case ".js":
		return "node"
	case ".rb":
		return "ruby"
	default:
		return "bash"
	}
}

// parseScriptMetadata parses metadata from a script
func (em *ExtensionManager) parseScriptMetadata(scriptPath string) (*ScriptMetadata, error) {
	// This would parse metadata from script comments
	// For now, return default metadata
	return &ScriptMetadata{
		ID:          filepath.Base(scriptPath),
		Name:        filepath.Base(scriptPath),
		Version:     "1.0.0",
		Description: "Script extension",
		Category:    "script",
		Author:      "Unknown",
	}, nil
}

// ScriptMetadata represents metadata for script extensions
type ScriptMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Author      string `json:"author"`
}

// Stop stops the extension manager
func (em *ExtensionManager) Stop(ctx context.Context) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.logger.Info(ctx, "stopping_extension_manager")

	for id, ext := range em.extensions {
		if err := ext.Handler.Cleanup(ctx); err != nil {
			em.logger.Error(ctx, "extension_cleanup_failed",
				observability.F("id", id),
				observability.F("error", err.Error()),
			)
		}
	}

	em.extensions = make(map[string]*Extension)
	em.hooks = make(map[string][]ExtensionHook)

	em.logger.Info(ctx, "extension_manager_stopped")
	return nil
}
