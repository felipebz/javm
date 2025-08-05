package cfg

import (
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"runtime"
)

func Dir() string {
	home := os.Getenv("JAVM_HOME")
	if home != "" {
		return filepath.Clean(home)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	switch runtime.GOOS {
	case "windows":
		// Windows: %LOCALAPPDATA%\javm\jdks
		localAppData := os.Getenv("LOCALAPPDATA")
		return filepath.Join(localAppData, "javm")
	case "darwin":
		// macOS: ~/Library/Application Support/javm/jdks
		return filepath.Join(homeDir, "Library", "Application Support", "javm")
	default:
		// Linux and others: ~/.local/share/javm/jdks
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			// Default according to XDG spec
			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}
		return filepath.Join(xdgDataHome, "javm")
	}
}
