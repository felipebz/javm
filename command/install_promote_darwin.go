//go:build darwin

package command

import "golang.org/x/sys/unix"

func promoteNoReplace(source, destination string) error {
	return unix.RenamexNp(source, destination, unix.RENAME_EXCL)
}
