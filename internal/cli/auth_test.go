package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/systeampl/syschecks-cli/internal/config"
)

func TestLoginReadsTokenFromStdinAndNeverEchoesIt(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/", 200, map[string]any{
		"email":      "a@b.com",
		"full_name":  "A",
		"id":         1,
		"created_at": "2026-01-01T00:00:00Z",
		"is_active":  true,
	})

	out, err := runCLI(t, "pat_secret123\n", "auth", "login", "--with-token")
	if err != nil {
		t.Fatalf("login: unexpected error: %v", err)
	}
	if strings.Contains(out, "pat_secret123") {
		t.Fatalf("token leaked into output: %q", out)
	}
	if !strings.Contains(out, "Logged in as") {
		t.Fatalf("want login confirmation, got %q", out)
	}
	if !strings.Contains(out, "a@b.com") {
		t.Fatalf("want the validated email in output, got %q", out)
	}
}

func TestLoginWithTokenArgNeverEchoesIt(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/", 200, map[string]any{
		"email":      "arg@b.com",
		"id":         2,
		"created_at": "2026-01-01T00:00:00Z",
		"is_active":  true,
	})

	out := runCLIOut(t, "auth", "login", "--with-token", "pat_from_arg")
	if strings.Contains(out, "pat_from_arg") {
		t.Fatalf("token leaked into output: %q", out)
	}
	if !strings.Contains(out, "Logged in as arg@b.com") {
		t.Fatalf("want login confirmation with email, got %q", out)
	}
}

func TestLoginRejectsInvalidToken(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/", 401, map[string]any{"detail": "invalid token"})

	err := runCLIErr(t, "auth", "login", "--with-token", "bad-token")
	if err == nil {
		t.Fatal("expected an error for a rejected token")
	}
	if exitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2 (config/auth error)", exitCode(err))
	}
}

func TestLoginWithoutTokenFlagIsRejected(t *testing.T) {
	newFakeAPI(t)
	err := runCLIErr(t, "auth", "login")
	if err == nil {
		t.Fatal("expected an error when --with-token is not passed")
	}
	if exitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode(err))
	}
}

func TestWhoamiPrintsEmail(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/", 200, map[string]any{
		"email":      "whoami@b.com",
		"full_name":  "Who Ami",
		"id":         3,
		"created_at": "2026-01-01T00:00:00Z",
		"is_active":  true,
	})

	out := runCLIOut(t, "auth", "whoami")
	if !strings.Contains(out, "whoami@b.com") {
		t.Fatalf("want the profile email in whoami output, got %q", out)
	}
	if !strings.Contains(out, "Who Ami") {
		t.Fatalf("want the full name in whoami output, got %q", out)
	}
}

func TestLogoutRemovesStoredToken(t *testing.T) {
	api := newFakeAPI(t)
	api.On("GET", "/api/profile/", 200, map[string]any{
		"email":      "logout@b.com",
		"id":         4,
		"created_at": "2026-01-01T00:00:00Z",
		"is_active":  true,
	})

	// Log in first so there is a token file to remove.
	runCLIOut(t, "auth", "login", "--with-token", "pat_to_remove")
	tokenPath := config.TokenPath(config.Dir(), "default")
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("expected token file to exist after login: %v", err)
	}

	out := runCLIOut(t, "auth", "logout")
	if out == "" {
		t.Fatal("expected some confirmation output from logout")
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Fatalf("expected token file removed after logout, stat err = %v", err)
	}
}

func TestLogoutIsIdempotentWithNoStoredToken(t *testing.T) {
	newFakeAPI(t)
	if err := runCLIErr(t, "auth", "logout"); err != nil {
		t.Fatalf("logout with no stored token should succeed, got: %v", err)
	}
}
