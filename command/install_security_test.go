package command

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/felipebz/javm/discoapi"
	"github.com/spf13/cobra"
	"github.com/ulikunitz/xz"
)

func TestInstallRollsBackAndCleansStaging(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: "jdk/readme.txt", body: "not a JDK", mode: 0644}})
	parent := t.TempDir()
	dst := filepath.Join(parent, "jdk")

	err := install(context.Background(), archive, dst)
	if err == nil || !strings.Contains(err.Error(), "installation rolled back") {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("partial destination remains after failure: %v", err)
	}
	staging, err := filepath.Glob(filepath.Join(parent, ".jdk.staging-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(staging) != 0 {
		t.Fatalf("staging directories remain after failure: %v", staging)
	}
}

func TestInstallCancellationCleansOwnedStaging(t *testing.T) {
	parent := t.TempDir()
	dst := filepath.Join(parent, "jdk")
	ctx, cancel := context.WithCancel(context.Background())

	err := installWithExtractor(ctx, "jdk.zip", dst, func(ctx context.Context, _, staging string) error {
		if err := os.WriteFile(filepath.Join(staging, "partial"), []byte("partial"), 0600); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("partial destination remains after cancellation: %v", err)
	}
	staging, err := filepath.Glob(filepath.Join(parent, ".jdk.staging-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(staging) != 0 {
		t.Fatalf("staging directories remain after cancellation: %v", staging)
	}
}

func TestInstallCommandDoesNotPrintUsageWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rootCmd := &cobra.Command{Use: "javm"}
	rootCmd.AddCommand(NewInstallCommand(installPackagesClient{}))
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"install", "21"})

	err := rootCmd.ExecuteContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
	if strings.Contains(output.String(), "Usage:") {
		t.Fatalf("usage was printed for cancellation:\n%s", output.String())
	}
}

func TestInstallNeverReplacesExistingDestination(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	dst := t.TempDir()
	sentinel := filepath.Join(dst, "owned-by-user")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := install(context.Background(), archive, dst); err == nil || !strings.Contains(err.Error(), "refusing to replace") {
		t.Fatalf("expected existing destination error, got %v", err)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing destination was modified: data=%q err=%v", data, err)
	}
}

func TestPromoteNoReplaceIsAtomic(t *testing.T) {
	parent := t.TempDir()
	source := filepath.Join(parent, "source")
	destination := filepath.Join(parent, "destination")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "new"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "old"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := promoteNoReplace(source, destination); err == nil {
		t.Fatal("expected promotion collision to fail")
	}
	if _, err := os.Stat(filepath.Join(source, "new")); err != nil {
		t.Fatalf("source changed after failed promotion: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(destination, "old")); err != nil || string(data) != "old" {
		t.Fatalf("destination changed after failed promotion: data=%q err=%v", data, err)
	}
}

func TestArchiveRejectsUnsafePathsAndSymlinks(t *testing.T) {
	t.Run("tar traversal", func(t *testing.T) {
		archive := makeTarGzArchive(t, []tarTestEntry{{name: "jdk/../../outside", body: "owned", typeflag: tar.TypeReg}})
		assertUnsafeInstall(t, archive, "unsafe archive path")
	})

	t.Run("tar escaping symlink", func(t *testing.T) {
		archive := makeTarGzArchive(t, []tarTestEntry{{name: "jdk/link", linkname: "../../outside", typeflag: tar.TypeSymlink}})
		assertUnsafeInstall(t, archive, "unsafe symlink")
	})

	t.Run("zip traversal", func(t *testing.T) {
		archive := makeZipArchive(t, []zipTestEntry{{name: "jdk/../../outside", body: "owned", mode: 0644}})
		assertUnsafeInstall(t, archive, "unsafe archive path")
	})

	t.Run("zip escaping symlink", func(t *testing.T) {
		archive := makeZipArchive(t, []zipTestEntry{{name: "jdk/link", body: "../../outside", mode: os.ModeSymlink | 0777}})
		assertUnsafeInstall(t, archive, "unsafe symlink")
	})

	t.Run("tar xz escaping symlink", func(t *testing.T) {
		archive := makeTarXzArchive(t, []tarTestEntry{{name: "jdk/link", linkname: "/outside", typeflag: tar.TypeSymlink}})
		assertUnsafeInstall(t, archive, "unsafe symlink")
	})
}

func TestTarRejectsWriteThroughArchiveSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	archive := makeTarGzArchive(t, []tarTestEntry{
		{name: "jdk/real", typeflag: tar.TypeDir},
		{name: "jdk/link", linkname: "real", typeflag: tar.TypeSymlink},
		{name: "jdk/link/payload", body: "owned", typeflag: tar.TypeReg},
	})
	assertUnsafeInstall(t, archive, "traverses symlink")
}

