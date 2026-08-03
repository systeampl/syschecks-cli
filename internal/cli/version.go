package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/buildinfo"
)

type ctxKey int

const globalsKey ctxKey = 0

func withGlobals(ctx context.Context, gf *GlobalFlags) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, globalsKey, gf)
}

func globalsFrom(cmd *cobra.Command) *GlobalFlags {
	if v, ok := cmd.Context().Value(globalsKey).(*GlobalFlags); ok {
		return v
	}
	return &GlobalFlags{Output: "table"}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("syschecks %s\n", buildinfo.Version)
			return nil
		},
	}
}
