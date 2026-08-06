package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestIncidentListStatusFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, []map[string]any{
		{"start_log_id": 1, "max_status": "DOWN"},
		{"start_log_id": 2, "max_status": "DEGRADED"},
	})

	out := runCLIOut(t, "incident", "list", "--status", "DOWN")
	if !strings.Contains(out, "1") {
		t.Fatalf("incident list --status DOWN output = %q (want id 1)", out)
	}
	if strings.Contains(out, "DEGRADED") {
		t.Fatalf("incident list --status DOWN output = %q (should not contain filtered-out row)", out)
	}
}

func TestIncidentListNoFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, []map[string]any{
		{"start_log_id": 1, "check_name": "web", "max_status": "DOWN", "started_at": "2026-01-01T00:00:00Z"},
		{"start_log_id": 2, "check_name": "db", "max_status": "DEGRADED", "started_at": "2026-01-02T00:00:00Z"},
	})

	out := runCLIOut(t, "incident", "list")
	if !strings.Contains(out, "DOWN") || !strings.Contains(out, "DEGRADED") {
		t.Fatalf("incident list output = %q (want both statuses)", out)
	}
}

func TestAgentList(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1/agents", 200, []map[string]any{
		{"id": 1, "name": "agent-a", "status": "online", "last_seen": "2026-01-01T00:00:00Z"},
	})

	out := runCLIOut(t, "--org", "1", "agent", "list")
	if !strings.Contains(out, "agent-a") {
		t.Fatalf("agent list output = %q", out)
	}
}

// The real API wraps the list in an object (GET /api/incidents ->
// {total,offset,limit,incidents:[...]}); the CLI must handle that, not only a
// bare array. Regression test for the wrapped shape found in local testing.
func TestIncidentListWrappedShape(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, map[string]any{
		"total": 2, "offset": 0, "limit": 50,
		"incidents": []map[string]any{
			{"start_log_id": 1, "max_status": "DOWN"},
			{"start_log_id": 2, "max_status": "DEGRADED"},
		},
	})

	out := runCLIOut(t, "incident", "list", "--status", "DOWN")
	if !strings.Contains(out, "1") || strings.Contains(out, "DEGRADED") {
		t.Fatalf("wrapped incident list output = %q (want id 1, not the filtered-out row)", out)
	}
}

// GET /api/organizations/{id}/agents -> {agents:[...],count,max_private_agents}.
func TestAgentListWrappedShape(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1/agents", 200, map[string]any{
		"agents":             []map[string]any{{"id": 1, "name": "agent-a", "status": "online"}},
		"count":              1,
		"max_private_agents": 5,
	})

	out := runCLIOut(t, "--org", "1", "agent", "list")
	if !strings.Contains(out, "agent-a") {
		t.Fatalf("wrapped agent list output = %q", out)
	}
}

func TestNotificationList(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/notification-channels/", 200, []map[string]any{
		{"id": 1, "name": "slack-alerts", "channel_type": "slack", "is_active": true},
	})

	out := runCLIOut(t, "notification", "list")
	if !strings.Contains(out, "slack-alerts") || !strings.Contains(out, "slack") {
		t.Fatalf("notification list output = %q", out)
	}
}

func TestNotificationTest(t *testing.T) {
	api := newFakeAPI(t)
	api.On("POST", "/api/notification-channels/1/test", 200, map[string]any{
		"success": true, "message": "delivered",
	})

	out := runCLIOut(t, "notification", "test", "1")
	if !strings.Contains(out, "true") || !strings.Contains(out, "delivered") {
		t.Fatalf("notification test output = %q", out)
	}
}

// TestIncidentGet drives `incident get <check_id> <log_id>`: it must GET
// /api/checks/{check_id}/incidents/{log_id} and render the decoded (untyped)
// response object.
func TestIncidentGet(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/5/incidents/9", 200, map[string]any{
		"start_log_id": 9, "check_id": 5, "max_status": "DOWN", "check_name": "web",
	})

	out := runCLIOut(t, "incident", "get", "5", "9")
	if !strings.Contains(out, "DOWN") || !strings.Contains(out, "web") {
		t.Fatalf("incident get output = %q", out)
	}
}

// TestIncidentAcknowledge drives `incident acknowledge <check_id> <log_id>
// --note`: it must POST /api/checks/{check_id}/incidents/{log_id}/ack with
// {"note": "..."} in the body.
func TestIncidentAcknowledge(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/checks/5/incidents/9/ack", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding incident acknowledge request body: %v", err)
		}
		return 200, map[string]any{"start_log_id": 9, "acknowledged": true, "note": "hi"}
	})

	out := runCLIOut(t, "incident", "acknowledge", "5", "9", "--note", "hi")

	if gotBody["note"] != "hi" {
		t.Fatalf("incident acknowledge body note = %v, want hi (body=%#v)", gotBody["note"], gotBody)
	}
	if !strings.Contains(out, "true") {
		t.Fatalf("incident acknowledge output = %q", out)
	}
}

