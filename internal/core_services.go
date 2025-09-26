package internal

import (
	"context"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// CoreServices represents the simplified service layer with 5 main services
type CoreServices struct {
	Account AccountManager
	SSH     SSHManager
	Git     GitManager
	GitHub  GitHubManager
	System  SystemManager
}

// ServiceContainer provides dependency injection for core services
type ServiceContainer struct {
	services *CoreServices
	logger   observability.Logger
}

// NewServiceContainer creates a new service container with all core services
func NewServiceContainer(logger observability.Logger) *ServiceContainer {
	container := &ServiceContainer{
		logger: logger,
	}

	// Create services with proper dependency injection
	accountManager := NewAccountManager(logger).(*RealAccountManager)
	sshManager := NewSSHManager(logger)
	gitManager := NewGitManager(logger)
	githubManager := NewGitHubManager(logger)
	systemManager := NewSystemManager(logger).(*RealSystemManager)

	// Wire dependencies
	accountManager.SetDependencies(sshManager, gitManager, githubManager)
	systemManager.SetDependencies(accountManager, sshManager, gitManager, githubManager)

	container.services = &CoreServices{
		Account: accountManager,
		SSH:     sshManager,
		Git:     gitManager,
		GitHub:  githubManager,
		System:  systemManager,
	}

	return container
}

// GetServices returns the core services
func (c *ServiceContainer) GetServices() *CoreServices {
	return c.services
}

// HealthCheck performs health check on all services
func (c *ServiceContainer) HealthCheck(ctx context.Context) error {
	return c.services.System.PerformHealthCheck(ctx)
}

// AccountManager interface - consolidated account operations
type AccountManager interface {
	// Core account operations
	CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error)
	GetAccount(ctx context.Context, alias string) (*Account, error)
	ListAccounts(ctx context.Context) ([]*Account, error)
	UpdateAccount(ctx context.Context, alias string, updates AccountUpdates) error
	DeleteAccount(ctx context.Context, alias string) error

	// Account switching
	SwitchAccount(ctx context.Context, alias string) error
	GetCurrentAccount(ctx context.Context) (*Account, error)

	// Account validation
	ValidateAccount(ctx context.Context, alias string) (*ValidationResult, error)
	ValidateAllAccounts(ctx context.Context) ([]*ValidationResult, error)

	// Discovery and auto-setup
	DiscoverAccounts(ctx context.Context) ([]*DiscoveredAccount, error)
	SetupAccount(ctx context.Context, discovered *DiscoveredAccount) (*Account, error)
}

// SSHManager interface - unified SSH operations
type SSHManager interface {
	// SSH key management
	GenerateKey(ctx context.Context, req GenerateKeyRequest) (*SSHKey, error)
	ListKeys(ctx context.Context) ([]*SSHKeyInfo, error)
	ValidateKey(ctx context.Context, keyPath string) (*SSHKeyInfo, error)
	DeleteKey(ctx context.Context, keyPath string) error

	// SSH configuration
	GenerateConfig(ctx context.Context, accounts []*Account) (string, error)
	InstallConfig(ctx context.Context, configContent string) error
	ValidateConfig(ctx context.Context) (*SSHConfigValidation, error)

	// SSH agent management
	StartAgent(ctx context.Context, account *Account) error
	StopAgent(ctx context.Context, account *Account) error
	LoadKey(ctx context.Context, keyPath string) error

	// SSH testing and diagnostics
	TestConnectivity(ctx context.Context, account *Account) (*ConnectivityResult, error)
	DiagnoseIssues(ctx context.Context) ([]*SSHIssue, error)
	FixIssues(ctx context.Context, issues []*SSHIssue) error
}

// GitManager interface - Git configuration management
type GitManager interface {
	// Git configuration
	SetGlobalConfig(ctx context.Context, account *Account) error
	SetLocalConfig(ctx context.Context, account *Account, repoPath string) error
	GetCurrentConfig(ctx context.Context) (*GitConfig, error)

	// Repository operations
	DetectRepository(ctx context.Context, path string) (*RepositoryInfo, error)
	SuggestAccount(ctx context.Context, repoInfo *RepositoryInfo) (*Account, error)

	// Git validation
	ValidateConfig(ctx context.Context) (*GitValidationResult, error)
	FixConfig(ctx context.Context, issues []*GitIssue) error
}

