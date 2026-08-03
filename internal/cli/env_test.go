package cli

import (
	"testing"

	"github.com/systeampl/syschecks-cli/internal/clierr"
)

// TestCmdEnvRequiresAPIURL drives cmdEnv directly (whoami/Task 6 doesn't
// exist yet): with no context, no flags, and no env vars set, config
// resolution must fail as a config error (exit code 2).
func TestCmdEnvRequiresAPIURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty config dir
	t.Setenv("SYSCHECKS_API_URL", "")
	t.Setenv("SYSCHECKS_TOKEN", "")

	cmd := NewRootCmd()

	_, err := cmdEnv(cmd)
	if err == nil {
		t.Fatal("expected config error when no API URL configured")
	}
	if code := clierr.Code(err); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
