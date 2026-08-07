package command

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

type mockClient struct {
	distributions []discoapi.Distribution
	err           error
}

func (m *mockClient) GetDistributionsContext(ctx context.Context) ([]discoapi.Distribution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.distributions, m.err
}

func TestLsDistributionsPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := NewLsDistributionsCommand(&mockClient{})
	cmd.SetArgs(nil)

	if err := cmd.ExecuteContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestNewLsDistributionsCommand(t *testing.T) {
	mock := &mockClient{distributions: []discoapi.Distribution{
		{Name: "Temurin", APIParameter: "temurin"},
		{Name: "Zulu", APIParameter: "zulu"},
	}}
	cmd := NewLsDistributionsCommand(mock)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	want := `Identifier           Name
temurin              Temurin
zulu                 Zulu
`
	if got != want {
		t.Errorf("unexpected output:\n%q\nwant:\n%q", got, want)
	}
}
