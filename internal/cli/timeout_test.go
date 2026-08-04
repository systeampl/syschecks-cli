package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// `verify` and `probe http` are sold as CI building blocks, but both used a
// client with no timeout: against a server that accepts the connection and
// never answers, the command hangs and takes the job with it instead of
// failing on the exit-code contract.

func hangingServer(t *testing.T) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(func() {
		close(release)
		s.Close()
	})
	return s
}

func runsWithin(t *testing.T, d time.Duration, fn func()) bool {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

func TestVerifyGivesUpOnAHangingServer(t *testing.T) {
	s := hangingServer(t)

	var err error
	if !runsWithin(t, 10*time.Second, func() {
		err = runCLIErr(t, "verify", "--url", s.URL, "--timeout", "300ms")
	}) {
		t.Fatal("verify hung against an unresponsive server")
	}
	if err == nil {
		t.Fatal("verify against an unresponsive server returned no error")
	}
	if got := exitCode(err); got != 1 {
		t.Fatalf("verify timeout exit code = %d, want 1 (the target failed on its own terms)", got)
	}
}

func TestProbeHTTPGivesUpOnAHangingServer(t *testing.T) {
	s := hangingServer(t)

	var err error
	if !runsWithin(t, 10*time.Second, func() {
		err = runCLIErr(t, "probe", "http", s.URL, "--timeout", "300ms")
	}) {
		t.Fatal("probe http hung against an unresponsive server")
	}
	if err == nil {
		t.Fatal("probe http against an unresponsive server returned no error")
	}
}
