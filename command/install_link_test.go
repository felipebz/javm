package command

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// symlinkChainEntries builds the nine-entry escape chain: every hop stores a
// target that path.Clean accepts as inside the archive root ("a1/.." cleans to
// "."), while the kernel resolves each hop one level further out because the
// referenced component is itself a symlink.
func symlinkChainEntries(hops int, tail ...tarTestEntry) []tarTestEntry {
	entries := []tarTestEntry{{name: "jdk/a1", linkname: ".", typeflag: tar.TypeSymlink}}
	for i := 2; i <= hops; i++ {
		entries = append(entries, tarTestEntry{
			name:     fmt.Sprintf("jdk/a%d", i),
			linkname: fmt.Sprintf("a%d/..", i-1),
			typeflag: tar.TypeSymlink,
		})
	}
	return append(entries, tail...)
}

func assertNoEscapingSymlinks(t *testing.T, root string) {
	t.Helper()
	physicalRoot := physicalPath(root)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		resolved, evalErr := filepath.EvalSymlinks(path)
		if evalErr != nil {
			return nil
		}
		if !pathWithinRoot(physicalRoot, resolved) {
			t.Errorf("symlink %s resolves outside the extraction root: %s", path, resolved)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// jdkHomeFor returns the directory that holds bin for goos, matching the
// layouts discovery accepts.
func jdkHomeFor(root, goos string) string {
	if goos == "darwin" {
		return filepath.Join(root, "Contents", "Home")
	}
	return root
}

func javaExecutableName(goos string) string {
	if goos == "windows" {
		return "java.exe"
	}
	return "java"
}

func TestArchiveRejectsSymlinkChainEscapingRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	chain := func() []tarTestEntry {
		return symlinkChainEntries(8, tarTestEntry{name: "jdk/leak", linkname: "a8/etc/passwd", typeflag: tar.TypeSymlink})
	}

	t.Run("tar.gz", func(t *testing.T) {
		assertUnsafeInstall(t, makeTarGzArchive(t, chain()), "escapes extraction root")
	})
	t.Run("tar.xz", func(t *testing.T) {
		assertUnsafeInstall(t, makeTarXzArchive(t, chain()), "escapes extraction root")
	})
	t.Run("zip", func(t *testing.T) {
		var entries []zipTestEntry
		for _, entry := range chain() {
			entries = append(entries, zipTestEntry{name: entry.name, body: entry.linkname, mode: os.ModeSymlink | 0777})
		}
		assertUnsafeInstall(t, makeZipArchive(t, entries), "escapes extraction root")
	})
}

func TestUntgzRejectsChainedSymlinksResolvingOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	root := t.TempDir()
	archive := makeTarGzArchive(t, symlinkChainEntries(8, tarTestEntry{name: "jdk/leak", linkname: "a8/etc/passwd", typeflag: tar.TypeSymlink}))

	err := untgz(context.Background(), archive, root, true)
	if err == nil || !strings.Contains(err.Error(), "escapes extraction root") {
		t.Fatalf("untgz() error = %v, want a containment error", err)
	}
	assertNoEscapingSymlinks(t, root)
}

func TestInstallRejectsJDKWhoseBinEscapesThroughSymlinkChain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "java"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	outsideRel := strings.TrimPrefix(filepath.ToSlash(outside), "/")
	entries := symlinkChainEntries(12, tarTestEntry{name: "jdk/bin", linkname: "a12/" + outsideRel, typeflag: tar.TypeSymlink})
	dst := filepath.Join(t.TempDir(), "jdk")

	err := install(context.Background(), makeTarGzArchive(t, entries), dst)
	if err == nil || !strings.Contains(err.Error(), "escapes extraction root") {
		t.Fatalf("install() error = %v, want a containment error", err)
	}
	if _, statErr := os.Lstat(dst); !os.IsNotExist(statErr) {
		t.Fatalf("destination exists after unsafe archive: %v", statErr)
	}
}

func TestHardlinkRejectsSymlinkedParentEscapingRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("TOPSECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "bin")); err != nil {
		t.Fatal(err)
	}
	state := newExtractionState(root, true, extractionLimits{maxEntries: 10, maxBytes: 1 << 20})

	err := state.makeHardlink(filepath.Join(root, "h"), filepath.Join(root, "bin", "secret.txt"))
	if err == nil || !strings.Contains(err.Error(), "escapes extraction root") {
		t.Fatalf("makeHardlink() error = %v, want a containment error", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "h")); !os.IsNotExist(err) {
		t.Fatalf("hardlink was created inside the root: %v", err)
	}
}

