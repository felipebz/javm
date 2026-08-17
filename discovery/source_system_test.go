package discovery

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
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

	jdks, err := src.Discover(context.Background())
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

func TestSystemSource_DiscoverRootsPreservesWindowsProgramFilesPath(t *testing.T) {
	programFiles := t.TempDir()
	relativeJDKPath := filepath.Join("Java", "jdk-21")
	javaPath := filepath.Join(programFiles, filepath.FromSlash(ExpectedJavaPath(filepath.ToSlash(relativeJDKPath), runtime.GOOS)))
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	releasePath := filepath.Join(programFiles, filepath.FromSlash(path.Join(ExpectedJDKDir(filepath.ToSlash(relativeJDKPath), runtime.GOOS), "release")))
	if err := os.WriteFile(releasePath, []byte("JAVA_VERSION=\"21\"\nJAVA_VENDOR=\"TestVendor\"\nOS_ARCH=\"x64\""), 0o644); err != nil {
		t.Fatal(err)
	}

	source := &SystemSource{runner: fakeRunner{}}
	jdks, err := source.discoverRoots(context.Background(), systemRoots("windows", programFiles, ""))
	if err != nil {
		t.Fatalf("discoverRoots() error = %v", err)
	}
	if len(jdks) != 1 {
		t.Fatalf("discoverRoots() returned %d JDKs, want 1", len(jdks))
	}
	wantPath := filepath.Join(programFiles, relativeJDKPath)
	if jdks[0].Path != wantPath {
		t.Fatalf("JDK.Path = %q, want %q", jdks[0].Path, wantPath)
	}
}
