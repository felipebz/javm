package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/felipebz/javm/discovery"
	"github.com/spf13/cobra"
)

var lookPath = exec.LookPath

// macOSJavaLauncher and javaHomeRunner are variables so tests can simulate
// the macOS launcher stub and /usr/libexec/java_home.
var (
	macOSJavaLauncher                  = "/usr/bin/java"
	javaHomeRunner    discovery.Runner = discovery.ExecRunner{}
)

var errNoActiveJDK = errors.New("no JDK known to javm is active in this shell")

func NewCurrentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the JDK active in the current shell",
		Args:  UsageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			ver, err := current(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), sanitizeTerminalText(ver)); err != nil {
				return fmt.Errorf("write current version: %w", err)
			}
			return nil
		},
	}
}

// current returns the identifier of the discovered JDK whose home is active:
// the JAVA_HOME javm activated, or else the home of the java found on PATH.
func current(ctx context.Context) (string, error) {
	home, fromMarker := activeJavaHome(os.Environ())
	javaPath := ""
	if !fromMarker || home == "" {
		fromMarker = false
		found, err := lookPath("java")
		if err != nil {
			return "", NotFoundError(errNoActiveJDK)
		}
		javaPath = found
		if resolved, err := filepath.EvalSymlinks(found); err == nil {
			javaPath = resolved
		}
		home = filepath.Dir(filepath.Dir(javaPath))
	}
	launcher := !fromMarker && hostGOOS == "darwin" && javaPath == macOSJavaLauncher
	if launcher {
		home = launcherJavaHome(ctx)
		if home == "" {
			return "", NotFoundError(fmt.Errorf("%w (java on PATH is the macOS launcher %s, which finds no JDK to run)", errNoActiveJDK, macOSJavaLauncher))
		}
	}

	jdks, err := LsContext(ctx, false)
	if err != nil {
		return "", err
	}
	if jdk, ok := jdkWithHome(jdks, home); ok {
		return jdk.Identifier, nil
	}
	if fromMarker {
		return "", NotFoundError(fmt.Errorf("javm activated %s, which is no longer a known JDK", sanitizeTerminalText(home)))
	}
	if launcher {
		return "", NotFoundError(fmt.Errorf("%w (java on PATH is the macOS launcher %s, which runs %s)", errNoActiveJDK, macOSJavaLauncher, sanitizeTerminalText(home)))
	}
	return "", NotFoundError(fmt.Errorf("%w (java on PATH is %s)", errNoActiveJDK, sanitizeTerminalText(javaPath)))
}

// launcherJavaHome returns the JDK home the macOS java launcher runs:
// JAVA_HOME when it holds a java, otherwise what /usr/libexec/java_home
// reports, or "" when there is none.
func launcherJavaHome(ctx context.Context) string {
	if home := os.Getenv("JAVA_HOME"); home != "" {
		if info, err := os.Stat(filepath.Join(home, "bin", "java")); err == nil && info.Mode().IsRegular() {
			return home
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := javaHomeRunner.CombinedOutput(ctx, "/usr/libexec/java_home")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// jdkWithHome finds the JDK whose home is home, preferring one managed by
// javm, then the first in discovery order. On macOS a JDK's home is
// <path>/Contents/Home, so both forms match.
func jdkWithHome(jdks []discovery.JDK, home string) (discovery.JDK, bool) {
	want := canonicalHome(home)
	var match discovery.JDK
	found := false
	for _, jdk := range jdks {
		if !sameHome(canonicalHome(jdk.Path), want) && !sameHome(canonicalHome(filepath.Join(jdk.Path, "Contents", "Home")), want) {
			continue
		}
		if jdk.Source == "javm" {
			return jdk, true
		}
		if !found {
			match, found = jdk, true
		}
	}
	return match, found
}

func canonicalHome(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	return filepath.Clean(path)
}

func sameHome(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
