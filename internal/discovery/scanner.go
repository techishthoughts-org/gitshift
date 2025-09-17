package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"github.com/techishthoughts/GitPersona/internal/models"
)

// AccountDiscovery handles automatic detection of existing Git accounts
type AccountDiscovery struct {
	homeDir string
}

// NewAccountDiscovery creates a new account discovery service
func NewAccountDiscovery() *AccountDiscovery {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}

	return &AccountDiscovery{
		homeDir: homeDir,
	}
}

// DiscoveredAccount represents an account found during discovery
type DiscoveredAccount struct {
	*models.Account
	Source      string // where it was found
	Confidence  int    // confidence level (1-10)
	Conflicting bool   // if there are conflicting accounts
}

// ScanExistingAccounts scans the system for existing Git configurations
func (d *AccountDiscovery) ScanExistingAccounts() ([]*DiscoveredAccount, error) {
	var discovered []*DiscoveredAccount

	// 1. Skip global Git configuration (can contain outdated/irrelevant data)
	// if globalAccounts, err := d.scanGlobalGitConfig(); err == nil {
	//	discovered = append(discovered, globalAccounts...)
	// }

	// 2. Scan Git config files in ~/.config/git/
	if configAccounts, err := d.scanGitConfigFiles(); err == nil {
		discovered = append(discovered, configAccounts...)
	}

	// 3. Scan SSH configuration for GitHub keys
	if sshAccounts, err := d.scanSSHConfig(); err == nil {
		discovered = append(discovered, sshAccounts...)
	}

	// 4. Check GitHub CLI authentication
	if ghAccounts, err := d.scanGitHubCLI(); err == nil {
		discovered = append(discovered, ghAccounts...)
	}

	// 5. Merge and deduplicate accounts
	merged := d.mergeDiscoveredAccounts(discovered)

	return merged, nil
}

// scanGlobalGitConfig scans the global ~/.gitconfig file
func (d *AccountDiscovery) scanGlobalGitConfig() ([]*DiscoveredAccount, error) {
	gitConfigPath := filepath.Join(d.homeDir, ".gitconfig")

	if _, err := os.Stat(gitConfigPath); os.IsNotExist(err) {
		return nil, nil
	}

	// Read and parse the Git config file manually since Viper has issues with INI format
	content, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return nil, err
	}

	// Simple INI parser for Git config
	var name, email, sshCommand string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			continue // Skip section headers
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "name":
					name = value
				case "email":
					email = value
				case "sshCommand":
					sshCommand = value
				}
			}
		}
	}

	if name == "" || email == "" {
		return nil, nil
	}

	// Extract SSH key from sshCommand
	var sshKeyPath string
	if sshCommand != "" {
		if key := d.extractSSHKeyFromCommand(sshCommand); key != "" {
			sshKeyPath = key
		}
	}

	// Generate alias from email domain or name
	alias := d.generateAlias(email, name, "global")

	account := &DiscoveredAccount{
		Account: &models.Account{
			Alias:       alias,
			Name:        name,
			Email:       email,
			SSHKeyPath:  sshKeyPath,
			Description: "Found in global Git configuration",
		},
		Source:     "~/.gitconfig",
		Confidence: 8,
	}

	return []*DiscoveredAccount{account}, nil
}

// scanGitConfigFiles scans Git config files in ~/.config/git/
func (d *AccountDiscovery) scanGitConfigFiles() ([]*DiscoveredAccount, error) {
	configDir := filepath.Join(d.homeDir, ".config", "git")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, nil
	}

	var discovered []*DiscoveredAccount

	files, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "gitconfig-") {
			continue
		}

		configPath := filepath.Join(configDir, file.Name())
		if accounts, err := d.parseGitConfigFile(configPath); err == nil {
			discovered = append(discovered, accounts...)
		}
	}

	return discovered, nil
}

