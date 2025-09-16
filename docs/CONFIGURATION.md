# ⚙️ GitPersona Configuration Guide

> **Complete guide to configuring GitPersona for optimal GitHub account management**

---

## 📖 **Table of Contents**

1. [Configuration File Structure](#configuration-file-structure)
2. [Account Configuration](#account-configuration)
3. [Global Settings](#global-settings)
4. [Environment Variables](#environment-variables)
5. [SSH Configuration](#ssh-configuration)
6. [Zsh Secrets Integration](#zsh-secrets-integration)
7. [Project-Specific Configuration](#project-specific-configuration)
8. [Configuration Management](#configuration-management)
9. [Advanced Configuration](#advanced-configuration)

---

## 📁 **Configuration File Structure**

GitPersona stores its configuration in `~/.config/gitpersona/config.yaml`:

```yaml
# GitPersona Configuration File
# Version: 1.0.0
# Last Updated: 2025-01-16T10:30:00Z

# Account configurations
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
    missing_fields: []

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
    missing_fields: []

# Pending accounts (incomplete setup)
pending_accounts:
  client-project:
    alias: client-project
    github_username: john-client
    partial_data:
      name: "John Doe"
      email: "john@client.com"
    missing_fields: ["ssh_key_path"]
    source: "discovery"
    confidence: 85
    created_at: "2025-01-16T09:00:00Z"

# Current active account
current_account: personal

# Global settings
global_git_config: true
auto_detect: true
config_version: "1.0.0"
```

---

## 👤 **Account Configuration**

### **Account Fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `alias` | string | ✅ | Unique identifier for the account |
| `name` | string | ✅ | Git user.name |
| `email` | string | ✅ | Git user.email (must be valid email) |
| `ssh_key_path` | string | ❌ | Path to SSH private key file |
| `github_username` | string | ✅ | GitHub username |
| `description` | string | ❌ | Human-readable description |
| `is_default` | boolean | ❌ | Whether this is the default account |
| `status` | string | ❌ | Account status (active, pending, disabled) |
| `created_at` | timestamp | ❌ | When the account was created |
| `last_used` | timestamp | ❌ | When the account was last used |
| `missing_fields` | array | ❌ | List of missing required fields |

### **Account Status Values**

- `active`: Account is fully configured and ready to use
- `pending`: Account needs additional configuration
- `disabled`: Account is temporarily disabled

### **Example Account Configurations**

#### **Personal Account**
```yaml
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
```

#### **Work Account**
```yaml
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
```

#### **Client Account**
```yaml
client-project:
  alias: client-project
  name: John Doe
  email: john@client.com
  ssh_key_path: /Users/john/.ssh/id_ed25519_client
  github_username: john-client
  description: Client project account
  is_default: false
  status: active
  created_at: "2025-01-15T12:00:00Z"
  last_used: "2025-01-16T07:30:00Z"
```

---

## 🌐 **Global Settings**

### **Global Configuration Options**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `current_account` | string | `""` | Currently active account alias |
| `global_git_config` | boolean | `true` | Use global Git configuration |
| `auto_detect` | boolean | `true` | Enable automatic account detection |
| `config_version` | string | `"1.0.0"` | Configuration file version |

### **Global Settings Explained**

#### **global_git_config**
```yaml
global_git_config: true  # Use global Git configuration
global_git_config: false # Use local Git configuration only
```

**When `true`**:
- Git configuration is set globally (`git config --global`)
- All repositories use the same account settings
- Faster switching between repositories

**When `false`**:
- Git configuration is set locally per repository
- Each repository can have different account settings
- More granular control but requires manual setup

#### **auto_detect**
```yaml
auto_detect: true  # Enable automatic account detection
auto_detect: false # Disable automatic account detection
```

**When `true`**:
- GitPersona automatically detects account based on repository
- Uses `.gitpersona.yaml` files in project directories
- Provides seamless workflow integration

**When `false`**:
- Manual account switching required
- No automatic detection or switching
- More predictable but less automated

---

## 🔧 **Environment Variables**

### **Core Environment Variables**

| Variable | Default | Description |
|----------|---------|-------------|
| `GITPERSONA_CONFIG_PATH` | `~/.config/gitpersona` | Configuration directory path |
| `GITPERSONA_DEBUG` | `false` | Enable debug logging |
| `GITPERSONA_SSH_DIR` | `~/.ssh` | SSH keys directory |
| `GITPERSONA_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

### **GitHub Integration Variables**

| Variable | Default | Description |
|----------|---------|-------------|
| `GITHUB_CLI_PATH` | `gh` | Path to GitHub CLI executable |
| `GITHUB_TOKEN` | `""` | GitHub API token (managed by zsh_secrets) |
| `GITHUB_API_URL` | `https://api.github.com` | GitHub API base URL |

### **SSH Configuration Variables**

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_AUTH_SOCK` | `""` | SSH agent socket path |
| `SSH_CONFIG_FILE` | `~/.ssh/config` | SSH configuration file path |
| `SSH_KEY_DIR` | `~/.ssh` | SSH keys directory |

### **Setting Environment Variables**

#### **Temporary (Current Session)**
```bash
# Enable debug logging
export GITPERSONA_DEBUG=true

# Set custom config path
export GITPERSONA_CONFIG_PATH="/custom/path"

# Set custom SSH directory
export GITPERSONA_SSH_DIR="/custom/ssh"
```

#### **Permanent (Shell Profile)**
Add to `~/.zshrc`, `~/.bashrc`, or `~/.profile`:

```bash
# GitPersona Configuration
export GITPERSONA_CONFIG_PATH="$HOME/.config/gitpersona"
export GITPERSONA_DEBUG=false
export GITPERSONA_SSH_DIR="$HOME/.ssh"
export GITPERSONA_LOG_LEVEL="info"

# GitHub CLI Configuration
export GITHUB_CLI_PATH="gh"
export GITHUB_API_URL="https://api.github.com"
```

#### **Project-Specific (Environment File)**
Create `.env` file in project directory:

```bash
# .env
GITPERSONA_DEBUG=true
GITPERSONA_LOG_LEVEL=debug
GITHUB_TOKEN=ghp_your_token_here
```

---

## 🔐 **SSH Configuration**

### **SSH Key Configuration**

#### **SSH Key Paths**
```yaml
# Recommended SSH key naming convention
ssh_key_path: "/Users/john/.ssh/id_ed25519_personal"
ssh_key_path: "/Users/john/.ssh/id_ed25519_work"
ssh_key_path: "/Users/john/.ssh/id_ed25519_client"
```

#### **SSH Key Types**
```yaml
# Ed25519 (recommended)
ssh_key_path: "/Users/john/.ssh/id_ed25519_personal"

# RSA (legacy)
ssh_key_path: "/Users/john/.ssh/id_rsa_work"

# ECDSA
ssh_key_path: "/Users/john/.ssh/id_ecdsa_client"
```

### **SSH Config Integration**

GitPersona can generate SSH configuration entries:

```bash
# Generate SSH config
gitpersona ssh-config generate
```

**Generated SSH Config Example**:
```bash
# SSH Configuration for GitPersona
# Generated automatically - do not edit manually

Host github-personal
    HostName github.com
    User git
    IdentityFile /Users/john/.ssh/id_ed25519_personal
    IdentitiesOnly yes
    UseKeychain yes
    AddKeysToAgent yes

Host github-work
    HostName github.com
    User git
    IdentityFile /Users/john/.ssh/id_ed25519_work
    IdentitiesOnly yes
    UseKeychain yes
    AddKeysToAgent yes
```

### **SSH Agent Configuration**

#### **SSH Agent Settings**
```yaml
# SSH Agent configuration
ssh_agent:
  auto_start: true
  socket_path: "~/.ssh/socket"
  key_lifetime: "3600"  # seconds
  max_keys: 10
```

#### **SSH Socket Directories**
```bash
# GitPersona creates these directories automatically
~/.ssh/socket/     # SSH agent sockets
~/.ssh/sockets/    # Additional socket storage
~/.ssh/control/    # SSH control connections
```

---

## 🔒 **Zsh Secrets Integration**

### **Zsh Secrets Configuration**

GitPersona automatically manages `GITHUB_TOKEN` in your `zsh_secrets` file:

#### **Supported File Locations**
```yaml
# Priority order (first found is used)
zsh_secrets_locations:
  - "~/.zsh_secrets"
  - "~/.config/zsh_secrets"
  - "~/.secrets/zsh_secrets"
  - "~/.zsh/secrets"
```

#### **Zsh Secrets File Format**
```bash
# ~/.zsh_secrets
# GitPersona managed file - do not edit manually

# GitHub Token (updated automatically)
export GITHUB_TOKEN="ghp_your_token_here"

# Other secrets
export API_KEY="your_api_key"
export DATABASE_URL="your_database_url"
```

### **Zsh Secrets Management**

#### **Configuration Options**
```yaml
# Zsh secrets configuration
zsh_secrets:
  auto_update: true
  auto_reload: true
  backup_enabled: true
  backup_count: 5
  file_permissions: "0600"
```

#### **Manual Management**
```bash
# Update token manually
gitpersona secrets update-token

# Get current token
gitpersona secrets get-token

# Validate zsh_secrets file
gitpersona secrets validate

# Backup zsh_secrets file
gitpersona secrets backup

# Restore from backup
gitpersona secrets restore --backup 1
```

---

## 📁 **Project-Specific Configuration**

### **Project Configuration File**

Create `.gitpersona.yaml` in your project directory:

```yaml
# .gitpersona.yaml
account: work
description: "Work project - use work account"
created_at: "2025-01-16T10:00:00Z"
auto_switch: true
git_config:
  user.name: "John Doe"
  user.email: "john@company.com"
ssh_config:
  host: "github-work"
  identity_file: "/Users/john/.ssh/id_ed25519_work"
```

### **Project Configuration Fields**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account` | string | ✅ | Account alias to use |
| `description` | string | ❌ | Project description |
| `created_at` | timestamp | ❌ | When config was created |
| `auto_switch` | boolean | ❌ | Auto-switch when entering directory |
| `git_config` | object | ❌ | Override Git configuration |
| `ssh_config` | object | ❌ | Override SSH configuration |

### **Project Configuration Examples**

#### **Basic Project Config**
```yaml
# .gitpersona.yaml
account: work
description: "Company project"
```

#### **Advanced Project Config**
```yaml
# .gitpersona.yaml
account: client-project
description: "Client project with specific requirements"
auto_switch: true
git_config:
  user.name: "John Doe"
  user.email: "john@client.com"
  core.editor: "code"
  init.defaultBranch: "main"
ssh_config:
  host: "github-client"
  identity_file: "/Users/john/.ssh/id_ed25519_client"
  port: 22
  compression: true
```

---

## 🛠️ **Configuration Management**

### **Configuration Commands**

#### **View Configuration**
```bash
# Show current configuration
gitpersona config show

# Show specific account
gitpersona config show --account work

# Show in different formats
gitpersona config show --format yaml
gitpersona config show --format json
gitpersona config show --format toml
```

#### **Edit Configuration**
```bash
# Edit configuration file
gitpersona config edit

# Edit specific account
gitpersona config edit --account work

# Edit with specific editor
gitpersona config edit --editor vim
```

#### **Set Configuration Values**
```bash
# Set global settings
gitpersona config set global_git_config false
gitpersona config set auto_detect true

# Set account-specific settings
gitpersona config set --account work email "new@email.com"
gitpersona config set --account work ssh_key_path "/new/path"
```

#### **Validate Configuration**
```bash
# Validate entire configuration
gitpersona config validate

# Validate specific account
gitpersona config validate --account work

# Fix configuration issues
gitpersona config validate --fix
```

### **Configuration Backup and Restore**

#### **Backup Configuration**
```bash
# Create backup
gitpersona config backup

# Create backup with description
gitpersona config backup --description "Before major changes"

# List backups
gitpersona config backup --list
```

#### **Restore Configuration**
```bash
# Restore from latest backup
gitpersona config restore

# Restore from specific backup
gitpersona config restore --backup 1

# Restore with confirmation
gitpersona config restore --confirm
```

---

## 🔧 **Advanced Configuration**

### **Custom SSH Configuration**

#### **SSH Config Templates**
```yaml
# Custom SSH config template
ssh_config_template: |
  Host github-{{.Alias}}
      HostName github.com
      User git
      IdentityFile {{.SSHKeyPath}}
      IdentitiesOnly yes
      UseKeychain yes
      AddKeysToAgent yes
      ServerAliveInterval 60
      ServerAliveCountMax 3
```

#### **SSH Key Generation Settings**
```yaml
# SSH key generation configuration
ssh_key_generation:
  default_type: "ed25519"
  default_size: 256
  default_comment: "{{.Email}}"
  key_directory: "~/.ssh"
  naming_pattern: "id_{{.Type}}_{{.Alias}}"
  permissions: "0600"
```

### **GitHub Integration Settings**

#### **GitHub API Configuration**
```yaml
# GitHub API settings
github:
  api_url: "https://api.github.com"
  timeout: 30
  retry_count: 3
  retry_delay: 1
  rate_limit: 5000
```

#### **GitHub CLI Integration**
```yaml
# GitHub CLI settings
github_cli:
  enabled: true
  path: "gh"
  auth_method: "oauth"
  scopes: ["repo", "read:user", "user:email", "admin:public_key"]
```

### **Logging Configuration**

#### **Log Settings**
```yaml
# Logging configuration
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "file"  # file, stdout, stderr
  file_path: "~/.config/gitpersona/logs/gitpersona.log"
  max_size: "10MB"
  max_backups: 5
  max_age: 30
  compress: true
```

#### **Debug Configuration**
```yaml
# Debug settings
debug:
  enabled: false
  verbose_output: false
  performance_metrics: false
  memory_profiling: false
  cpu_profiling: false
```

### **Performance Configuration**

#### **Performance Settings**
```yaml
# Performance configuration
performance:
  cache_enabled: true
  cache_ttl: 300  # seconds
  parallel_operations: 4
  timeout: 30  # seconds
  retry_attempts: 3
  retry_delay: 1  # seconds
```

#### **Validation Settings**
```yaml
# Validation configuration
validation:
  ssh_timeout: 10  # seconds
  github_timeout: 15  # seconds
  git_timeout: 5  # seconds
  retry_count: 3
  parallel_validation: true
```

---

## 🔍 **Configuration Validation**

### **Validation Rules**

#### **Account Validation**
```yaml
# Account validation rules
account_validation:
  required_fields: ["alias", "name", "email", "github_username"]
  email_format: true
  github_username_format: true
  ssh_key_exists: true
  ssh_key_permissions: "0600"
  unique_alias: true
  unique_email: false
  unique_github_username: true
```

#### **SSH Validation**
```yaml
# SSH validation rules
ssh_validation:
  key_exists: true
  key_readable: true
  key_permissions: "0600"
  key_format: true
  github_authentication: true
  agent_connection: true
```

#### **Git Validation**
```yaml
# Git validation rules
git_validation:
  git_installed: true
  config_valid: true
  user_name_set: true
  user_email_set: true
  ssh_command_set: true
```

### **Validation Commands**

```bash
# Validate entire configuration
gitpersona config validate

# Validate specific components
gitpersona config validate --accounts
gitpersona config validate --ssh
gitpersona config validate --git

# Validate with fixes
gitpersona config validate --fix

# Validate with detailed output
gitpersona config validate --verbose
```

---

## 📚 **Configuration Examples**

### **Complete Configuration Example**

```yaml
# Complete GitPersona Configuration
# This is a comprehensive example showing all available options

# Account configurations
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

# Advanced configuration
ssh_config:
  auto_generate: true
  template: |
    Host github-{{.Alias}}
        HostName github.com
        User git
        IdentityFile {{.SSHKeyPath}}
        IdentitiesOnly yes
        UseKeychain yes

zsh_secrets:
  auto_update: true
  auto_reload: true
  backup_enabled: true
  file_permissions: "0600"

logging:
  level: "info"
  format: "json"
  file_path: "~/.config/gitpersona/logs/gitpersona.log"

performance:
  cache_enabled: true
  cache_ttl: 300
  parallel_operations: 4
  timeout: 30
```

---

## 🚨 **Configuration Troubleshooting**

### **Common Configuration Issues**

#### **Invalid YAML Syntax**
```bash
# Check YAML syntax
gitpersona config validate

# Fix YAML syntax errors
gitpersona config validate --fix
```

#### **Missing Required Fields**
```bash
# Check for missing fields
gitpersona config validate --accounts

# Fix missing fields
gitpersona config validate --fix
```

#### **Invalid File Permissions**
```bash
# Check file permissions
ls -la ~/.config/gitpersona/config.yaml

# Fix permissions
chmod 600 ~/.config/gitpersona/config.yaml
```

### **Configuration Recovery**

#### **Reset to Defaults**
```bash
# Reset configuration to defaults
gitpersona config reset

# Reset specific account
gitpersona config reset --account work
```

#### **Restore from Backup**
```bash
# List available backups
gitpersona config backup --list

# Restore from backup
gitpersona config restore --backup 1
```

---

<div align="center">

**Need help with configuration?** Check our [Troubleshooting Guide](TROUBLESHOOTING.md) or [User Guide](USER_GUIDE.md)!

</div>
