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
			{Name: "db_password", Kind: AttrString, Secret: true},
			{Name: "db_ssl_enabled", Kind: AttrBool},
			{Name: "db_query", Kind: AttrString},
			{Name: "db_expected_result", Kind: AttrString},
			// HTTP auth (auth_password / auth_bearer_token are write-only secrets).
			{Name: "auth_username", Kind: AttrString},
			{Name: "auth_password", Kind: AttrString, Secret: true},
			{Name: "auth_bearer_token", Kind: AttrString, Secret: true},
			{Name: "http_body", Kind: AttrString},
			{Name: "http_body_type", Kind: AttrString},
			{Name: "http_follow_redirects", Kind: AttrBool},
			{Name: "content_match_enabled", Kind: AttrBool},
			{Name: "content_match_text", Kind: AttrString},
			{Name: "content_match_type", Kind: AttrString},
			{Name: "content_match_case_sensitive", Kind: AttrBool},
			{Name: "http_form_login_enabled", Kind: AttrBool},
			{Name: "http_form_login_url", Kind: AttrString},
			{Name: "http_form_login_success_text", Kind: AttrString},
			{Name: "http_form_check_after_login_url", Kind: AttrString},
			// FTP/SFTP (ftp_password is a write-only secret in the provider, too —
			// not in this task's required Secret set; see report concerns).
			{Name: "ftp_username", Kind: AttrString},
			{Name: "ftp_password", Kind: AttrString},
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
			// JSON-object/array fields (jsontypes.Normalized StringAttribute —
			// HCL-wise still a string, written as jsonencode(...)).
			{Name: "http_headers", Kind: AttrString},
			{Name: "http_form_login_data", Kind: AttrString},
			{Name: "api_scenario_steps", Kind: AttrString},
			{Name: "api_scenario_secrets", Kind: AttrString},
			{Name: "oidc_config", Kind: AttrString},
			{Name: "dns_records_config", Kind: AttrString},
			{Name: "check_source_critical", Kind: AttrString},
			{Name: "response_assertions", Kind: AttrString},
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
		SecretMapKeys: []string{
			"webhook_url", "url", "token", "api_key", "auth_token",
			"integration_key", "routing_key", "password", "secret",
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
