package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestOrgListTable(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/", 200, []map[string]any{
		{"id": 1, "name": "Acme", "slug": "acme"},
	})

	out := runCLIOut(t, "org", "list")
	if !strings.Contains(out, "Acme") || !strings.Contains(out, "acme") {
		t.Fatalf("org list output = %q", out)
	}
}

func TestOrgGetBySlug(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})

	out := runCLIOut(t, "org", "get", "acme")
	if !strings.Contains(out, "Acme") || !strings.Contains(out, "acme") {
		t.Fatalf("org get output = %q", out)
	}
}

// TestOrgDeleteConfirmsWithTheOrgName covers the fix for a bug where `org
// delete` always sent an empty `confirm` query param: the API requires
// `confirm` to equal the organization's name, so a real backend rejected
// every delete. The deleteFn must first GET the org to learn its name, then
// send it as `confirm` on the DELETE — the user already passed the CLI's own
// --yes gate, so auto-supplying the fetched name is correct.
func TestOrgDeleteConfirmsWithTheOrgName(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "org", "delete", "1", "--yes")

	if !deleteCalled {
		t.Fatal("org delete did not call DELETE /api/organizations/1")
	}
	if got := api.query("DELETE", "/api/organizations/1").Get("confirm"); got != "Acme" {
		t.Fatalf("org delete sent confirm=%q, want %q (the org's own name)", got, "Acme")
	}
	if !strings.Contains(out, "1") {
		t.Fatalf("org delete output = %q", out)
	}
}

func TestProjectListResolvesOrgIDFromFlag(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1/projects", 200, []map[string]any{
		{"id": 10, "name": "web", "user_id": 1, "created_at": "2026-01-01T00:00:00Z"},
	})

	out := runCLIOut(t, "--org", "1", "project", "list")
	if !strings.Contains(out, "web") {
		t.Fatalf("project list output = %q", out)
	}
}

func TestProjectListResolvesOrgIDFromSlug(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/projects", 200, []map[string]any{
		{"id": 10, "name": "web", "user_id": 1, "created_at": "2026-01-01T00:00:00Z"},
	})

	out := runCLIOut(t, "--org", "acme", "project", "list")
	if !strings.Contains(out, "web") {
		t.Fatalf("project list output = %q", out)
	}
}

func TestProjectGet(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1/projects/10", 200, map[string]any{
		"id": 10, "name": "web", "description": "the web project",
		"user_id": 1, "created_at": "2026-01-01T00:00:00Z",
	})

	out := runCLIOut(t, "--org", "1", "project", "get", "10")
	if !strings.Contains(out, "web") || !strings.Contains(out, "the web project") {
		t.Fatalf("project get output = %q", out)
	}
}

func TestProjectListNoOrgFails(t *testing.T) {
	newFakeAPI(t)
	err := runCLIErr(t, "project", "list")
	if err == nil {
		t.Fatalf("want error when no --org given")
	}
	if exitCode(err) != 2 {
		t.Fatalf("want exit code 2, got %d", exitCode(err))
	}
}
