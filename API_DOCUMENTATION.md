# GitPersona API Documentation

## Overview

GitPersona provides a comprehensive service-oriented architecture with well-defined interfaces for all major functionality. This documentation covers the service APIs available for developers extending or integrating with GitPersona.

## Service Architecture

```
┌─────────────────────┐
│   Command Layer     │  ← CLI Commands
├─────────────────────┤
│  Service Container  │  ← Dependency Injection
├─────────────────────┤
│   Service Layer     │  ← Business Logic APIs
├─────────────────────┤
│   Infrastructure    │  ← SSH, Git, GitHub APIs
└─────────────────────┘
```

---

## 🏗️ Service Container

### SimpleContainer

The main dependency injection container providing type-safe access to all services.

**Location:** `internal/container/container_simple.go`

#### Methods

```go
// Service Getters
GetConfigService() services.ConfigurationService
GetAccountService() services.AccountService
GetSSHService() services.SSHService
GetSSHAgentService() services.SSHAgentService
GetGitService() services.GitConfigManager
GetGitHubService() services.GitHubService
GetHealthService() services.HealthService
GetValidationService() services.ValidationService

// Lifecycle Management
Initialize(ctx context.Context) error
Shutdown(ctx context.Context) error
GetLogger() observability.Logger
```

#### Usage Example

```go
import "github.com/techishthoughts/GitPersona/internal/container"

// Get global container
container := container.GetGlobalSimpleContainer()

// Initialize services
ctx := context.Background()
if err := container.Initialize(ctx); err != nil {
    log.Fatal("Failed to initialize container:", err)
}

// Access services
configService := container.GetConfigService()
sshService := container.GetSSHService()
```

---

## 📁 Configuration Service

### ConfigurationService Interface

Manages application configuration including accounts, settings, and persistence.

**Interface Location:** `internal/services/interfaces.go:72-92`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Load(ctx)` | Load configuration from disk | `context.Context` | `error` |
| `Save(ctx)` | Save configuration to disk | `context.Context` | `error` |
| `Reload(ctx)` | Reload configuration | `context.Context` | `error` |
| `Validate(ctx)` | Validate configuration | `context.Context` | `error` |
| `GetAccount(ctx, alias)` | Get account by alias | `context.Context, string` | `*models.Account, error` |
| `SetAccount(ctx, account)` | Set/update account | `context.Context, *models.Account` | `error` |
| `DeleteAccount(ctx, alias)` | Delete account | `context.Context, string` | `error` |
| `ListAccounts(ctx)` | List all accounts | `context.Context` | `[]*models.Account, error` |
| `SetCurrentAccount(ctx, alias)` | Set active account | `context.Context, string` | `error` |
| `GetCurrentAccount(ctx)` | Get active account alias | `context.Context` | `string` |
| `ValidateConfiguration(ctx)` | Validate entire config | `context.Context` | `error` |
| `CheckForConflicts(ctx)` | Check for conflicts | `context.Context` | `[]*ConfigConflict, error` |

#### Usage Example

```go
configService := container.GetConfigService()

// Load configuration
if err := configService.Load(ctx); err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Get account
account, err := configService.GetAccount(ctx, "work")
if err != nil {
    return fmt.Errorf("account not found: %w", err)
}

// Set current account
if err := configService.SetCurrentAccount(ctx, "work"); err != nil {
    return fmt.Errorf("failed to set current account: %w", err)
}
```

### Configuration Model

```go
type Config struct {
    Accounts        map[string]*Account    `json:"accounts"`
    PendingAccounts map[string]*PendingAccount `json:"pending_accounts,omitempty"`
    CurrentAccount  string                 `json:"current_account,omitempty"`
    GlobalGitConfig bool                   `json:"global_git_config"`
    AutoDetect      bool                   `json:"auto_detect"`
    ConfigVersion   string                 `json:"config_version"`
}
```

---

## 👤 Account Service

### AccountService Interface

Manages GitHub account operations and state.

**Interface Location:** `internal/services/interfaces.go:11-27`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `GetAccount(ctx, alias)` | Retrieve account | `context.Context, string` | `*models.Account, error` |
| `CreateAccount(ctx, account)` | Create new account | `context.Context, *models.Account` | `error` |
| `UpdateAccount(ctx, account)` | Update existing account | `context.Context, *models.Account` | `error` |
| `DeleteAccount(ctx, alias)` | Delete account | `context.Context, string` | `error` |
| `ListAccounts(ctx)` | List all accounts | `context.Context` | `[]*models.Account, error` |
| `SetCurrentAccount(ctx, alias)` | Set active account | `context.Context, string` | `error` |
| `GetCurrentAccount(ctx)` | Get current account alias | `context.Context` | `string` |
| `MarkAccountAsUsed(ctx, alias)` | Update usage timestamp | `context.Context, string` | `error` |
| `ValidateAccount(ctx, account)` | Validate account config | `context.Context, *models.Account` | `error` |
| `TestAccountSSH(ctx, account)` | Test SSH connectivity | `context.Context, *models.Account` | `error` |

#### Account Model

```go
type Account struct {
    Alias           string      `json:"alias"`
    Name            string      `json:"name"`
    Email           string      `json:"email"`
    SSHKeyPath      string      `json:"ssh_key_path"`
    GitHubUsername  string      `json:"github_username"`
    Description     string      `json:"description,omitempty"`
    IsDefault       bool        `json:"is_default"`
    CreatedAt       time.Time   `json:"created_at"`
    LastUsed        *time.Time  `json:"last_used,omitempty"`
    Status          AccountStatus `json:"status"`
    MissingFields   []string    `json:"missing_fields,omitempty"`
}
```

#### Usage Example

```go
accountService := container.GetAccountService()

