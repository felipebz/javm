//go:build !linux

package discoapi

func isMuslLibc() bool {
	return false
}
