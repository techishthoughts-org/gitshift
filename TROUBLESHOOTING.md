# GitPersona Troubleshooting Guide

## 🚨 Quick Diagnostics

Before diving into specific issues, run the comprehensive diagnostic tool:

```bash
# Run full system diagnostics
gitpersona diagnose --verbose

# Auto-fix common issues
gitpersona diagnose --fix

# Focus on specific areas
gitpersona diagnose --ssh-only
gitpersona diagnose --accounts-only
gitpersona diagnose --git-only
```

---

## 🔑 SSH Authentication Issues

### Problem: `Permission denied (publickey)` when pushing to GitHub

**Symptoms:**
```
git push origin main
git@github.com: Permission denied (publickey).
fatal: Could not read from remote repository.
```

**Diagnosis:**
```bash
# Check current SSH configuration
gitpersona diagnose --ssh-only --verbose

# Test SSH connection manually
ssh -T git@github.com

# Check which key is being used
gitpersona ssh-agent --status
```

**Solutions:**

1. **Wrong SSH Key Loaded**
   ```bash
   # Clear all keys and load correct one
   gitpersona ssh-agent --clear
   gitpersona switch your-account

   # Verify correct key is loaded
   ssh -T git@github.com
   ```

2. **SSH Key Not on GitHub**
   ```bash
   # Check if key exists on GitHub
   gh ssh-key list

   # Add key to GitHub automatically
   gitpersona add-github your-account --automated-ssh
   ```

3. **Key Permissions Issues**
   ```bash
   # Fix key permissions
   chmod 600 ~/.ssh/id_ed25519_*
   chmod 644 ~/.ssh/id_ed25519_*.pub
   chmod 700 ~/.ssh
   ```

### Problem: Multiple SSH Keys Conflict

**Symptoms:**
- Authenticating as wrong GitHub user
- `git push` works but commits show wrong author

**Diagnosis:**
```bash
# Check loaded keys
gitpersona ssh-agent --status

# Test which account each key authenticates as
for key in ~/.ssh/id_ed25519_*; do
    echo "Testing $key:"
    ssh -i "$key" -T git@github.com
done
```

**Solutions:**

1. **Use Account Isolation**
   ```bash
   # Switch to account (isolates SSH keys)
   gitpersona switch work

   # Verify isolation
   gitpersona ssh-agent --status
   ```

2. **Clear Conflicting Keys**
   ```bash
   # Remove all keys from agent
   gitpersona ssh-agent --clear

   # Load only the key you need
   gitpersona ssh-agent --load ~/.ssh/id_ed25519_work
   ```

---

## ⚙️ Git Configuration Issues

### Problem: Commits Show Wrong Author Name/Email

**Symptoms:**
```bash
git log --oneline -1
abc1234 (HEAD -> main) Fix bug [wrong-email@example.com]
```

**Diagnosis:**
```bash
# Check current Git configuration
git config --global --list | grep user
git config --local --list | grep user

# Validate Git config
gitpersona validate-git --verbose
```

**Solutions:**

1. **Switch to Correct Account**
   ```bash
   gitpersona switch correct-account

   # Verify configuration was updated
   git config --global user.name
   git config --global user.email
   ```

2. **Manual Configuration Fix**
   ```bash
   # Set correct user info
   git config --global user.name "Your Name"
   git config --global user.email "your@email.com"

   # For project-specific config
   git config --local user.name "Work Name"
   git config --local user.email "work@company.com"
   ```

### Problem: SSH Command Not Set Correctly

**Symptoms:**
- SSH authentication fails after account switch
- Git uses default SSH key instead of account-specific key

**Diagnosis:**
```bash
# Check SSH command configuration
git config --global core.sshcommand
git config --local core.sshcommand
```

**Solutions:**

1. **Automatic Fix**
   ```bash
   gitpersona switch your-account --force

   # Or run diagnostics with fix
   gitpersona diagnose --git-only --fix
   ```

2. **Manual SSH Command Setup**
   ```bash
   git config --global core.sshcommand "ssh -i ~/.ssh/id_ed25519_work -o IdentitiesOnly=yes"
   ```

---

## 📁 Account Management Issues

### Problem: Account Not Found

**Symptoms:**
```
Error: Account 'work' not found
```

**Diagnosis:**
```bash
# List all configured accounts
gitpersona list

# Check configuration file
cat ~/.config/gitpersona/config.yaml
```

**Solutions:**

1. **Add Missing Account**
   ```bash
   # Add account with full setup
   gitpersona add-github work --email "work@company.com" --name "Work Name"
   ```

2. **Discover Existing Accounts**
   ```bash
   # Auto-discover accounts from Git configs
   gitpersona discover --auto-add
   ```

### Problem: Invalid Account Configuration

**Symptoms:**
```
Error: Account validation failed: missing required field 'email'
```

**Diagnosis:**
```bash
# Validate all accounts
gitpersona diagnose --accounts-only

# Check specific account
gitpersona status work
```

**Solutions:**

1. **Complete Missing Information**
   ```bash
   # Update account with missing fields
   gitpersona update work --email "work@company.com"
   ```

2. **Recreate Account**
   ```bash
   # Remove and recreate
   gitpersona remove work
   gitpersona add-github work --email "work@company.com" --name "Work Name"
   ```

---

## 🔧 System Issues

### Problem: GitPersona Commands Not Working

**Symptoms:**
```
command not found: gitpersona
```

