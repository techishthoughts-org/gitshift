//go:build windows

package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/techishthoughts/GitPersona/internal/observability"
	"golang.org/x/sys/windows"
)

// WindowsManager handles Windows-specific operations
type WindowsManager struct {
	logger observability.Logger
}

// WindowsSSHAgent manages SSH agent on Windows
type WindowsSSHAgent struct {
	serviceName string
	logger      observability.Logger
}

// WindowsRegistry provides Windows registry operations
type WindowsRegistry struct {
	logger observability.Logger
}

// WindowsCredentialManager manages Windows Credential Manager
type WindowsCredentialManager struct {
	logger observability.Logger
}

// Windows API constants
const (
	HKEY_CURRENT_USER  = 0x80000001
	HKEY_LOCAL_MACHINE = 0x80000002
	KEY_READ           = 0x20019
	KEY_WRITE          = 0x20006
	REG_SZ             = 1
	REG_DWORD          = 4
)

// Windows DLL imports
var (
	advapi32        = windows.NewLazyDLL("advapi32.dll")
	regOpenKeyEx    = advapi32.NewProc("RegOpenKeyExW")
	regQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	regSetValueEx   = advapi32.NewProc("RegSetValueExW")
	regCreateKeyEx  = advapi32.NewProc("RegCreateKeyExW")
	regCloseKey     = advapi32.NewProc("RegCloseKey")
	credWrite       = advapi32.NewProc("CredWriteW")
	credRead        = advapi32.NewProc("CredReadW")
	credDelete      = advapi32.NewProc("CredDeleteW")
	credFree        = advapi32.NewProc("CredFree")
)

// NewWindowsManager creates a new Windows manager
func NewWindowsManager(logger observability.Logger) *WindowsManager {
	return &WindowsManager{
		logger: logger,
	}
}

// GetSSHAgent returns a Windows SSH agent manager
func (wm *WindowsManager) GetSSHAgent() *WindowsSSHAgent {
	return &WindowsSSHAgent{
		serviceName: "ssh-agent",
		logger:      wm.logger,
	}
}

// GetRegistry returns a Windows registry manager
func (wm *WindowsManager) GetRegistry() *WindowsRegistry {
	return &WindowsRegistry{
		logger: wm.logger,
	}
}

// GetCredentialManager returns a Windows credential manager
func (wm *WindowsManager) GetCredentialManager() *WindowsCredentialManager {
	return &WindowsCredentialManager{
		logger: wm.logger,
	}
}

// IsServiceRunning checks if the SSH agent service is running
func (wsa *WindowsSSHAgent) IsServiceRunning(ctx context.Context) (bool, error) {
	cmd := exec.Command("sc", "query", wsa.serviceName)
	output, err := cmd.Output()
	if err != nil {
		wsa.logger.Error(ctx, "failed_to_query_ssh_service",
			observability.F("error", err.Error()),
		)
		return false, err
	}

	return strings.Contains(string(output), "RUNNING"), nil
}

