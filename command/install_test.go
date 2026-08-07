package command

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestPrepareStagedJDK(t *testing.T) {
	for _, scenario := range []struct {
		os     string
		bin    string
		prefix string
		paths  []string
	}{
		{
			os:     "linux",
			bin:    "java",
			prefix: "",
			paths:  []string{""},
		},
		{
			os:     "darwin",
			bin:    "java",
			prefix: filepath.Join("Contents", "Home"),
			paths: []string{
				"",
				filepath.Join("Home"),
				filepath.Join("Contents", "Home"),
			},
		},
		{
			os:     "windows",
			bin:    "java.exe",
			prefix: "",
			paths:  []string{""},
		},
	} {
		for index, p := range scenario.paths {
			t.Run(fmt.Sprintf("%s-layout-%d", scenario.os, index), func(t *testing.T) {
				transactionDir := t.TempDir()
				extractRoot := filepath.Join(transactionDir, "extract")
				if err := touch(extractRoot, "nested", p, "bin", scenario.bin); err != nil {
					t.Fatal(err)
				}
				readyRoot, err := prepareStagedJDK(context.Background(), extractRoot, transactionDir, scenario.os)
				if err != nil {
					t.Fatal(err)
				}
				if err := file(readyRoot, scenario.prefix, "bin", scenario.bin); err != nil {
					t.Fatal(err)
				}
			})
		}
	}

	t.Run("missing java", func(t *testing.T) {
		transactionDir := t.TempDir()
		extractRoot := filepath.Join(transactionDir, "extract")
		if err := touch(extractRoot, "bin", "not-java"); err != nil {
			t.Fatal(err)
		}
		if _, err := prepareStagedJDK(context.Background(), extractRoot, transactionDir, runtime.GOOS); err == nil {
			t.Fatal("expected invalid staged JDK to fail")
		}
	})
}

func touch(path ...string) error {
	filename := filepath.Join(path...)
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filename, nil, 0755); err != nil {
		return err
	}
	return nil
}

func file(path ...string) error {
	if _, err := os.Stat(filepath.Join(path...)); os.IsNotExist(err) {
		return err
	}
	return nil
}

func TestValidateChecksum(t *testing.T) {
	content := []byte("test content")
	tmpfile, err := os.CreateTemp("", "checksum_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// SHA256 of "test content"
	expectedSha256 := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"
	err = validateChecksum(tmpfile.Name(), expectedSha256, "sha256")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	// SHA1 of "test content"
	expectedSha1 := "1eebdf4fdc9fc7bf283031b93f9aef3338de9052"
	err = validateChecksum(tmpfile.Name(), expectedSha1, "sha1")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	err = validateChecksum(tmpfile.Name(), "wrongchecksum", "sha256")
	if err == nil {
		t.Error("Expected error for mismatching checksum, got nil")
	}

	err = validateChecksum(tmpfile.Name(), expectedSha256, "md5")
	if err == nil {
		t.Error("Expected error for unsupported algorithm, got nil")
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"file.tar.gz", ".tar.gz"},
		{"file.tar.xz", ".tar.xz"},
		{"file.zip", ".zip"},
		{"file.txt", ".txt"},
		{"path/to/file.tar.gz", ".tar.gz"},
	}

	for _, tt := range tests {
		result := getFileExtension(tt.filename)
		if result != tt.expected {
			t.Errorf("getFileExtension(%q) = %q, want %q", tt.filename, result, tt.expected)
		}
	}
}

func TestInstallZip(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	// Create bin/java inside zip to match expectation for valid java home
	var binJava string
	if runtime.GOOS == "windows" {
		binJava = "jdk-test/bin/java.exe"
	} else {
		binJava = "jdk-test/bin/java"
	}

	iw, err := w.Create(binJava)
	if err != nil {
		t.Fatal(err)
	}
	_, err = iw.Write([]byte("mock java"))
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	destDir := filepath.Join(t.TempDir(), "jdk")

	err = install(context.Background(), zipPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	switch runtime.GOOS {
	case "windows":
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	case "darwin":
		expectedPath = filepath.Join(destDir, "Contents", "Home", "bin", "java")
	default:
		expectedPath = filepath.Join(destDir, "bin", "java")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("%s not found after install", expectedPath)
	}
}

func TestInstallTgz(t *testing.T) {
	tgzPath := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(tgzPath)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Create bin/java inside tar to match expectation for valid java home
	var binJava string
	if runtime.GOOS == "windows" {
		binJava = "jdk-test/bin/java.exe"
	} else {
		binJava = "jdk-test/bin/java"
	}

	header := &tar.Header{
		Name: binJava,
		Mode: 0755,
		Size: int64(len("mock java")),
	}

	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("mock java")); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	destDir := filepath.Join(t.TempDir(), "jdk")

	err = install(context.Background(), tgzPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	switch runtime.GOOS {
	case "windows":
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	case "darwin":
		expectedPath = filepath.Join(destDir, "Contents", "Home", "bin", "java")
	default:
		expectedPath = filepath.Join(destDir, "bin", "java")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("%s not found after install", expectedPath)
	}
}

func TestInstallTxz(t *testing.T) {
	txzPath := filepath.Join(t.TempDir(), "test.tar.xz")
	f, err := os.Create(txzPath)
	if err != nil {
		t.Fatal(err)
	}

	xw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xw)

	// Create bin/java inside tar to match expectation for valid java home
	var binJava string
	if runtime.GOOS == "windows" {
		binJava = "jdk-test/bin/java.exe"
	} else {
		binJava = "jdk-test/bin/java"
	}

	header := &tar.Header{
		Name: binJava,
		Mode: 0755,
		Size: int64(len("mock java")),
	}

	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("mock java")); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	destDir := filepath.Join(t.TempDir(), "jdk")

	err = install(context.Background(), txzPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	switch runtime.GOOS {
	case "windows":
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	case "darwin":
		expectedPath = filepath.Join(destDir, "Contents", "Home", "bin", "java")
	default:
		expectedPath = filepath.Join(destDir, "bin", "java")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("%s not found after install", expectedPath)
	}
}
