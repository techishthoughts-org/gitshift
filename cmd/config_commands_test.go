package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
	// Model an account that a previous switch applied.
	if err := manager.SetCurrentAccount(account.Alias); err != nil {
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

func setDiscoverFlag(t *testing.T, name, value string) {
	t.Helper()
	flag := discoverCmd.Flags().Lookup(name)
	previousChanged := flag.Changed
	if name == "alias" {
		previous, err := discoverCmd.Flags().GetStringArray(name)
		if err != nil {
			t.Fatal(err)
		}
		replacer, ok := flag.Value.(interface{ Replace([]string) error })
		if !ok {
			t.Fatal("alias flag does not support replacement")
		}
		if err := replacer.Replace([]string{value}); err != nil {
			t.Fatal(err)
		}
		flag.Changed = true
		t.Cleanup(func() {
			_ = replacer.Replace(previous)
			flag.Changed = previousChanged
		})
		return
	}
	previousValue := flag.Value.String()
	if err := discoverCmd.Flags().Set(name, value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = discoverCmd.Flags().Set(name, previousValue)
		flag.Changed = previousChanged
	})
}

func TestCurrentUsesLoadedCustomConfig(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	if err := currentCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = currentCmd.Flags().Set("json", "false") })
	var output bytes.Buffer
	currentCmd.SetOut(&output)
	t.Cleanup(func() { currentCmd.SetOut(nil) })
	if err := runCurrentCommand(currentCmd, nil); err != nil {
		t.Fatalf("runCurrentCommand() error = %v", err)
	}
	if bytes.Contains(output.Bytes(), []byte("token_path")) || bytes.Contains(output.Bytes(), []byte("isolation_metadata")) {
		t.Fatalf("current JSON exposed internal fields: %s", output.String())
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

func TestUpdateCanonicalUsernameAndPlatformDefaults(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	for name, value := range map[string]string{"username": "gitlab-user", "platform": "gitlab"} {
		if err := updateCmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
		flagName := name
		t.Cleanup(func() {
			_ = updateCmd.Flags().Set(flagName, "")
			updateCmd.Flags().Lookup(flagName).Changed = false
		})
	}

	if err := updateCmd.RunE(updateCmd, []string{"work"}); err != nil {
		t.Fatalf("update command error = %v", err)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	account, err := manager.GetAccount("work")
	if err != nil {
		t.Fatal(err)
	}
	if account.GetUsername() != "gitlab-user" || account.GetPlatform() != "gitlab" || account.GetDomain() != "gitlab.com" || account.APIEndpoint != "" {
		t.Fatalf("provider update left mismatched state: %#v", account)
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

func TestDiscoverDryRunScansWithExistingAccountsAndDoesNotWrite(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	before, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := discoverCmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = discoverCmd.Flags().Set("dry-run", "false")
		discoverCmd.Flags().Lookup("dry-run").Changed = false
	})

	scanned := false
	err = runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		scanned = true
		return []*discovery.DiscoveredAccount{{
			Account:    &models.Account{Alias: "personal", Username: "person", GitHubUsername: "person", Platform: "github", Domain: "github.com"},
			Source:     "github-cli",
			Confidence: 10,
		}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !scanned {
		t.Fatal("dry run stopped before scanning existing configuration")
	}
	after, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dry run changed configuration")
	}
}

func TestDiscoverAliasValidationFailureDoesNotWrite(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	before, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	setDiscoverFlag(t, "alias", "unknown=personal")

	err = runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{aliasTestResult("suggested", "github|github.com|person")}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "unknown discovery selector") {
		t.Fatalf("error = %v", err)
	}
	after, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("alias validation failure changed configuration")
	}
}

func TestDiscoverAliasAddsToExistingConfiguration(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	setDiscoverFlag(t, "alias", "github|github.com|person=personal")

	err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{aliasTestResult("person", "github|github.com|person")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAccounts()) != 2 {
		t.Fatalf("accounts = %#v", manager.ListAccounts())
	}
	if account, err := manager.GetAccount("personal"); err != nil || account.GetUsername() != "person" {
		t.Fatalf("personal account = %#v, %v", account, err)
	}
}

