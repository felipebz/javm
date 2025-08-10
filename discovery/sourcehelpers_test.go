package discovery

import (
	"io/fs"
	"os"
	"path"
	"runtime"
	"testing"
	"testing/fstest"
)

func createFakeJDK(t *testing.T, vfs fstest.MapFS, baseDir, name string) string {
	t.Helper()

	jdkDir := path.Join(baseDir, name)
	binDir := path.Join(jdkDir, "bin")
	java := "java"
	if runtime.GOOS == "windows" {
		java = "java.exe"
	}

	vfs[path.Join(binDir, java)] = &fstest.MapFile{
		Data: []byte(""),
		Mode: fs.FileMode(0o755),
	}

	release := []byte(
		`JAVA_VERSION="21"
JAVA_VENDOR="TestVendor"
OS_ARCH="x64"
IMPLEMENTOR="JDK"`,
	)
	vfs[path.Join(jdkDir, "release")] = &fstest.MapFile{
		Data: release,
		Mode: fs.FileMode(0o644),
	}

	vfs[jdkDir] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
	vfs[binDir] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}

	return jdkDir
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
