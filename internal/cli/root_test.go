package cli

import (
	"bytes"
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
