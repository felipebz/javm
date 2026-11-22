package command

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/felipebz/javm/cfg"
)

func TestDeactivate(t *testing.T) {
	sep := string(os.PathListSeparator)
	javaHome := filepath.Join(cfg.Dir(), "jdk", "zulu@1.8.72")
	javaPath := filepath.Join(javaHome, "bin")
	actual := deactivateRecords([]string{
		"PATH=/usr/local/bin" + sep + javaPath + sep + "/system-jdk/bin" + sep + "/usr/bin",
		"JAVA_HOME=" + javaHome,
		"JAVA_HOME_BEFORE_JAVM=/system-jdk",
		"JAVM_ACTIVE_JAVA_HOME=" + javaHome,
	})
	expected := []string{
		"SET\tPATH\t" + "/usr/local/bin" + sep + "/system-jdk/bin" + sep + "/usr/bin",
		"SET\tJAVA_HOME\t" + "/system-jdk",
		"UNSET\tJAVA_HOME_BEFORE_JAVM",
		"UNSET\tJAVM_ACTIVE_JAVA_HOME",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}

func TestDeactivateInUnusedEnv(t *testing.T) {
	sep := string(os.PathListSeparator)
	path := "/usr/local/bin" + sep + "/system-jdk/bin" + sep + "/usr/bin"
	actual := deactivateRecords([]string{"PATH=" + path, "JAVA_HOME=/system-jdk"})
	expected := []string{
		"SET\tPATH\t" + path,
		"UNSET\tJAVA_HOME_BEFORE_JAVM",
		"UNSET\tJAVM_ACTIVE_JAVA_HOME",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("actual: %v != expected: %v", actual, expected)
	}
}
