package config

import "testing"

func TestResolvePrecedence(t *testing.T) {
	f := &File{CurrentContext: "prod", Contexts: map[string]Context{
		"prod": {APIURL: "https://api.example.com", Organization: "acme"},
	}}
	env := map[string]string{"SYSCHECKS_TOKEN": "envtok"}
	get := func(k string) string { return env[k] }

	// env token wins; flag api-url overrides context; org falls back to context
	r, err := Resolve(f, Flags{APIURL: "http://localhost:8001"}, get)
	if err != nil {
		t.Fatal(err)
	}
	if r.Token != "envtok" {
		t.Errorf("token = %q, want envtok", r.Token)
	}
	if r.APIURL != "http://localhost:8001" {
		t.Errorf("api-url = %q, want flag override", r.APIURL)
	}
	if r.Org != "acme" {
		t.Errorf("org = %q, want context fallback", r.Org)
	}
}
