package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readGenFile reads path within dir, failing the test if it's missing.
func readGenFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}

// assertFileAbsent fails the test if path exists within dir.
func assertFileAbsent(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		t.Fatalf("%s unexpectedly written", name)
	}
}

// seedGenerateOrg registers the org-slug-resolution route every generate
// test needs.
func seedGenerateOrg(api *fakeAPI) {
	api.On("GET", "/api/organizations/by-slug/acme", 200, map[string]any{
		"id": 7, "name": "Acme", "slug": "acme",
	})
}

func seedGenerateCheck(api *fakeAPI) {
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 101, "name": "web", "type": "http", "status": "UP"},
	})
	api.On("GET", "/api/checks/101", 200, map[string]any{
		"id": 101, "name": "web", "type": "http", "status": "UP",
		"project_id": 55, "url": "https://example.com", "interval": 60, "is_active": true,
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "uuid": "u",
	})
}

const generateSecretWebhook = "https://hooks.example.com/T00/B00/verysecret"

func seedGenerateNotification(api *fakeAPI) {
	channel := map[string]any{
		"id": 202, "name": "pager", "channel_type": "webhook", "is_active": true, "organization_id": 7,
		"config":     map[string]any{"webhook_url": generateSecretWebhook},
		"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "user_id": 1,
	}
	api.On("GET", "/api/notification-channels/", 200, []map[string]any{channel})
	api.On("GET", "/api/notification-channels/202", 200, channel)
}

func seedGenerateTeam(api *fakeAPI) {
	api.On("GET", "/api/organizations/7/teams", 200, []map[string]any{
		{"id": 303, "name": "Platform", "slug": "platform", "organization_id": 7,
			"created_by_id": 1, "is_active": true, "member_count": 0,
			"integration_key_count": 0, "policy_count": 0, "project_count": 0, "schedule_count": 0},
	})
	api.On("GET", "/api/organizations/7/teams/303", 200, map[string]any{
		"id": 303, "name": "Platform", "slug": "platform", "organization_id": 7,
		"created_by_id": 1, "is_active": true, "member_count": 0,
		"integration_key_count": 0, "policy_count": 0, "project_count": 0, "schedule_count": 0,
		"members": []map[string]any{},
	})
}

// TestGenerateTerraformWritesFullOrg drives `generate terraform --org acme`
// against one check, one notification channel (carrying a webhook secret in
// its config map), and one team, and asserts the whole generated directory:
// provider.tf, one .tf file per type, imports.tf, variables.tf, and that the
// secret never appears in plaintext anywhere in the output.
func TestGenerateTerraformWritesFullOrg(t *testing.T) {
	api := newFakeAPI(t)
	seedGenerateOrg(api)
	seedGenerateCheck(api)
	seedGenerateNotification(api)
	seedGenerateTeam(api)

	dir := t.TempDir()
	out, err := runCLI(t, "", "generate", "terraform", "--org", "acme", "--out", dir)
	if err != nil {
		t.Fatalf("generate terraform: unexpected error: %v\noutput: %s", err, out)
	}

	providerTF := readGenFile(t, dir, "provider.tf")
	if !strings.Contains(providerTF, `source  = "systeampl/systeam"`) {
		t.Fatalf("provider.tf missing provider source: %s", providerTF)
	}

	checksTF := readGenFile(t, dir, "checks.tf")
	if !strings.Contains(checksTF, `resource "systeam_check"`) {
		t.Fatalf("checks.tf missing check resource block: %s", checksTF)
	}

	channelsTF := readGenFile(t, dir, "notification_channels.tf")
	if !strings.Contains(channelsTF, `resource "systeam_notification_channel"`) {
		t.Fatalf("notification_channels.tf missing channel resource block: %s", channelsTF)
	}
	if strings.Contains(channelsTF, generateSecretWebhook) {
		t.Fatalf("notification_channels.tf leaks the webhook secret in plaintext: %s", channelsTF)
	}

	teamsTF := readGenFile(t, dir, "teams.tf")
	if !strings.Contains(teamsTF, `resource "systeam_team"`) {
		t.Fatalf("teams.tf missing team resource block: %s", teamsTF)
	}

	importsTF := readGenFile(t, dir, "imports.tf")
	for _, want := range []string{"systeam_check", "systeam_notification_channel", "systeam_team"} {
		if !strings.Contains(importsTF, want) {
			t.Fatalf("imports.tf missing import for %s: %s", want, importsTF)
		}
	}
	if got := strings.Count(importsTF, "import {"); got != 3 {
		t.Fatalf("imports.tf has %d import blocks, want 3: %s", got, importsTF)
	}

	variablesTF := readGenFile(t, dir, "variables.tf")
	if !strings.Contains(variablesTF, "sensitive = true") {
		t.Fatalf("variables.tf missing a sensitive variable: %s", variablesTF)
	}
	if strings.Contains(variablesTF, generateSecretWebhook) {
		t.Fatalf("variables.tf leaks the webhook secret in plaintext: %s", variablesTF)
	}

	// No plaintext secret anywhere in the output directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading output dir: %v", err)
	}
	for _, e := range entries {
		content := readGenFile(t, dir, e.Name())
		if strings.Contains(content, generateSecretWebhook) {
			t.Fatalf("%s leaks the webhook secret in plaintext: %s", e.Name(), content)
		}
	}

	if !strings.Contains(out, "WARNING") {
		t.Fatalf("generate terraform output = %q, want a WARNING about sensitive variables", out)
	}

	for method := range api.seenMethods() {
		if method != "GET" {
			t.Fatalf("generate terraform issued a %s request -- it must be strictly read-only", method)
		}
	}
}

