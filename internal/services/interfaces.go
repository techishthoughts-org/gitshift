package services

import (
	"context"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
)

// AccountService handles account management operations
type AccountService interface {
	// Account CRUD operations
	GetAccount(ctx context.Context, alias string) (*models.Account, error)
	CreateAccount(ctx context.Context, account *models.Account) error
	UpdateAccount(ctx context.Context, account *models.Account) error
	DeleteAccount(ctx context.Context, alias string) error
	ListAccounts(ctx context.Context) ([]*models.Account, error)

	// Account state management
	SetCurrentAccount(ctx context.Context, alias string) error
	GetCurrentAccount(ctx context.Context) string
	MarkAccountAsUsed(ctx context.Context, alias string) error

	// Account validation
	ValidateAccount(ctx context.Context, account *models.Account) error
	TestAccountSSH(ctx context.Context, account *models.Account) error
}

// SSHService handles SSH key and configuration management
type SSHService interface {
	// SSH key management
	GenerateKey(ctx context.Context, keyType string, email string, keyPath string) (*SSHKey, error)
	ValidateKey(ctx context.Context, keyPath string) (*SSHKeyInfo, error)
	ListKeys(ctx context.Context) ([]*SSHKeyInfo, error)
	DeleteKey(ctx context.Context, keyPath string) error

	// SSH configuration management
	ValidateConfiguration(ctx context.Context) (*SSHValidationResult, error)
	FixPermissions(ctx context.Context, keyPath string) error
	GenerateSSHConfig(ctx context.Context) (string, error)
	TestGitHubAuthentication(ctx context.Context, keyPath string) error

	// SSH troubleshooting
	DiagnoseIssues(ctx context.Context) ([]*SSHIssue, error)
	FixIssues(ctx context.Context, issues []*SSHIssue) error
}

// GitHubService handles GitHub API operations
type GitHubService interface {
	// Authentication
	Authenticate(ctx context.Context, token string) error
	GetAuthenticatedUser(ctx context.Context) (*GitHubUser, error)
	TestSSHKey(ctx context.Context, keyPath string) error

	// Repository operations
	ListRepositories(ctx context.Context) ([]*GitHubRepository, error)
	GetRepository(ctx context.Context, owner, name string) (*GitHubRepository, error)
	CreateRepository(ctx context.Context, name string, private bool) (*GitHubRepository, error)
	DeleteRepository(ctx context.Context, owner, name string) error

	// SSH key management
	ListSSHKeys(ctx context.Context) ([]*GitHubSSHKey, error)
	AddSSHKey(ctx context.Context, title, key string) (*GitHubSSHKey, error)
	DeleteSSHKey(ctx context.Context, keyID int) error

	// Organization operations
	ListOrganizations(ctx context.Context) ([]*GitHubOrganization, error)
	GetOrganization(ctx context.Context, name string) (*GitHubOrganization, error)
}

// ConfigurationService handles application configuration
type ConfigurationService interface {
	// Configuration management
	Load(ctx context.Context) error
	Save(ctx context.Context) error
	Reload(ctx context.Context) error
	Validate(ctx context.Context) error

	// Account configuration
	GetAccount(ctx context.Context, alias string) (*models.Account, error)
	SetAccount(ctx context.Context, account *models.Account) error
	DeleteAccount(ctx context.Context, alias string) error
	ListAccounts(ctx context.Context) ([]*models.Account, error)

	// Current account management
	SetCurrentAccount(ctx context.Context, alias string) error
	GetCurrentAccount(ctx context.Context) string

	// Configuration validation
	ValidateConfiguration(ctx context.Context) error
	CheckForConflicts(ctx context.Context) ([]*ConfigConflict, error)
}

// HealthService handles system health monitoring
type HealthService interface {
	// Health checks
	CheckSystemHealth(ctx context.Context) (*HealthStatus, error)
	CheckServiceHealth(ctx context.Context, serviceName string) (*ServiceHealth, error)
	CheckDependencies(ctx context.Context) ([]*DependencyHealth, error)

	// Monitoring
	GetMetrics(ctx context.Context) (*SystemMetrics, error)
	GetPerformanceStats(ctx context.Context) (*PerformanceStats, error)
}

// ValidationService handles various validation operations
type ValidationService interface {
	// Git validation
	ValidateGitConfiguration(ctx context.Context) (*GitValidationResult, error)
	FixGitConfiguration(ctx context.Context, issues []*GitIssue) error

	// SSH validation
	ValidateSSHConfiguration(ctx context.Context) (*SSHValidationResult, error)
	FixSSHConfiguration(ctx context.Context, issues []*SSHIssue) error

	// Account validation
	ValidateAccountConfiguration(ctx context.Context, account *models.Account) (*AccountValidationResult, error)
	FixAccountConfiguration(ctx context.Context, account *models.Account, issues []*AccountIssue) error

	// System validation
	ValidateSystemConfiguration(ctx context.Context) (*SystemValidationResult, error)
}

// Data structures for service responses

// SSHKey represents an SSH key
type SSHKey struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	CreatedAt   string `json:"created_at"`
}

