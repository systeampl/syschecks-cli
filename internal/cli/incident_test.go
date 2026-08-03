package cli

import (
	"strings"
	"testing"
)

func TestIncidentListStatusFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, []map[string]any{
		{"id": 1, "status": "open"},
		{"id": 2, "status": "resolved"},
	})

	out := runCLIOut(t, "incident", "list", "--status", "open")
	if !strings.Contains(out, "1") {
		t.Fatalf("incident list --status open output = %q (want id 1)", out)
	}
	if strings.Contains(out, "resolved") {
		t.Fatalf("incident list --status open output = %q (should not contain filtered-out row)", out)
	}
}

func TestIncidentListNoFilter(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, []map[string]any{
		{"id": 1, "check": "web", "status": "open", "started_at": "2026-01-01T00:00:00Z"},
		{"id": 2, "check": "db", "status": "resolved", "started_at": "2026-01-02T00:00:00Z"},
	})

	out := runCLIOut(t, "incident", "list")
	if !strings.Contains(out, "open") || !strings.Contains(out, "resolved") {
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
