package command

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/felipebz/javm/cfg"
)

func Use(selector string) ([]string, error) {
	aliasValue := GetAlias(selector)
	if aliasValue != "" {
		selector = aliasValue
	}

	jdks, err := Ls(false)
	if err != nil {
		return nil, err
	}
	jdk, err := FindBestMatchJDK(jdks, selector)
	if err != nil {
		return nil, err
	}
	return usePath(jdk.Path)
}

func usePath(path string) ([]string, error) {
	sep := string(os.PathListSeparator)
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	pth, _ := os.LookupEnv("PATH")
	rgxp := regexp.MustCompile(regexp.QuoteMeta(filepath.Join(cfg.Dir(), "jdk")) + "[^" + sep + "]+[" + sep + "]")
	// strip references to ~/.jabba/jdk/*, otherwise leave unchanged
	pth = rgxp.ReplaceAllString(pth, "")
	if runtime.GOOS == "darwin" {
		path = filepath.Join(path, "Contents", "Home")
	}
	systemJavaHome, overrideWasSet := os.LookupEnv("JAVA_HOME_BEFORE_JABBA")
	if !overrideWasSet {
		systemJavaHome, _ = os.LookupEnv("JAVA_HOME")
	}
	return []string{
		"SET\tPATH\t" + filepath.Join(path, "bin") + string(os.PathListSeparator) + pth,
		"SET\tJAVA_HOME\t" + path,
		"SET\tJAVA_HOME_BEFORE_JABBA\t" + systemJavaHome,
	}, nil
}
