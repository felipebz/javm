package command

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipebz/javm/discovery"
	"github.com/ulikunitz/xz"
)

const (
	maxArchiveEntries = 100_000
	maxExtractedSize  = int64(4 << 30) // 4 GiB uncompressed
	maxSymlinkSize    = int64(4 << 10)
	// maxSymlinkHops bounds a link target resolved against the extraction tree.
	maxSymlinkHops = 64
)

type extractionLimits struct {
	maxEntries int
	maxBytes   int64
}

type archiveExtractor func(context.Context, string, string) error

func install(ctx context.Context, file string, dst string) (err error) {
	return installWithExtractor(ctx, file, dst, extractArchive)
}

func installWithExtractor(ctx context.Context, file string, dst string, extract archiveExtractor) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("resolve installation destination: %w", err)
	}
	parent := filepath.Dir(dst)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("create installation parent: %w", err)
	}

	if _, statErr := os.Lstat(dst); statErr == nil {
		return fmt.Errorf("installation destination %q already exists; refusing to replace it", dst)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect installation destination: %w", statErr)
	}

	transactionDir, err := os.MkdirTemp(parent, "."+filepath.Base(dst)+".staging-*")
	if err != nil {
		return fmt.Errorf("create installation staging directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(transactionDir); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove installation staging directory: %w", removeErr))
		}
	}()

	extractRoot := filepath.Join(transactionDir, "extract")
	if err := os.Mkdir(extractRoot, 0700); err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	if err := extract(ctx, file, extractRoot); err != nil {
		return fmt.Errorf("extract archive into staging: %w; installation rolled back", err)
	}

	readyRoot, err := prepareStagedJDK(ctx, extractRoot, transactionDir, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("validate staged JDK: %w; installation rolled back", err)
	}
	if err := assertJavaDistribution(readyRoot, runtime.GOOS); err != nil {
		return fmt.Errorf("validate staged JDK: %w; installation rolled back", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("installation canceled before promotion: %w; installation rolled back", err)
	}

	if err := promoteNoReplace(readyRoot, dst); err != nil {
		return fmt.Errorf("promote staged JDK to %q: %w; installation rolled back", dst, err)
	}
	return nil
}

func extractArchive(ctx context.Context, src, dst string) error {
	switch getFileExtension(src) {
	case ".zip":
		return installFromZip(ctx, src, dst)
	case ".tar.gz":
		return installFromTgz(ctx, src, dst)
	case ".tar.xz":
		return installFromTgx(ctx, src, dst)
	default:
		return fmt.Errorf("unsupported file type: %s", src)
	}
}