// Create new account
account := models.NewAccount("work", "John Doe", "john@company.com", "~/.ssh/id_ed25519_work")
account.GitHubUsername = "john-company"

if err := accountService.CreateAccount(ctx, account); err != nil {
    return fmt.Errorf("failed to create account: %w", err)
}

// Set as current account
if err := accountService.SetCurrentAccount(ctx, "work"); err != nil {
    return fmt.Errorf("failed to set current account: %w", err)
}
```

---

## 🔐 SSH Service

### SSHService Interface

Manages SSH keys, configuration, and validation.

**Interface Location:** `internal/services/interfaces.go:29-46`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `GenerateKey(ctx, keyType, email, keyPath)` | Generate new SSH key | `context.Context, string, string, string` | `*SSHKey, error` |
| `ValidateKey(ctx, keyPath)` | Validate SSH key | `context.Context, string` | `*SSHKeyInfo, error` |
| `ListKeys(ctx)` | List available SSH keys | `context.Context` | `[]*SSHKeyInfo, error` |
| `DeleteKey(ctx, keyPath)` | Delete SSH key | `context.Context, string` | `error` |
| `ValidateConfiguration(ctx)` | Validate SSH config | `context.Context` | `*SSHValidationResult, error` |
| `FixPermissions(ctx, keyPath)` | Fix key permissions | `context.Context, string` | `error` |
| `GenerateSSHConfig(ctx)` | Generate SSH config | `context.Context` | `string, error` |
| `TestGitHubAuthentication(ctx, keyPath)` | Test GitHub auth | `context.Context, string` | `error` |
| `DiagnoseIssues(ctx)` | Diagnose SSH issues | `context.Context` | `[]*SSHIssue, error` |
| `FixIssues(ctx, issues)` | Fix SSH issues | `context.Context, []*SSHIssue` | `error` |

#### SSH Data Types

```go
type SSHKey struct {
    Path        string `json:"path"`
    Type        string `json:"type"`
    Size        int    `json:"size"`
    Fingerprint string `json:"fingerprint"`
    PublicKey   string `json:"public_key"`
    CreatedAt   string `json:"created_at"`
}

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

type SSHValidationResult struct {
    Valid           bool          `json:"valid"`
    Issues          []*SSHIssue   `json:"issues"`
    Recommendations []string      `json:"recommendations"`
    Keys            []*SSHKeyInfo `json:"keys"`
}
```

#### Usage Example

```go
sshService := container.GetSSHService()

// Generate new SSH key
key, err := sshService.GenerateKey(ctx, "ed25519", "user@example.com", "~/.ssh/id_ed25519_work")
if err != nil {
    return fmt.Errorf("failed to generate key: %w", err)
}

// Validate SSH configuration
result, err := sshService.ValidateConfiguration(ctx)
if err != nil {
    return fmt.Errorf("validation failed: %w", err)
}

