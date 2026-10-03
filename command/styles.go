package command

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/muesli/termenv"
)

// accentHex is the javm accent color (the orange used by the site and the logo).
// It is defined once: termenv degrades it to the closest color supported by the
// terminal profile.
const accentHex = "#d8540a"

// styleRole is the visual hierarchy level of a table cell.
type styleRole int

const (
	// roleDefault renders the value untouched.
	roleDefault styleRole = iota
	// roleHeader renders the value as a column header.
	roleHeader
	// roleAccent renders the value with the javm accent color.
	roleAccent
	// roleDim renders the value dimmed.
	roleDim
)

// styles renders text with the javm output hierarchy. A value built for a
// writer that is not a color capable terminal renders plain text.
type styles struct {
	header termenv.Style
	accent termenv.Style
	dim    termenv.Style
}

// newStyles returns the styles for w. By default, styling is emitted only for
// color-capable terminals, while pipes, redirections, and buffers stay plain.
// termenv also honors NO_COLOR, CLICOLOR, CLICOLOR_FORCE, and TERM.
func newStyles(w io.Writer, opts ...termenv.OutputOption) styles {
	out := termenv.NewOutput(w, opts...)
	accent := out.Color(accentHex)
	return styles{
		header: out.String("").Bold(),
		accent: out.String("").Foreground(accent),
		dim:    out.String("").Faint(),
	}
}

// enableANSIConsole turns on virtual terminal processing for w, which Windows
// consoles need before they render escape sequences. It is a no-op on other
// systems and for writers that are not terminals. The returned function
// restores the previous console mode.
func enableANSIConsole(w io.Writer) func() {
	restore, err := termenv.EnableVirtualTerminalProcessing(termenv.NewOutput(w))
	if err != nil || restore == nil {
		return func() {}
	}
	return func() { _ = restore() }
}

// Header renders a column header.
func (s styles) Header(v string) string { return s.styled(s.header, v) }

// Accent renders the value that the table highlights.
func (s styles) Accent(v string) string { return s.styled(s.accent, v) }

// Dim renders a secondary value.
func (s styles) Dim(v string) string { return s.styled(s.dim, v) }

func (s styles) styled(style termenv.Style, v string) string {
	if v == "" {
		return ""
	}
	return style.Styled(v)
}

// tableCell is a plain value plus the role it is rendered with.
type tableCell struct {
	value string
	role  styleRole
}

func (s styles) cell(c tableCell) string {
	value := sanitizeTerminalText(c.value)
	switch c.role {
	case roleHeader:
		return s.Header(value)
	case roleAccent:
		return s.Accent(value)
	case roleDim:
		return s.Dim(value)
	default:
		return value
	}
}

// sanitizeTerminalText removes the control characters a terminal emulator would
// interpret as escape sequences. Managed JDK names, installation paths, vendor
// metadata read from an archive release file, and remote package fields are not
// authored by javm, so they must not reach the terminal verbatim.
func sanitizeTerminalText(v string) string {
	return strings.Map(func(r rune) rune {
		if isTerminalControl(r) {
			return -1
		}
		return r
	}, v)
}

// isTerminalControl reports whether r is a C0, DEL, or C1 control character.
func isTerminalControl(r rune) bool {
	return r < 0x20 || r == 0x7f || r >= 0x80 && r <= 0x9f
}

// writeTable writes rows as a space aligned table. The layout is computed by
// text/tabwriter over the plain sanitized values, so neither ANSI escapes nor
// control characters take part in the column width calculation; the styles are
// applied to the finished cells afterwards, preserving the spacing of the
// unstyled table.
func writeTable(w io.Writer, styles styles, rows [][]tableCell) error {
	if len(rows) == 0 {
		return nil
	}
	plain, err := layoutTable(rows)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, styles.styleTable(plain, rows)); err != nil {
		return fmt.Errorf("write table: %w", err)
	}
	return nil
}

// layoutTable renders the plain values of rows.
func layoutTable(rows [][]tableCell) (string, error) {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	for _, row := range rows {
		values := make([]string, len(row))
		for i, cell := range row {
			values[i] = sanitizeTerminalText(cell.value)
		}
		if _, err := fmt.Fprintln(tw, strings.Join(values, "\t")); err != nil {
			return "", fmt.Errorf("write table row: %w", err)
		}
	}
	if err := tw.Flush(); err != nil {
		return "", fmt.Errorf("flush table: %w", err)
	}
	return buf.String(), nil
}

// styleTable applies the cell styles to a table laid out by layoutTable.
func (s styles) styleTable(plain string, rows [][]tableCell) string {
	lines := strings.Split(plain, "\n")
	for i, row := range rows {
		if i < len(lines) {
			lines[i] = s.styleLine(lines[i], row)
		}
	}
	return strings.Join(lines, "\n")
}

// styleLine replaces the values of a single laid out line with their styled
// versions, leaving the padding between them untouched. The values are located
// in order because text/tabwriter lays out each group of lines independently,
// so the columns of a line are not necessarily at the offsets of the header.
func (s styles) styleLine(line string, row []tableCell) string {
	var b strings.Builder
	b.Grow(len(line))
	offset := 0
	for _, cell := range row {
		value := sanitizeTerminalText(cell.value)
		i := strings.Index(line[offset:], value)
		if i < 0 {
			continue
		}
		start := offset + i
		b.WriteString(line[offset:start])
		b.WriteString(s.cell(cell))
		offset = start + len(value)
	}
	b.WriteString(line[offset:])
	return b.String()
}
