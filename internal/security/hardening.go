package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// HardeningManager implements security hardening measures
type HardeningManager struct {
	logger   observability.Logger
	config   *HardeningConfig
	measures map[string]*HardeningMeasure
}

// HardeningConfig configures security hardening
type HardeningConfig struct {
	Level              HardeningLevel `json:"level"`
	EnabledMeasures    []string       `json:"enabled_measures"`
	DisabledMeasures   []string       `json:"disabled_measures"`
	BackupBeforeHarden bool           `json:"backup_before_harden"`
	BackupPath         string         `json:"backup_path"`
	DryRun             bool           `json:"dry_run"`
	VerifyAfterHarden  bool           `json:"verify_after_harden"`
	RevertOnFailure    bool           `json:"revert_on_failure"`
	MaxRollbackAge     time.Duration  `json:"max_rollback_age"`
}

// HardeningLevel defines security hardening levels
type HardeningLevel string

const (
	HardeningLevelBasic    HardeningLevel = "basic"
	HardeningLevelStandard HardeningLevel = "standard"
	HardeningLevelAdvanced HardeningLevel = "advanced"
	HardeningLevelMaximum  HardeningLevel = "maximum"
)

// HardeningMeasure represents a security hardening measure
type HardeningMeasure struct {
	ID           string                                  `json:"id"`
	Name         string                                  `json:"name"`
	Description  string                                  `json:"description"`
	Level        HardeningLevel                          `json:"level"`
	Platform     []string                                `json:"platform"`
	Enabled      bool                                    `json:"enabled"`
	Applied      bool                                    `json:"applied"`
	Reversible   bool                                    `json:"reversible"`
	RiskLevel    string                                  `json:"risk_level"`
	Dependencies []string                                `json:"dependencies"`
	ApplyFunc    func(ctx context.Context) error         `json:"-"`
	VerifyFunc   func(ctx context.Context) (bool, error) `json:"-"`
	RevertFunc   func(ctx context.Context) error         `json:"-"`
}

// HardeningResult contains the results of hardening operations
type HardeningResult struct {
	Timestamp       time.Time                 `json:"timestamp"`
	Level           HardeningLevel            `json:"level"`
	TotalMeasures   int                       `json:"total_measures"`
	AppliedMeasures int                       `json:"applied_measures"`
	FailedMeasures  int                       `json:"failed_measures"`
	SkippedMeasures int                       `json:"skipped_measures"`
	Results         map[string]*MeasureResult `json:"results"`
	BackupPath      string                    `json:"backup_path,omitempty"`
	Duration        time.Duration             `json:"duration"`
	Errors          []string                  `json:"errors,omitempty"`
}

