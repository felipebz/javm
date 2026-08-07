package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/semver"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
			ver, err := runInstall(cmd.Context(), client, ver, customInstallDestination)
			if err != nil {
				return err
			}
			if customInstallDestination == "" {
				if err := discovery.DeleteCacheFile(cfg.Dir()); err != nil {
					return err
				}

				if err := linkLatest(); err != nil {
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
			"  javm install ~1.8.73 # same as \">=1.8.73 <1.9.0\"",
	}
	cmd.Flags().StringVarP(&customInstallDestination, "output", "o", "",
		"New, non-existing custom destination (JDKs outside $JAVM_HOME/jdk are unmanaged unless linked)")
	return cmd
}

func runInstall(ctx context.Context, client PackagesWithInfoClient, selector string, dst string) (string, error) {
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
	packageIndex, err := makePackageIndexContext(ctx, client, runtime.GOOS, runtime.GOARCH, distribution)
	if err != nil {
		return "", err
	}
	sort.Sort(sort.Reverse(semver.VersionSlice(packageIndex.Sorted)))
	for _, v := range packageIndex.Sorted {
		if rng.Contains(v) {
			ver = v
			packageInfo, err := client.GetPackageInfoContext(ctx, packageIndex.ByVersion[ver].Id)
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
		local, err := Ls(true)
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
	}
	var file string
	var removeDownload bool
	if after, ok := strings.CutPrefix(url, "file://"); ok {
		file = after
		if runtime.GOOS == "windows" {
			// file:///C:/path/...
			file = strings.Replace(strings.TrimPrefix(file, "/"), "/", "\\", -1)
		}
	} else {
		log.Info("Downloading ", ver)
		log.Debug("URL: ", url)
		file, err = download(ctx, url)
		if err != nil {
			return "", err
		}
		removeDownload = true
		defer func() {
			if removeDownload {
				if removeErr := os.Remove(file); removeErr != nil && !os.IsNotExist(removeErr) {
					log.Warn("Failed to remove temporary download: ", removeErr)
				}
			}
		}()
	}
	if expectedChecksum != "" && checksumType != "" {
		if err := validateChecksum(file, expectedChecksum, checksumType); err != nil {
			return "", fmt.Errorf("verify downloaded artifact: %w", err)
		}
	} else {
		log.Warn("No checksum provided by DiscoAPI for this artifact; skipping integrity verification")
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		err = install(ctx, file, dst)
	default:
		err = errors.New(runtime.GOOS + " OS is not supported")
	}
	return ver.String(), err
}
