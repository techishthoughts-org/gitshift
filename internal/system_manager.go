package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// RealSystemManager implements the SystemManager interface
type RealSystemManager struct {
	logger         observability.Logger
	accountManager AccountManager
	sshManager     SSHManager
	gitManager     GitManager
	githubManager  GitHubManager
}

// NewSystemManager creates a new system manager
func NewSystemManager(logger observability.Logger) SystemManager {
	return &RealSystemManager{
		logger: logger,
	}
}

// SetDependencies injects other service dependencies
func (sm *RealSystemManager) SetDependencies(account AccountManager, ssh SSHManager, git GitManager, github GitHubManager) {
	sm.accountManager = account
	sm.sshManager = ssh
	sm.gitManager = git
	sm.githubManager = github
}

// PerformHealthCheck performs a comprehensive system health check
func (sm *RealSystemManager) PerformHealthCheck(ctx context.Context) error {
	sm.logger.Info(ctx, "performing_system_health_check")

	checks := []func(context.Context) error{
		sm.checkGitInstallation,
		sm.checkSSHInstallation,
		sm.checkConfigurationDirectory,
		sm.checkAccountConfiguration,
	}

	for i, check := range checks {
		if err := check(ctx); err != nil {
			sm.logger.Error(ctx, "health_check_failed",
				observability.F("check_index", i),
				observability.F("error", err.Error()),
			)
			return fmt.Errorf("health check failed: %w", err)
		}
	}

	sm.logger.Info(ctx, "system_health_check_passed")
	return nil
}

// GetSystemInfo returns system information
func (sm *RealSystemManager) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	sm.logger.Info(ctx, "getting_system_info")

	info := &SystemInfo{
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
		Version:  "2.0.0", // GitPersona version
	}

	// Get SSH version
	if sshVersion, err := sm.getSSHVersion(ctx); err == nil {
		info.SSHVersion = sshVersion
	}

	// Get Git version
	if gitVersion, err := sm.getGitVersion(ctx); err == nil {
		info.GitVersion = gitVersion
	}

	sm.logger.Info(ctx, "system_info_retrieved",
		observability.F("platform", info.Platform),
		observability.F("ssh_version", info.SSHVersion),
		observability.F("git_version", info.GitVersion),
	)

	return info, nil
}

// RunDiagnostics runs comprehensive system diagnostics
func (sm *RealSystemManager) RunDiagnostics(ctx context.Context) (*DiagnosticsReport, error) {
	sm.logger.Info(ctx, "running_comprehensive_diagnostics")

	report := &DiagnosticsReport{
		Overall:  "healthy",
		Checks:   []*DiagnosticCheck{},
		FixCount: 0,
	}

	// System checks
	systemChecks := []*DiagnosticCheck{
		sm.diagnoseBinaryInstallations(ctx),
		sm.diagnosePermissions(ctx),
		sm.diagnoseConfiguration(ctx),
	}

	// SSH checks
	if sm.sshManager != nil {
		if sshIssues, err := sm.sshManager.DiagnoseIssues(ctx); err == nil {
			for _, issue := range sshIssues {
				check := &DiagnosticCheck{
					Name:    fmt.Sprintf("SSH: %s", issue.Type),
					Status:  sm.severityToStatus(issue.Severity),
					Message: issue.Description,
					Fix:     issue.Fix,
				}
				systemChecks = append(systemChecks, check)
			}
		}
	}

	// Git checks
	if sm.gitManager != nil {
		if gitValidation, err := sm.gitManager.ValidateConfig(ctx); err == nil {
			for _, issue := range gitValidation.Issues {
				check := &DiagnosticCheck{
					Name:    fmt.Sprintf("Git: %s", issue.Type),
					Status:  "fail",
					Message: issue.Description,
					Fix:     issue.Fix,
				}
				systemChecks = append(systemChecks, check)
			}
		}
	}

	// Account checks
	if sm.accountManager != nil {
		if accounts, err := sm.accountManager.ListAccounts(ctx); err == nil {
			for _, account := range accounts {
				if validation, err := sm.accountManager.ValidateAccount(ctx, account.Alias); err == nil {
					status := "pass"
					if !validation.Valid {
						status = "fail"
						if report.Overall == "healthy" {
							report.Overall = "issues"
						}
					}

					check := &DiagnosticCheck{
						Name:    fmt.Sprintf("Account: %s", account.Alias),
						Status:  status,
						Message: fmt.Sprintf("Account validation: %d issues", len(validation.Issues)),
						Fix:     "Run 'gitpersona account validate " + account.Alias + " --fix'",
					}
					systemChecks = append(systemChecks, check)
				}
			}
		}
	}

	report.Checks = systemChecks

	// Determine overall status
	failCount := 0
	for _, check := range report.Checks {
		if check.Status == "fail" {
			failCount++
		}
	}

	if failCount > 0 {
		if failCount > len(report.Checks)/2 {
			report.Overall = "critical"
		} else {
			report.Overall = "issues"
		}
	}

	report.Summary = fmt.Sprintf("Completed %d diagnostic checks. %d issues found.", len(report.Checks), failCount)

	sm.logger.Info(ctx, "comprehensive_diagnostics_completed",
		observability.F("overall", report.Overall),
		observability.F("total_checks", len(report.Checks)),
		observability.F("failed_checks", failCount),
	)

	return report, nil
}

