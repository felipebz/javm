package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/discoapi"
	log "github.com/sirupsen/logrus"
)

func remoteCatalog(t *testing.T) *distributionsClient {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	var packages []discoapi.Package
	for _, p := range []struct{ distribution, version string }{
		{"temurin", "17.0.12"}, {"temurin", "21.0.4"}, {"temurin", "25.0.1"}, {"zulu", "21.0.5"}, {"zulu", "25.0.2"},
	} {
		packages = append(packages, discoapi.Package{Id: p.distribution + p.version, JavaVersion: p.version, Distribution: p.distribution, DistributionVersion: p.version})
	}
	return &distributionsClient{versionSelectionClient: versionSelectionClient{packages: packages}}
}

func runLsRemoteCommand(t *testing.T, client *distributionsClient, args ...string) (string, string, error) {
	t.Helper()
	var out, logs bytes.Buffer
	logger := log.New()
	logger.SetOutput(&logs)
	logger.SetFormatter(&log.TextFormatter{DisableQuote: true, DisableTimestamp: true})
	ctx := WithRuntime(context.Background(), Runtime{Logger: logger, Err: &bytes.Buffer{}})
	cmd := NewLsRemoteCommand(client)
	cmd.SetArgs(append([]string{"--os=linux", "--arch=amd64"}, args...))
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	err := cmd.ExecuteContext(ctx)
	return out.String(), logs.String(), err
}

func identifiers(listing string) []string {
	var ids []string
	for i, line := range strings.Split(strings.TrimSpace(listing), "\n") {
		if i > 0 && line != "" {
			ids = append(ids, strings.Fields(line)[0])
		}
	}
	return ids
}

func TestLsRemoteSelectorChoosesDistribution(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"zulu"}, want: "zulu@21 zulu@25"},
		{args: []string{"zulu@"}, want: "zulu@21 zulu@25"},
		{args: []string{"zulu@25"}, want: "zulu@25"},
		{args: []string{"21"}, want: "temurin@21"},
		{args: nil, want: "temurin@17 temurin@21 temurin@25"},
		{args: []string{"all@21"}, want: "temurin@21 zulu@21"},
		{args: []string{"*@21"}, want: "temurin@21 zulu@21"},
		{args: []string{"all"}, want: "temurin@17 temurin@21 temurin@25 zulu@21 zulu@25"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			client := remoteCatalog(t)
			out, logs, err := runLsRemoteCommand(t, client, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(sortedStrings(identifiers(out)), " "); got != tt.want {
				t.Fatalf("ls-remote %q = %q, want %q", tt.args, got, tt.want)
			}
			if client.calls != 0 || logs != "" {
				t.Fatalf("%d distribution list requests, logs %q, want none", client.calls, logs)
			}
		})
	}
}

func TestLsRemoteRejectsUnknownOrInvalidSelectors(t *testing.T) {
	for arg, want := range map[string]string{
		"nope":    `unknown distribution "nope"`,
		"nope@21": `unknown distribution "nope"`,
		"TEMURIN": `did you mean "temurin"?`,
		"zu lu":   "is not a valid version",
		">=":      "is not a valid version",
	} {
		t.Run(arg, func(t *testing.T) {
			_, _, err := runLsRemoteCommand(t, remoteCatalog(t), arg)
			if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), want) {
				t.Fatalf("ls-remote %q error = %v, want a usage error containing %q", arg, err, want)
			}
		})
	}
}

func TestLsRemoteDistributionFlagIsDeprecated(t *testing.T) {
	for _, args := range [][]string{{"--distribution", "zulu", "25"}, {"--distribution", "zulu", "zulu@25"}} {
		out, logs, err := runLsRemoteCommand(t, remoteCatalog(t), args...)
		if err != nil || strings.Join(identifiers(out), " ") != "zulu@25" {
			t.Fatalf("ls-remote %q = %q, %v, want zulu@25", args, out, err)
		}
		if strings.Count(logs, "deprecated") != 1 || !strings.Contains(logs, "javm ls-remote zulu@25") {
			t.Fatalf("logs = %q, want one deprecation warning showing the new form", logs)
		}
	}
	_, _, err := runLsRemoteCommand(t, remoteCatalog(t), "--distribution", "zulu", "temurin@25")
	if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `"zulu"`) || !strings.Contains(err.Error(), `"temurin"`) {
		t.Fatalf("conflicting distributions error = %v, want a usage error naming both", err)
	}
}

func TestInstallRequiresVersion(t *testing.T) {
	for _, selector := range []string{"zulu@", "zulu"} {
		t.Run(selector, func(t *testing.T) {
			client := remoteCatalog(t)
			destination := filepath.Join(t.TempDir(), "jdk")
			_, err := runInstall(context.Background(), client, selector, destination)
			if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `use "zulu@<version>"`) || !strings.Contains(err.Error(), "javm ls-remote zulu") {
				t.Fatalf("install %s error = %v, want a usage error asking for a version", selector, err)
			}
			if client.infoID != "" {
				t.Fatalf("install %s requested package %q", selector, client.infoID)
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("something was installed: %v", err)
			}
		})
	}
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
