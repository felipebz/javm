package command

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/felipebz/javm/discovery"
)

type fakeManager struct {
	jdks []discovery.JDK
	err  error
}

func (f *fakeManager) DiscoverAll(context.Context) ([]discovery.JDK, error) {
	return f.jdks, f.err
}

func TestDiscoverRefreshCommand_Success(t *testing.T) {
	previousFactory := newManagerWithAllSources
	newManagerWithAllSources = func(configDir string, forceRefresh bool, warn func(error)) (discoverRunner, error) {
		return &fakeManager{}, nil
	}
	t.Cleanup(func() { newManagerWithAllSources = previousFactory })

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
