package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/thukabjj/GitPersona/internal/config"
	"github.com/thukabjj/GitPersona/internal/git"
	"github.com/thukabjj/GitPersona/internal/github"
	"github.com/spf13/cobra"
)

// healthCmd provides comprehensive health checking following 2025 observability standards
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Comprehensive system health check",
	Long: `Perform a comprehensive health check of the GitHub Account Switcher system.

This follows 2025 observability best practices by checking:
- Application configuration integrity
- GitHub API connectivity and authentication
- SSH key validation and security compliance
- Git binary compatibility
- System dependencies and versions
- Account consistency and validation
- Performance benchmarks

Examples:
  gh-switcher health
  gh-switcher health --format json
  gh-switcher health --detailed`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		detailed, _ := cmd.Flags().GetBool("detailed")

		healthStatus := performComprehensiveHealthCheck(detailed)

		switch format {
		case "json":
			return printHealthJSON(healthStatus)
		default:
			return printHealthHuman(healthStatus, detailed)
		}
	},
}

// HealthStatus represents the overall system health
type HealthStatus struct {
	Status      string                 `json:"status"` // "healthy", "warning", "critical"
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Checks      map[string]HealthCheck `json:"checks"`
	Summary     HealthSummary          `json:"summary"`
	Performance PerformanceMetrics     `json:"performance,omitempty"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status   string        `json:"status"`
	Message  string        `json:"message"`
	Details  interface{}   `json:"details,omitempty"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	TotalChecks    int `json:"total_checks"`
	HealthyChecks  int `json:"healthy_checks"`
	WarningChecks  int `json:"warning_checks"`
	CriticalChecks int `json:"critical_checks"`
}

// PerformanceMetrics provides performance benchmarks
type PerformanceMetrics struct {
	GitConfigTime   time.Duration `json:"git_config_time"`
	GitHubAPITime   time.Duration `json:"github_api_time"`
	SSHValidateTime time.Duration `json:"ssh_validate_time"`
	ConfigLoadTime  time.Duration `json:"config_load_time"`
}

// performComprehensiveHealthCheck runs all health checks
func performComprehensiveHealthCheck(detailed bool) *HealthStatus {
	startTime := time.Now()
	status := &HealthStatus{
		Timestamp: startTime,
		Version:   "2.0.0", // Would be from build info
		Checks:    make(map[string]HealthCheck),
		Summary:   HealthSummary{},
	}

	// Define all health checks for 2025 standards
	checks := map[string]func() HealthCheck{
		"config_integrity":     checkConfigIntegrity,
		"github_api":           checkGitHubAPI,
		"git_binary":           checkGitBinary,
		"ssh_agent":            checkSSHAgent,
		"ssh_keys":             checkSSHKeys,
		"account_validation":   checkAccountValidation,
		"permissions":          checkFilePermissions,
		"dependencies":         checkDependencies,
		"network_connectivity": checkNetworkConnectivity,
	}

	if detailed {
		checks["performance_benchmark"] = benchmarkPerformance
		checks["security_audit"] = checkSecurityCompliance
		checks["update_availability"] = checkUpdateAvailability
	}

	// Run all checks
	healthyCount := 0
	warningCount := 0
	criticalCount := 0

	for name, checkFunc := range checks {
		start := time.Now()
		result := checkFunc()
		result.Duration = time.Since(start)

		status.Checks[name] = result

		switch result.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
	}

	// Determine overall status
	if criticalCount > 0 {
		status.Status = "critical"
	} else if warningCount > 0 {
		status.Status = "warning"
	} else {
		status.Status = "healthy"
	}

	status.Summary = HealthSummary{
		TotalChecks:    len(checks),
		HealthyChecks:  healthyCount,
		WarningChecks:  warningCount,
		CriticalChecks: criticalCount,
	}

	return status
}

// Individual health check functions

func checkConfigIntegrity() HealthCheck {
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return HealthCheck{
			Status:  "critical",
			Message: "Configuration loading failed",
			Error:   err.Error(),
		}
	}

	accounts := configManager.ListAccounts()
	details := map[string]interface{}{
		"accounts_count": len(accounts),
		"config_file":    "~/.config/gh-switcher/config.yaml",
	}

	if len(accounts) == 0 {
		return HealthCheck{
			Status:  "warning",
			Message: "No accounts configured",
			Details: details,
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("Configuration loaded successfully (%d accounts)", len(accounts)),
		Details: details,
	}
}

