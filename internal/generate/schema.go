// Package generate builds provider-aligned attribute specs for `syschecks
// generate terraform`: for each resource, the set of HCL attributes the
// published terraform-provider-systeam actually accepts, so generated
// configuration matches the provider's schema instead of drifting from it.
//
// The specs below are derived from the provider's resource.Schema() —
// specifically its `Attributes: map[string]schema.Attribute{...}` literal —
// at /home/destine/GIT/wlasne/terraform-provider-systeam/internal/resources/<res>/resource.go.
// Only attributes that are Required or Optional are writable from HCL and
// included here; attributes that are Computed-only (id, created_at,
// updated_at, status, uuid, member_count, ...) are read-only and excluded.
// Attr.Secret mirrors the provider's Sensitive: true 1:1 — every attribute the
// provider marks Sensitive is Secret here, so an HCL emitter that redacts on
// Attr.Secret never writes a provider-sensitive value in plaintext.
// See hack/extract-provider-schema.sh for the extraction recipe used to
// regenerate this list when the provider schema changes.
package generate

// AttrKind is the HCL/SDK value shape of a writable attribute.
type AttrKind int

const (
	AttrString AttrKind = iota
	AttrInt
	AttrBool
	AttrList
	AttrMap
)

// Attr is one writable HCL attribute. Name is both the snake_case HCL
// attribute name and the SDK JSON key (the provider and SDK share naming).
type Attr struct {
	Name   string
	Kind   AttrKind
	Secret bool
}

// ResourceSpec is the attribute spec for one syschecks-cli resource kind.
type ResourceSpec struct {
	// TFType is the Terraform resource type name (e.g. "systeam_check").
	TFType string
	// Attrs is the writable attribute list, in provider schema order.
	Attrs []Attr
	// SecretMapKeys are the keys of a MapAttribute (see Attr.Kind == AttrMap)
	// whose values carry secrets, e.g. notification-channel's "config" map.
	SecretMapKeys []string
}

