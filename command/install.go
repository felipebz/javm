package command

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/command/fileiter"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/semver"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ulikunitz/xz"
)

func NewInstallCommand(client PackagesWithInfoClient) *cobra.Command {
	var customInstallDestination string

	cmd := &cobra.Command{
		Use:   "install [version to install]",
		Short: "Download and install JDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				ver = cfg.ReadJavaVersion()
				if ver == "" {
					return pflag.ErrHelp
				}
			} else {
				ver = args[0]
			}
			ver, err := runInstall(client, ver, customInstallDestination)
			if err != nil {
				return err
			}
			if customInstallDestination == "" {
				if err := LinkLatest(); err != nil {
					return err
				}
				// TODO change to call the "use" command after it's refactored
				//return use(ver)
				return nil
			} else {
				return nil
			}
		},
		Example: "  javm install 1.8\n" +
			"  javm install ~1.8.73 # same as \">=1.8.73 <1.9.0\"\n" +
			"  javm install 1.8.73=dmg+http://.../jdk-9-ea+110_osx-x64_bin.dmg",
	}
	cmd.Flags().StringVarP(&customInstallDestination, "output", "o", "",
		"Custom destination (any JDK outside of $JAVM_HOME/jdk is considered to be unmanaged, i.e. not available to javm ls, use, etc. (unless `javm link`ed))")
	return cmd
}

func runInstall(client PackagesWithInfoClient, selector string, dst string) (string, error) {
	var releaseMap map[*semver.Version]string
	var ver *semver.Version
	var err error
	var expectedChecksum string
	var checksumType string

	rng, err := semver.ParseRange(selector)
	if err != nil {
		return "", err
	}
	distribution := rng.Qualifier
	if distribution == "" {
		var derr error
		distribution, derr = cfg.EffectiveValue("java.default_distribution")
		if derr != nil {
			return "", derr
		}
	}
	packageIndex, err := makePackageIndex(client, runtime.GOOS, runtime.GOARCH, distribution)
	if err != nil {
		return "", err
	}
	sort.Sort(sort.Reverse(semver.VersionSlice(packageIndex.Sorted)))
	for _, v := range packageIndex.Sorted {
		if rng.Contains(v) {
			ver = v
			packageInfo, err := client.GetPackageInfo(packageIndex.ByVersion[ver].Id)
			if err != nil {
				return "", err
			}

			downloadUri := packageInfo.DirectDownloadUri
			expectedChecksum = packageInfo.Checksum
			checksumType = packageInfo.ChecksumType
			releaseMap = map[*semver.Version]string{ver: downloadUri}
			break
		}
	}
	if ver == nil {
		tt := make([]string, len(packageIndex.Sorted))
		for i, v := range packageIndex.Sorted {
			tt[i] = v.String()
		}
		return "", errors.New("No compatible version found for " + selector +
			"\nValid install targets: " + strings.Join(tt, ", "))
	}

	// check whether requested version is already installed
	if dst == "" {
		local, err := Ls()
		if err != nil {
			return "", err
		}
		if slices.ContainsFunc(local, func(jdk discovery.JDK) bool {
			v, _ := semver.ParseVersion(jdk.Version)
			vID, _ := semver.ParseVersion(jdk.Identifier)
			return (v != nil && v.Equals(ver)) || (vID != nil && vID.Equals(ver))
		}) {
			return ver.String(), nil
		}
	}
	url := releaseMap[ver]
	if dst == "" {
		dst = filepath.Join(cfg.Dir(), "jdk", ver.String())
	} else {
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			if err == nil { // dst exists
				if empty, _ := isEmptyDir(dst); !empty {
					err = fmt.Errorf("\"%s\" is not empty", dst)
				}
			} // or is inaccessible
			if err != nil {
				return "", err
			}
		}
	}
	var file string
	var deleteFileWhenFinnished bool
	if after, ok := strings.CutPrefix(url, "file://"); ok {
		file = after
		if runtime.GOOS == "windows" {
			// file:///C:/path/...
			file = strings.Replace(strings.TrimPrefix(file, "/"), "/", "\\", -1)
		}
	} else {
		log.Info("Downloading ", ver)
		log.Debug("URL: ", url)
		file, err = download(url)
		if err != nil {
			return "", err
		}
		deleteFileWhenFinnished = true
		// Validate checksum when provided by DiscoAPI
		if expectedChecksum != "" && checksumType != "" {
			if err := validateChecksum(file, expectedChecksum, checksumType); err != nil {
				os.Remove(file)
				return "", err
			}
		} else {
			log.Warn("No checksum provided by DiscoAPI for this artifact; skipping integrity verification")
		}
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		err = install(file, dst)
	default:
		err = errors.New(runtime.GOOS + " OS is not supported")
	}
	if err == nil && deleteFileWhenFinnished {
		os.Remove(file)
	}
	return ver.String(), err
}