// parseGitConfigFile parses a specific Git config file
func (d *AccountDiscovery) parseGitConfigFile(configPath string) ([]*DiscoveredAccount, error) {
	viper := viper.New()
	viper.SetConfigFile(configPath)
	viper.SetConfigType("ini")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	name := viper.GetString("user.name")
	email := viper.GetString("user.email")
	githubUser := viper.GetString("github.user")
	sshCommand := viper.GetString("core.sshCommand")

	if name == "" || email == "" {
		return nil, nil
	}

	// Extract SSH key from sshCommand
	var sshKeyPath string
	if sshCommand != "" {
		if key := d.extractSSHKeyFromCommand(sshCommand); key != "" {
			sshKeyPath = key
		}
	}

	// Generate alias from filename (gitconfig-work -> work)
	filename := filepath.Base(configPath)
	alias := strings.TrimPrefix(filename, "gitconfig-")
	if alias == filename { // fallback
		alias = d.generateAlias(email, name, "config")
	}

	account := &DiscoveredAccount{
		Account: &models.Account{
			Alias:          alias,
			Name:           name,
			Email:          email,
			SSHKeyPath:     sshKeyPath,
			GitHubUsername: githubUser,
			Description:    fmt.Sprintf("Found in %s", filename),
		},
		Source:     configPath,
		Confidence: 9,
	}

	return []*DiscoveredAccount{account}, nil
}

// scanSSHConfig scans SSH configuration for GitHub-related keys
func (d *AccountDiscovery) scanSSHConfig() ([]*DiscoveredAccount, error) {
	sshConfigPath := filepath.Join(d.homeDir, ".ssh", "config")

	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(sshConfigPath)
	if err != nil {
		return nil, err
	}

	var discovered []*DiscoveredAccount

	// Parse SSH config for GitHub hosts
	hosts := d.parseSSHHosts(string(content))

	for _, host := range hosts {
		if host.IsGitHub && host.IdentityFile != "" {
			// Try to determine account info from key name
			alias := d.generateAliasFromSSHKey(host.IdentityFile)

			account := &DiscoveredAccount{
				Account: &models.Account{
					Alias:       alias,
					Name:        "", // Will be filled later if found
					Email:       "", // Will be filled later if found
					SSHKeyPath:  host.IdentityFile,
					Description: fmt.Sprintf("Found SSH key for %s", host.Host),
				},
				Source:     "~/.ssh/config",
				Confidence: 6,
			}

			discovered = append(discovered, account)
		}
	}

	return discovered, nil
}

