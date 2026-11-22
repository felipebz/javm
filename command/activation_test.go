package command

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// applyShellRecords applies SET/UNSET records to env the way the shell
// wrappers do. dropEmpty simulates Windows shells, where setting a variable to
// an empty string removes it.
func applyShellRecords(t *testing.T, env, records []string, dropEmpty bool) []string {
	t.Helper()
	for _, record := range records {
		fields := strings.SplitN(record, "\t", 3)
		switch {
		case fields[0] == "SET" && len(fields) == 3:
			env = setEnvironmentValue(env, fields[1], fields[2])
		case fields[0] == "UNSET" && len(fields) == 2:
			env = removeEnvironmentValue(env, fields[1])
		default:
			t.Fatalf("malformed record %q", record)
		}
	}
	if dropEmpty {
		env = slices.DeleteFunc(env, func(entry string) bool { return strings.HasSuffix(entry, "=") })
	}
	return env
}

func removeEnvironmentValue(env []string, name string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(entry string) bool {
		key, _, ok := strings.Cut(entry, "=")
		return ok && environmentKeyEqual(key, name)
	})
}

func joinPath(entries ...string) string {
	return strings.Join(entries, string(filepath.ListSeparator))
}

func javaHomeForTest(jdk string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(jdk, "Contents", "Home")
	}
	return jdk
}

func binForTest(jdk string) string {
	return filepath.Join(javaHomeForTest(jdk), "bin")
}

func useForTest(t *testing.T, env []string, jdk string, dropEmpty bool) []string {
	t.Helper()
	records, err := useRecords(jdk, env)
	if err != nil {
		t.Fatalf("useRecords(%q) error = %v", jdk, err)
	}
	return applyShellRecords(t, env, records, dropEmpty)
}

func assertEnvironmentValue(t *testing.T, env []string, name, want string, wantSet bool) {
	t.Helper()
	got, ok := lookupEnvironmentValue(env, name)
	if ok != wantSet || got != want {
		t.Fatalf("%s = %q (set %v), want %q (set %v)", name, got, ok, want, wantSet)
	}
}

func TestUseThenDeactivateRestoresOriginalEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	managed17 := filepath.Join(home, "jdk", "temurin@17.0.12")
	managed21 := filepath.Join(home, "jdk", "temurin@21.0.4")
	outside := t.TempDir()
	discovered21 := filepath.Join(outside, "java-21-openjdk-amd64")
	discovered17 := filepath.Join(outside, "java-17-openjdk-amd64")

	tests := []struct {
		name string
		path string
		jdks []string
	}{
		{name: "managed twice", path: joinPath("/usr/bin", "/bin"), jdks: []string{managed17, managed17}},
		{name: "managed then other managed", path: joinPath("/usr/bin", "/bin"), jdks: []string{managed17, managed21}},
		{name: "discovered twice", path: joinPath("/usr/bin", "/bin"), jdks: []string{discovered21, discovered21}},
		{name: "discovered then other discovered", path: joinPath("/usr/bin", "/bin"), jdks: []string{discovered21, discovered17}},
		{name: "discovered then managed", path: joinPath("/usr/bin", "/bin"), jdks: []string{discovered21, managed17, discovered17}},
		// The user's own entry for the same JDK must survive: javm removes only
		// the entry it prepended.
		{name: "original PATH already has the discovered bin", path: joinPath("/usr/bin", binForTest(discovered21), "/bin"), jdks: []string{discovered21, discovered21}},
		{name: "empty PATH entry is preserved", path: joinPath("/usr/bin", "", "/bin"), jdks: []string{discovered21, managed17}},
	}
	originals := []struct {
		name     string
		javaHome string
		set      bool
	}{
		{name: "with JAVA_HOME", javaHome: "/orig/java", set: true},
		{name: "without JAVA_HOME"},
	}
	for _, tt := range tests {
		for _, original := range originals {
			for _, dropEmpty := range []bool{false, true} {
				name := tt.name + "/" + original.name
				if dropEmpty {
					name += "/empty variables dropped"
				}
				t.Run(name, func(t *testing.T) {
					env := []string{"PATH=" + tt.path, "HOME=/home/user"}
					if original.set {
						env = append(env, "JAVA_HOME="+original.javaHome)
					}
					for _, jdk := range tt.jdks {
						env = useForTest(t, env, jdk, dropEmpty)
						assertEnvironmentValue(t, env, "PATH", joinPath(binForTest(jdk), tt.path), true)
						assertEnvironmentValue(t, env, "JAVA_HOME", javaHomeForTest(jdk), true)
						assertEnvironmentValue(t, env, activeJavaHomeKey, javaHomeForTest(jdk), true)
					}
					env = applyShellRecords(t, env, deactivateRecords(env), dropEmpty)
					assertEnvironmentValue(t, env, "PATH", tt.path, true)
					assertEnvironmentValue(t, env, "JAVA_HOME", original.javaHome, original.set)
					assertEnvironmentValue(t, env, javaHomeBeforeJavmKey, "", false)
					assertEnvironmentValue(t, env, activeJavaHomeKey, "", false)
				})
			}
		}
	}
}

