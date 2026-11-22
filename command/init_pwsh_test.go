package command

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// isolatePowerShellInit points JAVM_HOME and the system temp directory at
// fresh directories and returns JAVM_HOME and the temp directory.
func isolatePowerShellInit(t *testing.T) (string, string) {
	t.Helper()
	home, temp := t.TempDir(), t.TempDir()
	t.Setenv("JAVM_HOME", home)
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	return home, temp
}

func initScripts(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "javm-init-*.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func writeInitScript(t *testing.T, script string) string {
	t.Helper()
	path, err := realWritePowerShellInitScript(script)
	if err != nil {
		t.Fatalf("realWritePowerShellInitScript() error = %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != script {
		t.Fatalf("%s = %q, %v, want %q", path, data, err, script)
	}
	return path
}

func TestPowerShellInitScriptIsReusedFromJavmHome(t *testing.T) {
	home, temp := isolatePowerShellInit(t)
	dir := filepath.Join(home, "init")

	first := writeInitScript(t, "# first\n")
	if filepath.Dir(first) != dir {
		t.Fatalf("script path = %q, want it in %q", first, dir)
	}
	if again := writeInitScript(t, "# first\n"); again != first {
		t.Fatalf("second call path = %q, want %q", again, first)
	}
	if got := initScripts(t, dir); len(got) != 1 {
		t.Fatalf("init scripts = %v, want one", got)
	}
	if got := initScripts(t, temp); len(got) != 0 {
		t.Fatalf("temp directory scripts = %v, want none", got)
	}
	if runtime.GOOS != "windows" {
		for path, want := range map[string]os.FileMode{dir: 0o700, first: 0o600} {
			if info, err := os.Stat(path); err != nil || info.Mode().Perm() != want {
				t.Fatalf("%s mode = %v, %v, want %v", path, info.Mode().Perm(), err, want)
			}
		}
	}
}

func TestPowerShellInitScriptReplacesPreviousScript(t *testing.T) {
	home, _ := isolatePowerShellInit(t)
	dir := filepath.Join(home, "init")
	notes := filepath.Join(dir, "notes.txt")

	first := writeInitScript(t, "# first\n")
	if err := os.WriteFile(notes, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := writeInitScript(t, "# second\n")
	if second == first {
		t.Fatalf("a different script reused %q", first)
	}
	if got := initScripts(t, dir); len(got) != 1 || got[0] != second {
		t.Fatalf("init scripts = %v, want only %q", got, second)
	}
	if _, err := os.Stat(notes); err != nil {
		t.Fatalf("unrelated file was removed: %v", err)
	}
}

func TestPowerShellInitScriptConcurrentWriters(t *testing.T) {
	home, _ := isolatePowerShellInit(t)
	script := strings.Repeat("# concurrent shells\n", 4096)

	paths := make([]string, 8)
	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i := range paths {
		wg.Go(func() { paths[i], errs[i] = realWritePowerShellInitScript(script) })
	}
	wg.Wait()
	for i := range paths {
		if errs[i] != nil || paths[i] != paths[0] {
			t.Fatalf("writer %d = %q, %v, want %q", i, paths[i], errs[i], paths[0])
		}
	}
	if data, err := os.ReadFile(paths[0]); err != nil || string(data) != script {
		t.Fatalf("script content is incomplete: %d bytes, %v", len(data), err)
	}
	if got := initScripts(t, filepath.Join(home, "init")); len(got) != 1 {
		t.Fatalf("init scripts = %v, want one", got)
	}
}

func TestInitPowerShellFailsWhenJavmHomeIsReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions enforced for the current user")
	}
	home, _ := isolatePowerShellInit(t)
	if err := os.Chmod(home, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(home, 0o700) })
	previous := writePowerShellInitScript
	writePowerShellInitScript = realWritePowerShellInitScript
	t.Cleanup(func() { writePowerShellInitScript = previous })

	cmd := NewInitCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"pwsh"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), filepath.Join(home, "init")) {
		t.Fatalf("init pwsh error = %v, want an error naming %s", err, filepath.Join(home, "init"))
	}
	if out.Len() != 0 {
		t.Fatalf("init pwsh printed %q, want nothing", out.String())
	}
}
