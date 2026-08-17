// Package state contains the shared filesystem primitives used for javm state.
package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWriteFile writes data to a temporary file in the destination directory,
// syncs it and replaces the destination without exposing a partial file.
// Callers that perform a read-modify-write sequence should hold WithFileLock
// for the whole sequence.
func AtomicWriteFile(destination string, data []byte, permission fs.FileMode) (err error) {
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, temporaryPattern(destination))
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if temporaryPath == "" {
			return
		}
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, fmt.Errorf("remove temporary state file: %w", removeErr))
		}
	}()

	if err := temporary.Chmod(permission.Perm()); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary state file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary state file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary state file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary state file: %w", err)
	}

	if err := replaceFile(temporaryPath, destination); err != nil {
		return fmt.Errorf("replace state file: %w", err)
	}
	temporaryPath = ""

	if err := syncDirectory(directory); err != nil {
		return fmt.Errorf("sync state directory: %w", err)
	}
	return nil
}

func temporaryPattern(destination string) string {
	base := filepath.Base(destination)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return "." + base + "-*"
}
