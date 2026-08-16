package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	log "github.com/sirupsen/logrus"
)

// Mock Ls for testing purposes
var mockLsResult []discovery.JDK
var mockLsError error

func mockLs(context.Context) ([]discovery.JDK, error) {
	return mockLsResult, mockLsError
}

func setupMockLs() func() {
	originalLs := lsFunc
	lsFunc = mockLs
	return func() {
		lsFunc = originalLs
	}
}

func TestLsWarnsAndRecoversFromCorruptCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JAVM_HOME", home)
	if err := os.WriteFile(discovery.GetDefaultCacheFile(home), []byte(`{"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}
	config := &discovery.Config{
		Enabled: true,
		Sources: map[string]bool{
			"system": false, "jabba": false, "gradle": false, "intellij": false, "javm": false,
		},
		CacheTTL: time.Hour,
	}
	if err := config.SaveConfig(discovery.GetConfigFile(cfg.Dir())); err != nil {
		t.Fatal(err)
	}

	var warnings bytes.Buffer
	logger := log.New()
	logger.SetOutput(&warnings)
	cmd := NewLsCommand()
	cmd.SetContext(WithRuntime(context.Background(), Runtime{Logger: logger, Err: &warnings}))
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warnings.String(), "ignoring corrupt discovery cache") {
		t.Fatalf("corrupt cache warning was not emitted: %q", warnings.String())
	}
	if _, err := os.Stat(filepath.Join(home, "cache.json")); err != nil {
		t.Fatalf("recovered cache was not persisted: %v", err)
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
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	wantLines := [][]string{
		{"NAME", "SOURCE"},
		{"c-jdk@21", "gradle"},
		{"a-jdk@17", "javm"},
		{"b-jdk@17", "system"},
	}
	if len(gotLines) != len(wantLines) {
		t.Fatalf("unexpected number of output lines: got %d, want %d:\n%s", len(gotLines), len(wantLines), got)
	}
	for i, wantFields := range wantLines {
		gotFields := strings.Fields(gotLines[i])
		if len(gotFields) != len(wantFields) {
			t.Fatalf("unexpected fields on output line %d: got %q, want %q:\n%s", i, gotFields, wantFields, got)
		}
		for j, want := range wantFields {
			if gotFields[j] != want {
				t.Fatalf("unexpected output line %d: got %q, want %q:\n%s", i, gotFields, wantFields, got)
			}
		}
	}
}

func TestNewLsCommand_FiltersRangeAndDetails(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{
			Identifier:   "temurin@17.0.1",
			Version:      "17.0.1",
			Source:       "javm",
			Vendor:       "Eclipse Adoptium",
			Architecture: "amd64",
			Path:         "/jdks/temurin@17.0.1",
		},
		{
			Identifier:   "temurin@21.0.1",
			Version:      "21.0.1",
			Source:       "javm",
			Vendor:       "Eclipse Adoptium",
			Architecture: "amd64",
			Path:         "/jdks/temurin@21.0.1",
		},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"21"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected filtered output error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "temurin@21.0.1") {
		t.Fatalf("filtered output does not contain matching JDK:\n%s", got)
	}
	if strings.Contains(got, "temurin@17.0.1") {
		t.Fatalf("filtered output contains excluded JDK:\n%s", got)
	}

	details := NewLsCommand()
	var detailsOut bytes.Buffer
	details.SetOut(&detailsOut)
	details.SetArgs([]string{"--details", "21"})
	if err := details.Execute(); err != nil {
		t.Fatalf("unexpected detailed filtered output error: %v", err)
	}
	gotDetails := detailsOut.String()
	for _, value := range []string{"SOURCE", "NAME", "Eclipse Adoptium", "/jdks/temurin@21.0.1"} {
		if !strings.Contains(gotDetails, value) {
			t.Fatalf("detailed output missing %q:\n%s", value, gotDetails)
		}
	}
	if strings.Contains(gotDetails, "temurin@17.0.1") || strings.Contains(gotDetails, "/jdks/temurin@17.0.1") {
		t.Fatalf("detailed output contains excluded JDK:\n%s", gotDetails)
	}
}
