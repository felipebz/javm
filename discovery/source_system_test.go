package discovery

import (
	"testing"
	"testing/fstest"
)

func TestSystemSource_Name(t *testing.T) {
	src := NewSystemSource()
	if src.Name() != "system" {
		t.Errorf("expected name 'system', got %q", src.Name())
	}
}

func TestSystemSource_Discover_FindsJDK(t *testing.T) {
	vfs := fstest.MapFS{}
	fakeJDK := createFakeJDK(t, vfs, ".", "jdk-21")

	src := &SystemSource{
		vfs:       vfs,
		locations: []string{fakeJDK},
	}

	jdks, err := src.Discover()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jdks) != 1 {
		t.Fatalf("expected 1 JDK found, got %d", len(jdks))
	}
	if jdks[0].Path != fakeJDK {
		t.Errorf("expected path %q, got %q", fakeJDK, jdks[0].Path)
	}
}
