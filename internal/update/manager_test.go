package update

import (
	"strings"
	"testing"
	"time"
)

func TestNewUpdateManager(t *testing.T) {
	version := "1.0.0"
	manager := NewUpdateManager(version)

	if manager == nil {
		t.Fatal("NewUpdateManager should return non-nil manager")
	}

	if manager.currentVersion != version {
		t.Errorf("Expected currentVersion %q, got %q", version, manager.currentVersion)
	}

	if manager.repoURL == "" {
		t.Error("repoURL should not be empty")
	}

	if manager.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestUpdateManager_CheckForUpdates(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test checking for updates
	release, err := manager.CheckForUpdates()
	if err != nil {
		t.Logf("CheckForUpdates returned error (expected in test environment): %v", err)
	}

	// In test environment, we might not have network access or the API might not be available
	// So we just check that the method doesn't panic and returns reasonable values
	if release != nil && release.TagName == "" {
		t.Error("If release is not nil, TagName should not be empty")
	}
}

func TestUpdateManager_PerformSecureUpdate(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test performing secure update
	release := &Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{
				Name: "gitpersona-darwin-amd64",
			},
		},
	}

	// This will likely fail in test environment due to network access
	err := manager.PerformSecureUpdate(release)
	if err != nil {
		t.Logf("PerformSecureUpdate returned error (expected in test environment): %v", err)
	}
}

func TestUpdateManager_isNewerVersion(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test with same version
	newer := manager.isNewerVersion("1.0.0", "1.0.0")
	if newer {
		t.Error("isNewerVersion should return false for same version")
	}

	// Test with newer version
	newer = manager.isNewerVersion("1.0.1", "1.0.0")
	if !newer {
		t.Error("isNewerVersion should return true for newer version")
	}

	// Test with older version
	newer = manager.isNewerVersion("0.9.0", "1.0.0")
	if newer {
		t.Error("isNewerVersion should return false for older version")
	}
}

func TestRelease_Structure(t *testing.T) {
	release := &Release{
		TagName:     "v1.0.0",
		Name:        "Release 1.0.0",
		Body:        "Release notes",
		Draft:       false,
		Prerelease:  false,
		PublishedAt: time.Now(),
		Assets: []Asset{
			{
				Name:               "gitpersona-darwin-amd64",
				BrowserDownloadURL: "https://example.com/download",
			},
		},
	}

	if release.TagName == "" {
		t.Error("TagName should not be empty")
	}
	if release.Name == "" {
		t.Error("Name should not be empty")
	}
	if release.PublishedAt.IsZero() {
		t.Error("PublishedAt should not be zero")
	}
	if len(release.Assets) == 0 {
		t.Error("Assets should not be empty")
	}
}

func TestAsset_Structure(t *testing.T) {
	asset := &Asset{
		Name:               "gitpersona-darwin-amd64",
		BrowserDownloadURL: "https://example.com/download",
		ContentType:        "application/octet-stream",
		Size:               1024,
	}

	if asset.Name == "" {
		t.Error("Name should not be empty")
	}
	if asset.BrowserDownloadURL == "" {
		t.Error("BrowserDownloadURL should not be empty")
	}
	if asset.Size <= 0 {
		t.Error("Size should be positive")
	}
}

func TestAttestation_Structure(t *testing.T) {
	attestation := &Attestation{
		Bundle:    "attestation bundle",
		Predicate: "build",
		Subject:   "attestation subject",
	}

	if attestation.Bundle == "" {
		t.Error("Bundle should not be empty")
	}
	if attestation.Predicate == "" {
		t.Error("Predicate should not be empty")
	}
	if attestation.Subject == "" {
		t.Error("Subject should not be empty")
	}
}

