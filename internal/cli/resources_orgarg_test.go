package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestTeamCreatePostsToOrgTeamsPath drives `team create` end-to-end through
// the registry: the flags must land in a JSON body shaped like
// models.TeamCreate (snake_case keys) POSTed to
// /api/organizations/{id}/teams, with {id} being the numeric id resolved
// from the --org slug.
func TestTeamCreatePostsToOrgTeamsPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/teams", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding team create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 42, "name": "Platform", "slug": "platform", "organization_id": 1,
			"created_by_id": 1, "is_active": true, "member_count": 0,
			"integration_key_count": 0, "policy_count": 0, "project_count": 0, "schedule_count": 0,
		}
	})

	out := runCLIOut(t, "--org", "acme", "team", "create", "--name", "Platform", "--slug", "platform")

	if gotBody == nil {
		t.Fatal("team create never POSTed /api/organizations/1/teams")
	}
	if gotBody["name"] != "Platform" {
		t.Fatalf("team create body name = %v, want Platform (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["slug"] != "platform" {
		t.Fatalf("team create body slug = %v, want platform (body=%#v)", gotBody["slug"], gotBody)
	}
	if !strings.Contains(out, "42") || !strings.Contains(out, "Platform") {
		t.Fatalf("team create output = %q", out)
	}
}

// TestTeamListGetsOrgTeamsPath drives `team list --org acme`: the org slug
// must resolve to its numeric id and that id must land in the GET path
// (/api/organizations/{id}/teams), not the slug itself.
func TestTeamListGetsOrgTeamsPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/teams", 200, []map[string]any{
		{"id": 7, "name": "Platform", "slug": "platform", "organization_id": 1,
			"created_by_id": 1, "is_active": true, "member_count": 3,
			"integration_key_count": 0, "policy_count": 0, "project_count": 0, "schedule_count": 0},
	})

	out := runCLIOut(t, "--org", "acme", "team", "list")

	if api.query("GET", "/api/organizations/1/teams") == nil {
		t.Fatal("team list did not call GET /api/organizations/1/teams (org id not resolved into the path)")
	}
	if !strings.Contains(out, "Platform") {
		t.Fatalf("team list output = %q", out)
	}
}

// TestServiceDeleteHitsOrgServicesPath drives `service delete <id> --yes`:
// it must DELETE /api/organizations/{id}/services/{id}, with {id} the
// numeric org id resolved from --org.
func TestServiceDeleteHitsOrgServicesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/services/5", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "service", "delete", "5", "--yes")

	if !deleteCalled {
		t.Fatal("service delete did not call DELETE /api/organizations/1/services/5")
	}
	if !strings.Contains(out, "5") {
		t.Fatalf("service delete output = %q", out)
	}
}

// TestServiceListRendersHealthAndCountColumns drives `service list`: the
// rendered table must show health_status/checks_count, which only exist on
// models.ServiceListItemResponse (Resource.ListCols), not on the
// get/create/update response shapes (Resource.Cols).
func TestServiceListRendersHealthAndCountColumns(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/services", 200, []map[string]any{
		{"id": 3, "name": "checkout", "slug": "checkout", "organization_id": 1,
			"is_active": true, "health_status": "healthy", "checks_count": 5, "down_checks": 0, "degraded_checks": 0},
	})

	out := runCLIOut(t, "--org", "acme", "service", "list")

	for _, want := range []string{"health_status", "checks_count", "healthy", "checkout"} {
		if !strings.Contains(out, want) {
			t.Fatalf("service list output missing %q; output = %q", want, out)
		}
	}
}

// TestServiceGetOutputExcludesListOnlyColumns drives `service get <id>`: it
// renders models.ServiceDetailResponse, which has no health_status field
// (checks_count exists there but not in ServiceResponse from create/update,
// so Resource.Cols deliberately excludes both) — the get table must not
// carry a health_status column at all.
func TestServiceGetOutputExcludesListOnlyColumns(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/services/3", 200, map[string]any{
		"id": 3, "name": "checkout", "slug": "checkout", "organization_id": 1,
		"is_active": true, "tier": "tier-1", "checks_count": 5,
	})

	out := runCLIOut(t, "--org", "acme", "service", "get", "3")

	if strings.Contains(out, "health_status") {
		t.Fatalf("service get output must not contain the list-only health_status column: %q", out)
	}
	if !strings.Contains(out, "checkout") || !strings.Contains(out, "tier-1") {
		t.Fatalf("service get output = %q, want it to contain the detail fields", out)
	}
}

// TestTeamAndServiceRequireOrg checks OrgArg's required-org contract: an
// empty --org is a clierr.Config error (exit code 2), not a panic or a
// silent zero-value org id in the path.
func TestTeamAndServiceRequireOrg(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "team", "list"); exitCode(err) != 2 {
		t.Fatalf("team list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
	if err := runCLIErr(t, "service", "list"); exitCode(err) != 2 {
		t.Fatalf("service list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
