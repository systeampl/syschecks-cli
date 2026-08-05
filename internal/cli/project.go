package cli

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-go/models"
)

// projectCols is the column set for both `project list` and `project get`:
// list and get render through the same registry Resource, so this has to be
// the union of what each showed before (get additionally has description).
var projectCols = []string{"id", "name", "description"}

// projectFields is the flag/-f schema for `project create`/`project update`,
// generated from the flat scalar fields of models.ProjectCreate: just name.
var projectFields = []Field{
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
}

func init() {
	register(&Resource{
		Name: "project", Cols: projectCols, Org: OrgArg, Fields: projectFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			projects, err := env.SDK.Organizations.ListOrganizationProjects(context.Background(), *orgID, nil)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*projects))
			for _, p := range *projects {
				m, err := toMap(p)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			p, err := env.SDK.Organizations.GetOrganizationProject(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ProjectCreate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Organizations.CreateOrganizationProject(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ProjectUpdate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Organizations.UpdateOrganizationProject(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Organizations.DeleteOrganizationProject(context.Background(), *orgID, id)
			return err
		},
	})
}

// newProjectCmd builds the `project` command straight from the registry:
// unlike org/check, project's `get` was already a plain numeric lookup, so
// the generic factory's list/get/create/update/delete need no overrides.
func newProjectCmd() *cobra.Command {
	return newResourceCmd(registry["project"])
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
