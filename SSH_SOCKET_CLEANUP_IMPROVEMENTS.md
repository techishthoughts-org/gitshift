# SSH Socket Cleanup Improvements for GitPersona

## Overview

This document describes the improvements made to GitPersona to automatically handle SSH socket cleanup when switching between GitHub accounts, preventing authentication conflicts.

## Problem Solved

Previously, when switching between GitHub accounts using GitPersona, SSH authentication conflicts could occur due to:
- Existing SSH control sockets maintaining connections to the wrong account
- SSH multiplexing connections persisting across account switches
- Socket files in `~/.ssh/socket/` and `~/.ssh/sockets/` causing authentication issues

## Solution Implemented

### 1. Enhanced SSH Agent Service

**File**: `internal/services/ssh_agent_service.go`

Added new methods to the `RealSSHAgentService`:

- `CleanupSSHSockets(ctx context.Context) error` - Main cleanup function
- `cleanupSocketFile(ctx context.Context, socketPath string) error` - Removes specific socket files
- `closeExistingSSHConnections(ctx context.Context) error` - Closes active SSH connections
- `SwitchToAccountWithCleanup(ctx context.Context, keyPath string) error` - Enhanced switch with cleanup

### 2. Updated Interface

**File**: `internal/services/interfaces.go`

Extended the `SSHAgentService` interface to include:
- `SwitchToAccountWithCleanup(ctx context.Context, keyPath string) error`
- `CleanupSSHSockets(ctx context.Context) error`

### 3. Enhanced Switch Command

**File**: `cmd/switch.go`

Modified the `manageSSHAgent` method to use `SwitchToAccountWithCleanup` instead of `SwitchToAccount`, ensuring socket cleanup happens automatically during account switches.

### 4. New SSH Agent Command

**File**: `cmd/ssh-agent.go`

Added a new `--cleanup` flag to the SSH agent command:
- `gitpersona ssh-agent --cleanup` - Manually clean up SSH sockets
- Updated help text and examples

## Socket Cleanup Process

The cleanup process handles:

1. **Socket File Removal**:
   - `~/.ssh/socket/` directory and contents
   - `~/.ssh/sockets/` directory and contents
   - `~/.ssh/control/` directory and contents
   - `/tmp/ssh-*` glob pattern matches

2. **Connection Cleanup**:
   - Attempts to close existing SSH connections using `ssh -O exit`
   - Targets common Git hosts: github.com, gitlab.com, bitbucket.org

3. **Error Handling**:
   - Graceful handling of missing files/directories
   - Continues operation even if cleanup partially fails
   - Comprehensive logging of cleanup operations

## Usage

### Automatic Cleanup (Recommended)

The socket cleanup now happens automatically when switching accounts:

```bash
gitpersona switch personal    # Automatically cleans up sockets
gitpersona switch work        # Automatically cleans up sockets
```

### Manual Cleanup

You can also manually clean up SSH sockets:

```bash
gitpersona ssh-agent --cleanup
```

### SSH Agent Management

Enhanced SSH agent commands:

```bash
gitpersona ssh-agent --status    # Show current status
gitpersona ssh-agent --clear     # Clear all keys
gitpersona ssh-agent --cleanup   # Clean up sockets
gitpersona ssh-agent --load ~/.ssh/id_ed25519_costaar7  # Load specific key
```

## Testing

A test script was created to demonstrate the functionality:

```bash
./test_socket_cleanup.sh
```

This script shows:
- Current SSH agent status
- Existing socket files
- Authentication test results
- Socket cleanup process
- Post-cleanup verification

## Benefits

1. **Eliminates Authentication Conflicts**: No more issues with wrong accounts being authenticated
2. **Automatic Operation**: Cleanup happens transparently during account switches
3. **Manual Override**: Users can manually clean up sockets if needed
4. **Comprehensive Coverage**: Handles all common SSH socket locations
5. **Robust Error Handling**: Continues operation even if cleanup fails
6. **Better User Experience**: Seamless account switching without manual intervention

## Technical Details

### Socket Locations Cleaned

- `~/.ssh/socket/` - SSH control master sockets
- `~/.ssh/sockets/` - Additional socket directory
- `~/.ssh/control/` - Control socket directory
- `/tmp/ssh-*` - Temporary SSH sockets

### Connection Cleanup

Uses `ssh -O exit <host>` to gracefully close existing connections to:
- github.com
- gitlab.com
- bitbucket.org

### Error Handling

- Non-blocking: Cleanup failures don't prevent account switching
- Logging: All operations are logged for debugging
- Graceful degradation: Continues with partial cleanup if some operations fail

## Future Enhancements

Potential future improvements:
1. Configurable socket cleanup paths
2. Selective cleanup based on account configuration
3. Integration with SSH config file parsing
4. Support for custom SSH hosts
5. Cleanup scheduling/automation options

## Conclusion

This improvement significantly enhances GitPersona's reliability when switching between GitHub accounts by automatically handling SSH socket cleanup, eliminating a common source of authentication conflicts.
