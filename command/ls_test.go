package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
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
		{"NAME", "SOURCE", "SELECTED", "BY"},
		{"c-jdk@21", "gradle", "21,", "c-jdk@21"},
		{"a-jdk@17", "javm", "17,", "a-jdk@17"},
		{"b-jdk@17", "system", "b-jdk@17"},
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
	for _, value := range []string{"SOURCE", "NAME", "SELECTED BY", "Eclipse Adoptium", "/jdks/temurin@21.0.1"} {
		if !strings.Contains(gotDetails, value) {
			t.Fatalf("detailed output missing %q:\n%s", value, gotDetails)
		}
	}
	if strings.Contains(gotDetails, "temurin@17.0.1") || strings.Contains(gotDetails, "/jdks/temurin@17.0.1") {
		t.Fatalf("detailed output contains excluded JDK:\n%s", gotDetails)
	}
}

type lsRow struct {
	name       string
	source     string
	selectedBy string
}

func parseLsOutput(t *testing.T, output string) []lsRow {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil
	}
	header := lines[0]
	nameIdx := strings.Index(header, "NAME")
	sourceIdx := strings.Index(header, "SOURCE")
	selectedByIdx := strings.Index(header, "SELECTED BY")
	if nameIdx < 0 || sourceIdx < 0 || selectedByIdx < 0 {
		t.Fatalf("unexpected header format: %q", header)
	}

	var rows []lsRow
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var name, source, selectedBy string
		if len(line) > selectedByIdx {
			name = strings.TrimSpace(line[nameIdx:sourceIdx])
			source = strings.TrimSpace(line[sourceIdx:selectedByIdx])
			selectedBy = strings.TrimSpace(line[selectedByIdx:])
		} else if len(line) > sourceIdx {
			name = strings.TrimSpace(line[nameIdx:sourceIdx])
			source = strings.TrimSpace(line[sourceIdx:])
		} else {
			name = strings.TrimSpace(line)
		}
		rows = append(rows, lsRow{name: name, source: source, selectedBy: selectedBy})
	}
	return rows
}

func TestLsSelectedBy_SameMajorMultipleDistributions(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "liberica@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "graalvm@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "liberica@25.0.2", Version: "25.0.2", Source: "javm"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4:\n%s", len(rows), out.String())
	}

	// Determine who the real resolver picks for "25"
	winner25, err := resolveJDKFromList(mockLsResult, "25")
	if err != nil {
		t.Fatalf("resolveJDKFromList(\"25\") error: %v", err)
	}
	winnerTemurin25, err := resolveJDKFromList(mockLsResult, "temurin@25")
	if err != nil {
		t.Fatalf("resolveJDKFromList(\"temurin@25\") error: %v", err)
	}
	winnerLiberica25, err := resolveJDKFromList(mockLsResult, "liberica@25")
	if err != nil {
		t.Fatalf("resolveJDKFromList(\"liberica@25\") error: %v", err)
	}
	winnerGraalvm25, err := resolveJDKFromList(mockLsResult, "graalvm@25")
	if err != nil {
		t.Fatalf("resolveJDKFromList(\"graalvm@25\") error: %v", err)
	}

	for _, row := range rows {
		selectors := strings.Split(row.selectedBy, ", ")
		if row.selectedBy == "" {
			selectors = nil
		}

		// Check "25" appears only in the winner's row
		has25 := slices.Contains(selectors, "25")
		if row.name == winner25.Identifier {
			if !has25 {
				t.Errorf("expected %q to have selector \"25\", got: %q", row.name, row.selectedBy)
			}
		} else {
			if has25 {
				t.Errorf("expected %q NOT to have selector \"25\", got: %q", row.name, row.selectedBy)
			}
		}

		// Check "temurin@25"
		hasTemurin25 := slices.Contains(selectors, "temurin@25")
		if row.name == winnerTemurin25.Identifier {
			if !hasTemurin25 {
				t.Errorf("expected %q to have selector \"temurin@25\", got: %q", row.name, row.selectedBy)
			}
		} else {
			if hasTemurin25 {
				t.Errorf("expected %q NOT to have selector \"temurin@25\", got: %q", row.name, row.selectedBy)
			}
		}

		// Check "liberica@25"
		hasLiberica25 := slices.Contains(selectors, "liberica@25")
		if row.name == winnerLiberica25.Identifier {
			if !hasLiberica25 {
				t.Errorf("expected %q to have selector \"liberica@25\", got: %q", row.name, row.selectedBy)
			}
		} else {
			if hasLiberica25 {
				t.Errorf("expected %q NOT to have selector \"liberica@25\", got: %q", row.name, row.selectedBy)
			}
		}

		// Check "graalvm@25"
		hasGraalvm25 := slices.Contains(selectors, "graalvm@25")
		if row.name == winnerGraalvm25.Identifier {
			if !hasGraalvm25 {
				t.Errorf("expected %q to have selector \"graalvm@25\", got: %q", row.name, row.selectedBy)
			}
		} else {
			if hasGraalvm25 {
				t.Errorf("expected %q NOT to have selector \"graalvm@25\", got: %q", row.name, row.selectedBy)
			}
		}

		// Check liberica@25.0.2 wins nothing
		if row.name == "liberica@25.0.2" {
			if row.selectedBy != "" {
				t.Errorf("expected liberica@25.0.2 to have empty selectedBy, got: %q", row.selectedBy)
			}
		}
	}
}

