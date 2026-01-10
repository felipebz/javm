package command

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestBinJavaRelocation(t *testing.T) {
	ok := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	nok := func(err error) {
		if err == nil {
			t.Fatal(err)
		}
	}
	dir, err := os.MkdirTemp("", "install_test")
	ok(err)
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
		for _, p := range scenario.paths {
			test1 := filepath.Join(dir, "test1")
			ok(touch(test1, p, "bin", scenario.bin))
			ok(normalizePathToBinJava(test1, scenario.os))
			ok(file(test1, scenario.prefix, "bin", scenario.bin))

			test2 := filepath.Join(dir, "test2")
			ok(touch(test2, "subdir", p, "bin", scenario.bin))
			ok(normalizePathToBinJava(test2, scenario.os))
			ok(file(test2, scenario.prefix, "bin", scenario.bin))

			test3 := filepath.Join(dir, "test3")
			ok(touch(test3, "subdir", "subdir", p, "bin", scenario.bin))
			ok(normalizePathToBinJava(test3, scenario.os))
			ok(file(test3, scenario.prefix, "bin", scenario.bin))

			test4 := filepath.Join(dir, "test4")
			ok(touch(test4, "file"))
			ok(touch(test4, "subdir", "subdir", p, "bin", scenario.bin))
			ok(normalizePathToBinJava(test4, scenario.os))
			ok(file(test4, scenario.prefix, "bin", scenario.bin))

			test5 := filepath.Join(dir, "test5")
			ok(touch(test5, "bin", "file"))
			nok(normalizePathToBinJava(test5, scenario.os))
			ok(file(test5, "bin", "file"))
		}
	}
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

func TestExpectedJavaPath(t *testing.T) {
	tests := []struct {
		dir      string
		os       string
		expected string
	}{
		{"/opt/java", "linux", filepath.Join("/opt/java", "bin", "java")},
		{"/opt/java", "darwin", filepath.Join("/opt/java", "Contents", "Home", "bin", "java")},
		{"C:\\Java", "windows", filepath.Join("C:\\Java", "bin", "java.exe")},
	}

	for _, tt := range tests {
		result := expectedJavaPath(tt.dir, tt.os)

		var expected string
		switch tt.os {
		case "darwin":
			expected = filepath.Join(tt.dir, "Contents", "Home", "bin", "java")
		case "windows":
			expected = filepath.Join(tt.dir, "bin", "java.exe")
		default:
			expected = filepath.Join(tt.dir, "bin", "java")
		}

		if result != expected {
			t.Errorf("expectedJavaPath(%q, %q) = %q, want %q", tt.dir, tt.os, result, expected)
		}
	}
}

func TestIsEmptyDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "empty_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	empty, err := isEmptyDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Error("Expected directory to be empty")
	}

	f, _ := os.CreateTemp(dir, "file")
	f.Close()

	empty, err = isEmptyDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Error("Expected directory to not be empty")
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

	destDir := t.TempDir()

	err = install(zipPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	if runtime.GOOS == "windows" {
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	} else {
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

	destDir := t.TempDir()

	err = install(tgzPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	if runtime.GOOS == "windows" {
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	} else {
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

	destDir := t.TempDir()

	err = install(txzPath, destDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Verify content
	var expectedPath string
	if runtime.GOOS == "windows" {
		expectedPath = filepath.Join(destDir, "bin", "java.exe")
	} else {
		expectedPath = filepath.Join(destDir, "bin", "java")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("%s not found after install", expectedPath)
	}
}