// GitHubManager interface - GitHub API and token management
type GitHubManager interface {
	// Token management
	SetToken(ctx context.Context, account *Account, token string) error
	GetToken(ctx context.Context, account *Account) (string, error)
	ValidateToken(ctx context.Context, account *Account) (*TokenValidation, error)
	RefreshToken(ctx context.Context, account *Account) error

	// GitHub API operations
	GetUserInfo(ctx context.Context, account *Account) (*GitHubUser, error)
	ListRepositories(ctx context.Context, account *Account) ([]*GitHubRepository, error)
	UploadSSHKey(ctx context.Context, account *Account, keyContent, title string) error

	// GitHub validation
	TestAPIAccess(ctx context.Context, account *Account) error
	ValidateSSHAccess(ctx context.Context, account *Account) error
}

// SystemManager interface - system health and diagnostics
type SystemManager interface {
	// Health checks
	PerformHealthCheck(ctx context.Context) error
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)

	// Diagnostics
	RunDiagnostics(ctx context.Context) (*DiagnosticsReport, error)
	GetTroubleshootingInfo(ctx context.Context) (*TroubleshootingInfo, error)

	// Auto-fix capabilities
	AutoFix(ctx context.Context, issues []*SystemIssue) error

	// Migration and updates
	MigrateConfiguration(ctx context.Context, fromVersion, toVersion string) error
	BackupConfiguration(ctx context.Context) (string, error)
}

// Request/Response types for the service interfaces
type CreateAccountRequest struct {
	Alias          string
	Name           string
	Email          string
	GitHubUsername string
	SSHKeyPath     string
	Description    string
}

type AccountUpdates struct {
	Name           *string
	Email          *string
	GitHubUsername *string
	SSHKeyPath     *string
	Description    *string
}

type GenerateKeyRequest struct {
	Type      string // "ed25519", "rsa"
	Email     string
	KeyPath   string
	Overwrite bool
}

// Domain types
type Account struct {
	Alias          string
	Name           string
	Email          string
	GitHubUsername string
	SSHKeyPath     string
	Description    string
	IsActive       bool
	CreatedAt      string
	LastUsed       *string
}

type DiscoveredAccount struct {
	Source       string
	Confidence   int
	Username     string
	Email        string
	SSHKeyPath   string
	Repositories []string
}

type ValidationResult struct {
	Account string
	Valid   bool
	Issues  []string
	Fixed   []string
}

type SSHKey struct {
	Path        string
	Type        string
	Fingerprint string
	PublicKey   string
	CreatedAt   string
}

type SSHKeyInfo struct {
	Path        string
	Type        string
	Size        int
	Fingerprint string
	Email       string
	Valid       bool
	Exists      bool
	Readable    bool
}

type SSHConfigValidation struct {
	Valid  bool
	Issues []string
}

type ConnectivityResult struct {
	Account string
	Success bool
	Message string
	Latency int64
	Details map[string]interface{}
}

type SSHIssue struct {
	Type        string
	Severity    string
	Description string
	Fix         string
	AutoFixable bool
	Fixed       bool
}

type GitConfig struct {
	Name  string
	Email string
	Scope string // "global" or "local"
}

type RepositoryInfo struct {
	Path         string
	RemoteURL    string
	Organization string
	IsGitHub     bool
	SSHRemote    bool
}

type GitValidationResult struct {
	Valid  bool
	Issues []*GitIssue
}

type GitIssue struct {
	Type        string
	Description string
	Fix         string
}

type TokenValidation struct {
	Valid     bool
	Username  string
	Scopes    []string
	ExpiresAt *string
	Message   string
}

type GitHubUser struct {
	Username string
	Name     string
	Email    string
	ID       int64
}

type GitHubRepository struct {
	Name     string
	FullName string
	Private  bool
	SSHURL   string
	HTTPSURL string
}

type SystemInfo struct {
	Version    string
	Platform   string
	SSHVersion string
	GitVersion string
}

type DiagnosticsReport struct {
	Overall  string // "healthy", "issues", "critical"
	Checks   []*DiagnosticCheck
	Summary  string
	FixCount int
}

type DiagnosticCheck struct {
	Name    string
	Status  string // "pass", "warn", "fail"
	Message string
	Fix     string
}

type TroubleshootingInfo struct {
	CommonIssues []string
	Solutions    map[string]string
	LogFiles     []string
}

type SystemIssue struct {
	Type        string
	Severity    string
	Description string
	AutoFixable bool
}
