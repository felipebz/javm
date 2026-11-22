package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WithFileLock serializes state mutations across goroutines and processes.
// The lock for <dir>/<name> is <dir>/.locks/<name>.lock, so the retained lock
// files stay in one directory, hidden on Windows, instead of next to the
// user's files. They are intentionally retained: removing a lock file after
// releasing it would race with a process that is opening the same lock.
func WithFileLock(path string, fn func() error) (err error) {
	if fn == nil {
		return errors.New("state lock callback is nil")
	}

	directory := filepath.Join(filepath.Dir(path), ".locks")
	if _, statErr := os.Stat(directory); errors.Is(statErr, fs.ErrNotExist) {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create state lock directory: %w", err)
		}
		hideDirectory(directory)
	}
	lockPath := filepath.Join(directory, filepath.Base(path)+".lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open state lock: %w", err)
	}
	if err := lock.Chmod(0o600); err != nil {
		_ = lock.Close()
		return fmt.Errorf("restrict state lock permissions: %w", err)
	}
	if err := lockExclusive(lock); err != nil {
		_ = lock.Close()
		return fmt.Errorf("acquire state lock: %w", err)
	}
	defer func() {
		if unlockErr := unlockExclusive(lock); unlockErr != nil {
			err = errors.Join(err, fmt.Errorf("release state lock: %w", unlockErr))
		}
		if closeErr := lock.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close state lock: %w", closeErr))
		}
	}()

	return fn()
}
