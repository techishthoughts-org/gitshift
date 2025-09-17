# 🚨 GitPersona Troubleshooting Guide

> **Comprehensive guide to diagnosing and fixing common GitPersona issues**

---

## 📖 **Table of Contents**

1. [Quick Diagnostics](#quick-diagnostics)
2. [Common Issues](#common-issues)
3. [SSH Issues](#ssh-issues)
4. [Git Configuration Issues](#git-configuration-issues)
5. [Account Management Issues](#account-management-issues)
6. [GitHub Integration Issues](#github-integration-issues)
7. [Zsh Secrets Issues](#zsh-secrets-issues)
8. [Performance Issues](#performance-issues)
9. [Advanced Troubleshooting](#advanced-troubleshooting)
10. [Getting Help](#getting-help)

---

## 🔍 **Quick Diagnostics**

### **Run Comprehensive Diagnostics**

```bash
# Full system diagnostics
gitpersona diagnose --verbose

# Auto-fix detected issues
gitpersona diagnose --fix

# Focus on specific components
gitpersona diagnose --ssh-only
gitpersona diagnose --git-only
gitpersona diagnose --accounts-only
```

### **Quick Health Check**

```bash
# Check system health
gitpersona health

# Check current account status
gitpersona status

# Validate current configuration
gitpersona config validate
```

---

## 🚨 **Common Issues**

### **Issue: Command Not Found**

**Problem**: `gitpersona: command not found`

**Solutions**:
```bash
# 1. Check if binary exists
ls -la gitpersona

# 2. Make binary executable
chmod +x gitpersona

# 3. Install system-wide
sudo mv gitpersona /usr/local/bin/

# 4. Add to PATH
export PATH="$PATH:/path/to/gitpersona"
echo 'export PATH="$PATH:/path/to/gitpersona"' >> ~/.zshrc
```

### **Issue: Permission Denied**

**Problem**: `Permission denied` when running GitPersona

**Solutions**:
```bash
# 1. Check file permissions
ls -la gitpersona

# 2. Fix permissions
chmod +x gitpersona

# 3. Check directory permissions
ls -la /usr/local/bin/gitpersona
```

### **Issue: Configuration File Not Found**

**Problem**: `Configuration file not found`

**Solutions**:
```bash
# 1. Check config directory
ls -la ~/.config/gitpersona/

# 2. Create config directory
mkdir -p ~/.config/gitpersona

# 3. Initialize configuration
gitpersona diagnose --fix

# 4. Set custom config path
export GITPERSONA_CONFIG_PATH="/custom/path"
```

---

## 🔐 **SSH Issues**

### **Issue: SSH Authentication Failures**

**Problem**: `Permission denied (publickey)` when connecting to GitHub

#### **Diagnosis**
```bash
# 1. Test SSH connection manually
ssh -T git@github.com

# 2. Test with specific key
ssh -T git@github.com -i ~/.ssh/id_ed25519_work

# 3. Check SSH agent
ssh-add -l

# 4. Diagnose with GitPersona
gitpersona diagnose --ssh-only --verbose
```

#### **Solutions**
```bash
# 1. Use GitPersona's comprehensive SSH troubleshooting
gitpersona ssh-troubleshoot --verbose

# 2. Auto-fix SSH issues
gitpersona ssh-troubleshoot --auto-fix

# 3. Check SSH key permissions
ls -la ~/.ssh/id_ed25519_work
# Should be: -rw------- (600)

# 4. Fix permissions
chmod 600 ~/.ssh/id_ed25519_work
chmod 700 ~/.ssh

# 5. Add key to SSH agent
ssh-add ~/.ssh/id_ed25519_work

# 6. Test key with GitHub
ssh -T git@github.com -i ~/.ssh/id_ed25519_work

# 7. Verify key is added to GitHub account
# Go to GitHub Settings > SSH and GPG keys
```

### **Issue: SSH Socket Directory Errors**

**Problem**: `unix_listener: cannot bind to path /Users/username/.ssh/socket/...`

#### **Diagnosis**
```bash
# 1. Check socket directories
ls -la ~/.ssh/socket/
ls -la ~/.ssh/sockets/
ls -la ~/.ssh/control/

# 2. Check SSH agent status
gitpersona ssh-agent --status
```

#### **Solutions**
```bash
# 1. Create missing directories
mkdir -p ~/.ssh/socket ~/.ssh/sockets ~/.ssh/control

# 2. Set correct permissions
chmod 700 ~/.ssh/socket ~/.ssh/sockets ~/.ssh/control

# 3. Let GitPersona fix automatically
gitpersona diagnose --fix

# 4. Restart SSH agent
gitpersona ssh-agent --clear
gitpersona ssh-agent --start
```

### **Issue: Multiple SSH Keys Conflict**

**Problem**: Wrong account authenticating due to multiple keys

#### **Diagnosis**
```bash
# 1. List loaded keys
ssh-add -l

# 2. Check SSH config
cat ~/.ssh/config

# 3. Diagnose with GitPersona
gitpersona diagnose --ssh-only --verbose
```

#### **Solutions**
```bash
# 1. Use GitPersona's SSH troubleshooting for key conflicts
gitpersona ssh-troubleshoot --verbose

# 2. Auto-fix SSH key conflicts
gitpersona ssh-troubleshoot --auto-fix

# 3. Generate proper SSH configuration
gitpersona ssh-config generate --apply

# 4. Clear all keys from agent
gitpersona ssh-agent --clear

# 5. Load only the required key
gitpersona ssh-agent --load ~/.ssh/id_ed25519_work

# 6. Use SSH config with IdentitiesOnly
echo "IdentitiesOnly yes" >> ~/.ssh/config

# 7. Test with specific key
ssh -T git@github.com -i ~/.ssh/id_ed25519_work
```

### **Issue: Repository Not Found (SSH Key Conflicts)**

**Problem**: `Repository not found` error when you have access to the repository

This is a common issue when multiple SSH keys are loaded in the SSH agent, causing GitHub to authenticate with the wrong key.

#### **Diagnosis**
```bash
# 1. Use GitPersona's comprehensive SSH diagnostics
gitpersona ssh-troubleshoot --verbose

# 2. Check for SSH key conflicts
gitpersona diagnose --ssh-only --verbose

# 3. List loaded SSH keys
ssh-add -l
```

#### **Solutions**
```bash
# 1. Auto-fix SSH key conflicts
gitpersona ssh-troubleshoot --auto-fix

# 2. Generate proper SSH configuration
gitpersona ssh-config generate --apply

# 3. Clean up SSH agent
gitpersona ssh-agent --cleanup

# 4. Test GitHub connectivity
gitpersona ssh-troubleshoot --test-github
```

### **Issue: SSH Key Not Found**

**Problem**: `No such file or directory` for SSH key

#### **Diagnosis**
```bash
# 1. Check if key file exists
ls -la ~/.ssh/id_ed25519_work

# 2. Check account configuration
gitpersona config show --account work
```

#### **Solutions**
```bash
# 1. Generate new SSH key
gitpersona ssh-keys generate work

# 2. Update account configuration
gitpersona config set --account work ssh_key_path "/new/path"

# 3. Use existing key
gitpersona config set --account work ssh_key_path "/existing/path"
```

---

## ⚙️ **Git Configuration Issues**

### **Issue: Wrong Git User Information**

**Problem**: Commits showing wrong name/email

#### **Diagnosis**
```bash
# 1. Check current Git config
git config --global --list
git config --local --list

# 2. Check current account
gitpersona status

# 3. Validate Git configuration
gitpersona validate-git
```

#### **Solutions**
```bash
# 1. Switch to correct account
gitpersona switch work

# 2. Manually set Git config
git config --global user.name "John Doe"
git config --global user.email "john@company.com"

# 3. Fix with GitPersona
gitpersona diagnose --git-only --fix
```

### **Issue: SSH Command Not Set**

**Problem**: Git not using correct SSH key

#### **Diagnosis**
```bash
# 1. Check SSH command
git config --global core.sshCommand

# 2. Check account configuration
gitpersona config show --account work
```

#### **Solutions**
```bash
# 1. Set SSH command manually
git config --global core.sshCommand "ssh -i ~/.ssh/id_ed25519_work -o IdentitiesOnly=yes"

# 2. Switch account (auto-sets SSH command)
gitpersona switch work

# 3. Fix with GitPersona
gitpersona diagnose --git-only --fix
```

### **Issue: Git Config Conflicts**

**Problem**: Local and global Git config conflicts

#### **Diagnosis**
```bash
# 1. Check config precedence
git config --list --show-origin

# 2. Check for local config
ls -la .git/config
```

#### **Solutions**
```bash
# 1. Remove conflicting local config
git config --local --unset user.name
git config --local --unset user.email

# 2. Use global config only
gitpersona config set global_git_config true

# 3. Set up project-specific config
gitpersona project set work
```

---

## 👤 **Account Management Issues**

### **Issue: Account Not Found**

**Problem**: `Account 'work' not found`

#### **Diagnosis**
```bash
# 1. List all accounts
gitpersona list

# 2. Check configuration
gitpersona config show

# 3. Validate configuration
gitpersona config validate
```

#### **Solutions**
```bash
# 1. Add missing account
gitpersona add-github username --alias work

# 2. Check account alias
gitpersona list --account work

# 3. Fix account configuration
gitpersona config validate --fix
```

### **Issue: Account Switch Fails**

**Problem**: Account switch fails with validation errors

#### **Diagnosis**
```bash
# 1. Force switch with verbose output
gitpersona switch work --force --verbose

# 2. Check account validation
gitpersona diagnose --accounts-only

# 3. Check SSH validation
gitpersona diagnose --ssh-only
```

#### **Solutions**
```bash
# 1. Fix account configuration
gitpersona diagnose --fix

# 2. Reset SSH agent
gitpersona ssh-agent --clear

# 3. Skip validation (temporary)
gitpersona switch work --skip-validation

# 4. Force switch
gitpersona switch work --force
```

### **Issue: Duplicate Account Aliases**

**Problem**: Multiple accounts with same alias

#### **Diagnosis**
```bash
# 1. List all accounts
gitpersona list

# 2. Check configuration file
cat ~/.config/gitpersona/config.yaml
```

#### **Solutions**
```bash
# 1. Remove duplicate account
gitpersona remove duplicate-alias

# 2. Rename account
gitpersona config set --account old-alias alias new-alias

# 3. Fix configuration manually
gitpersona config edit
```

---

## 🔗 **GitHub Integration Issues**

### **Issue: GitHub CLI Not Found**

**Problem**: `GitHub CLI (gh) is not installed`

#### **Solutions**
```bash
# 1. Install GitHub CLI
# macOS
brew install gh

# Ubuntu/Debian
sudo apt install gh

# Windows
winget install GitHub.cli

# 2. Verify installation
gh --version

# 3. Authenticate
gh auth login
```

### **Issue: GitHub Authentication Failed**

**Problem**: `GitHub authentication failed`

#### **Diagnosis**
```bash
# 1. Check GitHub CLI status
gh auth status

# 2. Test GitHub API
gh api user

# 3. Check token
gh auth token
```

#### **Solutions**
```bash
# 1. Re-authenticate
gh auth login

# 2. Logout and login again
gh auth logout
gh auth login

# 3. Check network connectivity
curl -I https://api.github.com
```

### **Issue: GitHub API Rate Limit**

**Problem**: `API rate limit exceeded`

#### **Solutions**
```bash
# 1. Check rate limit
gh api rate_limit

# 2. Wait and retry
sleep 60
gitpersona switch work

# 3. Use personal access token
export GITHUB_TOKEN="ghp_your_token"
```

---

## 🔒 **Zsh Secrets Issues**

### **Issue: Zsh Secrets File Not Found**

**Problem**: `zsh_secrets file not found`

#### **Diagnosis**
```bash
# 1. Check common locations
ls -la ~/.zsh_secrets
ls -la ~/.config/zsh_secrets
ls -la ~/.secrets/zsh_secrets
ls -la ~/.zsh/secrets

# 2. Check GitPersona logs
tail -f ~/.config/gitpersona/logs/gitpersona.log
```

#### **Solutions**
```bash
# 1. Create zsh_secrets file
touch ~/.zsh_secrets
chmod 600 ~/.zsh_secrets

# 2. Let GitPersona create it
gitpersona switch work

# 3. Set custom location
export GITPERSONA_ZSH_SECRETS_PATH="/custom/path"
```

### **Issue: GITHUB_TOKEN Not Updating**

**Problem**: Token not updating in zsh_secrets file

#### **Diagnosis**
```bash
# 1. Check current token
gh auth token

# 2. Check zsh_secrets content
cat ~/.zsh_secrets

# 3. Test token update
gitpersona secrets update-token
```

#### **Solutions**
```bash
# 1. Update token manually
gitpersona secrets update-token

# 2. Check file permissions
ls -la ~/.zsh_secrets
chmod 600 ~/.zsh_secrets

# 3. Validate zsh_secrets file
gitpersona secrets validate

# 4. Reload zsh_secrets
gitpersona secrets reload
```

### **Issue: Zsh Secrets File Corruption**

**Problem**: Invalid syntax in zsh_secrets file

#### **Diagnosis**
```bash
# 1. Check file syntax
bash -n ~/.zsh_secrets

# 2. Validate with GitPersona
gitpersona secrets validate

# 3. Check file content
cat ~/.zsh_secrets
```

#### **Solutions**
```bash
# 1. Backup corrupted file
cp ~/.zsh_secrets ~/.zsh_secrets.backup

# 2. Fix syntax errors
gitpersona secrets validate --fix

# 3. Restore from backup
gitpersona secrets restore --backup 1

# 4. Recreate file
rm ~/.zsh_secrets
gitpersona switch work
```

---

## ⚡ **Performance Issues**

### **Issue: Slow Account Switching**

**Problem**: Account switch takes too long

#### **Diagnosis**
```bash
# 1. Enable debug logging
export GITPERSONA_DEBUG=true

# 2. Run with verbose output
gitpersona switch work --verbose

# 3. Check system resources
top
df -h
```

#### **Solutions**
```bash
# 1. Skip SSH validation for speed
gitpersona switch work --skip-validation

# 2. Clear SSH agent cache
gitpersona ssh-agent --clear

# 3. Optimize SSH configuration
gitpersona ssh-config optimize

# 4. Check disk space
df -h
```

### **Issue: High Memory Usage**

**Problem**: GitPersona using too much memory

#### **Solutions**
```bash
# 1. Check memory usage
ps aux | grep gitpersona

# 2. Restart GitPersona
pkill gitpersona

# 3. Clear caches
gitpersona cache clear

# 4. Optimize configuration
gitpersona config optimize
```

---

## 🔧 **Advanced Troubleshooting**

### **Debug Mode**

#### **Enable Debug Logging**
```bash
# 1. Set debug environment variable
export GITPERSONA_DEBUG=true

# 2. Set log level
export GITPERSONA_LOG_LEVEL=debug

# 3. Run command with verbose output
gitpersona switch work --verbose

# 4. Check logs
tail -f ~/.config/gitpersona/logs/gitpersona.log
```

#### **Debug Information Collection**
```bash
# 1. Collect system information
gitpersona diagnose --include-system > debug-info.txt

# 2. Include in bug reports
cat debug-info.txt
```

### **Configuration Recovery**

#### **Reset Configuration**
```bash
# 1. Backup current configuration
gitpersona config backup

# 2. Reset to defaults
gitpersona config reset

# 3. Restore from backup if needed
gitpersona config restore --backup 1
```

#### **Clean Installation**
```bash
# 1. Remove configuration
rm -rf ~/.config/gitpersona

# 2. Remove SSH keys (optional)
rm -rf ~/.ssh/id_ed25519_*

# 3. Reinstall GitPersona
go build -o gitpersona
sudo mv gitpersona /usr/local/bin/

# 4. Reconfigure
gitpersona add-github username
```

### **Network Issues**

#### **Proxy Configuration**
```bash
# 1. Set proxy environment variables
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080

# 2. Configure Git with proxy
git config --global http.proxy http://proxy.company.com:8080
git config --global https.proxy http://proxy.company.com:8080

# 3. Configure SSH with proxy
# Add to ~/.ssh/config:
# Host github.com
#     ProxyCommand nc -X connect -x proxy.company.com:8080 %h %p
```

#### **Firewall Issues**
```bash
# 1. Test GitHub connectivity
curl -I https://api.github.com

# 2. Test SSH connectivity
ssh -T git@github.com

# 3. Check firewall rules
sudo ufw status
```

---

## 🆘 **Getting Help**

### **Self-Help Resources**

#### **Built-in Help**
```bash
# General help
gitpersona --help

# Command-specific help
gitpersona diagnose --help
gitpersona switch --help
gitpersona ssh-keys --help
```

#### **Diagnostic Commands**
```bash
# Comprehensive diagnostics
gitpersona diagnose --verbose

# System information
gitpersona diagnose --include-system

# Health check
gitpersona health
```

### **Community Support**

#### **GitHub Issues**
- **[Report Bugs](https://github.com/techishthoughts/GitPersona/issues)** - Report issues and bugs
- **[Feature Requests](https://github.com/techishthoughts/GitPersona/issues)** - Request new features
- **[Discussions](https://github.com/techishthoughts/GitPersona/discussions)** - Community support

#### **Bug Report Template**
```markdown
**Bug Description**
Brief description of the issue

**Steps to Reproduce**
1. Run command: `gitpersona switch work`
2. See error: `Permission denied (publickey)`

**Expected Behavior**
Account should switch successfully

**Actual Behavior**
Switch fails with SSH authentication error

**Environment**
- OS: macOS 14.0
- GitPersona Version: 1.0.0
- Go Version: 1.21.0

**Debug Information**
```
$ gitpersona diagnose --include-system
[Output here]
```

**Additional Context**
Any other relevant information
```

### **Professional Support**

#### **Enterprise Support**
For enterprise users requiring professional support:
- **Email**: support@gitpersona.com
- **Documentation**: [Enterprise Guide](ENTERPRISE.md)
- **SLA**: 24-hour response time

#### **Consulting Services**
- **Implementation**: Help with GitPersona deployment
- **Customization**: Tailored configurations
- **Training**: Team training sessions

---

## 📚 **Additional Resources**

- **[User Guide](USER_GUIDE.md)** - Complete user documentation
- **[Configuration Guide](CONFIGURATION.md)** - Detailed configuration options
- **[Architecture Guide](ARCHITECTURE.md)** - Technical architecture details
- **[Security Guide](SECURITY.md)** - Security best practices
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute

---

<div align="center">

**Still need help?**

- 🐛 **[Report a Bug](https://github.com/techishthoughts/GitPersona/issues)**
- 💬 **[Join Discussions](https://github.com/techishthoughts/GitPersona/discussions)**
- 📧 **[Contact Support](mailto:support@gitpersona.com)**

</div>
