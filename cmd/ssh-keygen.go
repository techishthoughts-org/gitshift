package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
)

var sshKeygenCmd = &cobra.Command{
	Use:   "ssh-keygen [account-alias]",
	Short: "🔑 Generate and manage SSH keys for Git platform accounts",
	Long: `Generate SSH keys with proper parameters and manage them within gitshift.
This command handles:
- SSH key generation with custom parameters
- Proper file naming and storage in ~/.ssh
- SSH key permissions (600 for private, 644 for public)
- Known hosts management
- Integration with gitshift account system

The generated keys can be used with any Git platform:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)
- Bitbucket (coming soon)
- Custom Git platforms`,
	Example: `  # Generate key for GitHub account
  gitshift ssh-keygen work-github

  # Generate key for GitLab account
  gitshift ssh-keygen work-gitlab

  # Generate key with custom email
  gitshift ssh-keygen myaccount --email user@company.com

  # Generate RSA key instead of Ed25519
  gitshift ssh-keygen myaccount --type rsa --bits 4096

  # Generate key and add to GitHub automatically
  gitshift ssh-keygen myaccount --add-to-github`,
	Args: cobra.ExactArgs(1),
	RunE: runSSHKeygen,
}

var (
	keyType       string
	keyBits       int
	keyEmail      string
	keyPassphrase string
	addToGitHub   bool
	force         bool
)

func init() {
	rootCmd.AddCommand(sshKeygenCmd)

	sshKeygenCmd.Flags().StringVar(&keyType, "type", "ed25519", "SSH key type (ed25519, rsa, ecdsa)")
	sshKeygenCmd.Flags().IntVar(&keyBits, "bits", 0, "Key size in bits (RSA: 2048/4096, ECDSA: 256/384/521)")
	sshKeygenCmd.Flags().StringVar(&keyEmail, "email", "", "Email for SSH key comment")
	sshKeygenCmd.Flags().StringVar(&keyPassphrase, "passphrase", "", "Passphrase for private key (empty for no passphrase)")
	sshKeygenCmd.Flags().BoolVar(&addToGitHub, "add-to-github", false, "Automatically add the public key to GitHub")
	sshKeygenCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing SSH key if present")
}

func runSSHKeygen(cmd *cobra.Command, args []string) error {
	accountAlias := args[0]

	fmt.Printf("🔑 Generating SSH key for account: %s\n", accountAlias)

	// Load configuration to check if account exists
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if account exists
	account, err := configManager.GetAccount(accountAlias)
	var accountEmail string
	if err == nil && account != nil {
		accountEmail = account.Email
		fmt.Printf("📧 Found existing account with email: %s\n", accountEmail)
	}

	// Determine email to use
	if keyEmail == "" {
		if accountEmail != "" {
			keyEmail = accountEmail
		} else {
			return fmt.Errorf("no email specified and account '%s' not found. Use --email flag", accountAlias)
		}
	}

	// Validate key type and set defaults
	if err := validateKeyParameters(); err != nil {
		return err
	}

	// Generate the SSH key
	keyManager := &SSHKeyManager{}
	keyPath, err := keyManager.GenerateKey(GenerateKeyParams{
		Alias:      accountAlias,
		Email:      keyEmail,
		Type:       keyType,
		Bits:       keyBits,
		Passphrase: keyPassphrase,
		Force:      force,
	})
	if err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	fmt.Printf("✅ SSH key generated: %s\n", keyPath)
	fmt.Printf("📋 Public key: %s.pub\n", keyPath)

	// Update account if it exists
	if account != nil {
		account.SSHKeyPath = keyPath
		// Remove and re-add the account to update it
		if err := configManager.RemoveAccount(accountAlias); err == nil {
			if err := configManager.AddAccount(account); err != nil {
				fmt.Printf("⚠️  Warning: Failed to update account with new SSH key path: %v\n", err)
			} else {
				fmt.Printf("🔗 Updated account '%s' with new SSH key\n", accountAlias)
			}
		}
	}

	// Setup known hosts for Git platforms
	if err := keyManager.SetupKnownHosts(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to setup known hosts: %v\n", err)
	} else {
		fmt.Printf("🌐 Git platforms added to known hosts\n")
	}

	// Show public key content and copy to clipboard
	if err := showPublicKey(keyPath + ".pub"); err != nil {
		fmt.Printf("⚠️  Warning: Could not display public key: %v\n", err)
	}

	// Copy public key to clipboard using pbcopy
	if err := copyToClipboard(keyPath + ".pub"); err != nil {
		fmt.Printf("⚠️  Warning: Could not copy to clipboard: %v\n", err)
	} else {
		fmt.Printf("📋 Public key copied to clipboard!\n")
	}

	// Add to GitHub if requested
	if addToGitHub {
		if err := addKeyToGitHub(keyPath+".pub", accountAlias); err != nil {
			fmt.Printf("⚠️  Warning: Failed to add key to GitHub: %v\n", err)
			fmt.Printf("💡 Please add this key manually: https://github.com/settings/keys\n")
		} else {
			fmt.Printf("🚀 SSH key added to GitHub account!\n")
		}
	} else {
		fmt.Printf("\n💡 To add this key to your Git platform:\n")
		fmt.Printf("   1. Copy the public key above\n")
		fmt.Printf("   2. Add it to your platform:\n")
		fmt.Printf("      • GitHub: https://github.com/settings/keys\n")
		fmt.Printf("      • GitLab: https://gitlab.com/-/profile/keys\n")
		fmt.Printf("      • GitHub Enterprise: https://your-domain/settings/keys\n")
		fmt.Printf("      • GitLab Self-hosted: https://your-domain/-/profile/keys\n")
		fmt.Printf("   3. Click 'New SSH key' and paste the content\n")
		fmt.Printf("   OR for GitHub: gitshift ssh-keygen %s --add-to-github\n", accountAlias)
	}

	return nil
}

