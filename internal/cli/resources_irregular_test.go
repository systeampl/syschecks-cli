package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestContactMethodListWorksWithoutOrg confirms contact-method's OrgNone
// contract: `list` needs no --org at all and hits the bare
// /api/profile/contact-methods path (contact methods are scoped to the
// authenticated user/token, not an organization).
func TestContactMethodListWorksWithoutOrg(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/contact-methods", 200, []map[string]any{
		{"id": 3, "kind": "email", "label": "work", "value": "me@example.com", "enabled": true, "verified": true, "created_at": "2026-08-05T00:00:00Z"},
	})

	out := runCLIOut(t, "contact-method", "list")

	if !strings.Contains(out, "email") || !strings.Contains(out, "me@example.com") {
		t.Fatalf("contact-method list output = %q", out)
	}
}

// TestContactMethodCreatePostsToProfilePath drives `contact-method create`
// end-to-end through the registry: the flags must land in a JSON body
// shaped like models.ContactMethodCreate (snake_case keys) POSTed to
// /api/profile/contact-methods, with no --org involved anywhere.
// CreateContactMethod only treats 201 as success (resources_gen.go checks
// resp.JSON201 != nil, not any 2xx), so the fake API must answer 201.
func TestContactMethodCreatePostsToProfilePath(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/profile/contact-methods", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding contact-method create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 3, "kind": "email", "label": "work", "value": "me@example.com",
			"enabled": true, "verified": false, "created_at": "2026-08-05T00:00:00Z",
		}
	})

	out := runCLIOut(t, "contact-method", "create", "--kind", "email", "--label", "work", "--value", "me@example.com")

	if gotBody == nil {
		t.Fatal("contact-method create never POSTed /api/profile/contact-methods")
	}
	if gotBody["kind"] != "email" {
		t.Fatalf("contact-method create body kind = %v, want email (body=%#v)", gotBody["kind"], gotBody)
	}
	if gotBody["value"] != "me@example.com" {
		t.Fatalf("contact-method create body value = %v, want me@example.com (body=%#v)", gotBody["value"], gotBody)
	}
	if !strings.Contains(out, "3") || !strings.Contains(out, "email") {
		t.Fatalf("contact-method create output = %q", out)
	}
}

// TestContactMethodUpdatePatchesProfilePath drives `contact-method update
// <id>`: it must PATCH /api/profile/contact-methods/{id} with a body shaped
// like models.ContactMethodUpdate (enabled/label only — UpdateContactMethod
// only treats 200 as success, resources_gen.go checks resp.JSON200 != nil).
func TestContactMethodUpdatePatchesProfilePath(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("PATCH", "/api/profile/contact-methods/3", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding contact-method update request body: %v", err)
		}
		return 200, map[string]any{
			"id": 3, "kind": "email", "label": "personal", "value": "me@example.com",
			"enabled": false, "verified": false, "created_at": "2026-08-05T00:00:00Z",
		}
	})

	out := runCLIOut(t, "contact-method", "update", "3", "--label", "personal")

	if gotBody == nil {
		t.Fatal("contact-method update never PATCHed /api/profile/contact-methods/3")
	}
	if gotBody["label"] != "personal" {
		t.Fatalf("contact-method update body label = %v, want personal (body=%#v)", gotBody["label"], gotBody)
	}
	if !strings.Contains(out, "personal") {
		t.Fatalf("contact-method update output = %q", out)
	}
}

// TestContactMethodDeleteHitsProfilePath drives `contact-method delete <id>
// --yes`: it must DELETE /api/profile/contact-methods/{id} directly, no org
// anywhere in the path.
func TestContactMethodDeleteHitsProfilePath(t *testing.T) {
	api := newFakeAPI(t)
	var deleteCalled bool
	api.OnRequest("DELETE", "/api/profile/contact-methods/3", func(*http.Request) (int, any) {
		deleteCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "contact-method", "delete", "3", "--yes")

	if !deleteCalled {
		t.Fatal("contact-method delete did not call DELETE /api/profile/contact-methods/3")
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("contact-method delete output = %q", out)
	}
}

// TestContactMethodHasNoGetSubcommand checks the SDK-shape contract
// directly: ContactMethods has no GetContactMethod endpoint at all, so
// getFn is nil and newResourceCmd (crud.go) must not generate a `get`
// leaf — there is nowhere for it to call.
func TestContactMethodHasNoGetSubcommand(t *testing.T) {
	c := newResourceCmd(registry["contact-method"])
	for _, sc := range c.Commands() {
		if sc.Name() == "get" {
			t.Fatalf("contact-method has a %q subcommand, want none (ContactMethods has no GetContactMethod endpoint)", sc.Name())
		}
	}
	if _, _, err := c.Find([]string{"get"}); err == nil {
		t.Fatal("contact-method: cobra resolved a \"get\" subcommand, want not found")
	}
}

// TestIntegrationKeyListGetsOrgIntegrationKeysPath drives `integration-key
// list --org acme`: the org slug must resolve to its numeric id and that id
// must land in the GET path (/api/organizations/{id}/integration-keys), not
// the slug itself. The response is untyped (json.RawMessage per the SDK),
// decoded via extractItems(raw, "integration_keys").
func TestIntegrationKeyListGetsOrgIntegrationKeysPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/integration-keys", 200, map[string]any{
		"integration_keys": []map[string]any{
			{"id": 9, "name": "prod-webhook", "key": "abcd1234", "created_at": "2026-08-05T00:00:00Z"},
		},
	})

	out := runCLIOut(t, "--org", "acme", "integration-key", "list")

	if api.query("GET", "/api/organizations/1/integration-keys") == nil {
		t.Fatal("integration-key list did not call GET /api/organizations/1/integration-keys (org id not resolved into the path)")
	}
	if !strings.Contains(out, "prod-webhook") {
		t.Fatalf("integration-key list output = %q", out)
	}
}

