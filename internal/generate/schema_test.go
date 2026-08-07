package generate

import "testing"

func TestCheckSpecTFType(t *testing.T) {
	got := specs["check"]
	if got == nil {
		t.Fatal("specs[\"check\"] is nil")
	}
	if got.TFType != "systeam_check" {
		t.Errorf("TFType = %q, want %q", got.TFType, "systeam_check")
	}
}

func TestCheckSpecIncludesWritableAttrs(t *testing.T) {
	spec := specs["check"]
	if spec == nil {
		t.Fatal("specs[\"check\"] is nil")
	}

	want := []string{"name", "type", "project_id", "url", "interval", "is_active"}
	for _, name := range want {
		if !hasAttr(spec.Attrs, name) {
			t.Errorf("specs[\"check\"] missing writable attr %q", name)
		}
	}
}

func TestCheckSpecExcludesComputedOnlyAttrs(t *testing.T) {
	spec := specs["check"]
	if spec == nil {
		t.Fatal("specs[\"check\"] is nil")
	}

	excluded := []string{"id", "created_at", "status"}
	for _, name := range excluded {
		if hasAttr(spec.Attrs, name) {
			t.Errorf("specs[\"check\"] should NOT include computed-only attr %q", name)
		}
	}
}

func TestCheckSpecMarksSecrets(t *testing.T) {
	spec := specs["check"]
	if spec == nil {
		t.Fatal("specs[\"check\"] is nil")
	}

	for _, name := range []string{"auth_password", "db_password", "auth_bearer_token"} {
		a := findAttr(spec.Attrs, name)
		if a == nil {
			t.Fatalf("specs[\"check\"] missing attr %q", name)
		}
		if !a.Secret {
			t.Errorf("specs[\"check\"] attr %q should be Secret", name)
		}
	}
}

func TestCheckSpecSecretMatchesProviderSensitive(t *testing.T) {
	spec := specs["check"]
	if spec == nil {
		t.Fatal("specs[\"check\"] is nil")
	}

	// Previously-missing secrets: the provider marks these Sensitive: true too
	// (ftp_password, http_headers, http_form_login_data, api_scenario_secrets),
	// alongside the original three (auth_password, db_password,
	// auth_bearer_token). An HCL emitter that redacts only on Attr.Secret must
	// not have a gap here, or these render as plaintext in generated .tf.
	for _, name := range []string{"ftp_password", "http_headers", "http_form_login_data", "api_scenario_secrets"} {
		a := findAttr(spec.Attrs, name)
		if a == nil {
			t.Fatalf("specs[\"check\"] missing attr %q", name)
		}
		if !a.Secret {
			t.Errorf("specs[\"check\"] attr %q should be Secret (provider marks it Sensitive: true)", name)
		}
	}

	// The provider marks exactly 7 check attributes Sensitive: true
	// (db_password, auth_password, auth_bearer_token, ftp_password,
	// http_headers, http_form_login_data, api_scenario_secrets). Attr.Secret
	// must mirror that 1:1 — no more, no fewer.
	const wantSecretCount = 7
	got := 0
	for _, a := range spec.Attrs {
		if a.Secret {
			got++
		}
	}
	if got != wantSecretCount {
		t.Errorf("specs[\"check\"] has %d Secret:true attrs, want %d (must match provider's Sensitive:true count)", got, wantSecretCount)
	}
}

// TestCheckSpecExcludesJSONBlobObjectAttrs guards against the plan-breaking
// regression where the SDK hands back these four attrs already decoded as
// map[string]any/[]any (not as a JSON string), even though the provider
// declares them string (JSON) attrs. Rendering them via the AttrString path
// would emit an HCL object/tuple into a string attribute, and `terraform
// plan` hard-errors on that ("Inappropriate value for attribute: string
// required") for any check that has one set — aborting the whole plan. Phase
// 1 omits them rather than mis-render them; see schema.go's comment on the
// check spec and hack/extract-provider-schema.sh for the full rationale.
func TestCheckSpecExcludesJSONBlobObjectAttrs(t *testing.T) {
	spec := specs["check"]
	if spec == nil {
		t.Fatal("specs[\"check\"] is nil")
	}

	excluded := []string{"api_scenario_steps", "oidc_config", "dns_records_config", "response_assertions"}
	for _, name := range excluded {
		if hasAttr(spec.Attrs, name) {
			t.Errorf("specs[\"check\"] should NOT include JSON-blob-as-object attr %q (SDK returns it decoded, not as a JSON string; rendering it would hard-error terraform plan)", name)
		}
	}
}

