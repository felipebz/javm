package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "exec [--jdk <selector>] [--] <command> [args...]",
		Short:              "Execute a command with a selected JDK",
		DisableFlagParsing: true,
		Long: "Select a JDK and execute a process with JAVA_HOME and PATH set only for that process.\n" +
			"Native executables run directly; Windows .cmd and .bat wrappers use the system command interpreter.\n" +
			"Use --jdk <selector> or --jdk=<selector> to select a JDK explicitly; otherwise the selector is read\n" +
			"from the nearest .java-version in the current directory or its parents.\n" +
			"javm options end at the first argument that is not one of them, or at --; everything after the\n" +
			"command goes to it unchanged. javm exits with the command's exit status (128+n if a signal ends it).\n" +
			"JAVA_HOME and PATH are changed only for the child process; javm init is not required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseExecArgs(args)
			if err != nil {
				return err
			}
			logger := loggerFromContext(cmd.Context())
			if parsed.debug {
				logger.SetLevel(log.DebugLevel)
			} else if parsed.quiet && logger.GetLevel() != log.DebugLevel {
				logger.SetLevel(log.WarnLevel)
			}
			err = runExec(cmd.Context(), cmd, parsed.selector, parsed.command)
			var childErr *childExitError
			if errors.As(err, &childErr) {
				// The child has already reported its failure.
				cmd.SilenceErrors = true
			}
			return err
		},
		Example: "  javm exec java --version\n" +
			"  javm exec ./gradlew test\n" +
			"  javm exec --jdk 21 java --version\n" +
			"  javm exec --jdk=temurin@21 -- mvn test",
	}
}

type execArgs struct {
	selector     string
	command      []string
	quiet, debug bool
}

// parseExecArgs reads javm options up to the first other argument or "--";
// the rest is the command and its arguments, passed through unchanged.
func parseExecArgs(args []string) (execArgs, error) {
	const usage = "usage: javm exec [--jdk <selector>] [--] <command> [args...]"
	var parsed execArgs
	i := 0
options:
	for ; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			i++
			break options
		case arg == "-h" || arg == "--help":
			return execArgs{}, pflag.ErrHelp
		case arg == "--jdk" || strings.HasPrefix(arg, "--jdk="):
			value, inline := strings.CutPrefix(arg, "--jdk=")
			if !inline {
				value = ""
				if i+1 < len(args) {
					i++
					value = args[i]
				}
			}
			if value == "" {
				return execArgs{}, UsageError(errors.New("--jdk requires a selector; " + usage))
			}
			parsed.selector = value
		case arg == "--quiet":
			parsed.quiet = true
		case arg == "--debug":
			parsed.debug = true
		case strings.HasPrefix(arg, "-"):
			return execArgs{}, UsageError(fmt.Errorf("unknown exec option %q; put -- before a command that starts with '-'; %s", arg, usage))
		default:
			break options
		}
	}
	if i >= len(args) || args[i] == "" {
		return execArgs{}, UsageError(errors.New("no command specified; " + usage))
	}
	parsed.command = args[i:]

	if parsed.selector == "" {
		selector, err := javaVersionSelector()
		if err != nil {
			return execArgs{}, UsageError(err)
		}
		parsed.selector = selector
	}
	return parsed, nil
}

// childExitError carries the exit status of the command started by exec, so
// javm exits with it without printing anything of its own.
type childExitError struct {
	code int
	err  *osExec.ExitError
}

func (e *childExitError) Error() string { return e.err.Error() }
func (e *childExitError) Unwrap() error { return e.err }
func (e *childExitError) ExitCode() int { return e.code }

// childExitStatus is the exit code, or 128+n when signal n ended the child,
// as shells report it. Windows has no such signals and keeps ExitCode.
func childExitStatus(err *osExec.ExitError) int {
	status, ok := err.Sys().(interface {
		Signaled() bool
		Signal() syscall.Signal
	})
	if ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return err.ExitCode()
}

