package services

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

type fakeRunner struct {
	calls   []string
	outputs map[string][]byte
	failRun bool
}

func (f *fakeRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if out, ok := f.outputs[key]; ok {
		return out, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) error {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if f.failRun {
		return errors.New("run failed")
	}
	return nil
}

func TestSetUserConfiguration_callsRunner(t *testing.T) {
	fr := &fakeRunner{outputs: map[string][]byte{}}
	logger := observability.NewDefaultLogger()
	svc := NewGitConfigService(logger, fr)

	ctx := context.Background()
	if err := svc.SetUserConfiguration(ctx, "Alice Example", "alice@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"git config --global user.name Alice Example",
		"git config --global user.email alice@example.com",
	}

	if !reflect.DeepEqual(fr.calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, fr.calls)
	}
}

func TestSetSSHCommand_setsAndUnsets(t *testing.T) {
	fr := &fakeRunner{outputs: map[string][]byte{}}
	logger := observability.NewDefaultLogger()
	svc := NewGitConfigService(logger, fr)

	ctx := context.Background()

	// Test setting ssh command
	sshCmd := "ssh -i /home/alice/.ssh/id_rsa -o IdentitiesOnly=yes"
	if err := svc.SetSSHCommand(ctx, sshCmd); err != nil {
		t.Fatalf("unexpected error setting ssh command: %v", err)
	}

	// Expect unset global, unset local, then set global
	expectedPrefix := []string{
		"git config --unset --global core.sshcommand",
		"git config --unset --local core.sshcommand",
		"git config --global core.sshcommand ssh -i /home/alice/.ssh/id_rsa -o IdentitiesOnly=yes",
	}

	if len(fr.calls) < len(expectedPrefix) {
		t.Fatalf("expected at least %d calls, got %d", len(expectedPrefix), len(fr.calls))
	}

	for i, v := range expectedPrefix {
		if fr.calls[i] != v {
			t.Fatalf("expected call %d to be %q, got %q", i, v, fr.calls[i])
		}
	}

	// Test unset behavior when empty string
	fr.calls = nil
	if err := svc.SetSSHCommand(ctx, ""); err != nil {
		t.Fatalf("unexpected error unsetting ssh command: %v", err)
	}

	expectedUnset := []string{"git config --unset --global core.sshcommand", "git config --unset --local core.sshcommand"}
	if !reflect.DeepEqual(fr.calls, expectedUnset) {
		t.Fatalf("expected unset calls %v, got %v", expectedUnset, fr.calls)
	}
}
