package discovery

import (
	"path"
	"testing"
	"testing/fstest"
)

type fakeConfig struct{}

func (f *fakeConfig) IsSourceEnabled(name string) bool { return true }

func TestJabbaSource_Name(t *testing.T) {
	src := NewJabbaSource()
	if src.Name() != "jabba" {
		t.Errorf("expected name 'jabba', got %q", src.Name())
	}
}

func TestJabbaSource_getLocations_RealHome(t *testing.T) {
	vfs := fstest.MapFS{}

	jabbaJdkDir := path.Join(".jabba", "jdk")
	jdkPath := createFakeJDK(t, vfs, jabbaJdkDir, "openjdk-21")

	src := &JabbaSource{vfs: vfs}

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
	if jdks[0].Source != "jabba" {
		t.Errorf("expected source 'jabba', got %q", jdks[0].Source)
	}
}