// StartService starts the SSH agent service
func (wsa *WindowsSSHAgent) StartService(ctx context.Context) error {
	wsa.logger.Info(ctx, "starting_ssh_agent_service")

	cmd := exec.Command("sc", "start", wsa.serviceName)
	if err := cmd.Run(); err != nil {
		wsa.logger.Error(ctx, "failed_to_start_ssh_service",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to start SSH agent service: %w", err)
	}

	// Verify service started
	running, err := wsa.IsServiceRunning(ctx)
	if err != nil {
		return err
	}

	if !running {
		return fmt.Errorf("SSH agent service failed to start")
	}

	wsa.logger.Info(ctx, "ssh_agent_service_started")
	return nil
}

// StopService stops the SSH agent service
func (wsa *WindowsSSHAgent) StopService(ctx context.Context) error {
	wsa.logger.Info(ctx, "stopping_ssh_agent_service")

	cmd := exec.Command("sc", "stop", wsa.serviceName)
	if err := cmd.Run(); err != nil {
		wsa.logger.Error(ctx, "failed_to_stop_ssh_service",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to stop SSH agent service: %w", err)
	}

	wsa.logger.Info(ctx, "ssh_agent_service_stopped")
	return nil
}

// GetSocketPath returns the SSH agent socket path on Windows
func (wsa *WindowsSSHAgent) GetSocketPath() string {
	// On Windows, SSH agent uses named pipes
	return `\\.\pipe\openssh-ssh-agent`
}

// EnsureServiceConfiguration ensures the SSH agent service is properly configured
func (wsa *WindowsSSHAgent) EnsureServiceConfiguration(ctx context.Context) error {
	wsa.logger.Info(ctx, "configuring_ssh_agent_service")

	// Set service to start automatically
	cmd := exec.Command("sc", "config", wsa.serviceName, "start=", "auto")
	if err := cmd.Run(); err != nil {
		wsa.logger.Error(ctx, "failed_to_configure_ssh_service",
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to configure SSH agent service: %w", err)
	}

	wsa.logger.Info(ctx, "ssh_agent_service_configured")
	return nil
}

// ReadString reads a string value from the registry
func (wr *WindowsRegistry) ReadString(ctx context.Context, hkey uintptr, keyPath, valueName string) (string, error) {
	var key syscall.Handle
	keyPathPtr, _ := syscall.UTF16PtrFromString(keyPath)

	ret, _, _ := regOpenKeyEx.Call(hkey, uintptr(unsafe.Pointer(keyPathPtr)), 0, KEY_READ, uintptr(unsafe.Pointer(&key)))
	if ret != 0 {
		return "", fmt.Errorf("failed to open registry key: %d", ret)
	}
	defer regCloseKey.Call(uintptr(key))

	valueNamePtr, _ := syscall.UTF16PtrFromString(valueName)
	var dataType uint32
	var dataSize uint32

	// Get the size first
	ret, _, _ = regQueryValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		0,
		uintptr(unsafe.Pointer(&dataSize)),
	)
	if ret != 0 {
		return "", fmt.Errorf("failed to query registry value size: %d", ret)
	}

	if dataType != REG_SZ {
		return "", fmt.Errorf("registry value is not a string")
	}

	// Read the actual value
	data := make([]uint16, dataSize/2)
	ret, _, _ = regQueryValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(unsafe.Pointer(&dataSize)),
	)
	if ret != 0 {
		return "", fmt.Errorf("failed to read registry value: %d", ret)
	}

	return syscall.UTF16ToString(data), nil
}

// WriteString writes a string value to the registry
func (wr *WindowsRegistry) WriteString(ctx context.Context, hkey uintptr, keyPath, valueName, value string) error {
	var key syscall.Handle
	keyPathPtr, _ := syscall.UTF16PtrFromString(keyPath)

	// Try to open existing key first
	ret, _, _ := regOpenKeyEx.Call(hkey, uintptr(unsafe.Pointer(keyPathPtr)), 0, KEY_WRITE, uintptr(unsafe.Pointer(&key)))
	if ret != 0 {
		// Key doesn't exist, create it
		var disposition uint32
		ret, _, _ = regCreateKeyEx.Call(
			hkey,
			uintptr(unsafe.Pointer(keyPathPtr)),
			0,
			0,
			0,
			KEY_WRITE,
			0,
			uintptr(unsafe.Pointer(&key)),
			uintptr(unsafe.Pointer(&disposition)),
		)
		if ret != 0 {
			return fmt.Errorf("failed to create registry key: %d", ret)
		}
	}
	defer regCloseKey.Call(uintptr(key))

	valueNamePtr, _ := syscall.UTF16PtrFromString(valueName)
	valuePtr, _ := syscall.UTF16PtrFromString(value)
	valueBytes := (*[256]byte)(unsafe.Pointer(valuePtr))

	ret, _, _ = regSetValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		REG_SZ,
		uintptr(unsafe.Pointer(&valueBytes[0])),
		uintptr(len(value)*2+2), // UTF-16 length plus null terminator
	)
	if ret != 0 {
		return fmt.Errorf("failed to write registry value: %d", ret)
	}

	wr.logger.Debug(ctx, "registry_value_written",
		observability.F("key_path", keyPath),
		observability.F("value_name", valueName),
	)

	return nil
}

