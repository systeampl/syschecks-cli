package cli

import (
	"encoding/json"
	"net/http"
	"testing"

	yaml "sigs.k8s.io/yaml"
)

// This file proves (and documents the limits of) the CRUD surface's headline
// invariant: `get -o yaml` output should be acceptable input to
// `apply -f -`, reproducing the same request body a direct `update` would
// have sent. Two real gaps surface along the way (see the per-test comments):
// no resource's `get -o yaml` currently emits a "kind" field, so a bare pipe
// (`syschecks check get 55 -o yaml | syschecks apply -f -`) cannot route
// itself yet -- a caller has to inject `kind: <resource>` first. Fixing that
// is out of this task's scope (flagged in the report); what's tested here is
// that once a document carries "kind" (+ "id" to select update over create),
// the rest of a real get response really does flow through apply unchanged
// into the same PUT body a direct `update` call would produce.

// TestRoundTripCheckGetYAMLToApplyUpdate covers `check`, an OrgParam-scoped
// resource whose `get` is a bespoke command (newCheckGetCmd in check.go),
// not the registry's generic getFn -- it renders only five hand-picked
// fields (id/name/type/status/url), not the full GetCheckDetails response.
func TestRoundTripCheckGetYAMLToApplyUpdate(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/checks/55", 200, map[string]any{
		"id": 55, "name": "web", "type": "http", "status": "UP", "url": "https://example.com/health",
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})

	out := runCLIOut(t, "check", "get", "55", "-o", "yaml")

	var rows []map[string]any
	if err := yaml.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("unmarshaling check get -o yaml output: %v\noutput: %s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("check get -o yaml output = %d rows, want 1 (output: %s)", len(rows), out)
	}
	row := rows[0]

	// Gap 1: no "kind" field at all -- apply cannot route this document
	// as-is. If this ever starts failing, get -o yaml has grown a kind
	// field and the concern in the Task 10 report is resolved; update both.
	if _, ok := row["kind"]; ok {
		t.Fatalf("check get -o yaml unexpectedly carries a kind field now: %#v", row)
	}
	// Gap 2: the bespoke check `get` only emits its five display columns,
	// not the full GetCheckDetails response, even though the fake API
	// returned created_at/updated_at/uuid too.
	for _, absent := range []string{"created_at", "updated_at", "uuid"} {
		if _, ok := row[absent]; ok {
			t.Fatalf("check get -o yaml unexpectedly carries %q now: %#v", absent, row)
		}
	}
	for _, want := range []string{"id", "name", "type", "status", "url"} {
		if _, ok := row[want]; !ok {
			t.Fatalf("check get -o yaml missing %q: %#v", want, row)
		}
	}

	// Build the document a script would have to construct today: get's row
	// plus the "kind" apply needs to route it (id is already in the row).
	doc := map[string]any{"kind": "check"}
	for k, v := range row {
		doc[k] = v
	}
	docYAML, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshaling apply document: %v", err)
	}

	var gotBody map[string]any
	api.OnRequest("PUT", "/api/checks/55", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding apply PUT body: %v", err)
		}
		return 200, map[string]any{
			"id": 55, "name": "web", "type": "http", "status": "UP", "url": "https://example.com/health",
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
		}
	})

	if _, err := runCLI(t, string(docYAML), "apply", "-f", "-"); err != nil {
		t.Fatalf("apply -f - (stdin): %v", err)
	}

	if gotBody == nil {
		t.Fatal("apply -f - never PUT /api/checks/55")
	}
	if gotBody["url"] != "https://example.com/health" {
		t.Fatalf("apply PUT body url = %v, want https://example.com/health (body=%#v)", gotBody["url"], gotBody)
	}
	if gotBody["name"] != "web" {
		t.Fatalf("apply PUT body name = %v, want web (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["type"] != "http" {
		t.Fatalf("apply PUT body type = %v, want http (body=%#v)", gotBody["type"], gotBody)
	}
	// status/id/kind must not leak into the update body: status isn't a
	// models.CheckUpdate field at all (mapToBody's JSON round-trip through
	// the SDK model silently drops anything the target struct doesn't
	// declare), id/kind are stripped by applyDoc itself before the call.
	if _, ok := gotBody["status"]; ok {
		t.Fatalf("apply PUT body must not carry status (not a CheckUpdate field): %#v", gotBody)
	}
	if _, ok := gotBody["id"]; ok {
		t.Fatalf("apply PUT body must not carry id: %#v", gotBody)
	}
	if _, ok := gotBody["kind"]; ok {
		t.Fatalf("apply PUT body must not carry kind: %#v", gotBody)
	}
}

