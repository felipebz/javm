//go:build windows

package state

import "golang.org/x/sys/windows"

// hideDirectory marks dir hidden in Explorer and dir; a failure only leaves it
// visible, so it is ignored.
func hideDirectory(dir string) {
	path, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return
	}
	attributes, err := windows.GetFileAttributes(path)
	if err != nil {
		return
	}
	_ = windows.SetFileAttributes(path, attributes|windows.FILE_ATTRIBUTE_HIDDEN)
}