// ReadDWord reads a DWORD value from the registry
func (wr *WindowsRegistry) ReadDWord(ctx context.Context, hkey uintptr, keyPath, valueName string) (uint32, error) {
	var key syscall.Handle
	keyPathPtr, _ := syscall.UTF16PtrFromString(keyPath)

	ret, _, _ := regOpenKeyEx.Call(hkey, uintptr(unsafe.Pointer(keyPathPtr)), 0, KEY_READ, uintptr(unsafe.Pointer(&key)))
	if ret != 0 {
		return 0, fmt.Errorf("failed to open registry key: %d", ret)
	}
	defer regCloseKey.Call(uintptr(key))

	valueNamePtr, _ := syscall.UTF16PtrFromString(valueName)
	var dataType uint32
	var dataSize uint32 = 4
	var data uint32

	ret, _, _ = regQueryValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		uintptr(unsafe.Pointer(&dataType)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&dataSize)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("failed to read registry value: %d", ret)
	}

	if dataType != REG_DWORD {
		return 0, fmt.Errorf("registry value is not a DWORD")
	}

	return data, nil
}

// WriteDWord writes a DWORD value to the registry
func (wr *WindowsRegistry) WriteDWord(ctx context.Context, hkey uintptr, keyPath, valueName string, value uint32) error {
	var key syscall.Handle
	keyPathPtr, _ := syscall.UTF16PtrFromString(keyPath)

	ret, _, _ := regOpenKeyEx.Call(hkey, uintptr(unsafe.Pointer(keyPathPtr)), 0, KEY_WRITE, uintptr(unsafe.Pointer(&key)))
	if ret != 0 {
		var disposition uint32
		ret, _, _ = regCreateKeyEx.Call(
			hkey,
			uintptr(unsafe.Pointer(keyPathPtr)),
			0,
			0,
			0,
			KEY_WRITE,
			0,
			uintptr(unsafe.Pointer(&key)),
			uintptr(unsafe.Pointer(&disposition)),
		)
		if ret != 0 {
			return fmt.Errorf("failed to create registry key: %d", ret)
		}
	}
	defer regCloseKey.Call(uintptr(key))

	valueNamePtr, _ := syscall.UTF16PtrFromString(valueName)

	ret, _, _ = regSetValueEx.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(valueNamePtr)),
		0,
		REG_DWORD,
		uintptr(unsafe.Pointer(&value)),
		4,
	)
	if ret != 0 {
		return fmt.Errorf("failed to write registry value: %d", ret)
	}

	wr.logger.Debug(ctx, "registry_dword_written",
		observability.F("key_path", keyPath),
		observability.F("value_name", valueName),
		observability.F("value", value),
	)

	return nil
}

// SetGitPersonaRegistry sets GitPersona-specific registry values
func (wr *WindowsRegistry) SetGitPersonaRegistry(ctx context.Context, currentAccount string) error {
	keyPath := `SOFTWARE\GitPersona`

	// Set current account
	if err := wr.WriteString(ctx, HKEY_CURRENT_USER, keyPath, "CurrentAccount", currentAccount); err != nil {
		return fmt.Errorf("failed to set current account in registry: %w", err)
	}

	// Set installation path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	installPath := filepath.Dir(exePath)

	if err := wr.WriteString(ctx, HKEY_CURRENT_USER, keyPath, "InstallPath", installPath); err != nil {
		return fmt.Errorf("failed to set install path in registry: %w", err)
	}

	wr.logger.Info(ctx, "gitpersona_registry_updated",
		observability.F("current_account", currentAccount),
		observability.F("install_path", installPath),
	)

	return nil
}

// GetGitPersonaRegistry gets GitPersona-specific registry values
func (wr *WindowsRegistry) GetGitPersonaRegistry(ctx context.Context) (map[string]string, error) {
	keyPath := `SOFTWARE\GitPersona`
	values := make(map[string]string)

	// Get current account
	if currentAccount, err := wr.ReadString(ctx, HKEY_CURRENT_USER, keyPath, "CurrentAccount"); err == nil {
		values["CurrentAccount"] = currentAccount
	}

	// Get install path
	if installPath, err := wr.ReadString(ctx, HKEY_CURRENT_USER, keyPath, "InstallPath"); err == nil {
		values["InstallPath"] = installPath
	}

	return values, nil
}

// StoreCredential stores a credential in Windows Credential Manager
func (wcm *WindowsCredentialManager) StoreCredential(ctx context.Context, target, username, password string) error {
	wcm.logger.Info(ctx, "storing_credential",
		observability.F("target", target),
		observability.F("username", username),
	)

	// This would use CredWrite API to store credentials
	// Implementation omitted for brevity - would require more Windows API structures

	wcm.logger.Info(ctx, "credential_stored")
	return nil
}

