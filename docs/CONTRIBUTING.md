# 🤝 gitshift Contributing Guide

> **How to contribute to gitshift and make it better for everyone**

---

## 📖 **Table of Contents**

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Code Style Guidelines](#code-style-guidelines)
4. [Testing Guidelines](#testing-guidelines)
5. [Pull Request Process](#pull-request-process)
6. [Issue Guidelines](#issue-guidelines)
7. [Documentation Guidelines](#documentation-guidelines)
8. [Release Process](#release-process)
9. [Community Guidelines](#community-guidelines)

---

## 🚀 **Getting Started**

### **Ways to Contribute**

- 🐛 **Bug Reports**: Report issues and help us fix them
- ✨ **Feature Requests**: Suggest new features and improvements
- 💻 **Code Contributions**: Submit bug fixes and new features
- 📚 **Documentation**: Improve documentation and examples
- 🧪 **Testing**: Help us test and improve quality
- 💬 **Community Support**: Help other users in discussions

### **Before You Start**

1. **Check existing issues** - Your idea might already be discussed
2. **Read the documentation** - Understand the project structure
3. **Join the community** - Participate in discussions
4. **Start small** - Begin with good first issues

---

## 🛠️ **Development Setup**

### **Prerequisites**

- **Go 1.24+** - [Download Go](https://golang.org/dl/)
- **Git** - [Download Git](https://git-scm.com/downloads)
- **GitHub CLI** (optional) - [Download gh](https://cli.github.com/)

### **Fork and Clone**

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/gitshift.git
cd gitshift

# 3. Add upstream remote
git remote add upstream https://github.com/techishthoughts-org/gitshift.git

# 4. Verify remotes
git remote -v
# origin    https://github.com/YOUR_USERNAME/gitshift.git (fetch)
# origin    https://github.com/YOUR_USERNAME/gitshift.git (push)
# upstream  https://github.com/techishthoughts-org/gitshift.git (fetch)
# upstream  https://github.com/techishthoughts-org/gitshift.git (push)
```

### **Development Environment**

```bash
# 1. Install dependencies
go mod download

# 2. Build the project
go build -o gitshift

# 3. Run tests
go test ./...

# 4. Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

### **IDE Setup**

#### **VS Code**
```json
// .vscode/settings.json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.testFlags": ["-v"],
    "go.buildTags": "",
    "go.testTimeout": "30s"
}
```

#### **GoLand/IntelliJ**
- Enable Go modules
- Configure Go linter to use golangci-lint
- Set up code formatting on save

---

## 📝 **Code Style Guidelines**

### **Go Code Style**

#### **Formatting**
```bash
# Use gofmt for formatting
gofmt -s -w .

# Use goimports for import organization
goimports -w .

# Use golangci-lint for comprehensive linting
golangci-lint run
```

#### **Naming Conventions**
```go
// Package names: lowercase, single word
package services

// Types: PascalCase
type AccountService struct {}

// Interfaces: PascalCase with descriptive names
type ConfigurationService interface {}

// Functions: PascalCase for public, camelCase for private
func NewAccountService() *AccountService {}
func (s *AccountService) validateAccount() error {}

// Constants: PascalCase or UPPER_CASE
const DefaultConfigPath = "~/.config/gitshift"
const MAX_RETRY_ATTEMPTS = 3

// Variables: camelCase
var currentAccount string
var configManager *Manager
```

#### **Error Handling**
```go
// Use wrapped errors with context
func (s *Service) DoSomething(input string) error {
    if input == "" {
        return errors.New("input cannot be empty").
            WithContext("field", "input").
            WithContext("value", input)
    }

    result, err := s.processInput(input)
    if err != nil {
        return errors.Wrap(err, "failed to process input").
            WithContext("operation", "process_input").
            WithContext("input", input)
    }

    return nil
}
```

#### **Logging**
```go
// Use structured logging with context
func (s *Service) DoSomething(ctx context.Context, input string) error {
    s.logger.Info(ctx, "starting_operation",
        observability.F("input", input),
        observability.F("operation", "do_something"),
    )

    // ... operation logic ...

    s.logger.Info(ctx, "operation_completed",
        observability.F("input", input),
        observability.F("result", result),
    )

    return nil
}
```

### **Project Structure**

```
internal/
├── commands/          # CLI command implementations
│   ├── category.go    # Command categories
│   └── command.go     # Base command structure
├── config/           # Configuration management
│   ├── config.go     # Configuration logic
│   ├── config_test.go # Configuration tests
│   └── validation.go  # Configuration validation
├── container/        # Dependency injection
│   ├── container_simple.go      # Service container
│   └── container_simple_test.go # Container tests
├── services/         # Business logic services
│   ├── interfaces.go # Service interfaces
│   ├── account_service.go       # Account management
│   ├── ssh_service.go           # SSH operations
│   └── github_service.go        # GitHub integration
└── models/          # Data models
    ├── account.go    # Account model
    ├── account_test.go # Account tests
    └── errors.go     # Error definitions
```

### **File Organization**

```go
// File header with package and imports
package services

import (
    "context"
    "fmt"

    "github.com/techishthoughts/gitshift/internal/observability"
)

// Type definitions
type MyService struct {
    logger observability.Logger
}

// Constructor
func NewMyService(logger observability.Logger) *MyService {
    return &MyService{
        logger: logger,
    }
}

// Public methods
func (s *MyService) PublicMethod(ctx context.Context) error {
    return s.privateMethod(ctx)
}

// Private methods
func (s *MyService) privateMethod(ctx context.Context) error {
    // Implementation
    return nil
}
```

---

## 🧪 **Testing Guidelines**

### **Test Structure**

```go
// Test file: service_test.go
package services

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMyService_PublicMethod(t *testing.T) {
    // Arrange
    logger := observability.NewTestLogger()
    service := NewMyService(logger)
    ctx := context.Background()

    // Act
    err := service.PublicMethod(ctx)

    // Assert
    assert.NoError(t, err)
}

func TestMyService_PublicMethod_ErrorCase(t *testing.T) {
    // Arrange
    logger := observability.NewTestLogger()
    service := NewMyService(logger)
    ctx := context.Background()

    // Act
    err := service.PublicMethod(ctx)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "expected error message")
}
```

### **Test Categories**

#### **Unit Tests**
```go
// Test individual functions and methods
func TestAccount_Validate(t *testing.T) {
    tests := []struct {
        name    string
        account *models.Account
        wantErr bool
    }{
        {
            name: "valid account",
            account: &models.Account{
                Alias:          "test",
                Name:           "Test User",
                Email:          "test@example.com",
                GitHubUsername: "testuser",
            },
            wantErr: false,
        },
        {
            name: "missing alias",
            account: &models.Account{
                Name:           "Test User",
                Email:          "test@example.com",
                GitHubUsername: "testuser",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.account.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### **Integration Tests**
```go
// Test service interactions
func TestAccountService_CreateAccount(t *testing.T) {
    // Setup
    tempDir := t.TempDir()
    configService := services.NewRealConfigService(tempDir, logger)
    accountService := services.NewRealAccountService(configService, logger)

    // Test
    account := &models.Account{
        Alias:          "test",
        Name:           "Test User",
        Email:          "test@example.com",
        GitHubUsername: "testuser",
    }

    err := accountService.CreateAccount(context.Background(), account)
    assert.NoError(t, err)

    // Verify
    retrieved, err := accountService.GetAccount(context.Background(), "test")
    assert.NoError(t, err)
    assert.Equal(t, account.Alias, retrieved.Alias)
}
```

#### **Mock Testing**
```go
// Use mocks for external dependencies
type MockSSHService struct {
    GenerateKeyFunc func(ctx context.Context, keyType string, email string, keyPath string) (*SSHKey, error)
}

func (m *MockSSHService) GenerateKey(ctx context.Context, keyType string, email string, keyPath string) (*SSHKey, error) {
    if m.GenerateKeyFunc != nil {
        return m.GenerateKeyFunc(ctx, keyType, email, keyPath)
    }
    return nil, nil
}

func TestAccountService_WithMockSSH(t *testing.T) {
    mockSSH := &MockSSHService{
        GenerateKeyFunc: func(ctx context.Context, keyType string, email string, keyPath string) (*SSHKey, error) {
            return &SSHKey{Path: keyPath}, nil
        },
    }

    // Test with mock
    // ...
}
```

### **Running Tests**

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -run TestAccount_Validate ./internal/models/

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

---

## 🔄 **Pull Request Process**

### **Before Submitting**

1. **Create a feature branch**
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

2. **Make your changes**
```bash
# Make your code changes
# Add tests
# Update documentation
```

3. **Run quality checks**
```bash
# Format code
gofmt -s -w .
goimports -w .

# Run linter
golangci-lint run

# Run tests
go test ./...

# Run security checks
gosec ./...

# Build the project
go build -o gitshift
```

4. **Commit your changes**
```bash
# Use semantic commit messages
git add .
git commit -m "feat: add new feature for account management

- Add new account validation logic
- Update configuration schema
- Add comprehensive tests
- Update documentation

Fixes #123"
```

### **Pull Request Template**

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed
- [ ] All existing tests pass

## Checklist
- [ ] Code follows the project's style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or clearly documented)
- [ ] Tests added/updated
- [ ] Performance impact considered

## Related Issues
Fixes #123
Closes #456
```

### **Review Process**

1. **Automated Checks**
   - Code formatting
   - Linting
   - Tests
   - Security scans
   - Build verification

2. **Manual Review**
   - Code quality
   - Architecture alignment
   - Test coverage
   - Documentation completeness

3. **Feedback Integration**
   - Address review comments
   - Update code as needed
   - Re-run tests
   - Update documentation

---

## 🐛 **Issue Guidelines**

### **Bug Reports**

#### **Bug Report Template**
```markdown
**Bug Description**
A clear and concise description of what the bug is.

**Steps to Reproduce**
1. Run command: `gitshift switch work`
2. See error: `Permission denied (publickey)`

**Expected Behavior**
Account should switch successfully

**Actual Behavior**
Switch fails with SSH authentication error

**Environment**
- OS: macOS 14.0
- gitshift Version: 1.0.0
- Go Version: 1.21.0
- Shell: zsh 5.9

**Debug Information**
```
$ gitshift diagnose --include-system
[Output here]
```

**Additional Context**
Any other relevant information about the problem.
```

### **Feature Requests**

#### **Feature Request Template**
```markdown
**Feature Description**
A clear and concise description of the feature you'd like to see.

**Use Case**
Describe the problem this feature would solve or the workflow it would improve.

**Proposed Solution**
Describe how you think this feature should work.

**Alternatives Considered**
Describe any alternative solutions or features you've considered.

**Additional Context**
Add any other context, mockups, or examples about the feature request.
```

### **Good First Issues**

Look for issues labeled with:
- `good first issue`
- `help wanted`
- `documentation`
- `testing`

---

## 📚 **Documentation Guidelines**

### **Documentation Types**

#### **Code Documentation**
```go
// Package services provides business logic services for gitshift.
package services

// AccountService handles account management operations.
// It provides methods for creating, updating, deleting, and validating accounts.
type AccountService interface {
    // GetAccount retrieves an account by its alias.
    // Returns an error if the account is not found.
    GetAccount(ctx context.Context, alias string) (*models.Account, error)

    // CreateAccount creates a new account with the provided information.
    // Returns an error if the account already exists or validation fails.
    CreateAccount(ctx context.Context, account *models.Account) error
}
```

#### **User Documentation**
```markdown
# Command Name

Brief description of what the command does.

## Usage

```bash
gitshift command [flags] [arguments]
```

## Examples

```bash
# Basic usage
gitshift command example

# With flags
gitshift command --flag value
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `--flag` | Description of flag | `default` |
```

### **Documentation Standards**

- **Clear and concise** - Write for your audience
- **Examples included** - Show real usage scenarios
- **Up-to-date** - Keep documentation current with code
- **Searchable** - Use clear headings and structure
- **Accessible** - Use inclusive language and clear formatting

---

## 🚀 **Release Process**

### **Version Numbering**

We use [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### **Release Checklist**

1. **Update version numbers**
   - `VERSION` file
   - `go.mod` file
   - Documentation

2. **Update changelog**
   - Add new features
   - List bug fixes
   - Note breaking changes

3. **Run full test suite**
   - Unit tests
   - Integration tests
   - Manual testing

4. **Create release**
   - Tag the release
   - Create GitHub release
   - Upload binaries

---

## 👥 **Community Guidelines**

### **Code of Conduct**

We are committed to providing a welcoming and inclusive environment for all contributors. Please:

- **Be respectful** - Treat everyone with respect and kindness
- **Be constructive** - Provide helpful feedback and suggestions
- **Be patient** - Remember that everyone is learning and contributing
- **Be collaborative** - Work together to improve the project

### **Communication Channels**

- **GitHub Issues** - Bug reports and feature requests
- **GitHub Discussions** - General questions and community support
- **Pull Requests** - Code contributions and reviews
- **Email** - Security issues and private matters

### **Getting Help**

- **Read the documentation** - Check existing guides first
- **Search issues** - Your question might already be answered
- **Ask in discussions** - Community members are happy to help
- **Be specific** - Provide context and details when asking questions

---

## 🏆 **Recognition**

### **Contributor Recognition**

We recognize contributors in several ways:
- **Contributors list** - All contributors are listed in the project
- **Release notes** - Contributors are credited in release notes
- **Special badges** - Significant contributors receive special recognition
- **Community highlights** - Outstanding contributions are highlighted

### **Types of Contributions**

- **Code contributions** - Bug fixes, features, improvements
- **Documentation** - Guides, examples, API documentation
- **Testing** - Test cases, bug reports, quality assurance
- **Community** - Helping others, answering questions
- **Design** - UI/UX improvements, user experience

---

## 📚 **Additional Resources**

- **[User Guide](USER_GUIDE.md)** - Complete user documentation
- **[Configuration Guide](CONFIGURATION.md)** - Detailed configuration options
- **[Architecture Guide](ARCHITECTURE.md)** - Technical architecture details
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Common issues and solutions

---

<div align="center">

**Ready to contribute?**

- 🐛 **[Report a Bug](https://github.com/techishthoughts-org/gitshift/issues)**
- ✨ **[Request a Feature](https://github.com/techishthoughts-org/gitshift/issues)**
- 💻 **[Submit a Pull Request](https://github.com/techishthoughts-org/gitshift/pulls)**
- 💬 **[Join Discussions](https://github.com/techishthoughts-org/gitshift/discussions)**

**Thank you for contributing to gitshift!** 🙏

</div>
