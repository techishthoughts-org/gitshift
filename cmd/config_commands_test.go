package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/techishthoughts/gitshift/internal/config"
	"github.com/techishthoughts/gitshift/internal/discovery"
	"github.com/techishthoughts/gitshift/internal/models"
)

func createCommandTestConfig(t *testing.T) (string, *models.Account) {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "custom", "gitshift.yaml")
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	account := models.NewAccount("work", "Original User", "work@example.com", "/tmp/work")
	account.APIEndpoint = "https://git.example.com/api"
	account.TokenPath = "/secure/token"
	account.Status = models.AccountStatusIsolated
	account.IsolationLevel = models.IsolationLevelStrict
	account.ValidationErrors = []string{"preserve me"}
	account.AccountMetadata = map[string]string{"owner": "platform"}
	if err := manager.AddAccount(account); err != nil {
		t.Fatal(err)
	}
	return configFile, account
}

func useCommandTestConfig(t *testing.T, configFile string) {
	t.Helper()
	previous := cfgFile
	cfgFile = configFile
	t.Cleanup(func() { cfgFile = previous })
}

func TestCurrentUsesLoadedCustomConfig(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	if err := runCurrentCommand(currentCmd, nil); err != nil {
		t.Fatalf("runCurrentCommand() error = %v", err)
	}
}

func TestUpdatePreservesUneditedFieldsAndDefault(t *testing.T) {
	configFile, original := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	if err := updateCmd.Flags().Set("name", "Updated User"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = updateCmd.Flags().Set("name", "") })

	if err := updateCmd.RunE(updateCmd, []string{"work"}); err != nil {
		t.Fatalf("update command error = %v", err)
	}

	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	updated, err := manager.GetCurrentAccount()
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated User" || updated.Alias != "work" || !updated.IsDefault {
		t.Fatalf("updated current account = %#v", updated)
	}
	if updated.APIEndpoint != original.APIEndpoint || updated.TokenPath != original.TokenPath ||
		updated.Status != original.Status || updated.IsolationLevel != original.IsolationLevel ||
		updated.AccountMetadata["owner"] != "platform" || len(updated.ValidationErrors) != 1 {
		t.Fatalf("update dropped fields: %#v", updated)
	}
}

func TestDiscoverOverwritePreservesAccountsOnScanFailureOrNoResults(t *testing.T) {
	tests := []struct {
		name    string
		scan    func() ([]*discovery.DiscoveredAccount, error)
		wantErr bool
	}{
		{
			name: "scan failure",
			scan: func() ([]*discovery.DiscoveredAccount, error) {
				return nil, errors.New("scan unavailable")
			},
			wantErr: true,
		},
		{
			name: "no results",
			scan: func() ([]*discovery.DiscoveredAccount, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile, _ := createCommandTestConfig(t)
			useCommandTestConfig(t, configFile)
			if err := discoverCmd.Flags().Set("overwrite", "true"); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = discoverCmd.Flags().Set("overwrite", "false") })

			err := runDiscoverCommandWithScanner(discoverCmd, nil, tt.scan)
			if (err != nil) != tt.wantErr {
				t.Fatalf("runDiscoverCommandWithScanner() error = %v, wantErr %t", err, tt.wantErr)
			}

			manager := config.NewManagerWithPath(configFile)
			if err := manager.Load(); err != nil {
				t.Fatal(err)
			}
			current, err := manager.GetCurrentAccount()
			if err != nil {
				t.Fatalf("existing current account was removed: %v", err)
			}
			if current.Alias != "work" || len(manager.ListAccounts()) != 1 {
				t.Fatalf("existing accounts changed: %#v", manager.ListAccounts())
			}
		})
	}
}
