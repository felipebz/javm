package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

func vcsBuildInfo(mainVersion, modified string) *debug.BuildInfo {
	return &debug.BuildInfo{
		Main: debug.Module{Path: "github.com/felipebz/javm", Version: mainVersion},
		Settings: []debug.BuildSetting{
			{Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: "c8a53c0cac054bc448dc3e8ad1c1cab01871d6b6"},
			{Key: "vcs.time", Value: "2026-11-22T13:09:22Z"},
			{Key: "vcs.modified", Value: modified},
		},
	}
}

func TestVersionText(t *testing.T) {
	tests := []struct {
		name                  string
		version, commit, date string
		info                  *debug.BuildInfo
		want                  string
	}{
		{
			name:    "release ldflags win over build info",
			version: "0.17.0", commit: "0123456789abcdef0123456789abcdef01234567", date: "2026-11-01T10:00:00Z",
			info: vcsBuildInfo("v9.9.9", "true"),
			want: "0.17.0 (commit 0123456789abcdef0123456789abcdef01234567, built at 2026-11-01T10:00:00Z)",
		},
		{
			name:    "release version only",
			version: "0.17.0",
			want:    "0.17.0",
		},
		{
			name: "go install of a tag",
			info: &debug.BuildInfo{Main: debug.Module{Version: "v0.17.0"}},
			want: "0.17.0",
		},
		{
			name: "devel build in a modified checkout",
			info: vcsBuildInfo("(devel)", "true"),
			want: "dev (commit c8a53c0cac054bc448dc3e8ad1c1cab01871d6b6-dirty, built at 2026-11-22T13:09:22Z)",
		},
		{
			name: "pseudo-version already marked dirty",
			info: vcsBuildInfo("v0.17.1-0.20261122130922-c8a53c0cac05+dirty", "true"),
			want: "0.17.1-0.20261122130922-c8a53c0cac05+dirty (commit c8a53c0cac054bc448dc3e8ad1c1cab01871d6b6, built at 2026-11-22T13:09:22Z)",
		},
		{
			name: "clean checkout",
			info: vcsBuildInfo("(devel)", "false"),
			want: "dev (commit c8a53c0cac054bc448dc3e8ad1c1cab01871d6b6, built at 2026-11-22T13:09:22Z)",
		},
		{
			name: "no build info",
			want: "dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionText(tt.version, tt.commit, tt.date, tt.info); got != tt.want {
				t.Fatalf("versionText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	isolatedWorkspace(t)
	stdout, stderr, code := executeRoot(t, "--version")
	if code != exitSuccess || strings.TrimSpace(stdout) == "" || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("exit %d, stdout %q, stderr %q, want one non-empty line", code, stdout, stderr)
	}
}
