# gitshift v2.0 Migration Guide

## 🚀 Quick Start Migration

### Automatic Migration
```bash
# Check if migration is needed
gitshift migrate check

# Run automatic migration with backup
gitshift migrate run --backup

# Verify migration
gitshift account list --detailed
```

### Manual Migration Steps
```bash
# 1. Create backup
gitshift migrate backup create

# 2. Import existing accounts
gitshift migrate run --from-legacy

# 3. Validate configuration
gitshift diagnose health
```

## 📋 Pre-Migration Checklist

### System Requirements
- [ ] Go 1.21+ installed
- [ ] Git 2.30+ installed
- [ ] SSH client available
- [ ] Backup of existing configuration

### Account Preparation
- [ ] List current Git identities
- [ ] Document SSH key locations
- [ ] Note GitHub tokens/credentials
- [ ] Identify repository contexts

### Verification Commands
```bash
# Check current Git config
git config --global --list

# List SSH keys
ls -la ~/.ssh/

# Verify gitshift installation
gitshift version
```

## 🔄 Migration Scenarios

### Scenario 1: First-Time Installation

**From:** No gitshift installation
**To:** gitshift v2.0 with discovered accounts

```bash
# Install gitshift v2.0
go install github.com/techishthoughts/gitshift@v2.0.0

# Auto-discover accounts from Git config and SSH keys
gitshift smart detect --auto-import

# Create your first account if none discovered
gitshift account add main \
  --name "Your Name" \
  --email "your@email.com"

# Generate SSH key
gitshift ssh keys generate \
  --name main \
  --type ed25519 \
  --email "your@email.com"
```

### Scenario 2: Legacy gitshift v1.x

**From:** gitshift v1.x configuration
**To:** gitshift v2.0 with enhanced features

```bash
# Backup existing config
cp ~/.gitshift/config.yaml ~/.gitshift/config.yaml.backup

# Run migration
gitshift migrate run --from-v1

# Verify enhanced features
gitshift account list --json | jq '.accounts[].isolation_level'

# Test new functionality
gitshift security audit
```

### Scenario 3: Multiple Git Identities

**From:** Manual Git config switching
**To:** Managed account switching

```bash
# Discover existing identities
gitshift smart detect --comprehensive

# Import discovered accounts
gitshift migrate run --interactive

# Configure account isolation
gitshift account update work --isolation-level strict
gitshift account update personal --isolation-level standard

# Test account switching
gitshift account switch work
gitshift git config show
```

### Scenario 4: SSH Key Management

**From:** Manual SSH key handling
**To:** Automated SSH management

```bash
# Import existing SSH keys
gitshift ssh keys import --discover

# Associate keys with accounts
gitshift account update work --ssh-key ~/.ssh/id_ed25519_work
gitshift account update personal --ssh-key ~/.ssh/id_ed25519_personal

# Test SSH functionality
gitshift ssh test
gitshift ssh fix --auto
```

### Scenario 5: GitHub Integration

**From:** Manual token management
**To:** Secure token storage

```bash
# Set up GitHub tokens securely
gitshift github token set --account work
gitshift github token set --account personal

# Test GitHub integration
gitshift github test --account work

# Upload SSH keys to GitHub
gitshift ssh keys upload --account work --title "Work Laptop"
```

## 🛠️ Configuration Migration

### Legacy Config Structure
```yaml
# ~/.gitshift/config.yaml (v1.x)
current_account: work
global_git_mode: true
accounts:
  work:
    name: "John Doe"
    email: "john@company.com"
    ssh_key_path: "~/.ssh/id_ed25519_work"
    github_username: "johndoe-work"
  personal:
    name: "John Doe"
    email: "john.personal@gmail.com"
    ssh_key_path: "~/.ssh/id_ed25519_personal"
    github_username: "johndoe"
```

### New Config Structure
```yaml
# ~/.gitshift/config.yaml (v2.0)
config_version: "2.0.0"
current_account: work
global_git_config: true
auto_detect: true
accounts:
  work:
    alias: work
    name: "John Doe"
    email: "john@company.com"
    ssh_key_path: "~/.ssh/id_ed25519_work"
    github_username: "johndoe-work"
    isolation_level: "strict"
    isolation_metadata:
      ssh_isolation:
        use_isolated_agent: true
        force_identities_only: true
        agent_timeout: 3600
      token_isolation:
        use_encrypted_storage: true
        auto_validation: true
        validation_interval: 60
      git_isolation:
        use_local_config: true
        isolate_ssh_command: true
      environment_isolation:
        isolate_environment: false
        preserve_environment: ["HOME", "USER", "PATH"]
    account_metadata:
      created_at: "2024-01-15T10:30:00Z"
      last_used: "2024-01-16T09:15:00Z"
```

## 🔑 SSH Key Migration

### Existing SSH Keys Discovery
```bash
# Automatic discovery
gitshift ssh keys discover

# Manual import
gitshift ssh keys import --path ~/.ssh/id_ed25519_work --account work
gitshift ssh keys import --path ~/.ssh/id_ed25519_personal --account personal

# Validate imported keys
gitshift ssh keys validate --all
```

### SSH Key Naming Convention
```
Before: Manual naming
id_rsa
id_ed25519
github_key

After: Account-based naming
id_ed25519_work
id_ed25519_personal
id_ed25519_opensource
```