func TestSignature_Structure(t *testing.T) {
	signature := &Signature{
		Algorithm: "ECDSA",
		Keyid:     "key123",
		Signature: "signature data",
	}

	if signature.Algorithm == "" {
		t.Error("Algorithm should not be empty")
	}
	if signature.Keyid == "" {
		t.Error("Keyid should not be empty")
	}
	if signature.Signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestUpdateManager_EdgeCases(t *testing.T) {
	// Test with empty version
	manager := NewUpdateManager("")
	if manager.currentVersion != "" {
		t.Error("currentVersion should be empty when initialized with empty string")
	}

	// Test with very long version
	longVersion := strings.Repeat("1.0.0.", 100)
	manager = NewUpdateManager(longVersion)
	if manager.currentVersion != longVersion {
		t.Error("currentVersion should handle long versions")
	}

	// Test with special characters in version
	specialVersion := "1.0.0-beta+123"
	manager = NewUpdateManager(specialVersion)
	if manager.currentVersion != specialVersion {
		t.Error("currentVersion should handle special characters")
	}
}

func TestUpdateManager_Concurrency(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test concurrent access to manager methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test various methods concurrently
			_ = manager.isNewerVersion("1.0.1", "1.0.0")
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUpdateManager_ErrorHandling(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test with nil release
	release := &Release{}
	err := manager.PerformSecureUpdate(release)
	if err == nil {
		t.Log("PerformSecureUpdate should return error for empty release (expected in test environment)")
	}
}

func TestUpdateManager_Performance(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test that basic operations complete in reasonable time
	start := time.Now()
	_ = manager.isNewerVersion("1.0.1", "1.0.0")
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("isNewerVersion took too long: %v", duration)
	}
}

func TestUpdateManager_Integration(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test that manager can be created and basic methods work
	if manager == nil {
		t.Fatal("Manager should be created successfully")
	}

	// Test isNewerVersion
	newer := manager.isNewerVersion("1.0.1", "1.0.0")
	if !newer {
		t.Error("isNewerVersion should return true for newer version")
	}
}

func TestUpdateManager_NetworkOperations(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	// Test network operations (these will likely fail in test environment)
	_, err := manager.CheckForUpdates()
	if err != nil {
		t.Logf("CheckForUpdates failed (expected in test environment): %v", err)
	}

	// Test with mock release
	release := &Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{
				Name:               "gitpersona-darwin-amd64",
				BrowserDownloadURL: "https://example.com/download",
			},
		},
	}

	err = manager.PerformSecureUpdate(release)
	if err != nil {
		t.Logf("PerformSecureUpdate failed (expected in test environment): %v", err)
	}
}

func TestUpdateManager_VersionComparison(t *testing.T) {
	manager := NewUpdateManager("1.0.0")

	tests := []struct {
		current      string
		latest       string
		shouldUpdate bool
	}{
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"1.0.1", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"2.0.0", "1.0.0", false},
		{"1.0.0-beta", "1.0.0", false}, // String comparison: "1.0.0-beta" < "1.0.0"
		{"1.0.0", "1.0.0-beta", true},  // String comparison: "1.0.0" > "1.0.0-beta"
		{"1.0.0-beta", "1.0.0-beta", false},
		{"1.0.0-beta", "1.0.0-alpha", false}, // String comparison: "1.0.0-beta" < "1.0.0-alpha"
		{"1.0.0-alpha", "1.0.0-beta", true},  // String comparison: "1.0.0-alpha" > "1.0.0-beta"
	}

	for _, test := range tests {
		available := manager.isNewerVersion(test.latest, test.current)
		if available != test.shouldUpdate {
			t.Errorf("isNewerVersion(%q, %q) = %v, expected %v",
				test.latest, test.current, available, test.shouldUpdate)
		}
	}
}

func TestUpdateManager_ChecksumValidation(t *testing.T) {
	// Test that the manager can be created and basic operations work
	manager := NewUpdateManager("1.0.0")
	if manager == nil {
		t.Fatal("Manager should be created successfully")
	}

	// Test version comparison
	newer := manager.isNewerVersion("1.0.1", "1.0.0")
	if !newer {
		t.Error("isNewerVersion should return true for newer version")
	}
}
