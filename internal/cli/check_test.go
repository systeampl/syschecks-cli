package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestCheckRunWaitDownReturnsExitCode1 is the load-bearing CI contract for
// this task: `check run <id> --wait` must exit 1 when the check settles DOWN.
func TestCheckRunWaitDownReturnsExitCode1(t *testing.T) {
	api := newFakeAPI(t)
	api.On("POST", "/api/checks/5/run-now", 200, map[string]any{})
	api.On("GET", "/api/checks/5", 200, map[string]any{
		"id": 5, "name": "x", "status": "DOWN",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	err := runCLIErr(t, "check", "run", "5", "--wait", "--timeout", "2s")
	if got := exitCode(err); got != 1 {
		t.Fatalf("exit code = %d, want 1 (assertion failed): err=%v", got, err)
	}
}

func TestCheckRunWaitUpSucceeds(t *testing.T) {
	api := newFakeAPI(t)
	api.On("POST", "/api/checks/7/run-now", 200, map[string]any{})
	api.On("GET", "/api/checks/7", 200, map[string]any{
		"id": 7, "name": "y", "status": "UP",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "run", "7", "--wait", "--timeout", "2s")
	if !strings.Contains(out, "UP") {
		t.Fatalf("check run --wait output = %q", out)
	}
}

func TestCheckRunNoWaitTriggersOnly(t *testing.T) {
	api := newFakeAPI(t)
	api.On("POST", "/api/checks/9/run-now", 200, map[string]any{})

	out := runCLIOut(t, "check", "run", "9")
	if !strings.Contains(out, "9") {
		t.Fatalf("check run output = %q", out)
	}
}

func TestCheckList(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 1, "name": "web", "type": "http", "status": "UP"},
		{"id": 2, "name": "db", "type": "tcp", "status": "DOWN"},
	})

	out := runCLIOut(t, "check", "list")
	if !strings.Contains(out, "web") || !strings.Contains(out, "db") {
		t.Fatalf("check list output = %q", out)
	}
}

func TestCheckGetByID(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/3", 200, map[string]any{
		"id": 3, "name": "web", "type": "http", "status": "UP", "url": "https://example.com",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "get", "3")
	if !strings.Contains(out, "web") || !strings.Contains(out, "https://example.com") {
		t.Fatalf("check get output = %q", out)
	}
}

func TestCheckGetByName(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 1, "name": "web", "type": "http", "status": "UP"},
		{"id": 2, "name": "db", "type": "tcp", "status": "DOWN"},
	})
	api.On("GET", "/api/checks/1", 200, map[string]any{
		"id": 1, "name": "web", "type": "http", "status": "UP", "url": "https://example.com",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "get", "web")
	if !strings.Contains(out, "https://example.com") {
		t.Fatalf("check get by name output = %q", out)
	}
}

func TestCheckGetByNameAmbiguous(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 1, "name": "web", "type": "http", "status": "UP"},
		{"id": 2, "name": "web", "type": "tcp", "status": "DOWN"},
	})

	err := runCLIErr(t, "check", "get", "web")
	if exitCode(err) != 2 {
		t.Fatalf("want exit code 2 for ambiguous name, got %d (err=%v)", exitCode(err), err)
	}
}

func TestCheckGetByNameNotFound(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/", 200, []map[string]any{})

	err := runCLIErr(t, "check", "get", "nope")
	if exitCode(err) != 2 {
		t.Fatalf("want exit code 2 for not-found name, got %d (err=%v)", exitCode(err), err)
	}
}

