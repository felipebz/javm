//go:build windows

package command

import (
	"context"
	"fmt"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func newExecProcess(ctx context.Context, resolved string, args, env []string) (*osExec.Cmd, error) {
	ext := strings.ToLower(filepath.Ext(resolved))
	if ext != ".bat" && ext != ".cmd" {
		return osExec.CommandContext(ctx, resolved, args...), nil
	}

	comspec, ok := lookupEnvironmentValue(env, "COMSPEC")
	if !ok || comspec == "" {
		comspec = "cmd.exe"
	}
	resolvedComspec, err := findExecutableInEnvironment(comspec, env)
	if err != nil {
		return nil, fmt.Errorf("locate Windows command interpreter %q: %w", comspec, err)
	}

	child := osExec.CommandContext(ctx, resolvedComspec)
	child.Args = nil
	child.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: windowsBatchCommandLine(resolvedComspec, resolved, args),
	}
	return child, nil
}

func windowsBatchCommandLine(comspec, batch string, args []string) string {
	command := quoteCmdArgument(batch)
	for _, arg := range args {
		command += " " + quoteCmdArgument(arg)
	}
	return quoteWindowsProcessArgument(comspec) + " /d /v:off /s /c \"" + command + "\""
}

// quoteCmdArgument quotes one token for cmd.exe. Batch files are interpreted
// by cmd.exe rather than by CreateProcess, so Go's normal Windows argument
// quoting cannot be used for this path. Double quotes protect spaces and cmd
// metacharacters; backslash-quote preserves an embedded quote when the batch
// file forwards %* to the native process it wraps.
func quoteCmdArgument(value string) string {
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// quoteWindowsProcessArgument follows the CreateProcess quoting convention
// for the command interpreter path itself. The path normally comes from the
// absolute COMSPEC environment variable, but it may still contain spaces.
func quoteWindowsProcessArgument(value string) string {
	if value == "" {
		return `""`
	}
	if !strings.ContainsAny(value, " \t\"") {
		return value
	}

	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range value {
		switch r {
		case '\\':
			backslashes++
		case '"':
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
			b.WriteRune(r)
			backslashes = 0
		default:
			b.WriteString(strings.Repeat("\\", backslashes))
			b.WriteRune(r)
			backslashes = 0
		}
	}
	b.WriteString(strings.Repeat("\\", backslashes*2))
	b.WriteByte('"')
	return b.String()
}