func TestArchiveExtractionLimits(t *testing.T) {
	t.Run("entry count", func(t *testing.T) {
		var archive bytes.Buffer
		tw := tar.NewWriter(&archive)
		writeTarEntries(t, tw, []tarTestEntry{
			{name: "jdk/one", body: "1", typeflag: tar.TypeReg},
			{name: "jdk/two", body: "2", typeflag: tar.TypeReg},
		})
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		err := extractTarWithLimits(context.Background(), &archive, t.TempDir(), true, extractionLimits{maxEntries: 1, maxBytes: 100})
		if err == nil || !strings.Contains(err.Error(), "exceeds 1 entries") {
			t.Fatalf("expected entry limit error, got %v", err)
		}
	})

	t.Run("expanded bytes", func(t *testing.T) {
		archive := makeZipArchive(t, []zipTestEntry{{name: "jdk/large", body: "12345", mode: 0644}})
		err := unzipWithLimits(context.Background(), archive, t.TempDir(), true, extractionLimits{maxEntries: 10, maxBytes: 4})
		if err == nil || !strings.Contains(err.Error(), "exceeds 4 extracted bytes") {
			t.Fatalf("expected extracted size error, got %v", err)
		}
	})
}

func TestTarExtractionStopsWhenContextIsCanceled(t *testing.T) {
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	writeTarEntries(t, tw, []tarTestEntry{{name: "jdk/bin/java", body: "java", typeflag: tar.TypeReg}})
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelAfterRead{reader: bytes.NewReader(archive.Bytes()), cancel: cancel}
	err := extractTar(ctx, reader, t.TempDir(), true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
}

func TestDownloadValidatesStatusSizeAndCancellation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "archive")
		}))
		defer server.Close()
		file, err := downloadWithClient(context.Background(), server.Client(), server.URL+"/jdk.zip?token=x", 32)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)
		if filepath.Ext(file) != ".zip" {
			t.Fatalf("download extension = %q, want .zip", filepath.Ext(file))
		}
		data, err := os.ReadFile(file)
		if err != nil || string(data) != "archive" {
			t.Fatalf("downloaded data=%q err=%v", data, err)
		}
	})

	t.Run("non-2xx", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()
		if _, err := downloadWithClient(context.Background(), server.Client(), server.URL+"/jdk.zip", 32); err == nil || !strings.Contains(err.Error(), "HTTP 502") {
			t.Fatalf("expected status error, got %v", err)
		}
	})

	t.Run("progress uses configured diagnostic stream", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "7")
			_, _ = io.WriteString(w, "archive")
		}))
		defer server.Close()
		var diagnostics bytes.Buffer
		ctx := WithRuntime(context.Background(), Runtime{Err: &diagnostics, ShowProgress: true})
		file, err := downloadWithClient(ctx, server.Client(), server.URL+"/jdk.zip", 32)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)
		if diagnostics.Len() == 0 {
			t.Fatal("progress did not use the configured diagnostic stream")
		}
	})

	t.Run("chunked response over limit is removed", func(t *testing.T) {
		t.Setenv("TMPDIR", t.TempDir())
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			_, _ = io.WriteString(w, "123456")
		}))
		defer server.Close()
		if _, err := downloadWithClient(context.Background(), server.Client(), server.URL+"/jdk.zip", 5); err == nil || !strings.Contains(err.Error(), "exceeds 5 bytes") {
			t.Fatalf("expected size error, got %v", err)
		}
		matches, err := filepath.Glob(filepath.Join(os.TempDir(), "javm-download-*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("incomplete downloads remain: %v", matches)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if _, err := downloadWithClient(ctx, server.Client(), server.URL+"/jdk.zip", 32); err == nil || !errorsIsContext(err) {
			t.Fatalf("expected cancellation error, got %v", err)
		}
	})
}