func runExec(ctx context.Context, cmd *cobra.Command, selector string, childArgs []string) error {
	jdk, err := resolveJDKContext(ctx, selector)
	if err != nil {
		return err
	}

	childEnv := os.Environ()
	selected, err := buildJDKEnvironment(jdk.Path, childEnv)
	if err != nil {
		return fmt.Errorf("prepare environment for %s: %w", sanitizeTerminalText(selector), err)
	}
	if err := validateJDKHome(selected.JavaHome); err != nil {
		return fmt.Errorf("selected JDK %q is inaccessible: %w", jdk.Identifier, err)
	}
	childEnv = setEnvironmentValue(childEnv, "JAVA_HOME", selected.JavaHome)
	childEnv = setEnvironmentValue(childEnv, "PATH", selected.Path)

	resolved, err := findExecutableInEnvironment(childArgs[0], childEnv)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("command %q is not executable in the environment for %s: %w", childArgs[0], sanitizeTerminalText(selector), err)
		}
		return NotFoundError(fmt.Errorf("command %q was not found in the environment for %s", childArgs[0], sanitizeTerminalText(selector)))
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	// Ctrl-C also reaches the child, which shares the terminal's process
	// group; the child decides how to stop, and javm reports how it exited.
	child, err := newExecProcess(context.WithoutCancel(ctx), resolved, childArgs[1:], childEnv)
	if err != nil {
		return fmt.Errorf("prepare command %q: %w", childArgs[0], err)
	}
	child.Env = childEnv
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()

	if err := child.Start(); err != nil {
		return fmt.Errorf("start command %q: %w", childArgs[0], err)
	}
	if err := child.Wait(); err != nil {
		var exitErr *osExec.ExitError
		if errors.As(err, &exitErr) {
			return &childExitError{code: childExitStatus(exitErr), err: exitErr}
		}
		return fmt.Errorf("execute command %q: %w", childArgs[0], err)
	}
	return nil
}

func validateJDKHome(javaHome string) error {
	info, err := os.Stat(javaHome)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("JDK home is not a directory")
	}
	binInfo, err := os.Stat(filepath.Join(javaHome, "bin"))
	if err != nil {
		return fmt.Errorf("inspect JDK bin directory: %w", err)
	}
	if !binInfo.IsDir() {
		return fmt.Errorf("JDK bin path is not a directory")
	}
	return nil
}

func findExecutableInEnvironment(file string, env []string) (string, error) {
	if file == "" {
		return "", osExec.ErrNotFound
	}
	pathExt := pathExtensions(env)
	if commandHasPath(file) {
		return findExecutableCandidate(file, pathExt)
	}

	pathValue, _ := lookupEnvironmentValue(env, "PATH")
	var permissionErr error
	for _, dir := range filepath.SplitList(pathValue) {
		if runtime.GOOS == "windows" && dir == "" {
			// Match Windows os/exec behavior: empty PATH entries are not
			// implicit current-directory lookups.
			continue
		}
		candidate := filepath.Join(dir, file)
		if !commandHasPath(candidate) {
			candidate = "." + string(filepath.Separator) + candidate
		}
		if resolved, err := findExecutableCandidate(candidate, pathExt); err == nil {
			return resolved, nil
		} else if errors.Is(err, os.ErrPermission) && permissionErr == nil {
			permissionErr = err
		}
	}
	if permissionErr != nil {
		return "", permissionErr
	}
	return "", osExec.ErrNotFound
}

func commandHasPath(file string) bool {
	if runtime.GOOS == "windows" {
		return strings.ContainsAny(file, `:\/`) || filepath.VolumeName(file) != ""
	}
	return strings.Contains(file, string(filepath.Separator))
}

func findExecutableCandidate(file string, extensions []string) (string, error) {
	if runtime.GOOS != "windows" {
		info, err := os.Stat(file)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", os.ErrPermission
		}
		if info.Mode().Perm()&0o111 == 0 {
			return "", os.ErrPermission
		}
		return file, nil
	}

	if info, err := os.Stat(file); err == nil {
		if info.IsDir() {
			return "", os.ErrPermission
		}
		return file, nil
	}
	for _, extension := range extensions {
		candidate := file + extension
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", osExec.ErrNotFound
}

func pathExtensions(env []string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	value, ok := lookupEnvironmentValue(env, "PATHEXT")
	if !ok || value == "" {
		value = ".com;.exe;.bat;.cmd"
	}
	var result []string
	for extension := range strings.SplitSeq(value, ";") {
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		result = append(result, strings.ToLower(extension))
	}
	return result
}