type GenerateKeyParams struct {
	Alias      string
	Email      string
	Type       string
	Bits       int
	Passphrase string
	Force      bool
}

type SSHKeyManager struct{}

func (m *SSHKeyManager) GenerateKey(params GenerateKeyParams) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate key file path
	keyPath := filepath.Join(sshDir, fmt.Sprintf("id_%s_%s", params.Type, params.Alias))

	// Check if key already exists
	if !params.Force {
		if _, err := os.Stat(keyPath); err == nil {
			return "", fmt.Errorf("SSH key already exists at %s (use --force to overwrite)", keyPath)
		}
	}

	fmt.Printf("🔧 Generating %s key with %d bits...\n", strings.ToUpper(params.Type), params.Bits)

	// Build ssh-keygen command
	args := []string{
		"-t", params.Type,
		"-C", params.Email,
		"-f", keyPath,
	}

	// Add key size if specified
	if params.Bits > 0 {
		args = append(args, "-b", fmt.Sprintf("%d", params.Bits))
	}

	// Add passphrase (empty means no passphrase)
	if params.Passphrase == "" {
		args = append(args, "-N", "")
	} else {
		args = append(args, "-N", params.Passphrase)
	}

	// Execute ssh-keygen
	cmd := exec.Command("ssh-keygen", args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	// Set proper permissions
	if err := os.Chmod(keyPath, 0600); err != nil {
		return "", fmt.Errorf("failed to set private key permissions: %w", err)
	}

	if err := os.Chmod(keyPath+".pub", 0644); err != nil {
		return "", fmt.Errorf("failed to set public key permissions: %w", err)
	}

	fmt.Printf("🔒 Set proper key permissions (600 for private, 644 for public)\n")

	// Automatically add the key to ssh-agent
	if err := m.addKeyToAgent(keyPath); err != nil {
		fmt.Printf("⚠️  Warning: Failed to add key to ssh-agent: %v\n", err)
	} else {
		fmt.Printf("🔑 Key automatically added to ssh-agent\n")
	}

	return keyPath, nil
}