// TestRoundTripTeamGetYAMLToApplyUpdate covers `team`, an OrgArg-scoped
// resource that (unlike check) uses the registry's generic getFn: its
// `get -o yaml` carries the full GetTeam response (toMap round-trip), not a
// hand-picked subset, so this is the "full fidelity" half of the invariant.
func TestRoundTripTeamGetYAMLToApplyUpdate(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/1/teams/7", 200, map[string]any{
		"id": 7, "name": "Platform", "slug": "platform", "organization_id": 1,
		"description": "owns the platform", "created_by_id": 1, "is_active": true,
		"member_count": 3, "integration_key_count": 0, "policy_count": 0,
		"project_count": 0, "schedule_count": 0, "members": []map[string]any{},
	})

	out := runCLIOut(t, "--org", "1", "team", "get", "7", "-o", "yaml")

	var rows []map[string]any
	if err := yaml.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("unmarshaling team get -o yaml output: %v\noutput: %s", err, out)
	}
	if len(rows) != 1 {
		t.Fatalf("team get -o yaml output = %d rows, want 1 (output: %s)", len(rows), out)
	}
	row := rows[0]

	if _, ok := row["kind"]; ok {
		t.Fatalf("team get -o yaml unexpectedly carries a kind field now: %#v", row)
	}
	// Unlike check's bespoke get, team's get is the registry's generic
	// getFn (toMap(GetTeam response)): it carries the FULL API response,
	// including fields Resource.Cols doesn't render in table form.
	for _, want := range []string{"id", "name", "slug", "description", "organization_id", "member_count"} {
		if _, ok := row[want]; !ok {
			t.Fatalf("team get -o yaml missing %q: %#v", want, row)
		}
	}

	doc := map[string]any{"kind": "team"}
	for k, v := range row {
		doc[k] = v
	}
	docYAML, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshaling apply document: %v", err)
	}

	var gotBody map[string]any
	api.OnRequest("PUT", "/api/organizations/1/teams/7", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding apply PUT body: %v", err)
		}
		return 200, map[string]any{
			"id": 7, "name": "Platform", "slug": "platform", "organization_id": 1,
		}
	})

	if _, err := runCLI(t, string(docYAML), "--org", "1", "apply", "-f", "-"); err != nil {
		t.Fatalf("apply -f - (stdin): %v", err)
	}

	if gotBody == nil {
		t.Fatal("apply -f - never PUT /api/organizations/1/teams/7")
	}
	if gotBody["name"] != "Platform" {
		t.Fatalf("apply PUT body name = %v, want Platform (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["description"] != "owns the platform" {
		t.Fatalf("apply PUT body description = %v, want %q (body=%#v)", gotBody["description"], "owns the platform", gotBody)
	}
	// slug is on the get response and on teamFields (so `team update --slug`
	// exists as a flag), but models.TeamUpdate has no Slug field at all --
	// mapToBody's JSON round-trip through TeamUpdate silently drops it, so a
	// slug carried over from a real get -o yaml never reaches the wire on
	// update. Genuine `-f`-driven gap, pre-existing (not introduced by
	// apply/round-trip), documented here rather than fixed.
	if _, ok := gotBody["slug"]; ok {
		t.Fatalf("apply PUT body must not carry slug (not a TeamUpdate field): %#v", gotBody)
	}
	if _, ok := gotBody["id"]; ok {
		t.Fatalf("apply PUT body must not carry id: %#v", gotBody)
	}
	if _, ok := gotBody["kind"]; ok {
		t.Fatalf("apply PUT body must not carry kind: %#v", gotBody)
	}
}
