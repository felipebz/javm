//go:build darwin

package command

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func parentShellHint() (shellIntegrationHint, bool) {
	parentPID := os.Getppid()
	if parentPID <= 0 {
		return shellIntegrationHint{}, false
	}
	process, err := unix.SysctlKinfoProc("kern.proc.pid", parentPID)
	if err != nil {
		return shellIntegrationHint{}, false
	}
	name := strings.TrimRight(string(process.Proc.P_comm[:]), "\x00")
	return shellIntegrationHintForName(shellName(name))
}
