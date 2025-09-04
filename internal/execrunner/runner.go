package execrunner

import (
	"context"
	"os/exec"
)

// CmdRunner provides an abstraction around running system commands so we can mock it in tests
type CmdRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
	Run(ctx context.Context, name string, args ...string) error
}

// RealCmdRunner is the production implementation using exec.CommandContext
type RealCmdRunner struct{}

// CombinedOutput runs a command and returns its combined standard output and standard error
func (r *RealCmdRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// Run runs a command and returns an error if it fails
func (r *RealCmdRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
