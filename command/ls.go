package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/semver"
	"github.com/spf13/cobra"
)

var readDir = os.ReadDir

func NewLsCommand() *cobra.Command {
	var trimTo string
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List installed versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			var rng *semver.Range
			if len(args) > 0 {
				var err error
				rng, err = semver.ParseRange(args[0])
				if err != nil {
					return err
				}
			}

			vs, err := Ls()
			if err != nil {
				return err
			}
			if trimTo != "" {
				vs = semver.VersionSlice(vs).TrimTo(parseTrimTo(trimTo))
			}
			printInstalledVersions(cmd.OutOrStdout(), vs, rng)
			return nil
		},
	}
	cmd.Flags().StringVar(&trimTo, "latest", "", "Part of the version to trim to (\"major\", \"minor\" or \"patch\")")
	return cmd
}

func Ls() ([]*semver.Version, error) {
	files, _ := readDir(filepath.Join(cfg.Dir(), "jdk"))
	var r []*semver.Version
	for _, f := range files {
		info, _ := f.Info()
		if f.IsDir() || (info.Mode()&os.ModeSymlink == os.ModeSymlink && strings.HasPrefix(f.Name(), "system@")) {
			v, err := semver.ParseVersion(f.Name())
			if err != nil {
				return nil, err
			}
			r = append(r, v)
		}
	}
	sort.Sort(sort.Reverse(semver.VersionSlice(r)))
	return r, nil
}

func LsBestMatch(selector string) (ver string, err error) {
	vs, err := Ls()
	if err != nil {
		return
	}
	return LsBestMatchWithVersionSlice(vs, selector)
}

func LsBestMatchWithVersionSlice(vs []*semver.Version, selector string) (ver string, err error) {
	rng, err := semver.ParseRange(selector)
	if err != nil {
		return
	}
	for _, v := range vs {
		if rng.Contains(v) {
			ver = v.String()
			break
		}
	}
	if ver == "" {
		err = fmt.Errorf("%s isn't installed", rng)
	}
	return
}

func printInstalledVersions(w io.Writer, vs []*semver.Version, rng *semver.Range) {
	for _, v := range vs {
		if rng != nil && !rng.Contains(v) {
			continue
		}
		fmt.Fprintln(w, v)
	}
}
