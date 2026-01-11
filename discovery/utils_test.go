package discovery

import (
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
)

func TestScanLocationsForJDKs_FindsValidJDK(t *testing.T) {
	vfs := fstest.MapFS{}
	fakeJDK := createFakeJDK(t, vfs, ".", "jdk-21")

	jdks, err := ScanLocationsForJDKs("", vfs, fakeRunner{}, []string{"."}, "testsource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 1 {
		t.Fatalf("expected 1 JDK, got %d", len(jdks))
	}
	if jdks[0].Path != fakeJDK {
		t.Errorf("expected path %q, got %q", fakeJDK, jdks[0].Path)
	}
	if jdks[0].Source != "testsource" {
		t.Errorf("expected source %q, got %q", "testsource", jdks[0].Source)
	}
}

func TestScanLocationsForJDKs_SkipsNonJDKDirs(t *testing.T) {
	vfs := fstest.MapFS{
		"not-a-jdk": &fstest.MapFile{Mode: fs.ModeDir},
	}

	jdks, err := ScanLocationsForJDKs("", vfs, fakeRunner{}, []string{"."}, "testsource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs, got %d", len(jdks))
	}
}

func TestScanLocationsForJDKs_IgnoresMissingLocations(t *testing.T) {
	vfs := fstest.MapFS{}

	jdks, err := ScanLocationsForJDKs("", vfs, fakeRunner{}, []string{"definitely-not-there"}, "testsource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs, got %d", len(jdks))
	}
}

func TestValidateJDK(t *testing.T) {
	vfs := fstest.MapFS{}

	// Invalid path (no bin/java)
	jdk, ok, err := ValidateJDK(vfs, fakeRunner{}, "", ".", "test")
	if ok {
		t.Error("Should return false for invalid JDK path")
	}
	if err != nil {
		t.Error("Should not return error for invalid JDK path")
	}
	if jdk != (JDK{}) {
		t.Error("Should return empty JDK for invalid path")
	}

	// Test with valid JDK path
	jdkPath := createFakeJDK(t, vfs, ".", "openjdk-21")
	jdk, ok, err = ValidateJDK(vfs, fakeRunner{}, "", jdkPath, "test-source")
	if !ok {
		t.Error("Should return true for valid JDK path")
	}
	if err != nil {
		t.Error("Should not return error for valid JDK path")
	}
	if jdk.Path != jdkPath {
		t.Errorf("Path = %v, want %v", jdk.Path, jdkPath)
	}
	if jdk.Version != "21" {
		t.Errorf("Version = %v, want 21", jdk.Version)
	}
	if jdk.Vendor != "TestVendor" {
		t.Errorf("Vendor = %v, want TestVendor", jdk.Vendor)
	}
	if jdk.Architecture != "x64" {
		t.Errorf("Architecture = %v, want x64", jdk.Architecture)
	}
	if jdk.Source != "test-source" {
		t.Errorf("Source = %v, want test-source", jdk.Source)
	}
	// "TestVendor" -> "testvendor", "test-source", "21" -> "testvendor-test-source@21"
	expectedID := "testvendor-test-source@21"
	if jdk.Identifier != expectedID {
		t.Errorf("Identifier = %v, want %v", jdk.Identifier, expectedID)
	}
}

func TestValidateJDK_IdentifierGeneration(t *testing.T) {
	vfs := fstest.MapFS{}

	tests := []struct {
		name         string
		source       string
		vendor       string
		version      string
		path         string
		wantID       string
		runnerOutput string
	}{
		{
			name:    "Javm Source",
			source:  "javm",
			vendor:  "Eclipse Adoptium",
			version: "17.0.17",
			path:    "temurin@17.0.17",
			wantID:  "temurin@17.0.17",
		},
		{
			name:    "System Source - Red Hat",
			source:  "system",
			vendor:  "Red Hat, Inc.",
			version: "25.0.1",
			path:    "redhat-25",
			wantID:  "red-hat-inc-system@25",
		},
		{
			name:    "Gradle Source - Adoptium",
			source:  "gradle",
			vendor:  "Eclipse Adoptium",
			version: "11.0.15",
			path:    "gradle-jdk",
			wantID:  "eclipse-adoptium-gradle@11",
		},
		{
			name:    "System Source - Oracle Legacy",
			source:  "system",
			vendor:  "Oracle Corporation",
			version: "1.8.0_202",
			path:    "oracle-8",
			wantID:  "oracle-corporation-system@8",
		},
		{
			name:    "Empty Vendor",
			source:  "custom",
			vendor:  "", // fallback to source
			version: "21",
			path:    "custom-jdk",
			wantID:  "custom-custom@21",
			runnerOutput: `java.vendor=
java.version=21
os.arch=x64`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := createFakeJDKWithVendor(t, vfs, "jdks", tt.path, tt.version, tt.vendor)

			jdk, ok, err := ValidateJDK(vfs, fakeRunner{out: tt.runnerOutput}, "", p, tt.source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ok {
				t.Fatalf("jdk not valid")
			}

			if jdk.Identifier != tt.wantID {
				t.Errorf("got identifier %q, want %q", jdk.Identifier, tt.wantID)
			}
		})
	}
}

