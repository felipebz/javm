//go:build linux

package command

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func parentShellHint() (shellIntegrationHint, bool) {
	parentPID := os.Getppid()
	if parentPID <= 0 {
		return shellIntegrationHint{}, false
	}
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(parentPID), "comm"))
	if err != nil {
		return shellIntegrationHint{}, false
	}
	return shellIntegrationHintForName(shellName(strings.TrimSpace(string(data))))
}
