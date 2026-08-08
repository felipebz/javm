//go:build !windows

package discovery

import "os"

func replaceCacheFile(source, destination string) error {
	return os.Rename(source, destination)
}
