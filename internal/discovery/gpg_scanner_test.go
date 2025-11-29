package discovery

import (
	"testing"
)

func TestGPGScanner_ScanGPGKeys(t *testing.T) {
	scanner := NewGPGScanner()

	// Test scanning GPG keys
	discovered, err := scanner.ScanGPGKeys()
	if err != nil {
		t.Fatalf("ScanGPGKeys() error = %v", err)
	}

	t.Logf("Found %d GPG key(s)", len(discovered))

	for _, account := range discovered {
		t.Logf("  - Alias: %s, Name: %s, Email: %s, KeyID: %s",
			account.Alias,
			account.Name,
			account.Email,
			account.GPGKeyID)
	}
}

func TestGPGScanner_ParseUID(t *testing.T) {
	scanner := NewGPGScanner()

	tests := []struct {
		uid           string
		expectedName  string
		expectedEmail string
	}{
		{
			uid:           "Jane Developer (Work GPG) <jane.dev@company.com>",
			expectedName:  "Jane Developer",
			expectedEmail: "jane.dev@company.com",
		},
		{
			uid:           "John Doe <john@example.com>",
			expectedName:  "John Doe",
			expectedEmail: "john@example.com",
		},
		{
			uid:           "Simple Name <simple@test.com>",
			expectedName:  "Simple Name",
			expectedEmail: "simple@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.uid, func(t *testing.T) {
			name, email := scanner.parseUID(tt.uid)
			if name != tt.expectedName {
				t.Errorf("parseUID() name = %v, want %v", name, tt.expectedName)
			}
			if email != tt.expectedEmail {
				t.Errorf("parseUID() email = %v, want %v", email, tt.expectedEmail)
			}
		})
	}
}
