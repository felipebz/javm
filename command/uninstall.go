package command

import (
	"os"
	"path/filepath"

	"github.com/felipebz/javm/cfg"
)

func Uninstall(selector string) error {
	ver, err := LsBestMatch(selector, true)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(cfg.Dir(), "jdk", ver))
}
