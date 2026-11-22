package command

import (
	"strings"
	"testing"
)

func TestLsRemoteDeprecationHintIsCopyable(t *testing.T) {
	for _, tt := range []struct {
		args            []string
		hint, identfied string
	}{
		{args: []string{"--distribution", "zulu", ">=21 <25"}, hint: `use: javm ls-remote "zulu@>=21 <25"`, identfied: "zulu@21"},
		{args: []string{"--distribution", "zulu", "25"}, hint: "use: javm ls-remote zulu@25\n", identfied: "zulu@25"},
	} {
		out, logs, err := runLsRemoteCommand(t, remoteCatalog(t), tt.args...)
		if err != nil || strings.Join(identifiers(out), " ") != tt.identfied {
			t.Fatalf("ls-remote %q = %q, %v, want %s", tt.args, out, err, tt.identfied)
		}
		if strings.Count(logs, "deprecated") != 1 || !strings.Contains(logs, tt.hint) {
			t.Fatalf("logs = %q, want one warning ending with %q", logs, tt.hint)
		}
	}
}
