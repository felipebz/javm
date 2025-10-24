package discovery

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"runtime"
	"strings"
)

func ScanLocationsForJDKs(vfs fs.FS, runner Runner, locations []string, sourceName string) ([]JDK, error) {
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
			jdk, ok, err := ValidateJDK(vfs, runner, p, sourceName)
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

func ValidateJDK(vfs fs.FS, runner Runner, p, source string) (JDK, bool, error) {
	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}
	javaPath := path.Join(p, "bin", javaExe)
	if _, err := fs.Stat(vfs, javaPath); err != nil {
		return JDK{}, false, nil
	}

	md, err := ExtractMetadataFromReleaseFile(vfs, p)

	result := JDK{
		Path:         p,
		Version:      md["JAVA_VERSION"],
		Vendor:       md["JAVA_VENDOR"],
		Architecture: md["OS_ARCH"],
		Source:       source,
	}

	if result.Version == "" || result.Vendor == "" || result.Architecture == "" {
		md, err = ExtractMetadataFromJavaVersion(runner, javaPath)
		if err != nil {
			return JDK{}, false, fmt.Errorf("failed to extract metadata: %w", err)
		}

		if result.Version == "" {
			result.Version = md["version"]
		}
		if result.Vendor == "" {
			result.Vendor = md["vendor"]
		}
		if result.Architecture == "" {
			result.Architecture = md["architecture"]
		}
	}

	return result, true, nil
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

func ExtractMetadataFromJavaVersion(runner Runner, javaPath string) (map[string]string, error) {
	out, err := runner.CombinedOutput(javaPath, "-XshowSettings:properties", "-version")
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
