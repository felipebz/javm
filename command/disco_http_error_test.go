package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/spf13/cobra"
)

// fakeDisco answers packages for the distributions in packages and 400, as
// the Disco API does, for any other distribution or an unknown OS.
type fakeDisco struct {
	packages          map[string][]discoapi.Package
	distributions     []string
	failDistributions bool
	packageCalls      int
	distributionCalls int
}

func (f *fakeDisco) client(t *testing.T) *discoapi.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/packages"):
			f.packageCalls++
			// Like the real API, the distribution matches in any case and the
			// packages keep their own spelling.
			distribution := r.URL.Query().Get("distribution")
			var packages []discoapi.Package
			ok := false
			for name, list := range f.packages {
				if strings.EqualFold(name, distribution) {
					packages, ok = list, true
				}
			}
			switch {
			case r.URL.Query().Get("operating_system") == "bogus":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"result":[],"message":"invalid operating system"}`))
			case !ok:
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"result":[],"message":"` + distribution + ` is an unsupported distribution"}`))
			default:
				_ = json.NewEncoder(w).Encode(discoapi.PackagesResponse{Packages: packages})
			}
		case strings.HasSuffix(r.URL.Path, "/distributions"):
			f.distributionCalls++
			if f.failDistributions {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var response discoapi.DistributionsResponse
			for _, name := range f.distributions {
				response.Distributions = append(response.Distributions, discoapi.Distribution{Name: name, APIParameter: name})
			}
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return &discoapi.Client{BaseURL: server.URL, HTTPClient: server.Client()}
}

func temurinAndZulu() *fakeDisco {
	return &fakeDisco{
		packages: map[string][]discoapi.Package{
			"temurin": {{Id: "t21", JavaVersion: "21.0.5+11", Distribution: "temurin", DistributionVersion: "21.0.5"}},
		},
		distributions: []string{"temurin", "zulu"},
	}
}

func runDiscoCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	t.Setenv("JAVM_HOME", t.TempDir())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceUsage = true
	cmd.SetArgs(append([]string{}, args...))
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

func assertUnknownDistribution(t *testing.T, err error, name, hint string) {
	t.Helper()
	text := ""
	if err != nil {
		text = err.Error()
	}
	if !errors.Is(err, ErrUsage) || !strings.Contains(text, `unknown distribution "`+name+`"`) ||
		!strings.Contains(text, "javm ls-distributions") || !strings.Contains(text, hint) || strings.Contains(text, "network error") {
		t.Fatalf("error = %v, want the unknown distribution usage error%s", err, hint)
	}
}

func TestUnknownDistributionAnsweredWith400IsUsageError(t *testing.T) {
	disco := temurinAndZulu()
	client := disco.client(t)

	_, err := runDiscoCommand(t, NewInstallCommand(client), "foo@21")
	assertUnknownDistribution(t, err, "foo", "")
	_, err = runDiscoCommand(t, NewLsRemoteCommand(client), "foo")
	assertUnknownDistribution(t, err, "foo", "")
}

func TestDistributionInWrongCaseIsUsageError(t *testing.T) {
	disco := temurinAndZulu()
	disco.packages["zulu"] = []discoapi.Package{{Id: "z21", JavaVersion: "21.0.5+11", Distribution: "zulu", DistributionVersion: "21.38.21"}}
	client := disco.client(t)

	for _, args := range [][]string{{"ZULU"}, {"ZULU@21"}, {"--distribution", "ZULU"}} {
		out, err := runDiscoCommand(t, NewLsRemoteCommand(client), args...)
		assertUnknownDistribution(t, err, "ZULU", `did you mean "zulu"?`)
		if out != "" {
			t.Fatalf("ls-remote %v printed %q", args, out)
		}
	}
	_, err := runDiscoCommand(t, NewInstallCommand(client), "ZULU@21")
	assertUnknownDistribution(t, err, "ZULU", `did you mean "zulu"?`)
	if entries, _ := os.ReadDir(filepath.Join(cfg.Dir(), "jdk")); len(entries) != 0 {
		t.Fatalf("install ZULU@21 left %d entries in jdk/", len(entries))
	}

	disco.packageCalls, disco.distributionCalls = 0, 0
	out, err := runDiscoCommand(t, NewLsRemoteCommand(client), "zulu")
	if err != nil || !strings.Contains(out, "zulu@21") || disco.packageCalls != 1 || disco.distributionCalls != 0 {
		t.Fatalf("ls-remote zulu = %q, %v; requests %d/%d, want output, 1 packages and 0 distributions requests", out, err, disco.packageCalls, disco.distributionCalls)
	}
}

