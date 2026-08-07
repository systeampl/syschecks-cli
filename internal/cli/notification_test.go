package cli

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestNotificationCreateWithOrgSetsOrganizationID drives `notification
// create --org acme`: the POST /api/notification-channels/ body must carry
// organization_id set to the numeric id resolved from --org, so the channel
// lands in the organization instead of being created as a personal (no-org)
// channel. Regression test for a bug where the notification createFn
// silently discarded the resolved org.
func TestNotificationCreateWithOrgSetsOrganizationID(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/notification-channels/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding notification create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 9, "name": "Primary", "channel_type": "email",
			"organization_id": 1, "user_id": 1, "is_active": true,
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
		}
	})

	path := writeTempFile(t, "channel.yaml", "config:\n  address: ops@example.com\n")

	runCLIOut(t, "--org", "acme", "notification", "create",
		"--name", "Primary", "--channel-type", "email", "-f", path)

	if gotBody == nil {
		t.Fatal("notification create never POSTed /api/notification-channels/")
	}
	if gotBody["organization_id"] != float64(1) {
		t.Fatalf("notification create body organization_id = %v, want 1 (body=%#v)", gotBody["organization_id"], gotBody)
	}
}

// TestNotificationCreateWithoutOrgOmitsOrganizationID drives `notification
// create` with no --org: the POST body must not carry an organization_id at
// all, so the channel is created as a personal channel exactly as before —
// the org-injection fix must not regress the no-org case.
func TestNotificationCreateWithoutOrgOmitsOrganizationID(t *testing.T) {
	api := newFakeAPI(t)
	var gotBody map[string]any
	api.OnRequest("POST", "/api/notification-channels/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding notification create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 9, "name": "Primary", "channel_type": "email",
			"user_id": 1, "is_active": true,
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
		}
	})

	path := writeTempFile(t, "channel.yaml", "config:\n  address: ops@example.com\n")

	runCLIOut(t, "notification", "create",
		"--name", "Primary", "--channel-type", "email", "-f", path)

	if gotBody == nil {
		t.Fatal("notification create never POSTed /api/notification-channels/")
	}
	if _, ok := gotBody["organization_id"]; ok {
		t.Fatalf("notification create body has organization_id = %v, want it absent (no --org)", gotBody["organization_id"])
	}
}

// TestNotificationCreateExplicitOrganizationIDFlagWins drives `notification
// create --org acme --organization-id 2`: an explicit --organization-id flag
// must win over the org resolved from --org, matching how every other -f/
// flag field takes precedence over implicit values.
func TestNotificationCreateExplicitOrganizationIDFlagWins(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 1, "name": "Acme", "slug": "acme",
	})
	var gotBody map[string]any
	api.OnRequest("POST", "/api/notification-channels/", func(r *http.Request) (int, any) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding notification create request body: %v", err)
		}
		return 201, map[string]any{
			"id": 9, "name": "Primary", "channel_type": "email",
			"organization_id": 2, "user_id": 1, "is_active": true,
			"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
		}
	})

	path := writeTempFile(t, "channel.yaml", "config:\n  address: ops@example.com\n")

	runCLIOut(t, "--org", "acme", "notification", "create",
		"--name", "Primary", "--channel-type", "email", "--organization-id", "2", "-f", path)

	if gotBody == nil {
		t.Fatal("notification create never POSTed /api/notification-channels/")
	}
	if gotBody["organization_id"] != float64(2) {
		t.Fatalf("notification create body organization_id = %v, want 2 (explicit flag should win over --org)", gotBody["organization_id"])
	}
}
