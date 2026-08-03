package clierr

import "testing"

func TestCode(t *testing.T) {
	if Code(nil) != 0 {
		t.Error("nil → 0")
	}
	if Code(Fail("down")) != 1 {
		t.Error("Fail → 1")
	}
	if Code(Config("bad")) != 2 {
		t.Error("Config → 2")
	}
	if Code(errPlain{}) != 2 {
		t.Error("plain error → 2")
	}
}

type errPlain struct{}

func (errPlain) Error() string { return "x" }
