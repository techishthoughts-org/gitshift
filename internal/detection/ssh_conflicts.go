package detection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/models"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHConflictDetector detects and resolves SSH key conflicts
type SSHConflictDetector struct {
	logger observability.Logger
}

// ConflictType represents different types of SSH conflicts
type ConflictType string

const (
	ConflictMultipleKeys  ConflictType = "multiple_keys"
	ConflictWrongKey      ConflictType = "wrong_key"
	ConflictPermissions   ConflictType = "permissions"
	ConflictSocketIssues  ConflictType = "socket_issues"
	ConflictAgentDead     ConflictType = "agent_dead"
	ConflictKeyMismatch   ConflictType = "key_mismatch"
	ConflictHostKeyIssues ConflictType = "host_key_issues"
)

// SSHConflict represents a detected SSH conflict
type SSHConflict struct {
	Type        ConflictType
	Severity    string
	Description string
	AffectedKey string
	Resolution  string
	AutoFix     bool
	Metadata    map[string]interface{}
}

// DetectionResult contains the results of SSH conflict detection
type DetectionResult struct {
	Conflicts       []SSHConflict
	Recommendations []string
	SystemHealth    string
	TotalIssues     int
}

// NewSSHConflictDetector creates a new SSH conflict detector
func NewSSHConflictDetector(logger observability.Logger) *SSHConflictDetector {
	return &SSHConflictDetector{
		logger: logger,
	}
}

// DetectConflicts performs comprehensive SSH conflict detection
func (d *SSHConflictDetector) DetectConflicts(ctx context.Context, accounts []*models.Account) (*DetectionResult, error) {
	d.logger.Info(ctx, "starting_ssh_conflict_detection",
		observability.F("account_count", len(accounts)),
	)

	result := &DetectionResult{
		Conflicts:       make([]SSHConflict, 0),
		Recommendations: make([]string, 0),
	}

	// Detect multiple keys in agent
	if conflicts := d.detectMultipleKeysInAgent(ctx); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect wrong key authentication
	if conflicts := d.detectWrongKeyAuthentication(ctx, accounts); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect permission issues
	if conflicts := d.detectPermissionIssues(ctx, accounts); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect socket issues
	if conflicts := d.detectSocketIssues(ctx); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect dead SSH agents
	if conflicts := d.detectDeadAgents(ctx); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect key mismatches
	if conflicts := d.detectKeyMismatches(ctx, accounts); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Detect host key issues
	if conflicts := d.detectHostKeyIssues(ctx); len(conflicts) > 0 {
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	// Generate recommendations
	result.Recommendations = d.generateRecommendations(result.Conflicts)
	result.TotalIssues = len(result.Conflicts)
	result.SystemHealth = d.assessSystemHealth(result.Conflicts)

	d.logger.Info(ctx, "ssh_conflict_detection_completed",
		observability.F("total_conflicts", result.TotalIssues),
		observability.F("system_health", result.SystemHealth),
	)

	return result, nil
}

// detectMultipleKeysInAgent detects multiple SSH keys loaded in agent
func (d *SSHConflictDetector) detectMultipleKeysInAgent(ctx context.Context) []SSHConflict {
	d.logger.Debug(ctx, "detecting_multiple_keys_in_agent")

	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "The agent has no identities." {
		return nil
	}

	lines := strings.Split(outputStr, "\\n")
	keyCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "The agent has no identities") {
			keyCount++
		}
	}

	if keyCount > 1 {
		return []SSHConflict{
			{
				Type:        ConflictMultipleKeys,
				Severity:    "warning",
				Description: fmt.Sprintf("Multiple SSH keys loaded in agent (%d keys)", keyCount),
				Resolution:  "Clear agent and load only required key",
				AutoFix:     true,
				Metadata: map[string]interface{}{
					"key_count": keyCount,
					"keys":      lines,
				},
			},
		}
	}

	return nil
}

