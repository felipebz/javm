package cfg

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadJavaVersion reads the .java-version file from the current working directory and returns the trimmed version
// string. If the file does not exist or cannot be read, it returns an empty string.
func ReadJavaVersion() string {
	cwd, err := os.Getwd()
	if err == nil {
		path := filepath.Join(cwd, ".java-version")
		b, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	return ""
}
