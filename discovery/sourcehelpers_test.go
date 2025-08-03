package discovery

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func createFakeJDK(t *testing.T, baseDir, name string) string {
	t.Helper()
	jdkPath := filepath.Join(baseDir, name)
	binDir := filepath.Join(jdkPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	javaExec := "java"
	if runtime.GOOS == "windows" {
		javaExec = "java.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, javaExec), []byte(""), 0755); err != nil {
		t.Fatalf("failed to create java executable: %v", err)
	}
	release := `JAVA_VERSION="21"
JAVA_VENDOR="TestVendor"
OS_ARCH="x64"
IMPLEMENTOR="JDK"`
	if err := os.WriteFile(filepath.Join(jdkPath, "release"), []byte(release), 0644); err != nil {
		t.Fatalf("failed to create release file: %v", err)
	}
	return jdkPath
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
