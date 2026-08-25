//go:build !windows

package command

import (
	"context"
	osExec "os/exec"
)

func newExecProcess(ctx context.Context, resolved string, args, _ []string) (*osExec.Cmd, error) {
	return osExec.CommandContext(ctx, resolved, args...), nil
}
