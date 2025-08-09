package discovery

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func ScanLocationsForJDKs(locations []string, sourceName string) ([]JDK, error) {
	var jdks []JDK

	for _, location := range locations {
		if _, err := os.Stat(location); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(location, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip this path on error
			}
			if !info.IsDir() {
				return nil
			}
			jdk, ok, err := ValidateJDK(path, sourceName)
			if err != nil {
				return nil // Skip this path on error
			}
			if ok {
				jdks = append(jdks, jdk)
				return filepath.SkipDir
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", location, err)
		}
	}

	return jdks, nil
}

func ValidateJDK(path string, source string) (JDK, bool, error) {
	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}
	javaPath := filepath.Join(path, "bin", javaExe)
	if _, err := os.Stat(javaPath); os.IsNotExist(err) {
		return JDK{}, false, nil
	}

	metadata, err := ExtractMetadataFromReleaseFile(path)
	if err == nil {
		return JDK{
			Path:           path,
			Version:        metadata["JAVA_VERSION"],
			Vendor:         metadata["JAVA_VENDOR"],
			Implementation: metadata["IMPLEMENTOR"],
			Architecture:   metadata["OS_ARCH"],
			Source:         source,
		}, true, nil
	}

	metadata, err = ExtractMetadataFromJavaVersion(javaPath)
	if err != nil {
		return JDK{}, false, fmt.Errorf("failed to extract metadata: %w", err)
	}

	return JDK{
		Path:           path,
		Version:        metadata["version"],
		Vendor:         metadata["vendor"],
		Implementation: metadata["implementation"],
		Architecture:   metadata["architecture"],
		Source:         source,
	}, true, nil
}

func ExtractMetadataFromReleaseFile(jdkPath string) (map[string]string, error) {
	releaseFile := filepath.Join(jdkPath, "release")
	if _, err := os.Stat(releaseFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("release file not found")
	}

	file, err := os.Open(releaseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open release file: %w", err)
	}
	defer file.Close()

	metadata := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		metadata[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read release file: %w", err)
	}

	return metadata, nil
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
