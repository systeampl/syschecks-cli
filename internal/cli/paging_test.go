package cli

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The API caps a page at 500 checks (and defaults to 100), so a single request
// silently returned a slice of a large organization. `check list` showed that
// slice with no indication, and resolveCheckID searched it for a name — so any
// check past the first page reported "no check named X found", which is a wrong
// answer rather than a failure.

func checkPage(offset, count int) []map[string]any {
	page := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		n := offset + i
		page = append(page, map[string]any{
			"id": n, "name": "check-" + strconv.Itoa(n), "type": "http", "status": "UP",
		})
	}
	return page
}

// pagedChecks serves `total` checks in pages of at most pageSize, honouring
// ?offset= the way the API does.
func pagedChecks(t *testing.T, api *fakeAPI, total, pageSize int) {
	t.Helper()
	api.OnRequest("GET", "/api/checks/", func(r *http.Request) (int, any) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset >= total {
			return 200, []map[string]any{}
		}
		n := total - offset
		if n > pageSize {
			n = pageSize
		}
		return 200, checkPage(offset, n)
	})
}

func TestCheckListReturnsEveryPage(t *testing.T) {
	api := newFakeAPI(t)
	pagedChecks(t, api, 501, 500)

	out := runCLIOut(t, "check", "list", "-q")

	if got := len(strings.Fields(out)); got != 501 {
		t.Fatalf("check list -q returned %d ids, want 501 (paging)", got)
	}
}

func TestCheckNameResolvesBeyondTheFirstPage(t *testing.T) {
	api := newFakeAPI(t)
	pagedChecks(t, api, 501, 500)
	api.On("POST", "/api/checks/500/run-now", 200, map[string]any{})

	out := runCLIOut(t, "check", "run", "check-500")

	if !strings.Contains(out, "500") {
		t.Fatalf("check run by name on page 2 output = %q", out)
	}
}

// A server that ignored ?offset= would hand back the same full page forever;
// the client must give up rather than spin.
func TestCheckListStopsWhenThePageDoesNotAdvance(t *testing.T) {
	api := newFakeAPI(t)
	api.OnRequest("GET", "/api/checks/", func(*http.Request) (int, any) {
		return 200, checkPage(0, 500)
	})

	done := make(chan string, 1)
	go func() { done <- runCLIOut(t, "check", "list", "-q") }()

	select {
	case <-done:
	case <-timeAfterTestTimeout():
		t.Fatal("check list did not terminate against a server ignoring offset")
	}
}

// The incident list is paged the same way, and `--status` filters client-side,
// so a truncated first page made the filter answer over a slice of the data.
func TestIncidentListFiltersOverEveryPage(t *testing.T) {
	api := newFakeAPI(t)
	api.OnRequest("GET", "/api/incidents", func(r *http.Request) (int, any) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		total := 250
		if offset >= total {
			return 200, map[string]any{"incidents": []map[string]any{}, "total": total}
		}
		n := total - offset
		if n > 200 {
			n = 200
		}
		items := make([]map[string]any, 0, n)
		for i := 0; i < n; i++ {
			idx := offset + i
			status := "UP"
			if idx == 240 { // only on the second page
				status = "DOWN"
			}
			items = append(items, map[string]any{
				"id": idx, "check_name": "c", "status": status, "started_at": "2026-01-01T00:00:00Z",
			})
		}
		return 200, map[string]any{"incidents": items, "total": total}
	})

	out := runCLIOut(t, "incident", "list", "--status", "DOWN", "-q")

	if got := strings.Fields(out); len(got) != 1 || got[0] != "240" {
		t.Fatalf("incident list --status DOWN returned %v, want [240] (found on page 2)", got)
	}
}

// timeAfterTestTimeout bounds the runaway-paging guard test.
func timeAfterTestTimeout() <-chan time.Time { return time.After(10 * time.Second) }
