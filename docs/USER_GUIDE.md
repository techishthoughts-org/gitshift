# 📋 gitshift User Guide

> **Complete guide to using gitshift for GitHub account management**

---

## 📖 **Table of Contents**

1. [Getting Started](#getting-started)
2. [Account Management](#account-management)
3. [SSH Key Management](#ssh-key-management)
4. [Account Switching](#account-switching)
5. [Diagnostics & Health Checks](#diagnostics--health-checks)
6. [Configuration](#configuration)
7. [Advanced Features](#advanced-features)
8. [Best Practices](#best-practices)
9. [Troubleshooting](#troubleshooting)

---

## 🚀 **Getting Started**

### **Installation**

#### **From Source**
```bash
# Clone the repository
git clone https://github.com/techishthoughts-org/gitshift.git
cd gitshift

# Build the binary
go build -o gitshift

# Install system-wide (optional)
sudo mv gitshift /usr/local/bin/
```

#### **Verify Installation**
```bash
# Check version
gitshift --version

# Get help
gitshift --help
```

### **First-Time Setup**

#### **1. System Diagnostics**
```bash
# Run comprehensive system check
gitshift diagnose --verbose

# This will check:
# - Git installation and configuration
# - SSH setup and key availability
# - GitHub CLI availability
# - System permissions
```

#### **2. Add Your First Account**
```bash
# Add GitHub account with automated setup
gitshift add-github yourusername \
  --email "your@email.com" \
  --name "Your Full Name" \
  --description "Personal GitHub account"

# This will:
# - Fetch your GitHub profile information
# - Generate SSH keys if needed
# - Upload SSH keys to GitHub (if authenticated)
# - Configure Git settings
```

#### **3. Switch to Your Account**
```bash
# Switch to the account
gitshift switch yourusername

# Verify the switch
gitshift status
```

---

## 👤 **Account Management**

### **Adding Accounts**

#### **Basic Account Addition**
```bash
# Add account with minimal information
gitshift add-github username

# gitshift will prompt for missing information
```

#### **Complete Account Setup**
```bash
# Add account with all information
gitshift add-github username \
  --email "user@company.com" \
  --name "Full Name" \
  --description "Work account for Company Inc" \
  --ssh-key "/path/to/existing/key"
```

#### **Account from GitHub Username**
```bash
# Automatically fetch information from GitHub
gitshift add-github @username

# This will:
# - Fetch public profile information
# - Use GitHub username as alias
# - Prompt for email if not public
```

### **Listing Accounts**

```bash
# List all configured accounts
gitshift list

# Output example:
# 📋 Configured Accounts:
#
# 🟢 personal (active)
#    Name: John Doe
#    Email: john@personal.com
#    GitHub: @johndoe
#    SSH Key: ~/.ssh/id_ed25519_personal
#
# ⚪ work
#    Name: John Doe
#    Email: john@company.com
#    GitHub: @john-company
#    SSH Key: ~/.ssh/id_ed25519_work
```

### **Account Information**

```bash
# View current account status
gitshift status

# View specific account details
gitshift list --account work

# Get account configuration
gitshift config --account personal
```

### **Removing Accounts**

```bash
# Remove an account
gitshift remove oldaccount

# Remove with confirmation
gitshift remove oldaccount --confirm

# Remove and clean up SSH keys
gitshift remove oldaccount --cleanup-ssh
```

---

## 🔑 **SSH Key Management**

### **SSH Key Operations**

#### **List SSH Keys**
```bash
# List all SSH keys
gitshift ssh-keys list

# List keys for specific account
gitshift ssh-keys list --account work

# Output example:
# 🔑 SSH Keys:
#
# ✅ id_ed25519_personal
#    Path: ~/.ssh/id_ed25519_personal
#    Type: ED25519
#    Fingerprint: SHA256:abc123...
#    Account: personal
#
# ⚠️  id_rsa_work
#    Path: ~/.ssh/id_rsa_work
#    Type: RSA
#    Fingerprint: SHA256:def456...
#    Account: work (needs attention)
```

#### **Generate New SSH Keys**
```bash
# Generate key for specific account
# Automatically adds to ssh-agent and copies to clipboard
gitshift ssh-keygen work --email work@company.com

# Generate with custom settings
gitshift ssh-keygen work \
  --type ed25519 \
  --email "work@company.com"

# Key features:
# - Automatically adds the key to ssh-agent (no manual ssh-add needed)
# - Copies public key to clipboard using pbcopy
# - Displays public key in terminal for verification
```

#### **Test SSH Keys**
```bash
# Test SSH connection for account
gitshift ssh-keys test work

# Test specific key
gitshift ssh-keys test --key ~/.ssh/id_ed25519_work

# Test all keys
gitshift ssh-keys test --all
```

#### **Setup SSH Keys**
```bash
# Setup SSH key for account (generate + upload)
gitshift ssh-keys setup work

# Setup with custom email
gitshift ssh-keys setup work --email "work@company.com"
```

### **SSH Agent Management**

#### **SSH Agent Status**
```bash
# Check SSH agent status
gitshift ssh-agent --status

# Output example:
# 🔐 SSH Agent Status:
#
# ✅ Running (PID: 12345)
# 📍 Socket: /Users/user/.ssh/socket/agent.12345
# 🔑 Loaded Keys: 2
#    - id_ed25519_personal
#    - id_ed25519_work
```

#### **SSH Agent Operations**
```bash
# Clear all keys from agent
gitshift ssh-agent --clear

# Load specific key
gitshift ssh-agent --load ~/.ssh/id_ed25519_work

# Start SSH agent
gitshift ssh-agent --start

# Stop SSH agent
gitshift ssh-agent --stop
```

### **SSH Configuration**

#### **Validate SSH Configuration**
```bash
# Validate SSH setup
gitshift validate-ssh

# Validate with verbose output
gitshift validate-ssh --verbose

# Fix SSH issues automatically
gitshift validate-ssh --fix
```

#### **SSH Config Management**
```bash
# Generate SSH config
gitshift ssh-config generate

# Validate SSH config
gitshift ssh-config validate

# Show SSH config
gitshift ssh-config show
```

---

## 🔄 **Account Switching**

### **Basic Switching**

```bash
# Switch to account
gitshift switch work

# Switch with validation
gitshift switch work --validate

# Force switch (skip validation)
gitshift switch work --force

# Skip SSH validation for speed
gitshift switch work --skip-validation
```

### **What Happens During Switch**

When you run `gitshift switch work`, the following happens:

1. **Validation**: Checks if the target account exists and is valid
2. **SSH Validation**: Tests SSH connection to GitHub (unless skipped)
3. **Git Configuration**: Updates `user.name` and `user.email`
4. **SSH Key Management**: Loads the account's SSH key into the agent
5. **GITHUB_TOKEN Update**: Updates `GITHUB_TOKEN` in your `zsh_secrets` file
6. **Verification**: Confirms the switch was successful

### **Switch Options**

```bash
# Switch with detailed output
gitshift switch work --verbose

# Switch and show configuration
gitshift switch work --show-config

# Switch with confirmation prompt
gitshift switch work --confirm
```

### **Project-Specific Switching**

```bash
# Set up project-specific account
gitshift project set work

# This creates a .gitshift.yaml file in the current directory
# with the specified account configuration

# Switch to project account
gitshift switch --project

# Remove project configuration
gitshift project unset
```

---

## 🔍 **Diagnostics & Health Checks**

### **System Diagnostics**

#### **Full System Check**
```bash
# Comprehensive system diagnostics
gitshift diagnose

# Verbose output with detailed information
gitshift diagnose --verbose

# Include system information
gitshift diagnose --include-system
```

#### **Focused Diagnostics**
```bash
# Check only accounts
gitshift diagnose --accounts-only

# Check only SSH configuration
gitshift diagnose --ssh-only

# Check only Git configuration
gitshift diagnose --git-only

# Check only GitHub integration
gitshift diagnose --github-only
```

#### **Auto-Fix Issues**
```bash
# Automatically fix detected issues
gitshift diagnose --fix

# Fix with confirmation prompts
gitshift diagnose --fix --confirm

# Fix specific categories
gitshift diagnose --fix --ssh-only
```

### **Health Check Categories**

| Category | Description | Auto-Fix Available |
|----------|-------------|-------------------|
| **🏥 System Health** | Git, SSH, GitHub CLI availability | ❌ |
| **👤 Account Config** | Email, name, SSH key validation | ✅ |
| **🔐 SSH Issues** | Key permissions, conflicts, authentication | ✅ |
| **⚙️ Git Config** | User settings, SSH commands, remotes | ✅ |
| **🔗 GitHub Integration** | API access, repository permissions | ❌ |

### **Example Diagnostic Output**

```bash
$ gitshift diagnose --verbose

🟢 Overall Health: EXCELLENT

📊 Summary:
  • Issues: 0
  • Warnings: 2
  • Accounts: 2 configured
  • SSH Keys: 3 available
  • Git Config: ✅ Valid

👤 Account Status:
  • personal: ✅ Valid
    - Email: john@personal.com
    - SSH Key: ~/.ssh/id_ed25519_personal
    - GitHub: @johndoe
  • work: ⚠️  SSH key needs attention
    - Email: john@company.com
    - SSH Key: ~/.ssh/id_rsa_work (permission issues)
    - GitHub: @john-company

🔐 SSH Status:
  ✅ SSH Agent: Running
  ✅ SSH Config: Valid
  ⚠️  Multiple keys loaded (3 keys)
  ⚠️  Key isolation: Disabled

⚙️ Git Configuration:
  ✅ Global config: Valid
  ✅ Current account: personal
  ✅ SSH command: Configured

⚠️ Warnings:
  • ssh: Multiple SSH keys loaded (3 keys)
    Recommendation: Use only one key at a time to avoid conflicts
  • system: GitHub CLI not found
    Recommendation: Install GitHub CLI for enhanced integration

💡 Run 'gitshift diagnose --fix' to automatically resolve fixable issues
```

---

## ⚙️ **Configuration**

### **Configuration File**

gitshift stores configuration in `~/.config/gitshift/config.yaml`:

```yaml
# gitshift Configuration
accounts:
  personal:
    alias: personal
    name: John Doe
    email: john@personal.com
    ssh_key_path: /Users/john/.ssh/id_ed25519_personal
    github_username: johndoe
    description: Personal GitHub account
    is_default: true
    status: active
    created_at: "2025-01-15T10:30:00Z"
    last_used: "2025-01-16T09:15:00Z"

  work:
    alias: work
    name: John Doe
    email: john@company.com
    ssh_key_path: /Users/john/.ssh/id_ed25519_work
    github_username: john-company
    description: Work account for Company Inc
    is_default: false
    status: active
    created_at: "2025-01-15T11:00:00Z"
    last_used: "2025-01-16T08:45:00Z"

# Current active account
current_account: personal

# Global settings
global_git_config: true
auto_detect: true
config_version: "1.0.0"
```

### **Configuration Management**

#### **View Configuration**
```bash
# Show current configuration
gitshift config show

# Show configuration for specific account
gitshift config show --account work

# Show configuration in different formats
gitshift config show --format yaml
gitshift config show --format json
```

#### **Edit Configuration**
```bash
# Edit configuration file
gitshift config edit

# Edit specific account
gitshift config edit --account work

# Set configuration values
gitshift config set global_git_config false
gitshift config set auto_detect true
```

### **Environment Variables**

```bash
# Configuration file location
export gitshift_CONFIG_PATH="~/.config/gitshift"

# Enable debug logging
export gitshift_DEBUG=true

# Default SSH key directory
export gitshift_SSH_DIR="~/.ssh"

# GitHub CLI path (if not in PATH)
export GITHUB_CLI_PATH="/usr/local/bin/gh"
```

---

## 🆕 **Advanced Features**

### **Zsh Secrets Integration**

gitshift automatically manages your `GITHUB_TOKEN` in your `zsh_secrets` file:

#### **Supported Locations**
- `~/.zsh_secrets` (default)
- `~/.config/zsh_secrets`
- `~/.secrets/zsh_secrets`
- `~/.zsh/secrets`

#### **How It Works**
```bash
# When you switch accounts
gitshift switch work

# gitshift automatically:
# 1. Gets the current GitHub token from 'gh auth token'
# 2. Updates the GITHUB_TOKEN in your zsh_secrets file
# 3. Optionally reloads the zsh_secrets file
```

#### **Manual Token Management**
```bash
# Update token manually
gitshift secrets update-token

# Get current token
gitshift secrets get-token

# Validate zsh_secrets file
gitshift secrets validate
```

### **Project-Specific Configuration**

#### **Project Configuration File**
Create a `.gitshift.yaml` file in your project directory:

```yaml
# .gitshift.yaml
account: work
description: "Work project - use work account"
created_at: "2025-01-16T10:00:00Z"
```

#### **Project Commands**
```bash
# Set project account
gitshift project set work

# Switch to project account
gitshift switch --project

# Remove project configuration
gitshift project unset

# List project configurations
gitshift project list
```

### **Automated GitHub Integration**

#### **SSH Key Upload**
```bash
# Add account with automatic SSH key upload
gitshift add-github username --upload-ssh

# Upload existing SSH key to GitHub
gitshift ssh-keys upload --key ~/.ssh/id_ed25519_work
```

#### **Repository Management**
```bash
# List repositories for current account
gitshift repos list

# Clone repository with correct account
gitshift repos clone owner/repo

# Set up repository with project account
gitshift repos setup owner/repo --account work
```

---

## 🎯 **Best Practices**

### **Account Organization**

#### **Naming Conventions**
```bash
# Use descriptive aliases
gitshift add-github john-doe --alias personal
gitshift add-github john-company --alias work
gitshift add-github john-client --alias client-project

# Use consistent email patterns
# Personal: yourname@gmail.com
# Work: yourname@company.com
# Client: yourname@client.com
```

#### **SSH Key Management**
```bash
# Use Ed25519 keys (recommended)
gitshift ssh-keys generate work --type ed25519

# Use descriptive key names
# ~/.ssh/id_ed25519_personal
# ~/.ssh/id_ed25519_work
# ~/.ssh/id_ed25519_client

# Keep keys organized
mkdir -p ~/.ssh/keys
# Store keys in ~/.ssh/keys/ directory
```

### **Workflow Integration**

#### **Shell Integration**
Add to your `~/.zshrc` or `~/.bashrc`:

```bash
# Auto-switch based on directory
cd() {
    builtin cd "$@"
    if [[ -f .gitshift.yaml ]]; then
        gitshift switch --project
    fi
}

# Show current account in prompt
export PS1='$(gitshift status --short) $PS1'
```

#### **Git Hooks**
Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Ensure correct account is active
gitshift switch --project
```

### **Security Best Practices**

#### **SSH Key Security**
```bash
# Use strong passphrases
gitshift ssh-keys generate work --passphrase

# Regular key rotation
gitshift ssh-keys rotate work

# Monitor key usage
gitshift ssh-keys audit
```

#### **Token Management**
```bash
# Use GitHub CLI for token management
gh auth login

# Regularly refresh tokens
gitshift secrets refresh-token

# Monitor token usage
gitshift secrets audit
```

---

## 🚨 **Troubleshooting**

### **Common Issues**

#### **SSH Authentication Failures**

**Problem**: `Permission denied (publickey)` when connecting to GitHub

**Solutions**:
```bash
# 1. Diagnose SSH issues
gitshift diagnose --ssh-only --verbose

# 2. Test SSH connection manually
ssh -T git@github.com -i ~/.ssh/id_ed25519_work

# 3. Check SSH key permissions
ls -la ~/.ssh/id_ed25519_work
# Should be: -rw------- (600)

# 4. Fix permissions
chmod 600 ~/.ssh/id_ed25519_work

# 5. Verify key is added to GitHub
gitshift ssh-keys test work
```

#### **Account Switch Failures**

**Problem**: Account switch fails with validation errors

**Solutions**:
```bash
# 1. Force switch with detailed output
gitshift switch work --force --verbose

# 2. Validate current configuration
gitshift diagnose --accounts-only

# 3. Reset SSH agent state
gitshift ssh-agent --clear
gitshift switch work

# 4. Check Git configuration
git config --global --list
```

#### **SSH Socket Directory Issues**

**Problem**: `unix_listener: cannot bind to path` errors

**Solutions**:
```bash
# 1. Create SSH socket directories
mkdir -p ~/.ssh/socket ~/.ssh/sockets ~/.ssh/control
chmod 700 ~/.ssh/socket ~/.ssh/sockets ~/.ssh/control

# 2. Let gitshift fix automatically
gitshift diagnose --fix

# 3. Verify directories exist
ls -la ~/.ssh/socket/
```

#### **GitHub Token Issues**

**Problem**: GITHUB_TOKEN not updating in zsh_secrets

**Solutions**:
```bash
# 1. Check GitHub CLI authentication
gh auth status

# 2. Re-authenticate if needed
gh auth login

# 3. Test token retrieval
gh auth token

# 4. Validate zsh_secrets file
gitshift secrets validate

# 5. Manually update token
gitshift secrets update-token
```

### **Getting Help**

#### **Command Help**
```bash
# General help
gitshift --help

# Command-specific help
gitshift diagnose --help
gitshift switch --help
gitshift ssh-keys --help
```

#### **Debug Mode**
```bash
# Enable debug logging
export gitshift_DEBUG=true

# Run command with verbose output
gitshift switch work --verbose

# Check logs
tail -f ~/.config/gitshift/logs/gitshift.log
```

#### **System Information**
```bash
# Get system information for bug reports
gitshift diagnose --include-system > system-info.txt

# Include in bug reports
cat system-info.txt
```

---

## 📚 **Additional Resources**

- **[Configuration Guide](CONFIGURATION.md)** - Detailed configuration options
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Comprehensive troubleshooting
- **[Architecture Guide](ARCHITECTURE.md)** - Technical architecture details
- **[Security Guide](SECURITY.md)** - Security best practices
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute

---

<div align="center">

**Need help?** Check our [Issues](https://github.com/techishthoughts-org/gitshift/issues)!

</div>