func TestSecureRedirectRejectsHTTP(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	if err := secureRedirect(req, nil); err == nil || !strings.Contains(err.Error(), "non-HTTPS") {
		t.Fatalf("expected insecure redirect error, got %v", err)
	}
}

func TestSecureRedirectLimitsRedirectChain(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	via := make([]*http.Request, 10)
	if err := secureRedirect(req, via); err == nil || !strings.Contains(err.Error(), "10 redirects") {
		t.Fatalf("expected redirect limit error, got %v", err)
	}
}

func errorsIsContext(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

type installPackagesClient struct {
	archivePath string
	checksum    string
}

type cancelAfterRead struct {
	reader io.Reader
	cancel context.CancelFunc
}

func (r *cancelAfterRead) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.cancel()
	return n, err
}

func (c installPackagesClient) GetPackagesContext(ctx context.Context, _, _, _, _ string) ([]discoapi.Package, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []discoapi.Package{{Id: "jdk", Distribution: "temurin", JavaVersion: "21.0.1"}}, nil
}

func (c installPackagesClient) GetPackageInfoContext(ctx context.Context, _ string) (*discoapi.PackageInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &discoapi.PackageInfo{
		DirectDownloadUri: "file://" + filepath.ToSlash(c.archivePath),
		Checksum:          c.checksum,
		ChecksumType:      "sha256",
	}, nil
}

func TestRunInstallVerifiesLocalArtifactAndPromotes(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	dst := filepath.Join(t.TempDir(), "jdk")
	version, err := runInstall(context.Background(), installPackagesClient{archivePath: archive, checksum: checksum}, "21", dst)
	if err != nil {
		t.Fatal(err)
	}
	if version != "temurin@21.0.1" {
		t.Fatalf("version = %q", version)
	}
	if err := assertJavaDistribution(dst, runtime.GOOS); err != nil {
		t.Fatal(err)
	}
}

func TestRunInstallChecksumFailureDoesNotCreateDestination(t *testing.T) {
	archive := makeZipArchive(t, []zipTestEntry{{name: javaArchivePath(), body: "java", mode: 0755}})
	dst := filepath.Join(t.TempDir(), "jdk")
	_, err := runInstall(context.Background(), installPackagesClient{archivePath: archive, checksum: strings.Repeat("0", sha256.Size*2)}, "21", dst)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum error, got %v", err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("destination exists after checksum failure: %v", err)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatalf("caller-owned file artifact was removed: %v", err)
	}
}

func assertUnsafeInstall(t *testing.T, archive, errorPart string) {
	t.Helper()
	parent := t.TempDir()
	dst := filepath.Join(parent, "jdk")
	external := filepath.Join(parent, "outside")
	if err := install(context.Background(), archive, dst); err == nil || !strings.Contains(err.Error(), errorPart) {
		t.Fatalf("expected error containing %q, got %v", errorPart, err)
	}
	if _, err := os.Lstat(dst); !os.IsNotExist(err) {
		t.Fatalf("destination exists after unsafe archive: %v", err)
	}
	if _, err := os.Lstat(external); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside staging: %v", err)
	}
}

func javaArchivePath() string {
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	return "jdk/bin/" + name
}

type zipTestEntry struct {
	name string
	body string
	mode os.FileMode
}

func makeZipArchive(t *testing.T, entries []zipTestEntry) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return archive
}

type tarTestEntry struct {
	name     string
	body     string
	linkname string
	typeflag byte
}

func makeTarGzArchive(t *testing.T, entries []tarTestEntry) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	writeTarEntries(t, tw, entries)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return archive
}

func makeTarXzArchive(t *testing.T, entries []tarTestEntry) string {
	t.Helper()
	archive := filepath.Join(t.TempDir(), "test.tar.xz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	xw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xw)
	writeTarEntries(t, tw, entries)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return archive
}

func writeTarEntries(t *testing.T, tw *tar.Writer, entries []tarTestEntry) {
	t.Helper()
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Linkname: entry.linkname, Typeflag: typeflag, Mode: 0755}
		if typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := io.WriteString(tw, entry.body); err != nil {
				t.Fatal(err)
			}
		}
	}
}
