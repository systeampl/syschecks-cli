package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestMaintenanceWindowCreatePostsToMaintenancePath drives `maintenance-window
// create` end-to-end through the registry: the flags must land in a JSON
// body shaped like models.MaintenanceWindowCreate (snake_case keys) POSTed to
// /api/maintenance/. Unlike oncall-schedule create (201), the generated SDK
// only treats 200 as success for CreateMaintenanceWindow
// (models.ParseCreateMaintenanceWindowResponse checks StatusCode == 200, not
// 201) — match the real contract.
func TestMaintenanceWindowCreatePostsToMaintenancePath(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/maintenance/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding maintenance-window create request body: %v", err)
		}
		return 200, map[string]any{
			"id": 3, "name": "DB upgrade", "start_time": "2026-08-06T00:00:00Z",
			"end_time": "2026-08-06T02:00:00Z", "is_active": true, "user_id": 1,
			"created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z",
		}
	})

	out := runCLIOut(t, "maintenance-window", "create", "--name", "DB upgrade",
		"--start-time", "2026-08-06T00:00:00Z", "--end-time", "2026-08-06T02:00:00Z")

	if gotBody == nil {
		t.Fatal("maintenance-window create never POSTed /api/maintenance/")
	}
	if gotBody["name"] != "DB upgrade" {
		t.Fatalf("maintenance-window create body name = %v, want %q (body=%#v)", gotBody["name"], "DB upgrade", gotBody)
	}
	if gotBody["start_time"] != "2026-08-06T00:00:00Z" {
		t.Fatalf("maintenance-window create body start_time = %v (body=%#v)", gotBody["start_time"], gotBody)
	}
	if gotBody["end_time"] != "2026-08-06T02:00:00Z" {
		t.Fatalf("maintenance-window create body end_time = %v (body=%#v)", gotBody["end_time"], gotBody)
	}
	if !strings.Contains(out, "3") || !strings.Contains(out, "DB upgrade") {
		t.Fatalf("maintenance-window create output = %q", out)
	}
}

// TestMaintenanceWindowListWorksWithoutOrg confirms maintenance-window's
// OrgParam contract: it is not org-scoped in the required sense (Org=OrgParam),
// so `list` with no --org at all must still succeed and send no
// organization_id filter.
func TestMaintenanceWindowListWorksWithoutOrg(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/maintenance/", 200, []map[string]any{
		{"id": 3, "name": "DB upgrade", "start_time": "2026-08-06T00:00:00Z",
			"end_time": "2026-08-06T02:00:00Z", "is_active": true, "user_id": 1,
			"created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z"},
	})

	out := runCLIOut(t, "maintenance-window", "list")

	if got := api.query("GET", "/api/maintenance/").Get("organization_id"); got != "" {
		t.Fatalf("maintenance-window list sent organization_id=%q, want it absent", got)
	}
	if !strings.Contains(out, "DB upgrade") {
		t.Fatalf("maintenance-window list output = %q", out)
	}
}

// TestMaintenanceWindowListSendsTheOrganizationFilter confirms --org, when
// given, actually reaches the wire: ListMaintenanceWindowsParams carries an
// OrganizationId field (json/query key organization_id, not org_id), and the
// listFn must wire orgID into it.
func TestMaintenanceWindowListSendsTheOrganizationFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/maintenance/", 200, []map[string]any{})

	runCLIOut(t, "maintenance-window", "list", "--org", "42")

	if got := api.query("GET", "/api/maintenance/").Get("organization_id"); got != "42" {
		t.Fatalf("maintenance-window list sent organization_id=%q, want %q", got, "42")
	}
}

// TestMaintenanceWindowDeleteHitsMaintenancePath drives `maintenance-window
// delete <id> --yes`: it must DELETE /api/maintenance/{id} directly — no
// organization in the path, since the item methods take only windowId.
func TestMaintenanceWindowDeleteHitsMaintenancePath(t *testing.T) {
	api := newFakeAPI(t)
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/maintenance/3", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "maintenance-window", "delete", "3", "--yes")

	if !deleteCalled {
		t.Fatal("maintenance-window delete did not call DELETE /api/maintenance/3")
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("maintenance-window delete output = %q", out)
	}
}

// TestPlaybookCreatePostsToOrgPlaybooksPath drives `playbook create`
// end-to-end through the registry: the flags must land in a JSON body shaped
// like models.PlaybookCreate (snake_case keys) POSTed to
// /api/organizations/{id}/playbooks, with {id} the numeric id resolved from
// the --org slug. CreatePlaybook expects 201
// (models.ParseCreatePlaybookResponse checks StatusCode == 201).
func TestPlaybookCreatePostsToOrgPlaybooksPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/playbooks", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding playbook create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 5, "name": "Restart service", "organization_id": 1, "created_by_id": 1,
			"is_active": true, "trigger_type": "manual", "suppress_default_notifications": false,
			"steps": []any{}, "created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z",
		}
	})

	out := runCLIOut(t, "--org", "acme", "playbook", "create", "--name", "Restart service", "--trigger-type", "manual")

	if gotBody == nil {
		t.Fatal("playbook create never POSTed /api/organizations/1/playbooks")
	}
	if gotBody["name"] != "Restart service" {
		t.Fatalf("playbook create body name = %v, want %q (body=%#v)", gotBody["name"], "Restart service", gotBody)
	}
	if gotBody["trigger_type"] != "manual" {
		t.Fatalf("playbook create body trigger_type = %v, want manual (body=%#v)", gotBody["trigger_type"], gotBody)
	}
	if !strings.Contains(out, "5") || !strings.Contains(out, "Restart service") {
		t.Fatalf("playbook create output = %q", out)
	}
}

// TestPlaybookListGetsOrgPlaybooksPath drives `playbook list --org acme`: the
// org slug must resolve to its numeric id and that id must land in the GET
// path (/api/organizations/{id}/playbooks), not the slug itself.
func TestPlaybookListGetsOrgPlaybooksPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/playbooks", 200, []map[string]any{
		{"id": 5, "name": "Restart service", "organization_id": 1, "created_by_id": 1,
			"is_active": true, "trigger_type": "manual", "suppress_default_notifications": false,
			"steps": []any{}, "created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z"},
	})

	out := runCLIOut(t, "--org", "acme", "playbook", "list")

	if api.query("GET", "/api/organizations/1/playbooks") == nil {
		t.Fatal("playbook list did not call GET /api/organizations/1/playbooks (org id not resolved into the path)")
	}
	if !strings.Contains(out, "Restart service") {
		t.Fatalf("playbook list output = %q", out)
	}
}

// TestPlaybookDeleteHitsOrgPlaybooksPath drives `playbook delete <id> --yes`:
// it must DELETE /api/organizations/{id}/playbooks/{id}, with {id} the
// numeric org id resolved from --org.
func TestPlaybookDeleteHitsOrgPlaybooksPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/playbooks/5", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "playbook", "delete", "5", "--yes")

	if !deleteCalled {
		t.Fatal("playbook delete did not call DELETE /api/organizations/1/playbooks/5")
	}
	if !strings.Contains(out, "5") {
		t.Fatalf("playbook delete output = %q", out)
	}
}

// TestPlaybookRequiresOrg checks OrgArg's required-org contract for playbook:
// an empty --org is a clierr.Config error (exit code 2), not a panic or a
// silent zero-value org id in the path.
func TestPlaybookRequiresOrg(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "playbook", "list"); exitCode(err) != 2 {
		t.Fatalf("playbook list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
