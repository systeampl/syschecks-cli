package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/systeampl/syschecks-cli/internal/clierr"
)

// registerTestResource registers r in the package-level registry for the
// duration of the test and removes it (and its aliases) on cleanup, so
// synthetic resources used to test the apply mechanism never leak into other
// tests that iterate the registry.
func registerTestResource(t *testing.T, r *Resource) {
	t.Helper()
	register(r)
	t.Cleanup(func() {
		delete(registry, r.Name)
		for _, a := range r.Aliases {
			delete(registry, a)
		}
	})
}

// writeTempFile writes content to a new file under t.TempDir() and returns
// its path, for handing to `-f`.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", p, err)
	}
	return p
}

// TestApplyDispatchesMultiDocToCreateAndUpdate covers the apply mechanism
// against two synthetic in-memory resources (real check/notification-channel
// resources don't exist until Task 3): a two-document file, one doc with no
// id (must route to createFn) and one with an id (must route to updateFn).
func TestApplyDispatchesMultiDocToCreateAndUpdate(t *testing.T) {
	setCrudTestEnv(t)

	var created map[string]any
	widget := &Resource{
		Name: "widget", Cols: []string{"id", "name"}, Org: OrgNone,
		Fields: []Field{{Name: "name", JSONKey: "name", Kind: "string"}},
		createFn: func(env *cmdCtx, o *int, b map[string]any) (map[string]any, error) {
			created = b
			return map[string]any{"id": 1, "name": b["name"]}, nil
		},
	}
	registerTestResource(t, widget)

	var updatedID int
	var updatedBody map[string]any
	gadget := &Resource{
		Name: "gadget", Cols: []string{"id", "label"}, Org: OrgNone,
		Fields: []Field{{Name: "label", JSONKey: "label", Kind: "string"}},
		updateFn: func(env *cmdCtx, o *int, id int, b map[string]any) (map[string]any, error) {
			updatedID = id
			updatedBody = b
			return map[string]any{"id": id, "label": b["label"]}, nil
		},
	}
	registerTestResource(t, gadget)

	doc := "kind: widget\nname: widget-one\n---\nkind: gadget\nid: 5\nlabel: gadget-five\n"
	path := writeTempFile(t, "apply.yaml", doc)

	cmd := newApplyCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-f", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply -f %s: %v\noutput: %s", path, err, buf.String())
	}

	if created == nil || created["name"] != "widget-one" {
		t.Fatalf("createFn body = %#v, want name=widget-one", created)
	}
	if updatedID != 5 {
		t.Fatalf("updateFn id = %d, want 5", updatedID)
	}
	if updatedBody == nil || updatedBody["label"] != "gadget-five" {
		t.Fatalf("updateFn body = %#v, want label=gadget-five", updatedBody)
	}
	// The id/kind bookkeeping fields must not leak into the request body.
	if _, ok := updatedBody["id"]; ok {
		t.Fatalf("updateFn body must not contain id: %#v", updatedBody)
	}
	if _, ok := updatedBody["kind"]; ok {
		t.Fatalf("updateFn body must not contain kind: %#v", updatedBody)
	}
}

// TestApplyUnknownKindIsConfigError checks that a document whose kind has no
// registry entry fails with clierr.Config (exit code 2), not a panic or a
// silent no-op.
func TestApplyUnknownKindIsConfigError(t *testing.T) {
	setCrudTestEnv(t)
	path := writeTempFile(t, "apply.yaml", "kind: no-such-resource\nname: x\n")

	cmd := newApplyCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-f", path})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("apply with unknown kind: want error, got nil")
	}
	if code := clierr.Code(err); code != 2 {
		t.Fatalf("clierr.Code(%v) = %d, want 2 (clierr.Config)", err, code)
	}
}

// TestApplyRequiresFileFlag checks that omitting -f is itself a config error
// rather than a nil-dereference or a silent no-op.
func TestApplyRequiresFileFlag(t *testing.T) {
	setCrudTestEnv(t)
	cmd := newApplyCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("apply with no -f: want error, got nil")
	}
	if code := clierr.Code(err); code != 2 {
		t.Fatalf("clierr.Code(%v) = %d, want 2 (clierr.Config)", err, code)
	}
}

// TestBodyFromFlagsAndFileFlagOverridesFile drives the real `create` leaf
// (through the factory, exactly like TestFactoryBuildsCrudAndRoutesFlags in
// crud_test.go) with both -f and an overriding flag set, and checks the flag
// wins the merge while any file-only field still comes through.
func TestBodyFromFlagsAndFileFlagOverridesFile(t *testing.T) {
	setCrudTestEnv(t)
	var created map[string]any
	r := &Resource{
		Name: "widget2", Cols: []string{"id", "name", "note"}, Org: OrgNone,
		Fields: []Field{
			{Name: "name", JSONKey: "name", Kind: "string", Required: true},
			{Name: "note", JSONKey: "note", Kind: "string"},
		},
		createFn: func(env *cmdCtx, o *int, b map[string]any) (map[string]any, error) {
			created = b
			return map[string]any{"id": 9, "name": b["name"], "note": b["note"]}, nil
		},
	}
	cmd := newResourceCmd(r)

	path := writeTempFile(t, "partial.yaml", "name: file-name\nnote: file-note\n")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"create", "--name", "flag-name", "-f", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute create -f: %v\noutput: %s", err, buf.String())
	}
	if created["name"] != "flag-name" {
		t.Fatalf("created[name] = %v, want flag-name (flag must win over file)", created["name"])
	}
	if created["note"] != "file-note" {
		t.Fatalf("created[note] = %v, want file-note (file-only field must come through)", created["note"])
	}
}

// TestBodyFromFlagsAndFileSatisfiesRequiredFromFile checks that a required
// field supplied only via -f (no matching flag set) satisfies the
// required-field check inherited from Task 1's bodyFromFlags.
func TestBodyFromFlagsAndFileSatisfiesRequiredFromFile(t *testing.T) {
	setCrudTestEnv(t)
	var created map[string]any
	r := &Resource{
		Name: "widget3", Cols: []string{"id", "name"}, Org: OrgNone,
		Fields: []Field{{Name: "name", JSONKey: "name", Kind: "string", Required: true}},
		createFn: func(env *cmdCtx, o *int, b map[string]any) (map[string]any, error) {
			created = b
			return map[string]any{"id": 9, "name": b["name"]}, nil
		},
	}
	cmd := newResourceCmd(r)
	path := writeTempFile(t, "full.yaml", "name: file-only-name\n")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"create", "-f", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute create -f (required satisfied by file): %v\noutput: %s", err, buf.String())
	}
	if created["name"] != "file-only-name" {
		t.Fatalf("created[name] = %v, want file-only-name", created["name"])
	}
}
