package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

// The real /usr/bin/java with JAVA_HOME pointing at a fake JDK, so the result
// does not depend on the JDKs installed on the machine.
func TestCurrentWithRealMacOSLauncherAndJavaHome(t *testing.T) {
	if info, err := os.Lstat("/usr/bin/java"); err != nil || !info.Mode().IsRegular() {
		t.Skip("/usr/bin/java is not the macOS launcher stub")
	}
	isolateCurrentEnvironment(t)
	lookPath = func(string) (string, error) { return "/usr/bin/java", nil }
	root := filepath.Join(t.TempDir(), "zulu@21.0.4")
	t.Setenv("JAVA_HOME", makeFakeJDK(t, root, true))
	mockLsResult = []discovery.JDK{{Identifier: "zulu@21.0.4", Source: "javm", Path: root}}

	out, err := runCurrent(t)
	if err != nil || strings.TrimSpace(out) != "zulu@21.0.4" {
		t.Fatalf("current = %q, %v; want zulu@21.0.4", out, err)
	}
}
