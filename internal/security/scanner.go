package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SecurityScanner performs comprehensive security scans
type SecurityScanner struct {
	logger   observability.Logger
	config   *ScannerConfig
	rules    map[string]*SecurityRule
	findings []SecurityFinding
}

// ScannerConfig configures the security scanner
type ScannerConfig struct {
	EnabledScans   []ScanType `json:"enabled_scans"`
	ExcludePaths   []string   `json:"exclude_paths"`
	MaxFileSize    int64      `json:"max_file_size"`
	FollowSymlinks bool       `json:"follow_symlinks"`
	Severity       string     `json:"min_severity"`
	OutputFormat   string     `json:"output_format"`
	ReportPath     string     `json:"report_path"`
	AutoFix        bool       `json:"auto_fix"`
	QuarantinePath string     `json:"quarantine_path"`
}

// ScanType defines types of security scans
type ScanType string

const (
	ScanTypeSSH           ScanType = "ssh"
	ScanTypeTokens        ScanType = "tokens"
	ScanTypeConfig        ScanType = "config"
	ScanTypePermissions   ScanType = "permissions"
	ScanTypeSecrets       ScanType = "secrets"
	ScanTypeVulnerability ScanType = "vulnerability"
	ScanTypeMalware       ScanType = "malware"
	ScanTypeNetwork       ScanType = "network"
)

// SecurityFinding represents a security issue found during scanning
type SecurityFinding struct {
	ID          string            `json:"id"`
	Type        ScanType          `json:"type"`
	Severity    SeverityLevel     `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	FilePath    string            `json:"file_path"`
	LineNumber  int               `json:"line_number,omitempty"`
	Evidence    string            `json:"evidence,omitempty"`
	Remediation string            `json:"remediation"`
	CVSS        float64           `json:"cvss,omitempty"`
	CWE         string            `json:"cwe,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Fixed       bool              `json:"fixed"`
	Metadata    map[string]string `json:"metadata"`
}

// SeverityLevel defines severity levels for findings
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// SecurityRule defines a security scanning rule
type SecurityRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        ScanType       `json:"type"`
	Severity    SeverityLevel  `json:"severity"`
	Pattern     *regexp.Regexp `json:"-"`
	PatternStr  string         `json:"pattern"`
	FileTypes   []string       `json:"file_types"`
	Enabled     bool           `json:"enabled"`
	AutoFix     bool           `json:"auto_fix"`
	Remediation string         `json:"remediation"`
	CWE         string         `json:"cwe,omitempty"`
}

