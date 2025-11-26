package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/gpg"
)

var gpgKeygenCmd = &cobra.Command{
	Use:   "gpg-keygen [account-alias]",
	Short: "🔐 Generate and manage GPG keys for commit signing",
	Long: `Generate GPG keys with proper parameters and manage them within gitshift.
This command handles:
- GPG key generation with custom parameters
- Proper key configuration for Git commit signing
- Integration with gitshift account system
- Public key export for adding to Git platforms

The generated keys can be used with any Git platform:
- GitHub (github.com and GitHub Enterprise)
- GitLab (gitlab.com and self-hosted)
- Bitbucket (coming soon)
- Any Git platform supporting GPG signatures`,
	Example: `  # Generate GPG key for GitHub account
  gitshift gpg-keygen work-github

  # Generate key with custom email
  gitshift gpg-keygen myaccount --email user@company.com

  # Generate ECC key instead of RSA
  gitshift gpg-keygen myaccount --type ECC

  # Generate key with 2-year expiration
  gitshift gpg-keygen myaccount --expire-date 2y

  # Generate key with passphrase protection
  gitshift gpg-keygen myaccount --passphrase "secure-passphrase"

  # Enable GPG signing automatically for this account
  gitshift gpg-keygen myaccount --enable`,
	Args: cobra.ExactArgs(1),
	RunE: runGPGKeygen,
}

var (
	gpgKeyType       string
	gpgKeyLength     int
	gpgKeyEmail      string
	gpgKeyPassphrase string
	gpgExpireDate    string
	gpgEnableSign    bool
	gpgForce         bool
)

func init() {
	rootCmd.AddCommand(gpgKeygenCmd)

	gpgKeygenCmd.Flags().StringVar(&gpgKeyType, "type", "RSA", "GPG key type (RSA, ECC, DSA)")
	gpgKeygenCmd.Flags().IntVar(&gpgKeyLength, "bits", 4096, "Key size in bits (RSA: 2048/4096, ignored for ECC)")
	gpgKeygenCmd.Flags().StringVar(&gpgKeyEmail, "email", "", "Email for GPG key (must match verified email on Git platform)")
	gpgKeygenCmd.Flags().StringVar(&gpgKeyPassphrase, "passphrase", "", "Passphrase for private key (empty for no passphrase)")
	gpgKeygenCmd.Flags().StringVar(&gpgExpireDate, "expire-date", "0", "Key expiration (0=never, 1y=1 year, 2y=2 years, etc.)")
	gpgKeygenCmd.Flags().BoolVar(&gpgEnableSign, "enable", false, "Automatically enable GPG commit signing for this account")
	gpgKeygenCmd.Flags().BoolVarP(&gpgForce, "force", "f", false, "Overwrite existing GPG key if present")
}

