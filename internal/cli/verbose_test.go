package cli

import (
	"strings"
	"testing"
)

// --verbose was declared on the root command and documented as available
// everywhere, but nothing ever read it: the flag was accepted and ignored.
// Its help text promises the token stays redacted, so that is part of the
// contract too.

func TestVerboseLogsTheRequest(t *testing.T) {
	newFakeAPI(t).On("GET", "/api/organizations/", 200, []map[string]any{
		{"id": 1, "name": "Acme", "slug": "acme"},
	})

	out := runCLIOut(t, "org", "list", "--verbose")

	if !strings.Contains(out, "GET") || !strings.Contains(out, "/api/organizations/") {
		t.Fatalf("--verbose did not log the request; output = %q", out)
	}
}

func TestVerboseNeverPrintsTheToken(t *testing.T) {
	newFakeAPI(t).On("GET", "/api/organizations/", 200, []map[string]any{})
	t.Setenv("SYSCHECKS_TOKEN", "pat_supersecretvalue")

	out := runCLIOut(t, "org", "list", "--verbose")

	if strings.Contains(out, "pat_supersecretvalue") {
		t.Fatalf("--verbose leaked the token into output: %q", out)
	}
}

func TestWithoutVerboseNothingIsLogged(t *testing.T) {
	newFakeAPI(t).On("GET", "/api/organizations/", 200, []map[string]any{
		{"id": 1, "name": "Acme", "slug": "acme"},
	})

	out := runCLIOut(t, "org", "list")

	if strings.Contains(out, "/api/organizations/") {
		t.Fatalf("request was logged without --verbose: %q", out)
	}
}
