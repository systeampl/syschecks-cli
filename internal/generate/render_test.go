package generate

import (
	"strings"
	"testing"
)

// TestRenderResourceGolden is the golden test from the Task 3 brief: only
// non-null spec attrs are emitted, in spec order, with read-only attrs
// (status/id/created_at) excluded because they simply aren't in the check
// spec — and the `=` signs are aligned to the longest emitted attr name.
func TestRenderResourceGolden(t *testing.T) {
	attrs := map[string]any{
		"name":       "web",
		"type":       "http",
		"project_id": 3,
		"url":        "https://x",
		"interval":   300,
		"status":     "UP",
		"id":         5,
		"created_at": "2026-01-01T00:00:00Z",
	}

	block, vars, err := renderResource("check", 5, "web", attrs)
	if err != nil {
		t.Fatalf("renderResource error: %v", err)
	}

	// spec.Attrs order (see schema.go) places interval before url, so the
	// emitted block follows that order even though the brief's illustrative
	// example listed url first.
	want := `resource "systeam_check" "web" {
  name       = "web"
  type       = "http"
  project_id = 3
  interval   = 300
  url        = "https://x"
}`
	if block != want {
		t.Errorf("renderResource block =\n%s\nwant\n%s", block, want)
	}
	if len(vars) != 0 {
		t.Errorf("renderResource vars = %v, want none (no secret attrs present)", vars)
	}
}

func TestRenderImportGolden(t *testing.T) {
	got := renderImport("systeam_check", "web", 5)
	want := "import {\n  to = systeam_check.web\n  id = \"5\"\n}"
	if got != want {
		t.Errorf("renderImport =\n%s\nwant\n%s", got, want)
	}
}

// TestRenderResourceUnknownSpec ensures a missing spec is a returned error,
// not a panic — generate must degrade gracefully on an unrecognized kind.
func TestRenderResourceUnknownSpec(t *testing.T) {
	_, _, err := renderResource("nonexistent-kind", 1, "x", map[string]any{})
	if err == nil {
		t.Fatal("renderResource with unknown spec: want error, got nil")
	}
}

// TestRenderResourceSecretScalarAttr checks that a Secret attr (check's
// db_password) never renders its plaintext value — it becomes a var
// reference, and the variable declaration is returned alongside the block.
func TestRenderResourceSecretScalarAttr(t *testing.T) {
	attrs := map[string]any{
		"name":        "pg",
		"type":        "database",
		"db_password": "hunter2",
	}

	block, vars, err := renderResource("check", 9, "pg", attrs)
	if err != nil {
		t.Fatalf("renderResource error: %v", err)
	}

	if got := "hunter2"; strings.Contains(block, got) {
		t.Errorf("renderResource block leaked plaintext secret: %s", block)
	}

	wantLine := "db_password = var.pg_db_password"
	if !strings.Contains(block, wantLine) {
		t.Errorf("renderResource block =\n%s\nwant line containing %q", block, wantLine)
	}

	wantVar := "variable \"pg_db_password\" {\n  type      = string\n  sensitive = true\n}"
	if !containsString(vars, wantVar) {
		t.Errorf("renderResource vars = %v, want to contain %q", vars, wantVar)
	}
}

// TestRenderResourceSecretMapKey covers the notification-channel `config`
// map attr: non-secret keys render as plain HCL values, but any key listed
// in spec.SecretMapKeys (e.g. webhook_url) becomes a var reference instead
// of the real value, with its own variable declaration collected.
func TestRenderResourceSecretMapKey(t *testing.T) {
	attrs := map[string]any{
		"name":         "slack-alerts",
		"channel_type": "webhook",
		"config": map[string]any{
			"webhook_url": "https://hooks.example.com/T000/B000/XXXX",
			"channel":     "#alerts",
		},
		"organization_id": 7,
	}

	block, vars, err := renderResource("notification-channel", 12, "slack_alerts", attrs)
	if err != nil {
		t.Fatalf("renderResource error: %v", err)
	}

	if strings.Contains(block, "https://hooks.example.com/T000/B000/XXXX") {
		t.Errorf("renderResource block leaked plaintext webhook_url: %s", block)
	}

	wantLine := "webhook_url = var.slack_alerts_webhook_url"
	if !strings.Contains(block, wantLine) {
		t.Errorf("renderResource block =\n%s\nwant line containing %q", block, wantLine)
	}
	wantNonSecretLine := `channel = "#alerts"`
	if !strings.Contains(block, wantNonSecretLine) {
		t.Errorf("renderResource block =\n%s\nwant line containing %q", block, wantNonSecretLine)
	}

	wantVar := "variable \"slack_alerts_webhook_url\" {\n  type      = string\n  sensitive = true\n}"
	if !containsString(vars, wantVar) {
		t.Errorf("renderResource vars = %v, want to contain %q", vars, wantVar)
	}
}

func TestRenderResourceSkipsMissingAndNilAttrs(t *testing.T) {
	attrs := map[string]any{
		"organization_id": 1,
		"name":            "core",
		"slug":            nil, // present but nil -> must be skipped
		// "description" absent entirely -> must be skipped
	}

	block, _, err := renderResource("team", 2, "core", attrs)
	if err != nil {
		t.Fatalf("renderResource error: %v", err)
	}
	want := `resource "systeam_team" "core" {
  organization_id = 1
  name            = "core"
}`
	if block != want {
		t.Errorf("renderResource block =\n%s\nwant\n%s", block, want)
	}
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
