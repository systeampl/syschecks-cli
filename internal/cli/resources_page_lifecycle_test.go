package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestStatusPageCreatePostsToStatusPagesPath drives `status-page create`
// end-to-end through the registry: the flags must land in a JSON body shaped
// like models.StatusPageCreate (snake_case keys) POSTed to
// /api/status-pages/. StatusPages is Org=OrgNone, so no --org is involved at
// all. CreateStatusPage only treats 200 as success
// (models.ParseCreateStatusPageResponse checks StatusCode == 200, not 201).
func TestStatusPageCreatePostsToStatusPagesPath(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/status-pages/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding status-page create request body: %v", err)
		}
		return 200, map[string]any{
			"id": 7, "name": "Public status", "slug": "public-status", "check_ids": []int{},
			"is_active": true, "is_public": true, "custom_domain": nil,
			"created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z", "user_id": 1,
		}
	})

	out := runCLIOut(t, "status-page", "create", "--name", "Public status", "--slug", "public-status", "--is-public")

	if gotBody == nil {
		t.Fatal("status-page create never POSTed /api/status-pages/")
	}
	if gotBody["name"] != "Public status" {
		t.Fatalf("status-page create body name = %v, want %q (body=%#v)", gotBody["name"], "Public status", gotBody)
	}
	if gotBody["slug"] != "public-status" {
		t.Fatalf("status-page create body slug = %v, want %q (body=%#v)", gotBody["slug"], "public-status", gotBody)
	}
	if gotBody["is_public"] != true {
		t.Fatalf("status-page create body is_public = %v, want true (body=%#v)", gotBody["is_public"], gotBody)
	}
	if !strings.Contains(out, "7") || !strings.Contains(out, "Public status") {
		t.Fatalf("status-page create output = %q", out)
	}
}

// TestStatusPageListWorksWithoutOrg confirms status-page's OrgNone contract:
// `list` needs no --org at all and hits the bare /api/status-pages/ path.
func TestStatusPageListWorksWithoutOrg(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/status-pages/", 200, []map[string]any{
		{"id": 7, "name": "Public status", "slug": "public-status", "check_ids": []int{},
			"is_active": true, "is_public": true, "custom_domain": nil,
			"created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z", "user_id": 1},
	})

	out := runCLIOut(t, "status-page", "list")

	if !strings.Contains(out, "Public status") {
		t.Fatalf("status-page list output = %q", out)
	}
}

// TestStatusPageGetHitsStatusPagesPath drives `status-page get <id>`: it must
// GET /api/status-pages/{id} directly — no organization in the path.
func TestStatusPageGetHitsStatusPagesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/status-pages/7", 200, map[string]any{
		"id": 7, "name": "Public status", "slug": "public-status", "check_ids": []int{},
		"is_active": true, "is_public": true, "custom_domain": nil,
		"created_at": "2026-08-05T00:00:00Z", "updated_at": "2026-08-05T00:00:00Z", "user_id": 1,
	})

	out := runCLIOut(t, "status-page", "get", "7")

	if !strings.Contains(out, "Public status") {
		t.Fatalf("status-page get output = %q", out)
	}
}

// TestStatusPageDeleteHitsStatusPagesPath drives `status-page delete <id>
// --yes`: it must DELETE /api/status-pages/{id} directly.
func TestStatusPageDeleteHitsStatusPagesPath(t *testing.T) {
	api := newFakeAPI(t)
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/status-pages/7", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "status-page", "delete", "7", "--yes")

	if !deleteCalled {
		t.Fatal("status-page delete did not call DELETE /api/status-pages/7")
	}
	if !strings.Contains(out, "7") {
		t.Fatalf("status-page delete output = %q", out)
	}
}

