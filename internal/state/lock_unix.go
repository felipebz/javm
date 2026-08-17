//go:build !windows

package state

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockExclusive(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlockExclusive(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