// ScanResult contains the results of a security scan
type ScanResult struct {
	ScanID       string            `json:"scan_id"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	FilesScanned int               `json:"files_scanned"`
	Findings     []SecurityFinding `json:"findings"`
	Summary      ScanSummary       `json:"summary"`
	Metadata     map[string]string `json:"metadata"`
}

// ScanSummary provides a summary of scan results
type ScanSummary struct {
	TotalFindings    int `json:"total_findings"`
	CriticalFindings int `json:"critical_findings"`
	HighFindings     int `json:"high_findings"`
	MediumFindings   int `json:"medium_findings"`
	LowFindings      int `json:"low_findings"`
	InfoFindings     int `json:"info_findings"`
	FixedFindings    int `json:"fixed_findings"`
	RiskScore        int `json:"risk_score"`
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner(logger observability.Logger) *SecurityScanner {
	config := &ScannerConfig{
		EnabledScans: []ScanType{
			ScanTypeSSH,
			ScanTypeTokens,
			ScanTypeConfig,
			ScanTypePermissions,
			ScanTypeSecrets,
		},
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		FollowSymlinks: false,
		Severity:       "info",
		OutputFormat:   "json",
		AutoFix:        false,
	}

	scanner := &SecurityScanner{
		logger:   logger,
		config:   config,
		rules:    make(map[string]*SecurityRule),
		findings: make([]SecurityFinding, 0),
	}

	scanner.loadDefaultRules()
	return scanner
}

// loadDefaultRules loads default security scanning rules
func (ss *SecurityScanner) loadDefaultRules() {
	rules := []*SecurityRule{
		{
			ID:          "ssh-001",
			Name:        "Weak SSH Key",
			Description: "SSH key with insufficient bit length detected",
			Type:        ScanTypeSSH,
			Severity:    SeverityMedium,
			PatternStr:  `-----BEGIN RSA PRIVATE KEY-----`,
			FileTypes:   []string{".pem", ".key", "id_rsa", "id_dsa"},
			Enabled:     true,
			Remediation: "Replace with RSA keys of at least 2048 bits or use Ed25519 keys",
			CWE:         "CWE-326",
		},
		{
			ID:          "ssh-002",
			Name:        "SSH Private Key Exposed",
			Description: "SSH private key found in accessible location",
			Type:        ScanTypeSSH,
			Severity:    SeverityHigh,
			PatternStr:  `-----BEGIN.*PRIVATE KEY-----`,
			FileTypes:   []string{"*"},
			Enabled:     true,
			Remediation: "Move private keys to secure location with proper permissions (600)",
			CWE:         "CWE-200",
		},
		{
			ID:          "token-001",
			Name:        "GitHub Token Exposed",
			Description: "GitHub personal access token found in plaintext",
			Type:        ScanTypeTokens,
			Severity:    SeverityCritical,
			PatternStr:  `ghp_[a-zA-Z0-9]{36}`,
			FileTypes:   []string{"*"},
			Enabled:     true,
			Remediation: "Remove token from code and use secure credential storage",
			CWE:         "CWE-798",
		},
		{
			ID:          "token-002",
			Name:        "Generic API Key",
			Description: "Potential API key found in plaintext",
			Type:        ScanTypeTokens,
			Severity:    SeverityHigh,
			PatternStr:  `(api[_-]?key|apikey|access[_-]?token)\s*[:=]\s*["']?[a-zA-Z0-9]{20,}["']?`,
			FileTypes:   []string{".json", ".yaml", ".yml", ".conf", ".config", ".env"},
			Enabled:     true,
			Remediation: "Use environment variables or secure credential storage",
			CWE:         "CWE-798",
		},
		{
			ID:          "config-001",
			Name:        "Insecure File Permissions",
			Description: "Configuration file has overly permissive permissions",
			Type:        ScanTypePermissions,
			Severity:    SeverityMedium,
			FileTypes:   []string{".config", ".conf", ".json", ".yaml"},
			Enabled:     true,
			Remediation: "Set restrictive permissions (600 or 644)",
			CWE:         "CWE-276",
		},
		{
			ID:          "secret-001",
			Name:        "Hard-coded Password",
			Description: "Potential hard-coded password found",
			Type:        ScanTypeSecrets,
			Severity:    SeverityHigh,
			PatternStr:  `(password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}["']`,
			FileTypes:   []string{"*"},
			Enabled:     true,
			Remediation: "Use secure credential storage or environment variables",
			CWE:         "CWE-798",
		},
		{
			ID:          "secret-002",
			Name:        "Database Connection String",
			Description: "Database connection string with credentials found",
			Type:        ScanTypeSecrets,
			Severity:    SeverityHigh,
			PatternStr:  `(mongodb|mysql|postgres|redis)://[^:]+:[^@]+@`,
			FileTypes:   []string{"*"},
			Enabled:     true,
			Remediation: "Use connection strings without embedded credentials",
			CWE:         "CWE-200",
		},
	}

	for _, rule := range rules {
		if rule.PatternStr != "" {
			rule.Pattern = regexp.MustCompile(`(?i)` + rule.PatternStr)
		}
		ss.rules[rule.ID] = rule
	}
}

