//go:build !windows

package state

// hideDirectory is a no-op: the leading dot already hides the directory.
func hideDirectory(string) {}
