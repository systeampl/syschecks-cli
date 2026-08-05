package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-go/models"
)

// orgCols is the column set shared by `org list` and `org get`.
var orgCols = []string{"id", "name", "slug"}

// orgFields is the flag/-f schema for `org create`/`org update`, generated
// from the flat scalar fields of models.OrganizationCreate; Name is the only
// field the API requires.
var orgFields = []Field{
	{Name: "billing-email", JSONKey: "billing_email", Kind: "string", Required: false, Help: "billing email"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "plan-id", JSONKey: "plan_id", Kind: "int", Required: false, Help: "plan id"},
	{Name: "slug", JSONKey: "slug", Kind: "string", Required: false, Help: "slug"},
}

func init() {
	register(&Resource{
		Name: "org", Cols: orgCols, Org: OrgNone, Fields: orgFields,
		listFn: func(env *cmdCtx, _ *int) ([]map[string]any, error) {
			orgs, err := env.SDK.Organizations.ListOrganizations(context.Background())
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*orgs))
			for _, o := range *orgs {
				items = append(items, map[string]any{"id": o.Id, "name": o.Name, "slug": deref(o.Slug)})
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, _ *int, id int) (map[string]any, error) {
			o, err := env.SDK.Organizations.GetOrganization(context.Background(), id)
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": o.Id, "name": o.Name, "slug": deref(o.Slug)}, nil
		},
		createFn: func(env *cmdCtx, _ *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.OrganizationCreate](body)
			if err != nil {
				return nil, err
			}
			o, err := env.SDK.Organizations.CreateOrganization(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(o)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.OrganizationUpdate](body)
			if err != nil {
				return nil, err
			}
			o, err := env.SDK.Organizations.UpdateOrganization(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(o)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.Organizations.DeleteOrganization(context.Background(), id, &models.DeleteOrganizationParams{})
			return err
		},
	})
}

// newOrgCmd builds the `org` command from the registry (list/create/
// update/delete) and swaps in the slug-aware `get` that predates the generic
// factory: it resolves its argument via GetOrganizationBySlug — the
// registry's generic `get <id>` only ever accepts an integer, which would
// break `org get <slug>`.
func newOrgCmd() *cobra.Command {
	c := newResourceCmd(registry["org"])
	swapGetCmd(c, newOrgGetCmd())
	return c
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

// swapGetCmd replaces c's registry-generated "get" subcommand (which only
// ever accepts a numeric id, see newGetCmd in crud.go) with replacement, a
// bespoke get command with resource-specific resolution (org: slug, check:
// name). Used right after newResourceCmd for resources whose `get` predates
// the generic factory and needs to keep working exactly as before.
func swapGetCmd(c *cobra.Command, replacement *cobra.Command) {
	for _, sc := range c.Commands() {
		if sc.Name() == "get" {
			c.RemoveCommand(sc)
			break
		}
	}
	c.AddCommand(replacement)
}
