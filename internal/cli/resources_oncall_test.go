package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestOncallScheduleCreatePostsToOrgSchedulesPath drives `oncall-schedule
// create` end-to-end through the registry: the flags must land in a JSON
// body shaped like models.ScheduleCreate (snake_case keys) POSTed to
// /api/organizations/{id}/oncall-schedules, with {id} being the numeric id
// resolved from the --org slug.
func TestOncallScheduleCreatePostsToOrgSchedulesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/oncall-schedules", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding oncall-schedule create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 9, "name": "Primary", "organization_id": 1, "created_by_id": 1,
			"is_active": true, "rotation_type": "weekly", "timezone": "UTC",
		}
	})

	out := runCLIOut(t, "--org", "acme", "oncall-schedule", "create", "--name", "Primary", "--rotation-type", "weekly", "--timezone", "UTC")

	if gotBody == nil {
		t.Fatal("oncall-schedule create never POSTed /api/organizations/1/oncall-schedules")
	}
	if gotBody["name"] != "Primary" {
		t.Fatalf("oncall-schedule create body name = %v, want Primary (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["rotation_type"] != "weekly" {
		t.Fatalf("oncall-schedule create body rotation_type = %v, want weekly (body=%#v)", gotBody["rotation_type"], gotBody)
	}
	if gotBody["timezone"] != "UTC" {
		t.Fatalf("oncall-schedule create body timezone = %v, want UTC (body=%#v)", gotBody["timezone"], gotBody)
	}
	if !strings.Contains(out, "9") || !strings.Contains(out, "Primary") {
		t.Fatalf("oncall-schedule create output = %q", out)
	}
}

// TestOncallScheduleListGetsOrgSchedulesPath drives `oncall-schedule list
// --org acme`: the org slug must resolve to its numeric id and that id must
// land in the GET path (/api/organizations/{id}/oncall-schedules), not the
// slug itself.
func TestOncallScheduleListGetsOrgSchedulesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/oncall-schedules", 200, []map[string]any{
		{"id": 9, "name": "Primary", "organization_id": 1, "created_by_id": 1,
			"is_active": true, "rotation_type": "weekly", "timezone": "UTC"},
	})

	out := runCLIOut(t, "--org", "acme", "oncall-schedule", "list")

	if api.query("GET", "/api/organizations/1/oncall-schedules") == nil {
		t.Fatal("oncall-schedule list did not call GET /api/organizations/1/oncall-schedules (org id not resolved into the path)")
	}
	if !strings.Contains(out, "Primary") {
		t.Fatalf("oncall-schedule list output = %q", out)
	}
}

// TestOncallScheduleDeleteHitsOrgSchedulesPath drives `oncall-schedule delete
// <id> --yes`: it must DELETE /api/organizations/{id}/oncall-schedules/{id},
// with {id} the numeric org id resolved from --org.
func TestOncallScheduleDeleteHitsOrgSchedulesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/oncall-schedules/9", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "oncall-schedule", "delete", "9", "--yes")

	if !deleteCalled {
		t.Fatal("oncall-schedule delete did not call DELETE /api/organizations/1/oncall-schedules/9")
	}
	if !strings.Contains(out, "9") {
		t.Fatalf("oncall-schedule delete output = %q", out)
	}
}

// TestEscalationPolicyCreatePostsToOrgPoliciesPath drives `escalation-policy
// create` end-to-end through the registry: the flags must land in a JSON
// body shaped like models.EscalationPolicyCreate (snake_case keys) POSTed to
// /api/organizations/{id}/escalation-policies, with {id} being the numeric
// id resolved from the --org slug.
func TestEscalationPolicyCreatePostsToOrgPoliciesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/escalation-policies", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding escalation-policy create request body: %v", err)
		}
		// Unlike oncall-schedule create (201), the generated SDK only treats
		// 200 as success for CreatePolicy (models.ParseCreatePolicyResponse
		// checks StatusCode == 200, not 201) — match the real contract.
		return 200, map[string]any{
			"id": 4, "name": "Default", "organization_id": 1, "is_active": true,
		}
	})

	out := runCLIOut(t, "--org", "acme", "escalation-policy", "create", "--name", "Default")

	if gotBody == nil {
		t.Fatal("escalation-policy create never POSTed /api/organizations/1/escalation-policies")
	}
	if gotBody["name"] != "Default" {
		t.Fatalf("escalation-policy create body name = %v, want Default (body=%#v)", gotBody["name"], gotBody)
	}
	if !strings.Contains(out, "4") || !strings.Contains(out, "Default") {
		t.Fatalf("escalation-policy create output = %q", out)
	}
}

// TestEscalationPolicyListGetsOrgPoliciesPath drives `escalation-policy list
// --org acme`: the org slug must resolve to its numeric id and that id must
// land in the GET path (/api/organizations/{id}/escalation-policies), not
// the slug itself.
func TestEscalationPolicyListGetsOrgPoliciesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/escalation-policies", 200, []map[string]any{
		{"id": 4, "name": "Default", "organization_id": 1, "is_active": true},
	})

	out := runCLIOut(t, "--org", "acme", "escalation-policy", "list")

	if api.query("GET", "/api/organizations/1/escalation-policies") == nil {
		t.Fatal("escalation-policy list did not call GET /api/organizations/1/escalation-policies (org id not resolved into the path)")
	}
	if !strings.Contains(out, "Default") {
		t.Fatalf("escalation-policy list output = %q", out)
	}
}

// TestEscalationPolicyDeleteHitsOrgPoliciesPath drives `escalation-policy
// delete <id> --yes`: it must DELETE
// /api/organizations/{id}/escalation-policies/{id}, with {id} the numeric
// org id resolved from --org.
func TestEscalationPolicyDeleteHitsOrgPoliciesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/escalation-policies/4", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "escalation-policy", "delete", "4", "--yes")

	if !deleteCalled {
		t.Fatal("escalation-policy delete did not call DELETE /api/organizations/1/escalation-policies/4")
	}
	if !strings.Contains(out, "4") {
		t.Fatalf("escalation-policy delete output = %q", out)
	}
}

// TestOncallScheduleAndEscalationPolicyRequireOrg checks OrgArg's
// required-org contract: an empty --org is a clierr.Config error (exit code
// 2), not a panic or a silent zero-value org id in the path.
func TestOncallScheduleAndEscalationPolicyRequireOrg(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "oncall-schedule", "list"); exitCode(err) != 2 {
		t.Fatalf("oncall-schedule list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
	if err := runCLIErr(t, "escalation-policy", "list"); exitCode(err) != 2 {
		t.Fatalf("escalation-policy list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
