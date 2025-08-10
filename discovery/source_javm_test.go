package discovery

import (
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
	vfs := fstest.MapFS{
		"jdk/": &fstest.MapFile{},
	}

	setEnvTemp(t, "JAVM_HOME", "jdk")

	src := &JavmSource{vfs: vfs}

	jdks, err := src.Discover()
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

	jdks, err := src.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 0 {
		t.Fatalf("expected 0 JDKs found, got %d", len(jdks))
	}
}
