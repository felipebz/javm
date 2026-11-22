package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/felipebz/javm/cfg"
	"github.com/felipebz/javm/discovery"
	"github.com/felipebz/javm/javaversion"
	"github.com/spf13/cobra"
)

func NewLsCommand() *cobra.Command {
	var showDetails bool
	cmd := &cobra.Command{
		Use:   "ls [range]",
		Short: "List installed and discovered JDKs",
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
	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "include source, vendor, architecture and path")
	return cmd
}

var readDir = os.ReadDir

var lsFunc = func(ctx context.Context) ([]discovery.JDK, error) {
	manager, err := discovery.NewConfiguredManager(cfg.Dir())
	if err != nil {
		return nil, fmt.Errorf("load discovery configuration: %w", err)
	}
	manager.Warn = func(err error) {
		warnings, ok := discoveryWarnings(err)
		if !ok {
			warnOnce(ctx, err)
			return
		}
		for _, warning := range warnings {
			warnOnce(ctx, warning)
		}
	}

	jdks, err := manager.DiscoverAll(ctx)
	if saveErr, ok := errors.AsType[*discovery.CacheSaveError](err); ok {
		// The list is complete without the cache; only discover refresh,
		// whose job is to write it, fails on this.
		warnOnce(ctx, fmt.Errorf("could not save the discovery cache: %w; javm will search for JDKs again on the next command", saveErr.Err))
		return jdks, nil
	}
	return jdks, err
}

func Ls(managedOnly bool) ([]discovery.JDK, error) {
	return LsContext(context.Background(), managedOnly)
}

func LsContext(ctx context.Context, managedOnly bool) ([]discovery.JDK, error) {
	if managedOnly {
		jdks, err := discovery.NewJavmSource().DiscoverManaged(ctx)
		if err == nil {
			return jdks, nil
		}
		// A broken managed JDK must not block commands that only need the
		// valid ones; any other failure, including cancellation, is fatal.
		warnings, ok := discoveryWarnings(err)
		if !ok {
			return nil, err
		}
		for _, warning := range warnings {
			warnOnce(ctx, warning)
		}
		return jdks, nil
	}
	return lsFunc(ctx)
}

// discoveryWarnings flattens a possibly joined error into its discovery
// warnings. ok is false when any part is not a *discovery.DiscoveryWarning.
func discoveryWarnings(err error) ([]error, bool) {
	if joined, isJoined := err.(interface{ Unwrap() []error }); isJoined {
		var warnings []error
		for _, part := range joined.Unwrap() {
			partWarnings, ok := discoveryWarnings(part)
			if !ok {
				return nil, false
			}
			warnings = append(warnings, partWarnings...)
		}
		return warnings, true
	}
	if _, ok := errors.AsType[*discovery.DiscoveryWarning](err); !ok {
		return nil, false
	}
	return []error{err}, true
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
	sort.Slice(candidates, func(i, j int) bool { return newerJDKFirst(candidates[i], candidates[j]) })

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

	return discovery.JDK{}, NotFoundError(fmt.Errorf("%s isn't installed%s", sanitizeTerminalText(rng.String()), legacySelectorHint(selector)))
}

// legacySelectorHint explains the modern numbering when a selector uses the
// 1.x form of Java 8 and earlier (1.8, ~1.8.73, zulu@1.8), which matches
// nothing because javm reads 1.8.0_502 as 8.0.502. Other selectors get "".
func legacySelectorHint(selector string) string {
	if at := strings.IndexByte(selector, '@'); at >= 0 {
		selector = selector[at+1:]
	}
	tokens := strings.FieldsFunc(selector, func(char rune) bool {
		return char == ',' || char == '|' || unicode.IsSpace(char)
	})
	for _, token := range tokens {
		version, legacy := strings.CutPrefix(strings.TrimLeft(token, "!<>=~^"), "1.")
		if !legacy {
			continue
		}
		feature := version[:len(version)-len(strings.TrimLeft(version, "0123456789"))]
		if feature == "" {
			feature = "8"
		}
		return ` (Java 8 and earlier are written without "1.", for example ` + feature + ")"
	}
	return ""
}

func printInstalledVersions(w io.Writer, styles styles, jdks []discovery.JDK, rng *javaversion.Range, showDetails bool) error {
	var groups []jdkGroup
	var displayed []discovery.JDK
	for _, group := range groupJDKs(jdks) {
		if len(filterByRange(group.members, rng)) > 0 {
			groups = append(groups, group)
			displayed = append(displayed, group.members...)
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		a, b := groups[i].primary, groups[j].primary
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return newerJDKFirst(a, b)
	})

	selectedBy := computeSelectedBy(jdks, displayed)
	if showDetails {
		return writeTable(w, styles, detailedRows(groups, selectedBy))
	}
	return writeTable(w, styles, compactRows(groups, selectedBy))
}

// jdkGroup is one JDK directory found as one or more entries, for example
// as a system@ link and by the source that found its real directory.
type jdkGroup struct {
	primary discovery.JDK
	members []discovery.JDK
}