func TestLsSelectedBy_SameDistributionMultipleVersions(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "liberica@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "liberica@25.0.2", Version: "25.0.2", Source: "javm"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2:\n%s", len(rows), out.String())
	}

	winner, err := resolveJDKFromList(mockLsResult, "liberica@25")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	for _, row := range rows {
		selectors := strings.Split(row.selectedBy, ", ")
		if row.selectedBy == "" {
			selectors = nil
		}
		if row.name == winner.Identifier {
			if !slices.Contains(selectors, "liberica@25") {
				t.Errorf("expected winner %q to have liberica@25, got %q", row.name, row.selectedBy)
			}
		} else {
			if slices.Contains(selectors, "liberica@25") {
				t.Errorf("expected non-winner %q NOT to have liberica@25, got %q", row.name, row.selectedBy)
			}
			if row.selectedBy != "" {
				t.Errorf("expected non-winner %q to have empty selectedBy, got %q", row.name, row.selectedBy)
			}
		}
	}
}

func TestLsSelectedBy_MajorWithOnlySystemJDK(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "oracle-corporation-system@8", Version: "1.8.0_442", Source: "system"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1:\n%s", len(rows), out.String())
	}

	winner, err := resolveJDKFromList(mockLsResult, "8")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if winner.Identifier != "oracle-corporation-system@8" {
		t.Fatalf("winner of 8 = %q, want oracle-corporation-system@8", winner.Identifier)
	}

	wantSelectedBy := "8, oracle-corporation-system@8"
	if rows[0].selectedBy != wantSelectedBy {
		t.Errorf("selectedBy = %q, want %q", rows[0].selectedBy, wantSelectedBy)
	}
}

