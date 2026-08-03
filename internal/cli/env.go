package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/config"
	"github.com/systeampl/syschecks-cli/internal/sdkclient"
	syschecks "github.com/systeampl/syschecks-go"
)

// cmdCtx bundles everything a leaf command needs: the SDK client, the
// output writer/format, and the resolved org — all derived from config
// resolution via cmdEnv.
type cmdCtx struct {
	SDK    *syschecks.Client
	Out    io.Writer
	Format string
	Quiet  bool
	Org    string
}

// cmdEnv resolves config (flags -> context -> env -> token file), builds the
// SDK client, and returns a cmdCtx ready for a command's RunE. On resolution
// failure it returns a clierr.Config error (exit code 2).
func cmdEnv(cmd *cobra.Command) (*cmdCtx, error) {
	gf := globalsFrom(cmd)
	f, err := config.Load(config.Dir())
	if err != nil {
		return nil, clierr.Config("loading config: %v", err)
	}
	r, err := config.Resolve(f, config.Flags{
		Context: gf.Context, Org: gf.Org, APIURL: gf.APIURL, Token: gf.Token,
	}, os.Getenv)
	if err != nil {
		return nil, clierr.Config("%v", err)
	}
	sdk, err := sdkclient.New(r)
	if err != nil {
		return nil, clierr.Config("building client: %v", err)
	}
	return &cmdCtx{SDK: sdk, Out: cmd.OutOrStdout(), Format: gf.Output, Quiet: gf.Quiet, Org: r.Org}, nil
}