// detectWrongKeyAuthentication detects wrong key authentication issues
func (d *SSHConflictDetector) detectWrongKeyAuthentication(ctx context.Context, accounts []*models.Account) []SSHConflict {
	d.logger.Debug(ctx, "detecting_wrong_key_authentication")

	conflicts := make([]SSHConflict, 0)

	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		// Test SSH connection with specific key
		cmd := exec.Command("ssh", "-T", "git@github.com", "-i", account.SSHKeyPath, "-o", "IdentitiesOnly=yes", "-o", "ConnectTimeout=5")
		output, _ := cmd.CombinedOutput()

		outputStr := string(output)

		// Check for authentication success but wrong user
		if strings.Contains(outputStr, "Hi ") {
			// Extract authenticated username
			re := regexp.MustCompile(`Hi ([^!]+)!`)
			matches := re.FindStringSubmatch(outputStr)
			if len(matches) > 1 {
				authenticatedUser := matches[1]
				if authenticatedUser != account.GitHubUsername {
					conflicts = append(conflicts, SSHConflict{
						Type:        ConflictWrongKey,
						Severity:    "critical",
						Description: fmt.Sprintf("Key %s authenticates as %s but account expects %s", account.SSHKeyPath, authenticatedUser, account.GitHubUsername),
						AffectedKey: account.SSHKeyPath,
						Resolution:  "Update SSH key or account configuration",
						AutoFix:     false,
						Metadata: map[string]interface{}{
							"expected_user":      account.GitHubUsername,
							"authenticated_user": authenticatedUser,
							"account_alias":      account.Alias,
						},
					})
				}
			}
		} else if strings.Contains(outputStr, "Permission denied") {
			conflicts = append(conflicts, SSHConflict{
				Type:        ConflictPermissions,
				Severity:    "error",
				Description: fmt.Sprintf("Permission denied for key %s", account.SSHKeyPath),
				AffectedKey: account.SSHKeyPath,
				Resolution:  "Check key permissions and GitHub account setup",
				AutoFix:     true,
				Metadata: map[string]interface{}{
					"account_alias": account.Alias,
					"output":        outputStr,
				},
			})
		}
	}

	return conflicts
}

// detectPermissionIssues detects SSH file permission issues
func (d *SSHConflictDetector) detectPermissionIssues(ctx context.Context, accounts []*models.Account) []SSHConflict {
	d.logger.Debug(ctx, "detecting_permission_issues")

	conflicts := make([]SSHConflict, 0)

	// Check SSH directory permissions
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return conflicts
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil {
		perm := info.Mode().Perm()
		if perm&0077 != 0 {
			conflicts = append(conflicts, SSHConflict{
				Type:        ConflictPermissions,
				Severity:    "warning",
				Description: fmt.Sprintf("SSH directory has overly permissive permissions: %o", perm),
				Resolution:  "Set directory permissions to 700",
				AutoFix:     true,
				Metadata: map[string]interface{}{
					"directory":     sshDir,
					"current_perm":  fmt.Sprintf("%o", perm),
					"expected_perm": "700",
				},
			})
		}
	}

	// Check individual key file permissions
	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		if info, err := os.Stat(account.SSHKeyPath); err == nil {
			perm := info.Mode().Perm()
			if perm&0077 != 0 {
				conflicts = append(conflicts, SSHConflict{
					Type:        ConflictPermissions,
					Severity:    "error",
					Description: fmt.Sprintf("SSH key %s has overly permissive permissions: %o", account.SSHKeyPath, perm),
					AffectedKey: account.SSHKeyPath,
					Resolution:  "Set key permissions to 600",
					AutoFix:     true,
					Metadata: map[string]interface{}{
						"account_alias": account.Alias,
						"current_perm":  fmt.Sprintf("%o", perm),
						"expected_perm": "600",
					},
				})
			}
		}
	}

	return conflicts
}

