package command

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// javaVersionSelector returns the selector from the nearest .java-version in
// the current directory or its parents, for commands run without a version
// argument, or an error that says why there is none. The nearest entry
// decides: an empty or unreadable file is an error rather than a reason to
// look further up. The walk follows the logical path from os.Getwd up to the
// filesystem root. Paths are sanitized because the error text reaches the
// terminal.
func javaVersionSelector() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find .java-version: %w", err)
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, ".java-version")
		data, err := os.ReadFile(path)
		if err == nil {
			selector := strings.TrimSpace(string(data))
			if selector == "" {
				return "", fmt.Errorf("%s is empty", sanitizeTerminalText(path))
			}
			return selector, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			// The cause stays reachable with errors.Is; its own text repeats
			// the unsanitized path, so only the inner error is wrapped.
			var pathErr *fs.PathError
			if errors.As(err, &pathErr) {
				err = pathErr.Err
			}
			return "", fmt.Errorf("read %s: %w", sanitizeTerminalText(path), err)
		}
		if filepath.Dir(dir) == dir {
			return "", fmt.Errorf("no version given and no .java-version found in %s or its parent directories", sanitizeTerminalText(cwd))
		}
	}
}
