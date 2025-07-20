package command

import (
	"bytes"
	"errors"
	"testing"

	"github.com/felipebz/javm/discoapi"
)

type mockDistributionsClient struct {
	distributions []discoapi.Distribution
	err           error
}

func (m *mockDistributionsClient) GetDistributions() ([]discoapi.Distribution, error) {
	return m.distributions, m.err
}

func TestLsDistributions(t *testing.T) {
	tests := []struct {
		name          string
		mockErr       error
		mockData      []discoapi.Distribution
		wantErr       bool
		wantCount     int
		wantFirstName string
	}{
		{
			name:          "success",
			mockData:      []discoapi.Distribution{{Name: "Temurin", APIParameter: "temurin"}},
			wantErr:       false,
			wantCount:     1,
			wantFirstName: "Temurin",
		},
		{
			name:      "api error",
			mockErr:   errors.New("boom"),
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockDistributionsClient{
				distributions: tc.mockData,
				err:           tc.mockErr,
			}
			distributions, err := LsDistributions(client)
			if (err != nil) != tc.wantErr {
				t.Fatalf("expected error: %v, got %v", tc.wantErr, err)
			}
			if len(distributions) != tc.wantCount {
				t.Fatalf("expected count: %d, got %d", tc.wantCount, len(distributions))
			}
			if tc.wantCount > 0 && distributions[0].Name != tc.wantFirstName {
				t.Errorf("expected first Name %q, got %q", tc.wantFirstName, distributions[0].Name)
			}
		})
	}
}

func TestPrintDistributions(t *testing.T) {
	distributions := []discoapi.Distribution{
		{Name: "Temurin", APIParameter: "temurin"},
		{Name: "Zulu", APIParameter: "zulu"},
	}

	var buf bytes.Buffer
	PrintDistributions(&buf, distributions)
	got := buf.String()
	want := `Identifier           Name
temurin              Temurin
zulu                 Zulu
`
	if got != want {
		t.Errorf("expected output:\n%q\ngot:\n%q", want, got)
	}
}
