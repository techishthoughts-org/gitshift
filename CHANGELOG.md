# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **GPG Key Discovery**: Automatic discovery of GPG signing keys from system keyring
  - Scans GPG keyring for secret keys with signing capability
  - Merges SSH and GPG key information by email address
  - Supports RSA, DSA, and EdDSA key types
  - Displays GPG key information in discovery output
- **Enhanced Platform Detection**: Improved automatic platform detection
  - Corporate email addresses default to GitLab
  - Personal email domains (gmail, yahoo, etc.) default to GitHub
  - Domain-based platform detection (gitlab.*, github.* in domain)
- **SSH Key Switching**: Fixed SSH key isolation when switching accounts
  - Properly sets `core.sshCommand` with account-specific SSH key
  - Uses `IdentitiesOnly=yes` for complete SSH isolation
  - Updates both global and local Git configuration

### Fixed
- **Switch Command Early Exit Bug**: Fixed premature exit when switching accounts
  - Removed early exit check that skipped configuration updates
  - Now always applies SSH and Git configuration when switching accounts
  - Ensures system configuration stays in sync with gitshift config
  - Fixes issue where `gitshift switch` would report success but not update SSH/Git configs
- **Platform Detection in List Command**: Fixed account categorization by platform
  - Accounts are now properly grouped by platform (GitHub, GitLab, etc.)
  - Platform field is now set correctly during account discovery
  - List command displays accounts under correct platform sections
  - Fixed issue where all accounts showed under "GitHub" regardless of actual platform
- **Configuration Persistence**: Fixed account alias corruption for aliases with dots
  - Replaced Viper-based config I/O with direct yaml.v3 marshaling
  - Properly handles account aliases containing special characters
- **SSH Key Path Configuration**: Account switching now correctly updates SSH command
  - Fixed bug where Git would use wrong SSH key after account switch
  - Ensures proper SSH key isolation per account
- **Make Install Path**: Changed install target from `/usr/local/bin` to `$GOPATH/bin`
  - No longer requires sudo for installation
  - Follows Go ecosystem conventions
- **GitHub Username Validation**: SSH scanner validates GitHub username format
  - Only sets GitHubUsername if it matches GitHub's format requirements
  - Prevents invalid usernames from SSH key filenames

### Changed
- Updated all documentation to reflect multi-platform support
- Enhanced documentation structure with new guides
- Configuration now uses direct YAML marshaling for better reliability

## [0.2.0] - 2025-11-12

### Added

#### Multi-Platform Support 🚀
- **GitHub Support**: Full support for GitHub.com and GitHub Enterprise
- **GitLab Support**: Full support for GitLab.com and self-hosted GitLab instances
- **Platform Abstraction Layer**: Clean interface for adding new platforms in the future
- **Custom Domain Support**: Use custom domains for GitHub Enterprise and self-hosted GitLab
- **Platform Auto-Detection**: Automatically detect platform from repository URLs
- **API Endpoint Configuration**: Custom API endpoints for enterprise/self-hosted instances

#### New Configuration Fields
- `platform` - Specify the Git hosting platform (github, gitlab)
- `username` - Platform-specific username (replaces `github_username`)
- `domain` - Custom domain for enterprise/self-hosted instances
- `api_endpoint` - Custom API endpoint URL

#### Platform-Specific Features
- Platform-aware SSH configuration generation
- Platform-specific SSH host entries
- Automatic platform detection from Git URLs
- Platform-based account grouping in `list` command

#### Documentation
- Added comprehensive [Multi-Platform Support Guide](docs/MULTI_PLATFORM_SUPPORT.md)
- Added migration guide from GitHub-only to multi-platform
- Added platform-specific examples in configuration guide
- Updated all documentation to reflect multi-platform capabilities

### Changed
- **Account Model**: Extended to support multiple platforms
- **SSH Configuration**: Enhanced to support platform-specific settings
- **CLI Output**: Shows platform information in `list` and `current` commands
- **Repository Detection**: Enhanced URL parsing for multi-platform support

### Deprecated
- `github_username` field (use `username` + `platform` instead)
  - Note: Still fully supported for backward compatibility

### Backward Compatibility
- ✅ Existing GitHub-only configurations work without changes
- ✅ Accounts without `platform` field default to GitHub
- ✅ Old `github_username` field still supported
- ✅ All existing CLI commands work as before

## [0.1.2] - 2025-11-11

