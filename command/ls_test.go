package command

import (
	"bytes"
	"io/fs"
	"testing"
	"time"

	"github.com/felipebz/javm/semver"
)

type fakeDirEntry struct {
	name    string
	isDir   bool
	symlink bool
}

func (f fakeDirEntry) Name() string { return f.name }
func (f fakeDirEntry) IsDir() bool  { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode {
	if f.symlink {
		return fs.ModeSymlink
	}
	if f.isDir {
		return fs.ModeDir
	}
	return 0
}
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo{mode: f.Type()}, nil }

type fakeFileInfo struct{ mode fs.FileMode }

func (f fakeFileInfo) Name() string       { return "" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

func TestLs_ParsesAndSortsSameQualifier(t *testing.T) {
	orig := readDir
	defer func() { readDir = orig }()
	readDir = func(path string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "temurin@1.8.0", isDir: true},
			fakeDirEntry{name: "temurin@17.0.1", isDir: true},
		}, nil
	}

	vs, err := Ls()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(vs) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(vs))
	}
	if got, want := vs[0].String(), "temurin@17.0.1"; got != want {
		t.Errorf("vs[0]=%q want %q", got, want)
	}
	if got, want := vs[1].String(), "temurin@1.8.0"; got != want {
		t.Errorf("vs[1]=%q want %q", got, want)
	}
}

func TestLs_IncludesSystemSymlink(t *testing.T) {
	orig := readDir
	defer func() { readDir = orig }()
	readDir = func(path string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "system@21.0.0", symlink: true},
		}, nil
	}
	vs, err := Ls()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].String() != "system@21.0.0" {
		t.Fatalf("expected [system@21.0.0], got %#v", versionsToStrings(vs))
	}
}

func TestLsBestMatchWithVersionSlice(t *testing.T) {
	mk := func(s string) *semver.Version { v, _ := semver.ParseVersion(s); return v }
	vs := []*semver.Version{
		mk("temurin@1.9.0"),
		mk("temurin@1.8.73"),
		mk("temurin@1.8.0"),
	}
	ver, err := LsBestMatchWithVersionSlice(vs, "~1.8.70")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "temurin@1.8.73" {
		t.Errorf("got %q want %q", ver, "temurin@1.8.73")
	}

	// invalid selector
	if _, err := LsBestMatchWithVersionSlice(vs, "not-a-range"); err == nil {
		t.Errorf("expected error for invalid range")
	}
	// no match
	if _, err := LsBestMatchWithVersionSlice(vs, ">=25"); err == nil {
		t.Errorf("expected error for no match")
	}
}

func TestNewLsCommand_PrintsAllAndFiltersAndLatest(t *testing.T) {
	orig := readDir
	defer func() { readDir = orig }()
	readDir = func(path string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "temurin@1.9.0", isDir: true},
			fakeDirEntry{name: "temurin@1.8.73", isDir: true},
			fakeDirEntry{name: "temurin@1.8.0", isDir: true},
		}, nil
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	// no args: print all in descending order
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := "temurin@1.9.0\n" +
		"temurin@1.8.73\n" +
		"temurin@1.8.0\n"
	if got != want {
		t.Errorf("all got:\n%q\nwant:\n%q", got, want)
	}

	// with range: only 1.8.x
	out.Reset()
	cmd.SetArgs([]string{"~1.8.0"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = out.String()
	want = "temurin@1.8.73\n" +
		"temurin@1.8.0\n"
	if got != want {
		t.Errorf("range got:\n%q\nwant:\n%q", got, want)
	}

	// with --latest=minor: keep latest per minor -> 1.8.73 and 1.9.0
	out.Reset()
	cmd.SetArgs([]string{"--latest=minor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = out.String()
	// TrimTo returns ascending order (1.8 then 1.9)
	want = "temurin@1.8.73\n" +
		"temurin@1.9.0\n"
	if got != want {
		t.Errorf("latest got:\n%q\nwant:\n%q", got, want)
	}
}

func versionsToStrings(vs []*semver.Version) []string {
	res := make([]string, len(vs))
	for i, v := range vs {
		res[i] = v.String()
	}
	return res
}
