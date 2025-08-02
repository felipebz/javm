package discovery

import (
	"testing"
)

func TestSystemSource_Name(t *testing.T) {
	src := NewSystemSource()
	if src.Name() != "system" {
		t.Errorf("expected name 'system', got %q", src.Name())
	}
}

func TestSystemSource_Enabled(t *testing.T) {
	src := NewSystemSource()
	if !src.Enabled(&Config{
		Enabled: true,
		Sources: map[string]bool{
			"system": true,
		},
	}) {
		t.Errorf("expected Enabled to return true")
	}
}

func TestSystemSource_Discover_FindsJDK(t *testing.T) {
	tmpDir := t.TempDir()
	jdkPath := createFakeJDK(t, tmpDir, "jdk-21")

	src := &SystemSource{
		locations: []string{tmpDir},
	}

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
}
