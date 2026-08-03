package cli

import (
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
