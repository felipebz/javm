package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return []byte(f.out), f.err
}

func createFakeJDK(t *testing.T, vfs fstest.MapFS, baseDir, name string) string {
	return createFakeJDKWithVendor(t, vfs, baseDir, name, "21", "TestVendor")
}

func createFakeJDKWithVendor(t *testing.T, vfs fstest.MapFS, baseDir, name string, version string, vendor string) string {
	t.Helper()

	var osSpecificSubDir = ""
	if runtime.GOOS == "darwin" {
		osSpecificSubDir = path.Join("Contents", "Home")
	}
	jdkDir := path.Join(baseDir, name, osSpecificSubDir)
	binDir := path.Join(jdkDir, "bin")
	java := "java"
	if runtime.GOOS == "windows" {
		java = "java.exe"
	}

	vfs[path.Join(binDir, java)] = &fstest.MapFile{
		Data: []byte(""),
		Mode: fs.FileMode(0o755),
	}

	release := fmt.Sprintf("JAVA_VERSION=\"%s\"\nJAVA_VENDOR=\"%s\"\nOS_ARCH=\"x64\"", version, vendor)
	vfs[path.Join(jdkDir, "release")] = &fstest.MapFile{
		Data: []byte(release),
		Mode: fs.FileMode(0o644),
	}

	vfs[jdkDir] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
	vfs[binDir] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}

	return filepath.Join(baseDir, name)
}

func setEnvTemp(t *testing.T, key, value string) {
	t.Helper()
	oldVal, hadOld := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if !hadOld {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, oldVal)
		}
	})
}
