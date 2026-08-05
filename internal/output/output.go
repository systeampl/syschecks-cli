package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type Row = map[string]any

type Table struct {
	Cols []string
	Rows []Row
}

// Option tweaks a single Render call. Kept variadic so the many existing call
// sites that need no options stay as they are.
type Option func(*renderOpts)

type renderOpts struct{ color bool }

// WithColor turns on ANSI colouring of the status column in table output.
// Callers decide with ShouldColor; Render never guesses.
func WithColor(on bool) Option { return func(o *renderOpts) { o.color = on } }

// ShouldColor reports whether output written to w should be coloured: only a
// terminal, and never when the user passed --no-color.
func ShouldColor(w io.Writer, noColor bool) bool {
	if noColor {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// statusColors maps a check/incident status to its ANSI colour. Anything not
// listed is left alone.
var statusColors = map[string]string{
	"UP":        "\x1b[32m",
	"DOWN":      "\x1b[31m",
	"DEGRADED":  "\x1b[33m",
	"LATE":      "\x1b[33m",
	"PAUSED":    "\x1b[90m",
	"NEW":       "\x1b[36m",
	"RESOLVED":  "\x1b[32m",
	"ONGOING":   "\x1b[31m",
	"UNKNOWN":   "\x1b[90m",
	"SUSPENDED": "\x1b[90m",
}

// colorizeStatuses wraps status words in `rendered` with ANSI colour. It runs
// AFTER tabwriter has laid the table out: escape sequences are zero-width on a
// terminal, so adding them here cannot shift the columns, whereas colouring the
// cells beforehand would have — tabwriter counts the escape bytes as content.
func colorizeStatuses(rendered string) string {
	var b strings.Builder
	for i, line := range strings.Split(rendered, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(colorizeLine(line))
	}
	return b.String()
}

func colorizeLine(line string) string {
	fields := strings.Fields(line)
	for _, f := range fields {
		color, ok := statusColors[f]
		if !ok {
			continue
		}
		// Replace the standalone word only, keeping the surrounding padding.
		idx := strings.Index(line, f)
		for idx >= 0 {
			endsWord := idx+len(f) == len(line) || line[idx+len(f)] == ' '
			startsWord := idx == 0 || line[idx-1] == ' '
			if startsWord && endsWord {
				return line[:idx] + color + f + "\x1b[0m" + line[idx+len(f):]
			}
			next := strings.Index(line[idx+1:], f)
			if next < 0 {
				break
			}
			idx += 1 + next
		}
	}
	return line
}

// cell renders one value for the human-facing formats. JSON numbers arrive as float64,
// and %v prints them with an exponent once they get long enough — an incident id came out
// as "2.774001e+06". Ids are meant to be copied into the next command, so integral values
// print as integers; anything fractional keeps its own formatting.
func cell(v any) string {
	if f, ok := v.(float64); ok && f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return fmt.Sprintf("%v", v)
}

func Render(w io.Writer, format string, quiet bool, t Table, opts ...Option) error {
	var o renderOpts
	for _, opt := range opts {
		opt(&o)
	}
	// Validate the format before anything else: --quiet used to short-circuit
	// first, so `-o xml -q` printed rows and exited 0 while `-o xml` alone was
	// rejected.
	switch format {
	case "json", "yaml", "table", "":
	default:
		return fmt.Errorf("unknown output format %q", format)
	}

	if quiet {
		for _, r := range t.Rows {
			fmt.Fprintln(w, cell(r[t.Cols[0]]))
		}
		return nil
	}

	// An empty result is still a row set. A nil slice encodes as `null`, which
	// breaks `-o json | jq` pipelines the exit-code contract invites.
	rows := t.Rows
	if rows == nil {
		rows = []Row{}
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "yaml":
		return yaml.NewEncoder(w).Encode(rows)
	case "table", "":
		var laid bytes.Buffer
		tw := tabwriter.NewWriter(&laid, 0, 2, 2, ' ', 0)
		for i, c := range t.Cols {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, c)
		}
		fmt.Fprintln(tw)
		for _, r := range rows {
			for i, c := range t.Cols {
				if i > 0 {
					fmt.Fprint(tw, "\t")
				}
				fmt.Fprint(tw, cell(r[c]))
			}
			fmt.Fprintln(tw)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		out := laid.String()
		if o.color {
			out = colorizeStatuses(out)
		}
		_, err := io.WriteString(w, out)
		return err
	default:
		return fmt.Errorf("unknown output format %q", format)
	}
}
