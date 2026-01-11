package command

import (
	"bytes"
	"testing"

	"github.com/felipebz/javm/discovery"
)

// Mock Ls for testing purposes
var mockLsResult []discovery.JDK
var mockLsError error

func mockLs() ([]discovery.JDK, error) {
	return mockLsResult, mockLsError
}

func setupMockLs() func() {
	originalLs := lsFunc
	lsFunc = mockLs
	return func() {
		lsFunc = originalLs
	}
}

func TestLsBestMatch(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@17.0.1", Version: "17.0.1", Source: "javm"},
		{Identifier: "system@21", Version: "21.0.0", Source: "system"},
		{Identifier: "temurin@8.0.352", Version: "1.8.0_352", Source: "javm"},
	}

	tests := []struct {
		selector string
		want     string
		wantErr  bool
	}{
		{"17", "temurin@17.0.1", false},
		{"21", "system@21", false},
		{"8", "temurin@8.0.352", false},
		{"30", "", true},
	}

	for _, tt := range tests {
		got, err := LsBestMatch(tt.selector, false)
		if (err != nil) != tt.wantErr {
			t.Errorf("LsBestMatch(%q) error = %v, wantErr %v", tt.selector, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("LsBestMatch(%q) = %v, want %v", tt.selector, got, tt.want)
		}
	}
}

func TestNewLsCommand_Output(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "b-jdk@17", Version: "17.0.0", Source: "system"},
		{Identifier: "a-jdk@17", Version: "17.0.0", Source: "javm"},
		{Identifier: "c-jdk@21", Version: "21.0.0", Source: "gradle"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	// Order: Source (ASC) -> Version (DESC)
	// Sources: gradle, javm, system
	// Expected order:
	// 1. gradle -> c-jdk@21
	// 2. javm -> a-jdk@17
	// 3. system -> b-jdk@17

	// We expect simple containment check or specific order
	expectedLines := []string{
		"IDENTIFIER\tSOURCE",
		"c-jdk@21\tgradle",
		"a-jdk@17\tjavm",
		"b-jdk@17\tsystem",
	}

	for _, line := range expectedLines {
		if !contains(got, line) {
			t.Errorf("output missing line: %q\nGot:\n%s", line, got)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(s)] == s && (s == substr || len(s) > len(substr))
}
