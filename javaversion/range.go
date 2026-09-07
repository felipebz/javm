package javaversion

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Range is a Java-version selector. Alternatives are separated by || and
// terms within an alternative are ANDed.
type Range struct {
	Qualifier string
	raw       string
	groups    [][]constraint
}

// String returns the selector as it was supplied to ParseRange.
func (r *Range) String() string { return r.raw }

type operator uint8

const (
	opExact operator = iota
	opNotEqual
	opLess
	opLessOrEqual
	opGreater
	opGreaterOrEqual
	opTildeConstraint
	opCaretConstraint
)

type constraint struct {
	op      operator
	pattern versionPattern
	any     bool
}

type versionPattern struct {
	version     *Version
	numeric     []uint64
	wildcard    bool
	hasBuild    bool
	hasPre      bool
	hasOptional bool
}

// Contains reports whether v matches the selector, including its optional
// distribution qualifier.
func (r *Range) Contains(v *Version) bool {
	if r.Qualifier != "" && r.Qualifier != "*" && r.Qualifier != v.qualifier {
		return false
	}
	for _, group := range r.groups {
		matches := true
		for _, term := range group {
			if !term.matches(v) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func (c constraint) matches(candidate *Version) bool {
	if c.any {
		return true
	}
	if c.op == opExact || c.op == opNotEqual {
		matched := c.pattern.matchesExact(candidate)
		if c.op == opNotEqual {
			return !matched
		}
		return matched
	}

	comparison := compareUnqualified(candidate, c.pattern.version)
	switch c.op {
	case opLess:
		return comparison < 0
	case opLessOrEqual:
		return comparison <= 0
	case opGreater:
		return comparison > 0
	case opGreaterOrEqual:
		return comparison >= 0
	default:
		return false
	}
}

func (p versionPattern) matchesExact(candidate *Version) bool {
	if p.wildcard {
		if len(candidate.numeric) < len(p.numeric) {
			return false
		}
		for i, value := range p.numeric {
			if candidate.numericValue(i) != value {
				return false
			}
		}
	} else if !sameNumericSequence(candidate.numeric, p.numeric) {
		return false
	}

	if p.hasPre {
		if compareIdentifiers(candidate.prerelease, p.version.prerelease, true) != 0 {
			return false
		}
	} else if len(candidate.prerelease) != 0 {
		return false
	}
	if p.hasBuild && (!candidate.hasBuild || candidate.build != p.version.build) {
		return false
	}
	if p.hasOptional && compareIdentifiers(candidate.optional, p.version.optional, false) != 0 {
		return false
	}
	return true
}

func sameNumericSequence(left, right []uint64) bool {
	maxLength := max(len(left), len(right))
	for i := range maxLength {
		var leftValue, rightValue uint64
		if i < len(left) {
			leftValue = left[i]
		}
		if i < len(right) {
			rightValue = right[i]
		}
		if leftValue != rightValue {
			return false
		}
	}
	return true
}

// ParseRange parses javm selectors such as 25, temurin@25.0.4,
// >=21, ~17.0.10, and compound comparisons.
func ParseRange(raw string) (*Range, error) {
	original := raw
	raw = strings.TrimSpace(raw)
	rangeResult := &Range{raw: original}

	if at := strings.IndexByte(raw, '@'); at >= 0 {
		rangeResult.Qualifier = raw[:at]
		raw = raw[at+1:]
		if rangeResult.Qualifier == "" || strings.Contains(rangeResult.Qualifier, "@") {
			return nil, invalidRange(original)
		}
		if raw == "" {
			// Preserve the convenient `distribution@` selector.
			raw = "*"
		}
	}

	if raw == "" {
		rangeResult.groups = [][]constraint{{{any: true}}}
		return rangeResult, nil
	}

	for alternative := range strings.SplitSeq(raw, "||") {
		if strings.TrimSpace(alternative) == "" {
			return nil, invalidRange(original)
		}
		group, err := parseGroup(alternative)
		if err != nil {
			return nil, invalidRange(original)
		}
		rangeResult.groups = append(rangeResult.groups, group)
	}
	return rangeResult, nil
}

func parseGroup(raw string) ([]constraint, error) {
	tokens := strings.FieldsFunc(raw, func(char rune) bool {
		return char == ',' || unicode.IsSpace(char)
	})
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty range")
	}

	var result []constraint
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		if token == "-" {
			return nil, fmt.Errorf("unexpected hyphen")
		}

		if index+2 < len(tokens) && tokens[index+1] == "-" {
			left, err := parsePattern(tokens[index])
			if err != nil {
				return nil, err
			}
			right, err := parsePattern(tokens[index+2])
			if err != nil {
				return nil, err
			}
			result = append(result, constraint{op: opGreaterOrEqual, pattern: left})
			if len(right.numeric) <= 2 && !right.hasPre && !right.hasBuild && !right.hasOptional {
				result = append(result, constraint{op: opLess, pattern: boundary(right, len(right.numeric)-1)})
			} else {
				result = append(result, constraint{op: opLessOrEqual, pattern: right})
			}
			index += 2
			continue
		}

		operatorValue, value := splitOperator(token)
		if value == "" {
			if index+1 >= len(tokens) {
				return nil, fmt.Errorf("operator without version")
			}
			index++
			value = tokens[index]
		}

		if value == "*" || strings.EqualFold(value, "x") {
			if operatorValue != opExact {
				return nil, fmt.Errorf("wildcard with comparison")
			}
			result = append(result, constraint{any: true})
			continue
		}

		pattern, err := parsePattern(value)
		if err != nil {
			return nil, err
		}
		switch operatorValue {
		case opTildeConstraint:
			result = append(result,
				constraint{op: opGreaterOrEqual, pattern: pattern},
				constraint{op: opLess, pattern: boundary(pattern, tildeBoundaryIndex(pattern))},
			)
		case opCaretConstraint:
			result = append(result,
				constraint{op: opGreaterOrEqual, pattern: pattern},
				constraint{op: opLess, pattern: boundary(pattern, caretBoundaryIndex(pattern))},
			)
		default:
			result = append(result, constraint{op: operatorValue, pattern: pattern})
		}
	}
	return result, nil
}

type selectorOperator uint8

const (
	opExactSelector selectorOperator = iota
	opNotEqualSelector
	opLessSelector
	opLessOrEqualSelector
	opGreaterSelector
	opGreaterOrEqualSelector
	opTildeSelector
	opCaretSelector
)

func splitOperator(token string) (operatorValue operator, value string) {
	selectorValue := opExactSelector
	for _, prefix := range []struct {
		text  string
		value selectorOperator
	}{
		{"!=", opNotEqualSelector},
		{">=", opGreaterOrEqualSelector},
		{"<=", opLessOrEqualSelector},
		{"==", opExactSelector},
		{">", opGreaterSelector},
		{"<", opLessSelector},
		{"=", opExactSelector},
		{"~", opTildeSelector},
		{"^", opCaretSelector},
	} {
		if strings.HasPrefix(token, prefix.text) {
			selectorValue = prefix.value
			value = token[len(prefix.text):]
			switch selectorValue {
			case opNotEqualSelector:
				return opNotEqual, value
			case opLessSelector:
				return opLess, value
			case opLessOrEqualSelector:
				return opLessOrEqual, value
			case opGreaterSelector:
				return opGreater, value
			case opGreaterOrEqualSelector:
				return opGreaterOrEqual, value
			case opTildeSelector:
				return opTildeConstraint, value
			case opCaretSelector:
				return opCaretConstraint, value
			default:
				return opExact, value
			}
		}
	}
	return opExact, token
}

func parsePattern(raw string) (versionPattern, error) {
	if raw == "" {
		return versionPattern{}, fmt.Errorf("empty version")
	}

	numericText := raw
	if separator := strings.IndexAny(numericText, "+-"); separator >= 0 {
		numericText = numericText[:separator]
	}
	numericParts := strings.Split(numericText, ".")
	wildcardIndex := -1
	numeric := make([]uint64, 0, len(numericParts))
	for index, part := range numericParts {
		if part == "*" || strings.EqualFold(part, "x") {
			if wildcardIndex < 0 {
				wildcardIndex = index
			}
			continue
		}
		if wildcardIndex >= 0 {
			return versionPattern{}, fmt.Errorf("numeric component after wildcard")
		}
		value, err := parseUint(part)
		if err != nil {
			return versionPattern{}, err
		}
		numeric = append(numeric, value)
	}
	if wildcardIndex == 0 && len(numeric) == 0 {
		return versionPattern{wildcard: true, version: &Version{numeric: []uint64{0}}}, nil
	}
	if wildcardIndex >= 0 && len(numeric) == 0 {
		return versionPattern{}, fmt.Errorf("missing numeric prefix")
	}

	parseText := raw
	if wildcardIndex >= 0 {
		parts := strings.Split(numericText, ".")
		for index := wildcardIndex; index < len(parts); index++ {
			parts[index] = "0"
		}
		replacement := strings.Join(parts, ".")
		parseText = replacement + raw[len(numericText):]
	}
	parsed, err := parseUnqualified(parseText)
	if err != nil {
		return versionPattern{}, err
	}

	pattern := versionPattern{
		version:     parsed,
		numeric:     numeric,
		wildcard:    wildcardIndex >= 0,
		hasBuild:    parsed.hasBuild,
		hasPre:      len(parsed.prerelease) > 0,
		hasOptional: len(parsed.optional) > 0,
	}
	// Keep the historic javm behavior where 25 and 25.0 mean a prefix range,
	// while 25.0.4 (and more precise versions) are exact numeric versions.
	if wildcardIndex < 0 && len(numeric) < 3 && !pattern.hasPre && !pattern.hasBuild && !pattern.hasOptional {
		pattern.wildcard = true
	}
	return pattern, nil
}

func tildeBoundaryIndex(pattern versionPattern) int {
	switch {
	case len(pattern.numeric) <= 1:
		return 0
	default:
		// Keep the feature and interim components fixed, matching the
		// established ~17.0.10 => <17.1.0 selector behavior.
		return 1
	}
}

func caretBoundaryIndex(pattern versionPattern) int {
	for index, value := range pattern.numeric {
		if value != 0 {
			return index
		}
	}
	return max(0, len(pattern.numeric)-1)
}

func boundary(pattern versionPattern, index int) versionPattern {
	numeric := append([]uint64(nil), pattern.numeric...)
	if len(numeric) <= index {
		padded := make([]uint64, index+1)
		copy(padded, numeric)
		numeric = padded
	}
	numeric[index]++
	numeric = numeric[:index+1]
	version := &Version{numeric: numeric, raw: strings.Join(uintsToStrings(numeric), ".")}
	return versionPattern{version: version, numeric: numeric}
}

func uintsToStrings(values []uint64) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strconv.FormatUint(value, 10)
	}
	return result
}

func invalidRange(raw string) error {
	return fmt.Errorf("%s is not a valid version", raw)
}
