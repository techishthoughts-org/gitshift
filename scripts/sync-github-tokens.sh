#!/bin/bash

# gitshift GitHub Token Synchronization Script
# This script automatically syncs GitHub tokens between gitshift accounts and MCP servers

set -euo pipefail

# Configuration
SCRIPT_NAME="sync-github-tokens"
LOG_LEVEL="${gitshift_LOG_LEVEL:-info}"
DRY_RUN="${DRY_RUN:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    if [[ "$LOG_LEVEL" != "error" ]]; then
        echo -e "${BLUE}[INFO]${NC} $1" >&2
    fi
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

# Help function
show_help() {
    cat << EOF
$SCRIPT_NAME - Sync GitHub tokens for gitshift and MCP servers

USAGE:
    $SCRIPT_NAME [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -d, --dry-run          Show what would be done without making changes
    -v, --verbose          Enable verbose logging
    -f, --force            Force sync even if tokens appear valid
    --mcp-only             Only sync MCP server tokens
    --shell-only           Only sync shell environment

ENVIRONMENT VARIABLES:
    DRY_RUN               Set to 'true' for dry-run mode
    gitshift_LOG_LEVEL  Set log level (error, info, debug)

EXAMPLES:
    $SCRIPT_NAME                    # Normal sync
    $SCRIPT_NAME --dry-run          # Show what would be done
    $SCRIPT_NAME --force            # Force resync all tokens
    $SCRIPT_NAME --mcp-only         # Only update MCP configuration

EOF
}

# Parse command line arguments
parse_args() {
    FORCE=false
    VERBOSE=false
    MCP_ONLY=false
    SHELL_ONLY=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                LOG_LEVEL="debug"
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            --mcp-only)
                MCP_ONLY=true
                shift
                ;;
            --shell-only)
                SHELL_ONLY=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check if gitshift is available
check_gitshift() {
    if ! command -v gitshift >/dev/null 2>&1; then
        log_error "gitshift not found in PATH"
        exit 1
    fi

    log_info "gitshift found: $(which gitshift)"
}

# Check if GitHub CLI is available
check_github_cli() {
    if ! command -v gh >/dev/null 2>&1; then
        log_error "GitHub CLI (gh) not found in PATH"
        exit 1
    fi

    # Check if authenticated
    if ! gh auth status >/dev/null 2>&1; then
        log_error "GitHub CLI is not authenticated. Run 'gh auth login' first."
        exit 1
    fi

    log_info "GitHub CLI found and authenticated"
}

# Get current gitshift account
get_current_account() {
    local current_account
    current_account=$(gitshift status --current-account-only 2>/dev/null || echo "")

    if [[ -z "$current_account" ]]; then
        log_error "No current gitshift account set"
        log_info "Set an account first: gitshift switch <account>"
        exit 1
    fi

    echo "$current_account"
}

# Get GitHub token for current account
get_github_token() {
    local token
    local current_account

    # Try to get token from gitshift first
    current_account=$(get_current_account)
    token=$(gitshift github-token get "$current_account" --export 2>/dev/null | grep "export GITHUB_TOKEN" | sed 's/export GITHUB_TOKEN="//' | sed 's/"$//' || echo "")

    if [[ -n "$token" ]]; then
        echo "$token"
        return 0
    fi

    # Fallback to GitHub CLI
    log_info "No gitshift token found, falling back to GitHub CLI..."
    token=$(gh auth token 2>/dev/null || echo "")

    if [[ -z "$token" ]]; then
        log_error "Failed to get GitHub token from gitshift or CLI"
        log_info "Run 'gitshift github-token set' to store a token"
        exit 1
    fi

    echo "$token"
}

# Validate GitHub token
validate_token() {
    local token="$1"

    log_info "Validating GitHub token..."

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would validate token: ${token:0:8}..."
        return 0
    fi

    # Test token by making a simple API call
    if ! gh api user --header "Authorization: token $token" >/dev/null 2>&1; then
        log_error "GitHub token validation failed"
        return 1
    fi

    log_success "GitHub token validation passed"
    return 0
}

