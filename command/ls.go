package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

func NewLsCommand() *cobra.Command {
	var showDetails bool
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List installed versions",
		Args:  UsageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var rng *javaversion.Range
			if len(args) > 0 {
				var err error
				rng, err = javaversion.ParseRange(args[0])
				if err != nil {
					return UsageError(err)
				}
			}

			jdks, err := LsContext(cmd.Context(), false)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			restoreConsole := enableANSIConsole(out)
			defer restoreConsole()
			return printInstalledVersions(out, newStyles(out), jdks, rng, showDetails)
		},
	}
	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed information about discovered JDKs")
	return cmd
}

var readDir = os.ReadDir

var lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
	manager, err := discovery.NewConfiguredManager(cfg.Dir())
	if err != nil {
		return nil, fmt.Errorf("load discovery configuration: %w", err)
	}
	manager.Warn = func(err error) {
		loggerFromContext(ctx).Warn(err)
	}

	return manager.DiscoverAll(ctx)
}

func Ls(managedOnly bool) ([]discovery.JDK, error) {
	return LsContext(context.Background(), managedOnly)
}

func LsContext(ctx context.Context, managedOnly bool) ([]discovery.JDK, error) {
	if managedOnly {
		return discovery.NewJavmSource().DiscoverManaged(ctx)
	}
	return lsFunc(ctx)
}

func LsBestMatch(selector string, managedOnly bool) (string, error) {
	return LsBestMatchContext(context.Background(), selector, managedOnly)
}

func LsBestMatchContext(ctx context.Context, selector string, managedOnly bool) (string, error) {
	jdks, err := LsContext(ctx, managedOnly)
	if err != nil {
		return "", err
	}
	jdk, err := FindBestMatchJDK(jdks, selector)
	if err != nil {
		return "", err
	}
	return jdk.Identifier, nil
}

func FindBestMatchJDK(jdks []discovery.JDK, selector string) (discovery.JDK, error) {
	rng, err := javaversion.ParseRange(selector)
	if err != nil {
		return discovery.JDK{}, UsageError(err)
	}

	candidates := slices.Clone(jdks)
	sort.Slice(candidates, func(i, j int) bool {
		v1, err1 := javaversion.ParseVersion(candidates[i].Version)
		v2, err2 := javaversion.ParseVersion(candidates[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		if candidates[i].Version != candidates[j].Version {
			return candidates[i].Version > candidates[j].Version
		}
		return candidates[i].Identifier < candidates[j].Identifier
	})

	var fallback discovery.JDK
	hasFallback := false

	for _, jdk := range candidates {
		v, err := parseJDKVersionForRange(jdk, rng)

		if err == nil && rng.Contains(v) {
			if jdk.Source == "javm" {
				return jdk, nil
			}

			if !hasFallback {
				fallback = jdk
				hasFallback = true
			}
		}
	}

	if hasFallback {
		return fallback, nil
	}

	return discovery.JDK{}, NotFoundError(fmt.Errorf("%s isn't installed", rng.String()))
}

func printInstalledVersions(w io.Writer, styles styles, jdks []discovery.JDK, rng *javaversion.Range, showDetails bool) error {
	// Filter by range
	var filtered []discovery.JDK
	for _, jdk := range jdks {
		if rng != nil {
			v, err := parseJDKVersionForRange(jdk, rng)
			if err != nil || !rng.Contains(v) {
				continue
			}
		}
		filtered = append(filtered, jdk)
	}

	// Sort by Source (ASC) then Version (DESC)
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Source != filtered[j].Source {
			return filtered[i].Source < filtered[j].Source
		}
		v1, err1 := javaversion.ParseVersion(filtered[i].Version)
		v2, err2 := javaversion.ParseVersion(filtered[j].Version)
		if err1 == nil && err2 == nil {
			return v2.LessThan(v1)
		}
		if filtered[i].Version != filtered[j].Version {
			return filtered[i].Version > filtered[j].Version
		}
		return filtered[i].Identifier < filtered[j].Identifier
	})

	selectedByMap := computeSelectedBy(jdks, filtered)

	if showDetails {
		rows := make([][]tableCell, 0, len(filtered)+1)
		rows = append(rows, []tableCell{
			{"SOURCE", roleHeader},
			{"NAME", roleHeader},
			{"SELECTED BY", roleHeader},
			{"VENDOR", roleHeader},
			{"ARCHITECTURE", roleHeader},
			{"PATH", roleHeader},
		})
		for _, jdk := range filtered {
			selectedBy := selectedByMap[jdkKey(jdk)]
			rows = append(rows, []tableCell{
				{jdk.Source, roleDim},
				{jdk.Identifier, nameRole(selectedBy)},
				{selectedBy, roleAccent},
				{jdk.Vendor, roleDefault},
				{jdk.Architecture, roleDefault},
				{jdk.Path, roleDefault},
			})
		}
		return writeTable(w, styles, rows)
	}

	rows := make([][]tableCell, 0, len(filtered)+1)
	rows = append(rows, []tableCell{
		{"NAME", roleHeader},
		{"SOURCE", roleHeader},
		{"SELECTED BY", roleHeader},
	})
	for _, jdk := range filtered {
		selectedBy := selectedByMap[jdkKey(jdk)]
		row := []tableCell{
			{jdk.Identifier, nameRole(selectedBy)},
			{jdk.Source, roleDim},
		}
		if selectedBy != "" {
			row = append(row, tableCell{selectedBy, roleAccent})
		}
		rows = append(rows, row)
	}
	return writeTable(w, styles, rows)
}