func prepareStagedJDK(ctx context.Context, extractRoot, transactionDir, goos string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := assertJavaDistribution(extractRoot, goos); err == nil {
		return extractRoot, nil
	}

	javaName := "java"
	if goos == "windows" {
		javaName = "java.exe"
	}
	var javaPath string
	err := filepath.WalkDir(extractRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 && entry.IsDir() {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == javaName && filepath.Base(filepath.Dir(current)) == "bin" {
			javaPath = current
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("locate bin/%s: %w", javaName, err)
	}
	if javaPath == "" {
		return "", fmt.Errorf("bin/%s was not found in the archive", javaName)
	}

	homeDir := filepath.Dir(filepath.Dir(javaPath))
	if goos != "darwin" {
		return homeDir, nil
	}
	if filepath.Base(homeDir) == "Home" && filepath.Base(filepath.Dir(homeDir)) == "Contents" {
		return filepath.Dir(filepath.Dir(homeDir)), nil
	}

	readyRoot := filepath.Join(transactionDir, "ready")
	homeTarget := filepath.Join(readyRoot, "Contents", "Home")
	if err := os.MkdirAll(filepath.Dir(homeTarget), 0700); err != nil {
		return "", fmt.Errorf("create macOS JDK layout: %w", err)
	}
	if err := os.Rename(homeDir, homeTarget); err != nil {
		return "", fmt.Errorf("normalize macOS JDK layout: %w", err)
	}
	return readyRoot, nil
}

func assertJavaDistribution(dir string, goos string) error {
	javaPath := filepath.FromSlash(discovery.ExpectedJavaPath(dir, goos))
	if err := assertNoEscapingComponents(dir, javaPath); err != nil {
		return err
	}
	info, err := os.Lstat(javaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s wasn't found", javaPath)
		}
		return fmt.Errorf("inspect %s: %w", javaPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(javaPath)
		if err != nil {
			return fmt.Errorf("resolve Java executable symlink: %w", err)
		}
		if !pathWithinRoot(dir, resolved) {
			return fmt.Errorf("Java executable symlink escapes installation root: %s", javaPath)
		}
		info, err = os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("inspect Java executable target: %w", err)
		}
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Java executable is not a regular file: %s", javaPath)
	}
	return nil
}

// assertNoEscapingComponents rejects a Java executable path whose intermediate
// directories are symlinks resolving outside root. The kernel follows those
// components, so the JDK shape check would otherwise accept a tree whose bin
// directory lives outside the managed JDK, and a missing component is left for
// the caller to report.
func assertNoEscapingComponents(root, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s is outside %s", target, root)
	}
	components := strings.Split(rel, string(os.PathSeparator))
	if len(components) < 2 {
		return nil
	}
	physicalRoot := root
	if evaluated, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		physicalRoot = evaluated
	}
	current := root
	for _, component := range components[:len(components)-1] {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil
			}
			return fmt.Errorf("inspect %s: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, evalErr := filepath.EvalSymlinks(current)
		if evalErr != nil {
			return fmt.Errorf("resolve %s: %w", current, evalErr)
		}
		if !pathWithinRoot(physicalRoot, resolved) {
			return fmt.Errorf("%s resolves outside the installation root: %s", current, resolved)
		}
	}
	return nil
}

func installFromTgz(ctx context.Context, src string, dst string) error {
	loggerFromContext(ctx).Debug("Extracting " + src + " to " + dst)
	return untgz(ctx, src, dst, true)
}

func untgz(ctx context.Context, src string, dst string, strip bool) (err error) {
	gzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := gzFile.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close gzip archive: %w", closeErr))
		}
	}()
	gzr, err := gzip.NewReader(&contextReader{ctx: ctx, reader: gzFile})
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := gzr.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close gzip stream: %w", closeErr))
		}
	}()
	return extractTar(ctx, gzr, dst, strip)
}

func extractTar(ctx context.Context, r io.Reader, dst string, strip bool) error {
	return extractTarWithLimits(ctx, r, dst, strip, extractionLimits{maxEntries: maxArchiveEntries, maxBytes: maxExtractedSize})
}

func extractTarWithLimits(ctx context.Context, r io.Reader, dst string, strip bool, limits extractionLimits) error {
	tr := tar.NewReader(&contextReader{ctx: ctx, reader: r})
	state := newExtractionState(dst, strip, limits)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		rel, skip, err := state.entryPath(header.Name)
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		if err := state.addEntry(header.Size); err != nil {
			return err
		}
		target := filepath.Join(state.root, filepath.FromSlash(rel))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := state.makeDir(target); err != nil {
				return err
			}
		case tar.TypeReg:
			if header.Size < 0 {
				return fmt.Errorf("archive entry %q has a negative size", header.Name)
			}
			if err := state.writeFile(target, os.FileMode(header.Mode), tr, header.Size); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := state.makeSymlink(rel, target, header.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			linkRel, skip, err := state.entryPathForLink(header.Linkname)
			if err != nil || skip {
				return fmt.Errorf("unsafe hardlink %q -> %q: %w", header.Name, header.Linkname, err)
			}
			if err := state.makeHardlink(target, filepath.Join(state.root, filepath.FromSlash(linkRel))); err != nil {
				return err
			}
		default:
			return fmt.Errorf("archive contains unsupported entry type %d at %q", header.Typeflag, header.Name)
		}
	}
}