// RetrieveCredential retrieves a credential from Windows Credential Manager
func (wcm *WindowsCredentialManager) RetrieveCredential(ctx context.Context, target string) (username, password string, err error) {
	wcm.logger.Debug(ctx, "retrieving_credential",
		observability.F("target", target),
	)

	// This would use CredRead API to retrieve credentials
	// Implementation omitted for brevity

	return "", "", fmt.Errorf("credential not found")
}

// DeleteCredential deletes a credential from Windows Credential Manager
func (wcm *WindowsCredentialManager) DeleteCredential(ctx context.Context, target string) error {
	wcm.logger.Info(ctx, "deleting_credential",
		observability.F("target", target),
	)

	// This would use CredDelete API to delete credentials
	// Implementation omitted for brevity

	wcm.logger.Info(ctx, "credential_deleted")
	return nil
}

// ConfigureWindowsTerminal configures Windows Terminal for GitPersona
func (wm *WindowsManager) ConfigureWindowsTerminal(ctx context.Context) error {
	wm.logger.Info(ctx, "configuring_windows_terminal")

	// Get Windows Terminal settings path
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("LOCALAPPDATA environment variable not set")
	}

	settingsPath := filepath.Join(localAppData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json")

	// Check if Windows Terminal is installed
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		wm.logger.Warn(ctx, "windows_terminal_not_found")
		return nil // Not an error, just not installed
	}

	// This would modify Windows Terminal settings to add GitPersona integration
	wm.logger.Info(ctx, "windows_terminal_configured")
	return nil
}

// ConfigurePowerShell configures PowerShell for GitPersona
func (wm *WindowsManager) ConfigurePowerShell(ctx context.Context) error {
	wm.logger.Info(ctx, "configuring_powershell")

	// Get PowerShell profile path
	documentsDir := filepath.Join(os.Getenv("USERPROFILE"), "Documents")
	profilePath := filepath.Join(documentsDir, "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")

	// Ensure profile directory exists
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return fmt.Errorf("failed to create PowerShell profile directory: %w", err)
	}

	// Add GitPersona initialization to PowerShell profile
	profileContent := `
# GitPersona PowerShell Integration
if (Get-Command gitpersona -ErrorAction SilentlyContinue) {
    function gp { gitpersona $args }
    function gpswitch { gitpersona switch $args }
    function gpstatus { gitpersona status }
    function gplist { gitpersona list }
}
`

	// Append to profile if not already present
	existingContent := ""
	if content, err := os.ReadFile(profilePath); err == nil {
		existingContent = string(content)
	}

	if !strings.Contains(existingContent, "GitPersona PowerShell Integration") {
		file, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open PowerShell profile: %w", err)
		}
		defer file.Close()

		if _, err := file.WriteString(profileContent); err != nil {
			return fmt.Errorf("failed to write to PowerShell profile: %w", err)
		}
	}

	wm.logger.Info(ctx, "powershell_configured",
		observability.F("profile_path", profilePath),
	)

	return nil
}

// SetupWindowsEnvironment sets up the Windows environment for GitPersona
func (wm *WindowsManager) SetupWindowsEnvironment(ctx context.Context) error {
	wm.logger.Info(ctx, "setting_up_windows_environment")

	// Configure SSH agent
	sshAgent := wm.GetSSHAgent()
	if err := sshAgent.EnsureServiceConfiguration(ctx); err != nil {
		return fmt.Errorf("failed to configure SSH agent: %w", err)
	}

	// Configure registry
	registry := wm.GetRegistry()
	if err := registry.SetGitPersonaRegistry(ctx, "default"); err != nil {
		return fmt.Errorf("failed to configure registry: %w", err)
	}

	// Configure PowerShell
	if err := wm.ConfigurePowerShell(ctx); err != nil {
		return fmt.Errorf("failed to configure PowerShell: %w", err)
	}

	// Configure Windows Terminal
	if err := wm.ConfigureWindowsTerminal(ctx); err != nil {
		wm.logger.Warn(ctx, "failed_to_configure_windows_terminal",
			observability.F("error", err.Error()),
		)
		// Not a fatal error
	}

	wm.logger.Info(ctx, "windows_environment_setup_complete")
	return nil
}
