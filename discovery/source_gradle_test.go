package discovery

import (
	"path"
	"testing"
	"testing/fstest"
)

func TestGradleSource_Name(t *testing.T) {
	src := NewGradleSource()
	if src.Name() != "gradle" {
		t.Errorf("expected name 'gradle', got %q", src.Name())
	}
}

func TestGradleSource_Discover_DefaultGradleHome(t *testing.T) {
	vfs := fstest.MapFS{}

	// Ensure code path for default (~/.gradle)
	setEnvTemp(t, "GRADLE_USER_HOME", "")

	jdkPath := createFakeJDK(t, vfs, path.Join(".gradle", "jdks"), "openjdk-21")

	src := &GradleSource{vfs: vfs}

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
	if jdks[0].Source != "gradle" {
		t.Errorf("expected source 'gradle', got %q", jdks[0].Source)
	}
}

func TestGradleSource_Discover_WithEnvOverride(t *testing.T) {
	vfs := fstest.MapFS{}

	// When GRADLE_USER_HOME is set, we expect the source to look under 'jdks' at the vfs root
	setEnvTemp(t, "GRADLE_USER_HOME", "some-gradle-home")

	jdkPath := createFakeJDK(t, vfs, "jdks", "openjdk-21")

	src := &GradleSource{vfs: vfs}

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
	if jdks[0].Source != "gradle" {
		t.Errorf("expected source 'gradle', got %q", jdks[0].Source)
	}
}