// addKeyToAgent adds a key to the SSH agent
func (m *SSHKeyManager) addKeyToAgent(keyPath string) error {
	cmd := exec.Command("ssh-add", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func (m *SSHKeyManager) SetupKnownHosts() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	// GitHub's SSH host keys (current as of 2024)
	githubHosts := []string{
		"github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
		"github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=",
		"github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=",
	}

	// GitLab's SSH host keys (current as of 2024)
	gitlabHosts := []string{
		"gitlab.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAfuCHKVTjquxvt6CM6tdG4SLp1Btn/nOeHHE5UOzRdf",
		"gitlab.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCsj2bNKTBSpIYDEGk9KxsGh3mySTRgMtXL583qmBpzeQ+jqCMRgBqB98u3z++J1sKlXHWfM9dyhSevkMwSbhoR8XIq/U0tCNyokEi/ueaBMCvbcTHhO7FcwzY92WK4Yt0aGROY5qX2UKSeOvuP4D6TPqKF1onrSzH9bx9XUf2lEdWT/ia1NEKjunUqu1xOB/StKDHMoX4/OKyIzuS0q/T1zOATthvasJFoPrAjkohTyaDUz2LN5JoH839hViyEG82yB+MjcFV5MU3N1l1QL3cVUCh93xSaua1N85qivl+siMkPGbO5xR/En4iEY6K2XPASUEMaieWVNTRCtJ4S8H+9",
		"gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY=",
	}

	// Combine all platform hosts
	allHosts := append(githubHosts, gitlabHosts...)

	// Read existing known_hosts
	var existingContent string
	if content, err := os.ReadFile(knownHostsPath); err == nil {
		existingContent = string(content)
	}

	// Check what needs to be added
	var toAdd []string
	for _, host := range allHosts {
		if !strings.Contains(existingContent, host) {
			toAdd = append(toAdd, host)
		}
	}

	if len(toAdd) == 0 {
		return nil // All hosts already present
	}

	// Append missing hosts
	file, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v", err)
		}
	}()

	for _, host := range toAdd {
		if _, err := file.WriteString(host + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func validateKeyParameters() error {
	switch keyType {
	case "ed25519":
		if keyBits != 0 {
			fmt.Printf("ℹ️  Ed25519 keys have a fixed size, ignoring --bits parameter\n")
			keyBits = 0
		}
	case "rsa":
		if keyBits == 0 {
			keyBits = 4096 // Default to 4096 for RSA
		} else if keyBits < 2048 {
			return fmt.Errorf("RSA key size must be at least 2048 bits")
		}
	case "ecdsa":
		if keyBits == 0 {
			keyBits = 256 // Default to 256 for ECDSA
		} else if keyBits != 256 && keyBits != 384 && keyBits != 521 {
			return fmt.Errorf("ECDSA key size must be 256, 384, or 521 bits")
		}
	default:
		return fmt.Errorf("unsupported key type: %s (supported: ed25519, rsa, ecdsa)", keyType)
	}

	return nil
}

func showPublicKey(pubKeyPath string) error {
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return err
	}

	fmt.Printf("\n📋 Public key content:\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ %s │\n", strings.TrimSpace(string(content)))
	fmt.Printf("└─────────────────────────────────────────────────────────────────┘\n")

	// Show key fingerprint
	cmd := exec.Command("ssh-keygen", "-lf", pubKeyPath)
	if output, err := cmd.Output(); err == nil {
		fmt.Printf("🔍 Key fingerprint: %s", output)
	}

	return nil
}

func addKeyToGitHub(pubKeyPath, alias string) error {
	// Check if GitHub CLI is available
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) not found. Install it first: https://cli.github.com/")
	}

	// Check if authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not authenticated with GitHub CLI. Run: gh auth login")
	}

	// Add the key
	title := fmt.Sprintf("gitshift-%s-%s", alias, generateKeyID())
	cmd = exec.Command("gh", "ssh-key", "add", pubKeyPath, "--title", title)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add SSH key to GitHub: %w", err)
	}

	return nil
}

func generateKeyID() string {
	// Generate a short random ID for the key title
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	hash := sha256.Sum256(bytes)
	return fmt.Sprintf("%x", hash[:4])
}

// copyToClipboard copies the public key content to the clipboard (cross-platform)
func copyToClipboard(pubKeyPath string) error {
	content, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: use pbcopy
		cmd = exec.Command("pbcopy")

	case "linux":
		// Linux: try multiple clipboard tools in order of preference
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			// Wayland clipboard
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found - install xclip (X11), xsel (X11), or wl-clipboard (Wayland)\n" +
				"  Ubuntu/Debian: sudo apt install xclip\n" +
				"  Fedora/RHEL:   sudo dnf install xclip\n" +
				"  Arch:          sudo pacman -S xclip")
		}

	case "windows":
		// Windows: use clip command (built-in)
		cmd = exec.Command("clip")

	default:
		return fmt.Errorf("clipboard not supported on %s - please copy the key manually", runtime.GOOS)
	}

	// Set up stdin pipe to write content to clipboard command
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start clipboard command: %w", err)
	}

	// Ensure process cleanup on any error
	var writeErr, closeErr, waitErr error
	defer func() {
		// If we haven't waited yet and there was an error, kill the process
		if writeErr != nil || closeErr != nil {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}()

	if _, writeErr = stdin.Write(content); writeErr != nil {
		return fmt.Errorf("failed to write to clipboard: %w", writeErr)
	}

	if closeErr = stdin.Close(); closeErr != nil {
		return fmt.Errorf("failed to close stdin: %w", closeErr)
	}

	if waitErr = cmd.Wait(); waitErr != nil {
		return fmt.Errorf("clipboard command failed: %w", waitErr)
	}

	return nil
}