// detectSocketIssues detects SSH socket-related issues
func (d *SSHConflictDetector) detectSocketIssues(ctx context.Context) []SSHConflict {
	d.logger.Debug(ctx, "detecting_socket_issues")

	conflicts := make([]SSHConflict, 0)

	// Check if SSH_AUTH_SOCK is set
	authSocket := os.Getenv("SSH_AUTH_SOCK")
	if authSocket == "" {
		conflicts = append(conflicts, SSHConflict{
			Type:        ConflictSocketIssues,
			Severity:    "error",
			Description: "SSH_AUTH_SOCK environment variable not set",
			Resolution:  "Start SSH agent or set SSH_AUTH_SOCK",
			AutoFix:     true,
		})
		return conflicts
	}

	// Check if socket file exists
	if _, err := os.Stat(authSocket); os.IsNotExist(err) {
		conflicts = append(conflicts, SSHConflict{
			Type:        ConflictSocketIssues,
			Severity:    "error",
			Description: fmt.Sprintf("SSH agent socket does not exist: %s", authSocket),
			Resolution:  "Restart SSH agent",
			AutoFix:     true,
			Metadata: map[string]interface{}{
				"socket_path": authSocket,
			},
		})
	}

	// Check for stale socket files
	homeDir, _ := os.UserHomeDir()
	socketDirs := []string{
		filepath.Join(homeDir, ".ssh", "socket"),
		filepath.Join(homeDir, ".ssh", "sockets"),
		"/tmp",
	}

	for _, dir := range socketDirs {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "ssh-") && entry.Name() != filepath.Base(authSocket) {
					conflicts = append(conflicts, SSHConflict{
						Type:        ConflictSocketIssues,
						Severity:    "warning",
						Description: fmt.Sprintf("Stale SSH socket found: %s", filepath.Join(dir, entry.Name())),
						Resolution:  "Clean up stale socket files",
						AutoFix:     true,
						Metadata: map[string]interface{}{
							"stale_socket": filepath.Join(dir, entry.Name()),
						},
					})
				}
			}
		}
	}

	return conflicts
}

// detectDeadAgents detects dead or unresponsive SSH agents
func (d *SSHConflictDetector) detectDeadAgents(ctx context.Context) []SSHConflict {
	d.logger.Debug(ctx, "detecting_dead_agents")

	conflicts := make([]SSHConflict, 0)

	// Test SSH agent responsiveness
	cmd := exec.Command("ssh-add", "-l")
	cmd.Env = os.Environ()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			conflicts = append(conflicts, SSHConflict{
				Type:        ConflictAgentDead,
				Severity:    "error",
				Description: "SSH agent is not responding",
				Resolution:  "Restart SSH agent",
				AutoFix:     true,
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
	case <-time.After(5 * time.Second):
		// Kill the command if it's hanging
		cmd.Process.Kill()
		conflicts = append(conflicts, SSHConflict{
			Type:        ConflictAgentDead,
			Severity:    "error",
			Description: "SSH agent is hanging/unresponsive",
			Resolution:  "Kill and restart SSH agent",
			AutoFix:     true,
		})
	}

	return conflicts
}

// detectKeyMismatches detects mismatches between configured and actual keys
func (d *SSHConflictDetector) detectKeyMismatches(ctx context.Context, accounts []*models.Account) []SSHConflict {
	d.logger.Debug(ctx, "detecting_key_mismatches")

	conflicts := make([]SSHConflict, 0)

	for _, account := range accounts {
		if account.SSHKeyPath == "" {
			continue
		}

		// Check if configured key file exists
		if _, err := os.Stat(account.SSHKeyPath); os.IsNotExist(err) {
			conflicts = append(conflicts, SSHConflict{
				Type:        ConflictKeyMismatch,
				Severity:    "error",
				Description: fmt.Sprintf("Configured SSH key does not exist: %s", account.SSHKeyPath),
				AffectedKey: account.SSHKeyPath,
				Resolution:  "Update key path or generate new key",
				AutoFix:     false,
				Metadata: map[string]interface{}{
					"account_alias": account.Alias,
				},
			})
		}

		// Check if public key exists
		pubKeyPath := account.SSHKeyPath + ".pub"
		if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
			conflicts = append(conflicts, SSHConflict{
				Type:        ConflictKeyMismatch,
				Severity:    "warning",
				Description: fmt.Sprintf("Public key missing: %s", pubKeyPath),
				AffectedKey: account.SSHKeyPath,
				Resolution:  "Generate public key from private key",
				AutoFix:     true,
				Metadata: map[string]interface{}{
					"account_alias":   account.Alias,
					"public_key_path": pubKeyPath,
				},
			})
		}
	}

	return conflicts
}

// detectHostKeyIssues detects SSH host key issues
func (d *SSHConflictDetector) detectHostKeyIssues(ctx context.Context) []SSHConflict {
	d.logger.Debug(ctx, "detecting_host_key_issues")

	conflicts := make([]SSHConflict, 0)

	// Check known_hosts file
	homeDir, _ := os.UserHomeDir()
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		conflicts = append(conflicts, SSHConflict{
			Type:        ConflictHostKeyIssues,
			Severity:    "info",
			Description: "SSH known_hosts file does not exist",
			Resolution:  "Accept host keys on first connection",
			AutoFix:     false,
		})
	} else {
		// Check for GitHub host keys
		content, err := os.ReadFile(knownHostsPath)
		if err == nil {
			contentStr := string(content)
			if !strings.Contains(contentStr, "github.com") {
				conflicts = append(conflicts, SSHConflict{
					Type:        ConflictHostKeyIssues,
					Severity:    "info",
					Description: "GitHub host key not found in known_hosts",
					Resolution:  "Connect to GitHub to accept host key",
					AutoFix:     true,
				})
			}
		}
	}

	return conflicts
}

