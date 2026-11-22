package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

// --- Mock implementation ---

type mockPackagesClient struct {
	Pkgs []discoapi.Package
	Err  error
}

func (m *mockPackagesClient) GetPackagesContext(ctx context.Context, os, arch, distribution, version string) ([]discoapi.Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.Pkgs, m.Err
}

func TestNewLsRemoteCommand_DefaultAndFlags(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{JavaVersion: "21.0.1", Distribution: "temurin", DistributionVersion: "21.0.1"},
			{JavaVersion: "17.0.8", Distribution: "temurin", DistributionVersion: "17.0.8"},
		},
	}
	cmd := NewLsRemoteCommand(mock)
	var out bytes.Buffer
	cmd.SetOut(&out)

	// Test default (should use "temurin" and print versions descending)
	cmd.SetArgs([]string{"--os=linux", "--arch=amd64", "--latest=patch"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier       Full Version   Distribution   Distribution Version
temurin@17.0.8   17.0.8         temurin        17.0.8
temurin@21.0.1   21.0.1         temurin        21.0.1
`
	if got != want {
		t.Errorf("default got:\n%q\nwant:\n%q", got, want)
	}

	// Test --distribution=zulu
	out.Reset()
	mock.Pkgs = []discoapi.Package{
		{JavaVersion: "19.0.1", Distribution: "zulu", DistributionVersion: "19.0.1"},
	}
	cmd.SetArgs([]string{"--distribution=zulu"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = out.String()
	want = `Identifier    Full Version   Distribution   Distribution Version
zulu@19.0.1   19.0.1         zulu           19.0.1
`
	if got != want {
		t.Errorf("--distribution=zulu got:\n%q\nwant:\n%q", got, want)
	}

	// Test --distribution= (empty: show all with <distribution>@<version>)
	out.Reset()
	mock.Pkgs = []discoapi.Package{
		{JavaVersion: "21.0.1", Distribution: "temurin", DistributionVersion: "21.0.1"},
	}
	cmd.SetArgs([]string{"--distribution="})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = out.String()
	want = `Identifier       Full Version   Distribution   Distribution Version
temurin@21.0.1   21.0.1         temurin        21.0.1
`
	if got != want {
		t.Errorf("--distribution= got:\n%q\nwant:\n%q", got, want)
	}

	// Test with a version range (e.g. >=20)
	out.Reset()
	mock.Pkgs = []discoapi.Package{
		{JavaVersion: "21.0.1", Distribution: "temurin", DistributionVersion: "21.0.1"},
		{JavaVersion: "17.0.8", Distribution: "temurin", DistributionVersion: "17.0.8"},
	}
	cmd.SetArgs([]string{">=20"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = out.String()
	want = `Identifier       Full Version   Distribution   Distribution Version
temurin@21.0.1   21.0.1         temurin        21.0.1
`
	if got != want {
		t.Errorf("range got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRunLsRemote_EmptyResult(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{},
	}
	var out bytes.Buffer
	err := runLsRemote(context.Background(), &out, mock, "", lsRemoteOptions{os: "linux", arch: "amd64", latest: "major"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := ""
	if got != want {
		t.Errorf("empty result got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRunLsRemote_SpecificDistribution(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{JavaVersion: "17.0.8+1", Distribution: "zulu", DistributionVersion: "20.0.8"},
		},
	}
	var out bytes.Buffer
	err := runLsRemote(context.Background(), &out, mock, "", lsRemoteOptions{os: "linux", arch: "amd64", distribution: "zulu", latest: "patch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier    Full Version   Distribution   Distribution Version
zulu@17.0.8   17.0.8+1       zulu           20.0.8
`
	if got != want {
		t.Errorf("specific got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRunLsRemote_WithRange(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{JavaVersion: "21.0.1", Distribution: "temurin", DistributionVersion: "21.0.1"},
			{JavaVersion: "19.0.1", Distribution: "temurin", DistributionVersion: "19.0.1"},
			{JavaVersion: "17.0.8", Distribution: "temurin", DistributionVersion: "17.0.8"},
		},
	}
	var out bytes.Buffer
	err := runLsRemote(context.Background(), &out, mock, ">=20", lsRemoteOptions{os: "linux", arch: "amd64", latest: "patch"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier       Full Version   Distribution   Distribution Version
temurin@21.0.1   21.0.1         temurin        21.0.1
`
	if got != want {
		t.Errorf("range got:\n%q\nwant:\n%q", got, want)
	}
}

func TestRunLsRemoteSelectsLatestCompleteJavaVersion(t *testing.T) {
	mock := &mockPackagesClient{
		Pkgs: []discoapi.Package{
			{Id: "security", JavaVersion: "25.0.4.1+1", Distribution: "temurin", DistributionVersion: "25.0.4.1"},
			{Id: "baseline", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
		},
	}
	var out bytes.Buffer
	if err := runLsRemote(context.Background(), &out, mock, "", lsRemoteOptions{os: "linux", arch: "amd64", distribution: "temurin", latest: "major"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `Identifier   Full Version   Distribution   Distribution Version
temurin@25   25.0.4.1+1     temurin        25.0.4.1
`
	if got := out.String(); got != want {
		t.Fatalf("latest Java version output =\n%q\nwant:\n%q", got, want)
	}

	// DiscoAPI ordering must not affect which complete Java version is
	// selected.
	out.Reset()
	mock.Pkgs = []discoapi.Package{
		{Id: "baseline", JavaVersion: "25.0.4+7", Distribution: "temurin", DistributionVersion: "25.0.4"},
		{Id: "security", JavaVersion: "25.0.4.1+1", Distribution: "temurin", DistributionVersion: "25.0.4.1"},
	}
	if err := runLsRemote(context.Background(), &out, mock, "", lsRemoteOptions{os: "linux", arch: "amd64", distribution: "temurin", latest: "major"}); err != nil {
		t.Fatalf("unexpected reverse-order error: %v", err)
	}
	if got := out.String(); got != want {
		t.Fatalf("reverse-order latest Java version output =\n%q\nwant:\n%q", got, want)
	}
}

func TestLsRemotePropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := NewLsRemoteCommand(&mockPackagesClient{})
	cmd.SetArgs(nil)

	if err := cmd.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestRunLsRemoteWithoutMatchPrintsNothing(t *testing.T) {
	mock := &mockPackagesClient{Pkgs: []discoapi.Package{
		{JavaVersion: "17.0.8", Distribution: "temurin", DistributionVersion: "17.0.8"},
	}}
	var out bytes.Buffer
	if err := runLsRemote(context.Background(), &out, mock, ">=99", lsRemoteOptions{os: "linux", arch: "amd64", latest: "patch"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want nothing when no version matches", out.String())
	}
}

// Every column starts where its header starts, however long the values are,
// and each row has one value per header.
func TestRunLsRemoteAlignsColumnsToContent(t *testing.T) {
	mock := &mockPackagesClient{Pkgs: []discoapi.Package{
		{JavaVersion: "23.1.6+2", Distribution: "liberica-native", DistributionVersion: "23.1.6"},
		{JavaVersion: "21.0.4+7", Distribution: "temurin", DistributionVersion: "21.0.4"},
	}}
	var out bytes.Buffer
	if err := runLsRemote(context.Background(), &out, mock, "", lsRemoteOptions{os: "linux", arch: "amd64", latest: "patch"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 3 || !strings.Contains(out.String(), "liberica-native@23.1.6") {
		t.Fatalf("output = %q, want a header and two rows", out.String())
	}

	headers := []string{"Identifier", "Full Version", "Distribution", "Distribution Version"}
	starts := make([]int, len(headers))
	offset := 0
	for i, header := range headers {
		index := strings.Index(lines[0][offset:], header)
		if index < 0 {
			t.Fatalf("header %q is missing from %q", header, lines[0])
		}
		starts[i] = offset + index
		offset = starts[i] + len(header)
	}
	for _, row := range lines[1:] {
		// The values contain no spaces, so whitespace splitting counts them.
		if fields := strings.Fields(row); len(fields) != len(headers) {
			t.Fatalf("row %q has %d values, want %d", row, len(fields), len(headers))
		}
		for _, start := range starts[1:] {
			if len(row) <= start || row[start] == ' ' || row[start-1] != ' ' {
				t.Fatalf("row %q does not start a column at offset %d like the header %q", row, start, lines[0])
			}
		}
	}
}
