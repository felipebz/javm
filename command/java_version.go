package command

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// javaVersionSelector returns the selector from .java-version for commands
// run without a version argument, or an error that says why there is none.
// Locating the file stays here, so callers do not change if the lookup moves
// beyond the current directory. Paths are sanitized because the error text
// reaches the terminal.
func javaVersionSelector() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("find .java-version: %w", err)
	}
	path := filepath.Join(cwd, ".java-version")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("no version given and no .java-version found in %s", sanitizeTerminalText(cwd))
	}
	if err != nil {
		// The cause stays reachable with errors.Is; its own text repeats the
		// unsanitized path, so only the inner error is wrapped.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return "", fmt.Errorf("read %s: %w", sanitizeTerminalText(path), err)
	}
	selector := strings.TrimSpace(string(data))
	if selector == "" {
		return "", fmt.Errorf("%s is empty", sanitizeTerminalText(path))
	}
	return selector, nil
}
