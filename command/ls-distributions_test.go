package command

import (
	"bytes"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

type mockClient struct {
	distributions []discoapi.Distribution
	err           error
}

func (m *mockClient) GetDistributions() ([]discoapi.Distribution, error) {
	return m.distributions, m.err
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
