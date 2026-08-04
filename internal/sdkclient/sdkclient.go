// Package sdkclient builds a syschecks-go SDK client from resolved CLI
// configuration.
package sdkclient

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/systeampl/syschecks-cli/internal/config"
	syschecks "github.com/systeampl/syschecks-go"
)

// New builds a syschecks SDK client from the resolved connection config.
func New(r config.Resolved) (*syschecks.Client, error) {
	return syschecks.New(r.APIURL, r.Token)
}

// NewVerbose is New with request logging to w: one "> METHOD url" line per
// outgoing request. The SDK sets Authorization on every request, so the log
// deliberately reports the method and URL only — never headers — which is what
// --verbose promises ("token redacted").
func NewVerbose(r config.Resolved, w io.Writer) (*syschecks.Client, error) {
	return syschecks.New(r.APIURL, r.Token, syschecks.WithRequestEditor(
		func(_ context.Context, req *http.Request) error {
			fmt.Fprintf(w, "> %s %s\n", req.Method, req.URL.String())
			return nil
		},
	))
}
