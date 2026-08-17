package discovery

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var identifierRegexp = regexp.MustCompile("[^a-z0-9]+")

func ScanLocationsForJDKs(root string, vfs fs.FS, runner Runner, locations []string, sourceName string) ([]JDK, error) {
	return ScanLocationsForJDKsContext(context.Background(), root, vfs, runner, locations, sourceName)
}

func ScanLocationsForJDKsContext(ctx context.Context, root string, vfs fs.FS, runner Runner, locations []string, sourceName string) ([]JDK, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var jdks []JDK

	for _, location := range locations {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := fs.Stat(vfs, location); err != nil {
			continue
		}

		err := fs.WalkDir(vfs, location, makeJDKWalkFunc(ctx, vfs, runner, root, sourceName, &jdks))

		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, fmt.Errorf("failed to walk directory %s: %w", location, err)
		}
	}

	return jdks, nil
}

func makeJDKWalkFunc(ctx context.Context, vfs fs.FS, runner Runner, root, sourceName string, jdks *[]JDK) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return nil // Skip this path on error
		}
		if !d.IsDir() {
			return nil
		}
		jdk, ok, err := ValidateJDKContext(ctx, vfs, runner, root, p, sourceName)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
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
		osSpecificSubDir = path.Join("Contents", "Home")
	}
	return path.Join(dir, osSpecificSubDir)
}

func ExpectedJavaPath(dir string, goos string) string {
	java := "java"
	if goos == "windows" {
		java = "java.exe"
	}
	return path.Join(ExpectedJDKDir(dir, goos), "bin", java)
}

func ValidateJDK(vfs fs.FS, runner Runner, root, p, source string) (JDK, bool, error) {
	return ValidateJDKContext(context.Background(), vfs, runner, root, p, source)
}

func ValidateJDKContext(ctx context.Context, vfs fs.FS, runner Runner, root, p, source string) (JDK, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return JDK{}, false, err
	}
	jdkPath := ExpectedJDKDir(p, runtime.GOOS)
	javaPath := ExpectedJavaPath(p, runtime.GOOS)
	if _, err := fs.Stat(vfs, javaPath); err != nil {
		return JDK{}, false, nil
	}

	md, err := ExtractMetadataFromReleaseFile(vfs, jdkPath)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return JDK{}, false, ctxErr
	}

	fullPath := filepath.Join(root, filepath.FromSlash(p))

	result := JDK{
		Path:         fullPath,
		Version:      md["JAVA_VERSION"],
		Vendor:       md["JAVA_VENDOR"],
		Architecture: normalizeArchitecture(md["OS_ARCH"]),
		Source:       source,
	}

	if result.Version == "" || result.Vendor == "" || result.Architecture == "" {
		execPath := filepath.Join(root, filepath.FromSlash(javaPath))
		md, err = ExtractMetadataFromJavaVersionContext(ctx, runner, execPath)
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
		result.Identifier = filepath.Base(filepath.FromSlash(p))
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
	return ExtractMetadataFromJavaVersionContext(context.Background(), runner, javaPath)
}

func ExtractMetadataFromJavaVersionContext(ctx context.Context, runner Runner, javaPath string) (map[string]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	out, err := runner.CombinedOutput(ctx, javaPath, "-XshowSettings:properties", "-version")
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
	v = identifierRegexp.ReplaceAllString(v, "-")
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