func runGPGKeygen(cmd *cobra.Command, args []string) error {
	accountAlias := args[0]

	fmt.Printf("🔐 Generating GPG key for account: %s\n", accountAlias)

	// Load configuration to check if account exists
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if account exists
	account, err := configManager.GetAccount(accountAlias)
	if err != nil || account == nil {
		return fmt.Errorf("account '%s' not found. Add it first with: gitshift add %s", accountAlias, accountAlias)
	}

	// Determine email and name to use
	email := gpgKeyEmail
	if email == "" {
		if account.Email != "" {
			email = account.Email
		} else {
			return fmt.Errorf("no email specified and account '%s' has no email. Use --email flag", accountAlias)
		}
	}

	name := account.Name
	if name == "" {
		return fmt.Errorf("account '%s' has no name configured. Update the account first", accountAlias)
	}

	fmt.Printf("📧 Using email: %s\n", email)
	fmt.Printf("👤 Using name: %s\n", name)

	// Check if account already has a GPG key
	if account.HasGPGKey() && !gpgForce {
		fmt.Printf("⚠️  Account '%s' already has GPG key: %s\n", accountAlias, account.GPGKeyID)
		fmt.Printf("💡 Use --force to generate a new key and replace the existing one\n")
		return fmt.Errorf("GPG key already exists for account '%s'", accountAlias)
	}

	// Validate and normalize key type
	gpgKeyType = normalizeKeyType(gpgKeyType)

	// Create GPG manager
	gpgManager := gpg.NewManager()

	// Generate the GPG key
	keyInfo, err := gpgManager.GenerateKey(gpg.GenerateKeyParams{
		Alias:      accountAlias,
		Name:       name,
		Email:      email,
		KeyType:    gpgKeyType,
		KeyLength:  gpgKeyLength,
		Passphrase: gpgKeyPassphrase,
		ExpireDate: gpgExpireDate,
		Force:      gpgForce,
	})
	if err != nil {
		return fmt.Errorf("failed to generate GPG key: %w", err)
	}

	fmt.Printf("✅ GPG key generated successfully\n")
	fmt.Printf("🔑 Key ID: %s\n", keyInfo.KeyID)
	fmt.Printf("🔍 Fingerprint: %s\n", keyInfo.Fingerprint)
	fmt.Printf("📅 Created: %s\n", keyInfo.CreatedAt.Format("2006-01-02 15:04:05"))

	if keyInfo.ExpiresAt != nil {
		fmt.Printf("⏰ Expires: %s\n", keyInfo.ExpiresAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("⏰ Expires: Never\n")
	}

	// Update account with GPG key information
	account.SetGPGKey(keyInfo.KeyID, keyInfo.KeyType, keyInfo.KeySize, keyInfo.Fingerprint, keyInfo.ExpiresAt)

	// Enable GPG signing if requested
	if gpgEnableSign {
		account.EnableGPGSigning()
		fmt.Printf("🔐 GPG commit signing enabled for account '%s'\n", accountAlias)
	}

	// Save the updated account configuration
	if err := configManager.RemoveAccount(accountAlias); err != nil {
		fmt.Printf("⚠️  Warning: Failed to remove old account config: %v\n", err)
		fmt.Printf("   GPG key generated successfully, but account config may not be updated\n")
		fmt.Printf("   You may need to manually update the account configuration\n")
	} else {
		if err := configManager.AddAccount(account); err != nil {
			fmt.Printf("⚠️  Warning: Failed to update account with GPG key: %v\n", err)
			fmt.Printf("   GPG key generated successfully, but not saved to account config\n")
		} else {
			fmt.Printf("💾 Updated account '%s' with GPG key information\n", accountAlias)
		}
	}

	// Export public key to file
	pubKeyFile, err := gpgManager.SavePublicKeyToFile(keyInfo.KeyID, accountAlias)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to save public key to file: %v\n", err)
	} else {
		fmt.Printf("📋 Public key saved to: %s\n", pubKeyFile)
	}

	// Show public key content
	fmt.Printf("\n📋 Public key content:\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────┐\n")
	pubKey, err := gpgManager.ExportPublicKey(keyInfo.KeyID)
	if err == nil {
		// Show first few lines for brevity
		lines := splitLines(pubKey, 5)
		for _, line := range lines {
			fmt.Printf("│ %s\n", line)
		}
		if len(lines) >= 5 {
			fmt.Printf("│ ... (truncated, see full key in file)\n")
		}
	}
	fmt.Printf("└─────────────────────────────────────────────────────────────────┘\n")

	// Copy public key to clipboard
	if err := copyGPGKeyToClipboard(keyInfo.KeyID); err != nil {
		fmt.Printf("⚠️  Warning: Could not copy to clipboard: %v\n", err)
	} else {
		fmt.Printf("📋 Public key copied to clipboard!\n")
	}

	// Test GPG key
	if err := gpgManager.TestGPGKey(keyInfo.KeyID); err != nil {
		fmt.Printf("⚠️  Warning: GPG key test failed: %v\n", err)
	}

	// Show instructions for adding to Git platforms
	fmt.Printf("\n💡 To add this GPG key to your Git platform:\n")
	fmt.Printf("   1. Copy the public key above (already in clipboard)\n")
	fmt.Printf("   2. Add it to your platform:\n")
	fmt.Printf("      • GitHub: https://github.com/settings/keys\n")
	fmt.Printf("      • GitLab: https://gitlab.com/-/profile/gpg_keys\n")
	fmt.Printf("      • GitHub Enterprise: https://your-domain/settings/keys\n")
	fmt.Printf("      • GitLab Self-hosted: https://your-domain/-/profile/gpg_keys\n")
	fmt.Printf("   3. Click 'New GPG key' and paste the content\n")
	fmt.Printf("   4. Switch to this account to enable signing:\n")
	fmt.Printf("      gitshift switch %s\n", accountAlias)

	if !gpgEnableSign {
		fmt.Printf("\n💡 To enable automatic GPG commit signing:\n")
		fmt.Printf("   gitshift gpg-keygen %s --enable\n", accountAlias)
		fmt.Printf("   OR edit your account and set gpg_enabled: true\n")
	}

	return nil
}

// normalizeKeyType normalizes the key type input
func normalizeKeyType(keyType string) string {
	switch keyType {
	case "rsa", "Rsa", "RSA":
		return "RSA"
	case "ecc", "Ecc", "ECC", "ed25519", "Ed25519", "EdDSA", "eddsa":
		return "ECC"
	case "dsa", "Dsa", "DSA":
		return "DSA"
	default:
		return "RSA" // Default to RSA
	}
}

// splitLines splits a string into lines, limiting the number of lines
func splitLines(s string, maxLines int) []string {
	lines := []string{}
	current := ""
	count := 0

	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
			count++
			if count >= maxLines {
				break
			}
		} else {
			current += string(c)
		}
	}

	if current != "" && count < maxLines {
		lines = append(lines, current)
	}

	return lines
}

// copyGPGKeyToClipboard copies the GPG public key to the clipboard (cross-platform)
func copyGPGKeyToClipboard(keyID string) error {
	// Export the public key
	cmd := exec.Command("gpg", "--armor", "--export", keyID)
	pubKey, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	var clipCmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS: use pbcopy
		clipCmd = exec.Command("pbcopy")

	case "linux":
		// Linux: try multiple clipboard tools in order of preference
		if _, err := exec.LookPath("xclip"); err == nil {
			clipCmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			clipCmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			// Wayland clipboard
			clipCmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found - install xclip (X11), xsel (X11), or wl-clipboard (Wayland)")
		}

	case "windows":
		// Windows: use clip command (built-in)
		clipCmd = exec.Command("clip")

	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	// Set up stdin pipe to write content to clipboard command
	stdin, err := clipCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := clipCmd.Start(); err != nil {
		return fmt.Errorf("failed to start clipboard command: %w", err)
	}

	// Ensure process cleanup on any error
	var writeErr, closeErr, waitErr error
	defer func() {
		// If we haven't waited yet and there was an error, kill the process
		if writeErr != nil || closeErr != nil {
			if clipCmd.Process != nil {
				_ = clipCmd.Process.Kill()
			}
		}
	}()

	if _, writeErr = stdin.Write(pubKey); writeErr != nil {
		return fmt.Errorf("failed to write to clipboard: %w", writeErr)
	}

	if closeErr = stdin.Close(); closeErr != nil {
		return fmt.Errorf("failed to close stdin: %w", closeErr)
	}

	if waitErr = clipCmd.Wait(); waitErr != nil {
		return fmt.Errorf("clipboard command failed: %w", waitErr)
	}

	return nil
}
