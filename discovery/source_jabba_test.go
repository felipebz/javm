package discovery

import (
	"path/filepath"
	"testing"
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
	tmpHome := t.TempDir()

	jabbaJdkDir := filepath.Join(tmpHome, ".jabba", "jdk")
	jdkPath := createFakeJDK(t, jabbaJdkDir, "openjdk-21")

	// Save and override HOME / USERPROFILE for test
	setEnvTemp(t, "HOME", tmpHome)
	setEnvTemp(t, "USERPROFILE", tmpHome)

	src := NewJabbaSource()

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