// TestIntegrationKeyListAcceptsBareArrayShape confirms extractItems'
// dual-shape handling covers the other plausible untyped response shape too
// (a bare JSON array, no wrapper object) — the exact wrapper key for this
// endpoint isn't confirmed against a real response (see the task report).
func TestIntegrationKeyListAcceptsBareArrayShape(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	api.On("GET", "/api/organizations/1/integration-keys", 200, []map[string]any{
		{"id": 9, "name": "prod-webhook", "key": "abcd1234", "created_at": "2026-08-05T00:00:00Z"},
	})

	out := runCLIOut(t, "--org", "acme", "integration-key", "list")

	if !strings.Contains(out, "prod-webhook") {
		t.Fatalf("integration-key list output (bare array shape) = %q", out)
	}
}

// TestIntegrationKeyCreatePostsToOrgIntegrationKeysPath drives
// `integration-key create` end-to-end through the registry: the flags must
// land in a JSON body shaped like models.IntegrationKeyCreate POSTed to
// /api/organizations/{id}/integration-keys. The response is untyped
// (json.RawMessage), decoded via json.Unmarshal into a plain map.
func TestIntegrationKeyCreatePostsToOrgIntegrationKeysPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/organizations/1/integration-keys", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding integration-key create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 9, "name": "prod-webhook", "key": "abcd1234", "created_at": "2026-08-05T00:00:00Z",
		}
	})

	out := runCLIOut(t, "--org", "acme", "integration-key", "create", "--name", "prod-webhook", "--escalation-policy-id", "5")

	if gotBody == nil {
		t.Fatal("integration-key create never POSTed /api/organizations/1/integration-keys")
	}
	if gotBody["name"] != "prod-webhook" {
		t.Fatalf("integration-key create body name = %v, want prod-webhook (body=%#v)", gotBody["name"], gotBody)
	}
	if gotBody["escalation_policy_id"] != float64(5) {
		t.Fatalf("integration-key create body escalation_policy_id = %v, want 5 (body=%#v)", gotBody["escalation_policy_id"], gotBody)
	}
	if !strings.Contains(out, "prod-webhook") {
		t.Fatalf("integration-key create output = %q", out)
	}
}

// TestIntegrationKeyDeleteRevokesOrgIntegrationKeysPath drives
// `integration-key delete <id> --yes`: deleteFn calls RevokeIntegrationKey,
// which per the generated SDK (NewRevokeIntegrationKeyRequest) is a real
// DELETE to /api/organizations/{id}/integration-keys/{key_id} — there is no
// separate "delete" endpoint, revoke is the only way to remove a key.
func TestIntegrationKeyDeleteRevokesOrgIntegrationKeysPath(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var revokeCalled bool
	api.OnRequest("DELETE", "/api/organizations/1/integration-keys/9", func(*http.Request) (int, any) {
		revokeCalled = true
		return 200, map[string]any{}
	})

	out := runCLIOut(t, "--org", "acme", "integration-key", "delete", "9", "--yes")

	if !revokeCalled {
		t.Fatal("integration-key delete did not call DELETE /api/organizations/1/integration-keys/9 (RevokeIntegrationKey)")
	}
	if !strings.Contains(out, "9") {
		t.Fatalf("integration-key delete output = %q", out)
	}
}

// TestIntegrationKeyRequiresOrg checks OrgArg's required-org contract for
// integration-key: an empty --org is a clierr.Config error (exit code 2),
// not a panic or a silent zero-value org id in the path.
func TestIntegrationKeyRequiresOrg(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "integration-key", "list"); exitCode(err) != 2 {
		t.Fatalf("integration-key list with no --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}

// TestIntegrationKeyHasNoGetOrUpdateSubcommand checks the SDK-shape
// contract directly: IntegrationKeys exposes no Get/Update endpoint, so
// getFn/updateFn are both nil and newResourceCmd (crud.go) must not
// generate either leaf.
func TestIntegrationKeyHasNoGetOrUpdateSubcommand(t *testing.T) {
	c := newResourceCmd(registry["integration-key"])
	for _, sc := range c.Commands() {
		if sc.Name() == "get" || sc.Name() == "update" {
			t.Fatalf("integration-key has a %q subcommand, want none (IntegrationKeys has no Get/Update endpoint)", sc.Name())
		}
	}
	if _, _, err := c.Find([]string{"get"}); err == nil {
		t.Fatal("integration-key: cobra resolved a \"get\" subcommand, want not found")
	}
	if _, _, err := c.Find([]string{"update"}); err == nil {
		t.Fatal("integration-key: cobra resolved an \"update\" subcommand, want not found")
	}
}
