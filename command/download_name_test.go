package command

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Oracle GraalVM downloads come from an URL without an extension; the
// Disco API's file name tells the archive type.
func TestDownloadTakesArchiveTypeFromFileName(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "archive")
	}))
	defer server.Close()

	for _, tc := range []struct {
		name, url, filename, want string
	}{
		{"file name without URL extension", "/api/artifacts/0335/content", "graalvm-jdk-25.0.5.1_windows-x64_bin.zip", ".zip"},
		{"tar.gz file name", "/api/artifacts/0335/content", "graalvm-jdk-25.0.5.1_linux-x64_bin.tar.gz", ".tar.gz"},
		{"URL extension without file name", "/jdk.zip", "", ".zip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := downloadWithClient(context.Background(), server.Client(), server.URL+tc.url, tc.filename, 32)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(file)
			if got := getFileExtension(file); got != tc.want {
				t.Fatalf("download %s has extension %q, want %q", file, got, tc.want)
			}
		})
	}
}
