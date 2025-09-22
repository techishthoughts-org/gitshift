package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// PluginManager manages CLI plugins and extensions
type PluginManager struct {
	logger      observability.Logger
	pluginsDir  string
	plugins     map[string]*LoadedPlugin
	hooks       map[string][]PluginHook
	mutex       sync.RWMutex
	enabled     bool
	sandboxMode bool
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	Commands     []PluginCommand        `json:"commands"`
	Hooks        []string               `json:"hooks"`
	Config       map[string]interface{} `json:"config"`
	LoadedAt     time.Time              `json:"loaded_at"`
	Plugin       *plugin.Plugin         `json:"-"`
	Instance     PluginInterface        `json:"-"`
	Enabled      bool                   `json:"enabled"`
	Dependencies []string               `json:"dependencies"`
}

// PluginCommand represents a command provided by a plugin
type PluginCommand struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Usage       string                    `json:"usage"`
	Flags       []PluginFlag              `json:"flags"`
	Aliases     []string                  `json:"aliases"`
	Category    string                    `json:"category"`
	Handler     func(args []string) error `json:"-"`
}

// PluginFlag represents a command flag
type PluginFlag struct {
	Name        string `json:"name"`
	Short       string `json:"short"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
}

// PluginHook represents a hook function
type PluginHook struct {
	Name     string
	Priority int
	Handler  func(ctx context.Context, data interface{}) error
}

// PluginInterface defines the interface that plugins must implement
type PluginInterface interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
	GetInfo() PluginInfo
	GetCommands() []PluginCommand
	GetHooks() []PluginHook
	Execute(ctx context.Context, command string, args []string) error
	Cleanup(ctx context.Context) error
}

// PluginInfo contains basic plugin information
type PluginInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	APIVersion   string   `json:"api_version"`
	Dependencies []string `json:"dependencies"`
}

// PluginManifest represents the plugin manifest file
type PluginManifest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	APIVersion   string                 `json:"api_version"`
	EntryPoint   string                 `json:"entry_point"`
	Dependencies []string               `json:"dependencies"`
	Permissions  []string               `json:"permissions"`
	Config       map[string]interface{} `json:"config"`
	Platforms    []string               `json:"platforms"`
}

// PluginRegistry manages plugin discovery and installation
type PluginRegistry struct {
	registryURL string
	localCache  string
	plugins     map[string]*RegistryPlugin
	mutex       sync.RWMutex
}

// RegistryPlugin represents a plugin in the registry
type RegistryPlugin struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Homepage    string            `json:"homepage"`
	Repository  string            `json:"repository"`
	License     string            `json:"license"`
	Tags        []string          `json:"tags"`
	Downloads   int64             `json:"downloads"`
	Rating      float64           `json:"rating"`
	Platforms   []string          `json:"platforms"`
	Checksums   map[string]string `json:"checksums"`
	Published   time.Time         `json:"published"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(logger observability.Logger, pluginsDir string) *PluginManager {
	if pluginsDir == "" {
		homeDir, _ := os.UserHomeDir()
		pluginsDir = filepath.Join(homeDir, ".gitpersona", "plugins")
	}

	// Ensure plugins directory exists
	os.MkdirAll(pluginsDir, 0755)

	return &PluginManager{
		logger:      logger,
		pluginsDir:  pluginsDir,
		plugins:     make(map[string]*LoadedPlugin),
		hooks:       make(map[string][]PluginHook),
		enabled:     true,
		sandboxMode: true, // Enable sandbox by default for security
	}
}

// LoadPlugins loads all plugins from the plugins directory
func (pm *PluginManager) LoadPlugins(ctx context.Context) error {
	if !pm.enabled {
		return nil
	}

	pm.logger.Info(ctx, "loading_plugins",
		observability.F("plugins_dir", pm.pluginsDir),
	)

	// Only load plugins on supported platforms
	if runtime.GOOS == "windows" || runtime.GOARCH != "amd64" {
		pm.logger.Warn(ctx, "plugin_loading_not_supported",
			observability.F("os", runtime.GOOS),
			observability.F("arch", runtime.GOARCH),
		)
		return nil
	}

	files, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	loadedCount := 0
	for _, file := range files {
		if file.IsDir() {
			pluginPath := filepath.Join(pm.pluginsDir, file.Name())
			if err := pm.loadPlugin(ctx, pluginPath); err != nil {
				pm.logger.Error(ctx, "failed_to_load_plugin",
					observability.F("plugin", file.Name()),
					observability.F("error", err.Error()),
				)
				continue
			}
			loadedCount++
		}
	}

	pm.logger.Info(ctx, "plugins_loaded",
		observability.F("count", loadedCount),
	)

	return nil
}

// loadPlugin loads a single plugin
func (pm *PluginManager) loadPlugin(ctx context.Context, pluginPath string) error {
	// Read plugin manifest
	manifestPath := filepath.Join(pluginPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Check platform compatibility
	if !pm.isPlatformSupported(manifest.Platforms) {
		return fmt.Errorf("plugin not supported on platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Check API version compatibility
	if !pm.isAPIVersionSupported(manifest.APIVersion) {
		return fmt.Errorf("unsupported API version: %s", manifest.APIVersion)
	}

	// Load the plugin binary
	binaryPath := filepath.Join(pluginPath, manifest.EntryPoint)
	if !pm.fileExists(binaryPath) {
		return fmt.Errorf("plugin binary not found: %s", binaryPath)
	}

	// Load plugin using Go's plugin system
	p, err := plugin.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Get the plugin symbol
	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin symbol not found: %w", err)
	}

	// Cast to plugin interface
	pluginInstance, ok := symbol.(PluginInterface)
	if !ok {
		return fmt.Errorf("invalid plugin interface")
	}

	// Initialize plugin
	if err := pluginInstance.Initialize(ctx, manifest.Config); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Get plugin info
	_ = pluginInstance.GetInfo()

	// Create loaded plugin
	loadedPlugin := &LoadedPlugin{
		Name:         manifest.Name,
		Version:      manifest.Version,
		Description:  manifest.Description,
		Author:       manifest.Author,
		Commands:     pluginInstance.GetCommands(),
		Config:       manifest.Config,
		LoadedAt:     time.Now(),
		Plugin:       p,
		Instance:     pluginInstance,
		Enabled:      true,
		Dependencies: manifest.Dependencies,
	}

	// Register hooks
	hooks := pluginInstance.GetHooks()
	for _, hook := range hooks {
		pm.registerHook(hook.Name, hook)
	}

	// Store plugin
	pm.mutex.Lock()
	pm.plugins[manifest.Name] = loadedPlugin
	pm.mutex.Unlock()

	pm.logger.Info(ctx, "plugin_loaded",
		observability.F("name", manifest.Name),
		observability.F("version", manifest.Version),
		observability.F("commands", len(loadedPlugin.Commands)),
		observability.F("hooks", len(hooks)),
	)

	return nil
}

// ExecutePluginCommand executes a plugin command
func (pm *PluginManager) ExecutePluginCommand(ctx context.Context, pluginName, command string, args []string) error {
	pm.mutex.RLock()
	plugin, exists := pm.plugins[pluginName]
	pm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	if !plugin.Enabled {
		return fmt.Errorf("plugin disabled: %s", pluginName)
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return plugin.Instance.Execute(execCtx, command, args)
}

// GetPluginCommands returns all available plugin commands
func (pm *PluginManager) GetPluginCommands() map[string][]PluginCommand {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	commands := make(map[string][]PluginCommand)
	for name, plugin := range pm.plugins {
		if plugin.Enabled {
			commands[name] = plugin.Commands
		}
	}

	return commands
}

// ExecuteHook executes all hooks for a given event
func (pm *PluginManager) ExecuteHook(ctx context.Context, hookName string, data interface{}) error {
	pm.mutex.RLock()
	hooks, exists := pm.hooks[hookName]
	pm.mutex.RUnlock()

	if !exists {
		return nil
	}

	// Execute hooks in priority order
	for _, hook := range hooks {
		if err := hook.Handler(ctx, data); err != nil {
			pm.logger.Error(ctx, "hook_execution_failed",
				observability.F("hook", hookName),
				observability.F("error", err.Error()),
			)
			return err
		}
	}

	return nil
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() map[string]*LoadedPlugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugins := make(map[string]*LoadedPlugin)
	for name, plugin := range pm.plugins {
		// Create a copy without the actual plugin instance
		pluginCopy := *plugin
		pluginCopy.Plugin = nil
		pluginCopy.Instance = nil
		plugins[name] = &pluginCopy
	}

	return plugins
}

// EnablePlugin enables a plugin
func (pm *PluginManager) EnablePlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.Enabled = true

	pm.logger.Info(ctx, "plugin_enabled",
		observability.F("name", name),
	)

	return nil
}

// DisablePlugin disables a plugin
func (pm *PluginManager) DisablePlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.Enabled = false

	pm.logger.Info(ctx, "plugin_disabled",
		observability.F("name", name),
	)

	return nil
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Cleanup plugin
	if err := plugin.Instance.Cleanup(ctx); err != nil {
		pm.logger.Error(ctx, "plugin_cleanup_failed",
			observability.F("name", name),
			observability.F("error", err.Error()),
		)
	}

	// Remove hooks
	for hookName, hooks := range pm.hooks {
		newHooks := make([]PluginHook, 0)
		for _, hook := range hooks {
			if hook.Name != name {
				newHooks = append(newHooks, hook)
			}
		}
		pm.hooks[hookName] = newHooks
	}

	// Remove plugin
	delete(pm.plugins, name)

	pm.logger.Info(ctx, "plugin_unloaded",
		observability.F("name", name),
	)

	return nil
}

// InstallPlugin installs a plugin from the registry
func (pm *PluginManager) InstallPlugin(ctx context.Context, name, version string) error {
	// Implementation would download and install plugin from registry
	pm.logger.Info(ctx, "installing_plugin",
		observability.F("name", name),
		observability.F("version", version),
	)

	// This is a placeholder - real implementation would:
	// 1. Download plugin from registry
	// 2. Verify checksums
	// 3. Extract to plugins directory
	// 4. Load the plugin

	return fmt.Errorf("plugin installation not implemented")
}

// registerHook registers a plugin hook
func (pm *PluginManager) registerHook(hookName string, hook PluginHook) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.hooks[hookName] == nil {
		pm.hooks[hookName] = make([]PluginHook, 0)
	}

	pm.hooks[hookName] = append(pm.hooks[hookName], hook)
}

// isPlatformSupported checks if the current platform is supported
func (pm *PluginManager) isPlatformSupported(platforms []string) bool {
	if len(platforms) == 0 {
		return true // No platform restrictions
	}

	currentPlatform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	for _, platform := range platforms {
		if platform == currentPlatform || platform == runtime.GOOS || platform == "all" {
			return true
		}
	}

	return false
}

// isAPIVersionSupported checks if the API version is supported
func (pm *PluginManager) isAPIVersionSupported(version string) bool {
	// Simple version check - in real implementation, use semantic versioning
	supportedVersions := []string{"1.0", "1.1", "1.2"}
	for _, supported := range supportedVersions {
		if version == supported {
			return true
		}
	}
	return false
}

// fileExists checks if a file exists
func (pm *PluginManager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Stop stops the plugin manager and unloads all plugins
func (pm *PluginManager) Stop(ctx context.Context) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.logger.Info(ctx, "stopping_plugin_manager")

	for name, plugin := range pm.plugins {
		if err := plugin.Instance.Cleanup(ctx); err != nil {
			pm.logger.Error(ctx, "plugin_cleanup_failed",
				observability.F("name", name),
				observability.F("error", err.Error()),
			)
		}
	}

	pm.plugins = make(map[string]*LoadedPlugin)
	pm.hooks = make(map[string][]PluginHook)
	pm.enabled = false

	pm.logger.Info(ctx, "plugin_manager_stopped")
	return nil
}

// GetPluginInfo returns information about a specific plugin
func (pm *PluginManager) GetPluginInfo(name string) (*LoadedPlugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	// Return a copy
	pluginCopy := *plugin
	pluginCopy.Plugin = nil
	pluginCopy.Instance = nil

	return &pluginCopy, nil
}

// ValidatePlugin validates a plugin before loading
func (pm *PluginManager) ValidatePlugin(pluginPath string) error {
	// Read and validate manifest
	manifestPath := filepath.Join(pluginPath, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("manifest not found: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Validate required fields
	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if manifest.EntryPoint == "" {
		return fmt.Errorf("plugin entry point is required")
	}

	// Check if binary exists
	binaryPath := filepath.Join(pluginPath, manifest.EntryPoint)
	if !pm.fileExists(binaryPath) {
		return fmt.Errorf("plugin binary not found: %s", binaryPath)
	}

	return nil
}
