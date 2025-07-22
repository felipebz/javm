package cfg

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func Dir() string {
	home := os.Getenv("JABBA_HOME")
	if home != "" {
		return filepath.Clean(home)
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dir, ".jabba")
}

func Index() string {
	registry := os.Getenv("JABBA_INDEX")
	if registry == "" {
		registry = "https://github.com/felipebz/javm/raw/master/index.json"
	}
	return registry
}