func TestDiscoverPromptAliasCanCreatePendingAccount(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	setDiscoverFlag(t, "prompt-aliases", "true")
	discoverCmd.SetIn(strings.NewReader("personal\n"))
	discoverCmd.SetOut(&bytes.Buffer{})
	t.Cleanup(func() {
		discoverCmd.SetIn(nil)
		discoverCmd.SetOut(nil)
	})

	result := aliasTestResult("person", "github|github.com|person")
	result.Name = ""
	result.Email = ""
	if err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{result}, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if pending, err := manager.GetPendingAccount("personal"); err != nil || pending.GitHubUsername != "person" {
		t.Fatalf("pending account = %#v, %v", pending, err)
	}
}

func TestDiscoverFreshInteractiveSetupPromptsAndAppliesSelection(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "gitshift.yaml")
	useCommandTestConfig(t, configFile)

	previousTerminalCheck := terminalCheck
	previousApply := applyDiscoveredAccount
	terminalCheck = func() bool { return true }
	applied := ""
	applyDiscoveredAccount = func(_ *cobra.Command, manager *config.Manager, alias string, _, _, _ bool) error {
		applied = alias
		return manager.SetCurrentAccount(alias)
	}
	t.Cleanup(func() {
		terminalCheck = previousTerminalCheck
		applyDiscoveredAccount = previousApply
	})

	discoverCmd.SetIn(strings.NewReader("personal\nwork\n2\n"))
	discoverCmd.SetOut(&bytes.Buffer{})
	t.Cleanup(func() {
		discoverCmd.SetIn(nil)
		discoverCmd.SetOut(nil)
	})

	results := []*discovery.DiscoveredAccount{
		aliasTestResult("alice", "github|github.com|alice"),
		aliasTestResult("bob", "github|github.com|bob"),
	}
	if err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return results, nil
	}); err != nil {
		t.Fatal(err)
	}
	if applied != "work" {
		t.Fatalf("applied account = %q, want work", applied)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAccounts()) != 2 {
		t.Fatalf("accounts = %#v", manager.ListAccounts())
	}
	current, err := manager.GetCurrentAccount()
	if err != nil || current.Alias != "work" {
		t.Fatalf("current account = %#v, %v", current, err)
	}
}

func TestDiscoverAliasOverwriteReplacesAccountsAndCleansRules(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.AddAutoSwitchRule(t.TempDir(), "work"); err != nil {
		t.Fatal(err)
	}
	setDiscoverFlag(t, "overwrite", "true")
	setDiscoverFlag(t, "alias", "github|github.com|person=personal")

	if err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{aliasTestResult("person", "github|github.com|person")}, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager = config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if len(manager.ListAccounts()) != 1 || len(manager.GetConfig().AutoSwitch.Rules) != 0 {
		t.Fatalf("replacement state = %#v", manager.GetConfig())
	}
	if _, err := manager.GetAccount("personal"); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAliasDryRunPreviewsWithoutWriting(t *testing.T) {
	configFile, _ := createCommandTestConfig(t)
	useCommandTestConfig(t, configFile)
	before, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	setDiscoverFlag(t, "dry-run", "true")
	setDiscoverFlag(t, "alias", "github|github.com|person=personal")
	result := aliasTestResult("person", "github|github.com|person")
	if err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{result}, nil
	}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dry-run alias preview changed configuration")
	}
	if result.Alias != "personal" {
		t.Fatalf("preview alias = %q", result.Alias)
	}
}