func isEmptyDir(name string) (bool, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

type RedirectTracer struct {
	Transport http.RoundTripper
}

func (self RedirectTracer) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	transport := self.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err = transport.RoundTrip(req)
	if err != nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		log.Debug("Following ", resp.StatusCode, " redirect to ", resp.Header.Get("Location"))
	}
	return
}

func download(url string) (file string, err error) {
	if !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("insecure download URL: only HTTPS is allowed, got: %s", url)
	}

	ext := getFileExtension(url)
	tmp, err := os.CreateTemp("", "javm-d-*"+ext)
	if err != nil {
		return
	}
	defer tmp.Close()

	file = tmp.Name()
	log.Debug("Saving ", url, " to ", file)
	client := http.Client{
		Transport: RedirectTracer{},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL == nil || strings.ToLower(req.URL.Scheme) != "https" {
				return fmt.Errorf("insecure redirect to non-HTTPS URL: %v", req.URL)
			}
			return nil
		},
	}
	res, err := client.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()

	bar := progressbar.DefaultBytes(
		res.ContentLength,
		"downloading",
	)
	_, err = io.Copy(io.MultiWriter(tmp, bar), res.Body)
	if err != nil {
		return
	}
	return
}

func validateChecksum(path string, expected string, algo string) error {
	algo = strings.ToLower(strings.TrimSpace(algo))
	expected = strings.ToLower(strings.TrimSpace(expected))

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var h io.Writer
	var sumFunc func() string
	switch algo {
	case "sha256":
		sha := sha256.New()
		h = sha
		sumFunc = func() string { return fmt.Sprintf("%x", sha.Sum(nil)) }
	case "sha1":
		sha := sha1.New()
		h = sha
		sumFunc = func() string { return fmt.Sprintf("%x", sha.Sum(nil)) }
	default:
		return fmt.Errorf("unsupported checksum type: %s", algo)
	}

	if _, err := io.Copy(h.(io.Writer), f); err != nil { // write file to hash
		return err
	}
	actual := sumFunc()
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s (%s), got %s", expected, algo, actual)
	}
	log.Debugf("Checksum verified with %s: %s", algo, actual)
	return nil
}

func getFileExtension(file string) string {
	if strings.HasSuffix(file, ".tar.gz") {
		return ".tar.gz"
	}
	if strings.HasSuffix(file, ".tar.xz") {
		return ".tar.xz"
	}
	return filepath.Ext(file)
}

func install(file string, dst string) (err error) {
	ext := getFileExtension(file)
	switch ext {
	case ".zip":
		err = installFromZip(file, dst)
	case ".tar.gz":
		err = installFromTgz(file, dst)
	case ".tar.xz":
		err = installFromTgx(file, dst)
	default:
		return errors.New("Unsupported file type: " + file)
	}
	if err == nil {
		err = normalizePathToBinJava(dst, runtime.GOOS)
	}
	if err != nil {
		os.RemoveAll(dst)
	}
	return
}

// **/{Contents/Home,Home,}bin/java -> <dir>/Contents/Home/bin/java
func normalizePathToBinJava(dir string, goos string) error {
	dir = filepath.Clean(dir)
	if _, err := os.Stat(expectedJavaPath(dir, goos)); os.IsNotExist(err) {
		java := "java"
		if goos == "windows" {
			java = "java.exe"
		}
		var javaPath string
		for it := fileiter.New(dir, fileiter.BreadthFirst()); it.Next(); {
			if err := it.Err(); err != nil {
				return err
			}
			if !it.IsDir() && filepath.Base(it.Dir()) == "bin" && it.Name() == java {
				javaPath = filepath.Join(it.Dir(), it.Name())
				break
			}
		}
		if javaPath != "" {
			log.Debugf("Found %s", javaPath)
			tmp := dir + "~"
			javaPath = strings.Replace(javaPath, dir, tmp, 1)
			log.Debugf("Moving %s to %s", dir, tmp)
			if err := os.Rename(dir, tmp); err != nil {
				return err
			}
			defer func() {
				log.Debugf("Removing %s", tmp)
				os.RemoveAll(tmp)
			}()
			homeDir := filepath.Dir(filepath.Dir(javaPath))
			var src, dst string
			if goos == "darwin" {
				if filepath.Base(homeDir) == "Home" {
					src = filepath.Dir(homeDir)
					dst = filepath.Join(dir, "Contents")
				} else {
					src = homeDir
					dst = filepath.Join(dir, "Contents", "Home")
				}
			} else {
				src = homeDir
				dst = dir
			}
			log.Debugf("Moving %s to %s", src, dst)
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err = os.Rename(src, dst); err != nil {
				return err
			}
		}
		return assertJavaDistribution(dir, goos)
	}
	return nil
}

