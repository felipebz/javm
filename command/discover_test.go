package command

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/felipebz/javm/discovery"
)

type fakeManager struct {
	jdks []discovery.JDK
	err  error
}

func (f *fakeManager) DiscoverAll() ([]discovery.JDK, error) {
	return f.jdks, f.err
}

func TestDiscoverRefreshCommand_Success(t *testing.T) {
	newManagerWithAllSources = func(cacheFile string, ttl time.Duration) discoverRunner {
		return &fakeManager{}
	}

	cmd := newDiscoverRefreshCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()

	if !strings.Contains(got, "Discovery cache refreshed successfully") {
		t.Errorf("expected success message, got: %s", got)
	}
}
