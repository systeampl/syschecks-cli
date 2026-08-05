package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
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

// setCrudTestEnv points a test at an isolated, unreachable config: cmdEnv's
// config.Load/config.Resolve/sdkclient.New all succeed against these values
// without ever making a network call, since the synthetic Resource's
// closures never touch env.SDK. That is what lets these tests execute a
// leaf's real RunE (through cmdEnv) while staying fully in-memory.
func setCrudTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", "http://127.0.0.1:0")
	t.Setenv("SYSCHECKS_TOKEN", "test-token")
}

// newWidgetResource builds the same synthetic in-memory Resource the brief's
// test uses, capturing the body createFn receives into *created so callers
// can assert on it after executing the command.
func newWidgetResource(created *map[string]any) *Resource {
	return &Resource{
		Name: "widget", Cols: []string{"id", "name"}, Org: OrgNone,
		Fields: []Field{{Name: "name", JSONKey: "name", Kind: "string", Required: true}},
		listFn: func(env *cmdCtx, o *int) ([]map[string]any, error) {
			return []map[string]any{{"id": 1, "name": "a"}}, nil
		},
		getFn: func(env *cmdCtx, o *int, id int) (map[string]any, error) {
			return map[string]any{"id": id, "name": "a"}, nil
		},
		createFn: func(env *cmdCtx, o *int, b map[string]any) (map[string]any, error) {
			*created = b
			return map[string]any{"id": 9, "name": b["name"]}, nil
		},
	}
}

func TestFactoryBuildsCrudAndRoutesFlags(t *testing.T) {
	setCrudTestEnv(t)
	var created map[string]any
	r := newWidgetResource(&created)
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

	// Drive the create leaf end-to-end: cobra flag parsing ->
	// registerFieldFlags -> bodyFromFlags -> the createFn closure ->
	// renderOne/renderTable. This is the part the structural checks above
	// don't exercise at all.
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"create", "--name", "widget-x"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute create: %v\noutput: %s", err, buf.String())
	}
	if created == nil || created["name"] != "widget-x" {
		t.Fatalf("createFn body = %#v, want name=%q", created, "widget-x")
	}
	if !strings.Contains(buf.String(), "widget-x") {
		t.Fatalf("output = %q, want it to contain the created resource", buf.String())
	}
}

// TestCreateLeafRequiresRequiredField checks that a Required field left
// unset on `create` (no flag, and no -f in Task 1's scope) is a clierr.Config
// error (exit code 2), not a silent empty-bodied create. Required
// enforcement is bodyFromFlags' job in this task; Task 2's
// bodyFromFlagsAndFile must preserve it once a -f document can also satisfy
// the field.
func TestCreateLeafRequiresRequiredField(t *testing.T) {
	setCrudTestEnv(t)
	var created map[string]any
	r := newWidgetResource(&created)
	cmd := newResourceCmd(r)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"create"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute create with no --name: want error, got nil (body=%#v)", created)
	}
	if code := clierr.Code(err); code != 2 {
		t.Fatalf("clierr.Code(%v) = %d, want 2 (clierr.Config)", err, code)
	}
	if created != nil {
		t.Fatalf("createFn must not be called when a required field is missing, got body=%#v", created)
	}
}