func TestUseRecordsTrackOriginalJavaHome(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdk := filepath.Join(t.TempDir(), "jdk")
	tests := []struct {
		name string
		env  []string
		want string
	}{
		{name: "original JAVA_HOME", env: []string{"PATH=/usr/bin", "JAVA_HOME=/orig"}, want: "SET\tJAVA_HOME_BEFORE_JAVM\t/orig"},
		{name: "no original JAVA_HOME", env: []string{"PATH=/usr/bin"}, want: "UNSET\tJAVA_HOME_BEFORE_JAVM"},
		{name: "active without original", env: []string{"PATH=/usr/bin", "JAVA_HOME=/jdk", activeJavaHomeKey + "=/jdk"}, want: "UNSET\tJAVA_HOME_BEFORE_JAVM"},
		{name: "active with original", env: []string{"PATH=/usr/bin", "JAVA_HOME=/jdk", activeJavaHomeKey + "=/jdk", javaHomeBeforeJavmKey + "=/orig"}, want: "SET\tJAVA_HOME_BEFORE_JAVM\t/orig"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := useRecords(jdk, tt.env)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{
				"SET\tPATH\t" + joinPath(binForTest(jdk), "/usr/bin"),
				"SET\tJAVA_HOME\t" + javaHomeForTest(jdk),
				"SET\t" + activeJavaHomeKey + "\t" + javaHomeForTest(jdk),
				tt.want,
			}
			if !slices.Equal(records, want) {
				t.Fatalf("records = %q, want %q", records, want)
			}
		})
	}
}

// Shells activated by an older javm only have JAVA_HOME_BEFORE_JAVM.
func TestUseAndDeactivateUnderstandOlderActivation(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	outside := t.TempDir()
	old := filepath.Join(outside, "old-jdk")
	next := filepath.Join(outside, "next-jdk")
	env := []string{
		"PATH=" + joinPath(filepath.Join(old, "bin"), "/usr/bin"),
		"JAVA_HOME=" + old,
		javaHomeBeforeJavmKey + "=/orig",
	}

	assertDeactivate := deactivateRecords(env)
	wantDeactivate := []string{
		"SET\tPATH\t/usr/bin",
		"SET\tJAVA_HOME\t/orig",
		"UNSET\t" + javaHomeBeforeJavmKey,
		"UNSET\t" + activeJavaHomeKey,
	}
	if !slices.Equal(assertDeactivate, wantDeactivate) {
		t.Fatalf("deactivate records = %q, want %q", assertDeactivate, wantDeactivate)
	}

	env = useForTest(t, env, next, false)
	assertEnvironmentValue(t, env, "PATH", joinPath(binForTest(next), "/usr/bin"), true)
	assertEnvironmentValue(t, env, javaHomeBeforeJavmKey, "/orig", true)
}

func TestDeactivateRemovesManagedEntryAtEndOfPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	managed := filepath.Join(home, "jdk", "temurin@17.0.12", "bin")
	records := deactivateRecords([]string{"PATH=" + joinPath("/usr/bin", managed)})
	want := []string{
		"SET\tPATH\t/usr/bin",
		"UNSET\t" + javaHomeBeforeJavmKey,
		"UNSET\t" + activeJavaHomeKey,
	}
	if !slices.Equal(records, want) {
		t.Fatalf("records = %q, want %q", records, want)
	}
}

func TestDeactivateLeavesJavaHomeAloneWhenInactive(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	for _, env := range [][]string{
		{"PATH=/usr/bin", "JAVA_HOME=/user/java"},
		{"PATH=/usr/bin"},
	} {
		for _, record := range deactivateRecords(env) {
			if strings.Contains(record, "\tJAVA_HOME\t") || strings.HasSuffix(record, "\tJAVA_HOME") {
				t.Fatalf("inactive deactivate changed JAVA_HOME: %q", record)
			}
		}
	}
}

func TestActivePathEntryMatchesTrailingSeparator(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	jdk := filepath.Join(t.TempDir(), "jdk")
	env := []string{
		"PATH=" + joinPath(binForTest(jdk)+string(filepath.Separator), "/usr/bin"),
		"JAVA_HOME=" + javaHomeForTest(jdk),
		activeJavaHomeKey + "=" + javaHomeForTest(jdk),
	}
	if got := deactivateRecords(env)[0]; got != "SET\tPATH\t/usr/bin" {
		t.Fatalf("PATH record = %q, want the active entry removed", got)
	}
}

func TestActivePathEntryIgnoresCaseOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows paths are case-insensitive")
	}
	t.Setenv("JAVM_HOME", t.TempDir())
	env := []string{
		`Path=c:\jdks\temurin-21\BIN\;C:\Windows`,
		`JAVA_HOME=C:\Jdks\Temurin-21`,
		activeJavaHomeKey + `=C:\Jdks\Temurin-21`,
	}
	if got := deactivateRecords(env)[0]; got != "SET\tPATH\t"+`C:\Windows` {
		t.Fatalf("PATH record = %q, want the active entry removed", got)
	}
}
