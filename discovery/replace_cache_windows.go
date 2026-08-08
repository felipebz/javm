//go:build windows

package discovery

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

func replaceCacheFile(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}

	const attempts = 10
	for attempt := 0; attempt < attempts; attempt++ {
		err = windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if err == nil {
			return nil
		}
		if !errors.Is(err, windows.ERROR_ACCESS_DENIED) && !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return err
}
