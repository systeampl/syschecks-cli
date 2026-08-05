package cli

import (
	"strings"
	"testing"
)

// The incident list rendered four columns from keys the API does not return: `id`,
// `check` and `status` are always absent, so every row printed empty values and
// `-q` printed "<nil>". The existing fixtures invented those keys, which is why the
// suite stayed green while the command showed nothing useful against production.
//
// The real payload (GET /api/incidents) carries start_log_id, check_name, max_status
// and started_at.

func realIncident(id int, checkName, status string) map[string]any {
	return map[string]any{
		"start_log_id": id,
		"check_id":     307,
		"check_name":   checkName,
		"check_type":   "uptime",
		"max_status":   status,
		"started_at":   "2026-07-16T21:50:10Z",
		"ended_at":     "2026-07-19T11:31:54Z",
		"resolved":     true,
	}
}

func TestIncidentListShowsTheRealFields(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, map[string]any{
		"total": 1, "offset": 0, "limit": 200,
		"incidents": []map[string]any{realIncident(2774001, "VERIFY-native-chat", "DOWN")},
	})

	out := runCLIOut(t, "incident", "list")

	for _, want := range []string{"2774001", "VERIFY-native-chat", "DOWN"} {
		if !strings.Contains(out, want) {
			t.Fatalf("incident list output = %q, missing %q", out, want)
		}
	}
	if strings.Contains(out, "<nil>") {
		t.Fatalf("incident list rendered nil cells: %q", out)
	}
}

func TestIncidentListStatusFilterMatchesTheRealField(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/incidents", 200, map[string]any{
		"total": 2, "offset": 0, "limit": 200,
		"incidents": []map[string]any{
			realIncident(1, "web", "DOWN"),
			realIncident(2, "db", "DEGRADED"),
		},
	})

	out := runCLIOut(t, "incident", "list", "--status", "DOWN", "-q")

	if got := strings.Fields(out); len(got) != 1 || got[0] != "1" {
		t.Fatalf("incident list --status DOWN returned %v, want [1]", got)
	}
}
