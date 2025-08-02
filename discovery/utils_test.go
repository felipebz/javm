package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScanLocationsForJDKs_FindsValidJDK(t *testing.T) {
	tmpDir := t.TempDir()
	fakeJDK := createFakeJDK(t, tmpDir, "jdk-21")

	jdks, err := ScanLocationsForJDKs([]string{tmpDir}, "testsource")
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
	tmpDir := t.TempDir()

	// Create a directory without bin/java or release file
	nonJDK := filepath.Join(tmpDir, "not-a-jdk")
	if err := os.MkdirAll(nonJDK, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	jdks, err := ScanLocationsForJDKs([]string{tmpDir}, "testsource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs, got %d", len(jdks))
	}
}

func TestScanLocationsForJDKs_IgnoresMissingLocations(t *testing.T) {
	missingDir := filepath.Join(os.TempDir(), "definitely-not-there")

	jdks, err := ScanLocationsForJDKs([]string{missingDir}, "testsource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs, got %d", len(jdks))
	}
}

func TestValidateJDK(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: Invalid path (no bin/java)
	jdk, ok, err := ValidateJDK(tempDir, "test")
	if ok {
		t.Error("Should return false for invalid JDK path")
	}
	if err != nil {
		t.Error("Should not return error for invalid JDK path")
	}
	if jdk != (JDK{}) {
		t.Error("Should return empty JDK for invalid path")
	}

	// Test case 2: Create a mock JDK structure
	binDir := filepath.Join(tempDir, "bin")
	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	javaExe := "java"
	if runtime.GOOS == "windows" {
		javaExe = "java.exe"
	}
	javaPath := filepath.Join(binDir, javaExe)
	err = os.WriteFile(javaPath, []byte("mock java executable"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock java executable: %v", err)
	}

	// Create a mock release file
	releaseContent := `JAVA_VERSION="17.0.2"
JAVA_VENDOR="Oracle Corporation"
IMPLEMENTOR="Oracle"
OS_ARCH="x64"
`
	err = os.WriteFile(filepath.Join(tempDir, "release"), []byte(releaseContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock release file: %v", err)
	}

	// Test with valid JDK path
	jdk, ok, err = ValidateJDK(tempDir, "test-source")
	if !ok {
		t.Error("Should return true for valid JDK path")
	}
	if err != nil {
		t.Error("Should not return error for valid JDK path")
	}
	if jdk.Path != tempDir {
		t.Errorf("Path = %v, want %v", jdk.Path, tempDir)
	}
	if jdk.Version != "17.0.2" {
		t.Errorf("Version = %v, want 17.0.2", jdk.Version)
	}
	if jdk.Vendor != "Oracle Corporation" {
		t.Errorf("Vendor = %v, want Oracle Corporation", jdk.Vendor)
	}
	if jdk.Implementation != "Oracle" {
		t.Errorf("Implementation = %v, want Oracle", jdk.Implementation)
	}
	if jdk.Architecture != "x64" {
		t.Errorf("Architecture = %v, want x64", jdk.Architecture)
	}
	if jdk.Source != "test-source" {
		t.Errorf("Source = %v, want test-source", jdk.Source)
	}
}

func TestExtractMetadataFromReleaseFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: Non-existent release file
	metadata, err := ExtractMetadataFromReleaseFile(tempDir)
	if err == nil {
		t.Error("Should return error for non-existent release file")
	}
	if metadata != nil {
		t.Error("Should return nil metadata for non-existent release file")
	}

	// Test case 2: Valid release file
	releaseContent := `JAVA_VERSION="17.0.2"
JAVA_VENDOR="Oracle Corporation"
IMPLEMENTOR="Oracle"
OS_ARCH="x64"
`
	err = os.WriteFile(filepath.Join(tempDir, "release"), []byte(releaseContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock release file: %v", err)
	}

	metadata, err = ExtractMetadataFromReleaseFile(tempDir)
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
	invalidContent := `JAVA_VERSION 17.0.2
JAVA_VENDOR=Oracle Corporation
IMPLEMENTOR="Oracle"
`
	err = os.WriteFile(filepath.Join(tempDir, "release"), []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create mock release file: %v", err)
	}

	metadata, err = ExtractMetadataFromReleaseFile(tempDir)
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

func TestExtractMetadataFromJavaVersion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "javm-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if runtime.GOOS == "windows" {
		// On Windows, we can't easily create executable files for testing
		// So we'll create a batch file that outputs the expected java -version output
		mockJavaContent := `@echo off
echo java version "17.0.2" 2>&1
echo OpenJDK Runtime Environment (build 17.0.2+8) 2>&1
echo OpenJDK 64-Bit Server VM (build 17.0.2+8, mixed mode) 2>&1
`
		mockJavaPath := filepath.Join(tempDir, "java.bat")
		err = os.WriteFile(mockJavaPath, []byte(mockJavaContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create mock java batch file: %v", err)
		}

		metadata, err := ExtractMetadataFromJavaVersion(mockJavaPath)
		if err != nil {
			t.Errorf("Should not return error for mock java executable: %v", err)
		}

		if got := metadata["version"]; got != "17.0.2" {
			t.Errorf("version = %v, want 17.0.2", got)
		}
		if got := metadata["vendor"]; got != "OpenJDK" {
			t.Errorf("vendor = %v, want OpenJDK", got)
		}
		if got := metadata["implementation"]; got != "JDK" {
			t.Errorf("implementation = %v, want JDK", got)
		}
		if got := metadata["architecture"]; got != "x64" {
			t.Errorf("architecture = %v, want x64", got)
		}
	} else {
		// On Unix-like systems, we can create a shell script
		mockJavaContent := `#!/bin/sh
echo 'java version "17.0.2"' >&2
echo 'OpenJDK Runtime Environment (build 17.0.2+8)' >&2
echo 'OpenJDK 64-Bit Server VM (build 17.0.2+8, mixed mode)' >&2
`
		mockJavaPath := filepath.Join(tempDir, "java")
		err = os.WriteFile(mockJavaPath, []byte(mockJavaContent), 0755)
		if err != nil {
			t.Fatalf("Failed to create mock java script: %v", err)
		}

		metadata, err := ExtractMetadataFromJavaVersion(mockJavaPath)
		if err != nil {
			t.Errorf("Should not return error for mock java executable: %v", err)
		}
		if got := metadata["version"]; got != "17.0.2" {
			t.Errorf("version = %v, want 17.0.2", got)
		}
		if got := metadata["vendor"]; got != "OpenJDK" {
			t.Errorf("vendor = %v, want OpenJDK", got)
		}
		if got := metadata["implementation"]; got != "JDK" {
			t.Errorf("implementation = %v, want JDK", got)
		}
		if got := metadata["architecture"]; got != "x64" {
			t.Errorf("architecture = %v, want x64", got)
		}
	}
}

func TestDeduplicateJDKs(t *testing.T) {
	jdk1 := JDK{
		Path:           "/path/to/jdk1",
		Version:        "17.0.2",
		Vendor:         "Oracle",
		Implementation: "JDK",
		Architecture:   "x64",
		Source:         "test",
	}

	jdk2 := JDK{
		Path:           "/path/to/jdk2",
		Version:        "11.0.14",
		Vendor:         "OpenJDK",
		Implementation: "JDK",
		Architecture:   "x64",
		Source:         "test",
	}

	jdk3 := JDK{
		Path:           "/path/to/jdk1", // Duplicate path
		Version:        "17.0.2",
		Vendor:         "Oracle",
		Implementation: "JDK",
		Architecture:   "x64",
		Source:         "another-source", // Different source
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