// TestLifecycleWatchCreatePostsUpsert drives `lifecycle-watch create`
// end-to-end through the registry: there is no separate create endpoint, so
// the flags must land in a JSON body shaped like
// models.LifecycleWatchUpsert POSTed to
// /api/organizations/{id}/lifecycle-watches (UpsertLifecycleWatch is a POST,
// not a PUT — see NewUpsertLifecycleWatchRequestWithBody in the generated
// SDK), with {id} the numeric id resolved from the --org slug. The endpoint
// returns an untyped body, decoded by upsertLifecycleWatch into a plain map.
func TestLifecycleWatchCreatePostsUpsert(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/lifecycle-watches", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding lifecycle-watch create request body: %v", err)
		}
		return 200, map[string]any{
			"id": 11, "organization_id": 1, "user_id": 1, "vendor": "aws",
			"resource_type": "rds-instance", "resource_id": "db-1", "platform": "postgres",
			"notify_on_new": true, "notify_7d": true, "notify_30d": true, "notify_90d": false,
		}
	})

	out := runCLIOut(t, "--org", "acme", "lifecycle-watch", "create",
		"--vendor", "aws", "--resource-type", "rds-instance", "--resource-id", "db-1")

	if gotBody == nil {
		t.Fatal("lifecycle-watch create never POSTed /api/organizations/1/lifecycle-watches")
	}
	if gotBody["vendor"] != "aws" {
		t.Fatalf("lifecycle-watch create body vendor = %v, want aws (body=%#v)", gotBody["vendor"], gotBody)
	}
	if gotBody["resource_type"] != "rds-instance" {
		t.Fatalf("lifecycle-watch create body resource_type = %v, want rds-instance (body=%#v)", gotBody["resource_type"], gotBody)
	}
	if !strings.Contains(out, "11") || !strings.Contains(out, "aws") {
		t.Fatalf("lifecycle-watch create output = %q", out)
	}
}

// TestLifecycleWatchUpdateAlsoPostsUpsert drives `lifecycle-watch update <id>`
// and confirms the irregular contract: update ignores the id path arg
// entirely and reuses the very same upsert POST as create (there is no PUT
// /api/organizations/{id}/lifecycle-watches/{id} in the generated SDK).
func TestLifecycleWatchUpdateAlsoPostsUpsert(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/lifecycle-watches", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding lifecycle-watch update request body: %v", err)
		}
		return 200, map[string]any{
			"id": 11, "organization_id": 1, "user_id": 1, "vendor": "aws",
			"resource_type": "rds-instance", "resource_id": "db-1", "platform": "postgres",
			"notify_on_new": true, "notify_7d": true, "notify_30d": true, "notify_90d": true,
		}
	})

	out := runCLIOut(t, "--org", "acme", "lifecycle-watch", "update", "11",
		"--vendor", "aws", "--resource-type", "rds-instance", "--resource-id", "db-1", "--notify-90d")

	if gotBody == nil {
		t.Fatal("lifecycle-watch update never POSTed /api/organizations/1/lifecycle-watches")
	}
	if gotBody["notify_90d"] != true {
		t.Fatalf("lifecycle-watch update body notify_90d = %v, want true (body=%#v)", gotBody["notify_90d"], gotBody)
	}
	if !strings.Contains(out, "11") {
		t.Fatalf("lifecycle-watch update output = %q", out)
	}
}

// TestLifecycleWatchListGetsOrgLifecycleWatchesPath drives `lifecycle-watch
// list --org acme`: the org slug must resolve to its numeric id and that id
// must land in the GET path (/api/organizations/{id}/lifecycle-watches), not
// the slug itself.
func TestLifecycleWatchListGetsOrgLifecycleWatchesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/lifecycle-watches", 200, []map[string]any{
		{"id": 11, "organization_id": 1, "user_id": 1, "vendor": "aws",
			"resource_type": "rds-instance", "resource_id": "db-1", "platform": "postgres",
			"notify_on_new": true, "notify_7d": true, "notify_30d": true, "notify_90d": false},
	})

	out := runCLIOut(t, "--org", "acme", "lifecycle-watch", "list")

	if api.query("GET", "/api/organizations/1/lifecycle-watches") == nil {
		t.Fatal("lifecycle-watch list did not call GET /api/organizations/1/lifecycle-watches (org id not resolved into the path)")
	}
	if !strings.Contains(out, "aws") {
		t.Fatalf("lifecycle-watch list output = %q", out)
	}
}

// TestLifecycleWatchDeleteHitsOrgLifecycleWatchesPath drives `lifecycle-watch
// delete <id> --yes`: it must DELETE
// /api/organizations/{id}/lifecycle-watches/{id}, with {id} the numeric org
// id resolved from --org.
func TestLifecycleWatchDeleteHitsOrgLifecycleWatchesPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/lifecycle-watches/11", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "lifecycle-watch", "delete", "11", "--yes")

	if !deleteCalled {
		t.Fatal("lifecycle-watch delete did not call DELETE /api/organizations/1/lifecycle-watches/11")
	}
	if !strings.Contains(out, "11") {
		t.Fatalf("lifecycle-watch delete output = %q", out)
	}
}

// TestLifecycleWatchRequiresOrg checks OrgArg's required-org contract for
// lifecycle-watch: an empty --org is a clierr.Config error (exit code 2), not
// a panic or a silent zero-value org id in the path.
func TestLifecycleWatchRequiresOrg(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "lifecycle-watch", "list"); exitCode(err) != 2 {
		t.Fatalf("lifecycle-watch list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
