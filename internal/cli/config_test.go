package cli

import (
	"strings"
	"testing"
)

func TestSetContextThenGetContexts(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := runCLIErr(t, "config", "set-context", "prod", "--api-url", "https://x", "--org", "acme"); err != nil {
		t.Fatal(err)
	}
	out := runCLIOut(t, "config", "get-contexts")
	if !strings.Contains(out, "prod") {
		t.Fatalf("get-contexts = %q, want prod", out)
	}
}
