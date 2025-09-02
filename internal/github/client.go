package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v57/github"
	"github.com/techishthoughts/GitPersona/internal/models"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

// Client handles GitHub API interactions
type Client struct {
	client *github.Client
	ctx    context.Context
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	ctx := context.Background()

	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	return &Client{
		client: client,
		ctx:    ctx,
	}
}

// UserInfo contains GitHub user information
type UserInfo struct {
	Login     string
	Name      string
	Email     string
	AvatarURL string
	Company   string
	Bio       string
	Location  string
}

// FetchUserInfo fetches user information from GitHub API
func (c *Client) FetchUserInfo(username string) (*UserInfo, error) {
	user, _, err := c.client.Users.Get(c.ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info for '%s': %w", username, err)
	}

	// Get primary email if authenticated
	var email string
	if c.isAuthenticated() {
		fmt.Printf("🔍 Fetching private email addresses...\n")
		if emails, _, err := c.client.Users.ListEmails(c.ctx, nil); err == nil {
			// Try to find primary verified email first
			for _, e := range emails {
				if e.GetPrimary() && e.GetVerified() {
					email = e.GetEmail()
					fmt.Printf("✅ Found primary email: %s\n", email)
					break
				}
			}
			// If no primary found, use first verified email
			if email == "" {
				for _, e := range emails {
					if e.GetVerified() {
						email = e.GetEmail()
						fmt.Printf("✅ Found verified email: %s\n", email)
						break
					}
				}
			}
		} else {
			fmt.Printf("⚠️  Could not fetch private emails: %v\n", err)
		}
	}

	// Use public email if no private email found
	if email == "" && user.GetEmail() != "" {
		email = user.GetEmail()
		fmt.Printf("✅ Using public email: %s\n", email)
	}

	return &UserInfo{
		Login:     user.GetLogin(),
		Name:      user.GetName(),
		Email:     email,
		AvatarURL: user.GetAvatarURL(),
		Company:   user.GetCompany(),
		Bio:       user.GetBio(),
		Location:  user.GetLocation(),
	}, nil
}

// isAuthenticated checks if the client has authentication
func (c *Client) isAuthenticated() bool {
	_, _, err := c.client.Users.Get(c.ctx, "")
	return err == nil
}

// SetupAccountFromUsername creates a complete account setup from just a GitHub username
func (c *Client) SetupAccountFromUsername(username string, alias string, providedEmail string, providedName string) (*models.Account, error) {
	fmt.Printf("🔍 Fetching GitHub user information for @%s...\n", username)

	// Fetch user info from GitHub API
	userInfo, err := c.FetchUserInfo(username)
	if err != nil {
		return nil, err
	}

	// Generate alias if not provided
	if alias == "" {
		alias = c.generateAlias(userInfo)
	}

	// Handle name with priority: provided name > GitHub API name > username fallback
	finalName := providedName
	if finalName == "" {
		finalName = userInfo.Name
		if finalName == "" {
			finalName = userInfo.Login // fallback to username if no name
			fmt.Printf("💡 Using GitHub username as display name: %s\n", finalName)
		} else {
			fmt.Printf("✅ Using GitHub display name: %s\n", finalName)
		}
	} else {
		fmt.Printf("✅ Using provided name: %s\n", finalName)
	}

	fmt.Printf("✅ Found GitHub user: %s (%s)\n", finalName, userInfo.Login)

	// Handle email with priority: provided email > GitHub API email > no-reply email
	email := providedEmail
	if email != "" {
		fmt.Printf("✅ Using provided email: %s\n", email)
	} else {
		email = userInfo.Email
		if email == "" {
			email = c.promptForEmail(userInfo.Login)
		}
	}

	// Generate SSH key
	fmt.Printf("🔑 Generating SSH key for %s...\n", alias)
	sshKeyPath, err := c.generateSSHKey(alias, email)
	if err != nil {
		fmt.Printf("⚠️  SSH key generation failed: %v\n", err)
		fmt.Println("   You can add an SSH key manually later.")
		sshKeyPath = ""
	} else {
		fmt.Printf("✅ SSH key generated: %s\n", sshKeyPath)

		// If authenticated, automatically upload SSH key to GitHub
		if c.isAuthenticated() {
			fmt.Printf("🚀 Automatically uploading SSH key to GitHub...\n")
			keyTitle := fmt.Sprintf("gitpersona-%s-%s", alias, userInfo.Login)
			if err := c.UploadSSHKeyToGitHub(sshKeyPath, keyTitle); err != nil {
				fmt.Printf("⚠️  Failed to upload SSH key automatically: %v\n", err)
				fmt.Println("💡 You can add it manually at: https://github.com/settings/keys")
				c.showSSHPublicKey(sshKeyPath)
			} else {
				fmt.Printf("🎉 SSH key automatically configured in your GitHub account!\n")
			}
		} else {
			// Show public key for manual addition
			if err := c.showSSHPublicKey(sshKeyPath); err == nil {
				fmt.Println("💡 Please add this SSH key to your GitHub account:")
				fmt.Println("   → https://github.com/settings/keys")
			}
		}
	}

	// Create account
	account := models.NewAccount(alias, finalName, email, sshKeyPath)
	account.GitHubUsername = userInfo.Login
	account.Description = fmt.Sprintf("Auto-setup from GitHub @%s", userInfo.Login)

	return account, nil
}

