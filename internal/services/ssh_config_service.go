package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/techishthoughts/GitPersona/internal/execrunner"
	"github.com/techishthoughts/GitPersona/internal/observability"
)

// SSHConfigService manages SSH configuration for multiple GitHub accounts
type SSHConfigService struct {
	logger observability.Logger
	runner execrunner.CmdRunner
}

// NewSSHConfigService creates a new SSH config service
func NewSSHConfigService(logger observability.Logger, runner execrunner.CmdRunner) *SSHConfigService {
	if runner == nil {
		runner = &execrunner.RealCmdRunner{}
	}

	return &SSHConfigService{
		logger: logger,
		runner: runner,
	}
}

// SSHConfigEntry represents an SSH config entry for a GitHub account
type SSHConfigEntry struct {
	Host                     string `json:"host"`
	HostName                 string `json:"hostname"`
	User                     string `json:"user"`
	IdentityFile             string `json:"identity_file"`
	UseKeychain              bool   `json:"use_keychain"`
	AddKeysToAgent           bool   `json:"add_keys_to_agent"`
	IdentitiesOnly           bool   `json:"identities_only"`
	ClearAllForwardings      bool   `json:"clear_all_forwardings"`
	PreferredAuthentications string `json:"preferred_authentications"`
	Description              string `json:"description"`
}

// SSHConfigResult represents the result of SSH config operations
type SSHConfigResult struct {
	ConfigPath      string           `json:"config_path"`
	Exists          bool             `json:"exists"`
	Valid           bool             `json:"valid"`
	Entries         []SSHConfigEntry `json:"entries"`
	Issues          []SSHConfigIssue `json:"issues"`
	Recommendations []string         `json:"recommendations"`
}

// SSHConfigIssue represents an SSH config issue
type SSHConfigIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Fix         string `json:"fix"`
	Automated   bool   `json:"automated"`
}

// GetSSHConfigPath returns the path to the SSH config file
func (s *SSHConfigService) GetSSHConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.ssh/config"
	}
	return filepath.Join(homeDir, ".ssh", "config")
}

// ReadSSHConfig reads and parses the SSH config file
func (s *SSHConfigService) ReadSSHConfig(ctx context.Context) (*SSHConfigResult, error) {
	s.logger.Info(ctx, "reading_ssh_config")

	configPath := s.GetSSHConfigPath()
	result := &SSHConfigResult{
		ConfigPath:      configPath,
		Entries:         []SSHConfigEntry{},
		Issues:          []SSHConfigIssue{},
		Recommendations: []string{},
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		result.Exists = false
		result.Issues = append(result.Issues, SSHConfigIssue{
			Type:        "missing_config",
			Severity:    "medium",
			Description: "SSH config file does not exist",
			Fix:         "Create SSH config file with proper entries",
			Automated:   true,
		})
		return result, nil
	}

	result.Exists = true

	// Read config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		result.Issues = append(result.Issues, SSHConfigIssue{
			Type:        "config_unreadable",
			Severity:    "high",
			Description: fmt.Sprintf("Cannot read SSH config file: %v", err),
			Fix:         "Check file permissions and ownership",
			Automated:   false,
		})
		return result, nil
	}

	// Parse config entries
	entries, issues := s.parseSSHConfig(string(content))
	result.Entries = entries
	result.Issues = append(result.Issues, issues...)

	// Validate entries
	s.validateSSHConfigEntries(ctx, result)

	// Generate recommendations
	s.generateSSHConfigRecommendations(ctx, result)

	// Determine overall validity
	result.Valid = len(result.Issues) == 0

	s.logger.Info(ctx, "ssh_config_read_complete",
		observability.F("entries_count", len(result.Entries)),
		observability.F("issues_count", len(result.Issues)),
		observability.F("valid", result.Valid),
	)

	return result, nil
}

// parseSSHConfig parses SSH config content into entries
func (s *SSHConfigService) parseSSHConfig(content string) ([]SSHConfigEntry, []SSHConfigIssue) {
	entries := []SSHConfigEntry{}
	issues := []SSHConfigIssue{}

	lines := strings.Split(content, "\n")
	currentEntry := &SSHConfigEntry{}
	inHostBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for Host directive
		if strings.HasPrefix(line, "Host ") {
			// Save previous entry if it exists
			if inHostBlock && currentEntry.Host != "" {
				entries = append(entries, *currentEntry)
			}

			// Start new entry
			currentEntry = &SSHConfigEntry{
				Host: strings.TrimSpace(strings.TrimPrefix(line, "Host")),
			}
			inHostBlock = true
			continue
		}

		// Parse other directives
		if inHostBlock {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				directive := strings.ToLower(parts[0])
				value := strings.Join(parts[1:], " ")

				switch directive {
				case "hostname":
					currentEntry.HostName = value
				case "user":
					currentEntry.User = value
				case "identityfile":
					currentEntry.IdentityFile = value
				case "usekeychain":
					currentEntry.UseKeychain = strings.ToLower(value) == "yes"
				case "addkeystoagent":
					currentEntry.AddKeysToAgent = strings.ToLower(value) == "yes"
				case "identitiesonly":
					currentEntry.IdentitiesOnly = strings.ToLower(value) == "yes"
				case "clearallforwardings":
					currentEntry.ClearAllForwardings = strings.ToLower(value) == "yes"
				case "preferredauthentications":
					currentEntry.PreferredAuthentications = value
				}
			}
		}
	}

	// Add the last entry
	if inHostBlock && currentEntry.Host != "" {
		entries = append(entries, *currentEntry)
	}

	return entries, issues
}