// Scan performs a comprehensive security scan
func (ss *SecurityScanner) Scan(ctx context.Context, targetPath string) (*ScanResult, error) {
	scanID := ss.generateScanID()
	startTime := time.Now()

	ss.logger.Info(ctx, "starting_security_scan",
		observability.F("scan_id", scanID),
		observability.F("target_path", targetPath),
		observability.F("enabled_scans", ss.config.EnabledScans),
	)

	result := &ScanResult{
		ScanID:    scanID,
		StartTime: startTime,
		Findings:  make([]SecurityFinding, 0),
		Metadata:  make(map[string]string),
	}

	// Reset findings
	ss.findings = make([]SecurityFinding, 0)

	// Perform different types of scans
	filesScanned := 0

	if ss.isScanEnabled(ScanTypeSSH) {
		count, err := ss.scanSSHSecurity(ctx, targetPath)
		if err != nil {
			ss.logger.Error(ctx, "ssh_scan_failed",
				observability.F("error", err.Error()),
			)
		}
		filesScanned += count
	}

	if ss.isScanEnabled(ScanTypeTokens) {
		count, err := ss.scanTokenSecurity(ctx, targetPath)
		if err != nil {
			ss.logger.Error(ctx, "token_scan_failed",
				observability.F("error", err.Error()),
			)
		}
		filesScanned += count
	}

	if ss.isScanEnabled(ScanTypeConfig) {
		count, err := ss.scanConfigSecurity(ctx, targetPath)
		if err != nil {
			ss.logger.Error(ctx, "config_scan_failed",
				observability.F("error", err.Error()),
			)
		}
		filesScanned += count
	}

	if ss.isScanEnabled(ScanTypePermissions) {
		count, err := ss.scanPermissions(ctx, targetPath)
		if err != nil {
			ss.logger.Error(ctx, "permissions_scan_failed",
				observability.F("error", err.Error()),
			)
		}
		filesScanned += count
	}

	if ss.isScanEnabled(ScanTypeSecrets) {
		count, err := ss.scanSecrets(ctx, targetPath)
		if err != nil {
			ss.logger.Error(ctx, "secrets_scan_failed",
				observability.F("error", err.Error()),
			)
		}
		filesScanned += count
	}

	// Complete scan result
	endTime := time.Now()
	result.EndTime = endTime
	result.Duration = endTime.Sub(startTime)
	result.FilesScanned = filesScanned
	result.Findings = ss.findings
	result.Summary = ss.generateSummary()

	ss.logger.Info(ctx, "security_scan_completed",
		observability.F("scan_id", scanID),
		observability.F("duration", result.Duration.String()),
		observability.F("files_scanned", filesScanned),
		observability.F("total_findings", len(ss.findings)),
		observability.F("risk_score", result.Summary.RiskScore),
	)

	return result, nil
}