// scanGitHubCLI checks GitHub CLI for authenticated accounts
func (d *AccountDiscovery) scanGitHubCLI() ([]*DiscoveredAccount, error) {
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil // GitHub CLI not installed or not authenticated
	}

	var discovered []*DiscoveredAccount

	// Parse gh auth status output
	lines := strings.Split(string(output), "\n")
	var currentAccount *DiscoveredAccount

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Logged in to") && strings.Contains(line, "account") {
			// Extract account name: "✓ Logged in to github.com account username"
			re := regexp.MustCompile(`account\s+(\w+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				username := matches[1]

				// Try to enrich account with GitHub API data
				enrichedAccount := d.enrichAccountFromGitHubAPI(username)

				// Check if SSH key exists for this account
				sshKeyPath := d.findOrGenerateSSHKey(username, enrichedAccount.Email)

				currentAccount = &DiscoveredAccount{
					Account: &models.Account{
						Alias:          d.generateAlias("", username, "gh"),
						Name:           enrichedAccount.Name,
						Email:          enrichedAccount.Email,
						GitHubUsername: username,
						SSHKeyPath:     sshKeyPath,
						Description:    "Found via GitHub CLI authentication",
					},
					Source:     "gh auth status",
					Confidence: 7,
				}

				// Increase confidence if we got additional info
				if enrichedAccount.Name != "" {
					currentAccount.Confidence = 8
				}

				discovered = append(discovered, currentAccount)
			}
		}
	}

	return discovered, nil
}

// enrichAccountFromGitHubAPI tries to fetch additional user information from GitHub API
func (d *AccountDiscovery) enrichAccountFromGitHubAPI(username string) *models.Account {
	// Try to get user info from GitHub API using gh command
	cmd := exec.Command("gh", "api", "user")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &models.Account{} // Return empty account if API call fails
	}

	// Simple JSON parsing for the user data
	var userData struct {
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.Unmarshal(output, &userData); err != nil {
		return &models.Account{} // Return empty account if parsing fails
	}

	// Only return data if it matches the username we're looking for
	if userData.Login == username {
		return &models.Account{
			Name:  userData.Name,
			Email: userData.Email,
		}
	}

	return &models.Account{}
}

// SSHHost represents an SSH host configuration
type SSHHost struct {
	Host         string
	HostName     string
	IdentityFile string
	User         string
	IsGitHub     bool
}

// parseSSHHosts parses SSH config content for host configurations
func (d *AccountDiscovery) parseSSHHosts(content string) []SSHHost {
	var hosts []SSHHost
	var currentHost *SSHHost

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "Host ") {
			// Save previous host
			if currentHost != nil {
				hosts = append(hosts, *currentHost)
			}

			// Start new host
			hostName := strings.TrimPrefix(line, "Host ")
			currentHost = &SSHHost{
				Host:     hostName,
				IsGitHub: strings.Contains(hostName, "github") || hostName == "github.com",
			}
		} else if currentHost != nil {
			// Parse host properties
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "HostName":
				currentHost.HostName = value
				if strings.Contains(value, "github") {
					currentHost.IsGitHub = true
				}
			case "IdentityFile":
				currentHost.IdentityFile = d.expandPath(value)
			case "User":
				currentHost.User = value
			}
		}
	}

	// Add last host
	if currentHost != nil {
		hosts = append(hosts, *currentHost)
	}

	return hosts
}

// mergeDiscoveredAccounts merges and deduplicates discovered accounts
func (d *AccountDiscovery) mergeDiscoveredAccounts(accounts []*DiscoveredAccount) []*DiscoveredAccount {
	// Group accounts by email or name
	groups := make(map[string][]*DiscoveredAccount)

	fmt.Printf("🔍 Merging %d accounts...\n", len(accounts))

	for _, account := range accounts {
		// Try to find a good grouping key
		var key string

		// First, check if this account should be grouped by SSH key (highest priority for linking)
		if account.SSHKeyPath != "" {
			normalizedSSHPath := d.expandPath(account.SSHKeyPath)

			// Check if any other account has the same SSH key
			for _, otherAccount := range accounts {
				if otherAccount != account && otherAccount.SSHKeyPath != "" {
					otherNormalizedPath := d.expandPath(otherAccount.SSHKeyPath)
					if normalizedSSHPath == otherNormalizedPath {
						key = "ssh:" + normalizedSSHPath

						break
					}
				}
			}

			// If no SSH key grouping, check if this SSH key contains a GitHub username
			if key == "" {
				for _, otherAccount := range accounts {
					if otherAccount != account && otherAccount.GitHubUsername != "" && otherAccount.SSHKeyPath == "" {
						// Check if the SSH key name contains the GitHub username
						sshKeyName := filepath.Base(account.SSHKeyPath)
						sshKeyName = strings.TrimPrefix(sshKeyName, "id_rsa_")
						sshKeyName = strings.TrimPrefix(sshKeyName, "id_ed25519_")
						sshKeyName = strings.TrimSuffix(sshKeyName, ".pub")

						if sshKeyName == otherAccount.GitHubUsername {
							key = "ssh+github:" + normalizedSSHPath

							break
						}
					}
				}
			}

			// If still no key, use SSH key path
			if key == "" {
				key = "ssh:" + normalizedSSHPath
			}
		}

		// If no SSH key grouping, try other grouping methods
		if key == "" {
			// Priority 1: Group by email (most reliable)
			if account.Email != "" {
				key = "email:" + account.Email
			} else if account.Name != "" {
				// Priority 2: Group by name if no email
				key = "name:" + account.Name
			} else if account.GitHubUsername != "" {
				// Priority 3: Check if this GitHub username should be grouped with an SSH key
				shouldGroupWithSSH := false
				for _, otherAccount := range accounts {
					if otherAccount != account && otherAccount.SSHKeyPath != "" {
						// Check if the SSH key name contains this GitHub username
						sshKeyName := filepath.Base(otherAccount.SSHKeyPath)
						sshKeyName = strings.TrimPrefix(sshKeyName, "id_rsa_")
						sshKeyName = strings.TrimPrefix(sshKeyName, "id_ed25519_")
						sshKeyName = strings.TrimSuffix(sshKeyName, ".pub")

						if sshKeyName == account.GitHubUsername {
							normalizedSSHPath := d.expandPath(otherAccount.SSHKeyPath)
							key = "ssh:" + normalizedSSHPath

							shouldGroupWithSSH = true
							break
						}
					}
				}

				// If not grouped with SSH, use GitHub username
				if !shouldGroupWithSSH {
					key = "github:" + account.GitHubUsername
				}
			} else {
				// Fallback to alias
				key = "alias:" + account.Alias
			}
		}

		groups[key] = append(groups[key], account)
	}

	var merged []*DiscoveredAccount

	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}

		// Merge multiple accounts with same key
		best := group[0]
		for _, account := range group[1:] {
			// Choose account with highest confidence
			if account.Confidence > best.Confidence {
				best = account
			}

			// Merge missing fields
			if best.Name == "" && account.Name != "" {
				best.Name = account.Name
			}
			if best.Email == "" && account.Email != "" {
				best.Email = account.Email
			}
			if best.SSHKeyPath == "" && account.SSHKeyPath != "" {
				best.SSHKeyPath = account.SSHKeyPath
			}
			if best.GitHubUsername == "" && account.GitHubUsername != "" {
				best.GitHubUsername = account.GitHubUsername
			}
		}

		// Mark as conflicting if there were multiple sources
		best.Conflicting = true
		best.Source = fmt.Sprintf("Merged from %d sources", len(group))

		merged = append(merged, best)
	}

	return merged
}

// Helper functions

func (d *AccountDiscovery) extractSSHKeyFromCommand(sshCommand string) string {
	// Extract key path from "ssh -i ~/.ssh/key_name"
	re := regexp.MustCompile(`-i\s+([^\s]+)`)
	if matches := re.FindStringSubmatch(sshCommand); len(matches) > 1 {
		return d.expandPath(matches[1])
	}
	return ""
}

func (d *AccountDiscovery) generateAlias(email, name, fallback string) string {
	if email != "" {
		// Use domain part of email
		parts := strings.Split(email, "@")
		if len(parts) == 2 {
			domain := parts[1]
			domain = strings.Split(domain, ".")[0]
			if domain != "gmail" && domain != "yahoo" && domain != "hotmail" {
				return domain
			}
		}
		// Use name part of email
		return parts[0]
	}

	if name != "" {
		// Use first name in lowercase
		parts := strings.Fields(name)
		if len(parts) > 0 {
			return strings.ToLower(parts[0])
		}
	}

	return fallback
}

func (d *AccountDiscovery) generateAliasFromSSHKey(keyPath string) string {
	filename := filepath.Base(keyPath)

	// Remove common prefixes/suffixes
	filename = strings.TrimPrefix(filename, "id_rsa_")
	filename = strings.TrimPrefix(filename, "id_ed25519_")
	filename = strings.TrimSuffix(filename, ".pub")

	if filename == "id_rsa" || filename == "id_ed25519" {
		return "default"
	}

	return filename
}

func (d *AccountDiscovery) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(d.homeDir, path[2:])
	}
	return path
}

// findOrGenerateSSHKey finds existing SSH key or generates a new one
func (d *AccountDiscovery) findOrGenerateSSHKey(username, email string) string {
	// Look for existing SSH keys that might match this account
	sshDir := filepath.Join(d.homeDir, ".ssh")

	// Common SSH key patterns to check
	patterns := []string{
		fmt.Sprintf("id_ed25519_%s", username),
		fmt.Sprintf("id_rsa_%s", username),
		fmt.Sprintf("id_ed25519"),
		fmt.Sprintf("id_rsa"),
	}

	for _, pattern := range patterns {
		keyPath := filepath.Join(sshDir, pattern)
		if _, err := os.Stat(keyPath); err == nil {
			// Key exists, return it
			return keyPath
		}
	}

	// No existing key found, generate a new one
	fmt.Printf("🔑 No SSH key found for %s, generating new one...\n", username)

	// Generate SSH key
	keyPath := filepath.Join(sshDir, fmt.Sprintf("id_ed25519_%s", username))
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Failed to generate SSH key: %v\n", err)
		return ""
	}

	fmt.Printf("✅ Generated SSH key: %s\n", keyPath)

	// Try to add to GitHub if authenticated
	if d.isGitHubCLIAuthenticated() {
		fmt.Printf("🚀 Adding SSH key to GitHub account...\n")
		addCmd := exec.Command("gh", "ssh-key", "add", keyPath+".pub", "--title", fmt.Sprintf("gitpersona-%s", username))
		if err := addCmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to add SSH key to GitHub: %v\n", err)
			fmt.Printf("💡 Please add this key manually: https://github.com/settings/keys\n")
		} else {
			fmt.Printf("🎉 SSH key added to GitHub account!\n")
		}
	} else {
		fmt.Printf("💡 Please add this SSH key to your GitHub account:\n")
		fmt.Printf("   https://github.com/settings/keys\n")
	}

	return keyPath
}

// isGitHubCLIAuthenticated checks if GitHub CLI is authenticated
func (d *AccountDiscovery) isGitHubCLIAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}
