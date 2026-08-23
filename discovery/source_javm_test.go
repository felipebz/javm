package discovery

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestJavmSource_Name(t *testing.T) {
	src := NewJavmSource()
	if src.Name() != "javm" {
		t.Errorf("expected name 'javm', got %q", src.Name())
	}
}

func TestJavmSource_Discover(t *testing.T) {
	vfs := fstest.MapFS{}

	setEnvTemp(t, "JAVM_HOME", ".")

	jdkPath := createFakeJDK(t, vfs, "jdk", "openjdk-21")

	src := &JavmSource{vfs: vfs}

	jdks, err := src.Discover(context.Background())
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
	vfs := fstest.MapFS{
		"jdk/": &fstest.MapFile{},
	}

	setEnvTemp(t, "JAVM_HOME", "jdk")

	src := &JavmSource{vfs: vfs}

	jdks, err := src.Discover(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs found, got %d", len(jdks))
	}
}

func TestJavmSource_Discover_DirectoryDoesNotExist(t *testing.T) {
	vfs := fstest.MapFS{}

	setEnvTemp(t, "JAVM_HOME", "does-not-exist")

	src := &JavmSource{vfs: vfs}

	jdks, err := src.Discover(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs found, got %d", len(jdks))
	}
}

func TestJavmSource_Discover_LinkedJDK(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}

	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	target := filepath.Join(home, "external-jdk")
	javaPath := filepath.FromSlash(ExpectedJavaPath(target, runtime.GOOS))
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	releasePath := filepath.Join(target, filepath.FromSlash(ExpectedJDKDir(".", runtime.GOOS)), "release")
	if err := os.WriteFile(releasePath, []byte("JAVA_VERSION=\"21.0.1\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\""), 0o644); err != nil {
		t.Fatal(err)
	}

	managedDir := filepath.Join(home, "jdk")
	if err := os.MkdirAll(managedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(managedDir, "system@21.0.1")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatal(err)
	}

	jdks, err := NewJavmSource().Discover(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 1 {
		t.Fatalf("expected 1 JDK found, got %d", len(jdks))
	}
	if jdks[0].Identifier != "system@21.0.1" {
		t.Errorf("expected identifier %q, got %q", "system@21.0.1", jdks[0].Identifier)
	}
	if jdks[0].Path != linkPath {
		t.Errorf("expected path %q, got %q", linkPath, jdks[0].Path)
	}
	if jdks[0].Source != "javm" {
		t.Errorf("expected source %q, got %q", "javm", jdks[0].Source)
	}
}
