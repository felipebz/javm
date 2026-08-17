package discovery

import (
	"context"
	"os/exec"
)

type Runner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}