// GetTroubleshootingInfo returns troubleshooting information
func (sm *RealSystemManager) GetTroubleshootingInfo(ctx context.Context) (*TroubleshootingInfo, error) {
	sm.logger.Info(ctx, "getting_troubleshooting_info")

	info := &TroubleshootingInfo{
		CommonIssues: []string{
			"SSH key permissions incorrect (should be 600)",
			"SSH agent not running",
			"Git user.name or user.email not set",
			"GitHub token expired or invalid",
			"SSH key not uploaded to GitHub",
			"Repository using wrong SSH key",
		},
		Solutions: map[string]string{
			"ssh_permissions": "chmod 600 ~/.ssh/id_*",
			"ssh_agent":       "eval $(ssh-agent) && ssh-add ~/.ssh/id_*",
			"git_config":      "git config --global user.name 'Your Name' && git config --global user.email 'email@example.com'",
			"github_token":    "Generate new token at https://github.com/settings/tokens",
			"ssh_key_upload":  "Upload key at https://github.com/settings/keys",
			"wrong_ssh_key":   "Use 'gitpersona ssh test' to verify key usage",
		},
		LogFiles: []string{
			"~/.gitpersona/logs/gitpersona.log",
			"/var/log/auth.log (SSH issues)",
			"~/.ssh/known_hosts (SSH connectivity)",
		},
	}

	return info, nil
}

// AutoFix attempts to automatically fix system issues
func (sm *RealSystemManager) AutoFix(ctx context.Context, issues []*SystemIssue) error {
	sm.logger.Info(ctx, "auto_fixing_system_issues",
		observability.F("issues_count", len(issues)),
	)

	fixedCount := 0

	for _, issue := range issues {
		if !issue.AutoFixable {
			continue
		}

		switch issue.Type {
		case "ssh_permissions":
			if err := sm.fixSSHPermissions(ctx); err == nil {
				fixedCount++
				sm.logger.Info(ctx, "fixed_ssh_permissions")
			}

		case "config_directory":
			if err := sm.createConfigDirectory(ctx); err == nil {
				fixedCount++
				sm.logger.Info(ctx, "created_config_directory")
			}

		case "ssh_directory":
			if err := sm.createSSHDirectory(ctx); err == nil {
				fixedCount++
				sm.logger.Info(ctx, "created_ssh_directory")
			}

		default:
			sm.logger.Info(ctx, "unknown_auto_fix_type",
				observability.F("type", issue.Type),
			)
		}
	}

	sm.logger.Info(ctx, "auto_fix_completed",
		observability.F("total", len(issues)),
		observability.F("fixed", fixedCount),
	)

	return nil
}

// MigrateConfiguration migrates configuration from old to new version
func (sm *RealSystemManager) MigrateConfiguration(ctx context.Context, fromVersion, toVersion string) error {
	sm.logger.Info(ctx, "migrating_configuration",
		observability.F("from_version", fromVersion),
		observability.F("to_version", toVersion),
	)

	// TODO: Implement version-specific migration logic
	return fmt.Errorf("migration from %s to %s not implemented", fromVersion, toVersion)
}

// BackupConfiguration creates a backup of the current configuration
func (sm *RealSystemManager) BackupConfiguration(ctx context.Context) (string, error) {
	sm.logger.Info(ctx, "backing_up_configuration")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := fmt.Sprintf("%s/.gitpersona", homeDir)
	backupName := fmt.Sprintf("gitpersona-backup-%s.tar.gz", time.Now().Format("20060102-150405"))
	backupPath := fmt.Sprintf("%s/%s", configDir, backupName)

	// Create tar.gz backup
	cmd := exec.CommandContext(ctx, "tar", "-czf", backupPath, "-C", homeDir, ".gitpersona")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	sm.logger.Info(ctx, "configuration_backed_up",
		observability.F("backup_path", backupPath),
	)

	return backupPath, nil
}

// Health check methods

func (sm *RealSystemManager) checkGitInstallation(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	return nil
}