func TestHardlinkInsideRootIsPreserved(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "bin", "java")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := newExtractionState(root, true, extractionLimits{maxEntries: 10, maxBytes: 1 << 20})

	link := filepath.Join(root, "java-link")
	if err := state.makeHardlink(link, target); err != nil {
		t.Fatal(err)
	}
	linkInfo, err := os.Stat(link)
	if err != nil {
		t.Fatal(err)
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(linkInfo, targetInfo) {
		t.Fatal("in-root hardlink does not share the target inode")
	}
}

func TestResolveLinkTargetRejectsSymlinkCycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	root := t.TempDir()
	if err := os.Symlink("b", filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a", filepath.Join(root, "b")); err != nil {
		t.Fatal(err)
	}
	state := newExtractionState(root, true, extractionLimits{maxEntries: 10, maxBytes: 1 << 20})

	if _, err := state.resolveLinkTarget(root, "a"); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("resolveLinkTarget() error = %v, want a hop limit error", err)
	}
}

func TestInstallPreservesInRootSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	prefix := "jdk/"
	if runtime.GOOS == "darwin" {
		prefix = "jdk/Contents/Home/"
	}
	archive := makeTarGzArchive(t, []tarTestEntry{
		{name: prefix + "lib/libfoo.so.1", body: "library", typeflag: tar.TypeReg},
		{name: prefix + "lib/libfoo.so", linkname: "libfoo.so.1", typeflag: tar.TypeSymlink},
		{name: prefix + "lib/libjava.so", linkname: "libfoo.so", typeflag: tar.TypeSymlink},
		{name: prefix + "bin/java", body: "java", typeflag: tar.TypeReg},
	})
	dst := filepath.Join(t.TempDir(), "jdk")

	if err := install(context.Background(), archive, dst); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	home := jdkHomeFor(dst, runtime.GOOS)
	resolved, err := filepath.EvalSymlinks(filepath.Join(home, "lib", "libjava.so"))
	if err != nil {
		t.Fatal(err)
	}
	if !pathWithinRoot(physicalPath(dst), resolved) {
		t.Fatalf("in-root symlink resolved outside the JDK: %s", resolved)
	}
	if err := assertJavaDistribution(dst, runtime.GOOS); err != nil {
		t.Fatal(err)
	}
}

// TestPrepareStagedJDKAcceptsMacOSBundle checks the darwin branch on every
// platform: a Contents/Home bundle with in-root symlinks is normalized and
// accepted, so the containment checks do not reject real bundle layouts.
func TestPrepareStagedJDKAcceptsMacOSBundle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	transactionDir := t.TempDir()
	extractRoot := filepath.Join(transactionDir, "extract")
	home := filepath.Join(extractRoot, "jdk-21.jdk", "Contents", "Home")
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "bin", "java"), []byte("java"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "lib", "libjli.dylib"), []byte("lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("libjli.dylib", filepath.Join(home, "lib", "libjli-current.dylib")); err != nil {
		t.Fatal(err)
	}

	readyRoot, err := prepareStagedJDK(context.Background(), extractRoot, transactionDir, "darwin")
	if err != nil {
		t.Fatalf("prepareStagedJDK() error = %v, want the bundle to be accepted", err)
	}
	if err := assertJavaDistribution(readyRoot, "darwin"); err != nil {
		t.Fatalf("assertJavaDistribution() error = %v, want the bundle to be accepted", err)
	}
}

func TestJavaDistributionRejectsSymlinkedBinResolvingOutside(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	outside := t.TempDir()
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			javaName := javaExecutableName(goos)
			if err := os.WriteFile(filepath.Join(outside, javaName), []byte("java"), 0o755); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			home := jdkHomeFor(root, goos)
			if err := os.MkdirAll(home, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(home, "bin")); err != nil {
				t.Fatal(err)
			}

			if err := assertJavaDistribution(root, goos); err == nil || !strings.Contains(err.Error(), "outside the installation root") {
				t.Fatalf("assertJavaDistribution() error = %v, want a containment error", err)
			}
		})
	}
}

// TestJavaDistributionAcceptsRootReachedThroughSymlink covers the macOS layout of
// temporary directories, where /var is a symlink to /private/var, so the managed
// root is reached through a symlinked component.
func TestJavaDistributionAcceptsRootReachedThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks requires privileges on Windows")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Fatal(err)
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			root := filepath.Join(link, goos)
			bin := filepath.Join(jdkHomeFor(root, goos), "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(bin, javaExecutableName(goos)), []byte("java"), 0o755); err != nil {
				t.Fatal(err)
			}

			if err := assertJavaDistribution(root, goos); err != nil {
				t.Fatalf("assertJavaDistribution() error = %v, want the JDK to be accepted", err)
			}
		})
	}
}