if !result.Valid {
    fmt.Printf("Found %d SSH issues\n", len(result.Issues))
    // Auto-fix issues
    if err := sshService.FixIssues(ctx, result.Issues); err != nil {
        return fmt.Errorf("failed to fix issues: %w", err)
    }
}
```

---

## 🤖 SSH Agent Service

### SSHAgentService Interface

Manages SSH agent operations for key isolation and switching.

**Interface Location:** `internal/services/interfaces.go:309-328`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `IsAgentRunning(ctx)` | Check if agent is running | `context.Context` | `bool, error` |
| `StartAgent(ctx)` | Start SSH agent | `context.Context` | `error` |
| `StopAgent(ctx)` | Stop SSH agent | `context.Context` | `error` |
| `LoadKey(ctx, keyPath)` | Load SSH key to agent | `context.Context, string` | `error` |
| `UnloadKey(ctx, keyPath)` | Unload SSH key from agent | `context.Context, string` | `error` |
| `ClearAllKeys(ctx)` | Clear all keys from agent | `context.Context` | `error` |
| `ListLoadedKeys(ctx)` | List loaded keys | `context.Context` | `[]string, error` |
| `SwitchToAccount(ctx, keyPath)` | Switch to account key | `context.Context, string` | `error` |
| `IsolateAccount(ctx, keyPath)` | Isolate account key | `context.Context, string` | `error` |
| `GetAgentStatus(ctx)` | Get agent status | `context.Context` | `*SSHAgentStatus, error` |
| `DiagnoseAgentIssues(ctx)` | Diagnose agent issues | `context.Context` | `[]string, error` |

#### SSH Agent Status

```go
type SSHAgentStatus struct {
    Running     bool      `json:"running"`
    PID         int       `json:"pid,omitempty"`
    SocketPath  string    `json:"socket_path,omitempty"`
    LoadedKeys  []string  `json:"loaded_keys"`
    KeyCount    int       `json:"key_count"`
    LastUpdated time.Time `json:"last_updated"`
}
```

#### Usage Example

```go
sshAgentService := container.GetSSHAgentService()

// Get current status
status, err := sshAgentService.GetAgentStatus(ctx)
if err != nil {
    return fmt.Errorf("failed to get status: %w", err)
}

// Clear all keys and load specific key (account isolation)
if err := sshAgentService.ClearAllKeys(ctx); err != nil {
    return fmt.Errorf("failed to clear keys: %w", err)
}

if err := sshAgentService.LoadKey(ctx, "~/.ssh/id_ed25519_work"); err != nil {
    return fmt.Errorf("failed to load key: %w", err)
}

// Or use high-level account switching
if err := sshAgentService.SwitchToAccount(ctx, "~/.ssh/id_ed25519_work"); err != nil {
    return fmt.Errorf("failed to switch account: %w", err)
}
```

---

## ⚙️ Git Config Service

### GitConfigManager Interface

Manages Git configuration for user settings and SSH commands.

**Interface Location:** `internal/services/git_config_service.go:19-24`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `SetUserConfiguration(ctx, name, email)` | Set Git user config | `context.Context, string, string` | `error` |
| `SetSSHCommand(ctx, sshCommand)` | Set Git SSH command | `context.Context, string` | `error` |

#### Extended Methods (Full Service)

```go
// Analysis and validation
AnalyzeConfiguration(ctx context.Context) (*GitConfig, error)
FixConfiguration(ctx context.Context, config *GitConfig) error

// Low-level operations
getUserConfig(ctx context.Context, config *GitConfig) error
getSSHCommandConfig(ctx context.Context, config *GitConfig) error
getRemoteConfig(ctx context.Context, config *GitConfig) error
```

#### Git Configuration Model

```go
type GitConfig struct {
    User struct {
        Name  string
        Email string
    }
    SSHCommand string
    Remotes    map[string]string
    Issues     []GitConfigIssue
}

type GitConfigIssue struct {
    Type        string
    Severity    string
    Description string
    Fix         string
    Fixed       bool
}
```

#### Usage Example

```go
gitService := container.GetGitService()

// Set user configuration for current account
if err := gitService.SetUserConfiguration(ctx, "John Doe", "john@company.com"); err != nil {
    return fmt.Errorf("failed to set user config: %w", err)
}

// Set SSH command for account isolation
sshCmd := "ssh -i ~/.ssh/id_ed25519_work -o IdentitiesOnly=yes"
if err := gitService.SetSSHCommand(ctx, sshCmd); err != nil {
    return fmt.Errorf("failed to set SSH command: %w", err)
}