func installFromTgx(ctx context.Context, src string, dst string) error {
	loggerFromContext(ctx).Debug("Extracting " + src + " to " + dst)
	return untgx(ctx, src, dst, true)
}

func untgx(ctx context.Context, src string, dst string, strip bool) (err error) {
	xzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := xzFile.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close xz archive: %w", closeErr))
		}
	}()
	xzr, err := xz.NewReader(&contextReader{ctx: ctx, reader: xzFile})
	if err != nil {
		return err
	}
	return extractTar(ctx, xzr, dst, strip)
}

func installFromZip(ctx context.Context, src string, dst string) error {
	loggerFromContext(ctx).Debug("Extracting " + src + " to " + dst)
	return unzip(ctx, src, dst, true)
}

func unzip(ctx context.Context, src string, dst string, strip bool) error {
	return unzipWithLimits(ctx, src, dst, strip, extractionLimits{maxEntries: maxArchiveEntries, maxBytes: maxExtractedSize})
}

func unzipWithLimits(ctx context.Context, src string, dst string, strip bool, limits extractionLimits) (err error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := r.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close zip archive: %w", closeErr))
		}
	}()
	state := newExtractionState(dst, strip, limits)
	for _, entry := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, skip, err := state.entryPath(entry.Name)
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		if entry.UncompressedSize64 > ^uint64(0)>>1 {
			return fmt.Errorf("archive entry %q is too large", entry.Name)
		}
		if err := state.addEntry(int64(entry.UncompressedSize64)); err != nil {
			return err
		}
		target := filepath.Join(state.root, filepath.FromSlash(rel))
		mode := entry.Mode()
		if isArchiveDirectory(entry.Name, entry.FileInfo()) {
			if err := state.makeDir(target); err != nil {
				return err
			}
			continue
		}

		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open archive entry %q: %w", entry.Name, err)
		}
		if mode&os.ModeSymlink != 0 {
			linkTarget, readErr := io.ReadAll(io.LimitReader(&contextReader{ctx: ctx, reader: rc}, maxSymlinkSize+1))
			closeErr := rc.Close()
			if readErr != nil {
				return fmt.Errorf("read symlink %q: %w", entry.Name, readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close symlink %q: %w", entry.Name, closeErr)
			}
			if int64(len(linkTarget)) > maxSymlinkSize {
				return fmt.Errorf("symlink target is too long at %q", entry.Name)
			}
			if err := state.makeSymlink(rel, target, string(linkTarget)); err != nil {
				return err
			}
			continue
		}
		if !mode.IsRegular() {
			return errors.Join(
				fmt.Errorf("archive contains unsupported entry at %q", entry.Name),
				rc.Close(),
			)
		}
		writeErr := state.writeFile(target, mode, &contextReader{ctx: ctx, reader: rc}, int64(entry.UncompressedSize64))
		closeErr := rc.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return fmt.Errorf("close archive entry %q: %w", entry.Name, closeErr)
		}
	}
	return nil
}

type extractionState struct {
	root       string
	strip      bool
	rootPrefix string
	limits     extractionLimits
	entries    int
	bytes      int64
	seen       map[string]struct{}
}

func newExtractionState(root string, strip bool, limits extractionLimits) *extractionState {
	return &extractionState{
		root:   filepath.Clean(root),
		strip:  strip,
		limits: limits,
		seen:   make(map[string]struct{}),
	}
}

func (s *extractionState) entryPath(name string) (string, bool, error) {
	clean, err := safeArchiveName(name)
	if err != nil {
		return "", false, fmt.Errorf("unsafe archive path %q: %w", name, err)
	}
	if clean == "." {
		return "", true, nil
	}
	if s.strip {
		parts := strings.Split(clean, "/")
		if s.rootPrefix == "" {
			s.rootPrefix = parts[0]
		}
		if parts[0] != s.rootPrefix {
			return "", false, fmt.Errorf("archive contains multiple roots %q and %q", s.rootPrefix, parts[0])
		}
		if len(parts) == 1 {
			return "", true, nil
		}
		clean = strings.Join(parts[1:], "/")
	}
	if _, exists := s.seen[clean]; exists {
		return "", false, fmt.Errorf("archive contains duplicate entry %q", clean)
	}
	s.seen[clean] = struct{}{}
	return clean, false, nil
}