// validateSSHConfigEntries validates SSH config entries
func (s *SSHConfigService) validateSSHConfigEntries(ctx context.Context, result *SSHConfigResult) {
	for _, entry := range result.Entries {
		// Check for required fields
		if entry.Host == "" {
			result.Issues = append(result.Issues, SSHConfigIssue{
				Type:        "missing_host",
				Severity:    "high",
				Description: "SSH config entry missing Host directive",
				Fix:         "Add Host directive to SSH config entry",
				Automated:   false,
			})
		}

		if entry.HostName == "" {
			result.Issues = append(result.Issues, SSHConfigIssue{
				Type:        "missing_hostname",
				Severity:    "high",
				Description: fmt.Sprintf("SSH config entry '%s' missing HostName", entry.Host),
				Fix:         "Add HostName github.com to SSH config entry",
				Automated:   true,
			})
		}

		if entry.IdentityFile == "" {
			result.Issues = append(result.Issues, SSHConfigIssue{
				Type:        "missing_identity_file",
				Severity:    "high",
				Description: fmt.Sprintf("SSH config entry '%s' missing IdentityFile", entry.Host),
				Fix:         "Add IdentityFile path to SSH config entry",
				Automated:   false,
			})
		} else {
			// Check if identity file exists
			if _, err := os.Stat(entry.IdentityFile); os.IsNotExist(err) {
				result.Issues = append(result.Issues, SSHConfigIssue{
					Type:        "missing_identity_file",
					Severity:    "high",
					Description: fmt.Sprintf("SSH key file does not exist: %s", entry.IdentityFile),
					Fix:         "Generate SSH key or update path in config",
					Automated:   false,
				})
			}
		}

		// Check for best practices
		if !entry.IdentitiesOnly {
			result.Issues = append(result.Issues, SSHConfigIssue{
				Type:        "missing_identities_only",
				Severity:    "medium",
				Description: fmt.Sprintf("SSH config entry '%s' should use IdentitiesOnly yes", entry.Host),
				Fix:         "Add IdentitiesOnly yes to prevent key conflicts",
				Automated:   true,
			})
		}

		// Update entry with description
		for j, e := range result.Entries {
			if e.Host == entry.Host {
				result.Entries[j].Description = s.generateEntryDescription(entry)
				break
			}
		}
	}
}

// generateEntryDescription generates a description for an SSH config entry
func (s *SSHConfigService) generateEntryDescription(entry SSHConfigEntry) string {
	if strings.Contains(entry.Host, "github") {
		if strings.Contains(entry.Host, "fanduel") || strings.Contains(entry.Host, "work") {
			return "FanDuel/Work GitHub Account"
		}
		return "Personal GitHub Account"
	}
	return "SSH Host Configuration"
}

// generateSSHConfigRecommendations generates recommendations for SSH config
func (s *SSHConfigService) generateSSHConfigRecommendations(ctx context.Context, result *SSHConfigResult) {
	if !result.Exists {
		result.Recommendations = append(result.Recommendations, "Create SSH config file with proper host entries")
		return
	}

	if len(result.Entries) == 0 {
		result.Recommendations = append(result.Recommendations, "Add SSH config entries for your GitHub accounts")
		return
	}

	// Check for duplicate hosts
	hosts := make(map[string]int)
	for _, entry := range result.Entries {
		hosts[entry.Host]++
	}

	for host, count := range hosts {
		if count > 1 {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Remove duplicate SSH config entries for host '%s'", host))
		}
	}

	// Check for missing best practices
	hasIdentitiesOnly := false
	for _, entry := range result.Entries {
		if entry.IdentitiesOnly {
			hasIdentitiesOnly = true
			break
		}
	}

	if !hasIdentitiesOnly {
		result.Recommendations = append(result.Recommendations,
			"Add IdentitiesOnly yes to all SSH config entries to prevent key conflicts")
	}

	if len(result.Issues) == 0 {
		result.Recommendations = append(result.Recommendations, "SSH config is properly configured")
	}
}

