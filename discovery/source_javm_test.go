package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavmSource_Name(t *testing.T) {
	src := NewJavmSource()
	if src.Name() != "javm" {
		t.Errorf("expected name 'javm', got %q", src.Name())
	}
}

func TestJavmSource_Discover(t *testing.T) {
	tmpDir := t.TempDir()

	setEnvTemp(t, "JAVM_HOME", tmpDir)

	jdksDir := filepath.Join(tmpDir, "jdk")
	jdkPath := createFakeJDK(t, jdksDir, "openjdk-21")

	src := NewJavmSource()

	jdks, err := src.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 1 {
		t.Fatalf("expected 1 JDK found, got %d", len(jdks))
	}
	if jdks[0].Path != jdkPath {
		t.Errorf("expected path %q, got %q", jdkPath, jdks[0].Path)
	}
	if jdks[0].Source != "javm" {
		t.Errorf("expected source 'javm', got %q", jdks[0].Source)
	}
}

func TestJavmSource_Discover_NoJDKs(t *testing.T) {
	tmpDir := t.TempDir()

	setEnvTemp(t, "JAVM_HOME", tmpDir)

	jdksDir := filepath.Join(tmpDir, "jdk")
	if err := os.MkdirAll(jdksDir, 0755); err != nil {
		t.Fatalf("failed to create jdk dir: %v", err)
	}

	// Create the source and test discovery
	src := NewJavmSource()

	jdks, err := src.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs found, got %d", len(jdks))
	}
}

func TestJavmSource_Discover_DirectoryDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	setEnvTemp(t, "JAVM_HOME", tmpDir)

	src := NewJavmSource()

	jdks, err := src.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs found, got %d", len(jdks))
	}
}
