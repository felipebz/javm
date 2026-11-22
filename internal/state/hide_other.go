//go:build !windows

package state

func hideDirectory(string) {
	// Nothing to do: the leading dot already hides the directory.
}