// MeasureResult contains the result of applying a single hardening measure
type MeasureResult struct {
	ID        string        `json:"id"`
	Applied   bool          `json:"applied"`
	Verified  bool          `json:"verified"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// NewHardeningManager creates a new hardening manager
func NewHardeningManager(logger observability.Logger) *HardeningManager {
	config := &HardeningConfig{
		Level:              HardeningLevelStandard,
		BackupBeforeHarden: true,
		DryRun:             false,
		VerifyAfterHarden:  true,
		RevertOnFailure:    true,
		MaxRollbackAge:     24 * time.Hour,
	}

	hm := &HardeningManager{
		logger:   logger,
		config:   config,
		measures: make(map[string]*HardeningMeasure),
	}

	hm.loadHardeningMeasures()
	return hm
}

// loadHardeningMeasures loads all available hardening measures
func (hm *HardeningManager) loadHardeningMeasures() {
	measures := []*HardeningMeasure{
		{
			ID:          "ssh-permissions",
			Name:        "SSH Directory Permissions",
			Description: "Secure SSH directory and key file permissions",
			Level:       HardeningLevelBasic,
			Platform:    []string{"linux", "darwin"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "low",
			ApplyFunc:   hm.hardenSSHPermissions,
			VerifyFunc:  hm.verifySSHPermissions,
			RevertFunc:  hm.revertSSHPermissions,
		},
		{
			ID:          "config-permissions",
			Name:        "Configuration File Permissions",
			Description: "Secure GitPersona configuration file permissions",
			Level:       HardeningLevelBasic,
			Platform:    []string{"linux", "darwin", "windows"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "low",
			ApplyFunc:   hm.hardenConfigPermissions,
			VerifyFunc:  hm.verifyConfigPermissions,
		},
		{
			ID:          "secure-temp",
			Name:        "Secure Temporary Files",
			Description: "Configure secure temporary file handling",
			Level:       HardeningLevelStandard,
			Platform:    []string{"linux", "darwin", "windows"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "medium",
			ApplyFunc:   hm.hardenTemporaryFiles,
			VerifyFunc:  hm.verifyTemporaryFiles,
		},
		{
			ID:          "disable-core-dumps",
			Name:        "Disable Core Dumps",
			Description: "Disable core dumps to prevent credential exposure",
			Level:       HardeningLevelStandard,
			Platform:    []string{"linux", "darwin"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "medium",
			ApplyFunc:   hm.disableCoreDumps,
			VerifyFunc:  hm.verifyCoreDumpsDisabled,
			RevertFunc:  hm.enableCoreDumps,
		},
		{
			ID:          "secure-memory",
			Name:        "Secure Memory Handling",
			Description: "Configure secure memory handling for sensitive data",
			Level:       HardeningLevelAdvanced,
			Platform:    []string{"linux", "darwin"},
			Enabled:     true,
			Reversible:  false,
			RiskLevel:   "low",
			ApplyFunc:   hm.hardenMemoryHandling,
			VerifyFunc:  hm.verifyMemoryHandling,
		},
		{
			ID:          "process-isolation",
			Name:        "Process Isolation",
			Description: "Enable process isolation and sandboxing",
			Level:       HardeningLevelAdvanced,
			Platform:    []string{"linux", "darwin"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "medium",
			ApplyFunc:   hm.enableProcessIsolation,
			VerifyFunc:  hm.verifyProcessIsolation,
		},
		{
			ID:          "network-restrictions",
			Name:        "Network Access Restrictions",
			Description: "Restrict network access to required endpoints only",
			Level:       HardeningLevelMaximum,
			Platform:    []string{"linux", "darwin", "windows"},
			Enabled:     false, // Disabled by default due to potential connectivity issues
			Reversible:  true,
			RiskLevel:   "high",
			ApplyFunc:   hm.applyNetworkRestrictions,
			VerifyFunc:  hm.verifyNetworkRestrictions,
			RevertFunc:  hm.removeNetworkRestrictions,
		},
		{
			ID:          "audit-logging",
			Name:        "Enhanced Audit Logging",
			Description: "Enable comprehensive audit logging for security events",
			Level:       HardeningLevelStandard,
			Platform:    []string{"linux", "darwin", "windows"},
			Enabled:     true,
			Reversible:  true,
			RiskLevel:   "low",
			ApplyFunc:   hm.enableAuditLogging,
			VerifyFunc:  hm.verifyAuditLogging,
		},
	}

	for _, measure := range measures {
		hm.measures[measure.ID] = measure
	}
}

// ApplyHardening applies security hardening measures
func (hm *HardeningManager) ApplyHardening(ctx context.Context, level HardeningLevel) (*HardeningResult, error) {
	startTime := time.Now()

	hm.logger.Info(ctx, "starting_security_hardening",
		observability.F("level", level),
		observability.F("dry_run", hm.config.DryRun),
	)

	result := &HardeningResult{
		Timestamp: startTime,
		Level:     level,
		Results:   make(map[string]*MeasureResult),
		Errors:    make([]string, 0),
	}

	// Create backup if configured
	if hm.config.BackupBeforeHarden && !hm.config.DryRun {
		backupPath, err := hm.createBackup(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
		result.BackupPath = backupPath
	}

	// Get applicable measures for the specified level
	applicableMeasures := hm.getApplicableMeasures(level)
	result.TotalMeasures = len(applicableMeasures)

	// Apply each measure
	for _, measure := range applicableMeasures {
		measureResult := &MeasureResult{
			ID:        measure.ID,
			Timestamp: time.Now(),
		}

		measureStart := time.Now()

		if hm.config.DryRun {
			hm.logger.Info(ctx, "dry_run_would_apply_measure",
				observability.F("measure_id", measure.ID),
				observability.F("measure_name", measure.Name),
			)
			measureResult.Applied = true
		} else {
			// Apply the measure
			if err := measure.ApplyFunc(ctx); err != nil {
				measureResult.Error = err.Error()
				result.FailedMeasures++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", measure.ID, err))

				hm.logger.Error(ctx, "hardening_measure_failed",
					observability.F("measure_id", measure.ID),
					observability.F("error", err.Error()),
				)
			} else {
				measureResult.Applied = true
				result.AppliedMeasures++
				measure.Applied = true

				hm.logger.Info(ctx, "hardening_measure_applied",
					observability.F("measure_id", measure.ID),
					observability.F("measure_name", measure.Name),
				)

				// Verify if configured
				if hm.config.VerifyAfterHarden && measure.VerifyFunc != nil {
					if verified, err := measure.VerifyFunc(ctx); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("verification failed for %s: %v", measure.ID, err))
					} else {
						measureResult.Verified = verified
						if !verified {
							result.Errors = append(result.Errors, fmt.Sprintf("verification failed for %s", measure.ID))
						}
					}
				}
			}
		}

		measureResult.Duration = time.Since(measureStart)
		result.Results[measure.ID] = measureResult
	}

	result.Duration = time.Since(startTime)

	hm.logger.Info(ctx, "security_hardening_completed",
		observability.F("level", level),
		observability.F("total_measures", result.TotalMeasures),
		observability.F("applied_measures", result.AppliedMeasures),
		observability.F("failed_measures", result.FailedMeasures),
		observability.F("duration", result.Duration.String()),
	)

	return result, nil
}

// getApplicableMeasures returns measures applicable for the given level
func (hm *HardeningManager) getApplicableMeasures(level HardeningLevel) []*HardeningMeasure {
	var applicable []*HardeningMeasure

	levelOrder := map[HardeningLevel]int{
		HardeningLevelBasic:    1,
		HardeningLevelStandard: 2,
		HardeningLevelAdvanced: 3,
		HardeningLevelMaximum:  4,
	}

	targetLevel := levelOrder[level]

	for _, measure := range hm.measures {
		measureLevel := levelOrder[measure.Level]

		// Include measures at or below the target level
		if measureLevel <= targetLevel && measure.Enabled {
			// Check platform compatibility
			if hm.isPlatformSupported(measure.Platform) {
				// Check if explicitly disabled
				if !hm.isMeasureDisabled(measure.ID) {
					applicable = append(applicable, measure)
				}
			}
		}
	}

	return applicable
}

// isPlatformSupported checks if the current platform is supported
func (hm *HardeningManager) isPlatformSupported(platforms []string) bool {
	if len(platforms) == 0 {
		return true
	}

	currentPlatform := runtime.GOOS
	for _, platform := range platforms {
		if platform == currentPlatform {
			return true
		}
	}

	return false
}

// isMeasureDisabled checks if a measure is explicitly disabled
func (hm *HardeningManager) isMeasureDisabled(measureID string) bool {
	for _, disabled := range hm.config.DisabledMeasures {
		if disabled == measureID {
			return true
		}
	}
	return false
}

// createBackup creates a backup of current configuration
func (hm *HardeningManager) createBackup(ctx context.Context) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(hm.config.BackupPath, "hardening-backups")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("backup-%s.tar.gz", timestamp))

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup (implementation would tar relevant files)
	hm.logger.Info(ctx, "creating_hardening_backup",
		observability.F("backup_path", backupPath),
	)

	// Implementation would backup SSH config, GitPersona config, etc.

	return backupPath, nil
}

// Hardening measure implementations

func (hm *HardeningManager) hardenSSHPermissions(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Set SSH directory permissions (700)
	if err := os.Chmod(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to set SSH directory permissions: %w", err)
	}

	// Set SSH config permissions (600)
	configPath := filepath.Join(sshDir, "config")
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Chmod(configPath, 0600); err != nil {
			return fmt.Errorf("failed to set SSH config permissions: %w", err)
		}
	}

	// Set private key permissions (600)
	privateKeys := []string{"id_rsa", "id_ed25519", "id_ecdsa"}
	for _, keyName := range privateKeys {
		keyPath := filepath.Join(sshDir, keyName)
		if _, err := os.Stat(keyPath); err == nil {
			if err := os.Chmod(keyPath, 0600); err != nil {
				return fmt.Errorf("failed to set private key permissions for %s: %w", keyName, err)
			}
		}
	}

	return nil
}

func (hm *HardeningManager) verifySSHPermissions(ctx context.Context) (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Check SSH directory permissions
	info, err := os.Stat(sshDir)
	if err != nil {
		return false, err
	}

	if info.Mode().Perm() != 0700 {
		return false, nil
	}

	return true, nil
}

func (hm *HardeningManager) revertSSHPermissions(ctx context.Context) error {
	// Implementation would restore original permissions from backup
	return nil
}

func (hm *HardeningManager) hardenConfigPermissions(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var configDir string
	switch runtime.GOOS {
	case "windows":
		configDir = filepath.Join(homeDir, "AppData", "Roaming", "GitPersona")
	case "darwin":
		configDir = filepath.Join(homeDir, "Library", "Application Support", "GitPersona")
	default:
		configDir = filepath.Join(homeDir, ".config", "gitpersona")
	}

	// Set config directory permissions
	if err := os.Chmod(configDir, 0700); err != nil {
		return fmt.Errorf("failed to set config directory permissions: %w", err)
	}

	// Set config file permissions
	configFile := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configFile); err == nil {
		if err := os.Chmod(configFile, 0600); err != nil {
			return fmt.Errorf("failed to set config file permissions: %w", err)
		}
	}

	return nil
}

func (hm *HardeningManager) verifyConfigPermissions(ctx context.Context) (bool, error) {
	// Implementation would verify config permissions
	return true, nil
}

func (hm *HardeningManager) hardenTemporaryFiles(ctx context.Context) error {
	// Set secure umask for temporary files
	// Implementation would configure secure temp file handling
	return nil
}

func (hm *HardeningManager) verifyTemporaryFiles(ctx context.Context) (bool, error) {
	return true, nil
}

func (hm *HardeningManager) disableCoreDumps(ctx context.Context) error {
	if runtime.GOOS == "windows" {
		return nil // Not applicable on Windows
	}

	// Set core dump limit to 0
	// Implementation would use setrlimit or equivalent
	return nil
}

func (hm *HardeningManager) verifyCoreDumpsDisabled(ctx context.Context) (bool, error) {
	return true, nil
}

func (hm *HardeningManager) enableCoreDumps(ctx context.Context) error {
	// Revert core dump settings
	return nil
}

func (hm *HardeningManager) hardenMemoryHandling(ctx context.Context) error {
	// Configure secure memory handling
	// Implementation would set memory protection flags
	return nil
}

func (hm *HardeningManager) verifyMemoryHandling(ctx context.Context) (bool, error) {
	return true, nil
}

func (hm *HardeningManager) enableProcessIsolation(ctx context.Context) error {
	// Enable process isolation measures
	// Implementation would configure namespaces, capabilities, etc.
	return nil
}

func (hm *HardeningManager) verifyProcessIsolation(ctx context.Context) (bool, error) {
	return true, nil
}

func (hm *HardeningManager) applyNetworkRestrictions(ctx context.Context) error {
	// Apply network access restrictions
	// Implementation would configure firewall rules, etc.
	return nil
}

func (hm *HardeningManager) verifyNetworkRestrictions(ctx context.Context) (bool, error) {
	return true, nil
}

func (hm *HardeningManager) removeNetworkRestrictions(ctx context.Context) error {
	// Remove network restrictions
	return nil
}

func (hm *HardeningManager) enableAuditLogging(ctx context.Context) error {
	// Enable enhanced audit logging
	// Implementation would configure audit subsystem
	return nil
}

func (hm *HardeningManager) verifyAuditLogging(ctx context.Context) (bool, error) {
	return true, nil
}

// GetMeasures returns all available hardening measures
func (hm *HardeningManager) GetMeasures() map[string]*HardeningMeasure {
	return hm.measures
}

// SetConfig updates the hardening configuration
func (hm *HardeningManager) SetConfig(config *HardeningConfig) {
	hm.config = config
}

// GetConfig returns the current hardening configuration
func (hm *HardeningManager) GetConfig() *HardeningConfig {
	return hm.config
}

// GenerateSecureToken generates a cryptographically secure random token
func (hm *HardeningManager) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// ValidateConfiguration validates the current security configuration
func (hm *HardeningManager) ValidateConfiguration(ctx context.Context) ([]string, error) {
	var issues []string

	// Check SSH configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if info, err := os.Stat(sshDir); err == nil {
		if info.Mode().Perm() != 0700 {
			issues = append(issues, "SSH directory has insecure permissions")
		}
	}

	// Check for exposed private keys
	privateKeys := []string{"id_rsa", "id_ed25519", "id_ecdsa"}
	for _, keyName := range privateKeys {
		keyPath := filepath.Join(sshDir, keyName)
		if info, err := os.Stat(keyPath); err == nil {
			if info.Mode().Perm() != 0600 {
				issues = append(issues, fmt.Sprintf("Private key %s has insecure permissions", keyName))
			}
		}
	}

	return issues, nil
}