// scanSSHSecurity scans for SSH-related security issues
func (ss *SecurityScanner) scanSSHSecurity(ctx context.Context, targetPath string) (int, error) {
	ss.logger.Debug(ctx, "scanning_ssh_security")

	filesScanned := 0
	sshDir := filepath.Join(targetPath, ".ssh")

	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.WalkDir(sshDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if ss.shouldSkipFile(path) {
			return nil
		}

		filesScanned++

		// Check file permissions
		if err := ss.checkSSHFilePermissions(ctx, path); err != nil {
			ss.logger.Error(ctx, "failed_to_check_ssh_permissions",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		// Scan file content
		if err := ss.scanFileWithRules(ctx, path, ScanTypeSSH); err != nil {
			ss.logger.Error(ctx, "failed_to_scan_ssh_file",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		return nil
	})

	return filesScanned, err
}

// scanTokenSecurity scans for exposed tokens and API keys
func (ss *SecurityScanner) scanTokenSecurity(ctx context.Context, targetPath string) (int, error) {
	ss.logger.Debug(ctx, "scanning_token_security")

	filesScanned := 0

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if ss.shouldSkipFile(path) {
			return nil
		}

		// Skip binary files
		if ss.isBinaryFile(path) {
			return nil
		}

		filesScanned++

		if err := ss.scanFileWithRules(ctx, path, ScanTypeTokens); err != nil {
			ss.logger.Error(ctx, "failed_to_scan_tokens",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		return nil
	})

	return filesScanned, err
}

// scanConfigSecurity scans configuration files for security issues
func (ss *SecurityScanner) scanConfigSecurity(ctx context.Context, targetPath string) (int, error) {
	ss.logger.Debug(ctx, "scanning_config_security")

	filesScanned := 0
	configExtensions := []string{".config", ".conf", ".json", ".yaml", ".yml", ".env"}

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if ss.shouldSkipFile(path) {
			return nil
		}

		// Check if it's a config file
		ext := filepath.Ext(path)
		isConfigFile := false
		for _, configExt := range configExtensions {
			if ext == configExt {
				isConfigFile = true
				break
			}
		}

		if !isConfigFile {
			return nil
		}

		filesScanned++

		if err := ss.scanFileWithRules(ctx, path, ScanTypeConfig); err != nil {
			ss.logger.Error(ctx, "failed_to_scan_config",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		return nil
	})

	return filesScanned, err
}

// scanPermissions scans file and directory permissions
func (ss *SecurityScanner) scanPermissions(ctx context.Context, targetPath string) (int, error) {
	ss.logger.Debug(ctx, "scanning_permissions")

	filesScanned := 0

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if ss.shouldSkipFile(path) {
			return nil
		}

		filesScanned++

		if err := ss.checkFilePermissions(ctx, path); err != nil {
			ss.logger.Error(ctx, "failed_to_check_permissions",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		return nil
	})

	return filesScanned, err
}

// scanSecrets scans for hardcoded secrets and credentials
func (ss *SecurityScanner) scanSecrets(ctx context.Context, targetPath string) (int, error) {
	ss.logger.Debug(ctx, "scanning_secrets")

	filesScanned := 0

	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if ss.shouldSkipFile(path) {
			return nil
		}

		if ss.isBinaryFile(path) {
			return nil
		}

		filesScanned++

		if err := ss.scanFileWithRules(ctx, path, ScanTypeSecrets); err != nil {
			ss.logger.Error(ctx, "failed_to_scan_secrets",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
		}

		return nil
	})

	return filesScanned, err
}

// scanFileWithRules scans a file using security rules
func (ss *SecurityScanner) scanFileWithRules(ctx context.Context, filePath string, scanType ScanType) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	for _, rule := range ss.rules {
		if !rule.Enabled || rule.Type != scanType {
			continue
		}

		if !ss.matchesFileType(filePath, rule.FileTypes) {
			continue
		}

		if rule.Pattern != nil {
			for lineNum, line := range lines {
				if matches := rule.Pattern.FindStringSubmatch(line); len(matches) > 0 {
					finding := SecurityFinding{
						ID:          ss.generateFindingID(),
						Type:        rule.Type,
						Severity:    rule.Severity,
						Title:       rule.Name,
						Description: rule.Description,
						FilePath:    filePath,
						LineNumber:  lineNum + 1,
						Evidence:    strings.TrimSpace(line),
						Remediation: rule.Remediation,
						CWE:         rule.CWE,
						Timestamp:   time.Now(),
						Metadata:    make(map[string]string),
					}

					if len(matches) > 1 {
						finding.Metadata["matched_pattern"] = matches[1]
					}

					ss.findings = append(ss.findings, finding)

					ss.logger.Warn(ctx, "security_finding_detected",
						observability.F("rule_id", rule.ID),
						observability.F("file", filePath),
						observability.F("line", lineNum+1),
						observability.F("severity", rule.Severity),
					)
				}
			}
		}
	}

	return nil
}

// checkSSHFilePermissions checks SSH file permissions
func (ss *SecurityScanner) checkSSHFilePermissions(ctx context.Context, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	mode := info.Mode()
	perm := mode.Perm()

	// SSH private keys should have 600 permissions
	if strings.Contains(strings.ToLower(filePath), "private") ||
		strings.HasSuffix(filePath, "id_rsa") ||
		strings.HasSuffix(filePath, "id_ed25519") ||
		strings.HasSuffix(filePath, "id_ecdsa") {
		if perm != 0600 {
			finding := SecurityFinding{
				ID:          ss.generateFindingID(),
				Type:        ScanTypePermissions,
				Severity:    SeverityHigh,
				Title:       "SSH Private Key Insecure Permissions",
				Description: fmt.Sprintf("SSH private key has permissions %o, should be 600", perm),
				FilePath:    filePath,
				Remediation: "chmod 600 " + filePath,
				Timestamp:   time.Now(),
				Metadata: map[string]string{
					"current_permissions":  fmt.Sprintf("%o", perm),
					"expected_permissions": "600",
				},
			}
			ss.findings = append(ss.findings, finding)
		}
	}

	return nil
}

// checkFilePermissions checks general file permissions
func (ss *SecurityScanner) checkFilePermissions(ctx context.Context, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	mode := info.Mode()
	perm := mode.Perm()

	// Check for world-writable files
	if perm&0002 != 0 {
		finding := SecurityFinding{
			ID:          ss.generateFindingID(),
			Type:        ScanTypePermissions,
			Severity:    SeverityMedium,
			Title:       "World-Writable File",
			Description: "File is writable by all users",
			FilePath:    filePath,
			Remediation: "Remove world-write permissions: chmod o-w " + filePath,
			Timestamp:   time.Now(),
			Metadata: map[string]string{
				"permissions": fmt.Sprintf("%o", perm),
			},
		}
		ss.findings = append(ss.findings, finding)
	}

	return nil
}

// Helper methods

func (ss *SecurityScanner) isScanEnabled(scanType ScanType) bool {
	for _, enabled := range ss.config.EnabledScans {
		if enabled == scanType {
			return true
		}
	}
	return false
}

func (ss *SecurityScanner) shouldSkipFile(filePath string) bool {
	for _, excludePath := range ss.config.ExcludePaths {
		if strings.Contains(filePath, excludePath) {
			return true
		}
	}

	// Skip common non-security relevant files
	skipPatterns := []string{
		".git/",
		".gitignore",
		".DS_Store",
		"node_modules/",
		".npm/",
		".cache/",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(filePath, pattern) {
			return true
		}
	}

	return false
}

func (ss *SecurityScanner) isBinaryFile(filePath string) bool {
	// Simple binary file detection
	binaryExtensions := []string{
		".exe", ".dll", ".so", ".dylib", ".bin", ".dat",
		".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip",
		".tar", ".gz", ".bz2", ".xz", ".7z",
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	for _, binExt := range binaryExtensions {
		if ext == binExt {
			return true
		}
	}

	return false
}

func (ss *SecurityScanner) matchesFileType(filePath string, fileTypes []string) bool {
	if len(fileTypes) == 0 {
		return true
	}

	for _, fileType := range fileTypes {
		if fileType == "*" {
			return true
		}

		if strings.HasPrefix(fileType, ".") {
			if strings.HasSuffix(filePath, fileType) {
				return true
			}
		} else {
			if strings.Contains(filepath.Base(filePath), fileType) {
				return true
			}
		}
	}

	return false
}

func (ss *SecurityScanner) generateScanID() string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("scan_%d", timestamp)
}

func (ss *SecurityScanner) generateFindingID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (ss *SecurityScanner) generateSummary() ScanSummary {
	summary := ScanSummary{}

	for _, finding := range ss.findings {
		summary.TotalFindings++

		switch finding.Severity {
		case SeverityCritical:
			summary.CriticalFindings++
		case SeverityHigh:
			summary.HighFindings++
		case SeverityMedium:
			summary.MediumFindings++
		case SeverityLow:
			summary.LowFindings++
		case SeverityInfo:
			summary.InfoFindings++
		}

		if finding.Fixed {
			summary.FixedFindings++
		}
	}

	// Calculate risk score (0-100)
	summary.RiskScore = summary.CriticalFindings*20 + summary.HighFindings*10 + summary.MediumFindings*5 + summary.LowFindings*2 + summary.InfoFindings*1

	if summary.RiskScore > 100 {
		summary.RiskScore = 100
	}

	return summary
}

// GetFindings returns all security findings
func (ss *SecurityScanner) GetFindings() []SecurityFinding {
	return ss.findings
}

// GetRules returns all security rules
func (ss *SecurityScanner) GetRules() map[string]*SecurityRule {
	return ss.rules
}

// AddRule adds a custom security rule
func (ss *SecurityScanner) AddRule(rule *SecurityRule) error {
	if rule.PatternStr != "" {
		pattern, err := regexp.Compile(`(?i)` + rule.PatternStr)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		rule.Pattern = pattern
	}

	ss.rules[rule.ID] = rule
	return nil
}
