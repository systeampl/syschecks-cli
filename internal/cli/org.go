package cli

import (
	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// orgCols is the column set shared by `org list` and `org get`.
var orgCols = []string{"id", "name", "slug"}

// newOrgCmd groups organization read commands: list, get <slug>.
func newOrgCmd() *cobra.Command {
	c := &cobra.Command{Use: "org", Short: "Organizations"}
	c.AddCommand(newOrgListCmd(), newOrgGetCmd())
	return c
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgs, err := env.SDK.Organizations.ListOrganizations(cmd.Context())
			if err != nil {
				return clierr.Config("listing orgs: %v", err)
			}
			var rows []output.Row
			for _, o := range *orgs {
				rows = append(rows, output.Row{"id": o.Id, "name": o.Name, "slug": deref(o.Slug)})
			}
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: orgCols, Rows: rows})
		},
	}
}

func newOrgGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <slug>",
		Short: "Show a single organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			o, err := env.SDK.Organizations.GetOrganizationBySlug(cmd.Context(), args[0])
			if err != nil {
				return clierr.Config("getting org %q: %v", args[0], err)
			}
			row := output.Row{"id": o.Id, "name": o.Name, "slug": deref(o.Slug)}
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: orgCols, Rows: []output.Row{row}})
		},
	}
}

// deref returns "" for a nil *string, else its pointee. Used to flatten the
// SDK's optional string fields into plain output.Row values.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
