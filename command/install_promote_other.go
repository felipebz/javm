//go:build !linux && !darwin && !windows

package command

import "os"

// javm currently installs JDKs only on Linux, macOS, and Windows. This fallback
// exists so the package remains buildable on other Go platforms.
func promoteNoReplace(source, destination string) error {
	return os.Rename(source, destination)
}
