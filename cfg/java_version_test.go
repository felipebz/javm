package cfg

import (
	"testing"
	"testing/fstest"
)

func TestReadJavaVersionFromFS(t *testing.T) {
	tests := []struct {
		name     string
		fs       fstest.MapFS
		expected string
	}{
		{
			name: "valid .java-version file",
			fs: fstest.MapFS{
				".java-version": &fstest.MapFile{Data: []byte("21\n")},
			},
			expected: "21",
		},
		{
			name: "valid .java-version file with spaces",
			fs: fstest.MapFS{
				".java-version": &fstest.MapFile{Data: []byte("  17.0.1  ")},
			},
			expected: "17.0.1",
		},
		{
			name:     "missing .java-version file",
			fs:       fstest.MapFS{},
			expected: "",
		},
		{
			name: "empty .java-version file",
			fs: fstest.MapFS{
				".java-version": &fstest.MapFile{Data: []byte("")},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReadJavaVersionFromFS(tt.fs)
			if got != tt.expected {
				t.Errorf("ReadJavaVersionFromFS() = %v, want %v", got, tt.expected)
			}
		})
	}
}
