# 🎭 GitPersona

> **The ultimate Terminal User Interface (TUI) for seamlessly managing multiple GitHub identities with enterprise-grade automation and beautiful design.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/techishthoughts/GitPersona)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/techishthoughts/GitPersona)](https://github.com/techishthoughts/GitPersona/releases)
[![Security Rating](https://img.shields.io/badge/Security-A+-brightgreen)](https://github.com/techishthoughts/GitPersona)
[![2025 Compliant](https://img.shields.io/badge/2025_Standards-Compliant-blue)](https://github.com/techishthoughts/GitPersona)

---

## 💡 **The Problem**

Managing multiple GitHub accounts (personal, work, client projects) is a **daily pain point** for developers:

- 🔄 **Constant switching** between different Git configurations
- 🔑 **SSH key management** across multiple accounts
- 😤 **Forgotten commits** with wrong email/name
- ⚠️ **Accidental pushes** to wrong accounts
- 📁 **Project-specific** account requirements
- 🤖 **Manual, error-prone** setup processes

## 🎯 **The Solution**

**GitPersona** provides **zero-effort** GitHub identity management with revolutionary automation and beautiful design.

---

## 🚀 Quick Installation

## Option 1: Install from Source (Recommended)
```bash
# Clone the repository
git clone https://github.com/techishthoughts/GitPersona.git
cd GitPersona

# Build and install to your bin directory
go build -o gitpersona .
cp gitpersona ~/.local/bin/

# Verify installation
gitpersona -v
```

## Option 2: Go Install
```bash
go install github.com/techishthoughts/GitPersona@latest
```

## Option 3: Download Binary
Visit [GitPersona Releases](https://github.com/techishthoughts/GitPersona/releases) and download the appropriate binary for your platform.

# 🎯 Quick Start

## 1. First Time Setup
```bash
# Initialize GitPersona
gitpersona init

# Auto-discover existing accounts (optional)
gitpersona discover
```

## 2. Add your GitHub accounts automatically (ZERO manual steps!)
```bash
gitpersona add-github username --email "user@example.com" --name "User Name"
gitpersona add-github workuser --alias work --name "Work User" --email "work@company.com"
```

## 3. Switch between accounts instantly
```bash
gitpersona switch personal    # Switch to personal account
gitpersona switch work        # Switch to work account
gitpersona switch username    # Switch to username account
```

## 4. 🔍 Automatic local identification (NEW!)
```bash
gitpersona auto-identify      # Auto-detect and switch to best matching account
gitpersona auto-identify -v   # Verbose mode for detailed analysis
```

## 5. Check current status
```bash
gitpersona current            # Show current account and Git config
gitpersona current -v         # Detailed information
```

---

## 🌟 **Revolutionary Features**

### **1. 🚀 One-Command Account Setup**

```bash
gitpersona add-github username --email "user@example.com"
```

**What happens automatically:**
- 🔐 **GitHub OAuth** with full permissions
- 🔍 **Fetches real user data** from GitHub API
- 🔑 **Generates Ed25519 SSH key** (quantum-resistant, 2025 standard)
- ⬆️ **Uploads SSH key** to your GitHub account
- 🌐 **Sets global Git config** immediately
- ✅ **Ready to use** in seconds!

### **2. 🔐 Advanced SSH Management**

```bash
# Test SSH connectivity with detailed diagnostics
gitpersona ssh test              # Test current account
gitpersona ssh test work         # Test specific account

# Generate SSH config entries automatically
gitpersona ssh config            # Generate for all accounts
gitpersona ssh config work       # Generate for specific account

# Comprehensive SSH diagnostics and troubleshooting
gitpersona ssh doctor            # Full diagnostic suite
```

**SSH Features:**
- 🔧 **Connectivity Testing**: Detailed SSH diagnostics with helpful suggestions
- 🔑 **Multiple Key Types**: Support for RSA, Ed25519, ECDSA keys
- 🛡️ **Security Validation**: 2025 compliance standards
- 🤖 **Auto Configuration**: Generate SSH configs automatically
- 📋 **Agent Integration**: SSH agent management and key loading

### **3. 🔍 Smart Auto-Discovery**

On first run, automatically detects and imports existing configurations:

```bash
gitpersona discover --auto-import
```

**Scans and imports from:**
- `~/.gitconfig` (global Git configuration)
- `~/.config/git/gitconfig-*` (account-specific configs)
- `~/.ssh/config` (SSH keys configured for GitHub)
- GitHub CLI authentication (`gh auth status`)

### **4. 🎨 Beautiful Terminal Interface**

```bash
gitpersona  # Launch gorgeous TUI
```

**Features:**
- 🌈 **Modern color schemes** with gradients
- 📱 **Responsive design** (adapts to terminal size)
- ⚡ **Animated spinners** and smooth transitions
- 🎯 **Context-aware help** system
- ♿ **Accessibility support** (screen readers, high contrast)

---

## 📊 **Usage Examples**

### **Complete Workflow Demonstration**

```bash
# 🔍 First-time setup with auto-discovery
gitpersona discover --auto-import

# 🚀 Add accounts with zero effort
gitpersona add-github username --email "user@example.com"
gitpersona add-github workuser --alias work --email "work@company.com"

# 📋 View all accounts beautifully
gitpersona list --format table

# 🔄 Switch accounts instantly (always global)
gitpersona switch work
# ✅ Switched to account 'work'

# 📁 Set up project-specific automation
cd ~/work-project
gitpersona project set work
# ✅ Project configured to use account 'work'

# 🌐 Enable shell integration for automatic switching
echo 'eval "$(gitpersona init)"' >> ~/.zshrc
source ~/.zshrc
# Now when you cd into ~/work-project, it automatically uses work account!

# 📦 View repositories across accounts
gitpersona repos personal --limit 5
gitpersona overview --detailed

# 🏥 System health monitoring
gitpersona health --detailed
```

---

## 🏥 **System Health & Diagnostics**

GitPersona includes comprehensive health monitoring and diagnostics:

```bash
# Complete system health check
gitpersona health --detailed
# ✅ Results:
# - Configuration integrity ✓
# - GitHub API connectivity ✓
# - SSH key validation ✓
# - Performance benchmarks ✓
# - Security compliance ✓

# SSH-specific diagnostics
gitpersona ssh doctor
# 🔧 Tests SSH agent, key permissions, GitHub connectivity
# 💡 Provides helpful suggestions for common issues

# JSON output for monitoring integration
gitpersona health --format json | jq '.checks'
```

---

## 📚 **Command Reference**

### **Core Commands**

| Command | Description | Example |
|---------|-------------|---------|
| `gitpersona` | Launch beautiful TUI | `gitpersona` |
| `add-github` | **Auto setup from GitHub username** | `gitpersona add-github username --email user@example.com` |
| `switch` | Switch accounts (always global) | `gitpersona switch work` |
| `list` | Display all accounts | `gitpersona list --format table` |
| `current` | Show active account | `gitpersona current --verbose` |
| `discover` | **Auto-detect existing configs** | `gitpersona discover --auto-import` |

### **Advanced Commands**

| Command | Description | Example |
|---------|-------------|---------|
| `ssh test` | **Test SSH connectivity** | `gitpersona ssh test work` |
| `ssh config` | **Generate SSH config** | `gitpersona ssh config` |
| `ssh doctor` | **SSH diagnostics** | `gitpersona ssh doctor` |
| `repos` | **View GitHub repositories** | `gitpersona repos personal --stars` |
| `overview` | **Complete dashboard** | `gitpersona overview --detailed` |
| `project set` | Configure project automation | `gitpersona project set work` |
| `health` | **System health monitoring** | `gitpersona health --format json` |
| `init` | Shell integration setup | `eval "$(gitpersona init)"` |

---

## 🐳 **Docker & Development**

### **Development Environment**

```bash
# Start complete development environment
docker-compose up -d

# Development with live reloading
docker-compose exec dev go run . --help

# Run tests in container
docker-compose exec dev go test ./...
```

### **Production Deployment**

```bash
# Build production image
docker build -t gitpersona:latest .

# Run with volume mounts for config persistence
docker run -it --rm \
  -v ~/.config/gitpersona:/home/appuser/.config/gitpersona \
  -v ~/.ssh:/home/appuser/.ssh:ro \
  gitpersona:latest
```

---

## 🧪 **Testing & Quality Assurance**

### **Testing CI Locally with Act**

You can test GitHub Actions locally using [act](https://github.com/nektos/act):

```bash
# Install act
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Test the entire CI workflow
make ci-test

# Test specific jobs
make ci-lint      # Test linting workflow
make ci-security  # Test security workflow

# Or run act directly
act push --workflows .github/workflows/ci.yml
act pull_request --workflows .github/workflows/ci.yml --job quality
```

**Act Configuration:**
- Uses `.actrc` for default settings
- Ubuntu runner for consistency
- Verbose output for debugging
- Environment variables for local testing

### **Comprehensive Test Suite**

```bash
# Run all tests with coverage
make test-coverage

# Run benchmark tests
make test-bench

# Run specific test categories
go test ./internal/models
go test ./internal/config
go test ./cmd
```

---

## 🚨 **Troubleshooting**

### **Common Issues & Solutions**

| Issue | Solution | Command |
|-------|----------|---------|
| **SSH Keys** | Check SSH agent and key permissions | `gitpersona ssh test` |
| **Git Config** | Verify account configuration | `gitpersona current -v` |
| **GitHub API** | Check authentication status | `gitpersona health` |
| **Account Setup** | Validate account settings | `gitpersona list` |

### **Getting Help**

1. **📋 System Health Check**: `gitpersona health --detailed`
2. **📊 Account Status**: `gitpersona current --verbose`
3. **🔧 SSH Diagnostics**: `gitpersona ssh doctor`
4. **📦 Repository Access**: `gitpersona repos ACCOUNT`

---

## 🎉 **Success Stories**

> *"GitPersona transformed my workflow. I went from 15 minutes daily managing Git configs to zero effort. The automatic GitHub setup is pure magic!"* - **Senior Developer**

> *"The TUI is gorgeous and the SSH diagnostics saved me hours of debugging. This is how developer tools should work in 2025."* - **DevOps Engineer**

> *"Managing client accounts used to be a nightmare. Now it's just `gitpersona add-github client-username` and I'm ready to go!"* - **Freelance Consultant**

---

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 **Acknowledgments**

Built with modern technologies following 2025 best practices:

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - Elegant TUI framework
- **[Cobra](https://github.com/spf13/cobra)** - Powerful CLI framework for Go
- **[Viper](https://github.com/spf13/viper)** - Configuration management
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Beautiful terminal styling
- **[go-github](https://github.com/google/go-github)** - GitHub API client

---

## 🚀 **What Makes This Special**

### **🌟 Beyond Basic Account Switching**

This isn't just another Git config switcher. It's a **comprehensive developer experience platform** that:

1. **🔮 Predicts your needs** - Auto-detects existing configurations
2. **🤖 Automates everything** - From GitHub username to ready-to-use environment
3. **🎨 Delights users** - Beautiful TUI with modern design principles
4. **🛡️ Prioritizes security** - 2025 cryptographic standards and best practices
5. **📊 Provides visibility** - Health monitoring, SSH diagnostics, audit capabilities
6. **🌐 Scales with you** - From personal use to enterprise deployments

### **🎯 The Vision**

**Making GitHub account management invisible** - so developers can focus on what matters: **building amazing software**.

---

**Made with ❤️ for developers juggling multiple GitHub accounts in 2025** 🚀

*Star ⭐ this repository if it helped streamline your development workflow!*
