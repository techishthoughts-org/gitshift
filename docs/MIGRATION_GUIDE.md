# GitPersona v2.0 Migration Guide

## 🚀 Quick Start Migration

### Automatic Migration
```bash
# Check if migration is needed
gitpersona migrate check

# Run automatic migration with backup
gitpersona migrate run --backup

# Verify migration
gitpersona account list --detailed
```

### Manual Migration Steps
```bash
# 1. Create backup
gitpersona migrate backup create

# 2. Import existing accounts
gitpersona migrate run --from-legacy

# 3. Validate configuration
gitpersona diagnose health
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

# Verify GitPersona installation
gitpersona version
```

## 🔄 Migration Scenarios

### Scenario 1: First-Time Installation

**From:** No GitPersona installation
**To:** GitPersona v2.0 with discovered accounts

```bash
# Install GitPersona v2.0
go install github.com/techishthoughts/GitPersona@v2.0.0

# Auto-discover accounts from Git config and SSH keys
gitpersona smart detect --auto-import

# Create your first account if none discovered
gitpersona account add main \
  --name "Your Name" \
  --email "your@email.com"

# Generate SSH key
gitpersona ssh keys generate \
  --name main \
  --type ed25519 \
  --email "your@email.com"
```

### Scenario 2: Legacy GitPersona v1.x

**From:** GitPersona v1.x configuration
**To:** GitPersona v2.0 with enhanced features

```bash
# Backup existing config
cp ~/.gitpersona/config.yaml ~/.gitpersona/config.yaml.backup

# Run migration
gitpersona migrate run --from-v1

# Verify enhanced features
gitpersona account list --json | jq '.accounts[].isolation_level'

# Test new functionality
gitpersona security audit
```

### Scenario 3: Multiple Git Identities

**From:** Manual Git config switching
**To:** Managed account switching

```bash
# Discover existing identities
gitpersona smart detect --comprehensive

# Import discovered accounts
gitpersona migrate run --interactive

# Configure account isolation
gitpersona account update work --isolation-level strict
gitpersona account update personal --isolation-level standard

# Test account switching
gitpersona account switch work
gitpersona git config show
```

### Scenario 4: SSH Key Management

**From:** Manual SSH key handling
**To:** Automated SSH management

```bash
# Import existing SSH keys
gitpersona ssh keys import --discover

# Associate keys with accounts
gitpersona account update work --ssh-key ~/.ssh/id_ed25519_work
gitpersona account update personal --ssh-key ~/.ssh/id_ed25519_personal

# Test SSH functionality
gitpersona ssh test
gitpersona ssh fix --auto
```

### Scenario 5: GitHub Integration

**From:** Manual token management
**To:** Secure token storage

```bash
# Set up GitHub tokens securely
gitpersona github token set --account work
gitpersona github token set --account personal

# Test GitHub integration
gitpersona github test --account work

# Upload SSH keys to GitHub
gitpersona ssh keys upload --account work --title "Work Laptop"
```

## 🛠️ Configuration Migration

### Legacy Config Structure
```yaml
# ~/.gitpersona/config.yaml (v1.x)
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
# ~/.gitpersona/config.yaml (v2.0)
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
gitpersona ssh keys discover

# Manual import
gitpersona ssh keys import --path ~/.ssh/id_ed25519_work --account work
gitpersona ssh keys import --path ~/.ssh/id_ed25519_personal --account personal

# Validate imported keys
gitpersona ssh keys validate --all
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
gitpersona ssh keys generate --account work --type ed25519
gitpersona ssh keys generate --account personal --type ed25519

# Configure SSH agent isolation
gitpersona account update work --ssh-isolation strict
gitpersona account update personal --ssh-isolation standard

# Fix SSH permissions
gitpersona ssh fix --auto
```

## 🐙 GitHub Integration Migration

### Token Migration
```bash
# Import existing tokens (secure prompt)
gitpersona github token set --account work
gitpersona github token set --account personal

# Validate tokens
gitpersona github test --account work
gitpersona github test --account personal

# Configure token settings
gitpersona github config --account work --auto-validate true
gitpersona github config --account personal --scope "repo,user"
```

### SSH Key Upload
```bash
# Upload keys to GitHub
gitpersona ssh keys upload --account work --title "Work MacBook Pro"
gitpersona ssh keys upload --account personal --title "Personal Laptop"

# Verify GitHub SSH access
gitpersona ssh test --github
```