### Security
- **Go Version Upgrade**: Updated from Go 1.23 to Go 1.24.0
  - Resolves stdlib security vulnerabilities
  - Updated in `go.mod` and all CI/CD workflows
  - See [CVE details](https://go.dev/security/) for specific vulnerabilities addressed

### Fixed
- Security vulnerabilities in Go standard library (via Go 1.24.0 upgrade)

## [0.1.1] - 2025-10-15

### Added
- Initial release of gitshift
- SSH-first Git account management
- Multiple GitHub account support
- SSH key generation and management
- Account switching with validation
- Auto-discovery of Git accounts
- GitHub CLI integration
- Token management for GitHub
- Automatic `GITHUB_TOKEN` updates in zsh secrets
- Comprehensive documentation

### Features

#### Core Commands
- `gitshift add` - Add new Git accounts
- `gitshift switch` - Switch between accounts
- `gitshift list` - List all configured accounts
- `gitshift current` - Show current active account
- `gitshift ssh-keygen` - Generate SSH keys
- `gitshift ssh-test` - Test SSH connections
- `gitshift discover` - Auto-discover Git accounts
- `gitshift remove` - Remove accounts
- `gitshift update` - Update account information

#### SSH Management
- Ed25519 SSH key generation (default)
- RSA SSH key support
- Complete SSH key isolation per account using `IdentitiesOnly=yes`
- Automatic SSH config generation
- SSH connection testing and validation

#### GitHub Integration
- GitHub CLI (`gh`) integration
- Automatic token management
- `GITHUB_TOKEN` updates in `~/.zsh_secrets`
- GitHub API support for account validation

#### Configuration
- YAML-based configuration (`~/.gitshift/config.yaml`)
- Support for multiple isolation levels (basic, standard, strict)
- Default account configuration
- Global Git config integration
- Auto-detect mode for repository-based account selection

#### Documentation
- Comprehensive user guide
- Configuration reference
- Architecture documentation
- Security best practices guide
- Troubleshooting guide
- Migration guide from other tools
- Contribution guidelines

---

## Version Comparison

### Key Differences Between Versions

| Feature | v0.1.1 | v0.1.2 | v0.2.0 |
|---------|--------|--------|--------|
| **Platform Support** | GitHub only | GitHub only | GitHub + GitLab + Enterprise |
| **Go Version** | 1.23+ | 1.24.0 | 1.24.0 |
| **Security Patches** | - | ✅ Stdlib fixes | ✅ Stdlib fixes |
| **Custom Domains** | ❌ | ❌ | ✅ Full support |
| **Platform Auto-Detection** | N/A | N/A | ✅ Automatic |
| **API Endpoints** | GitHub.com only | GitHub.com only | ✅ Configurable |
| **Backward Compatible** | N/A | ✅ | ✅ |

---

## Upgrade Guide

### Upgrading to v0.2.0 from v0.1.x

**No action required** - Your existing configuration will continue to work!

**Optional improvements:**

1. **Modernize your config** (recommended for clarity):
   ```yaml
   # Old format (still works)
   github_username: johndoe

   # New format (recommended)
   platform: github
   username: johndoe
   ```

2. **Add GitLab accounts** if needed:
   ```bash
   gitshift add my-gitlab --platform gitlab
   ```

3. **Test multi-platform features**:
   ```bash
   gitshift list  # Now shows platform column
   ```

See [Multi-Platform Support Guide](docs/MULTI_PLATFORM_SUPPORT.md) for detailed migration instructions.

### Upgrading to v0.1.2 from v0.1.1

**No configuration changes required** - Security update only.

1. Update gitshift binary
2. Verify Go version: `go version` (should show 1.24.0+)

---

## Deprecation Timeline

### Current Deprecations

| Field/Feature | Deprecated In | Alternative | Removal Planned |
|--------------|---------------|-------------|-----------------|
| `github_username` | v0.2.0 | `username` + `platform` | No removal planned (backward compat maintained) |

### Planned Changes

No breaking changes planned for the foreseeable future. gitshift maintains strong backward compatibility guarantees.

---

## Migration Guides

- **From GitHub-only to Multi-Platform**: See [Multi-Platform Support Guide](docs/MULTI_PLATFORM_SUPPORT.md#migration-from-github-only)
- **From other tools**: See [Migration Guide](docs/MIGRATION_GUIDE.md)

---

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines on:
- Reporting bugs
- Suggesting features
- Submitting pull requests
- Adding support for new platforms

---

## Links

- [Documentation](docs/)
- [GitHub Repository](https://github.com/yourusername/gitshift)
- [Issue Tracker](https://github.com/yourusername/gitshift/issues)
- [Releases](https://github.com/yourusername/gitshift/releases)

---

**Note:** Version numbers and dates reflect major milestones. See Git commit history for detailed change tracking.
