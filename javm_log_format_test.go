package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestSimpleFormatterPrefixesLevels(t *testing.T) {
	tests := []struct {
		level   log.Level
		message string
		fields  log.Fields
		want    string
	}{
		{log.WarnLevel, "default is gone", nil, "Warning: default is gone\n"},
		{log.WarnLevel, "skipped", log.Fields{"path": "/b", "id": "a"}, "Warning: skipped id=a path=/b\n"},
		{log.InfoLevel, "temurin@21 -> /jdk", nil, "temurin@21 -> /jdk\n"},
		{log.ErrorLevel, "broken", nil, "Error: broken\n"},
		{log.DebugLevel, "fetching packages", nil, "Debug: fetching packages\n"},
		{log.InfoLevel, "linked", log.Fields{"name": "x"}, "linked name=x\n"},
	}
	for _, tt := range tests {
		got, err := (&simpleFormatter{}).Format(&log.Entry{Level: tt.level, Message: tt.message, Data: tt.fields})
		if err != nil || string(got) != tt.want {
			t.Errorf("Format(%v %q %v) = %q, %v, want %q", tt.level, tt.message, tt.fields, got, err, tt.want)
		}
	}
}

func TestWarningsArePrefixedOnStderr(t *testing.T) {
	_, home := isolatedWorkspace(t)
	jdk := writeFakeManagedJDK(t, home)
	if err := os.WriteFile(filepath.Join(home, "17.alias"), []byte("temurin@21"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := executeRoot(t, "which", "17")
	if code != exitSuccess || strings.TrimSpace(stdout) != jdk {
		t.Fatalf("which 17: exit %d, stdout %q, want %s", code, stdout, jdk)
	}
	if strings.Count(stderr, "Warning: ") != 1 || !strings.HasPrefix(stderr, `Warning: alias "17" hides`) || strings.Contains(stdout, "Warning: ") {
		t.Fatalf("stderr = %q, want one line starting with the warning prefix", stderr)
	}
}