// GenerateSSHConfig generates a complete SSH config for multiple GitHub accounts
func (s *SSHConfigService) GenerateSSHConfig(ctx context.Context, accounts map[string]SSHConfigEntry) (string, error) {
	s.logger.Info(ctx, "generating_ssh_config",
		observability.F("accounts_count", len(accounts)),
	)

	var config strings.Builder

	// Add header comment
	config.WriteString("# SSH Configuration for Multiple GitHub Accounts\n")
	config.WriteString("# Generated by GitPersona\n")
	config.WriteString("# This configuration prevents SSH key conflicts\n\n")

	// Add entries for each account
	for _, entry := range accounts {
		config.WriteString(fmt.Sprintf("# %s\n", entry.Description))
		config.WriteString(fmt.Sprintf("Host %s\n", entry.Host))
		config.WriteString(fmt.Sprintf("    HostName %s\n", entry.HostName))
		config.WriteString(fmt.Sprintf("    User %s\n", entry.User))
		config.WriteString(fmt.Sprintf("    IdentityFile %s\n", entry.IdentityFile))
		config.WriteString(fmt.Sprintf("    UseKeychain %s\n", boolToString(entry.UseKeychain)))
		config.WriteString(fmt.Sprintf("    AddKeysToAgent %s\n", boolToString(entry.AddKeysToAgent)))
		config.WriteString(fmt.Sprintf("    IdentitiesOnly %s\n", boolToString(entry.IdentitiesOnly)))
		config.WriteString(fmt.Sprintf("    ClearAllForwardings %s\n", boolToString(entry.ClearAllForwardings)))
		config.WriteString(fmt.Sprintf("    PreferredAuthentications %s\n", entry.PreferredAuthentications))
		config.WriteString("\n")
	}

	// Add default GitHub entry
	config.WriteString("# Default GitHub (fallback)\n")
	config.WriteString("Host github.com\n")
	config.WriteString("    HostName github.com\n")
	config.WriteString("    User git\n")
	config.WriteString("    IdentitiesOnly yes\n")
	config.WriteString("    PreferredAuthentications publickey\n\n")

	s.logger.Info(ctx, "ssh_config_generated",
		observability.F("config_length", config.Len()),
	)

	return config.String(), nil
}

// WriteSSHConfig writes SSH config to file
func (s *SSHConfigService) WriteSSHConfig(ctx context.Context, config string) error {
	s.logger.Info(ctx, "writing_ssh_config")

	configPath := s.GetSSHConfigPath()

	// Create backup if file exists
	if _, err := os.Stat(configPath); err == nil {
		backupPath := configPath + ".backup." + fmt.Sprintf("%d", os.Getpid())
		if err := s.runner.Run(ctx, "cp", configPath, backupPath); err != nil {
			s.logger.Warn(ctx, "failed_to_create_backup",
				observability.F("backup_path", backupPath),
				observability.F("error", err.Error()),
			)
		} else {
			s.logger.Info(ctx, "ssh_config_backup_created",
				observability.F("backup_path", backupPath),
			)
		}
	}

	// Write new config
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		s.logger.Error(ctx, "failed_to_write_ssh_config",
			observability.F("config_path", configPath),
			observability.F("error", err.Error()),
		)
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	s.logger.Info(ctx, "ssh_config_written_successfully",
		observability.F("config_path", configPath),
	)

	return nil
}

// FixSSHConfigIssues automatically fixes SSH config issues
func (s *SSHConfigService) FixSSHConfigIssues(ctx context.Context, result *SSHConfigResult) error {
	s.logger.Info(ctx, "fixing_ssh_config_issues",
		observability.F("fixable_issues", len(result.Issues)),
	)

	fixedCount := 0

	for i, issue := range result.Issues {
		if !issue.Automated {
			continue
		}

		switch issue.Type {
		case "missing_config":
			// Create basic SSH config
			basicConfig := `# SSH Configuration for Multiple GitHub Accounts
# Generated by GitPersona

# Add your GitHub account entries here
# Example:
# Host github-work
#     HostName github.com
#     User git
#     IdentityFile ~/.ssh/id_ed25519_work
#     IdentitiesOnly yes
`
			if err := s.WriteSSHConfig(ctx, basicConfig); err != nil {
				s.logger.Error(ctx, "failed_to_create_ssh_config",
					observability.F("error", err.Error()),
				)
			} else {
				result.Issues[i].Fix = "Fixed automatically"
				fixedCount++
			}

		case "missing_hostname":
			// This would require more complex parsing and fixing
			// For now, just mark as fixed
			result.Issues[i].Fix = "Requires manual intervention"
		}
	}

	s.logger.Info(ctx, "ssh_config_issues_fix_complete",
		observability.F("fixed_count", fixedCount),
	)

	return nil
}

// boolToString converts boolean to "yes"/"no" string
func boolToString(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
