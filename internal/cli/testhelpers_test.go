package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/systeampl/syschecks-cli/internal/clierr"
)

// This file is the reusable CLI test harness: a fake API server plus
// runCLI/runCLIOut/runCLIErr wrappers. Later command test files (list/get/
// etc.) reuse it as-is — only the routes registered on the fake API differ.

// route is a canned (method, path) -> response the fake API replays.
type route struct {
	status int
	body   any
}

// fakeAPI is a minimal stand-in for the SysChecks REST API: tests register
// (method, path) -> (status, JSON body) routes on it; any unregistered
// request gets a 404.
type fakeAPI struct {
	*httptest.Server

	mu     sync.Mutex
	routes map[string]route
	seen   map[string]url.Values
}

// newFakeAPI starts a fake API server and wires the process environment so
// that any runCLI* call made later in the same test hits it: XDG_CONFIG_HOME
// points at a fresh temp dir (isolated config/token files) and
// SYSCHECKS_API_URL points at the server. It also seeds SYSCHECKS_TOKEN with
// a placeholder so commands that need *a* token (whoami, logout, ...) work
// without every test having to log in first; auth login ignores it and
// forces the token under validation instead (see cmdEnvWithToken).
func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{routes: map[string]route{}, seen: map[string]url.Values{}}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Server.Close)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", f.Server.URL)
	t.Setenv("SYSCHECKS_TOKEN", "placeholder-env-token")

	fakeAPIsMu.Lock()
	fakeAPIs[t] = f
	fakeAPIsMu.Unlock()
	t.Cleanup(func() {
		fakeAPIsMu.Lock()
		delete(fakeAPIs, t)
		fakeAPIsMu.Unlock()
	})

	return f
}

// On registers a canned response for method+path, replacing JSON body when
// the route is hit. Returns f for chaining: newFakeAPI(t).On(...).On(...).
func (f *fakeAPI) On(method, path string, status int, body any) *fakeAPI {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[method+" "+path] = route{status: status, body: body}
	return f
}

// query returns the query string the fake API last saw for method+path, so a
// test can assert on what the command actually sent (org scoping, filters).
func (f *fakeAPI) query(method, path string) url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seen[method+" "+path]
}

func (f *fakeAPI) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.seen[r.Method+" "+r.URL.Path] = r.URL.Query()
	rt, ok := f.routes[r.Method+" "+r.URL.Path]
	f.mu.Unlock()
	if !ok {
		http.Error(w, "no route registered for "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rt.status)
	if rt.body != nil {
		_ = json.NewEncoder(w).Encode(rt.body)
	}
}

// fakeAPIs lets runCLI find the fake API a test started, so it can resolve
// the "%SERVER%" placeholder in flag values without every call site having to
// thread the server URL through by hand.
var (
	fakeAPIsMu sync.Mutex
	fakeAPIs   = map[*testing.T]*fakeAPI{}
)

func serverURL(t *testing.T) string {
	fakeAPIsMu.Lock()
	defer fakeAPIsMu.Unlock()
	if f, ok := fakeAPIs[t]; ok {
		return f.Server.URL
	}
	return ""
}

// runCLI builds a fresh root command, feeds it stdin, executes it with args
// (substituting "%SERVER%" in any arg with the fake API URL registered via
// newFakeAPI, if one exists for this test), and returns captured
// stdout+stderr plus the Execute error.
func runCLI(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()

	url := serverURL(t)
	resolved := make([]string, len(args))
	for i, a := range args {
		resolved[i] = strings.ReplaceAll(a, "%SERVER%", url)
	}

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(resolved)

	err := root.Execute()
	return out.String(), err
}

// runCLIOut runs the CLI with no stdin and fails the test on a non-nil
// error, returning captured stdout for assertions on happy-path output.
func runCLIOut(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runCLI(t, "", args...)
	if err != nil {
		t.Fatalf("runCLI(%v): unexpected error: %v\noutput: %s", args, err, out)
	}
	return out
}

// runCLIErr runs the CLI with no stdin and returns the Execute error (nil on
// success), for tests asserting on failure behavior and exit codes.
func runCLIErr(t *testing.T, args ...string) error {
	t.Helper()
	_, err := runCLI(t, "", args...)
	return err
}

// exitCode maps a runCLI* error to the CLI's documented exit-code contract.
func exitCode(err error) int { return clierr.Code(err) }