// generateRecommendations generates actionable recommendations
func (d *SSHConflictDetector) generateRecommendations(conflicts []SSHConflict) []string {
	recommendations := make([]string, 0)

	hasMultipleKeys := false
	hasPermissionIssues := false
	hasSocketIssues := false

	for _, conflict := range conflicts {
		switch conflict.Type {
		case ConflictMultipleKeys:
			hasMultipleKeys = true
		case ConflictPermissions:
			hasPermissionIssues = true
		case ConflictSocketIssues:
			hasSocketIssues = true
		}
	}

	if hasMultipleKeys {
		recommendations = append(recommendations, "Run 'gitpersona ssh-agent --cleanup' to resolve multiple key conflicts")
	}

	if hasPermissionIssues {
		recommendations = append(recommendations, "Run 'chmod 700 ~/.ssh && chmod 600 ~/.ssh/*' to fix permission issues")
	}

	if hasSocketIssues {
		recommendations = append(recommendations, "Restart SSH agent with 'eval $(ssh-agent)' and reload keys")
	}

	if len(conflicts) > 3 {
		recommendations = append(recommendations, "Consider running 'gitpersona diagnose --fix' for comprehensive auto-repair")
	}

	return recommendations
}

// assessSystemHealth provides an overall health assessment
func (d *SSHConflictDetector) assessSystemHealth(conflicts []SSHConflict) string {
	if len(conflicts) == 0 {
		return "excellent"
	}

	criticalCount := 0
	errorCount := 0

	for _, conflict := range conflicts {
		switch conflict.Severity {
		case "critical":
			criticalCount++
		case "error":
			errorCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	} else if errorCount > 2 {
		return "poor"
	} else if len(conflicts) > 5 {
		return "fair"
	} else {
		return "good"
	}
}

// AutoFixConflicts automatically fixes conflicts that can be resolved
func (d *SSHConflictDetector) AutoFixConflicts(ctx context.Context, conflicts []SSHConflict) error {
	d.logger.Info(ctx, "starting_auto_fix_for_conflicts",
		observability.F("conflict_count", len(conflicts)),
	)

	for _, conflict := range conflicts {
		if !conflict.AutoFix {
			continue
		}

		d.logger.Info(ctx, "auto_fixing_conflict",
			observability.F("type", string(conflict.Type)),
			observability.F("description", conflict.Description),
		)

		switch conflict.Type {
		case ConflictMultipleKeys:
			if err := d.fixMultipleKeys(ctx); err != nil {
				d.logger.Error(ctx, "failed_to_fix_multiple_keys",
					observability.F("error", err.Error()),
				)
			}
		case ConflictPermissions:
			if err := d.fixPermissions(ctx, conflict); err != nil {
				d.logger.Error(ctx, "failed_to_fix_permissions",
					observability.F("error", err.Error()),
				)
			}
		case ConflictSocketIssues:
			if err := d.fixSocketIssues(ctx, conflict); err != nil {
				d.logger.Error(ctx, "failed_to_fix_socket_issues",
					observability.F("error", err.Error()),
				)
			}
		}
	}

	return nil
}

// Auto-fix implementation methods
func (d *SSHConflictDetector) fixMultipleKeys(ctx context.Context) error {
	cmd := exec.Command("ssh-add", "-D")
	return cmd.Run()
}

func (d *SSHConflictDetector) fixPermissions(ctx context.Context, conflict SSHConflict) error {
	if conflict.AffectedKey != "" {
		return os.Chmod(conflict.AffectedKey, 0600)
	}

	// Fix SSH directory permissions
	homeDir, _ := os.UserHomeDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	return os.Chmod(sshDir, 0700)
}

func (d *SSHConflictDetector) fixSocketIssues(ctx context.Context, conflict SSHConflict) error {
	if staleSocket, ok := conflict.Metadata["stale_socket"]; ok {
		if socketPath, ok := staleSocket.(string); ok {
			return os.Remove(socketPath)
		}
	}
	return nil
}