// generateAlias creates a suitable alias from user info
func (c *Client) generateAlias(userInfo *UserInfo) string {
	// Try different strategies
	candidates := []string{
		strings.ToLower(userInfo.Login),
	}

	// Add company-based alias if available
	if userInfo.Company != "" {
		company := strings.ToLower(strings.Fields(userInfo.Company)[0])
		company = strings.Trim(company, "@")
		if company != "" && company != "github" {
			candidates = append(candidates, company)
		}
	}

	// Add name-based alias
	if userInfo.Name != "" {
		parts := strings.Fields(userInfo.Name)
		if len(parts) > 0 {
			firstName := strings.ToLower(parts[0])
			candidates = append(candidates, firstName)
		}
	}

	// Return the first reasonable candidate
	for _, candidate := range candidates {
		if len(candidate) > 2 && candidate != "user" {
			return candidate
		}
	}

	return userInfo.Login
}

// promptForEmail prompts user for email if not available from GitHub API
func (c *Client) promptForEmail(username string) string {
	fmt.Printf("📧 GitHub user @%s has no accessible email addresses.\n", username)

	// Generate a sensible default based on username
	defaultEmail := fmt.Sprintf("%s@users.noreply.github.com", username)

	fmt.Printf("💡 Using GitHub no-reply email: %s\n", defaultEmail)
	fmt.Printf("   (You can update this later with: gitpersona add %s --email your@example.com --overwrite)\n", username)

	return defaultEmail
}

// generateSSHKey generates a new SSH key pair
func (c *Client) generateSSHKey(alias, email string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", err
	}

	// Generate key file names
	keyName := fmt.Sprintf("id_rsa_%s", alias)
	privateKeyPath := filepath.Join(sshDir, keyName)
	publicKeyPath := privateKeyPath + ".pub"

	// Check if key already exists
	if _, err := os.Stat(privateKeyPath); err == nil {
		fmt.Printf("🔑 SSH key already exists: %s\n", privateKeyPath)
		return privateKeyPath, nil
	}

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Save private key
	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}

	// Save public key
	publicKeyData := fmt.Sprintf("%s %s\n",
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))),
		email)

	if err := os.WriteFile(publicKeyPath, []byte(publicKeyData), 0644); err != nil {
		return "", fmt.Errorf("failed to write public key: %w", err)
	}

	// Add to SSH agent
	c.addKeyToSSHAgent(privateKeyPath)

	return privateKeyPath, nil
}

// addKeyToSSHAgent adds the SSH key to the SSH agent
func (c *Client) addKeyToSSHAgent(keyPath string) {
	cmd := exec.Command("ssh-add", keyPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Could not add key to SSH agent: %v\n", err)
		fmt.Println("   You may need to run: ssh-add " + keyPath)
	} else {
		fmt.Printf("✅ Added key to SSH agent\n")
	}
}