// SSHKeyInfo represents SSH key information
type SSHKeyInfo struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	Fingerprint string `json:"fingerprint"`
	Email       string `json:"email"`
	Exists      bool   `json:"exists"`
	Readable    bool   `json:"readable"`
	Valid       bool   `json:"valid"`
}

// SSHValidationResult represents SSH validation results
type SSHValidationResult struct {
	Valid           bool          `json:"valid"`
	Issues          []*SSHIssue   `json:"issues"`
	Recommendations []string      `json:"recommendations"`
	Keys            []*SSHKeyInfo `json:"keys"`
}

// SSHIssue represents an SSH configuration issue
type SSHIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Fixed       bool   `json:"fixed"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	FullName    string      `json:"full_name"`
	Description string      `json:"description"`
	Private     bool        `json:"private"`
	URL         string      `json:"html_url"`
	CloneURL    string      `json:"clone_url"`
	SSHURL      string      `json:"ssh_url"`
	Owner       *GitHubUser `json:"owner"`
}

// GitHubSSHKey represents a GitHub SSH key
type GitHubSSHKey struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Key   string `json:"key"`
	URL   string `json:"url"`
}

// GitHubOrganization represents a GitHub organization
type GitHubOrganization struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AvatarURL   string `json:"avatar_url"`
	Type        string `json:"type"`
}

// ConfigConflict represents a configuration conflict
type ConfigConflict struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Resolution  string `json:"resolution"`
}

// HealthStatus represents overall system health
type HealthStatus struct {
	Overall      string                    `json:"overall"`
	Services     map[string]*ServiceHealth `json:"services"`
	Dependencies []*DependencyHealth       `json:"dependencies"`
	Timestamp    string                    `json:"timestamp"`
}

// ServiceHealth represents the health of a specific service
type ServiceHealth struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// DependencyHealth represents the health of a dependency
type DependencyHealth struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Version   string `json:"version"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// SystemMetrics represents system metrics
type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	NetworkIO   int64   `json:"network_io"`
	Timestamp   string  `json:"timestamp"`
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	ResponseTime float64 `json:"response_time"`
	Throughput   int64   `json:"throughput"`
	ErrorRate    float64 `json:"error_rate"`
	Uptime       int64   `json:"uptime"`
	Timestamp    string  `json:"timestamp"`
}

// GitValidationResult represents Git validation results
type GitValidationResult struct {
	Valid           bool        `json:"valid"`
	Issues          []*GitIssue `json:"issues"`
	Recommendations []string    `json:"recommendations"`
	Configuration   *GitConfig  `json:"configuration"`
}

// GitIssue represents a Git configuration issue
type GitIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Fixed       bool   `json:"fixed"`
}

// AccountValidationResult represents account validation results
type AccountValidationResult struct {
	Valid           bool            `json:"valid"`
	Issues          []*AccountIssue `json:"issues"`
	Recommendations []string        `json:"recommendations"`
	Account         *models.Account `json:"account"`
}

// AccountIssue represents an account configuration issue
type AccountIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Fixed       bool   `json:"fixed"`
}

// SystemValidationResult represents system validation results
type SystemValidationResult struct {
	Valid           bool                      `json:"valid"`
	Issues          []*SystemIssue            `json:"issues"`
	Recommendations []string                  `json:"recommendations"`
	Services        map[string]*ServiceHealth `json:"services"`
}

// SystemIssue represents a system configuration issue
type SystemIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Fixed       bool   `json:"fixed"`
}

// GitService handles Git repository operations
type GitService interface {
	// Repository validation
	ValidateRepositoryExists(ctx context.Context, repoURL string) error

	// Remote URL management
	GetRemoteURL(ctx context.Context, remote string) (string, error)
	SetRemoteURL(ctx context.Context, remote, url string) error
}

// SSHAgentService manages SSH agent operations
type SSHAgentService interface {
	// Agent management
	IsAgentRunning(ctx context.Context) (bool, error)
	StartAgent(ctx context.Context) error
	StopAgent(ctx context.Context) error

	// Key management
	LoadKey(ctx context.Context, keyPath string) error
	UnloadKey(ctx context.Context, keyPath string) error
	ClearAllKeys(ctx context.Context) error
	ListLoadedKeys(ctx context.Context) ([]string, error)

	// Account-specific operations
	SwitchToAccount(ctx context.Context, keyPath string) error
	SwitchToAccountWithCleanup(ctx context.Context, keyPath string) error
	IsolateAccount(ctx context.Context, keyPath string) error

	// Socket cleanup operations
	CleanupSSHSockets(ctx context.Context) error

	// SSH validation operations
	ValidateSSHConnectionWithRetry(ctx context.Context, keyPath string) error

	// Status and diagnostics
	GetAgentStatus(ctx context.Context) (*SSHAgentStatus, error)
	DiagnoseAgentIssues(ctx context.Context) ([]string, error)
}

// SSHAgentStatus represents the current state of the SSH agent
type SSHAgentStatus struct {
	Running     bool      `json:"running"`
	PID         int       `json:"pid,omitempty"`
	SocketPath  string    `json:"socket_path,omitempty"`
	LoadedKeys  []string  `json:"loaded_keys"`
	KeyCount    int       `json:"key_count"`
	LastUpdated time.Time `json:"last_updated"`
}
