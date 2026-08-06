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