func (s *extractionState) entryPathForLink(name string) (string, bool, error) {
	clean, err := safeArchiveName(name)
	if err != nil {
		return "", false, err
	}
	if !s.strip {
		return clean, clean == ".", nil
	}
	parts := strings.Split(clean, "/")
	if len(parts) < 2 || parts[0] != s.rootPrefix {
		return "", false, fmt.Errorf("hardlink target is outside archive root")
	}
	return strings.Join(parts[1:], "/"), false, nil
}

func safeArchiveName(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("empty path or NUL byte")
	}
	name = normalizeArchiveSeparators(name)
	if path.IsAbs(name) || hasWindowsVolume(name) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path traversal is not allowed")
	}
	return clean, nil
}

func normalizeArchiveSeparators(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

func isArchiveDirectory(name string, info fs.FileInfo) bool {
	return info.IsDir() || strings.HasSuffix(normalizeArchiveSeparators(name), "/")
}

func hasWindowsVolume(name string) bool {
	return len(name) >= 2 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':'
}

func (s *extractionState) addEntry(size int64) error {
	if size < 0 {
		return fmt.Errorf("archive entry has a negative size")
	}
	s.entries++
	if s.entries > s.limits.maxEntries {
		return fmt.Errorf("archive exceeds %d entries", s.limits.maxEntries)
	}
	if size > s.limits.maxBytes-s.bytes {
		return fmt.Errorf("archive exceeds %d extracted bytes", s.limits.maxBytes)
	}
	s.bytes += size
	return nil
}

func (s *extractionState) makeDir(target string) error {
	if err := s.ensureParents(target); err != nil {
		return err
	}
	info, err := os.Lstat(target)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive directory collides with non-directory %q", target)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.Mkdir(target, 0755); err != nil {
		return fmt.Errorf("create archive directory %q: %w", target, err)
	}
	return nil
}

func (s *extractionState) writeFile(target string, mode os.FileMode, source io.Reader, expectedSize int64) error {
	if err := s.ensureParents(target); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm()|0600)
	if err != nil {
		return fmt.Errorf("create archive file %q: %w", target, err)
	}
	written, copyErr := io.CopyN(f, source, expectedSize)
	if copyErr == nil {
		var extra [1]byte
		n, readErr := source.Read(extra[:])
		if n != 0 || (readErr != nil && readErr != io.EOF) {
			copyErr = fmt.Errorf("archive entry exceeds its declared size")
		}
	}
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("write archive file %q after %d bytes: %w", target, written, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close archive file %q: %w", target, closeErr)
	}
	return nil
}

