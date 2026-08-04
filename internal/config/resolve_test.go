package config

import (
	"strings"
	"testing"
)

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

// A mistyped --context resolved to the zero-value context, so the command ran
// on whatever the environment supplied (or failed with "no API URL" / "token is
// required") instead of saying the context does not exist.
func TestUnknownContextIsAnError(t *testing.T) {
	f := &File{
		CurrentContext: "prod",
		Contexts:       map[string]Context{"prod": {APIURL: "https://api.example.com", Organization: "acme"}},
	}
	env := func(string) string { return "" }

	_, err := Resolve(f, Flags{Context: "nie-ma-takiego"}, env)
	if err == nil {
		t.Fatal("Resolve accepted an unknown context")
	}
	if !strings.Contains(err.Error(), "nie-ma-takiego") {
		t.Fatalf("error %q does not name the unknown context", err)
	}
}
