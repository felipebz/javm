package command

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/felipebz/javm/cfg"
)

func TestDeactivate(t *testing.T) {
	prevPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	defer func() { os.Setenv("PATH", prevPath) }()
	javaHome := filepath.Join(cfg.Dir(), "jdk", "zulu@1.8.72")
	javaPath := filepath.Join(javaHome, "bin")
	os.Setenv("PATH", "/usr/local/bin"+sep+javaPath+sep+"/system-jdk/bin"+sep+"/usr/bin")
	os.Setenv("JAVA_HOME", javaHome)
	os.Setenv("JAVA_HOME_BEFORE_JAVM", "/system-jdk")
	actual, err := deactivate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := []string{
		"SET\tPATH\t" + "/usr/local/bin" + sep + "/system-jdk/bin" + sep + "/usr/bin",
		"SET\tJAVA_HOME\t" + "/system-jdk",
		"UNSET\tJAVA_HOME_BEFORE_JAVM",
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
	os.Unsetenv("JAVA_HOME_BEFORE_JAVM")
	actual, err := deactivate()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	expected := []string{
		"SET\tPATH\t" + "/usr/local/bin:/system-jdk/bin:/usr/bin",
		"SET\tJAVA_HOME\t" + "/system-jdk",
		"UNSET\tJAVA_HOME_BEFORE_JAVM",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
