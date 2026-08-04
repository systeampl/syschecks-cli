package cli

import "testing"

// The listing commands take an organization from --org or the active context,
// but `check list`, `incident list` and `notification list` called the SDK with
// nil parameters, so the organization never reached the wire: every one of them
// returned whatever the token could see, identically for `--org a` and `--org b`.
//
// These assert on the query string the command actually sent, which is the only
// place the bug is visible — the rendered table looks the same either way.

func TestCheckListSendsTheOrganizationFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 1, "name": "web", "type": "http", "status": "UP"},
	})

	runCLIOut(t, "check", "list", "--org", "42")

	if got := api.query("GET", "/api/checks/").Get("org_id"); got != "42" {
		t.Fatalf("check list sent org_id=%q, want %q", got, "42")
	}
}

func TestIncidentListSendsTheOrganizationFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, map[string]any{"incidents": []map[string]any{}, "total": 0})

	runCLIOut(t, "incident", "list", "--org", "42")

	if got := api.query("GET", "/api/incidents").Get("organization_id"); got != "42" {
		t.Fatalf("incident list sent organization_id=%q, want %q", got, "42")
	}
}

func TestNotificationListSendsTheOrganizationFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/notification-channels/", 200, []map[string]any{})

	runCLIOut(t, "notification", "list", "--org", "42")

	if got := api.query("GET", "/api/notification-channels/").Get("org_id"); got != "42" {
		t.Fatalf("notification list sent org_id=%q, want %q", got, "42")
	}
}

// A slug has to be resolved to an id first — the same path `project list` takes.
func TestCheckListResolvesAnOrganizationSlug(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{"id": 7, "name": "Acme", "slug": "acme"})
	api.On("GET", "/api/checks/", 200, []map[string]any{})

	runCLIOut(t, "check", "list", "--org", "acme")

	if got := api.query("GET", "/api/checks/").Get("org_id"); got != "7" {
		t.Fatalf("check list sent org_id=%q, want %q (slug resolved)", got, "7")
	}
}

// Without an organization the command keeps listing everything the token can
// reach — scoping is what --org asks for, not a new requirement.
func TestCheckListWithoutAnOrganizationSendsNoFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{})

	runCLIOut(t, "check", "list")

	if got := api.query("GET", "/api/checks/").Get("org_id"); got != "" {
		t.Fatalf("check list sent org_id=%q, want it absent", got)
	}
}
