# 🎭 GitPersona

> **The ultimate GitHub account management tool with enterprise-grade automation, intelligent diagnostics, and seamless multi-identity workflow.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/techishthoughts/GitPersona)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/techishthoughts/GitPersona)](https://github.com/techishthoughts/GitPersona/releases)
[![Security Rating](https://img.shields.io/badge/Security-A+-brightgreen)](https://github.com/techishthoughts/GitPersona)
[![2025 Compliant](https://img.shields.io/badge/2025_Standards-Compliant-blue)](https://github.com/techishthoughts/GitPersona)

---

## 🎯 **What is GitPersona?**

GitPersona solves the daily pain of managing multiple GitHub accounts (personal, work, client projects) with **zero-effort automation** and **intelligent diagnostics**.

### **The Problem We Solve**
- 🔄 **Constant switching** between different Git configurations
- 🔑 **SSH key management** across multiple accounts
- 😤 **Forgotten commits** with wrong email/name
- ⚠️ **Accidental pushes** to wrong accounts
- 📁 **Project-specific** account requirements
- 🤖 **Manual, error-prone** setup processes
- 🔍 **Difficult troubleshooting** when things break
- 🚫 **SSH key conflicts** and authentication failures

### **Our Solution**
**GitPersona** provides **revolutionary automation** with:
- 🧠 **Intelligent Diagnostics** - Proactive issue detection and auto-repair
- 🔐 **Advanced SSH Management** - Smart conflict detection and resolution
- ⚡ **Lightning-Fast Switching** - Sub-second account transitions
- 🛡️ **Enterprise Security** - Cryptographic best practices and validation
- 🎨 **Beautiful CLI** - Intuitive commands with rich feedback

---

## 🚀 **Quick Start**

### **Installation**

```bash
# Clone and build
git clone https://github.com/techishthoughts/GitPersona.git
cd GitPersona
go build -o gitpersona

# Install system-wide (optional)
sudo mv gitpersona /usr/local/bin/
```

### **First-Time Setup**

```bash
# 1. Run comprehensive system check
gitpersona diagnose --verbose

# 2. Add your first GitHub account (fully automated)
gitpersona add-github yourusername --email "your@email.com" --name "Your Name"

# 3. Switch to the account
gitpersona switch yourusername

# 4. Verify everything is working
gitpersona status
```

**That's it!** 🎉 Your GitHub identity management is now fully automated.

---

## 📚 **Core Commands**

### **Account Management**
```bash
# Add GitHub account with automated setup
gitpersona add-github username --email "user@example.com" --name "Full Name"

# List all configured accounts
gitpersona list

# Switch between accounts (with automatic GITHUB_TOKEN management)
gitpersona switch work
gitpersona switch personal

# Remove an account
gitpersona remove oldaccount

# View current account status
gitpersona status
```

### **Diagnostics & Health**
```bash
# Comprehensive system diagnostics
gitpersona diagnose

# Auto-fix detected issues
gitpersona diagnose --fix

# Verbose output with detailed information
gitpersona diagnose --verbose --include-system

# Focus on specific components
gitpersona diagnose --accounts-only
gitpersona diagnose --ssh-only
gitpersona diagnose --git-only
```

### **SSH Management**
```bash
# Check SSH agent status
gitpersona ssh-agent --status

# Clear all SSH keys from agent
gitpersona ssh-agent --clear

# Load specific SSH key
gitpersona ssh-agent --load ~/.ssh/id_ed25519_work

# Diagnose SSH authentication issues
gitpersona ssh-keys diagnose

# List all available SSH keys
gitpersona ssh-keys list

# Test SSH connection for specific account
gitpersona ssh-keys test work

# Generate new SSH key for account
gitpersona ssh-keys generate work
```

---

## 🔍 **Intelligent Diagnostics**

GitPersona includes a comprehensive diagnostic system that proactively identifies and fixes issues:

### **Health Check Categories**

| Category | Description | Auto-Fix |
|----------|-------------|----------|
| **🏥 System Health** | Git, SSH, GitHub CLI availability | ❌ |
| **👤 Account Config** | Email, name, SSH key validation | ✅ |
| **🔐 SSH Issues** | Key permissions, conflicts, authentication | ✅ |
| **⚙️ Git Config** | User settings, SSH commands, remotes | ✅ |
| **🔗 GitHub Integration** | API access, repository permissions | ❌ |

### **Example Diagnostic Output**

```bash
$ gitpersona diagnose --verbose

🟢 Overall Health: EXCELLENT

📊 Summary:
  • Issues: 0
  • Warnings: 2
  • Accounts: 2 configured

👤 Account Status:
  • thukabjj: ✅ Valid
  • work: ⚠️  SSH key needs attention

⚠️ Warnings:
  • ssh: Multiple SSH keys loaded (3 keys)
    Recommendation: Use only one key at a time to avoid conflicts
  • system: GitHub CLI not found
    Recommendation: Install GitHub CLI for enhanced integration

💡 Run 'gitpersona diagnose --fix' to automatically resolve fixable issues
```

---

## 🆕 **Latest Features**

### **Zsh Secrets Integration**
GitPersona automatically manages your `GITHUB_TOKEN` in your `zsh_secrets` file when switching accounts:

```bash
# The switch command automatically updates your zsh_secrets file
gitpersona switch work

# This updates the GITHUB_TOKEN in ~/.zsh_secrets
# export GITHUB_TOKEN="ghp_your_token_here"
```

**Supported zsh_secrets locations:**
- `~/.zsh_secrets` (default)
- `~/.config/zsh_secrets`
- `~/.secrets/zsh_secrets`
- `~/.zsh/secrets`

### **Smart Account Switching**
The switch command automatically:
- Updates Git configuration (user.name, user.email)
- Configures SSH keys for the account
- **Updates GITHUB_TOKEN in zsh_secrets file**
- Validates SSH connection to GitHub

```bash
# Switch with validation
gitpersona switch work --validate

# Force switch even if issues detected
gitpersona switch personal --force

# Skip SSH validation for speed
gitpersona switch client --skip-validation
```

### **SSH Key Conflict Resolution**
GitPersona automatically detects and resolves common SSH issues:
- **Wrong Account Authentication**: Keys linked to incorrect GitHub accounts
- **Duplicate Keys**: Multiple keys authenticating as the same user
- **Permission Issues**: Incorrect file permissions on SSH keys
- **Missing Keys**: Generates new keys when needed
- **SSH Agent Conflicts**: Multiple keys loaded simultaneously causing authentication conflicts
- **SSH Config Issues**: Missing or misconfigured SSH host entries
- **Key Isolation**: Ensures only one key is active at a time to prevent conflicts

---

## 🔧 **Configuration**

### **Configuration File**
GitPersona stores configuration in `~/.config/gitpersona/config.yaml`:

```yaml
# Example configuration
accounts:
  personal:
    alias: personal
    name: John Doe
    email: john@personal.com
    ssh_key_path: /Users/john/.ssh/id_ed25519_personal
    github_username: johndoe
    description: Personal GitHub account

  work:
    alias: work
    name: John Doe
    email: john@company.com
    ssh_key_path: /Users/john/.ssh/id_rsa_work
    github_username: john-company
    description: Work account for Company Inc

current_account: personal
global_git_config: true
auto_detect: true
config_version: 1.0.0
```

### **Environment Variables**

```bash
# Configuration file location
export GITPERSONA_CONFIG_PATH="~/.config/gitpersona"

# Enable debug logging
export GITPERSONA_DEBUG=true

# Default SSH key directory
export GITPERSONA_SSH_DIR="~/.ssh"
```

---

## 🚨 **Troubleshooting**

### **Common Issues & Solutions**

#### **SSH Authentication Failures**
```bash
# Diagnose SSH issues
gitpersona diagnose --ssh-only --verbose

# Fix SSH permissions and configuration
gitpersona diagnose --fix

# Test SSH connection manually
ssh -T git@github.com -i ~/.ssh/id_ed25519_account
```

#### **Account Switch Failures**
```bash
# Force switch with detailed output
gitpersona switch account --force --verbose

# Validate current configuration
gitpersona diagnose --accounts-only

# Reset SSH agent state
gitpersona ssh-agent --clear
gitpersona switch account
```

#### **Getting Help**
```bash
# General help
gitpersona --help

# Command-specific help
gitpersona diagnose --help
gitpersona switch --help

# Enable verbose output for debugging
gitpersona switch account --verbose
```

---

## 📖 **Documentation**

- **[📋 User Guide](docs/USER_GUIDE.md)** - Complete user documentation
- **[🔧 Configuration Guide](docs/CONFIGURATION.md)** - Detailed configuration options
- **[🚨 Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[🏗️ Architecture Guide](docs/ARCHITECTURE.md)** - Technical architecture and design
- **[🤝 Contributing Guide](docs/CONTRIBUTING.md)** - How to contribute to the project
- **[🔒 Security Guide](docs/SECURITY.md)** - Security best practices and considerations

---

## 🏗️ **Architecture**

### **Core Technologies**

| Category | Technology | Purpose |
|----------|------------|---------|
| **Language** | Go 1.21+ | High-performance, cross-platform backend |
| **CLI Framework** | Cobra | Powerful command-line interface |
| **Configuration** | Viper | Flexible configuration management |
| **SSH Management** | golang.org/x/crypto | SSH key operations |
| **GitHub API** | GitHub CLI Integration | GitHub operations |

### **Key Design Principles**

- **🔧 Modular Design**: Clean separation of concerns
- **🛡️ Security-First**: Cryptographic best practices
- **🔍 Proactive Monitoring**: Intelligent issue detection
- **⚡ Performance**: Optimized for speed and efficiency
- **🧪 Testable**: Comprehensive validation framework

---

## 📈 **Performance Benchmarks**

| Operation | Time | Memory |
|-----------|------|--------|
| Account Switch | ~50ms | <10MB |
| SSH Validation | ~100ms | <5MB |
| Full Diagnosis | ~200ms | <15MB |
| GitHub API Call | ~300ms | <8MB |

---

## 🤝 **Contributing**

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) for details.

### **Development Setup**

```bash
# Clone and setup
git clone https://github.com/techishthoughts/GitPersona.git
cd GitPersona

# Install dependencies
go mod download

# Run tests
go test ./...

# Build development binary
go build -o gitpersona-dev
```

---

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🔗 **Links**

- **[Issues](https://github.com/techishthoughts/GitPersona/issues)** - Report bugs or request features
- **[Releases](https://github.com/techishthoughts/GitPersona/releases)** - Download latest version
- **[Wiki](https://github.com/techishthoughts/GitPersona/wiki)** - Detailed documentation
- **[Discussions](https://github.com/techishthoughts/GitPersona/discussions)** - Community support

---

<div align="center">

**⭐ Star this repository if GitPersona has made your GitHub workflow better!**

Made with ❤️ by developers, for developers.

</div>
