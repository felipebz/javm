//go:build windows

package state

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

func replaceFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}

	const attempts = 10
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		lastErr = windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if lastErr == nil {
			return nil
		}
		if !errors.Is(lastErr, windows.ERROR_ACCESS_DENIED) && !errors.Is(lastErr, windows.ERROR_SHARING_VIOLATION) {
			return lastErr
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return lastErr
}

func syncDirectory(string) error {
	// MOVEFILE_WRITE_THROUGH provides the durability guarantee on Windows.
	return nil
}
