package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// PlatformManager handles cross-platform compatibility
type PlatformManager struct {
	logger   observability.Logger
	platform Platform
	features *PlatformFeatures
}

// Platform represents the current platform
type Platform struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	Version      string `json:"version"`
	Distribution string `json:"distribution,omitempty"`
	Shell        string `json:"shell"`
	Terminal     string `json:"terminal,omitempty"`
	HomeDir      string `json:"home_dir"`
	ConfigDir    string `json:"config_dir"`
	CacheDir     string `json:"cache_dir"`
	TempDir      string `json:"temp_dir"`
}

// PlatformFeatures defines platform-specific feature availability
type PlatformFeatures struct {
	SSHAgent           bool `json:"ssh_agent"`
	FilePermissions    bool `json:"file_permissions"`
	SymbolicLinks      bool `json:"symbolic_links"`
	ProcessManagement  bool `json:"process_management"`
	EnvironmentVars    bool `json:"environment_vars"`
	ShellIntegration   bool `json:"shell_integration"`
	KeychainAccess     bool `json:"keychain_access"`
	NotificationSystem bool `json:"notification_system"`
	SystemTray         bool `json:"system_tray"`
	WindowsRegistry    bool `json:"windows_registry"`
	MacOSKeychain      bool `json:"macos_keychain"`
	LinuxKeyring       bool `json:"linux_keyring"`
}

// PathManager handles platform-specific path operations
type PathManager struct {
	platform Platform
	logger   observability.Logger
}

// ShellManager handles shell-specific operations
type ShellManager struct {
	platform Platform
	logger   observability.Logger
}

// ProcessManager handles platform-specific process operations
type ProcessManager struct {
	platform Platform
	logger   observability.Logger
}

// NewPlatformManager creates a new platform manager
func NewPlatformManager(logger observability.Logger) *PlatformManager {
	platform := detectPlatform()
	features := detectFeatures(platform)

	return &PlatformManager{
		logger:   logger,
		platform: platform,
		features: features,
	}
}

// detectPlatform detects the current platform
func detectPlatform() Platform {
	homeDir, _ := os.UserHomeDir()

	platform := Platform{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		HomeDir: homeDir,
		Shell:   detectShell(),
	}

	// Platform-specific detection
	switch runtime.GOOS {
	case "windows":
		platform.ConfigDir = filepath.Join(homeDir, "AppData", "Roaming", "GitPersona")
		platform.CacheDir = filepath.Join(homeDir, "AppData", "Local", "GitPersona", "Cache")
		platform.TempDir = os.TempDir()
		platform.Version = detectWindowsVersion()
	case "darwin":
		platform.ConfigDir = filepath.Join(homeDir, "Library", "Application Support", "GitPersona")
		platform.CacheDir = filepath.Join(homeDir, "Library", "Caches", "GitPersona")
		platform.TempDir = "/tmp"
		platform.Version = detectMacOSVersion()
		platform.Terminal = detectMacOSTerminal()
	case "linux":
		platform.ConfigDir = getLinuxConfigDir(homeDir)
		platform.CacheDir = getLinuxCacheDir(homeDir)
		platform.TempDir = "/tmp"
		platform.Version = detectLinuxVersion()
		platform.Distribution = detectLinuxDistribution()
	default:
		// Fallback for other Unix-like systems
		platform.ConfigDir = filepath.Join(homeDir, ".config", "gitpersona")
		platform.CacheDir = filepath.Join(homeDir, ".cache", "gitpersona")
		platform.TempDir = "/tmp"
	}

	return platform
}

