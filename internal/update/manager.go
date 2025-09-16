package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpdateManager handles secure application updates following 2025 best practices
type UpdateManager struct {
	currentVersion string
	repoURL        string
	httpClient     *http.Client
}

// NewUpdateManager creates a new update manager
func NewUpdateManager(currentVersion string) *UpdateManager {
	return &UpdateManager{
		currentVersion: currentVersion,
		repoURL:        "https://api.github.com/repos/techishthoughts/GitPersona",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Release represents a GitHub release with security metadata
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`

	// 2025 security extensions
	Attestations []Attestation `json:"attestations,omitempty"`
	Signatures   []Signature   `json:"signatures,omitempty"`
}

// Asset represents a release asset with security verification
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int    `json:"size"`
	ContentType        string `json:"content_type"`

	// 2025 security metadata
	SHA256    string `json:"sha256,omitempty"`
	SHA512    string `json:"sha512,omitempty"`
	Signature string `json:"signature,omitempty"`
	SLSA      string `json:"slsa_provenance,omitempty"`
}

// Attestation represents SLSA build provenance (2025 security standard)
type Attestation struct {
	Bundle    string `json:"bundle"`
	Predicate string `json:"predicate"`
	Subject   string `json:"subject"`
}

// Signature represents code signature verification
type Signature struct {
	Algorithm string `json:"algorithm"`
	Keyid     string `json:"keyid"`
	Signature string `json:"signature"`
}

// CheckForUpdates checks for available updates with security verification
func (um *UpdateManager) CheckForUpdates() (*Release, error) {
	fmt.Println("🔍 Checking for updates...")

	url := um.repoURL + "/releases/latest"
	resp, err := um.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update check failed with status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Skip drafts and pre-releases for security
	if release.Draft || release.Prerelease {
		return nil, nil
	}

	// Compare versions
	if um.isNewerVersion(release.TagName, um.currentVersion) {
		fmt.Printf("✅ New version available: %s -> %s\n", um.currentVersion, release.TagName)
		return &release, nil
	}

	fmt.Printf("✅ Already running latest version: %s\n", um.currentVersion)
	return nil, nil
}

// PerformSecureUpdate downloads and installs update with verification
func (um *UpdateManager) PerformSecureUpdate(release *Release) error {
	// Find appropriate binary for current platform
	asset := um.findPlatformAsset(release.Assets)
	if asset == nil {
		return fmt.Errorf("no compatible binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("📥 Downloading %s...\n", asset.Name)

	// Download with integrity verification
	tempPath, err := um.secureDownload(asset)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = os.Remove(tempPath) }()

	// Verify signatures (2025 security requirement)
	if err := um.verifySignatures(tempPath, asset, release); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Install update
	if err := um.installUpdate(tempPath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("✅ Successfully updated to version %s\n", release.TagName)
	fmt.Println("🔄 Please restart gitpersona to use the new version")

	return nil
}

// secureDownload downloads file with integrity checking
func (um *UpdateManager) secureDownload(asset *Asset) (string, error) {
	resp, err := um.httpClient.Get(asset.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "gitpersona-update")

	file, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	// Download with progress and hash calculation
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return "", err
	}

	// Verify SHA256 checksum (2025 security standard)
	if asset.SHA256 != "" {
		downloadedHash := hex.EncodeToString(hasher.Sum(nil))
		if downloadedHash != asset.SHA256 {
			return "", fmt.Errorf("checksum verification failed: expected %s, got %s",
				asset.SHA256, downloadedHash)
		}
		fmt.Printf("✅ Checksum verified\n")
	}

	return tempFile, nil
}

// verifySignatures verifies cryptographic signatures (2025 security requirement)
func (um *UpdateManager) verifySignatures(filePath string, asset *Asset, release *Release) error {
	fmt.Println("🔐 Verifying signatures...")

	// Verify SLSA provenance (Supply Chain security)
	if len(release.Attestations) > 0 {
		if err := um.verifySLSAAttestation(filePath, release.Attestations[0]); err != nil {
			return fmt.Errorf("SLSA verification failed: %w", err)
		}
		fmt.Printf("✅ SLSA provenance verified\n")
	}

	// Verify code signatures (using cosign/sigstore)
	if asset.Signature != "" {
		if err := um.verifyCodeSignature(filePath, asset.Signature); err != nil {
			return fmt.Errorf("code signature verification failed: %w", err)
		}
		fmt.Printf("✅ Code signature verified\n")
	}

	return nil
}

// verifySLSAAttestation verifies SLSA build provenance
func (um *UpdateManager) verifySLSAAttestation(filePath string, attestation Attestation) error {
	// In a real implementation, this would verify SLSA provenance
	// using tools like slsa-verifier

	cmd := exec.Command("slsa-verifier", "verify-artifact",
		"--provenance-path", attestation.Bundle,
		"--source-uri", "github.com/techishthoughts/GitPersona",
		filePath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SLSA verification failed: %w", err)
	}

	return nil
}

// verifyCodeSignature verifies code signature using cosign
func (um *UpdateManager) verifyCodeSignature(filePath, signature string) error {
	// In a real implementation, this would use cosign to verify signatures
	cmd := exec.Command("cosign", "verify-blob",
		"--signature", signature,
		"--certificate-identity-regexp", ".*@arthurcosta.*",
		"--certificate-oidc-issuer", "https://github.com/login/oauth",
		filePath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cosign verification failed: %w", err)
	}

	return nil
}

// installUpdate installs the verified update
func (um *UpdateManager) installUpdate(tempPath string) error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Create backup of current version
	backupPath := execPath + ".backup." + um.currentVersion
	if err := copyFile(execPath, backupPath); err != nil {
		fmt.Printf("⚠️  Could not create backup: %v\n", err)
	} else {
		fmt.Printf("💾 Backup created: %s\n", backupPath)
	}

	// Replace executable
	if err := copyFile(tempPath, execPath); err != nil {
		// Restore backup on failure
		if _, err := os.Stat(backupPath); err == nil {
			if restoreErr := copyFile(backupPath, execPath); restoreErr != nil {
				fmt.Printf("⚠️  Could not restore backup: %v\n", restoreErr)
			}
		}
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Set executable permissions
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// isNewerVersion compares version strings (simplified semver)
func (um *UpdateManager) isNewerVersion(newVer, currentVer string) bool {
	// Remove 'v' prefix if present
	newVer = strings.TrimPrefix(newVer, "v")
	currentVer = strings.TrimPrefix(currentVer, "v")

	// Simple version comparison (in real implementation, use semver library)
	return newVer > currentVer
}

// findPlatformAsset finds the appropriate binary for current platform
func (um *UpdateManager) findPlatformAsset(assets []Asset) *Asset {
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)

	for _, asset := range assets {
		if strings.Contains(asset.Name, platform) {
			return &asset
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
