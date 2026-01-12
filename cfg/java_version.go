package cfg

import (
	"io/fs"
	"os"
	"strings"
)

// ReadJavaVersion reads the .java-version file from the current working directory and returns the trimmed version
// string. If the file does not exist or cannot be read, it returns an empty string.
func ReadJavaVersion() string {
	cwd, err := os.Getwd()
	if err == nil {
		return ReadJavaVersionFromFS(os.DirFS(cwd))
	}
	return ""
}

func ReadJavaVersionFromFS(vfs fs.FS) string {
	b, err := fs.ReadFile(vfs, ".java-version")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
