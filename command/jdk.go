package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
)

// resolveJDKContext is the common local-resolution path used by commands
// that operate on a selected JDK. It deliberately keeps aliases and the
// discovery selector matching in one place.
func resolveJDKContext(ctx context.Context, selector string) (discovery.JDK, error) {
	if aliasValue := getAlias(selector); aliasValue != "" {
		selector = aliasValue
	}

	jdks, err := LsContext(ctx, false)
	if err != nil {
		return discovery.JDK{}, err
	}
	return FindBestMatchJDK(jdks, selector)
}

type jdkEnvironment struct {
	JavaHome string
	Path     string
}

// buildJDKEnvironment creates the PATH/JAVA_HOME values used by both shell
// activation and child-process execution. env is only read; this function
// never changes the process environment.
func buildJDKEnvironment(jdkPath string, env []string) (jdkEnvironment, error) {
	path, err := filepath.Abs(jdkPath)
	if err != nil {
		return jdkEnvironment{}, fmt.Errorf("resolve JDK path: %w", err)
	}
	if runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}

	currentPath, pathWasSet := lookupEnvironmentValue(env, "PATH")
	remainingPath, hasRemainingPathEntries := stripManagedJDKPathsWithPresence(currentPath)
	if !pathWasSet {
		hasRemainingPathEntries = false
	}
	childPath := filepath.Join(path, "bin")
	if hasRemainingPathEntries {
		childPath += string(os.PathListSeparator) + remainingPath
	}
	return jdkEnvironment{
		JavaHome: path,
		Path:     childPath,
	}, nil
}

func stripManagedJDKPaths(pathValue string) string {
	remaining, _ := stripManagedJDKPathsWithPresence(pathValue)
	return remaining
}

func stripManagedJDKPathsWithPresence(pathValue string) (string, bool) {
	if pathValue == "" {
		return "", true
	}
	entries := filepath.SplitList(pathValue)
	if len(entries) == 0 {
		return pathValue, false
	}

	managedRoot, err := filepath.Abs(filepath.Join(cfg.Dir(), "jdk"))
	if err != nil {
		return pathValue, true
	}

	filtered := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !isUnderPath(entry, managedRoot) {
			filtered = append(filtered, entry)
		}
	}
	return strings.Join(filtered, string(os.PathListSeparator)), len(filtered) > 0
}

func isUnderPath(candidate, root string) bool {
	if candidate == "" {
		return false
	}
	candidate, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func lookupEnvironmentValue(env []string, name string) (string, bool) {
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && environmentKeyEqual(key, name) {
			return value, true
		}
	}
	return "", false
}

func setEnvironmentValue(env []string, name, value string) []string {
	result := make([]string, 0, len(env)+1)
	found := false
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && environmentKeyEqual(key, name) {
			if !found {
				result = append(result, name+"="+value)
				found = true
			}
			continue
		}
		result = append(result, entry)
	}
	if !found {
		result = append(result, name+"="+value)
	}
	return result
}

func environmentKeyEqual(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