// showSSHPublicKey displays the public SSH key for copying
func (c *Client) showSSHPublicKey(privateKeyPath string) error {
	publicKeyPath := privateKeyPath + ".pub"
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	fmt.Println("\n🔑 Your new SSH public key:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Print(string(publicKeyData))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Try to copy to clipboard
	c.copyToClipboard(string(publicKeyData))

	return nil
}

// copyToClipboard attempts to copy text to system clipboard
func (c *Client) copyToClipboard(text string) {
	var cmd *exec.Cmd

	// Detect platform and use appropriate clipboard command
	switch {
	case commandExists("pbcopy"): // macOS
		cmd = exec.Command("pbcopy")
	case commandExists("xclip"): // Linux with xclip
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case commandExists("xsel"): // Linux with xsel
		cmd = exec.Command("xsel", "--clipboard", "--input")
	default:
		return // No clipboard support
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		fmt.Println("📋 SSH key copied to clipboard!")
	}
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// AuthenticateWithGitHub handles automatic GitHub authentication
func (c *Client) AuthenticateWithGitHub() error {
	fmt.Println("🔐 Setting up GitHub authentication...")

	// Check if GitHub CLI is installed
	if !commandExists("gh") {
		return fmt.Errorf("GitHub CLI (gh) is not installed. Please install it first: https://cli.github.com/")
	}

	// Check if already authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err == nil {
		fmt.Println("✅ Already authenticated with GitHub CLI")
		return nil
	}

	// Use device flow for automatic authentication without browser
	fmt.Println("🚀 Starting automatic GitHub authentication...")
	fmt.Println("📋 This will provide full access permissions for seamless account management")

	cmd = exec.Command("gh", "auth", "login",
		"--git-protocol", "ssh",
		"--scopes", "repo,read:user,user:email,admin:public_key",
		"--web")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	fmt.Println("✅ Successfully authenticated with GitHub!")
	fmt.Println("🎯 Full permissions granted for:")
	fmt.Println("   • Repository access")
	fmt.Println("   • User profile information")
	fmt.Println("   • Email addresses")
	fmt.Println("   • SSH key management")

	return nil
}

// GetGitHubToken gets an authenticated GitHub token from gh CLI
func (c *Client) GetGitHubToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// Repository represents a GitHub repository
type Repository struct {
	Name        string
	FullName    string
	Description string
	Private     bool
	Fork        bool
	Archived    bool
	Language    string
	Stars       int
	Forks       int
	UpdatedAt   string
	HTMLURL     string
}

// FetchUserRepositories fetches repositories for a given user
func (c *Client) FetchUserRepositories(username string) ([]*Repository, error) {
	opt := &github.RepositoryListOptions{
		Type:        "all",                            // all, owner, public, private, member
		ListOptions: github.ListOptions{PerPage: 100}, // Get up to 100 repos
	}

	repos, _, err := c.client.Repositories.List(c.ctx, username, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories for %s: %w", username, err)
	}

	var repositories []*Repository
	for _, repo := range repos {
		repositories = append(repositories, &Repository{
			Name:        repo.GetName(),
			FullName:    repo.GetFullName(),
			Description: repo.GetDescription(),
			Private:     repo.GetPrivate(),
			Fork:        repo.GetFork(),
			Archived:    repo.GetArchived(),
			Language:    repo.GetLanguage(),
			Stars:       repo.GetStargazersCount(),
			Forks:       repo.GetForksCount(),
			UpdatedAt:   repo.GetUpdatedAt().Format("2006-01-02"),
			HTMLURL:     repo.GetHTMLURL(),
		})
	}

	return repositories, nil
}

// FetchAuthenticatedUserRepositories fetches repositories for the authenticated user
func (c *Client) FetchAuthenticatedUserRepositories() ([]*Repository, error) {
	if !c.isAuthenticated() {
		return nil, fmt.Errorf("not authenticated with GitHub")
	}

	opt := &github.RepositoryListOptions{
		Type:        "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	repos, _, err := c.client.Repositories.List(c.ctx, "", opt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch authenticated user repositories: %w", err)
	}

	var repositories []*Repository
	for _, repo := range repos {
		repositories = append(repositories, &Repository{
			Name:        repo.GetName(),
			FullName:    repo.GetFullName(),
			Description: repo.GetDescription(),
			Private:     repo.GetPrivate(),
			Fork:        repo.GetFork(),
			Archived:    repo.GetArchived(),
			Language:    repo.GetLanguage(),
			Stars:       repo.GetStargazersCount(),
			Forks:       repo.GetForksCount(),
			UpdatedAt:   repo.GetUpdatedAt().Format("2006-01-02"),
			HTMLURL:     repo.GetHTMLURL(),
		})
	}

	return repositories, nil
}

// UploadSSHKeyToGitHub automatically uploads SSH key to GitHub account
func (c *Client) UploadSSHKeyToGitHub(keyPath, title string) error {
	if !c.isAuthenticated() {
		return fmt.Errorf("not authenticated with GitHub")
	}

	// Read public key
	publicKeyPath := keyPath + ".pub"
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse key content (remove email if present)
	keyContent := strings.TrimSpace(string(publicKeyData))
	parts := strings.Fields(keyContent)
	if len(parts) >= 2 {
		keyContent = parts[0] + " " + parts[1] // Only keep key type and key data
	}

	// Upload to GitHub
	key := &github.Key{
		Title: &title,
		Key:   &keyContent,
	}

	_, _, err = c.client.Users.CreateKey(c.ctx, key)
	if err != nil {
		// Check if key already exists
		if strings.Contains(err.Error(), "key is already in use") {
			fmt.Printf("🔑 SSH key already exists in GitHub account\n")
			return nil
		}
		return fmt.Errorf("failed to upload SSH key to GitHub: %w", err)
	}

	fmt.Printf("✅ SSH key automatically uploaded to GitHub!\n")
	return nil
}