### SSH Configuration Enhancement
```bash
# Generate new keys with proper naming
gitshift ssh keys generate --account work --type ed25519
gitshift ssh keys generate --account personal --type ed25519

# Configure SSH agent isolation
gitshift account update work --ssh-isolation strict
gitshift account update personal --ssh-isolation standard

# Fix SSH permissions
gitshift ssh fix --auto
```

## 🐙 GitHub Integration Migration

### Token Migration
```bash
# Import existing tokens (secure prompt)
gitshift github token set --account work
gitshift github token set --account personal

# Validate tokens
gitshift github test --account work
gitshift github test --account personal

# Configure token settings
gitshift github config --account work --auto-validate true
gitshift github config --account personal --scope "repo,user"
```

### SSH Key Upload
```bash
# Upload keys to GitHub
gitshift ssh keys upload --account work --title "Work MacBook Pro"
gitshift ssh keys upload --account personal --title "Personal Laptop"

# Verify GitHub SSH access
gitshift ssh test --github
```

## 🔒 Security Enhancement Migration

### Security Audit
```bash
# Run initial security audit
gitshift security audit --detailed

# Fix security violations
gitshift security fix --auto

# Enable ongoing monitoring
gitshift security config --auto-audit true --interval daily
```

### Isolation Levels
```bash
# Configure isolation levels per account
gitshift account update work --isolation-level strict
gitshift account update personal --isolation-level standard
gitshift account update opensource --isolation-level minimal

# Test isolation
gitshift account switch work
gitshift diagnose isolation
```

## 🚨 Troubleshooting Migration Issues

### Common Issues

#### Issue: Migration Fails with Permission Error
```bash
# Fix: Correct file permissions
chmod 700 ~/.gitshift
chmod 600 ~/.gitshift/config.yaml
gitshift migrate run --retry
```

#### Issue: SSH Keys Not Discovered
```bash
# Fix: Manual SSH key discovery
gitshift ssh keys discover --verbose
gitshift ssh keys import --path ~/.ssh/id_* --auto-associate
```

#### Issue: GitHub Token Invalid
```bash
# Fix: Regenerate GitHub token
gitshift github token set --account work --regenerate
gitshift github test --account work
```

#### Issue: Git Config Not Switching
```bash
# Fix: Reset Git configuration
gitshift git reset --account work
gitshift account switch work --force
gitshift git config show
```

#### Issue: Account Validation Fails
```bash
# Fix: Validate and repair account
gitshift account validate work --fix
gitshift account repair work --auto
```

### Recovery Commands
```bash
# Restore from backup
gitshift migrate restore --backup ~/.gitshift/backups/latest

# Reset to clean state
gitshift reset --confirm --backup

# Rebuild configuration
gitshift migrate run --clean-install
```

### Debug Mode
```bash
# Enable debug logging
export gitshift_LOG_LEVEL=debug

# Run commands with verbose output
gitshift account list --verbose --debug

# Check system diagnostics
gitshift diagnose system --comprehensive
```

## 📊 Post-Migration Verification

### Verification Checklist
- [ ] All accounts imported correctly
- [ ] SSH keys associated with accounts
- [ ] Git config switches properly
- [ ] GitHub integration working
- [ ] Security audit passes
- [ ] Account switching functional

### Verification Commands
```bash
# Comprehensive health check
gitshift diagnose health --verbose

# Test all accounts
gitshift account validate --all

# Test SSH functionality
gitshift ssh test --all-accounts

# Verify GitHub integration
gitshift github test --all-accounts

# Security verification
gitshift security audit --comprehensive
```

### Performance Validation
```bash
# Benchmark account operations
gitshift benchmark --operations account,ssh,git

# Test parallel execution
gitshift account list --parallel --time

# Measure improvement
gitshift stats --compare-v1
```

## 🔮 Advanced Migration Features

### Batch Operations
```bash
# Import multiple accounts from CSV
gitshift migrate import --csv accounts.csv

# Bulk SSH key generation
gitshift ssh keys generate --batch --accounts work,personal,oss

# Mass account validation
gitshift account validate --batch --fix-issues
```

### Custom Migration Scripts
```bash
# Export current config for custom processing
gitshift migrate export --format yaml > current-config.yaml

# Apply custom transformations
./custom-migration-script.sh current-config.yaml

# Import processed config
gitshift migrate import --config processed-config.yaml
```

### Integration Testing
```bash
# Test migration in sandbox
gitshift migrate test --sandbox

# Validate migration completeness
gitshift migrate validate --comprehensive

# Performance impact analysis
gitshift migrate benchmark --before-after
```

## 📞 Support & Resources

### Getting Help
- **Documentation**: `docs/` directory
- **Command Help**: `gitshift [command] --help`
- **Debug Info**: `gitshift diagnose system --debug`
- **Issues**: GitHub issue tracker

### Migration Support
- **Pre-migration Consultation**: Run `gitshift migrate check --advice`
- **Interactive Migration**: Use `gitshift migrate run --interactive`
- **Rollback Support**: Automatic backups enable safe rollback
- **Community Support**: GitHub discussions

### Best Practices
1. **Always backup** before migration
2. **Test in stages** - don't migrate everything at once
3. **Verify each step** before proceeding
4. **Keep backups** for rollback capability
5. **Document custom configurations** for future reference

---

**Welcome to gitshift v2.0!** - Your Git identity management just got 87% faster and infinitely more secure.
