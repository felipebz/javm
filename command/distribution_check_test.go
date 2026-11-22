package command

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
)

// distributionsClient serves packages filtered by distribution, like the Disco
// API, and counts the requests for the distribution list.
type distributionsClient struct {
	versionSelectionClient
	distributionsErr error
	calls            int
}

func (c *distributionsClient) GetPackagesContext(_ context.Context, _, _, distribution, _ string) ([]discoapi.Package, error) {
	var packages []discoapi.Package
	for _, pkg := range c.packages {
		if distribution == "" || pkg.Distribution == distribution {
			packages = append(packages, pkg)
		}
	}
	return packages, nil
}

func (c *distributionsClient) GetDistributionsContext(context.Context) ([]discoapi.Distribution, error) {
	c.calls++
	if c.distributionsErr != nil {
		return nil, c.distributionsErr
	}
	return []discoapi.Distribution{{Name: "Temurin", APIParameter: "temurin"}, {Name: "Zulu", APIParameter: "zulu"}}, nil
}

func temurin17Client(t *testing.T) *distributionsClient {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	return &distributionsClient{versionSelectionClient: versionSelectionClient{
		packages: []discoapi.Package{{Id: "17", JavaVersion: "17.0.12+7", Distribution: "temurin", DistributionVersion: "17.0.12"}},
		archive:  makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}}),
	}}
}

func TestInstallRejectsUnknownDistribution(t *testing.T) {
	for name, tt := range map[string]struct {
		selector, defaultDistribution, want string
	}{
		"qualifier":            {selector: "nope@17", want: `unknown distribution "nope"; run 'javm ls-distributions'`},
		"default distribution": {selector: "17", defaultDistribution: "nope", want: `unknown distribution "nope"`},
		"wrong case":           {selector: "TEMURIN@17", want: `did you mean "temurin"?`},
	} {
		t.Run(name, func(t *testing.T) {
			client := temurin17Client(t)
			if tt.defaultDistribution != "" {
				if err := cfg.SetValue("java.default_distribution", tt.defaultDistribution); err != nil {
					t.Fatal(err)
				}
			}
			destination := filepath.Join(t.TempDir(), "jdk")
			_, err := runInstall(context.Background(), client, tt.selector, destination)
			if !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("install %s error = %v, want a usage error containing %q", tt.selector, err, tt.want)
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("something was installed: %v", err)
			}
		})
	}
}

func TestInstallKnownDistributionMakesNoExtraRequest(t *testing.T) {
	client := temurin17Client(t)
	if _, err := runInstall(context.Background(), client, "temurin@17", filepath.Join(t.TempDir(), "jdk")); err != nil {
		t.Fatal(err)
	}
	if client.calls != 0 {
		t.Fatalf("distribution list requested %d times on the happy path", client.calls)
	}

	_, err := runInstall(context.Background(), client, "temurin@99", filepath.Join(t.TempDir(), "jdk"))
	if !errors.Is(err, ErrNotFound) || client.calls > 1 {
		t.Fatalf("install temurin@99 error = %v after %d list requests, want not found and at most one", err, client.calls)
	}
}

func TestInstallKeepsNotFoundWhenDistributionListFails(t *testing.T) {
	client := temurin17Client(t)
	client.distributionsErr = discoapi.ErrNetwork
	_, err := runInstall(context.Background(), client, "nope@17", filepath.Join(t.TempDir(), "jdk"))
	if !errors.Is(err, ErrNotFound) || errors.Is(err, ErrNetwork) || !strings.Contains(err.Error(), "no compatible version found for nope@17") {
		t.Fatalf("install error = %v, want today's not-found error", err)
	}
}

func lsRemoteWithStderr(t *testing.T, client *distributionsClient, distribution string) (string, string, error) {
	t.Helper()
	var out, stderr bytes.Buffer
	ctx := WithRuntime(context.Background(), Runtime{Err: &stderr})
	err := runLsRemote(ctx, &out, client, "", lsRemoteOptions{os: "linux", arch: "amd64", distribution: distribution, latest: "major"})
	return out.String(), stderr.String(), err
}

func TestLsRemoteRejectsUnknownDistribution(t *testing.T) {
	client := temurin17Client(t)
	if _, _, err := lsRemoteWithStderr(t, client, "nope"); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `unknown distribution "nope"`) {
		t.Fatalf("ls-remote --distribution nope error = %v, want a usage error", err)
	}

	out, stderr, err := lsRemoteWithStderr(t, client, "zulu")
	if err != nil || out != "" || strings.Count(stderr, "\n") != 1 || !strings.Contains(stderr, `"zulu"`) || !strings.Contains(stderr, "linux/amd64") {
		t.Fatalf("ls-remote --distribution zulu = %q, stderr %q, %v, want no listing and one stderr line", out, stderr, err)
	}

	client.calls = 0
	out, stderr, err = lsRemoteWithStderr(t, client, "all")
	if err != nil || !strings.Contains(out, "temurin@17") || stderr != "" || client.calls != 0 {
		t.Fatalf("ls-remote --distribution all = %q, stderr %q, %v, %d list requests", out, stderr, err, client.calls)
	}
}

func runConfigSet(t *testing.T, client DistributionsClient, value string) (string, error) {
	t.Helper()
	ctx, logs := contextWithLogBuffer(context.Background())
	cmd := NewConfigCommand(client)
	cmd.SetArgs([]string{"set", "java.default_distribution", value})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	err := cmd.ExecuteContext(ctx)
	return logs.String(), err
}

func TestConfigSetValidatesDistribution(t *testing.T) {
	t.Setenv("JAVM_HOME", t.TempDir())
	client := &distributionsClient{}

	if _, err := runConfigSet(t, client, "nope"); !errors.Is(err, ErrUsage) || !strings.Contains(err.Error(), `unknown distribution "nope"`) {
		t.Fatalf("config set nope error = %v, want a usage error", err)
	}
	if value, _ := cfg.EffectiveValue("java.default_distribution"); value != "temurin" {
		t.Fatalf("default distribution = %q after a rejected value", value)
	}

	if _, err := runConfigSet(t, client, "zulu"); err != nil {
		t.Fatal(err)
	}
	if value, _ := cfg.EffectiveValue("java.default_distribution"); value != "zulu" {
		t.Fatalf("default distribution = %q, want zulu", value)
	}

	client.distributionsErr = discoapi.ErrNetwork
	logs, err := runConfigSet(t, client, "corretto")
	if err != nil || strings.Count(logs, "could not check") != 1 {
		t.Fatalf("offline config set = %v, logs %q, want success and one warning", err, logs)
	}
	if value, _ := cfg.EffectiveValue("java.default_distribution"); value != "corretto" {
		t.Fatalf("default distribution = %q, want corretto", value)
	}
}
