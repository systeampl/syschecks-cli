package output

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestRenderQuietPrintsFirstColumn(t *testing.T) {
	var b bytes.Buffer
	tbl := Table{Cols: []string{"id", "name"}, Rows: []Row{{"id": 3, "name": "a"}, {"id": 5, "name": "b"}}}
	if err := Render(&b, "table", true, tbl); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "3\n5\n" {
		t.Fatalf("quiet = %q, want \"3\\n5\\n\"", got)
	}
}

func TestRenderJSON(t *testing.T) {
	var b bytes.Buffer
	tbl := Table{Cols: []string{"id"}, Rows: []Row{{"id": 3}}}
	if err := Render(&b, "json", false, tbl); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"id": 3`) {
		t.Fatalf("json = %q", b.String())
	}
}

// An empty row set is still a row set: the documented JSON shape is an array,
// and `... -o json | jq 'length'` is the advertised use. Encoding a nil slice
// gives `null`, which breaks every such pipeline.
func TestEmptyRowSetRendersAsAnArray(t *testing.T) {
	var b bytes.Buffer
	if err := Render(&b, "json", false, Table{Cols: []string{"id"}, Rows: nil}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := strings.TrimSpace(b.String()); got != "[]" {
		t.Fatalf("empty row set rendered as %q, want %q", got, "[]")
	}
}

// --quiet short-circuited before the format switch, so a bad --output was
// accepted and the command exited 0.
func TestQuietStillRejectsAnUnknownFormat(t *testing.T) {
	var b bytes.Buffer
	err := Render(&b, "xml", true, Table{Cols: []string{"id"}, Rows: []Row{{"id": 1}}})
	if err == nil {
		t.Fatal("Render(quiet, format=xml) accepted an unknown format")
	}
}

// --no-color was a documented flag nothing read. Colour is decided here: only
// for a terminal, and never when the user turned it off.
func TestShouldColor(t *testing.T) {
	var buf bytes.Buffer
	if ShouldColor(&buf, false) {
		t.Error("ShouldColor(non-terminal) = true, want false")
	}
	if ShouldColor(&buf, true) {
		t.Error("ShouldColor(--no-color) = true, want false")
	}
}

func TestTableColorsTheStatusColumn(t *testing.T) {
	tbl := Table{Cols: []string{"id", "status"}, Rows: []Row{{"id": 1, "status": "DOWN"}}}

	var colored bytes.Buffer
	if err := Render(&colored, "table", false, tbl, WithColor(true)); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(colored.String(), "\x1b[") {
		t.Fatalf("coloured table has no ANSI: %q", colored.String())
	}

	var plain bytes.Buffer
	if err := Render(&plain, "table", false, tbl); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Fatalf("plain table contains ANSI: %q", plain.String())
	}
}

// Colour must not change what the data looks like once the escapes are stripped
// — the column layout is computed before they are added.
func TestColorDoesNotChangeTheLayout(t *testing.T) {
	tbl := Table{Cols: []string{"id", "status"}, Rows: []Row{{"id": 1, "status": "UP"}, {"id": 22, "status": "DOWN"}}}

	var colored, plain bytes.Buffer
	_ = Render(&colored, "table", false, tbl, WithColor(true))
	_ = Render(&plain, "table", false, tbl)

	stripped := ansiPattern.ReplaceAllString(colored.String(), "")
	if stripped != plain.String() {
		t.Fatalf("stripped coloured output %q != plain %q", stripped, plain.String())
	}
}

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")
