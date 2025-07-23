package command

import (
	"bytes"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

// --- Mock implementation ---

type mockPackagesClient struct {
	Pkgs []discoapi.Package
	Err  error
}

func (m *mockPackagesClient) GetPackages(os, arch, distribution, version string) ([]discoapi.Package, error) {
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
	cmd.SetArgs([]string{"--os=linux", "--arch=amd64"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier           Full Version    Distribution Version
temurin@17.0.8       17.0.8          temurin 17.0.8
temurin@21.0.1       21.0.1          temurin 21.0.1
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
	want = `Identifier           Full Version    Distribution Version
zulu@19.0.1          19.0.1          zulu 19.0.1
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
	want = `Identifier           Full Version    Distribution Version
temurin@21.0.1       21.0.1          temurin 21.0.1
`
	if got != want {
		t.Errorf("--distribution= got:\n%q\nwant:\n%q", got, want)
	}

	// Test with semver range (e.g. >=20)
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
	want = `Identifier           Full Version    Distribution Version
temurin@21.0.1       21.0.1          temurin 21.0.1
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
	err := runLsRemote(&out, mock, "linux", "amd64", "", "", "")
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
	err := runLsRemote(&out, mock, "linux", "amd64", "zulu", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier           Full Version    Distribution Version
zulu@17.0.8          17.0.8+1        zulu 20.0.8
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
	err := runLsRemote(&out, mock, "linux", "amd64", "", "", ">=20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier           Full Version    Distribution Version
temurin@21.0.1       21.0.1          temurin 21.0.1
`
	if got != want {
		t.Errorf("range got:\n%q\nwant:\n%q", got, want)
	}
}