func TestCheckPause(t *testing.T) {
	api := newFakeAPI(t)
	api.On("PUT", "/api/checks/4", 200, map[string]any{
		"id": 4, "name": "web", "status": "PAUSED",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "pause", "4")
	if !strings.Contains(out, "4") {
		t.Fatalf("check pause output = %q", out)
	}
}

func TestCheckResume(t *testing.T) {
	api := newFakeAPI(t)
	api.On("PUT", "/api/checks/4", 200, map[string]any{
		"id": 4, "name": "web", "status": "UP",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "resume", "4")
	if !strings.Contains(out, "4") {
		t.Fatalf("check resume output = %q", out)
	}
}

func TestCheckTestAlert(t *testing.T) {
	api := newFakeAPI(t)
	api.On("POST", "/api/checks/4/test-alert", 200, map[string]any{})

	out := runCLIOut(t, "check", "test-alert", "4")
	if !strings.Contains(out, "4") {
		t.Fatalf("check test-alert output = %q", out)
	}
}

// TestCheckCreatePostsCheckCreateBody drives `check create` end-to-end
// through the registry: the flags must land in a JSON body shaped like
// models.CheckCreate (snake_case keys) POSTed to /api/checks/, and the
// rendered output must reflect the API's response.
func TestCheckCreatePostsCheckCreateBody(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/checks/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding check create request body: %v", err)
		}
		return 200, map[string]any{
			"id": 42, "name": "web", "status": "UP",
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
		}
	})

	out := runCLIOut(t, "check", "create", "--name", "web", "--project-id", "7", "--url", "https://example.com", "--interval", "60")

	if gotBody["name"] != "web" {
		t.Fatalf("check create body name = %v, want web (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["project_id"] != float64(7) {
		t.Fatalf("check create body project_id = %v, want 7 (body=%#v)", gotBody["project_id"], gotBody)
	}
	if gotBody["url"] != "https://example.com" {
		t.Fatalf("check create body url = %v, want https://example.com (body=%#v)", gotBody["url"], gotBody)
	}
	if gotBody["interval"] != float64(60) {
		t.Fatalf("check create body interval = %v, want 60 (body=%#v)", gotBody["interval"], gotBody)
	}
	if !strings.Contains(out, "42") || !strings.Contains(out, "web") {
		t.Fatalf("check create output = %q", out)
	}
}

// TestCheckCreateRequiresNameAndProjectID checks that CheckCreate's two
// required fields (name, project_id) are enforced before the SDK is ever
// called: create with neither is a clierr.Config error (exit code 2).
func TestCheckCreateRequiresNameAndProjectID(t *testing.T) {
	newFakeAPI(t)
	err := runCLIErr(t, "check", "create", "--url", "https://example.com")
	if exitCode(err) != 2 {
		t.Fatalf("check create with no --name/--project-id: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}

// TestCheckUpdatePutsBody drives `check update <id>` through the registry:
// the changed flag must land in a PUT to /api/checks/{id}.
func TestCheckUpdatePutsBody(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("PUT", "/api/checks/4", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding check update request body: %v", err)
		}
		return 200, map[string]any{
			"id": 4, "name": "web", "status": "UP",
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
		}
	})

	out := runCLIOut(t, "check", "update", "4", "--interval", "30")

	if gotBody["interval"] != float64(30) {
		t.Fatalf("check update body interval = %v, want 30 (body=%#v)", gotBody["interval"], gotBody)
	}
	if _, ok := gotBody["name"]; ok {
		t.Fatalf("check update body must not carry unset fields: %#v", gotBody)
	}
	if !strings.Contains(out, "4") {
		t.Fatalf("check update output = %q", out)
	}
}

// TestCheckDelete drives `check delete <id>` through the registry: it must
// DELETE /api/checks/{id} and report the deleted id.
func TestCheckDelete(t *testing.T) {
	api := newFakeAPI(t)
	var called bool
	api.OnRequest("DELETE", "/api/checks/4", func(*http.Request) (int, any) {
		called = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "check", "delete", "4", "--yes")

	if !called {
		t.Fatal("check delete did not call DELETE /api/checks/4")
	}
	if !strings.Contains(out, "4") {
		t.Fatalf("check delete output = %q", out)
	}
}