func (s *extractionState) makeSymlink(rel, target, linkTarget string) error {
	linkTarget = normalizeArchiveSeparators(linkTarget)
	if linkTarget == "" || strings.ContainsRune(linkTarget, '\x00') || path.IsAbs(linkTarget) || hasWindowsVolume(linkTarget) {
		return fmt.Errorf("archive contains unsafe symlink %q -> %q", rel, linkTarget)
	}
	resolved := path.Clean(path.Join(path.Dir(rel), linkTarget))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return fmt.Errorf("archive contains unsafe symlink %q -> %q", rel, linkTarget)
	}
	if err := s.ensureParents(target); err != nil {
		return err
	}
	if _, err := s.resolveLinkTarget(filepath.Dir(target), linkTarget); err != nil {
		return fmt.Errorf("archive contains unsafe symlink %q -> %q: %w", rel, linkTarget, err)
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("archive symlink collides with existing path %q", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Symlink(filepath.FromSlash(linkTarget), target); err != nil {
		return fmt.Errorf("create archive symlink %q: %w", rel, err)
	}
	return nil
}

func (s *extractionState) makeHardlink(target, linkTarget string) error {
	if err := s.ensureParents(target); err != nil {
		return err
	}
	info, err := os.Lstat(linkTarget)
	if err != nil {
		return fmt.Errorf("hardlink target is missing or appears after the link: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("hardlink target is not a regular file: %q", linkTarget)
	}
	// Lstat and link(2) follow the intermediate components of the target, so a
	// directory that is a symlink to outside the tree would leak an outside
	// file into the extracted JDK.
	if _, err := s.resolveLinkTarget(filepath.Dir(linkTarget), filepath.Base(linkTarget)); err != nil {
		return fmt.Errorf("archive contains unsafe hardlink %q -> %q: %w", target, linkTarget, err)
	}
	if err := os.Link(linkTarget, target); err != nil {
		return fmt.Errorf("create archive hardlink %q: %w", target, err)
	}
	return nil
}

// resolveLinkTarget resolves target the way the kernel resolves it when the
// extracted entry is used: starting at startDir and following the symlinks the
// extraction tree already contains. Lexical validation alone misses a relative
// target such as "a1/.." whose link component is a symlink pointing at the
// extraction root. The result is rejected whenever it leaves the root, and the
// number of followed links is bounded so a link cycle cannot stall extraction.
func (s *extractionState) resolveLinkTarget(startDir, target string) (string, error) {
	root := s.root
	if evaluated, err := filepath.EvalSymlinks(root); err == nil {
		root = evaluated
	}
	current, err := filepath.EvalSymlinks(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve link directory %q: %w", startDir, err)
	}
	if !pathWithinRoot(root, current) {
		return "", fmt.Errorf("link target escapes extraction root: %q", target)
	}

	components := strings.Split(filepath.ToSlash(target), "/")
	hops := 0
	for len(components) > 0 {
		component := components[0]
		components = components[1:]
		switch component {
		case "", ".":
			continue
		case "..":
			current = filepath.Dir(current)
		default:
			next := filepath.Join(current, filepath.FromSlash(component))
			info, statErr := os.Lstat(next)
			if statErr == nil && info.Mode()&os.ModeSymlink != 0 {
				hops++
				if hops > maxSymlinkHops {
					return "", fmt.Errorf("link target follows more than %d symlinks: %q", maxSymlinkHops, target)
				}
				stored, readErr := os.Readlink(next)
				if readErr != nil {
					return "", fmt.Errorf("read existing symlink %q: %w", next, readErr)
				}
				stored = normalizeArchiveSeparators(stored)
				if stored == "" || filepath.IsAbs(stored) || hasWindowsVolume(stored) {
					return "", fmt.Errorf("link target escapes extraction root: %q", target)
				}
				components = append(strings.Split(filepath.ToSlash(stored), "/"), components...)
				continue
			}
			if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
				return "", fmt.Errorf("inspect link target %q: %w", next, statErr)
			}
			current = next
		}
		if !pathWithinRoot(root, current) {
			return "", fmt.Errorf("link target escapes extraction root: %q", target)
		}
	}
	return current, nil
}

func (s *extractionState) ensureParents(target string) error {
	parent := filepath.Dir(target)
	rel, err := filepath.Rel(s.root, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("archive target escapes extraction root: %q", target)
	}
	current := s.root
	if rel == "." {
		return nil
	}
	for component := range strings.SplitSeq(rel, string(os.PathSeparator)) {
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			if err := os.Mkdir(current, 0755); err != nil {
				return fmt.Errorf("create archive parent %q: %w", current, err)
			}
			continue
		}
		if statErr != nil {
			return fmt.Errorf("inspect archive parent %q: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("archive path traverses symlink or non-directory %q", current)
		}
	}
	return nil
}

func pathWithinRoot(root, candidate string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}