// TestIncidentAcknowledgeAlias checks the `ack` alias reaches the same
// command as `acknowledge`.
func TestIncidentAcknowledgeAlias(t *testing.T) {
	api := newFakeAPI(t)
	var called bool
	api.OnRequest("POST", "/api/checks/5/incidents/9/ack", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{"start_log_id": 9, "acknowledged": true}
	})

	runCLIOut(t, "incident", "ack", "5", "9")

	if !called {
		t.Fatal("incident ack alias did not call POST /api/checks/5/incidents/9/ack")
	}
}

// TestIncidentAcknowledgeNoNote checks that an empty --note posts a body
// without a "note" key at all (the field is optional in
// AcknowledgeIncidentJSONRequestBody), not an empty string.
func TestIncidentAcknowledgeNoNote(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/checks/5/incidents/9/ack", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding incident acknowledge request body: %v", err)
		}
		return 200, map[string]any{"start_log_id": 9, "acknowledged": true}
	})

	runCLIOut(t, "incident", "acknowledge", "5", "9")

	if _, ok := gotBody["note"]; ok {
		t.Fatalf("incident acknowledge body must not carry an unset note: %#v", gotBody)
	}
}

// TestIncidentResolve drives `incident resolve <check_id> <log_id>`: it must
// POST /api/checks/{check_id}/incidents/{log_id}/resolve with no body.
func TestIncidentResolve(t *testing.T) {
	api := newFakeAPI(t)
	var called bool
	api.OnRequest("POST", "/api/checks/5/incidents/9/resolve", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{"start_log_id": 9, "max_status": "RESOLVED"}
	})

	out := runCLIOut(t, "incident", "resolve", "5", "9")

	if !called {
		t.Fatal("incident resolve did not call POST /api/checks/5/incidents/9/resolve")
	}
	if !strings.Contains(out, "RESOLVED") {
		t.Fatalf("incident resolve output = %q", out)
	}
}

// TestAgentToken drives `agent token --org acme`: it must POST
// /api/organizations/{id}/agents/registration-token and print the token from
// the response.
func TestAgentToken(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var called bool
	api.OnRequest("POST", "/api/organizations/1/agents/registration-token", func(*http.Request) (int, any) {
		called = true
		return 201, map[string]any{
			"token": "reg-tok-123", "expires_at": "2026-01-01T00:00:00Z",
			"install_curl": "curl ...", "install_docker": "docker run ...",
		}
	})

	out := runCLIOut(t, "--org", "acme", "agent", "token")

	if !called {
		t.Fatal("agent token did not call POST /api/organizations/1/agents/registration-token")
	}
	if !strings.Contains(out, "reg-tok-123") {
		t.Fatalf("agent token output = %q (want the token)", out)
	}
}

// TestAgentDelete drives `agent delete --org acme 3 --yes`: it must DELETE
// /api/organizations/{id}/agents/{agent_id}.
func TestAgentDelete(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var called bool
	api.OnRequest("DELETE", "/api/organizations/1/agents/3", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "agent", "delete", "3", "--yes")

	if !called {
		t.Fatal("agent delete did not call DELETE /api/organizations/1/agents/3")
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("agent delete output = %q", out)
	}
}

// TestAgentDeleteAlias checks the `rm` alias reaches the same command as
// `delete`.
func TestAgentDeleteAlias(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var called bool
	api.OnRequest("DELETE", "/api/organizations/1/agents/3", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{}
	})

	runCLIOut(t, "--org", "acme", "agent", "rm", "3", "--yes")

	if !called {
		t.Fatal("agent rm alias did not call DELETE /api/organizations/1/agents/3")
	}
}

// TestAgentDeleteWithoutYesRefusesNonInteractive checks that `agent delete`
// without --yes and with no TTY to prompt on refuses rather than deleting
// (same contract as the registry's generic delete).
func TestAgentDeleteWithoutYesRefusesNonInteractive(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var called bool
	api.OnRequest("DELETE", "/api/organizations/1/agents/3", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{}
	})

	err := runCLIErr(t, "--org", "acme", "agent", "delete", "3")

	if called {
		t.Fatal("agent delete without --yes must not call the API")
	}
	if exitCode(err) != 2 {
		t.Fatalf("agent delete without --yes: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
