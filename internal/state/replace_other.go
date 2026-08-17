//go:build !windows

package state

import (
	"errors"
	"os"
)

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}

func syncDirectory(directory string) error {
	dir, err := os.Open(directory)
	if err != nil {
		return err
	}

	syncErr := dir.Sync()
	closeErr := dir.Close()
	return errors.Join(syncErr, closeErr)
}