// detectFeatures detects available platform features
func detectFeatures(platform Platform) *PlatformFeatures {
	features := &PlatformFeatures{
		EnvironmentVars:   true, // Available on all platforms
		ProcessManagement: true, // Available on all platforms
	}

	switch platform.OS {
	case "windows":
		features.SSHAgent = checkWindowsSSHAgent()
		features.FilePermissions = false // Windows has different permission model
		features.SymbolicLinks = checkWindowsSymlinks()
		features.ShellIntegration = true
		features.KeychainAccess = false
		features.NotificationSystem = true
		features.SystemTray = true
		features.WindowsRegistry = true
		features.MacOSKeychain = false
		features.LinuxKeyring = false
	case "darwin":
		features.SSHAgent = true
		features.FilePermissions = true
		features.SymbolicLinks = true
		features.ShellIntegration = true
		features.KeychainAccess = true
		features.NotificationSystem = true
		features.SystemTray = true
		features.WindowsRegistry = false
		features.MacOSKeychain = true
		features.LinuxKeyring = false
	case "linux":
		features.SSHAgent = true
		features.FilePermissions = true
		features.SymbolicLinks = true
		features.ShellIntegration = true
		features.KeychainAccess = checkLinuxKeyring()
		features.NotificationSystem = checkLinuxNotifications()
		features.SystemTray = checkLinuxSystemTray()
		features.WindowsRegistry = false
		features.MacOSKeychain = false
		features.LinuxKeyring = checkLinuxKeyring()
	default:
		// Conservative defaults for unknown platforms
		features.SSHAgent = true
		features.FilePermissions = true
		features.SymbolicLinks = true
		features.ShellIntegration = false
		features.KeychainAccess = false
		features.NotificationSystem = false
		features.SystemTray = false
	}

	return features
}

// GetPlatform returns the current platform information
func (pm *PlatformManager) GetPlatform() Platform {
	return pm.platform
}

// GetFeatures returns the platform features
func (pm *PlatformManager) GetFeatures() *PlatformFeatures {
	return pm.features
}

// IsSupported checks if a feature is supported on the current platform
func (pm *PlatformManager) IsSupported(feature string) bool {
	switch feature {
	case "ssh_agent":
		return pm.features.SSHAgent
	case "file_permissions":
		return pm.features.FilePermissions
	case "symbolic_links":
		return pm.features.SymbolicLinks
	case "process_management":
		return pm.features.ProcessManagement
	case "environment_vars":
		return pm.features.EnvironmentVars
	case "shell_integration":
		return pm.features.ShellIntegration
	case "keychain_access":
		return pm.features.KeychainAccess
	case "notification_system":
		return pm.features.NotificationSystem
	case "system_tray":
		return pm.features.SystemTray
	case "windows_registry":
		return pm.features.WindowsRegistry
	case "macos_keychain":
		return pm.features.MacOSKeychain
	case "linux_keyring":
		return pm.features.LinuxKeyring
	default:
		return false
	}
}

// GetConfigDir returns the platform-appropriate configuration directory
func (pm *PlatformManager) GetConfigDir() string {
	return pm.platform.ConfigDir
}

// GetCacheDir returns the platform-appropriate cache directory
func (pm *PlatformManager) GetCacheDir() string {
	return pm.platform.CacheDir
}

// GetTempDir returns the platform-appropriate temporary directory
func (pm *PlatformManager) GetTempDir() string {
	return pm.platform.TempDir
}

// EnsureDirectories creates necessary directories with appropriate permissions
func (pm *PlatformManager) EnsureDirectories(ctx context.Context) error {
	dirs := []string{
		pm.platform.ConfigDir,
		pm.platform.CacheDir,
		filepath.Join(pm.platform.ConfigDir, "accounts"),
		filepath.Join(pm.platform.ConfigDir, "ssh"),
		filepath.Join(pm.platform.ConfigDir, "backups"),
		filepath.Join(pm.platform.CacheDir, "tokens"),
		filepath.Join(pm.platform.CacheDir, "ssh"),
	}

	for _, dir := range dirs {
		if err := pm.createDirectoryWithPermissions(dir); err != nil {
			pm.logger.Error(ctx, "failed_to_create_directory",
				observability.F("directory", dir),
				observability.F("error", err.Error()),
			)
			return err
		}
	}

	pm.logger.Info(ctx, "platform_directories_ensured",
		observability.F("config_dir", pm.platform.ConfigDir),
		observability.F("cache_dir", pm.platform.CacheDir),
	)

	return nil
}

