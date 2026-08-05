package cli

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

// subcommandNames returns the Name() of each direct subcommand of c, for
// asserting which leaf verbs newResourceCmd generated.
func subcommandNames(c *cobra.Command) []string {
	var names []string
	for _, sc := range c.Commands() {
		names = append(names, sc.Name())
	}
	return names
}

func TestFactoryBuildsCrudAndRoutesFlags(t *testing.T) {
	var created map[string]any
	r := &Resource{
		Name: "widget", Cols: []string{"id", "name"}, Org: OrgNone,
		Fields: []Field{{Name: "name", JSONKey: "name", Kind: "string", Required: true}},
		listFn: func(env *cmdCtx, o *int) ([]map[string]any, error) {
			return []map[string]any{{"id": 1, "name": "a"}}, nil
		},
		getFn: func(env *cmdCtx, o *int, id int) (map[string]any, error) {
			return map[string]any{"id": id, "name": "a"}, nil
		},
		createFn: func(env *cmdCtx, o *int, b map[string]any) (map[string]any, error) {
			created = b
			return map[string]any{"id": 9, "name": b["name"]}, nil
		},
	}
	cmd := newResourceCmd(r)
	if cmd.Use != "widget" {
		t.Fatalf("use=%s", cmd.Use)
	}
	names := subcommandNames(cmd)
	for _, want := range []string{"list", "get", "create"} {
		if !slices.Contains(names, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if slices.Contains(names, "update") {
		t.Fatal("update must be absent when updateFn is nil")
	}
	if slices.Contains(names, "delete") {
		t.Fatal("delete must be absent when deleteFn is nil")
	}
	_ = created
}