// specs holds the provider-derived attribute spec for each resource kind
// `generate terraform` supports. Keys are syschecks-cli resource names, not
// Terraform type names (see ResourceSpec.TFType for that).
var specs = map[string]*ResourceSpec{
	"check": {
		TFType: "systeam_check",
		Attrs: []Attr{
			{Name: "name", Kind: AttrString},
			{Name: "type", Kind: AttrString},
			{Name: "project_id", Kind: AttrInt},
			{Name: "description", Kind: AttrString},
			{Name: "is_active", Kind: AttrBool},
			{Name: "interval", Kind: AttrInt},
			{Name: "grace_period", Kind: AttrInt},
			{Name: "timeout", Kind: AttrInt},
			{Name: "url", Kind: AttrString},
			{Name: "host", Kind: AttrString},
			{Name: "port", Kind: AttrInt},
			{Name: "ping_count", Kind: AttrInt},
			{Name: "ssl_verify", Kind: AttrBool},
			{Name: "alert_after_failures", Kind: AttrInt},
			{Name: "geo_monitoring_enabled", Kind: AttrBool},
			{Name: "assigned_agent_id", Kind: AttrInt},
			{Name: "escalation_policy_id", Kind: AttrInt},
			{Name: "traceroute_on_timeout", Kind: AttrBool},
			{Name: "dns_server", Kind: AttrString},
			{Name: "dns_record_type", Kind: AttrString},
			{Name: "dns_expected_value", Kind: AttrString},
			{Name: "http_method", Kind: AttrString},
			{Name: "auth_method", Kind: AttrString},
			{Name: "schedule_type", Kind: AttrString},
			{Name: "cron_expression", Kind: AttrString},
			{Name: "cron_timezone", Kind: AttrString},
			{Name: "mail_domain", Kind: AttrString},
			{Name: "runbook_url", Kind: AttrString},
			// Database checks (type = "database").
			{Name: "db_type", Kind: AttrString},
			{Name: "db_host", Kind: AttrString},
			{Name: "db_port", Kind: AttrInt},
			{Name: "db_name", Kind: AttrString},
			{Name: "db_username", Kind: AttrString},
			{Name: "db_password", Kind: AttrString, Secret: true}, // Sensitive: true
			{Name: "db_ssl_enabled", Kind: AttrBool},
			{Name: "db_query", Kind: AttrString},
			{Name: "db_expected_result", Kind: AttrString},
			// HTTP auth (auth_password / auth_bearer_token are write-only secrets).
			{Name: "auth_username", Kind: AttrString},
			{Name: "auth_password", Kind: AttrString, Secret: true},     // Sensitive: true
			{Name: "auth_bearer_token", Kind: AttrString, Secret: true}, // Sensitive: true
			{Name: "http_body", Kind: AttrString},
			{Name: "http_body_type", Kind: AttrString},
			{Name: "http_follow_redirects", Kind: AttrBool},
			{Name: "content_match_enabled", Kind: AttrBool},
			{Name: "content_match_text", Kind: AttrString},
			{Name: "content_match_type", Kind: AttrString},
			{Name: "content_match_case_sensitive", Kind: AttrBool},
			// Content change / geo consistency monitoring.
			{Name: "content_change_enabled", Kind: AttrBool},
			{Name: "content_change_severity", Kind: AttrString},
			{Name: "geo_content_consistency_enabled", Kind: AttrBool},
			{Name: "http_form_login_enabled", Kind: AttrBool},
			{Name: "http_form_login_url", Kind: AttrString},
			{Name: "http_form_login_success_text", Kind: AttrString},
			{Name: "http_form_check_after_login_url", Kind: AttrString},
			// FTP/SFTP.
			{Name: "ftp_username", Kind: AttrString},
			{Name: "ftp_password", Kind: AttrString, Secret: true}, // Sensitive: true
			{Name: "ftp_protocol", Kind: AttrString},
			{Name: "ftp_path", Kind: AttrString},
			{Name: "ftp_passive", Kind: AttrBool},
			// DNS advanced.
			{Name: "dns_soa_alert_on_change", Kind: AttrBool},
			{Name: "dns_hijack_alert_enabled", Kind: AttrBool},
			{Name: "dns_hijack_alert_channel_ids", Kind: AttrString},
			{Name: "dns_txt_monitoring_enabled", Kind: AttrBool},
			{Name: "dns_dkim_selector", Kind: AttrString},
			{Name: "dns_multi_record_enabled", Kind: AttrBool},
			// Mail server.
			{Name: "mail_smtp_enabled", Kind: AttrBool},
			{Name: "mail_smtp_port", Kind: AttrInt},
			{Name: "mail_smtp_starttls", Kind: AttrBool},
			{Name: "mail_smtp_open_relay", Kind: AttrBool},
			{Name: "mail_imap_enabled", Kind: AttrBool},
			{Name: "mail_imap_port", Kind: AttrInt},
			{Name: "mail_imap_ssl", Kind: AttrBool},
			{Name: "mail_pop3_enabled", Kind: AttrBool},
			{Name: "mail_pop3_port", Kind: AttrInt},
			{Name: "mail_pop3_ssl", Kind: AttrBool},
			{Name: "mail_check_spf", Kind: AttrBool},
			{Name: "mail_check_dkim", Kind: AttrBool},
			{Name: "mail_dkim_selectors", Kind: AttrString},
			{Name: "mail_check_dmarc", Kind: AttrBool},
			{Name: "mail_check_ptr", Kind: AttrBool},
			{Name: "mail_check_blacklist", Kind: AttrBool},
			{Name: "mail_blacklist_servers", Kind: AttrString},
			// List fields.
			{Name: "expected_status_codes", Kind: AttrList},
			{Name: "dns_expected_ips", Kind: AttrList},
			{Name: "assigned_agent_ids", Kind: AttrList},
			{Name: "content_ignore_patterns", Kind: AttrList},
			// JSON-object/array fields (jsontypes.Normalized StringAttribute —
			// HCL-wise still a string, written as jsonencode(...)).
			//
			// The provider declares exactly 8 attrs with CustomType:
			// jsontypes.NormalizedType{}: http_headers, http_form_login_data,
			// api_scenario_secrets (Sensitive: true, kept below) and
			// api_scenario_steps, oidc_config, dns_records_config,
			// check_source_critical, response_assertions (NOT Sensitive,
			// excluded — see the comment block after these three).
			//
			// The rule (see also hack/extract-provider-schema.sh, which
			// enforces it mechanically): every jsontypes.NormalizedType attr
			// is dangerous to render via the plain AttrString path, because
			// the SDK hands it back already JSON-decoded
			// (map[string]any/[]any/{} for an unset one), not as a
			// JSON-encoded string — so hclValue would emit an HCL
			// object/tuple into a string attribute, which `terraform plan`
			// hard-errors on ("Inappropriate value for attribute: string
			// required"). A Secret: true jsontypes attr is exempted from
			// that exclusion because renderResource never runs its value
			// through hclValue at all: Attr.Secret redirects to a
			// `var.<label>_<attr>` string reference (see render.go), which
			// is a valid value for a string attribute regardless of what
			// shape the underlying secret has. A non-secret jsontypes attr
			// has no such escape hatch, so it must be excluded from Attrs
			// entirely until render.go grows an AttrKind that
			// jsonencode(...)s a decoded value back into a JSON string.
			{Name: "http_headers", Kind: AttrString, Secret: true},         // Sensitive: true — commonly carries Authorization/API keys
			{Name: "http_form_login_data", Kind: AttrString, Secret: true}, // Sensitive: true — typically contains {username, password}
			{Name: "api_scenario_secrets", Kind: AttrString, Secret: true}, // Sensitive: true — write-only
			// Excluded (non-secret jsontypes.NormalizedType — see rule
			// above): api_scenario_steps, oidc_config, dns_records_config,
			// check_source_critical, response_assertions. Phase 1 already
			// defers complex/nested attrs (see
			// docs/superpowers/plans/2026-08-06-generate-tf-phase1.md), so
			// omitting these is drift-at-worst — far better than the hard
			// plan error rendering them would cause. check_source_critical
			// in particular is set (non-empty) on most checks, so this
			// exclusion is load-bearing, not a corner case.
		},
	},
	"notification-channel": {
		TFType: "systeam_notification_channel",
		Attrs: []Attr{
			{Name: "name", Kind: AttrString},
			{Name: "channel_type", Kind: AttrString},
			{Name: "config", Kind: AttrMap},
			{Name: "is_active", Kind: AttrBool},
			{Name: "organization_id", Kind: AttrInt},
		},
		// config is a MapAttribute[string]string; these are the keys (across the
		// various channel_type configs) that carry secrets, so `generate` can mark
		// them sensitive rather than emit them as plain HCL string literals.
		//
		// This is the union of the CLI's own guesses and the backend's
		// authoritative SECRET_CONFIG_KEYS (healthchecks-backend
		// app/api/notification_channels.py) — the set of config keys the
		// backend masks to "******" before ever returning them. A key
		// missing from this list but present in the backend's set would
		// come back from the API already masked, and renderResource would
		// happily emit it as a literal `key = "******"` in
		// notification_channels.tf: a placeholder that a later `terraform
		// apply` would write straight back, overwriting the real secret
		// with the string "******". Keep this list a superset of the
		// backend's, not just an independent guess — see also
		// renderMapAttr's mask-sentinel check in render.go, which redacts
		// any "******" value even if its key isn't listed here, as a
		// defense-in-depth backstop for keys the backend adds later.
		SecretMapKeys: []string{
			"access_token", "account_sid", "api_key", "api_token",
			"auth_id", "auth_token", "bot_token", "client_secret",
			"connection_id", "integration_key", "password", "routing_key",
			"secret", "signing_secret", "smtp_password", "token", "url",
			"webhook_secret", "webhook_url",
		},
	},
	"team": {
		TFType: "systeam_team",
		Attrs: []Attr{
			{Name: "organization_id", Kind: AttrInt},
			{Name: "name", Kind: AttrString},
			{Name: "slug", Kind: AttrString},
			{Name: "description", Kind: AttrString},
		},
	},
}