// createDirectoryWithPermissions creates a directory with appropriate permissions
func (pm *PlatformManager) createDirectoryWithPermissions(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		var perm os.FileMode
		switch pm.platform.OS {
		case "windows":
			perm = 0755 // Windows will handle permissions differently
		default:
			perm = 0700 // Restrictive permissions for Unix-like systems
		}

		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Set additional permissions on Unix-like systems
		if pm.features.FilePermissions {
			if err := os.Chmod(dir, 0700); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", dir, err)
			}
		}
	}

	return nil
}

// GetPathManager returns a path manager for the current platform
func (pm *PlatformManager) GetPathManager() *PathManager {
	return &PathManager{
		platform: pm.platform,
		logger:   pm.logger,
	}
}

// GetShellManager returns a shell manager for the current platform
func (pm *PlatformManager) GetShellManager() *ShellManager {
	return &ShellManager{
		platform: pm.platform,
		logger:   pm.logger,
	}
}

// GetProcessManager returns a process manager for the current platform
func (pm *PlatformManager) GetProcessManager() *ProcessManager {
	return &ProcessManager{
		platform: pm.platform,
		logger:   pm.logger,
	}
}

// NormalizePath normalizes a path for the current platform
func (pm *PathManager) NormalizePath(path string) string {
	// Convert path separators
	normalized := filepath.FromSlash(path)

	// Handle Windows drive letters
	if pm.platform.OS == "windows" {
		// Convert /c/... to C:\...
		if len(normalized) >= 3 && normalized[0] == '/' && normalized[2] == '/' {
			if drive := normalized[1]; (drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z') {
				normalized = strings.ToUpper(string(drive)) + ":" + normalized[2:]
			}
		}
	}

	return normalized
}

// ExpandPath expands ~ and environment variables in path
func (pm *PathManager) ExpandPath(path string) string {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", pm.platform.HomeDir, 1)
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return pm.NormalizePath(path)
}

// JoinPath joins path elements in a platform-appropriate way
func (pm *PathManager) JoinPath(elements ...string) string {
	return filepath.Join(elements...)
}

// IsAbsolute checks if a path is absolute
func (pm *PathManager) IsAbsolute(path string) bool {
	return filepath.IsAbs(path)
}

// GetSSHConfigPath returns the SSH config file path
func (pm *PathManager) GetSSHConfigPath() string {
	sshDir := filepath.Join(pm.platform.HomeDir, ".ssh")
	return filepath.Join(sshDir, "config")
}

// GetSSHDir returns the SSH directory path
func (pm *PathManager) GetSSHDir() string {
	return filepath.Join(pm.platform.HomeDir, ".ssh")
}

