package command

import (
	"github.com/felipebz/javm/cfg"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type DirEntryMock string

func (d DirEntryMock) Name() string      { return string(d) }
func (d DirEntryMock) IsDir() bool       { return true }
func (d DirEntryMock) Type() os.FileMode { return os.FileMode(0) }
func (d DirEntryMock) Info() (os.FileInfo, error) {
	return FileInfoMock(d), nil
}

type FileInfoMock string

func (f FileInfoMock) Name() string       { return string(f) }
func (f FileInfoMock) Size() int64        { return 0 }
func (f FileInfoMock) Mode() os.FileMode  { return os.FileMode(0) }
func (f FileInfoMock) ModTime() time.Time { return time.Time{} }
func (f FileInfoMock) IsDir() bool        { return true }
func (f FileInfoMock) Sys() interface{}   { return nil }

func TestUse(t *testing.T) {
	prevPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)

	var suffix string
	if runtime.GOOS == "darwin" {
		suffix = "/Contents/Home"
	}
	javaHome := filepath.Join(cfg.Dir(), "jdk", "1.7.2", suffix)
	javaPath := filepath.Join(javaHome, "bin")

	defer func() { os.Setenv("PATH", prevPath) }()
	var prevReadDir = readDir
	defer func() { readDir = prevReadDir }()
	readDir = func(dirname string) ([]os.DirEntry, error) {
		return []os.DirEntry{
			DirEntryMock("1.6.0"), DirEntryMock("1.7.0"), DirEntryMock("1.7.2"), DirEntryMock("1.8.0"),
		}, nil
	}
	os.Setenv("PATH", "/usr/local/bin"+sep+filepath.Join(cfg.Dir(), "jdk", "1.6.0", "bin")+sep+"/usr/bin")
	os.Setenv("JAVA_HOME", "/system-jdk")
	actual, err := Use("1.7")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := []string{
		"SET\tPATH\t" + javaPath + sep + "/usr/local/bin" + sep + "/usr/bin",
		"SET\tJAVA_HOME\t" + javaHome,
		"SET\tJAVA_HOME_BEFORE_JABBA\t" + "/system-jdk",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
