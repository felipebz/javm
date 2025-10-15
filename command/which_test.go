package command

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type wtFakeDirEntry struct {
	name    string
	isDir   bool
	symlink bool
}

func (f wtFakeDirEntry) Name() string { return f.name }
func (f wtFakeDirEntry) IsDir() bool  { return f.isDir }
func (f wtFakeDirEntry) Type() fs.FileMode {
	if f.symlink {
		return fs.ModeSymlink
	}
	if f.isDir {
		return fs.ModeDir
	}
	return 0
}
func (f wtFakeDirEntry) Info() (fs.FileInfo, error) { return wtFakeFileInfo{mode: f.Type()}, nil }

type wtFakeFileInfo struct{ mode fs.FileMode }

func (f wtFakeFileInfo) Name() string       { return "" }
func (f wtFakeFileInfo) Size() int64        { return 0 }
func (f wtFakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f wtFakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f wtFakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f wtFakeFileInfo) Sys() any           { return nil }

func TestNewWhichCommand_WithArg(t *testing.T) {
	orig := readDir
	defer func() { readDir = orig }()
	readDir = func(path string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			wtFakeDirEntry{name: "temurin@17.0.1", isDir: true},
		}, nil
	}

	// ensure predictable base dir
	tmp := t.TempDir()
	oldHome := os.Getenv("JAVM_HOME")
	defer os.Setenv("JAVM_HOME", oldHome)
	os.Setenv("JAVM_HOME", tmp)

	cmd := NewWhichCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"temurin@>=17"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	want := filepath.Join(tmp, "jdk", "temurin@17.0.1") + "\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestNewWhichCommand_ReadsJavaVersion(t *testing.T) {
	orig := readDir
	defer func() { readDir = orig }()
	readDir = func(path string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			wtFakeDirEntry{name: "temurin@21.0.1", isDir: true},
		}, nil
	}

	// create a temp workspace with .java-version
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".java-version"), []byte("temurin@>=21\n"), 0666); err != nil {
		t.Fatalf("write .java-version: %v", err)
	}
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(workspace)

	// ensure predictable base dir
	tmp := t.TempDir()
	oldHome := os.Getenv("JAVM_HOME")
	defer os.Setenv("JAVM_HOME", oldHome)
	os.Setenv("JAVM_HOME", tmp)

	cmd := NewWhichCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := filepath.Join(tmp, "jdk", "temurin@21.0.1") + "\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}
