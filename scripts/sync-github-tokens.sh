#!/usr/bin/env bash

# Legacy entry point retained for compatibility.
#
# This script deliberately does not read, print, copy, validate, or persist
# GitHub tokens. GitHub CLI owns credentials in the system credential store;
# this helper only selects an already-authenticated account for a host.

set -euo pipefail

usage() {
    cat <<'EOF'
Usage: sync-github-tokens.sh --hostname HOST --user USER

Select an existing GitHub CLI account without exporting or copying its token.
Authenticate accounts first with:

  gh auth login --hostname HOST

Options:
  --hostname HOST   GitHub.com or a GitHub Enterprise hostname
  --user USER       Authenticated GitHub login to activate
  -h, --help        Show this help

Despite its historical filename, this command never handles plaintext tokens.
EOF
}

hostname=""
user=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --hostname)
            [[ $# -ge 2 ]] || { printf 'error: --hostname requires a value\n' >&2; exit 2; }
            hostname=$2
            shift 2
            ;;
        --user)
            [[ $# -ge 2 ]] || { printf 'error: --user requires a value\n' >&2; exit 2; }
            user=$2
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            printf 'error: unknown option: %s\n' "$1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

[[ -n "$hostname" ]] || { printf 'error: --hostname is required\n' >&2; exit 2; }
[[ -n "$user" ]] || { printf 'error: --user is required\n' >&2; exit 2; }

command -v gh >/dev/null 2>&1 || {
    printf 'error: GitHub CLI (gh) is not installed\n' >&2
    exit 1
}

gh auth switch --hostname "$hostname" --user "$user"
gh auth status --active --hostname "$hostname"
