package cli

import (
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
