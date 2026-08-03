package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProbeHTTPWorksWithoutConfig asserts the load-bearing property that
// distinguishes probe from every other command group: a plain measurement
// probe must not require any configured context, token, or API URL, since
// it never touches the SDK unless --save is passed.
func TestProbeHTTPWorksWithoutConfig(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer target.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", "")
	t.Setenv("SYSCHECKS_TOKEN", "")

	out := runCLIOut(t, "probe", "http", target.URL)
	if !strings.Contains(out, "204") {
		t.Fatalf("probe http output = %q, want it to contain status 204", out)
	}
}

func TestProbeDNS(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", "")
	t.Setenv("SYSCHECKS_TOKEN", "")

	out := runCLIOut(t, "probe", "dns", "localhost")
	if !strings.Contains(out, "127.0.0.1") && !strings.Contains(out, "::1") {
		t.Fatalf("probe dns output = %q, want a loopback address", out)
	}
}

// TestProbeHTTPSaveCreatesCheck exercises `probe http --save`: after
// measuring, it should POST to CreateNewCheck with the derived name, url,
// type, project id, and interval-in-seconds, then print the created check's
// id.
func TestProbeHTTPSaveCreatesCheck(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer target.Close()

	api := newFakeAPI(t)
	api.On("POST", "/api/checks/", 200, map[string]any{
		"id": 42, "name": "myhost", "status": "UP",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "probe", "http", target.URL, "--save", "--project", "7", "--interval", "30s")
	if !strings.Contains(out, "42") {
		t.Fatalf("probe http --save output = %q, want it to mention created check id 42", out)
	}
}

func TestProbeHTTPSaveWithoutProjectFails(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer target.Close()

	newFakeAPI(t)

	err := runCLIErr(t, "probe", "http", target.URL, "--save")
	if exitCode(err) != 2 {
		t.Fatalf("want exit code 2 for --save without --project, got %d (err=%v)", exitCode(err), err)
	}
}
