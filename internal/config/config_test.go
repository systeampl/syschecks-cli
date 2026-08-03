package config

import (
	"os"
	"testing"
)

// TestWriteTokenFixesDirPerms is a regression test: WriteToken must leave the
// config dir at 0700 even when it already existed with a looser mode (e.g.
// 0755), since os.MkdirAll alone is a no-op on a pre-existing directory and
// would silently leave it over-permissive.
func TestWriteTokenFixesDirPerms(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod setup: %v", err)
	}

	if err := WriteToken(dir, "prod", "secret123"); err != nil {
		t.Fatalf("WriteToken: %v", err)
	}

	dinfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dinfo.Mode().Perm(); got != 0o700 {
		t.Errorf("dir mode = %v, want 0700", got)
	}

	finfo, err := os.Stat(TokenPath(dir, "prod"))
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if got := finfo.Mode().Perm(); got != 0o600 {
		t.Errorf("token file mode = %v, want 0600", got)
	}
}