// TestGenerateTerraformTypeFilterLimitsFiles asserts `--type check` narrows
// both which resource kinds are listed at all (no team/notification routes
// registered here -- an unwanted listFn call would 404 and fail the test)
// and which files land in the output directory.
func TestGenerateTerraformTypeFilterLimitsFiles(t *testing.T) {
	api := newFakeAPI(t)
	seedGenerateOrg(api)
	seedGenerateCheck(api)

	dir := t.TempDir()
	runCLIOut(t, "generate", "terraform", "--org", "acme", "--type", "check", "--out", dir)

	if _, err := os.Stat(filepath.Join(dir, "checks.tf")); err != nil {
		t.Fatalf("checks.tf missing: %v", err)
	}
	assertFileAbsent(t, dir, "teams.tf")
	assertFileAbsent(t, dir, "notification_channels.tf")

	importsTF := readGenFile(t, dir, "imports.tf")
	if strings.Contains(importsTF, "systeam_team") || strings.Contains(importsTF, "systeam_notification_channel") {
		t.Fatalf("imports.tf contains out-of-scope imports with --type check: %s", importsTF)
	}
	if !strings.Contains(importsTF, "systeam_check") {
		t.Fatalf("imports.tf missing the in-scope check import: %s", importsTF)
	}
}

// TestGenerateTerraformCheckFilterLimitsIDs asserts --check narrows which
// checks are rendered: only the requested id's getFn should even be called.
func TestGenerateTerraformCheckFilterLimitsIDs(t *testing.T) {
	api := newFakeAPI(t)
	seedGenerateOrg(api)
	api.On("GET", "/api/checks/", 200, []map[string]any{
		{"id": 101, "name": "web", "type": "http", "status": "UP"},
		{"id": 102, "name": "api", "type": "http", "status": "UP"},
	})
	api.On("GET", "/api/checks/101", 200, map[string]any{
		"id": 101, "name": "web", "type": "http", "status": "UP", "project_id": 55,
	})
	// Deliberately no route for /api/checks/102: if the filter fails to
	// exclude it, the getFn call 404s and the test fails loudly.

	dir := t.TempDir()
	runCLIOut(t, "generate", "terraform", "--org", "acme", "--type", "check", "--check", "101", "--out", dir)

	checksTF := readGenFile(t, dir, "checks.tf")
	if !strings.Contains(checksTF, `"web"`) {
		t.Fatalf("checks.tf missing check 101: %s", checksTF)
	}
	if strings.Contains(checksTF, `"api"`) {
		t.Fatalf("checks.tf unexpectedly contains excluded check 102: %s", checksTF)
	}
}

func TestGenerateTerraformRequiresOrg(t *testing.T) {
	newFakeAPI(t)
	err := runCLIErr(t, "generate", "terraform", "--out", t.TempDir())
	if exitCode(err) != 2 {
		t.Fatalf("generate terraform without --org: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}

func TestGenerateTerraformRequiresOut(t *testing.T) {
	api := newFakeAPI(t)
	seedGenerateOrg(api)
	err := runCLIErr(t, "generate", "terraform", "--org", "acme")
	if exitCode(err) != 2 {
		t.Fatalf("generate terraform without --out: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}

func TestGenerateTerraformRejectsUnknownType(t *testing.T) {
	api := newFakeAPI(t)
	seedGenerateOrg(api)
	err := runCLIErr(t, "generate", "terraform", "--org", "acme", "--type", "bogus", "--out", t.TempDir())
	if exitCode(err) != 2 {
		t.Fatalf("generate terraform --type bogus: exit code = %d, want 2 (err=%v)", exitCode(err), err)
	}
}