func TestDiscoverAliasPlanningFailureDoesNotCreateConfig(t *testing.T) {
	tests := []struct {
		name  string
		flags func(*testing.T)
	}{
		{
			name: "unknown selector",
			flags: func(t *testing.T) {
				setDiscoverFlag(t, "alias", "unknown=personal")
			},
		},
		{
			name: "prompt EOF",
			flags: func(t *testing.T) {
				setDiscoverFlag(t, "prompt-aliases", "true")
				discoverCmd.SetIn(strings.NewReader(""))
				t.Cleanup(func() { discoverCmd.SetIn(nil) })
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "missing", "config.yaml")
			useCommandTestConfig(t, configFile)
			test.flags(t)
			err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
				return []*discovery.DiscoveredAccount{aliasTestResult("person", "github|github.com|person")}, nil
			})
			if err == nil {
				t.Fatal("expected alias planning error")
			}
			if _, statErr := os.Stat(configFile); !os.IsNotExist(statErr) {
				t.Fatalf("configuration was created: %v", statErr)
			}
			if _, statErr := os.Stat(filepath.Dir(configFile)); !os.IsNotExist(statErr) {
				t.Fatalf("configuration directory was created: %v", statErr)
			}
		})
	}
}

func TestDiscoverSuccessfulAliasPlanInitializesMissingConfig(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "missing", "config.yaml")
	useCommandTestConfig(t, configFile)
	setDiscoverFlag(t, "alias", "github|github.com|person=personal")

	if err := runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{aliasTestResult("person", "github|github.com|person")}, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager := config.NewManagerWithPath(configFile)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	if account, err := manager.GetAccount("personal"); err != nil || account.GetUsername() != "person" {
		t.Fatalf("personal account = %#v, %v", account, err)
	}
}

func TestDiscoverAliasPlanningFailureDoesNotMigrateOrChmodConfig(t *testing.T) {
	tests := []struct {
		name  string
		flags func(*testing.T)
	}{
		{
			name: "unknown selector",
			flags: func(t *testing.T) {
				setDiscoverFlag(t, "alias", "unknown=personal")
			},
		},
		{
			name: "prompt EOF",
			flags: func(t *testing.T) {
				setDiscoverFlag(t, "prompt-aliases", "true")
				discoverCmd.SetIn(strings.NewReader(""))
				t.Cleanup(func() { discoverCmd.SetIn(nil) })
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFile, _ := createCommandTestConfig(t)
			useCommandTestConfig(t, configFile)
			current, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatal(err)
			}
			legacy := bytes.Replace(current, []byte("config_version: 1.1.0"), []byte("config_version: 1.0.0"), 1)
			if bytes.Equal(legacy, current) {
				t.Fatal("test fixture did not contain the current config version")
			}
			if err := os.WriteFile(configFile, legacy, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(configFile, 0644); err != nil {
				t.Fatal(err)
			}
			test.flags(t)

			err = runDiscoverCommandWithScanner(discoverCmd, nil, func() ([]*discovery.DiscoveredAccount, error) {
				return []*discovery.DiscoveredAccount{aliasTestResult("person", "github|github.com|person")}, nil
			})
			if err == nil {
				t.Fatal("expected alias planning error")
			}
			after, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(legacy, after) {
				t.Fatal("planning failure migrated or rewrote configuration")
			}
			info, err := os.Stat(configFile)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0644 {
				t.Fatalf("config mode = %o, want unchanged 644", got)
			}
		})
	}
}

func setUpdateFlags(t *testing.T, values map[string]string) {
	t.Helper()
	for name, value := range values {
		flag := updateCmd.Flags().Lookup(name)
		previous, previousChanged := flag.Value.String(), flag.Changed
		if err := updateCmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = flag.Value.Set(previous)
			flag.Changed = previousChanged
		})
	}
}