func TestLsSelectedBy_PreservesRowOrdering(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "liberica@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "graalvm@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "liberica@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "temurin@24.0.2", Version: "24.0.2", Source: "javm"},
		{Identifier: "temurin@21.0.12.1", Version: "21.0.12.1", Source: "javm"},
		{Identifier: "temurin@17.0.20.1", Version: "17.0.20.1", Source: "javm"},
		{Identifier: "zulu@13.0.14", Version: "13.0.14", Source: "javm"},
		{Identifier: "temurin@11.0.32.1", Version: "11.0.32.1", Source: "javm"},
		{Identifier: "temurin@8.0.504", Version: "8.0.504", Source: "javm"},
		{Identifier: "oracle-corporation-system@8", Version: "1.8.0_442", Source: "system"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())

	// Compare with expected ordering: Source ASC, Version DESC, Identifier ASC
	expected := slices.Clone(mockLsResult)
	sort.Slice(expected, func(i, j int) bool {
		if expected[i].Source != expected[j].Source {
			return expected[i].Source < expected[j].Source
		}
		v1, err1 := javaversion.ParseVersion(expected[i].Version)
		v2, err2 := javaversion.ParseVersion(expected[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		if expected[i].Version != expected[j].Version {
			return expected[i].Version > expected[j].Version
		}
		return expected[i].Identifier < expected[j].Identifier
	})

	if len(rows) != len(expected) {
		t.Fatalf("got %d rows, want %d", len(rows), len(expected))
	}
	for i, row := range rows {
		if row.name != expected[i].Identifier || row.source != expected[i].Source {
			t.Errorf("row %d: got name=%q source=%q, want name=%q source=%q",
				i, row.name, row.source, expected[i].Identifier, expected[i].Source)
		}
	}
}

func TestLsSelectedBy_InvarianceRule(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "liberica@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "graalvm@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "liberica@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "temurin@24.0.2", Version: "24.0.2", Source: "javm"},
		{Identifier: "temurin@21.0.12.1", Version: "21.0.12.1", Source: "javm"},
		{Identifier: "temurin@17.0.20.1", Version: "17.0.20.1", Source: "javm"},
		{Identifier: "zulu@13.0.14", Version: "13.0.14", Source: "javm"},
		{Identifier: "temurin@11.0.32.1", Version: "11.0.32.1", Source: "javm"},
		{Identifier: "temurin@8.0.504", Version: "8.0.504", Source: "javm"},
		{Identifier: "oracle-corporation-system@8", Version: "1.8.0_442", Source: "system"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())

	// Invariance assertion:
	// For each selector displayed in SELECTED BY, resolving that selector
	// using the real resolver MUST return exactly the JDK on that row.
	for _, row := range rows {
		if row.selectedBy == "" {
			continue
		}
		selectors := strings.Split(row.selectedBy, ", ")
		for _, selector := range selectors {
			resolved, err := resolveJDKFromList(mockLsResult, selector)
			if err != nil {
				t.Errorf("selector %q on row %q failed to resolve: %v", selector, row.name, err)
				continue
			}
			if resolved.Identifier != row.name {
				t.Errorf("invariance violated for selector %q: resolved to %q, but row is %q",
					selector, resolved.Identifier, row.name)
			}
			if resolved.Source != row.source {
				t.Errorf("invariance violated for selector %q: resolved source %q, but row source is %q",
					selector, resolved.Source, row.source)
			}
		}
	}
}

func TestLsSelectedBy_SelectorOrderWithinColumn(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows := parseLsOutput(t, out.String())
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}

	want := "25, temurin@25"
	if rows[0].selectedBy != want {
		t.Errorf("selectedBy = %q, want %q", rows[0].selectedBy, want)
	}
}

func TestLsSelectedBy_FullPromptIllustration(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{Identifier: "temurin@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "liberica@25.0.4.1", Version: "25.0.4.1", Source: "javm"},
		{Identifier: "graalvm@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "liberica@25.0.2", Version: "25.0.2", Source: "javm"},
		{Identifier: "temurin@24.0.2", Version: "24.0.2", Source: "javm"},
		{Identifier: "temurin@21.0.12.1", Version: "21.0.12.1", Source: "javm"},
		{Identifier: "temurin@17.0.20.1", Version: "17.0.20.1", Source: "javm"},
		{Identifier: "zulu@13.0.14", Version: "13.0.14", Source: "javm"},
		{Identifier: "temurin@11.0.32.1", Version: "11.0.32.1", Source: "javm"},
		{Identifier: "temurin@8.0.504", Version: "8.0.504", Source: "javm"},
		{Identifier: "oracle-corporation-system@8", Version: "1.8.0_442", Source: "system"},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("\n%s", out.String())

	rows := parseLsOutput(t, out.String())
	if len(rows) != len(mockLsResult) {
		t.Fatalf("got %d rows, want %d", len(rows), len(mockLsResult))
	}

	// Verify invariance for every row
	for _, row := range rows {
		if row.selectedBy == "" {
			continue
		}
		selectors := strings.Split(row.selectedBy, ", ")
		for _, selector := range selectors {
			resolved, err := resolveJDKFromList(mockLsResult, selector)
			if err != nil {
				t.Fatalf("failed to resolve %q: %v", selector, err)
			}
			if resolved.Identifier != row.name {
				t.Fatalf("selector %q resolved to %q, want %q", selector, resolved.Identifier, row.name)
			}
		}
	}
}

func TestLsSelectedBy_DetailsOutput(t *testing.T) {
	cleanup := setupMockLs()
	defer cleanup()

	mockLsResult = []discovery.JDK{
		{
			Identifier:   "temurin@25.0.4.1",
			Version:      "25.0.4.1",
			Source:       "javm",
			Vendor:       "Eclipse Adoptium",
			Architecture: "x64",
			Path:         "/jdks/temurin@25.0.4.1",
		},
		{
			Identifier:   "liberica@25.0.2",
			Version:      "25.0.2",
			Source:       "javm",
			Vendor:       "BellSoft",
			Architecture: "x64",
			Path:         "/jdks/liberica@25.0.2",
		},
	}

	cmd := NewLsCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--details"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), got)
	}

	headerFields := strings.Fields(lines[0])
	wantHeader := []string{"SOURCE", "NAME", "SELECTED", "BY", "VENDOR", "ARCHITECTURE", "PATH"}
	if !slices.Equal(headerFields, wantHeader) {
		t.Errorf("header = %v, want %v", headerFields, wantHeader)
	}

	// temurin@25.0.4.1 should have "25, temurin@25"
	if !strings.Contains(lines[1], "25, temurin@25") {
		t.Errorf("expected line 1 to contain '25, temurin@25', got: %s", lines[1])
	}
	// liberica@25.0.2 has no selected by, but contains BellSoft and path
	if !strings.Contains(lines[2], "BellSoft") || !strings.Contains(lines[2], "/jdks/liberica@25.0.2") {
		t.Errorf("expected line 2 details, got: %s", lines[2])
	}
}
