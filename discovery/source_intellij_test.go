package discovery

import (
	"path"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestIntelliJSource_Name(t *testing.T) {
	src := NewIntelliJSource()
	if src.Name() != "intellij" {
		t.Errorf("expected name 'intellij', got %q", src.Name())
	}
}

func TestIntelliJSource_Discover(t *testing.T) {
	vfs := fstest.MapFS{}

	var base string
	if runtime.GOOS == "darwin" {
		base = path.Join("Library", "Java", "JavaVirtualMachines")
	} else {
		base = ".jdks"
	}
	jdkPath := createFakeJDK(t, vfs, base, "idea-jdk-21")

	src := &IntelliJSource{vfs: vfs}

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
	if jdks[0].Source != "intellij" {
		t.Errorf("expected source 'intellij', got %q", jdks[0].Source)
	}
}