func TestKnownDistributionAnsweredWith400StaysNetworkError(t *testing.T) {
	_, err := runDiscoCommand(t, NewLsRemoteCommand(temurinAndZulu().client(t)), "zulu")
	if !errors.Is(err, ErrNetwork) || !strings.Contains(err.Error(), "returned 400: zulu is an unsupported distribution") {
		t.Fatalf("error = %v, want a network error with the API message", err)
	}
}

func TestDistributionListFailureKeepsOriginalError(t *testing.T) {
	disco := temurinAndZulu()
	disco.failDistributions = true
	_, err := runDiscoCommand(t, NewLsRemoteCommand(disco.client(t)), "foo")
	if !errors.Is(err, ErrNetwork) || !strings.Contains(err.Error(), "returned 400: foo is an unsupported distribution") {
		t.Fatalf("error = %v, want the original packages error", err)
	}
}

func TestUnknownOSAnsweredWith400ShowsAPIMessage(t *testing.T) {
	_, err := runDiscoCommand(t, NewLsRemoteCommand(temurinAndZulu().client(t)), "--os", "bogus")
	if !errors.Is(err, ErrNetwork) || !strings.Contains(err.Error(), "invalid operating system") {
		t.Fatalf("error = %v, want a network error with the API message", err)
	}
}

func TestSuccessfulPackagesRequestIsTheOnlyRequest(t *testing.T) {
	disco := temurinAndZulu()
	out, err := runDiscoCommand(t, NewLsRemoteCommand(disco.client(t)), "temurin")
	if err != nil || !strings.Contains(out, "temurin@21") {
		t.Fatalf("ls-remote temurin = %q, %v", out, err)
	}
	if disco.packageCalls != 1 || disco.distributionCalls != 0 {
		t.Fatalf("requests: packages %d, distributions %d; want 1 and 0", disco.packageCalls, disco.distributionCalls)
	}
}

// httpUpdateClient takes packages and distributions from the fake Disco API
// and package downloads from the update fixture.
type httpUpdateClient struct {
	*discoapi.Client
	info *updateClient
}

func (c httpUpdateClient) GetPackageInfoContext(ctx context.Context, id string) (*discoapi.PackageInfo, error) {
	return c.info.GetPackageInfoContext(ctx, id)
}

func TestUpdateSkipsDistributionAnsweredWith400(t *testing.T) {
	fixture := temurinUpdateFixture(t)
	writeManagedJDK(t, "gone@17.0.12", "17.0.12+7", "")
	disco := &fakeDisco{packages: map[string][]discoapi.Package{"temurin": fixture.packages}, distributions: []string{"temurin"}}
	client := httpUpdateClient{Client: disco.client(t), info: fixture}
	ctx, logs := contextWithLogBuffer(context.Background())

	if err := runUpdate(ctx, client, &bytes.Buffer{}, "", updateOptions{}); err != nil {
		t.Fatalf("runUpdate() error = %v, want the other lines updated and no error", err)
	}
	assertExists(t, "temurin@25.0.5", true)
	assertExists(t, "gone@17.0.12", true)
	if got := countLines(logs, `distribution \"gone\" is not offered by the Disco API anymore`); got != 1 {
		t.Fatalf("logs = %q, want one warning explaining the skipped line", logs.String())
	}
}
