//go:build windows

package command

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

const promoteRetryTimeout = 10 * time.Second

func promoteNoReplace(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(promoteRetryTimeout)
	delay := 10 * time.Millisecond
	for {
		err := windows.MoveFile(from, to)
		if err == nil || !errors.Is(err, windows.ERROR_ACCESS_DENIED) && !errors.Is(err, windows.ERROR_SHARING_VIOLATION) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(delay)
		delay = min(delay*2, 500*time.Millisecond)
	}
}
