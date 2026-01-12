package discovery

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func ScanLocationsForJDKs(root string, vfs fs.FS, runner Runner, locations []string, sourceName string) ([]JDK, error) {
	var jdks []JDK

	for _, location := range locations {
		if _, err := fs.Stat(vfs, location); err != nil {
			continue
		}

		err := fs.WalkDir(vfs, location, makeJDKWalkFunc(vfs, runner, root, sourceName, &jdks))

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", location, err)
		}
	}

	return jdks, nil
}

func makeJDKWalkFunc(vfs fs.FS, runner Runner, root, sourceName string, jdks *[]JDK) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip this path on error
		}
		if !d.IsDir() {
			return nil
		}
		jdk, ok, err := ValidateJDK(vfs, runner, root, p, sourceName)
		if err != nil {
			return nil // Skip this path on error
		}
		if ok {
			*jdks = append(*jdks, jdk)
			return fs.SkipDir
		}
		return nil
	}
}

func ExpectedJDKDir(dir string, goos string) string {
	var osSpecificSubDir = ""
	if goos == "darwin" {
		osSpecificSubDir = filepath.Join("Contents", "Home")
	}
	return filepath.Join(dir, osSpecificSubDir)
}

func ExpectedJavaPath(dir string, goos string) string {
	java := "java"
	if goos == "windows" {
		java = "java.exe"
	}
	return filepath.Join(ExpectedJDKDir(dir, goos), "bin", java)
}

func ValidateJDK(vfs fs.FS, runner Runner, root, p, source string) (JDK, bool, error) {
	jdkPath := ExpectedJDKDir(p, runtime.GOOS)
	javaPath := ExpectedJavaPath(p, runtime.GOOS)
	if _, err := fs.Stat(vfs, javaPath); err != nil {
		return JDK{}, false, nil
	}

	md, err := ExtractMetadataFromReleaseFile(vfs, jdkPath)
	result := JDK{
		Path:         filepath.Join(root, p),
		Version:      md["JAVA_VERSION"],
		Vendor:       md["JAVA_VENDOR"],
		Architecture: normalizeArchitecture(md["OS_ARCH"]),
		Source:       source,
	}

	if result.Version == "" || result.Vendor == "" || result.Architecture == "" {
		md, err = ExtractMetadataFromJavaVersion(runner, filepath.Join(root, javaPath))
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
			result.Architecture = normalizeArchitecture(md["architecture"])
		}
	}

	if source == "javm" {
		result.Identifier = filepath.Base(p)
	} else {
		result.Identifier = generateSystemIdentifier(result.Vendor, result.Version, source)
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

func normalizeArchitecture(arch string) string {
	if arch == "x86_64" || arch == "amd64" {
		return "x64"
	}
	return arch
}

func generateSystemIdentifier(vendor, version, source string) string {
	v := strings.ToLower(vendor)
	reg, _ := regexp.Compile("[^a-z0-9]+")
	v = reg.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")

	if v == "" {
		v = source
	}

	major := version
	if strings.HasPrefix(version, "1.") {
		parts := strings.Split(version, ".")
		if len(parts) > 1 {
			major = parts[1]
		}
	} else {
		parts := strings.Split(version, ".")
		if len(parts) > 0 {
			major = parts[0]
		}
	}

	return fmt.Sprintf("%s-%s@%s", v, source, major)
}