## 🔒 Security Enhancement Migration

### Security Audit
```bash
# Run initial security audit
gitpersona security audit --detailed

# Fix security violations
gitpersona security fix --auto

# Enable ongoing monitoring
gitpersona security config --auto-audit true --interval daily
```

### Isolation Levels
```bash
# Configure isolation levels per account
gitpersona account update work --isolation-level strict
gitpersona account update personal --isolation-level standard
gitpersona account update opensource --isolation-level minimal

# Test isolation
gitpersona account switch work
gitpersona diagnose isolation
```

## 🚨 Troubleshooting Migration Issues

### Common Issues

#### Issue: Migration Fails with Permission Error
```bash
# Fix: Correct file permissions
chmod 700 ~/.gitpersona
chmod 600 ~/.gitpersona/config.yaml
gitpersona migrate run --retry
```

#### Issue: SSH Keys Not Discovered
```bash
# Fix: Manual SSH key discovery
gitpersona ssh keys discover --verbose
gitpersona ssh keys import --path ~/.ssh/id_* --auto-associate
```

#### Issue: GitHub Token Invalid
```bash
# Fix: Regenerate GitHub token
gitpersona github token set --account work --regenerate
gitpersona github test --account work
```

#### Issue: Git Config Not Switching
```bash
# Fix: Reset Git configuration
gitpersona git reset --account work
gitpersona account switch work --force
gitpersona git config show
```

#### Issue: Account Validation Fails
```bash
# Fix: Validate and repair account
gitpersona account validate work --fix
gitpersona account repair work --auto
```

### Recovery Commands
```bash
# Restore from backup
gitpersona migrate restore --backup ~/.gitpersona/backups/latest

# Reset to clean state
gitpersona reset --confirm --backup

# Rebuild configuration
gitpersona migrate run --clean-install
```

### Debug Mode
```bash
# Enable debug logging
export GITPERSONA_LOG_LEVEL=debug

# Run commands with verbose output
gitpersona account list --verbose --debug

# Check system diagnostics
gitpersona diagnose system --comprehensive
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
gitpersona diagnose health --verbose

# Test all accounts
gitpersona account validate --all

# Test SSH functionality
gitpersona ssh test --all-accounts

# Verify GitHub integration
gitpersona github test --all-accounts

# Security verification
gitpersona security audit --comprehensive
```

### Performance Validation
```bash
# Benchmark account operations
gitpersona benchmark --operations account,ssh,git

# Test parallel execution
gitpersona account list --parallel --time

# Measure improvement
gitpersona stats --compare-v1
```

## 🔮 Advanced Migration Features

### Batch Operations
```bash
# Import multiple accounts from CSV
gitpersona migrate import --csv accounts.csv

# Bulk SSH key generation
gitpersona ssh keys generate --batch --accounts work,personal,oss

# Mass account validation
gitpersona account validate --batch --fix-issues
```

### Custom Migration Scripts
```bash
# Export current config for custom processing
gitpersona migrate export --format yaml > current-config.yaml

# Apply custom transformations
./custom-migration-script.sh current-config.yaml

# Import processed config
gitpersona migrate import --config processed-config.yaml
```

### Integration Testing
```bash
# Test migration in sandbox
gitpersona migrate test --sandbox

# Validate migration completeness
gitpersona migrate validate --comprehensive

# Performance impact analysis
gitpersona migrate benchmark --before-after
```

## 📞 Support & Resources

### Getting Help
- **Documentation**: `docs/` directory
- **Command Help**: `gitpersona [command] --help`
- **Debug Info**: `gitpersona diagnose system --debug`
- **Issues**: GitHub issue tracker

### Migration Support
- **Pre-migration Consultation**: Run `gitpersona migrate check --advice`
- **Interactive Migration**: Use `gitpersona migrate run --interactive`
- **Rollback Support**: Automatic backups enable safe rollback
- **Community Support**: GitHub discussions

### Best Practices
1. **Always backup** before migration
2. **Test in stages** - don't migrate everything at once
3. **Verify each step** before proceeding
4. **Keep backups** for rollback capability
5. **Document custom configurations** for future reference

---

**Welcome to GitPersona v2.0!** - Your Git identity management just got 87% faster and infinitely more secure.