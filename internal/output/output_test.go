package output

import (
	"bytes"
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
