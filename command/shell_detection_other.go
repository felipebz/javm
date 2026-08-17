//go:build !linux && !darwin && !windows

package command

func parentShellHint() (shellIntegrationHint, bool) {
	return shellIntegrationHint{}, false
}
