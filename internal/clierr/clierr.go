// Package clierr defines the CLI's typed exit-code contract: 0 success,
// 1 assertion/check failure (Fail), 2 everything else (config, auth, API,
// usage errors).
package clierr

import "fmt"

type failErr struct{ msg string }

func (e failErr) Error() string { return e.msg }

// Fail marks an assertion/check failure → exit code 1.
func Fail(format string, a ...any) error { return failErr{fmt.Sprintf(format, a...)} }

// Config marks a config/auth/API/usage error → exit code 2.
func Config(format string, a ...any) error { return fmt.Errorf(format, a...) }

// Code maps an error to the CLI exit contract: 0 ok, 1 assertion failed, 2 else.
func Code(err error) int {
	if err == nil {
		return 0
	}
	if _, ok := err.(failErr); ok {
		return 1
	}
	return 2
}