func (sm *RealSystemManager) checkSSHInstallation(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "ssh", "-V")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh is not installed or not in PATH")
	}
	return nil
}

func (sm *RealSystemManager) checkConfigurationDirectory(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := fmt.Sprintf("%s/.gitpersona", homeDir)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("configuration directory does not exist: %s", configDir)
	}

	return nil
}

func (sm *RealSystemManager) checkAccountConfiguration(ctx context.Context) error {
	if sm.accountManager == nil {
		return nil
	}

	accounts, err := sm.accountManager.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts configured")
	}

	return nil
}

// Diagnostic methods

func (sm *RealSystemManager) diagnoseBinaryInstallations(ctx context.Context) *DiagnosticCheck {
	if err := sm.checkGitInstallation(ctx); err != nil {
		return &DiagnosticCheck{
			Name:    "Git Installation",
			Status:  "fail",
			Message: err.Error(),
			Fix:     "Install Git: https://git-scm.com/downloads",
		}
	}

	if err := sm.checkSSHInstallation(ctx); err != nil {
		return &DiagnosticCheck{
			Name:    "SSH Installation",
			Status:  "fail",
			Message: err.Error(),
			Fix:     "Install OpenSSH client",
		}
	}

	return &DiagnosticCheck{
		Name:    "Binary Installations",
		Status:  "pass",
		Message: "Git and SSH are properly installed",
		Fix:     "",
	}
}

func (sm *RealSystemManager) diagnosePermissions(ctx context.Context) *DiagnosticCheck {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &DiagnosticCheck{
			Name:    "Permissions",
			Status:  "fail",
			Message: "Cannot access home directory",
			Fix:     "Check home directory permissions",
		}
	}

	// Check SSH directory permissions
	sshDir := fmt.Sprintf("%s/.ssh", homeDir)
	if info, err := os.Stat(sshDir); err == nil {
		if info.Mode().Perm()&077 != 0 {
			return &DiagnosticCheck{
				Name:    "SSH Directory Permissions",
				Status:  "warn",
				Message: "SSH directory has incorrect permissions",
				Fix:     fmt.Sprintf("chmod 700 %s", sshDir),
			}
		}
	}

	return &DiagnosticCheck{
		Name:    "Permissions",
		Status:  "pass",
		Message: "Directory permissions are correct",
		Fix:     "",
	}
}

func (sm *RealSystemManager) diagnoseConfiguration(ctx context.Context) *DiagnosticCheck {
	if err := sm.checkConfigurationDirectory(ctx); err != nil {
		return &DiagnosticCheck{
			Name:    "Configuration",
			Status:  "warn",
			Message: err.Error(),
			Fix:     "Run 'gitpersona account setup' to initialize configuration",
		}
	}

	return &DiagnosticCheck{
		Name:    "Configuration",
		Status:  "pass",
		Message: "Configuration directory exists",
		Fix:     "",
	}
}

// Utility methods

func (sm *RealSystemManager) getSSHVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "ssh", "-V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// SSH version is typically in stderr and has format "OpenSSH_8.9p1"
	version := string(output)
	if strings.Contains(version, "OpenSSH_") {
		parts := strings.Fields(version)
		if len(parts) > 0 {
			return parts[0], nil
		}
	}

	return strings.TrimSpace(version), nil
}

func (sm *RealSystemManager) getGitVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Git version format: "git version 2.39.0"
	version := strings.TrimSpace(string(output))
	if strings.HasPrefix(version, "git version ") {
		return strings.TrimPrefix(version, "git version "), nil
	}

	return version, nil
}

func (sm *RealSystemManager) severityToStatus(severity string) string {
	switch severity {
	case "high":
		return "fail"
	case "medium":
		return "warn"
	case "low":
		return "warn"
	default:
		return "pass"
	}
}

// Auto-fix methods

func (sm *RealSystemManager) fixSSHPermissions(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)

	// Fix SSH directory permissions
	if err := os.Chmod(sshDir, 0700); err != nil {
		return err
	}

	// Fix key permissions
	if sm.sshManager != nil {
		if keys, err := sm.sshManager.ListKeys(ctx); err == nil {
			for _, key := range keys {
				if key.Exists {
					os.Chmod(key.Path, 0600)
					os.Chmod(key.Path+".pub", 0644)
				}
			}
		}
	}

	return nil
}

func (sm *RealSystemManager) createConfigDirectory(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := fmt.Sprintf("%s/.gitpersona", homeDir)
	return os.MkdirAll(configDir, 0755)
}

func (sm *RealSystemManager) createSSHDirectory(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)
	return os.MkdirAll(sshDir, 0700)
}
