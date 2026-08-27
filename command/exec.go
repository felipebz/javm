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

	"github.com/felipebz/javm/cfg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "exec [--jdk <selector>] <command> [args...]",
		Short:              "Execute a command with a selected JDK",
		DisableFlagParsing: true,
		Long: "Select a JDK and execute a process with JAVA_HOME and PATH set only for that process.\n" +
			"Native executables run directly; Windows .cmd and .bat wrappers use the system command interpreter.\n" +
			"Use --jdk to select a JDK explicitly; otherwise the selector is read from .java-version.\n" +
			"JAVA_HOME and PATH are changed only for the child process; javm init is not required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isExecHelpRequest(args) {
				return pflag.ErrHelp
			}
			selector, childArgs, err := parseExecArgs(cmd, args)
			if err != nil {
				return err
			}
			return runExec(cmd.Context(), cmd, selector, childArgs)
		},
		Example: "  javm exec java --version\n" +
			"  javm exec ./gradlew test\n" +
			"  javm exec --jdk 21 java --version\n" +
			"  javm exec --jdk temurin@21 mvn test",
	}
}

func parseExecArgs(cmd *cobra.Command, args []string) (string, []string, error) {
	const usage = "usage: javm exec [--jdk <selector>] <command> [args...]"
	if len(args) == 0 {
		return "", nil, UsageError(errors.New("no command specified; " + usage))
	}

	selector := ""
	commandIndex := 0
	if args[0] == "--jdk" {
		if len(args) < 2 || args[1] == "" {
			return "", nil, UsageError(errors.New("--jdk requires a selector; " + usage))
		}
		selector = args[1]
		commandIndex = 2
	}
	if commandIndex >= len(args) || args[commandIndex] == "" {
		return "", nil, UsageError(errors.New("no command specified; " + usage))
	}

	if selector == "" {
		selector = cfg.ReadJavaVersion()
		if selector == "" {
			return "", nil, UsageError(errors.New("no JDK selector specified; provide a selector or create .java-version"))
		}
	}
	return selector, args[commandIndex:], nil
}

func isExecHelpRequest(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}

func runExec(ctx context.Context, cmd *cobra.Command, selector string, childArgs []string) error {
	jdk, err := resolveJDKContext(ctx, selector)
	if err != nil {
		return err
	}

	childEnv := os.Environ()
	selected, err := buildJDKEnvironment(jdk.Path, childEnv)
	if err != nil {
		return fmt.Errorf("prepare environment for %s: %w", selector, err)
	}
	if err := validateJDKHome(selected.JavaHome); err != nil {
		return fmt.Errorf("selected JDK %q is inaccessible: %w", jdk.Identifier, err)
	}
	childEnv = setEnvironmentValue(childEnv, "JAVA_HOME", selected.JavaHome)
	childEnv = setEnvironmentValue(childEnv, "PATH", selected.Path)

	resolved, err := findExecutableInEnvironment(childArgs[0], childEnv)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("command %q is not executable in the environment for %s: %w", childArgs[0], selector, err)
		}
		return NotFoundError(fmt.Errorf("command %q was not found in the environment for %s", childArgs[0], selector))
	}

	child, err := newExecProcess(ctx, resolved, childArgs[1:], childEnv)
	if err != nil {
		return fmt.Errorf("prepare command %q: %w", childArgs[0], err)
	}
	child.Env = childEnv
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()

	if err := child.Start(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("start command %q: %w", childArgs[0], err)
	}
	if err := child.Wait(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
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
