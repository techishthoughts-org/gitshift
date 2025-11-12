# Multi-Platform Support

> **New in v0.2.0**: gitshift now supports multiple Git hosting platforms including GitHub, GitLab, GitHub Enterprise, and self-hosted instances.

## Table of Contents

- [Overview](#overview)
- [Supported Platforms](#supported-platforms)
- [Quick Start](#quick-start)
- [Platform Configuration](#platform-configuration)
- [Usage Examples](#usage-examples)
- [Platform Detection](#platform-detection)
- [SSH Configuration](#ssh-configuration)
- [Migration from GitHub-Only](#migration-from-github-only)
- [Troubleshooting](#troubleshooting)

---

## Overview

gitshift's multi-platform support allows you to:

- Manage accounts across **GitHub**, **GitLab**, **GitHub Enterprise**, and **self-hosted GitLab**
- Switch seamlessly between different platforms
- Use platform-specific features and APIs
- Maintain complete SSH key isolation per account
- Auto-detect platform from repository URLs
- Keep backward compatibility with GitHub-only configurations

**Key Features:**

✅ Unified CLI for all platforms
✅ Automatic platform detection
✅ Platform-specific SSH configuration
✅ Custom domain support (GitHub Enterprise, self-hosted GitLab)
✅ Backward compatible with existing configs

---

## Supported Platforms

| Platform | Status | Default Domain | API Support |
|----------|--------|----------------|-------------|
| **GitHub** | ✅ Fully Supported | `github.com` | ✅ Full |
| **GitLab** | ✅ Fully Supported | `gitlab.com` | ✅ Full |
| **GitHub Enterprise** | ✅ Fully Supported | Custom | ✅ Full |
| **Self-hosted GitLab** | ✅ Fully Supported | Custom | ✅ Full |
| **Bitbucket** | 🚧 Planned | `bitbucket.org` | 🚧 Planned |
| **Custom Git Servers** | 🚧 Planned | Custom | ⚠️ SSH only |

---

## Quick Start

### Adding Accounts for Different Platforms

**GitHub Account:**
```bash
gitshift add personal-github \
  --name "John Doe" \
  --email john@personal.com \
  --platform github
```

**GitLab Account:**
```bash
gitshift add personal-gitlab \
  --name "John Doe" \
  --email john@personal.com \
  --platform gitlab
```

**GitHub Enterprise:**
```bash
gitshift add enterprise \
  --name "John Doe" \
  --email john.doe@company.com \
  --platform github \
  --domain github.enterprise.com
```

**Self-hosted GitLab:**
```bash
gitshift add company-gitlab \
  --name "John Doe" \
  --email john.doe@company.com \
  --platform gitlab \
  --domain gitlab.company.com
```

### Switching Between Platforms

```bash
# Switch to GitHub account
gitshift switch personal-github

# Switch to GitLab account
gitshift switch personal-gitlab

# Switch to enterprise account
gitshift switch enterprise
```

### Listing Accounts with Platforms

```bash
gitshift list
```

Output shows platform information:
```
┌─────────────────────┬──────────────────────┬─────────────────────────┬──────────┬────────┐
│ ALIAS               │ NAME                 │ EMAIL                   │ PLATFORM │ STATUS │
├─────────────────────┼──────────────────────┼─────────────────────────┼──────────┼────────┤
│ personal-github *   │ John Doe             │ john@personal.com       │ github   │ active │
│ personal-gitlab     │ John Doe             │ john@personal.com       │ gitlab   │ active │
│ enterprise          │ John Doe             │ john.doe@company.com    │ github   │ active │
│ company-gitlab      │ John Doe             │ john.doe@company.com    │ gitlab   │ active │
└─────────────────────┴──────────────────────┴─────────────────────────┴──────────┴────────┘
```

---

## Platform Configuration

### Configuration Fields

Multi-platform accounts use these fields in `~/.gitshift/config.yaml`:

```yaml
accounts:
  account-name:
    alias: account-name           # Unique identifier
    name: John Doe                # Git commit name
    email: john@example.com       # Git commit email
    ssh_key_path: ~/.ssh/key      # Path to SSH key
    platform: github              # Platform type (NEW)
    username: johndoe             # Platform username (NEW)
    domain: github.com            # Platform domain (NEW, optional)
    api_endpoint: https://...     # API endpoint (NEW, optional)
    description: My account       # Human-readable description
    is_default: false             # Default account flag
    status: active                # Account status
    isolation_level: standard     # SSH isolation level
```

### Platform-Specific Fields

#### Required for All Platforms

- `platform` - Platform type: `github` or `gitlab`
- `username` - Your username on that platform

#### Optional but Recommended

- `domain` - Custom domain (required for enterprise/self-hosted)
- `api_endpoint` - Custom API endpoint (auto-generated if not specified)

#### Legacy Fields (Deprecated but Supported)

- `github_username` - Old field, now replaced by `username` + `platform`

---

## Usage Examples

### Example 1: Personal GitHub and GitLab

```yaml
accounts:
  personal-github:
    alias: personal-github
    name: John Doe
    email: john@personal.com
    ssh_key_path: ~/.ssh/id_ed25519_github_personal
    platform: github
    username: johndoe
    description: Personal GitHub account
    is_default: true

  personal-gitlab:
    alias: personal-gitlab
    name: John Doe
    email: john@personal.com
    ssh_key_path: ~/.ssh/id_ed25519_gitlab_personal
    platform: gitlab
    username: johndoe
    description: Personal GitLab account
```

**Workflow:**
```bash
# Work on GitHub project
gitshift switch personal-github
git clone git@github.com:johndoe/my-project.git

# Work on GitLab project
gitshift switch personal-gitlab
git clone git@gitlab.com:johndoe/my-project.git
```

---

### Example 2: Multiple GitHub Organizations + GitLab

```yaml
accounts:
  personal:
    platform: github
    username: johndoe
    email: john@personal.com
    ssh_key_path: ~/.ssh/id_ed25519_personal

  work-github:
    platform: github
    username: johndoe-work
    email: john.doe@company.com
    ssh_key_path: ~/.ssh/id_ed25519_work

  oss-gitlab:
    platform: gitlab
    username: johndoe
    email: john@personal.com
    ssh_key_path: ~/.ssh/id_ed25519_gitlab_oss
```

**Workflow:**
```bash
# Personal GitHub projects
gitshift switch personal

# Work GitHub projects
gitshift switch work-github

# Open source on GitLab
gitshift switch oss-gitlab
```

---

### Example 3: GitHub Enterprise + Self-Hosted GitLab

```yaml
accounts:
  github-enterprise:
    alias: github-enterprise
    name: John Doe
    email: john.doe@company.com
    ssh_key_path: ~/.ssh/id_ed25519_ghe
    platform: github
    username: jdoe
    domain: github.enterprise.com
    api_endpoint: https://github.enterprise.com/api/v3
    description: Company GitHub Enterprise

  self-hosted-gitlab:
    alias: self-hosted-gitlab
    name: John Doe
    email: john.doe@company.com
    ssh_key_path: ~/.ssh/id_ed25519_gitlab_company
    platform: gitlab
    username: jdoe
    domain: gitlab.company.com
    api_endpoint: https://gitlab.company.com/api/v4
    description: Company self-hosted GitLab
```

**Workflow:**
```bash
# Work on enterprise GitHub
gitshift switch github-enterprise
git clone git@github.enterprise.com:org/repo.git

# Work on company GitLab
gitshift switch self-hosted-gitlab
git clone git@gitlab.company.com:group/project.git
```

---

### Example 4: Freelancer with Multiple Clients

```yaml
accounts:
  client-a-github:
    platform: github
    username: freelancer-clienta
    email: freelancer@clienta.com
    domain: github.com

  client-b-gitlab:
    platform: gitlab
    username: freelancer
    email: freelancer@clientb.com
    domain: gitlab.clientb.com  # Client's self-hosted GitLab

  client-c-github-enterprise:
    platform: github
    username: freelancer
    email: freelancer@clientc.com
    domain: github.clientc.com
```

**Workflow:**
```bash
# Switch between client projects instantly
gitshift switch client-a-github
gitshift switch client-b-gitlab
gitshift switch client-c-github-enterprise

# Test SSH connections for each
gitshift ssh-test client-a-github
gitshift ssh-test client-b-gitlab
gitshift ssh-test client-c-github-enterprise
```

---

## Platform Detection

### Automatic Detection from Repository URLs

gitshift automatically detects the platform from Git repository URLs:

**GitHub Detection:**
```bash
git@github.com:user/repo.git                    → GitHub (github.com)
git@github.enterprise.com:user/repo.git         → GitHub (github.enterprise.com)
https://github.com/user/repo.git                → GitHub (github.com)
```

**GitLab Detection:**
```bash
git@gitlab.com:user/repo.git                    → GitLab (gitlab.com)
git@gitlab.company.com:user/repo.git            → GitLab (gitlab.company.com)
https://gitlab.com/user/repo.git                → GitLab (gitlab.com)
```

### How It Works

1. **Domain Extraction**: gitshift extracts the domain from the repository URL
2. **Account Matching**: Matches the domain to configured accounts
3. **Platform Selection**: Automatically selects the appropriate account for that platform
4. **SSH Configuration**: Applies platform-specific SSH settings

### Manual Platform Specification

You can also explicitly specify the platform:

```bash
# Add account with explicit platform
gitshift add my-account --platform gitlab --domain gitlab.company.com
```

---

## SSH Configuration

### Platform-Specific SSH Hosts

gitshift creates platform-specific SSH configurations for each account:

**Generated SSH Config (`~/.ssh/config`):**

```ssh-config
# GitHub account: personal-github
Host github.com-personal-github
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_github_personal
    IdentitiesOnly yes

# GitLab account: personal-gitlab
Host gitlab.com-personal-gitlab
    HostName gitlab.com
    User git
    IdentityFile ~/.ssh/id_ed25519_gitlab_personal
    IdentitiesOnly yes

# GitHub Enterprise: enterprise
Host github.enterprise.com-enterprise
    HostName github.enterprise.com
    User git
    IdentityFile ~/.ssh/id_ed25519_ghe
    IdentitiesOnly yes
```

### SSH Key Isolation

Each account gets its own SSH key with `IdentitiesOnly yes` to ensure:

- ✅ No SSH key fallback or trial-and-error
- ✅ Complete isolation between accounts
- ✅ Clear SSH authentication per platform
- ✅ No accidental key leakage

### Testing SSH Connections

Test SSH for any platform:

```bash
# Test GitHub account
gitshift ssh-test personal-github
# Output: Testing SSH connection for github.com...
#         ✓ Successfully authenticated as johndoe

# Test GitLab account
gitshift ssh-test personal-gitlab
# Output: Testing SSH connection for gitlab.com...
#         ✓ Successfully authenticated as johndoe

# Test enterprise account
gitshift ssh-test github-enterprise
# Output: Testing SSH connection for github.enterprise.com...
#         ✓ Successfully authenticated as jdoe
```

---

## Migration from GitHub-Only

### Backward Compatibility

**Good news:** Your existing GitHub-only configuration still works!

gitshift maintains full backward compatibility:

- Accounts without `platform` field default to `github`
- Old `github_username` field is still supported
- All existing commands work as before

### Migration Options

#### Option 1: Do Nothing (Recommended for Simple Setups)

If you only use GitHub, you don't need to change anything:

```yaml
# This still works perfectly
accounts:
  personal:
    alias: personal
    name: John Doe
    email: john@personal.com
    ssh_key_path: ~/.ssh/id_ed25519_personal
    github_username: johndoe  # Old field, still supported
```

#### Option 2: Migrate to New Format (Recommended for Multi-Platform)

Update your config to use the new fields:

**Before:**
```yaml
accounts:
  personal:
    alias: personal
    github_username: johndoe
    # ... other fields
```

**After:**
```yaml
accounts:
  personal:
    alias: personal
    platform: github         # NEW: explicitly set platform
    username: johndoe        # NEW: replaces github_username
    domain: github.com       # NEW: optional, defaults to github.com
    # ... other fields
```

### Migration Steps

1. **Backup your configuration:**
   ```bash
   cp ~/.gitshift/config.yaml ~/.gitshift/config.yaml.backup
   ```

2. **Update account fields:**
   ```bash
   # Edit configuration
   vim ~/.gitshift/config.yaml

   # For each account, add:
   # - platform: github
   # - username: <your-github-username>
   # - domain: github.com (optional)
   ```

3. **Test the configuration:**
   ```bash
   gitshift list
   gitshift current
   gitshift ssh-test <account-alias>
   ```

4. **Optional: Add GitLab or other platforms:**
   ```bash
   gitshift add my-gitlab --platform gitlab
   ```

### Deprecation Notice

The `github_username` field is **deprecated** but **still supported** for backward compatibility.

**Timeline:**
- **Now**: Both `github_username` and `username`+`platform` work
- **Future**: `github_username` will remain supported indefinitely for existing configs
- **Recommendation**: New accounts should use `username` + `platform`

---

## Troubleshooting

### Common Issues

#### Issue 1: "Unknown platform" error

**Error:**
```
Error: unknown platform 'bitbucket'
```

**Solution:**
Check [Supported Platforms](#supported-platforms) table. Use `github` or `gitlab`.

---

#### Issue 2: SSH authentication fails for custom domain

**Error:**
```
Permission denied (publickey)
```

**Solution:**
1. Verify domain is correct:
   ```bash
   gitshift list  # Check domain field
   ```

2. Test SSH manually:
   ```bash
   ssh -T git@your-domain.com -i ~/.ssh/your_key
   ```

3. Ensure SSH key is added to platform:
   - GitHub Enterprise: Settings → SSH Keys
   - Self-hosted GitLab: User Settings → SSH Keys

---

#### Issue 3: Platform not auto-detected from URL

**Symptoms:** gitshift doesn't recognize custom domain

**Solution:**
Explicitly add domain to account configuration:

```bash
gitshift add my-account \
  --platform gitlab \
  --domain gitlab.custom.com
```

---

#### Issue 4: API endpoint not working for custom domain

**Error:**
```
Error: failed to connect to API endpoint
```

**Solution:**
Manually specify the API endpoint:

```yaml
accounts:
  my-account:
    platform: gitlab
    domain: gitlab.custom.com
    api_endpoint: https://gitlab.custom.com/api/v4  # Explicit endpoint
```

**API Endpoint Defaults:**
- GitHub: `https://{domain}/api/v3`
- GitLab: `https://{domain}/api/v4`

---

#### Issue 5: Migrated config not working

**Symptoms:** After migration, account doesn't work

**Checklist:**
```bash
# 1. Verify config syntax
cat ~/.gitshift/config.yaml

# 2. Check required fields are present
# - platform
# - username
# - domain (for custom domains)

# 3. Test SSH connection
gitshift ssh-test <account-alias>

# 4. Verify SSH key exists
ls -la ~/.ssh/<ssh_key_path>

# 5. Restore backup if needed
cp ~/.gitshift/config.yaml.backup ~/.gitshift/config.yaml
```

---

### Getting Help

If you encounter issues not covered here:

1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for general issues
2. Search [GitHub Issues](https://github.com/techishthoughts-org/gitshift/issues)
3. Open a new issue with:
   - Platform type (GitHub, GitLab, etc.)
   - Domain (public or custom)
   - Sanitized configuration (remove sensitive data)
   - Error messages
   - Output of `gitshift list` and `gitshift ssh-test`

---

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Detailed configuration options
- [Architecture Guide](ARCHITECTURE.md) - How multi-platform support works internally
- [User Guide](USER_GUIDE.md) - Complete command reference
- [Contributing Guide](CONTRIBUTING.md) - Adding support for new platforms

---

**Last Updated:** 2025-11-12
**Version:** 0.2.0+
