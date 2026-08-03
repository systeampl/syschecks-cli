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