func TestUpdateValidatesSSHKeyAndGPGOptions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	key := filepath.Join(home, ".ssh", "id_work")
	if err := os.MkdirAll(filepath.Dir(key), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}
	previousScan := scanGPGKeys
	t.Cleanup(func() { scanGPGKeys = previousScan })
	scanGPGKeys = func() ([]*discovery.DiscoveredAccount, error) {
		return []*discovery.DiscoveredAccount{{}, {Account: &models.Account{Email: "WORK@example.com", GPGKeyID: "ABCDEF12"}}}, nil
	}

	tests := []struct {
		name    string
		flags   map[string]string
		gpgKey  string
		email   string
		wantErr string
		check   func(*testing.T, *models.Account)
	}{
		{name: "missing SSH key", flags: map[string]string{"ssh-key": filepath.Join(home, "missing")}, wantErr: "SSH key not found"},
		{
			name:  "SSH key path is normalized",
			flags: map[string]string{"ssh-key": "~/.ssh/id_work"},
			check: func(t *testing.T, a *models.Account) {
				if a.SSHKeyPath != key {
					t.Fatalf("SSH key = %q, want %q", a.SSHKeyPath, key)
				}
			},
		},
		{name: "auto-gpg without a matching key", flags: map[string]string{"auto-gpg": "true"}, email: "nobody@example.com", wantErr: "no GPG secret key"},
		{
			name:  "auto-gpg attaches the matching key",
			flags: map[string]string{"auto-gpg": "true"},
			email: "work@example.com",
			check: func(t *testing.T, a *models.Account) {
				if a.GPGKeyID != "ABCDEF12" || !a.GPGEnabled {
					t.Fatalf("GPG key=%q enabled=%t", a.GPGKeyID, a.GPGEnabled)
				}
			},
		},
		{name: "gpg signing needs a key", flags: map[string]string{"signing": "gpg"}, wantErr: "needs a GPG key"},
		{name: "unknown signing format", flags: map[string]string{"signing": "x509"}, wantErr: "use ssh, gpg or off"},
		{name: "missing signing key", flags: map[string]string{"signing": "ssh", "signing-key": filepath.Join(home, "nope")}, wantErr: "signing key not found"},
		{
			name:   "signing can be turned off",
			flags:  map[string]string{"signing": "off"},
			gpgKey: "ABCDEF12",
			check: func(t *testing.T, a *models.Account) {
				if format, _ := a.CommitSigning(); a.GPGKeyID != "ABCDEF12" || format != "" {
					t.Fatalf("GPG key=%q signing=%q, want key kept and signing off", a.GPGKeyID, format)
				}
			},
		},
		{
			name:   "ssh signing uses the account SSH key and replaces gpg",
			flags:  map[string]string{"signing": "ssh"},
			gpgKey: "ABCDEF12",
			check: func(t *testing.T, a *models.Account) {
				if format, key := a.CommitSigning(); format != "ssh" || key != a.SSHKeyPath || a.GPGEnabled {
					t.Fatalf("signing=%q key=%q gpgEnabled=%t", format, key, a.GPGEnabled)
				}
			},
		},
		{
			name:  "ssh signing with a separate normalized key",
			flags: map[string]string{"signing": "ssh", "signing-key": "~/.ssh/id_work"},
			check: func(t *testing.T, a *models.Account) {
				if format, signingKey := a.CommitSigning(); format != "ssh" || signingKey != key {
					t.Fatalf("signing=%q key=%q, want ssh %q", format, signingKey, key)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile, _ := createCommandTestConfig(t)
			useCommandTestConfig(t, configFile)
			manager := config.NewManagerWithPath(configFile)
			if err := manager.Load(); err != nil {
				t.Fatal(err)
			}
			account, _ := manager.GetAccount("work")
			edited := *account
			if tt.email != "" {
				edited.Email = tt.email
			}
			if tt.gpgKey != "" {
				edited.GPGKeyID, edited.GPGEnabled = tt.gpgKey, true
			}
			if err := manager.UpdateAccount(&edited); err != nil {
				t.Fatal(err)
			}
			setUpdateFlags(t, tt.flags)

			err := updateCmd.RunE(updateCmd, []string{"work"})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			reloaded := config.NewManagerWithPath(configFile)
			if err := reloaded.Load(); err != nil {
				t.Fatal(err)
			}
			got, _ := reloaded.GetAccount("work")
			tt.check(t, got)
		})
	}
}
