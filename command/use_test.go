package command

import (
	"github.com/felipebz/javm/cfg"
	"os"
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
	defer func() { os.Setenv("PATH", prevPath) }()
	var prevReadDir = readDir
	defer func() { readDir = prevReadDir }()
	readDir = func(dirname string) ([]os.DirEntry, error) {
		return []os.DirEntry{
			DirEntryMock("1.6.0"), DirEntryMock("1.7.0"), DirEntryMock("1.7.2"), DirEntryMock("1.8.0"),
		}, nil
	}
	os.Setenv("PATH", "/usr/local/bin:"+cfg.Dir()+"/jdk/1.6.0/bin:/usr/bin")
	os.Setenv("JAVA_HOME", "/system-jdk")
	actual, err := Use("1.7")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var suffix string
	if runtime.GOOS == "darwin" {
		suffix = "/Contents/Home"
	}
	expected := []string{
		"export PATH=\"" + cfg.Dir() + "/jdk/1.7.2" + suffix + "/bin:/usr/local/bin:/usr/bin\"",
		"export JAVA_HOME=\"" + cfg.Dir() + "/jdk/1.7.2" + suffix + "\"",
		"export JAVA_HOME_BEFORE_JABBA=\"/system-jdk\"",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
