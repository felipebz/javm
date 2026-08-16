package command

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandArgumentValidationUsesUsageClass(t *testing.T) {
	config := NewConfigCommand()
	discover := NewDiscoverCommand()

	commands := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{name: "install", cmd: NewInstallCommand(nil), args: []string{"17", "21"}},
		{name: "uninstall missing", cmd: NewUninstallCommand(), args: nil},
		{name: "link", cmd: NewLinkCommand(), args: []string{"system@17", "/jdk", "extra"}},
		{name: "unlink", cmd: NewUnlinkCommand(), args: []string{"system@17", "extra"}},
		{name: "use", cmd: NewUseCommand(), args: []string{"17", "21"}},
		{name: "current", cmd: NewCurrentCommand(), args: []string{"extra"}},
		{name: "ls", cmd: NewLsCommand(), args: []string{"17", "21"}},
		{name: "ls-remote", cmd: NewLsRemoteCommand(nil), args: []string{"17", "21"}},
		{name: "ls-distributions", cmd: NewLsDistributionsCommand(nil), args: []string{"extra"}},
		{name: "deactivate", cmd: NewDeactivateCommand(), args: []string{"extra"}},
		{name: "alias", cmd: NewAliasCommand(), args: []string{"default", "17", "extra"}},
		{name: "unalias", cmd: NewUnaliasCommand(), args: []string{"default", "extra"}},
		{name: "which", cmd: NewWhichCommand(), args: []string{"17", "21"}},
		{name: "init", cmd: NewInitCommand(), args: nil},
		{name: "default", cmd: NewDefaultCommand(), args: nil},
		{name: "discover", cmd: discover, args: []string{"extra"}},
		{name: "config", cmd: config, args: []string{"extra"}},
	}

	for _, tt := range commands {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.ValidateArgs(tt.args)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("ValidateArgs(%q) = %v, want ErrUsage", tt.args, err)
			}
		})
	}
}

func TestFindBestMatchJDKClassifiesMissingVersion(t *testing.T) {
	_, err := FindBestMatchJDK(nil, "17")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindBestMatchJDK() error = %v, want ErrNotFound", err)
	}
}

func TestUsageErrorPreservesCause(t *testing.T) {
	cause := errors.New("invalid selector")
	wrapped := UsageError(cause)
	if !errors.Is(wrapped, ErrUsage) {
		t.Fatalf("UsageError() does not classify error: %v", wrapped)
	}
	if !errors.Is(wrapped, cause) {
		t.Fatalf("UsageError() does not preserve cause: %v", wrapped)
	}
}