# Update MCP server configuration
update_mcp_config() {
    local token="$1"
    local account="$2"

    log_info "Updating MCP server configuration..."

    # Common MCP config locations
    local mcp_config_dirs=(
        "$HOME/.config/claude-code"
        "$HOME/.config/claude"
        "$HOME/.claude"
    )

    for config_dir in "${mcp_config_dirs[@]}"; do
        if [[ -d "$config_dir" ]]; then
            local env_file="$config_dir/github-token"

            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "[DRY-RUN] Would update: $env_file"
                continue
            fi

            # Create environment file
            cat > "$env_file" << EOF
# GitHub token for gitshift account: $account
# Generated at: $(date -Iseconds)
# DO NOT EDIT MANUALLY - Managed by gitshift
export GITHUB_TOKEN="$token"
export GITHUB_TOKEN_gitshift="$token"
EOF

            chmod 600 "$env_file"
            log_success "Updated MCP config: $env_file"
        fi
    done
}

# Update shell environment files
update_shell_env() {
    local token="$1"
    local account="$2"

    log_info "Updating shell environment files..."

    local shell_files=(
        "$HOME/.zshrc"
        "$HOME/.bashrc"
        "$HOME/.profile"
    )

    for shell_file in "${shell_files[@]}"; do
        if [[ -f "$shell_file" ]]; then
            # Check if gitshift token export already exists
            if ! grep -q "# gitshift GitHub token export" "$shell_file"; then
                if [[ "$DRY_RUN" == "true" ]]; then
                    log_info "[DRY-RUN] Would add gitshift export to: $shell_file"
                    continue
                fi

                # Add gitshift token export section
                cat >> "$shell_file" << 'EOF'

# gitshift GitHub token export
# This section is managed by gitshift - do not edit manually
if command -v gitshift >/dev/null 2>&1; then
  CURRENT_TOKEN=$(gitshift config get-current-token 2>/dev/null || echo '')
  if [ -n "$CURRENT_TOKEN" ]; then
    export GITHUB_TOKEN="$CURRENT_TOKEN"
    export GITHUB_TOKEN_gitshift="$CURRENT_TOKEN"
  fi
fi
EOF

                log_success "Added gitshift export to: $shell_file"
            else
                log_info "gitshift export already exists in: $shell_file"
            fi
        fi
    done
}

# Update environment variables for current session
update_current_session() {
    local token="$1"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would export GITHUB_TOKEN for current session"
        return 0
    fi

    export GITHUB_TOKEN="$token"
    export GITHUB_TOKEN_gitshift="$token"

    log_success "Updated environment variables for current session"
}

# Main synchronization function
sync_tokens() {
    local current_account
    local github_token

    log_info "Starting GitHub token synchronization..."

    # Get current account
    current_account=$(get_current_account)
    log_info "Current gitshift account: $current_account"

    # Get GitHub token
    github_token=$(get_github_token)
    log_info "Retrieved GitHub token: ${github_token:0:8}..."

    # Validate token unless forced
    if [[ "$FORCE" != "true" ]]; then
        if ! validate_token "$github_token"; then
            log_error "Token validation failed. Use --force to skip validation."
            exit 1
        fi
    fi

    # Update MCP configuration
    if [[ "$SHELL_ONLY" != "true" ]]; then
        update_mcp_config "$github_token" "$current_account"
    fi

    # Update shell environment
    if [[ "$MCP_ONLY" != "true" ]]; then
        update_shell_env "$github_token" "$current_account"
        update_current_session "$github_token"
    fi

    log_success "GitHub token synchronization completed for account: $current_account"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Script failed with exit code: $exit_code"
    fi
    exit $exit_code
}

# Main function
main() {
    # Set up error handling
    trap cleanup EXIT

    # Parse arguments
    parse_args "$@"

    # Show dry-run notice
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "DRY-RUN MODE: No changes will be made"
    fi

    # Check prerequisites
    check_gitshift
    check_github_cli

    # Run synchronization
    sync_tokens

    log_success "All operations completed successfully!"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