func TestExtractMetadataFromReleaseFile(t *testing.T) {
	vfs := fstest.MapFS{
		"non-existent-release": &fstest.MapFile{Mode: fs.ModeDir},
		"jdk/release": {
			Data: []byte(`JAVA_VERSION="17.0.2"
JAVA_VENDOR="Oracle Corporation"
IMPLEMENTOR="Oracle"
OS_ARCH="x64"`),
			Mode: fs.FileMode(0o644),
		},
		"invalid-jdk/release": {
			Data: []byte(`JAVA_VERSION 17.0.2
JAVA_VENDOR=Oracle Corporation
IMPLEMENTOR="Oracle"`),
			Mode: fs.FileMode(0o644),
		},
	}

	// Test case 1: Non-existent release file
	metadata, err := ExtractMetadataFromReleaseFile(vfs, "non-existent-release")
	if err == nil {
		t.Error("Should return error for non-existent release file")
	}
	if metadata != nil {
		t.Error("Should return nil metadata for non-existent release file")
	}

	// Test case 2: Valid release file
	metadata, err = ExtractMetadataFromReleaseFile(vfs, "jdk")
	if err != nil {
		t.Error("Should not return error for valid release file")
	}
	if metadata["JAVA_VERSION"] != "17.0.2" {
		t.Errorf("JAVA_VERSION = %v, want 17.0.2", metadata["JAVA_VERSION"])
	}
	if metadata["JAVA_VENDOR"] != "Oracle Corporation" {
		t.Errorf("JAVA_VENDOR = %v, want Oracle Corporation", metadata["JAVA_VENDOR"])
	}
	if metadata["IMPLEMENTOR"] != "Oracle" {
		t.Errorf("IMPLEMENTOR = %v, want Oracle", metadata["IMPLEMENTOR"])
	}
	if metadata["OS_ARCH"] != "x64" {
		t.Errorf("OS_ARCH = %v, want x64", metadata["OS_ARCH"])
	}

	// Test case 3: Invalid format in release file
	metadata, err = ExtractMetadataFromReleaseFile(vfs, "invalid-jdk")
	if err != nil {
		t.Error("Should not return error for invalid format in release file")
	}
	if metadata["JAVA_VERSION"] != "" {
		t.Error("JAVA_VERSION should be empty for invalid format")
	}
	if metadata["IMPLEMENTOR"] != "Oracle" {
		t.Errorf("IMPLEMENTOR = %v, want Oracle", metadata["IMPLEMENTOR"])
	}
}

func mkdir(t *testing.T, vfs fstest.MapFS, name string) string {
	t.Helper()
	vfs[name] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
	return name
}

func mkfile(t *testing.T, vfs fstest.MapFS, name, content string) {
	t.Helper()
	vfs[name] = &fstest.MapFile{
		Data: []byte(content),
		Mode: 0o644,
	}
}

func TestExtractMetadataFromJavaVersion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockJavaVersion := `java.vendor=OpenJDK
java.version=17.0.2
os.arch=x64`

	metadata, err := ExtractMetadataFromJavaVersion(fakeRunner{out: mockJavaVersion}, "java")
	if err != nil {
		t.Errorf("Should not return error for mock java executable: %v", err)
	}

	if got := metadata["version"]; got != "17.0.2" {
		t.Errorf("version = %v, want 17.0.2", got)
	}
	if got := metadata["vendor"]; got != "OpenJDK" {
		t.Errorf("vendor = %v, want OpenJDK", got)
	}
	if got := metadata["architecture"]; got != "x64" {
		t.Errorf("architecture = %v, want x64", got)
	}
}

func TestDeduplicateJDKs(t *testing.T) {
	jdk1 := JDK{
		Path:         "/path/to/jdk1",
		Version:      "17.0.2",
		Vendor:       "Oracle",
		Architecture: "x64",
		Source:       "test",
	}

	jdk2 := JDK{
		Path:         "/path/to/jdk2",
		Version:      "11.0.14",
		Vendor:       "OpenJDK",
		Architecture: "x64",
		Source:       "test",
	}

	jdk3 := JDK{
		Path:         "/path/to/jdk1", // Duplicate path
		Version:      "17.0.2",
		Vendor:       "Oracle",
		Architecture: "x64",
		Source:       "another-source", // Different source
	}

	// Test case 1: No duplicates
	jdks := []JDK{jdk1, jdk2}
	result := DeduplicateJDKs(jdks)
	if len(result) != 2 {
		t.Errorf("len = %v, want 2", len(result))
	}

	// Test case 2: With duplicates
	jdks = []JDK{jdk1, jdk2, jdk3}
	result = DeduplicateJDKs(jdks)
	if len(result) != 2 {
		t.Errorf("len = %v, want 2", len(result))
	}

	// Verify the first occurrence is kept
	found := false
	for _, jdk := range result {
		if jdk.Path == "/path/to/jdk1" {
			if jdk.Source != "test" {
				t.Errorf("Source = %v, want test", jdk.Source)
			}
			found = true
		}
	}
	if !found {
		t.Error("Should find the first occurrence of the duplicate JDK")
	}
}
