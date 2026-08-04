package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootVersion(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := out.String(); got == "" || got[:9] != "syschecks" {
		t.Fatalf("version output = %q, want it to start with 'syschecks'", got)
	}
}

// runCLISeparate is like the runCLI family in testhelpers_test.go, but keeps
// stdout and stderr in separate buffers and drives the command through
// execute() (the package-private helper Execute() calls) rather than
// root.Execute() directly — that is the only path that prints a failing
// command's error, so a test asserting on stderr content must go through it
// rather than through runCLI/runCLIOut/runCLIErr (which only ever inspect
// the returned error value, never the process's actual stderr stream).
func runCLISeparate(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	code = execute(root)
	return outBuf.String(), errBuf.String(), code
}

// TestFailingCommandPrintsErrorToStderr is a regression test for the bug
// where SilenceErrors:true meant Execute() returned a non-zero exit code
// but never actually wrote the error anywhere: a CI job would see a bare
// exit status with no explanation on stderr. Clearing every config source
// (no context, no --api-url, no SYSCHECKS_API_URL) makes config resolution
// itself fail deterministically, independent of any fake API server.
func TestFailingCommandPrintsErrorToStderr(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", "")
	t.Setenv("SYSCHECKS_TOKEN", "")

	stdout, stderr, code := runCLISeparate(t, "org", "list")

	if code == 0 {
		t.Fatalf("expected a non-zero exit code, got 0 (stdout=%q stderr=%q)", stdout, stderr)
	}
	if stderr == "" {
		t.Fatal("expected a non-empty stderr message for a failing command, got empty")
	}
	if !strings.Contains(stderr, "Error") {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, "Error")
	}
	if !strings.Contains(stderr, "API URL") {
		t.Fatalf("stderr = %q, want the underlying resolution error text (API URL)", stderr)
	}
}

// TestFailingCommandWithJSONOutputPrintsJSONError checks the --output json
// branch: the error must be a `{"error":"..."}` line on stderr rather than
// the plain "Error: ..." line, so a CI job parsing stderr as JSON doesn't
// choke on human-readable text.
func TestFailingCommandWithJSONOutputPrintsJSONError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SYSCHECKS_API_URL", "")
	t.Setenv("SYSCHECKS_TOKEN", "")

	_, stderr, code := runCLISeparate(t, "--output", "json", "org", "list")

	if code == 0 {
		t.Fatalf("expected a non-zero exit code, got 0 (stderr=%q)", stderr)
	}
	stderr = strings.TrimSpace(stderr)
	if !strings.HasPrefix(stderr, `{"error":"`) {
		t.Fatalf("stderr = %q, want a {\"error\":...} JSON line", stderr)
	}
	if !strings.HasSuffix(stderr, `"}`) {
		t.Fatalf("stderr = %q, want it to end with a closed JSON object", stderr)
	}
}
