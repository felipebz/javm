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
	switch c.role {
	case roleHeader:
		return s.Header(c.value)
	case roleAccent:
		return s.Accent(c.value)
	case roleDim:
		return s.Dim(c.value)
	default:
		return c.value
	}
}

// writeTable writes rows as a space aligned table. The layout is computed by
// text/tabwriter over the plain values, so ANSI escapes never take part in the
// column width calculation; the styles are applied to the finished cells
// afterwards, preserving the spacing of the unstyled table.
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
			values[i] = cell.value
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
		i := strings.Index(line[offset:], cell.value)
		if i < 0 {
			continue
		}
		start := offset + i
		b.WriteString(line[offset:start])
		b.WriteString(s.cell(cell))
		offset = start + len(cell.value)
	}
	b.WriteString(line[offset:])
	return b.String()
}
