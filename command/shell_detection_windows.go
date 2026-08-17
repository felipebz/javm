//go:build windows

package command

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func parentShellHint() (shellIntegrationHint, bool) {
	parentPID := os.Getppid()
	if parentPID <= 0 {
		return shellIntegrationHint{}, false
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return shellIntegrationHint{}, false
	}
	defer windows.CloseHandle(snapshot)

	var process windows.ProcessEntry32
	process.Size = uint32(unsafe.Sizeof(process))
	if err := windows.Process32First(snapshot, &process); err != nil {
		return shellIntegrationHint{}, false
	}
	for {
		if int(process.ProcessID) == parentPID {
			return shellIntegrationHintForName(shellName(windows.UTF16ToString(process.ExeFile[:])))
		}
		if err := windows.Process32Next(snapshot, &process); err != nil {
			return shellIntegrationHint{}, false
		}
	}
}