// GetShellConfigPath returns the shell configuration file path
func (sm *ShellManager) GetShellConfigPath() string {
	switch sm.platform.Shell {
	case "bash":
		if sm.platform.OS == "darwin" {
			// macOS uses .bash_profile by default
			return filepath.Join(sm.platform.HomeDir, ".bash_profile")
		}
		return filepath.Join(sm.platform.HomeDir, ".bashrc")
	case "zsh":
		return filepath.Join(sm.platform.HomeDir, ".zshrc")
	case "fish":
		configDir := filepath.Join(sm.platform.HomeDir, ".config", "fish")
		return filepath.Join(configDir, "config.fish")
	case "powershell":
		if sm.platform.OS == "windows" {
			documentsDir := filepath.Join(sm.platform.HomeDir, "Documents")
			return filepath.Join(documentsDir, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return filepath.Join(sm.platform.HomeDir, ".config", "powershell", "profile.ps1")
	default:
		return filepath.Join(sm.platform.HomeDir, ".profile")
	}
}

// GetShellEnvironmentCommand returns the command to set an environment variable
func (sm *ShellManager) GetShellEnvironmentCommand(key, value string) string {
	switch sm.platform.Shell {
	case "bash", "zsh":
		return fmt.Sprintf("export %s=\"%s\"", key, value)
	case "fish":
		return fmt.Sprintf("set -x %s \"%s\"", key, value)
	case "powershell":
		return fmt.Sprintf("$env:%s = \"%s\"", key, value)
	case "cmd":
		return fmt.Sprintf("set %s=%s", key, value)
	default:
		return fmt.Sprintf("export %s=\"%s\"", key, value)
	}
}

// GetShellSourceCommand returns the command to source a file
func (sm *ShellManager) GetShellSourceCommand(file string) string {
	switch sm.platform.Shell {
	case "bash", "zsh":
		return fmt.Sprintf("source \"%s\"", file)
	case "fish":
		return fmt.Sprintf("source \"%s\"", file)
	case "powershell":
		return fmt.Sprintf(". \"%s\"", file)
	default:
		return fmt.Sprintf("source \"%s\"", file)
	}
}

// FindProcess finds a process by name
func (pm *ProcessManager) FindProcess(name string) ([]int, error) {
	switch pm.platform.OS {
	case "windows":
		return pm.findWindowsProcess(name)
	default:
		return pm.findUnixProcess(name)
	}
}

// KillProcess kills a process by PID
func (pm *ProcessManager) KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	switch pm.platform.OS {
	case "windows":
		return process.Kill()
	default:
		return process.Signal(syscall.SIGTERM)
	}
}

// Helper functions for platform detection

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = os.Getenv("ComSpec") // Windows
	}

	if shell != "" {
		return filepath.Base(shell)
	}

	// Default shells by platform
	switch runtime.GOOS {
	case "windows":
		return "cmd"
	case "darwin":
		return "zsh" // Default on macOS 10.15+
	default:
		return "bash"
	}
}

func detectWindowsVersion() string {
	// This would use Windows API to get version
	return "10.0" // Placeholder
}

func detectMacOSVersion() string {
	// This would use system calls to get macOS version
	return "12.0" // Placeholder
}

func detectMacOSTerminal() string {
	terminal := os.Getenv("TERM_PROGRAM")
	if terminal == "" {
		return "Terminal"
	}
	return terminal
}

func detectLinuxVersion() string {
	// This would read /etc/os-release or similar
	return "5.4" // Placeholder
}

func detectLinuxDistribution() string {
	// This would read /etc/os-release to get distribution
	return "ubuntu" // Placeholder
}

func getLinuxConfigDir(homeDir string) string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		return filepath.Join(xdgConfig, "gitpersona")
	}
	return filepath.Join(homeDir, ".config", "gitpersona")
}

func getLinuxCacheDir(homeDir string) string {
	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache != "" {
		return filepath.Join(xdgCache, "gitpersona")
	}
	return filepath.Join(homeDir, ".cache", "gitpersona")
}

func checkWindowsSSHAgent() bool {
	// Check if OpenSSH Authentication Agent service is available
	return true // Placeholder
}

func checkWindowsSymlinks() bool {
	// Check if running with administrator privileges
	return false // Placeholder
}

func checkLinuxKeyring() bool {
	// Check if libsecret or gnome-keyring is available
	return true // Placeholder
}

func checkLinuxNotifications() bool {
	// Check if libnotify is available
	return true // Placeholder
}

func checkLinuxSystemTray() bool {
	// Check if running in a desktop environment with system tray
	return true // Placeholder
}

func (pm *ProcessManager) findWindowsProcess(name string) ([]int, error) {
	// Implementation would use Windows API to find processes
	return []int{}, nil // Placeholder
}

func (pm *ProcessManager) findUnixProcess(name string) ([]int, error) {
	// Implementation would use ps command or proc filesystem
	return []int{}, nil // Placeholder
}