// groupJDKs groups the entries of the same JDK directory in discovery order.
// The row is named after a JDK managed by javm, then a system@ link the user
// created, then the first entry found.
func groupJDKs(jdks []discovery.JDK) []jdkGroup {
	var groups []jdkGroup
	byDir := make(map[string]int)
	for _, jdk := range jdks {
		if jdk.Path != "" {
			key := jdkDirKey(jdk.Path)
			if i, ok := byDir[key]; ok {
				groups[i].members = append(groups[i].members, jdk)
				if rowPreference(jdk) < rowPreference(groups[i].primary) {
					groups[i].primary = jdk
				}
				continue
			}
			byDir[key] = len(groups)
		}
		groups = append(groups, jdkGroup{primary: jdk, members: []discovery.JDK{jdk}})
	}
	return groups
}

func rowPreference(jdk discovery.JDK) int {
	switch {
	case jdk.Source == "javm" && !strings.HasPrefix(jdk.Identifier, "system@"):
		return 0
	case jdk.Source == "javm":
		return 1
	default:
		return 2
	}
}

// groupSelectedBy merges the selectors that pick any entry of group and
// appends "also <name>" for its other identifiers not already listed.
// hasSelectors reports whether any selector picks the group.
func groupSelectedBy(group jdkGroup, selectedBy map[string]string) (label string, hasSelectors bool) {
	members := append([]discovery.JDK{group.primary}, group.members...)
	var selectors []string
	for _, jdk := range members {
		for selector := range strings.SplitSeq(selectedBy[jdkKey(jdk)], ", ") {
			if selector != "" && !slices.Contains(selectors, selector) {
				selectors = append(selectors, selector)
			}
		}
	}
	var others []string
	for _, jdk := range group.members {
		if jdk.Identifier != group.primary.Identifier && !slices.Contains(selectors, jdk.Identifier) && !slices.Contains(others, jdk.Identifier) {
			others = append(others, jdk.Identifier)
		}
	}
	label = strings.Join(selectors, ", ")
	if len(others) > 0 {
		if label != "" {
			label += "; "
		}
		label += "also " + strings.Join(others, ", ")
	}
	return label, len(selectors) > 0
}

// newerJDKFirst orders JDKs by descending version, comparing unparsable
// versions as text, and then by identifier.
func newerJDKFirst(a, b discovery.JDK) bool {
	v1, err1 := javaversion.ParseVersion(a.Version)
	v2, err2 := javaversion.ParseVersion(b.Version)
	if err1 == nil && err2 == nil {
		return v2.LessThan(v1)
	}
	if a.Version != b.Version {
		return a.Version > b.Version
	}
	return a.Identifier < b.Identifier
}

// filterByRange keeps the JDKs inside rng; a nil range keeps all of them.
func filterByRange(jdks []discovery.JDK, rng *javaversion.Range) []discovery.JDK {
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
	return filtered
}

// detailedRows shows every entry: the group's row first, then its other
// sources and paths indented below it.
func detailedRows(groups []jdkGroup, selectedBy map[string]string) [][]tableCell {
	rows := make([][]tableCell, 0, len(groups)+1)
	rows = append(rows, []tableCell{
		{"SOURCE", roleHeader},
		{"NAME", roleHeader},
		{"SELECTED BY", roleHeader},
		{"VENDOR", roleHeader},
		{"ARCHITECTURE", roleHeader},
		{"PATH", roleHeader},
	})
	for _, group := range groups {
		label, hasSelectors := groupSelectedBy(group, selectedBy)
		jdk := group.primary
		rows = append(rows, []tableCell{
			{jdk.Source, roleDim},
			{jdk.Identifier, nameRole(hasSelectors)},
			{label, roleAccent},
			{jdk.Vendor, roleDefault},
			{jdk.Architecture, roleDefault},
			{jdk.Path, roleDefault},
		})
		for _, other := range group.members {
			if other == jdk {
				continue
			}
			rows = append(rows, []tableCell{
				{"  " + other.Source, roleDim},
				{"  " + other.Identifier, roleDim},
				{"", roleAccent},
				{other.Vendor, roleDefault},
				{other.Architecture, roleDefault},
				{other.Path, roleDefault},
			})
		}
	}
	return rows
}

func compactRows(groups []jdkGroup, selectedBy map[string]string) [][]tableCell {
	rows := make([][]tableCell, 0, len(groups)+1)
	rows = append(rows, []tableCell{
		{"NAME", roleHeader},
		{"SOURCE", roleHeader},
		{"SELECTED BY", roleHeader},
	})
	for _, group := range groups {
		label, hasSelectors := groupSelectedBy(group, selectedBy)
		row := []tableCell{
			{group.primary.Identifier, nameRole(hasSelectors)},
			{group.primary.Source, roleDim},
		}
		if label != "" {
			row = append(row, tableCell{label, roleAccent})
		}
		rows = append(rows, row)
	}
	return rows
}

// nameRole keeps the JDKs that win one of the selectors shown in SELECTED BY at
// normal intensity and dims the others, which stay readable but less prominent.
func nameRole(hasSelectors bool) styleRole {
	if !hasSelectors {
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