// Analyze configuration (full service)
if fullService, ok := gitService.(*services.GitConfigService); ok {
    config, err := fullService.AnalyzeConfiguration(ctx)
    if err != nil {
        return fmt.Errorf("analysis failed: %w", err)
    }

    if len(config.Issues) > 0 {
        if err := fullService.FixConfiguration(ctx, config); err != nil {
            return fmt.Errorf("failed to fix issues: %w", err)
        }
    }
}
```

---

## 🐙 GitHub Service

### GitHubService Interface

Integrates with GitHub API for repository and SSH key management.

**Interface Location:** `internal/services/interfaces.go:49-69`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `Authenticate(ctx, token)` | Authenticate with GitHub | `context.Context, string` | `error` |
| `GetAuthenticatedUser(ctx)` | Get current user | `context.Context` | `*GitHubUser, error` |
| `TestSSHKey(ctx, keyPath)` | Test SSH key auth | `context.Context, string` | `error` |
| `ListRepositories(ctx)` | List repositories | `context.Context` | `[]*GitHubRepository, error` |
| `GetRepository(ctx, owner, name)` | Get specific repository | `context.Context, string, string` | `*GitHubRepository, error` |
| `CreateRepository(ctx, name, private)` | Create repository | `context.Context, string, bool` | `*GitHubRepository, error` |
| `DeleteRepository(ctx, owner, name)` | Delete repository | `context.Context, string, string` | `error` |
| `ListSSHKeys(ctx)` | List SSH keys | `context.Context` | `[]*GitHubSSHKey, error` |
| `AddSSHKey(ctx, title, key)` | Add SSH key | `context.Context, string, string` | `*GitHubSSHKey, error` |
| `DeleteSSHKey(ctx, keyID)` | Delete SSH key | `context.Context, int` | `error` |
| `ListOrganizations(ctx)` | List organizations | `context.Context` | `[]*GitHubOrganization, error` |
| `GetOrganization(ctx, name)` | Get organization | `context.Context, string` | `*GitHubOrganization, error` |

#### GitHub Data Types

```go
type GitHubUser struct {
    ID        int    `json:"id"`
    Login     string `json:"login"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    AvatarURL string `json:"avatar_url"`
    Type      string `json:"type"`
}

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

type GitHubSSHKey struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    Key   string `json:"key"`
    URL   string `json:"url"`
}
```

#### Usage Example

```go
githubService := container.GetGitHubService()

// Authenticate with token
if err := githubService.Authenticate(ctx, "ghp_your_token_here"); err != nil {
    return fmt.Errorf("authentication failed: %w", err)
}

// Get authenticated user
user, err := githubService.GetAuthenticatedUser(ctx)
if err != nil {
    return fmt.Errorf("failed to get user: %w", err)
}

// Add SSH key
publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGitpersona_key user@example.com"
sshKey, err := githubService.AddSSHKey(ctx, "GitPersona Work Key", publicKey)
if err != nil {
    return fmt.Errorf("failed to add SSH key: %w", err)
}

fmt.Printf("Added SSH key with ID: %d\n", sshKey.ID)
```

---

## 🏥 Health & Validation Services

### HealthService Interface

Monitors system health and performance.

**Interface Location:** `internal/services/interfaces.go:95-104`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `CheckSystemHealth(ctx)` | Overall system health | `context.Context` | `*HealthStatus, error` |
| `CheckServiceHealth(ctx, serviceName)` | Specific service health | `context.Context, string` | `*ServiceHealth, error` |
| `CheckDependencies(ctx)` | Check dependencies | `context.Context` | `[]*DependencyHealth, error` |
| `GetMetrics(ctx)` | System metrics | `context.Context` | `*SystemMetrics, error` |
| `GetPerformanceStats(ctx)` | Performance statistics | `context.Context` | `*PerformanceStats, error` |

### ValidationService Interface

Provides comprehensive validation across all system components.

**Interface Location:** `internal/services/interfaces.go:107-122`

#### Methods

| Method | Description | Parameters | Returns |
|--------|-------------|------------|---------|
| `ValidateGitConfiguration(ctx)` | Validate Git config | `context.Context` | `*GitValidationResult, error` |
| `FixGitConfiguration(ctx, issues)` | Fix Git issues | `context.Context, []*GitIssue` | `error` |
| `ValidateSSHConfiguration(ctx)` | Validate SSH config | `context.Context` | `*SSHValidationResult, error` |
| `FixSSHConfiguration(ctx, issues)` | Fix SSH issues | `context.Context, []*SSHIssue` | `error` |
| `ValidateAccountConfiguration(ctx, account)` | Validate account | `context.Context, *models.Account` | `*AccountValidationResult, error` |
| `FixAccountConfiguration(ctx, account, issues)` | Fix account issues | `context.Context, *models.Account, []*AccountIssue` | `error` |
| `ValidateSystemConfiguration(ctx)` | System-wide validation | `context.Context` | `*SystemValidationResult, error` |

---

## 🔗 Service Integration Patterns

### Dependency Injection Pattern

```go
type MyCommand struct {
    *commands.BaseCommand
}

func (c *MyCommand) Run(ctx context.Context, args []string) error {
    container := c.GetContainer()

    // Get required services
    configService := container.GetConfigService()
    sshService := container.GetSSHService()
    githubService := container.GetGitHubService()

    // Use services with proper error handling
    if err := configService.Load(ctx); err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Chain service operations
    return c.performOperation(ctx, configService, sshService, githubService)
}
```

### Error Handling Pattern

```go
import "github.com/techishthoughts/GitPersona/internal/errors"

func (c *MyCommand) handleServiceError(err error, operation string) error {
    if err == nil {
        return nil
    }

    // Use GitPersona error types
    return errors.Wrap(err, errors.ErrCodeServiceFailure, fmt.Sprintf("%s failed", operation)).
        WithContext("command", c.Name()).
        WithContext("operation", operation)
}
```

### Service Composition Pattern

```go
func (c *SwitchCommand) performAccountSwitch(ctx context.Context, targetAlias string) error {
    container := c.GetContainer()

    // Get all required services
    configService := container.GetConfigService()
    sshAgentService := container.GetSSHAgentService()
    gitService := container.GetGitService()

    // Load configuration
    if err := configService.Load(ctx); err != nil {
        return fmt.Errorf("config load failed: %w", err)
    }

    // Get target account
    account, err := configService.GetAccount(ctx, targetAlias)
    if err != nil {
        return fmt.Errorf("account not found: %w", err)
    }

    // Orchestrate service operations
    if err := sshAgentService.SwitchToAccount(ctx, account.SSHKeyPath); err != nil {
        return fmt.Errorf("SSH agent switch failed: %w", err)
    }

    if err := gitService.SetUserConfiguration(ctx, account.Name, account.Email); err != nil {
        return fmt.Errorf("Git config update failed: %w", err)
    }

    if err := configService.SetCurrentAccount(ctx, targetAlias); err != nil {
        return fmt.Errorf("current account update failed: %w", err)
    }

    return nil
}
```

---

## 🧪 Testing Services

### Service Testing Pattern

```go
func TestAccountService(t *testing.T) {
    // Create test container
    container := container.NewSimpleContainer()
    ctx := context.Background()

    // Initialize with test configuration
    err := container.Initialize(ctx)
    require.NoError(t, err)

    accountService := container.GetAccountService()

    // Test account creation
    account := models.NewAccount("test", "Test User", "test@example.com", "/tmp/test_key")
    err = accountService.CreateAccount(ctx, account)
    require.NoError(t, err)

    // Test account retrieval
    retrieved, err := accountService.GetAccount(ctx, "test")
    require.NoError(t, err)
    assert.Equal(t, account.Name, retrieved.Name)

    // Cleanup
    container.Shutdown(ctx)
}
```

### Mocking Services

```go
type MockConfigService struct {
    accounts map[string]*models.Account
}

func (m *MockConfigService) GetAccount(ctx context.Context, alias string) (*models.Account, error) {
    if account, exists := m.accounts[alias]; exists {
        return account, nil
    }
    return nil, fmt.Errorf("account not found: %s", alias)
}

func (m *MockConfigService) SetCurrentAccount(ctx context.Context, alias string) error {
    // Mock implementation
    return nil
}

// Use in tests
func TestWithMockService(t *testing.T) {
    mockConfig := &MockConfigService{
        accounts: make(map[string]*models.Account),
    }

    // Test command with mock service
    cmd := NewSwitchCommand()
    err := cmd.performAccountSwitch(ctx, mockConfig, "test", testAccount)
    require.NoError(t, err)
}
```

---

## 📊 Performance Considerations

### Service Call Optimization

- Services are thread-safe with proper synchronization
- Connection pooling for GitHub API calls
- SSH agent operations are cached when possible
- Configuration loading is optimized with lazy initialization

### Memory Management

- Services use context-based cancellation
- Large objects are properly disposed
- Configuration is loaded on-demand
- SSH keys are not kept in memory unnecessarily

### Error Recovery

- Services implement graceful degradation
- Automatic retry logic for transient failures
- Comprehensive logging for debugging
- Health checks for proactive monitoring

This API documentation provides comprehensive coverage of all GitPersona services with practical examples and best practices for integration.
