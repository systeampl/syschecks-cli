package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type Row = map[string]any

type Table struct {
	Cols []string
	Rows []Row
}

func Render(w io.Writer, format string, quiet bool, t Table) error {
	if quiet {
		for _, r := range t.Rows {
			fmt.Fprintln(w, r[t.Cols[0]])
		}
		return nil
	}
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(t.Rows)
	case "yaml":
		return yaml.NewEncoder(w).Encode(t.Rows)
	case "table", "":
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		for i, c := range t.Cols {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, c)
		}
		fmt.Fprintln(tw)
		for _, r := range t.Rows {
			for i, c := range t.Cols {
				if i > 0 {
					fmt.Fprint(tw, "\t")
				}
				fmt.Fprintf(tw, "%v", r[c])
			}
			fmt.Fprintln(tw)
		}
		return tw.Flush()
	default:
		return fmt.Errorf("unknown output format %q", format)
	}
}
