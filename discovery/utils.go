package discovery

import (
	"bufio"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

func ScanLocationsForJDKs(vfs fs.FS, locations []string, sourceName string) ([]JDK, error) {
	var jdks []JDK

	for _, location := range locations {
		if _, err := fs.Stat(vfs, location); err != nil {
			continue
		}

		err := fs.WalkDir(vfs, location, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip this path on error
			}
			if !d.IsDir() {
				return nil
			}
			jdk, ok, err := ValidateJDK(vfs, p, sourceName)
			if err != nil {
				return nil // Skip this path on error
			}
			if ok {
				jdks = append(jdks, jdk)
				return fs.SkipDir
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", location, err)
		}
	}

	return jdks, nil
}

func ValidateJDK(vfs fs.FS, p, source string) (JDK, bool, error) {
	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}
	javaPath := path.Join(p, "bin", javaExe)
	if _, err := fs.Stat(vfs, javaPath); err != nil {
		return JDK{}, false, nil
	}

	md, err := ExtractMetadataFromReleaseFile(vfs, p)
	if err == nil {
		return JDK{
			Path:           p,
			Version:        md["JAVA_VERSION"],
			Vendor:         md["JAVA_VENDOR"],
			Implementation: md["IMPLEMENTOR"],
			Architecture:   md["OS_ARCH"],
			Source:         source,
		}, true, nil
	}

	md, err = ExtractMetadataFromJavaVersion(javaPath)
	if err != nil {
		return JDK{}, false, fmt.Errorf("failed to extract metadata: %w", err)
	}

	return JDK{
		Path:           p,
		Version:        md["version"],
		Vendor:         md["vendor"],
		Implementation: md["implementation"],
		Architecture:   md["architecture"],
		Source:         source,
	}, true, nil
}

func ExtractMetadataFromReleaseFile(vfs fs.FS, jdkDir string) (map[string]string, error) {
	b, err := fs.ReadFile(vfs, path.Join(jdkDir, "release"))
	if err != nil {
		return nil, err
	}
	md := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		md[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return md, nil
}

func ExtractMetadataFromJavaVersion(javaPath string) (map[string]string, error) {
	cmd := exec.Command(javaPath, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run java -version: %w", err)
	}

	md := ParseJavaVersionOutput(string(out))

	if md["version"] == "" {
		return nil, fmt.Errorf("failed to extract metadata")
	}
	return md, nil
}

func DeduplicateJDKs(jdks []JDK) []JDK {
	seen := make(map[string]bool)
	var result []JDK

	for _, jdk := range jdks {
		if !seen[jdk.Path] {
			seen[jdk.Path] = true
			result = append(result, jdk)
		}
	}

	return result
}
