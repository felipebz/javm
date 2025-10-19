package command

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/felipebz/javm/cfg"
)

func Deactivate() ([]string, error) {
	sep := string(os.PathListSeparator)
	pth, _ := os.LookupEnv("PATH")
	rgxp := regexp.MustCompile(regexp.QuoteMeta(filepath.Join(cfg.Dir(), "jdk")) + "[^" + sep + "]+[" + sep + "]")
	// strip references to ~/.jabba/jdk/*, otherwise leave unchanged
	pth = rgxp.ReplaceAllString(pth, "")
	javaHome, overrideWasSet := os.LookupEnv("JAVA_HOME_BEFORE_JABBA")
	if !overrideWasSet {
		javaHome, _ = os.LookupEnv("JAVA_HOME")
	}
	return []string{
		"SET\tPATH\t" + pth,
		"SET\tJAVA_HOME\t" + javaHome,
		"UNSET\tJAVA_HOME_BEFORE_JABBA",
	}, nil
}
