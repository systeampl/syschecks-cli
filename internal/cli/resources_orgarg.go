package cli

import (
	"context"

	"github.com/systeampl/syschecks-go/models"
)

// teamCols is the column set for `team list`/`get`/`create`/`update`: unlike
// service, every team response shape (TeamWithCountsResponse,
// TeamDetailResponse, TeamResponse) carries id/name/slug/member_count/
// is_active, so one Cols (no ListCols) covers all four verbs.
var teamCols = []string{"id", "name", "slug", "member_count", "is_active"}

// teamFields is the flag/-f schema for `team create`/`team update`, generated
// from the flat scalar fields of models.TeamCreate: description and slug are
// optional, name is the only required field. TeamCreate has no nested/array
// fields, so every field gets a flag.
var teamFields = []Field{
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "slug", JSONKey: "slug", Kind: "string", Required: false, Help: "slug"},
}

// serviceCols is the column set for `service get`/`create`/`update`, which
// render models.ServiceDetailResponse/ServiceResponse: fields present on
// both. health_status and checks_count live only on the list response
// (ServiceListItemResponse) — serviceListCols below, not here, so get/create/
// update don't render blank cells for them.
var serviceCols = []string{"id", "name", "slug", "tier", "is_active"}

// serviceListCols is the column set for `service list` specifically
// (models.ServiceListItemResponse), which carries health_status/checks_count
// that the single-item responses don't.
var serviceListCols = []string{"id", "name", "slug", "health_status", "checks_count", "is_active"}

// serviceFields is the flag/-f schema for `service create`/`service update`,
// generated from the flat scalar fields of models.ServiceCreate: name is the
// only required field. metadata_json (map) and notification_channel_ids
// ([]int) are nested/array fields with no flag — they are -f-only.
var serviceFields = []Field{
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "docs-url", JSONKey: "docs_url", Kind: "string", Required: false, Help: "docs url"},
	{Name: "escalation-policy-id", JSONKey: "escalation_policy_id", Kind: "int", Required: false, Help: "escalation policy id"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "owner-team-id", JSONKey: "owner_team_id", Kind: "int", Required: false, Help: "owner team id"},
	{Name: "repo-url", JSONKey: "repo_url", Kind: "string", Required: false, Help: "repo url"},
	{Name: "slack-channel", JSONKey: "slack_channel", Kind: "string", Required: false, Help: "slack channel"},
	{Name: "tier", JSONKey: "tier", Kind: "string", Required: false, Help: "tier"},
}

func init() {
	register(&Resource{
		Name: "team", Cols: teamCols, Org: OrgArg, Fields: teamFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			teams, err := env.SDK.Teams.ListTeams(context.Background(), *orgID)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*teams))
			for _, t := range *teams {
				m, err := toMap(t)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			t, err := env.SDK.Teams.GetTeam(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(t)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.TeamCreate](body)
			if err != nil {
				return nil, err
			}
			t, err := env.SDK.Teams.CreateTeam(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(t)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.TeamUpdate](body)
			if err != nil {
				return nil, err
			}
			t, err := env.SDK.Teams.UpdateTeam(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(t)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Teams.DeleteTeam(context.Background(), *orgID, id)
			return err
		},
	})

	register(&Resource{
		Name: "service", Cols: serviceCols, ListCols: serviceListCols, Org: OrgArg, Fields: serviceFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			services, err := env.SDK.Services.ListServices(context.Background(), *orgID)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*services))
			for _, s := range *services {
				m, err := toMap(s)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			s, err := env.SDK.Services.GetService(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ServiceCreate](body)
			if err != nil {
				return nil, err
			}
			s, err := env.SDK.Services.CreateService(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ServiceUpdate](body)
			if err != nil {
				return nil, err
			}
			s, err := env.SDK.Services.UpdateService(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Services.DeleteService(context.Background(), *orgID, id)
			return err
		},
	})
}