func TestNotificationChannelSpecConfigMap(t *testing.T) {
	spec := specs["notification-channel"]
	if spec == nil {
		t.Fatal("specs[\"notification-channel\"] is nil")
	}
	if spec.TFType != "systeam_notification_channel" {
		t.Errorf("TFType = %q, want %q", spec.TFType, "systeam_notification_channel")
	}

	config := findAttr(spec.Attrs, "config")
	if config == nil {
		t.Fatal("specs[\"notification-channel\"] missing attr \"config\"")
	}
	if config.Kind != AttrMap {
		t.Errorf("config.Kind = %v, want AttrMap", config.Kind)
	}

	found := false
	for _, k := range spec.SecretMapKeys {
		if k == "webhook_url" {
			found = true
		}
	}
	if !found {
		t.Errorf("specs[\"notification-channel\"].SecretMapKeys should contain %q, got %v", "webhook_url", spec.SecretMapKeys)
	}
}

// TestNotificationChannelSecretMapKeysMatchesBackend guards against the
// security-fragility regression: the backend masks a larger set of
// notification `config` keys to "******" (healthchecks-backend
// app/api/notification_channels.py's SECRET_CONFIG_KEYS) than the CLI used
// to redact. A key the backend masks but SecretMapKeys doesn't list would
// come back from the API already masked, and get emitted as a literal
// placeholder (e.g. `bot_token = "******"`) that a later `terraform apply`
// would write back — overwriting the real secret. SecretMapKeys must be a
// superset of the backend's set.
func TestNotificationChannelSecretMapKeysMatchesBackend(t *testing.T) {
	spec := specs["notification-channel"]
	if spec == nil {
		t.Fatal("specs[\"notification-channel\"] is nil")
	}

	backendSecretKeys := []string{
		// Shared across channel types.
		"api_key", "routing_key", "auth_token",
		// Backend SECRET_CONFIG_KEYS additions.
		"smtp_password", "bot_token", "access_token", "api_token",
		"signing_secret", "webhook_secret", "account_sid", "auth_id",
		"connection_id", "client_secret",
	}
	for _, k := range backendSecretKeys {
		found := false
		for _, got := range spec.SecretMapKeys {
			if got == k {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("specs[\"notification-channel\"].SecretMapKeys missing backend-masked key %q, got %v", k, spec.SecretMapKeys)
		}
	}
}

func TestTeamSpecTFType(t *testing.T) {
	spec := specs["team"]
	if spec == nil {
		t.Fatal("specs[\"team\"] is nil")
	}
	if spec.TFType != "systeam_team" {
		t.Errorf("TFType = %q, want %q", spec.TFType, "systeam_team")
	}

	want := []string{"organization_id", "name", "slug", "description"}
	for _, name := range want {
		if !hasAttr(spec.Attrs, name) {
			t.Errorf("specs[\"team\"] missing writable attr %q", name)
		}
	}
	if hasAttr(spec.Attrs, "id") || hasAttr(spec.Attrs, "member_count") || hasAttr(spec.Attrs, "is_active") {
		t.Errorf("specs[\"team\"] should not include computed-only attrs (id, member_count, is_active)")
	}
}

func hasAttr(attrs []Attr, name string) bool {
	return findAttr(attrs, name) != nil
}

func findAttr(attrs []Attr, name string) *Attr {
	for i := range attrs {
		if attrs[i].Name == name {
			return &attrs[i]
		}
	}
	return nil
}
