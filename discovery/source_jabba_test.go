package discovery

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestJabbaSource_Enabled(t *testing.T) {
	src := NewJabbaSource()
	if !src.Enabled(&Config{
		Enabled: true,
		Sources: map[string]bool{
			"jabba": true,
		},
	}) {
		t.Errorf("expected Enabled to return true")
	}
}

func TestJabbaSource_getLocations_RealHome(t *testing.T) {
	tmpHome := t.TempDir()

	jabbaJdkDir := filepath.Join(tmpHome, ".jabba", "jdk")
	jdkPath := createFakeJDK(t, jabbaJdkDir, "openjdk-21")

	// Save and override HOME / USERPROFILE for test
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Setenv("USERPROFILE", oldUserProfile)
	})
	if runtime.GOOS == "windows" {
		_ = os.Setenv("USERPROFILE", tmpHome)
	} else {
		_ = os.Setenv("HOME", tmpHome)
	}

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
