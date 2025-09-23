# GitPersona Token Management

GitPersona now includes a complete, self-contained GitHub token management system that eliminates the dependency on GitHub CLI for MCP server authentication.

## Overview

The new token management system provides:

- **Secure Token Storage**: Encrypted local storage of GitHub tokens
- **Multi-Account Support**: Store and manage tokens for multiple GitHub accounts
- **Environment Integration**: Automatic configuration of MCP servers and shell environment
- **Self-Contained Operation**: No dependency on external tools like GitHub CLI

## Key Components

### 1. Token Storage Service
- Encrypts and stores GitHub tokens locally
- Uses AES-256 encryption with unique keys per installation
- Stores tokens in `~/.config/gitpersona/tokens/`

### 2. Environment Service
- Manages MCP server configuration
- Updates shell environment files
- Ensures tokens are available to all GitPersona components

### 3. Integrated Commands
- `gitpersona github-token`: Manage GitHub tokens
- `gitpersona environment`: Configure environment integration

## Quick Start

### 1. Store a GitHub Token

```bash
# Interactive token entry (recommended)
gitpersona github-token set --interactive

# From environment variable
export GITHUB_TOKEN="your_token_here"
gitpersona github-token set --from-env

# Direct token (less secure)
gitpersona github-token set --token "your_token_here"
```

### 2. Set Up Environment

```bash
# Configure both MCP servers and shell environment
gitpersona environment setup

# Validate the setup
gitpersona environment validate
```

### 3. Restart Claude Code

After setting up tokens and environment, restart Claude Code to reload MCP servers with the new configuration.

## Command Reference

### GitHub Token Commands

#### Set Token
```bash
# Set token for current account
gitpersona github-token set --interactive

# Set token for specific account
gitpersona github-token set myaccount --interactive

# Import from GitHub CLI (one-time migration)
gitpersona github-token import-from-cli
```

#### Get Token
```bash
# Show masked token
gitpersona github-token get

# Show full token (be careful!)
gitpersona github-token get --show

# Export as environment variable
gitpersona github-token get --export
```

#### List Tokens
```bash
# Show all stored tokens
gitpersona github-token list
```

#### Validate Token
```bash
# Validate current account's token
gitpersona github-token validate

# Validate specific account's token
gitpersona github-token validate myaccount
```

#### Delete Token
```bash
# Delete token for specific account
gitpersona github-token delete myaccount
```

### Environment Commands

#### Setup
```bash
# Full environment setup
gitpersona environment setup

# MCP servers only
gitpersona environment setup --mcp-only

# Shell environment only
gitpersona environment setup --shell-only

# Force setup (overwrite existing)
gitpersona environment setup --force
```

#### Validate
```bash
# Comprehensive environment validation
gitpersona environment validate

# Quick status check
gitpersona environment status
```

#### Cleanup
```bash
# Clean up old configuration files
gitpersona environment cleanup
```

## File Locations

### Token Storage
- **Location**: `~/.config/gitpersona/tokens/`
- **Files**: `{account}.json` (encrypted token files)
- **Encryption Key**: `.encryption_key` (automatically generated)

### Environment Configuration
- **GitPersona Environment**: `~/.config/gitpersona/environment`
- **MCP Configuration**: `~/.config/gitpersona/mcp/`
- **Shell Integration**: Added to `~/.zshrc`, `~/.bashrc`, `~/.profile`

### MCP Server Files
- **Claude Code**: `~/.config/claude-code/github-token.env`
- **Claude**: `~/.config/claude/github-token.env`
- **GitPersona MCP**: `~/.config/gitpersona/mcp/github-token.env`

## Security Features

### Token Encryption
- **Algorithm**: AES-256-GCM
- **Key Storage**: Local encryption key (not transmitted)
- **File Permissions**: 600 (owner read/write only)

### Safe Practices
- Tokens are never logged in plaintext
- Environment files have secure permissions
- Token prefixes shown for identification (e.g., `ghp_****`)

## Migration from GitHub CLI

If you're currently using GitHub CLI for authentication:

```bash
# 1. Import existing token
gitpersona github-token import-from-cli

# 2. Set up environment
gitpersona environment setup

# 3. Validate everything works
gitpersona environment validate

# 4. Restart Claude Code
```

## Troubleshooting

### Token Issues

**Problem**: Token validation fails
```bash
# Check token format and permissions
gitpersona github-token validate

# Re-import from CLI if needed
gitpersona github-token import-from-cli
```

**Problem**: MCP servers can't access GitHub
```bash
# Validate environment setup
gitpersona environment validate

# Restart Claude Code after any changes
```

### Environment Issues

**Problem**: Shell doesn't have GitHub token
```bash
# Reload GitPersona environment
source ~/.config/gitpersona/environment

# Or restart your shell
```

**Problem**: MCP configuration missing
```bash
# Re-setup environment
gitpersona environment setup --force

# Restart Claude Code
```

## Advanced Usage

### Multiple Accounts

```bash
# Store tokens for different accounts
gitpersona github-token set work --interactive
gitpersona github-token set personal --interactive

# Switch environment to specific account
gitpersona environment setup --account work
```

### Custom Export

```bash
# Export token to custom script
gitpersona github-token get --export > my-github-env.sh
source my-github-env.sh
```

### Environment Variables

After setup, these variables are available:

- `GITHUB_TOKEN`: Primary GitHub token
- `GITHUB_PERSONAL_ACCESS_TOKEN`: Same as GITHUB_TOKEN
- `GITPERSONA_CURRENT_ACCOUNT`: Current account name
- `GITPERSONA_GITHUB_TOKEN`: GitPersona-managed token

## Integration with Other Tools

### MCP Servers
GitPersona automatically configures MCP servers to use stored tokens:

```json
{
  "env": {
    "GITHUB_TOKEN": "your_encrypted_token",
    "GITHUB_PERSONAL_ACCESS_TOKEN": "your_encrypted_token"
  }
}
```

### Shell Scripts
Use GitPersona tokens in your scripts:

```bash
#!/bin/bash
source ~/.config/gitpersona/environment

# Now GITHUB_TOKEN is available
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
```

## Best Practices

1. **Use Interactive Mode**: Always use `--interactive` for secure token entry
2. **Validate Setup**: Run `gitpersona environment validate` after changes
3. **Regular Cleanup**: Periodically run `gitpersona environment cleanup`
4. **Restart Claude Code**: Always restart after token/environment changes
5. **Secure Storage**: Never commit token files to version control

## Support

For issues with token management:

1. Check token validity: `gitpersona github-token validate`
2. Validate environment: `gitpersona environment validate`
3. Check logs in GitPersona's debug output
4. Re-run setup: `gitpersona environment setup --force`

The new token management system provides a robust, secure, and self-contained way to handle GitHub authentication across all GitPersona components and MCP servers.
