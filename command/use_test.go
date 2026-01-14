package command

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
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
func (f FileInfoMock) Sys() any           { return nil }

func TestUse(t *testing.T) {
	prevPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)

	var suffix string
	if runtime.GOOS == "darwin" {
		suffix = "/Contents/Home"
	}
	javaHome := filepath.Join(cfg.Dir(), "jdk", "1.7.2", suffix)
	mockJdkPath := filepath.Join(cfg.Dir(), "jdk", "1.7.2")

	javaPath := filepath.Join(javaHome, "bin")

	defer func() { os.Setenv("PATH", prevPath) }()

	cleanup := setupMockLs()
	defer cleanup()
	mockLsResult = []discovery.JDK{
		{Identifier: "1.6.0", Version: "1.6.0", Source: "javm", Path: filepath.Join(cfg.Dir(), "jdk", "1.6.0")},
		{Identifier: "1.7.0", Version: "1.7.0", Source: "javm", Path: filepath.Join(cfg.Dir(), "jdk", "1.7.0")},
		{Identifier: "1.7.2", Version: "1.7.2", Source: "javm", Path: mockJdkPath},
		{Identifier: "1.8.0", Version: "1.8.0", Source: "javm", Path: filepath.Join(cfg.Dir(), "jdk", "1.8.0")},
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
		"SET\tJAVA_HOME_BEFORE_JAVM\t" + "/system-jdk",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