func expectedJavaPath(dir string, goos string) string {
	var osSpecificSubDir = ""
	if goos == "darwin" {
		osSpecificSubDir = filepath.Join("Contents", "Home")
	}
	java := "java"
	if goos == "windows" {
		java = "java.exe"
	}
	return filepath.Join(dir, osSpecificSubDir, "bin", java)
}

func assertJavaDistribution(dir string, goos string) error {
	var path = expectedJavaPath(dir, goos)
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = errors.New(path + " wasn't found. " +
			"If you believe this is an error - please create a ticket at https://github.com/felipebz/javm/issues " +
			"(specify OS and command that was used)")
	}
	return err
}

func installFromTgz(src string, dst string) error {
	log.Debug("Extracting " + src + " to " + dst)
	return untgz(src, dst, true)
}

func untgz(src string, dst string, strip bool) error {
	gzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzr, err := gzip.NewReader(gzFile)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTar(gzr, dst, strip)
}

func extractTar(r io.Reader, dst string, strip bool) error {
	tr := tar.NewReader(r)

	dirCache := make(map[string]bool)
	var rootPrefix string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := header.Name

		if strip {
			if rootPrefix == "" {
				parts := strings.Split(strings.Trim(name, "/"), "/")
				if len(parts) > 0 {
					rootPrefix = parts[0] + "/"
				}
			}

			if strings.HasPrefix(name, rootPrefix) {
				name = strings.TrimPrefix(name, rootPrefix)
			} else {
				continue
			}
		}

		if name == "" || name == "/" {
			continue
		}

		target := filepath.Join(dst, name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("path traversal detected: %s", target)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if !dirCache[target] {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
				dirCache[target] = true
			}

		case tar.TypeReg:
			parent := filepath.Dir(target)
			if !dirCache[parent] {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return err
				}
				dirCache[parent] = true
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode)&0777|0600)
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()

		case tar.TypeSymlink:
			parent := filepath.Dir(target)
			if !dirCache[parent] {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return err
				}
				dirCache[parent] = true
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}

	return nil
}

func installFromTgx(src string, dst string) error {
	log.Debug("Extracting " + src + " to " + dst)
	return untgx(src, dst, true)
}

func untgx(src string, dst string, strip bool) error {
	xzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer xzFile.Close()

	xzr, err := xz.NewReader(xzFile)
	if err != nil {
		return err
	}

	return extractTar(xzr, dst, strip)
}

func installFromZip(src string, dst string) error {
	log.Debug("Extracting " + src + " to " + dst)
	return unzip(src, dst, true)
}

func unzip(src string, dst string, strip bool) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	dirCache := make(map[string]bool)
	var rootPrefix string

	for _, f := range r.File {
		name := f.Name

		if strip {
			if rootPrefix == "" {
				parts := strings.Split(strings.Trim(name, "/"), "/")
				if len(parts) > 0 {
					rootPrefix = parts[0] + "/"
				}
			}

			if strings.HasPrefix(name, rootPrefix) {
				name = strings.TrimPrefix(name, rootPrefix)
			} else {
				continue
			}
		}

		if name == "" || name == "/" {
			continue
		}

		target := filepath.Join(dst, name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("path traversal detected: %s", target)
		}

		fi := f.FileInfo()
		mode := fi.Mode()

		if mode.IsDir() {
			if !dirCache[target] {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
				dirCache[target] = true
			}
			continue
		}

		parent := filepath.Dir(target)
		if !dirCache[parent] {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return err
			}
			dirCache[parent] = true
		}

		if mode&os.ModeSymlink != 0 {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			linkTarget, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return err
			}

			if err := os.Symlink(string(linkTarget), target); err != nil {
				return err
			}
			continue
		}

		fFile, err := f.Open()
		if err != nil {
			return err
		}

		dFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode&0777|0600)
		if err != nil {
			fFile.Close()
			return err
		}

		if _, err := io.Copy(dFile, fFile); err != nil {
			dFile.Close()
			fFile.Close()
			return err
		}

		dFile.Close()
		fFile.Close()
	}

	return nil
}
