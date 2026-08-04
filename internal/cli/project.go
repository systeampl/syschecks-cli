package cli

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// projectListCols/projectGetCols are the column sets for `project list` and
// `project get` respectively; get additionally shows the description.
var (
	projectListCols = []string{"id", "name"}
	projectGetCols  = []string{"id", "name", "description"}
)

// newProjectCmd groups project read commands: list, get <id>. Both need an
// organization, resolved via resolveOrgID from the --org global flag.
func newProjectCmd() *cobra.Command {
	c := &cobra.Command{Use: "project", Short: "Projects"}
	c.AddCommand(newProjectListCmd(), newProjectGetCmd())
	return c
}

func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects in an organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			projects, err := env.SDK.Organizations.ListOrganizationProjects(cmd.Context(), orgID, nil)
			if err != nil {
				return clierr.Config("listing projects: %v", err)
			}
			var rows []output.Row
			for _, p := range *projects {
				rows = append(rows, output.Row{"id": p.Id, "name": p.Name})
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: projectListCols, Rows: rows})
		},
	}
}

func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show a single project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			projID, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid project id %q: %v", args[0], err)
			}
			p, err := env.SDK.Organizations.GetOrganizationProject(cmd.Context(), orgID, projID)
			if err != nil {
				return clierr.Config("getting project %d: %v", projID, err)
			}
			row := output.Row{"id": p.Id, "name": p.Name, "description": deref(p.Description)}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: projectGetCols, Rows: []output.Row{row}})
		},
	}
}

// resolveOrgID resolves the organization id project commands operate on from
// the --org global flag (env.Org): a bare integer is used as-is, anything
// else is treated as a slug and resolved via GetOrganizationBySlug. An empty
// --org is a config error (exit code 2), not a panic or silent zero-value.
func resolveOrgID(cmd *cobra.Command, env *cmdCtx) (int, error) {
	if env.Org == "" {
		return 0, clierr.Config("no organization specified: pass --org <id|slug>")
	}
	if id, err := strconv.Atoi(env.Org); err == nil {
		return id, nil
	}
	o, err := env.SDK.Organizations.GetOrganizationBySlug(cmd.Context(), env.Org)
	if err != nil {
		return 0, clierr.Config("resolving org %q: %v", env.Org, err)
	}
	return o.Id, nil
}

// optionalOrgID is resolveOrgID for the cross-organization listings (checks,
// incidents, notification channels): those stay valid without an organization
// — they then list everything the token reaches — so an empty --org yields a
// nil filter rather than the config error resolveOrgID raises.
func optionalOrgID(cmd *cobra.Command, env *cmdCtx) (*int, error) {
	if env.Org == "" {
		return nil, nil
	}
	id, err := resolveOrgID(cmd, env)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