// nameRole keeps the JDKs that win one of the selectors shown in SELECTED BY at
// normal intensity and dims the others, which stay readable but less prominent.
func nameRole(selectedBy string) styleRole {
	if selectedBy == "" {
		return roleDim
	}
	return roleDefault
}

func computeSelectedBy(allJDKs, displayedJDKs []discovery.JDK) map[string]string {
	selectedBy := make(map[string]string, len(displayedJDKs))
	if len(allJDKs) == 0 || len(displayedJDKs) == 0 {
		return selectedBy
	}

	uniqueMajors := make(map[string]struct{})
	uniqueDistMajors := make(map[string]struct{})

	for _, jdk := range displayedJDKs {
		dist, major := extractJDKDistributionAndMajor(jdk)
		if major != "" {
			uniqueMajors[major] = struct{}{}
			if dist != "" {
				uniqueDistMajors[dist+"@"+major] = struct{}{}
			}
		}
	}

	majorWinners := make(map[string]string)
	distMajorWinners := make(map[string][]string)

	for major := range uniqueMajors {
		winner, err := resolveJDKFromList(allJDKs, major)
		if err == nil {
			majorWinners[jdkKey(winner)] = major
		}
	}

	for distMajor := range uniqueDistMajors {
		winner, err := resolveJDKFromList(allJDKs, distMajor)
		if err == nil {
			winnerKey := jdkKey(winner)
			distMajorWinners[winnerKey] = append(distMajorWinners[winnerKey], distMajor)
		}
	}

	for _, jdk := range displayedJDKs {
		key := jdkKey(jdk)
		var selectors []string
		if major, ok := majorWinners[key]; ok {
			selectors = append(selectors, major)
		}
		if dms, ok := distMajorWinners[key]; ok {
			slices.Sort(dms)
			selectors = append(selectors, dms...)
		}
		if len(selectors) > 0 {
			selectedBy[key] = strings.Join(selectors, ", ")
		}
	}

	return selectedBy
}

func jdkKey(jdk discovery.JDK) string {
	return jdk.Source + "|" + jdk.Identifier + "|" + jdk.Version + "|" + jdk.Path
}

func extractJDKDistributionAndMajor(jdk discovery.JDK) (distribution, major string) {
	id := jdk.Identifier
	var verPart string
	if dist, ver, ok := strings.Cut(id, "@"); ok {
		distribution = dist
		verPart = ver
	} else {
		verPart = id
	}

	major = extractMajor(verPart)
	if major == "" && jdk.Version != "" {
		major = extractMajor(jdk.Version)
	}
	return distribution, major
}

func extractMajor(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "1.") {
		parts := strings.Split(v, ".")
		if len(parts) > 1 {
			prefix := extractNumericPrefix(parts[1])
			if prefix != "" && prefix != "0" {
				return prefix
			}
		}
	}
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		prefix := extractNumericPrefix(parts[0])
		if prefix != "" && prefix != "0" {
			return prefix
		}
	}
	return ""
}

func extractNumericPrefix(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}

func parseJDKVersionForRange(jdk discovery.JDK, rng *javaversion.Range) (*javaversion.Version, error) {
	version, versionErr := javaversion.ParseVersion(jdk.Version)
	identifier, identifierErr := javaversion.ParseVersion(jdk.Identifier)

	if rng.Qualifier != "" && rng.Qualifier != "*" {
		if identifierErr != nil || identifier.Qualifier() != rng.Qualifier {
			return nil, fmt.Errorf("JDK qualifier does not match selector")
		}
		if versionErr == nil {
			return javaversion.ParseVersion(rng.Qualifier + "@" + version.String())
		}
		return identifier, nil
	}
	if versionErr == nil {
		return version, nil
	}
	return identifier, identifierErr
}
