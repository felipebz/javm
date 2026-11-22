package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discoapi"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

func NewInstallCommand(client PackagesWithInfoClient) *cobra.Command {
	var customInstallDestination string

	cmd := &cobra.Command{
		Use:   "install [version to install]",
		Short: "Download and install JDK",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var ver string
			if len(args) == 0 {
				selector, err := javaVersionSelector()
				if err != nil {
					return UsageError(err)
				}
				ver = selector
			} else {
				ver = args[0]
			}
			if _, err := runInstall(cmd.Context(), client, ver, customInstallDestination); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					cmd.SilenceUsage = true
				}
				return err
			}
			if customInstallDestination != "" {
				return nil
			}
			return warnLinkFailures(cmd.Context(), linkLatest(cmd.Context()))
		},
		Example: "  javm install 21\n" +
			"  javm install ~21.0.4 # same as \">=21.0.4 <21.1.0\"",
	}
	cmd.Flags().StringVarP(&customInstallDestination, "output", "o", "",
		"New, non-existing custom destination (JDKs outside $JAVM_HOME/jdk are unmanaged unless linked)")
	return cmd
}

func runInstall(ctx context.Context, client PackagesWithInfoClient, selector string, dst string) (string, error) {
	rng, err := javaversion.ParseRange(selector)
	if err != nil {
		return "", UsageError(err)
	}
	distribution := rng.Qualifier
	if distribution == "" {
		var derr error
		distribution, derr = cfg.EffectiveValue("java.default_distribution")
		if derr != nil {
			return "", derr
		}
	}
	packageIndex, err := makePackageIndex(ctx, client, runtime.GOOS, runtime.GOARCH, distribution)
	if err != nil {
		return "", err
	}
	ver := packageIndex.latestMatching(rng)
	if ver == nil {
		targets := make([]string, 0, len(packageIndex.Sorted))
		for _, v := range slices.Backward(packageIndex.Sorted) {
			targets = append(targets, sanitizeTerminalText(v.String()))
		}
		return "", NotFoundError(errors.New("no compatible version found for " + sanitizeTerminalText(selector) + legacySelectorHint(selector) +
			"\nValid install targets: " + strings.Join(targets, ", ")))
	}
	packageInfo, err := client.GetPackageInfoContext(ctx, packageIndex.ByVersion[ver].Id)
	if err != nil {
		return "", NetworkError(err)
	}

	if dst == "" {
		local, err := LsContext(ctx, true)
		if err != nil {
			return "", err
		}
		if slices.ContainsFunc(local, func(jdk discovery.JDK) bool { return isInstalledPackage(jdk, ver) }) {
			return ver.String(), nil
		}
		dst, err = managedInstallDestination(cfg.Dir(), ver.WithoutBuild())
		if err != nil {
			return "", err
		}
		if err := refuseBrokenDestination(local, dst); err != nil {
			return "", err
		}
	}
	return ver.String(), fetchAndInstall(ctx, ver, packageInfo, dst, false)
}

// refuseBrokenDestination reports a managed destination that exists but is not
// a JDK discovery could validate, before anything is downloaded. Removing it
// is left to the user.
func refuseBrokenDestination(local []discovery.JDK, dst string) error {
	if _, err := os.Lstat(dst); err != nil {
		return nil
	}
	name := filepath.Base(dst)
	if slices.ContainsFunc(local, func(jdk discovery.JDK) bool { return jdk.Identifier == name }) {
		return nil
	}
	return fmt.Errorf("%s already exists but is not a valid JDK; run \"javm uninstall %s\" to remove it first",
		sanitizeTerminalText(dst), sanitizeTerminalText(name))
}

// isInstalledPackage reports whether jdk already provides ver. A JDK whose
// metadata carries no build number counts as the same release.
func isInstalledPackage(jdk discovery.JDK, ver *javaversion.Version) bool {
	identifier, err := javaversion.ParseVersion(jdk.Identifier)
	if err != nil || identifier.Qualifier() != ver.Qualifier() {
		return false
	}
	installed := identifier
	if version, err := javaversion.ParseVersion(jdk.Version); err == nil {
		installed, err = javaversion.ParseVersion(ver.Qualifier() + "@" + version.String())
		if err != nil {
			return false
		}
	}
	if installed.Equals(ver) {
		return true
	}
	_, hasBuild := installed.BuildNumber()
	return !hasBuild && installed.SameRelease(ver)
}

// fetchAndInstall downloads (or opens a file:// artifact), verifies the checksum
// and installs the package at dst, replacing an existing JDK directory when
// replace is set.
func fetchAndInstall(ctx context.Context, ver *javaversion.Version, info *discoapi.PackageInfo, dst string, replace bool) error {
	url := info.DirectDownloadUri
	var file string
	if after, ok := strings.CutPrefix(url, "file://"); ok {
		file = after
		if runtime.GOOS == "windows" {
			// file:///C:/path/...
			file = strings.Replace(strings.TrimPrefix(file, "/"), "/", "\\", -1)
		}
	} else {
		loggerFromContext(ctx).Info("Downloading ", ver)
		loggerFromContext(ctx).Debug("URL: ", url)
		var err error
		file, err = download(ctx, url)
		if err != nil {
			return err
		}
		defer func() {
			if removeErr := os.Remove(file); removeErr != nil && !os.IsNotExist(removeErr) {
				loggerFromContext(ctx).Warn("Failed to remove temporary download: ", removeErr)
			}
		}()
	}
	if info.Checksum != "" && info.ChecksumType != "" {
		if err := validateChecksum(file, info.Checksum, info.ChecksumType); err != nil {
			return fmt.Errorf("verify downloaded artifact: %w", err)
		}
	} else {
		loggerFromContext(ctx).Warn("No checksum provided by DiscoAPI for this artifact; skipping integrity verification")
	}
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		if replace {
			return installReplacing(ctx, file, dst)
		}
		return install(ctx, file, dst)
	default:
		return errors.New(runtime.GOOS + " OS is not supported")
	}
}

// Reject unsafe components before Join can normalize untrusted input.
func managedInstallDestination(home, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:") {
		return "", fmt.Errorf("invalid managed JDK name %q", name)
	}
	for _, c := range name {
		if c < 0x20 || c == 0x7f {
			return "", fmt.Errorf("invalid managed JDK name %q", name)
		}
	}
	root, err := filepath.Abs(filepath.Join(home, "jdk"))
	if err != nil {
		return "", fmt.Errorf("resolve managed JDK root: %w", err)
	}
	dst := filepath.Join(root, name)
	relative, err := filepath.Rel(root, dst)
	if err != nil || relative != name || filepath.Dir(dst) != root {
		return "", fmt.Errorf("managed JDK destination escapes root: %q", name)
	}
	return dst, nil
}