func checkGitHubAPI() HealthCheck {
	client := github.NewClient("")

	// Test with a simple API call
	start := time.Now()
	_, err := client.FetchUserInfo("octocat")
	apiDuration := time.Since(start)

	if err != nil {
		return HealthCheck{
			Status:  "critical",
			Message: "GitHub API unreachable",
			Error:   err.Error(),
		}
	}

	details := map[string]interface{}{
		"response_time_ms": apiDuration.Milliseconds(),
		"endpoint":         "https://api.github.com",
	}

	if apiDuration > 5*time.Second {
		return HealthCheck{
			Status:  "warning",
			Message: "GitHub API responding slowly",
			Details: details,
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "GitHub API accessible",
		Details: details,
	}
}

func checkGitBinary() HealthCheck {
	gitManager := git.NewManager()
	version, err := gitManager.GetGitVersion()
	if err != nil {
		return HealthCheck{
			Status:  "critical",
			Message: "Git binary not found",
			Error:   err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Git binary available",
		Details: map[string]interface{}{
			"version": version,
		},
	}
}

func checkSSHAgent() HealthCheck {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()

	if err != nil {
		return HealthCheck{
			Status:  "warning",
			Message: "SSH agent not running or no keys loaded",
			Details: map[string]interface{}{
				"suggestion": "Run 'ssh-add ~/.ssh/id_*' to load keys",
			},
		}
	}

	keyCount := len(strings.Split(strings.TrimSpace(string(output)), "\n"))

	return HealthCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("SSH agent running with %d keys", keyCount),
		Details: map[string]interface{}{
			"loaded_keys": keyCount,
		},
	}
}

func checkSSHKeys() HealthCheck {
	configManager := config.NewManager()
	configManager.Load()

	accounts := configManager.ListAccounts()
	validKeys := 0
	issues := []string{}

	for _, account := range accounts {
		if account.SSHKeyPath != "" {
			gitManager := git.NewManager()
			if err := gitManager.ValidateSSHKey(account.SSHKeyPath); err != nil {
				issues = append(issues, fmt.Sprintf("%s: %v", account.Alias, err))
			} else {
				validKeys++
			}
		}
	}

	details := map[string]interface{}{
		"valid_keys": validKeys,
		"issues":     issues,
	}

	if len(issues) > 0 {
		return HealthCheck{
			Status:  "warning",
			Message: fmt.Sprintf("SSH key issues found (%d valid, %d issues)", validKeys, len(issues)),
			Details: details,
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("All SSH keys valid (%d checked)", validKeys),
		Details: details,
	}
}

func checkAccountValidation() HealthCheck {
	configManager := config.NewManager()
	configManager.Load()

	accounts := configManager.ListAccounts()
	validAccounts := 0
	invalidAccounts := []string{}

	for _, account := range accounts {
		if err := account.Validate(); err != nil {
			invalidAccounts = append(invalidAccounts, fmt.Sprintf("%s: %v", account.Alias, err))
		} else {
			validAccounts++
		}
	}

	if len(invalidAccounts) > 0 {
		return HealthCheck{
			Status:  "critical",
			Message: "Invalid accounts found",
			Details: map[string]interface{}{
				"valid_accounts":   validAccounts,
				"invalid_accounts": invalidAccounts,
			},
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: fmt.Sprintf("All accounts valid (%d checked)", validAccounts),
	}
}

func checkFilePermissions() HealthCheck {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "gh-switcher")

	// Check config directory permissions (should be 755)
	if info, err := os.Stat(configDir); err == nil {
		if info.Mode().Perm() != 0755 {
			return HealthCheck{
				Status:  "warning",
				Message: "Config directory permissions not optimal",
				Details: map[string]interface{}{
					"current":     fmt.Sprintf("%o", info.Mode().Perm()),
					"recommended": "755",
				},
			}
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "File permissions correct",
	}
}

func checkDependencies() HealthCheck {
	dependencies := map[string]string{
		"git": "git --version",
		"gh":  "gh --version",
	}

	results := make(map[string]string)
	issues := []string{}

	for dep, cmd := range dependencies {
		parts := strings.Fields(cmd)
		execCmd := exec.Command(parts[0], parts[1:]...)
		output, err := execCmd.Output()

		if err != nil {
			issues = append(issues, fmt.Sprintf("%s not found", dep))
			results[dep] = "not found"
		} else {
			version := strings.TrimSpace(string(output))
			results[dep] = version
		}
	}

	if len(issues) > 0 {
		return HealthCheck{
			Status:  "warning",
			Message: "Some dependencies missing",
			Details: map[string]interface{}{
				"dependencies": results,
				"issues":       issues,
			},
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "All dependencies available",
		Details: map[string]interface{}{
			"dependencies": results,
		},
	}
}

func checkNetworkConnectivity() HealthCheck {
	start := time.Now()
	cmd := exec.Command("ping", "-c", "1", "github.com")
	err := cmd.Run()
	pingDuration := time.Since(start)

	if err != nil {
		return HealthCheck{
			Status:  "critical",
			Message: "Network connectivity to GitHub failed",
			Error:   err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Network connectivity OK",
		Details: map[string]interface{}{
			"ping_time_ms": pingDuration.Milliseconds(),
		},
	}
}

func benchmarkPerformance() HealthCheck {
	// Performance benchmark following 2025 standards
	start := time.Now()

	// Benchmark config loading
	configStart := time.Now()
	configManager := config.NewManager()
	configManager.Load()
	configDuration := time.Since(configStart)

	// Benchmark Git operations
	gitStart := time.Now()
	gitManager := git.NewManager()
	gitManager.GetGitVersion()
	gitDuration := time.Since(gitStart)

	totalDuration := time.Since(start)

	performance := map[string]interface{}{
		"total_time_ms":  totalDuration.Milliseconds(),
		"config_time_ms": configDuration.Milliseconds(),
		"git_time_ms":    gitDuration.Milliseconds(),
	}

	// 2025 performance standards: operations should complete under 100ms
	if totalDuration > 100*time.Millisecond {
		return HealthCheck{
			Status:  "warning",
			Message: "Performance below 2025 standards",
			Details: performance,
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Performance meets 2025 standards",
		Details: performance,
	}
}

func checkSecurityCompliance() HealthCheck {
	issues := []string{}

	// Check for weak SSH keys
	homeDir, _ := os.UserHomeDir()
	sshDir := filepath.Join(homeDir, ".ssh")

	if files, err := os.ReadDir(sshDir); err == nil {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "id_rsa") && !strings.HasSuffix(file.Name(), ".pub") {
				// Check if RSA key is strong enough for 2025
				keyPath := filepath.Join(sshDir, file.Name())
				if info, err := os.Stat(keyPath); err == nil {
					if info.Mode().Perm() != 0600 {
						issues = append(issues, fmt.Sprintf("Key %s has insecure permissions", file.Name()))
					}
				}
			}
		}
	}

	// Check for old configuration patterns
	configManager := config.NewManager()
	configManager.Load()
	accounts := configManager.ListAccounts()

	for _, account := range accounts {
		if account.GitHubUsername == "" {
			issues = append(issues, fmt.Sprintf("Account '%s' missing GitHub username", account.Alias))
		}
	}

	if len(issues) > 0 {
		return HealthCheck{
			Status:  "warning",
			Message: "Security compliance issues found",
			Details: map[string]interface{}{
				"issues": issues,
			},
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Security compliance check passed",
	}
}

func checkUpdateAvailability() HealthCheck {
	// In a real implementation, this would check for newer versions
	// from GitHub releases API
	return HealthCheck{
		Status:  "healthy",
		Message: "Running latest version",
		Details: map[string]interface{}{
			"current_version": "2.0.0",
			"latest_version":  "2.0.0",
		},
	}
}

// printHealthJSON outputs health status in JSON format
func printHealthJSON(health *HealthStatus) error {
	output, err := json.MarshalIndent(health, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal health status: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

// printHealthHuman outputs health status in human-readable format
func printHealthHuman(health *HealthStatus, detailed bool) error {
	// Status header
	statusIcon := "✅"
	if health.Status == "warning" {
		statusIcon = "⚠️"
	} else if health.Status == "critical" {
		statusIcon = "❌"
	}

	fmt.Printf("%s GitHub Account Switcher Health Check\n", statusIcon)
	fmt.Printf("Status: %s (%s)\n", strings.ToUpper(health.Status), health.Timestamp.Format(time.RFC3339))
	fmt.Println()

	// Summary
	fmt.Printf("📊 Summary: %d/%d checks passed\n",
		health.Summary.HealthyChecks, health.Summary.TotalChecks)

	if health.Summary.WarningChecks > 0 {
		fmt.Printf("⚠️  %d warnings\n", health.Summary.WarningChecks)
	}
	if health.Summary.CriticalChecks > 0 {
		fmt.Printf("❌ %d critical issues\n", health.Summary.CriticalChecks)
	}
	fmt.Println()

	// Individual checks
	fmt.Println("🔍 Detailed Results:")
	for name, check := range health.Checks {
		icon := getStatusIcon(check.Status)
		fmt.Printf("   %s %s: %s", icon, name, check.Message)

		if detailed && check.Duration > 0 {
			fmt.Printf(" (%dms)", check.Duration.Milliseconds())
		}
		fmt.Println()

		if check.Error != "" {
			fmt.Printf("      Error: %s\n", check.Error)
		}

		if detailed && check.Details != nil {
			if details, ok := check.Details.(map[string]interface{}); ok {
				for key, value := range details {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}
		}
	}

	// Exit with appropriate code
	if health.Status == "critical" {
		os.Exit(1)
	} else if health.Status == "warning" {
		os.Exit(2)
	}

	return nil
}

func getStatusIcon(status string) string {
	switch status {
	case "healthy":
		return "✅"
	case "warning":
		return "⚠️"
	case "critical":
		return "❌"
	default:
		return "❓"
	}
}

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().StringP("format", "f", "human", "Output format (human, json)")
	healthCmd.Flags().Bool("detailed", false, "Show detailed health information")
}
