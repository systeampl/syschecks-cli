// Package sdkclient builds a syschecks-go SDK client from resolved CLI
// configuration.
package sdkclient

import (
	"github.com/systeampl/syschecks-cli/internal/config"
	syschecks "github.com/systeampl/syschecks-go"
)

// New builds a syschecks SDK client from the resolved connection config.
func New(r config.Resolved) (*syschecks.Client, error) {
	return syschecks.New(r.APIURL, r.Token)
}
