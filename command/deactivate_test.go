package command

import (
	"github.com/felipebz/javm/cfg"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeactivate(t *testing.T) {
	prevPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	defer func() { os.Setenv("PATH", prevPath) }()
	javaHome := filepath.Join(cfg.Dir(), "jdk", "zulu@1.8.72")
	javaPath := filepath.Join(javaHome, "bin")
	os.Setenv("PATH", "/usr/local/bin"+sep+javaPath+sep+"/system-jdk/bin"+sep+"/usr/bin")
	os.Setenv("JAVA_HOME", javaHome)
	os.Setenv("JAVA_HOME_BEFORE_JABBA", "/system-jdk")
	actual, err := Deactivate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := []string{
		"export PATH=\"/usr/local/bin" + sep + "/system-jdk/bin" + sep + "/usr/bin\"",
		"export JAVA_HOME=\"/system-jdk\"",
		"unset JAVA_HOME_BEFORE_JABBA",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}

func TestDeactivateInUnusedEnv(t *testing.T) {
	prevPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", prevPath) }()
	os.Setenv("PATH", "/usr/local/bin:/system-jdk/bin:/usr/bin")
	os.Setenv("JAVA_HOME", "/system-jdk")
	os.Unsetenv("JAVA_HOME_BEFORE_JABBA")
	actual, err := Deactivate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := []string{
		"export PATH=\"/usr/local/bin:/system-jdk/bin:/usr/bin\"",
		"export JAVA_HOME=\"/system-jdk\"",
		"unset JAVA_HOME_BEFORE_JABBA",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
