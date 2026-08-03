package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// verify hits an arbitrary --url, not the SysChecks API, so a plain
// httptest.Server started inline is simplest here (no fakeAPI needed).

func TestVerifyStatusMismatchExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200")
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d, want 1 (err=%v)", exitCode(err), err)
	}
}

func TestVerifyJSONTrueExit0(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status == "ok"`)
	if exitCode(err) != 0 {
		t.Fatalf("exit = %d, want 0 (err=%v)", exitCode(err), err)
	}
}

func TestVerifyJSONFalseExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"degraded"}`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status == "ok"`)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d, want 1 (err=%v)", exitCode(err), err)
	}
}

func TestVerifyBadJQExprExit2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status ===`)
	if exitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2 for a gojq parse error (err=%v)", exitCode(err), err)
	}
}

func TestVerifyNonBooleanResultExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status`)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d, want 1 for a non-boolean jq result (err=%v)", exitCode(err), err)
	}
}

func TestVerifyMissingURLExit2(t *testing.T) {
	err := runCLIErr(t, "verify", "--expect-status", "200")
	if exitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2 for missing --url (err=%v)", exitCode(err), err)
	}
}

// TestVerifyNonJSONBodyExit1 asserts that a response body which fails to
// decode as JSON is the TARGET's fault (same category as a wrong status
// code), not the user's -- so it's an assertion failure (exit 1), not a
// config error (exit 2).
func TestVerifyNonJSONBodyExit1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`plain text`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status`)
	if exitCode(err) != 1 {
		t.Fatalf("exit = %d, want 1 for a non-JSON response body (err=%v)", exitCode(err), err)
	}
}

// TestVerifyJQEvalErrorExit2 asserts that a gojq expression which is
// syntactically valid but errors during evaluation (adding an int to a
// string) is the USER's fault -- a bad --expect-json expression -- so it's a
// config error (exit 2), not an assertion failure.
func TestVerifyJQEvalErrorExit2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	err := runCLIErr(t, "verify", "--url", srv.URL, "--expect-status", "200",
		"--expect-json", `.status + 1`)
	if exitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2 for a gojq eval error (err=%v)", exitCode(err), err)
	}
}