**Solutions:**

1. **Installation Check**
   ```bash
   # Check if binary exists
   ls -la /usr/local/bin/gitpersona

   # Reinstall if needed
   go build -o gitpersona
   sudo mv gitpersona /usr/local/bin/
   ```

2. **Path Issues**
   ```bash
   # Add to PATH temporarily
   export PATH=$PATH:/usr/local/bin

   # Add to shell profile permanently
   echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
   source ~/.bashrc
   ```

### Problem: Configuration File Issues

**Symptoms:**
```
Error: Failed to load configuration: yaml: unmarshal errors
```

**Diagnosis:**
```bash
# Check config file exists and is readable
ls -la ~/.config/gitpersona/config.yaml

# Validate YAML syntax
cat ~/.config/gitpersona/config.yaml | yaml-validator
```

**Solutions:**

1. **Backup and Reset Configuration**
   ```bash
   # Backup current config
   cp ~/.config/gitpersona/config.yaml ~/.config/gitpersona/config.yaml.backup

   # Reset to default
   rm ~/.config/gitpersona/config.yaml
   gitpersona discover
   ```

2. **Manual Configuration Fix**
   ```bash
   # Edit configuration file
   nano ~/.config/gitpersona/config.yaml

   # Example valid configuration:
   ```
   ```yaml
   accounts:
     personal:
       alias: personal
       name: John Doe
       email: john@personal.com
       ssh_key_path: /Users/john/.ssh/id_ed25519_personal
       github_username: johndoe
   current_account: personal
   global_git_config: true
   auto_detect: true
   config_version: "1.0.0"
   ```

---

## 🌐 GitHub Integration Issues

### Problem: GitHub CLI Not Working

**Symptoms:**
```
Error: GitHub CLI not found
```

**Solutions:**

1. **Install GitHub CLI**
   ```bash
   # macOS
   brew install gh

   # Ubuntu/Debian
   sudo apt install gh

   # Or download from https://cli.github.com/
   ```

2. **Authenticate GitHub CLI**
   ```bash
   gh auth login --with-token < your-token.txt
   # Or interactive login
   gh auth login
   ```

### Problem: API Rate Limiting

**Symptoms:**
```
Error: API rate limit exceeded
```

**Solutions:**

1. **Use Personal Access Token**
   ```bash
   # Create token at https://github.com/settings/tokens
   # Then authenticate
   gh auth login --with-token < token.txt
   ```

2. **Wait and Retry**
   ```bash
   # Check rate limit status
   gh api rate_limit

   # Wait for reset time or use different authentication
   ```

---

## 🔍 Advanced Debugging

### Enable Debug Mode

```bash
# Set debug environment variable
export GITPERSONA_DEBUG=true

# Run commands with verbose output
gitpersona switch work --verbose

# Check logs
tail -f ~/.config/gitpersona/logs/gitpersona.log
```

### Manual SSH Testing

```bash
# Test SSH connection with specific key
ssh -i ~/.ssh/id_ed25519_work -T git@github.com

# Debug SSH connection
ssh -i ~/.ssh/id_ed25519_work -T -v git@github.com

# Check SSH agent
ssh-add -l
```

### Git Configuration Debugging

```bash
# Show all Git configuration
git config --list --show-origin

# Test Git operations
git ls-remote origin

# Check remote URLs
git remote -v
```

---

## 📞 Getting Help

### Built-in Help

```bash
# General help
gitpersona --help

# Command-specific help
gitpersona switch --help
gitpersona diagnose --help

# Show version and system info
gitpersona --version
```

### Self-Diagnostics Checklist

Before reporting issues, run through this checklist:

1. **System Health**
   ```bash
   gitpersona diagnose --include-system
   ```

2. **Account Status**
   ```bash
   gitpersona list
   gitpersona status
   ```

3. **SSH Connectivity**
   ```bash
   ssh -T git@github.com
   gitpersona ssh-agent --status
   ```

4. **Git Configuration**
   ```bash
   git config --global --list
   gitpersona validate-git
   ```

5. **GitHub Integration**
   ```bash
   gh auth status
   gh ssh-key list
   ```

### Reporting Issues

When reporting issues, include:

1. **System Information**
   ```bash
   gitpersona --version
   git --version
   gh --version
   uname -a
   ```

2. **Configuration Export**
   ```bash
   # Export configuration (remove sensitive data)
   cat ~/.config/gitpersona/config.yaml
   ```

3. **Debug Output**
   ```bash
   GITPERSONA_DEBUG=true gitpersona your-failing-command --verbose
   ```

4. **Steps to Reproduce**
   - Exact commands that fail
   - Expected vs actual behavior
   - Error messages (full output)

---

## 🚀 Performance Optimization

### Faster Account Switching

```bash
# Skip validation for faster switching
gitpersona switch work --skip-validation

# Use force to bypass checks
gitpersona switch work --force
```

### Reduce SSH Agent Overhead

```bash
# Clear unused keys periodically
gitpersona ssh-agent --clear

# Only load keys when needed
gitpersona ssh-agent --load ~/.ssh/specific_key
```

### Configuration Optimization

```yaml
# In ~/.config/gitpersona/config.yaml
global_git_config: true  # Faster than local configs
auto_detect: false       # Skip auto-detection if not needed
```

This troubleshooting guide covers the most common issues and their solutions. For additional help, use the built-in diagnostic tools and refer to the project documentation.
