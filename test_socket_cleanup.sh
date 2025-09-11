#!/bin/bash

# Test script to demonstrate SSH socket cleanup functionality
# This script simulates the socket cleanup that was added to GitPersona

echo "🧪 Testing SSH Socket Cleanup Functionality"
echo "============================================="

# Function to cleanup SSH sockets
cleanup_ssh_sockets() {
    echo "🔧 Cleaning up SSH sockets..."

    # Get home directory
    HOME_DIR="$HOME"

    # Common SSH socket locations
    SOCKET_PATHS=(
        "$HOME_DIR/.ssh/socket"
        "$HOME_DIR/.ssh/sockets"
        "$HOME_DIR/.ssh/control"
        "/tmp/ssh-*"
    )

    CLEANED_COUNT=0

    for socket_path in "${SOCKET_PATHS[@]}"; do
        if [[ "$socket_path" == *"*"* ]]; then
            # Handle glob patterns
            for match in $socket_path; do
                if [[ -e "$match" ]]; then
                    echo "  🗑️  Removing: $match"
                    rm -rf "$match" 2>/dev/null && ((CLEANED_COUNT++))
                fi
            done
        else
            # Handle specific paths
            if [[ -e "$socket_path" ]]; then
                echo "  🗑️  Removing: $socket_path"
                rm -rf "$socket_path" 2>/dev/null && ((CLEANED_COUNT++))
            fi
        fi
    done

    # Try to close existing SSH connections
    echo "  🔌 Attempting to close existing SSH connections..."
    ssh -O exit github.com 2>/dev/null || true
    ssh -O exit gitlab.com 2>/dev/null || true
    ssh -O exit bitbucket.org 2>/dev/null || true

    echo "✅ SSH socket cleanup completed (cleaned $CLEANED_COUNT items)"
}

# Function to test SSH authentication
test_ssh_auth() {
    echo "🔐 Testing SSH authentication..."

    # Test GitHub authentication
    if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
        echo "  ✅ GitHub authentication successful"
        ssh -T git@github.com 2>&1 | head -1
    else
        echo "  ❌ GitHub authentication failed"
    fi
}

# Main execution
echo ""
echo "📋 Current SSH Agent Status:"
ssh-add -l 2>/dev/null || echo "  No SSH agent running or no keys loaded"

echo ""
echo "🔍 Current SSH Socket Status:"
ls -la ~/.ssh/socket* 2>/dev/null || echo "  No SSH sockets found"

echo ""
test_ssh_auth

echo ""
cleanup_ssh_sockets

echo ""
echo "🔍 SSH Socket Status After Cleanup:"
ls -la ~/.ssh/socket* 2>/dev/null || echo "  No SSH sockets found"

echo ""
echo "🎯 This demonstrates the socket cleanup functionality that was added to GitPersona!"
echo "   The 'gitpersona switch' command now automatically cleans up SSH sockets"
echo "   to prevent authentication conflicts when switching between accounts."
