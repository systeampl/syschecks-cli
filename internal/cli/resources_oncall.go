package cli

import (
	"context"

	"github.com/systeampl/syschecks-go/models"
)

// oncallScheduleCols is the column set for `oncall-schedule`
// list/get/create/update: every verb renders models.OncallScheduleResponse
// (Oncall.ListSchedules returns *[]OncallScheduleResponse, same shape as the
// single-item get/create/update responses), so one Cols (no ListCols) covers
// all four verbs, same as team.
var oncallScheduleCols = []string{"id", "name", "rotation_type", "timezone", "is_active"}

// oncallScheduleFields is the flag/-f schema for `oncall-schedule create`/
// `oncall-schedule update`, generated from the flat scalar fields of
// models.ScheduleCreate: name is the only required field. Participants
// ([]ParticipantSchema) is a nested/array field with no flag — it is -f-only.
var oncallScheduleFields = []Field{
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "rotation-type", JSONKey: "rotation_type", Kind: "string", Required: false, Help: "rotation type"},
	{Name: "timezone", JSONKey: "timezone", Kind: "string", Required: false, Help: "timezone"},
}

// escalationPolicyCols is the column set for `escalation-policy`
// list/get/create/update: every verb renders models.EscalationPolicyResponse
// (Oncall.ListPolicies returns *[]EscalationPolicyResponse, same shape as the
// single-item get/create/update responses), so one Cols (no ListCols) covers
// all four verbs.
var escalationPolicyCols = []string{"id", "name", "is_active", "team_id", "team_name"}

// escalationPolicyFields is the flag/-f schema for `escalation-policy
// create`/`escalation-policy update`, generated from the flat scalar fields
// of models.EscalationPolicyCreate: name is the only field, and it's
// required. Steps ([]EscalationStepSchema) is a nested/array field with no
// flag — it is -f-only.
var escalationPolicyFields = []Field{
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
}

func init() {
	register(&Resource{
		Name: "oncall-schedule", Cols: oncallScheduleCols, Org: OrgArg, Fields: oncallScheduleFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			schedules, err := env.SDK.Oncall.ListSchedules(context.Background(), *orgID, nil)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*schedules))
			for _, s := range *schedules {
				m, err := toMap(s)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			s, err := env.SDK.Oncall.GetSchedule(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ScheduleCreate](body)
			if err != nil {
				return nil, err
			}
			s, err := env.SDK.Oncall.CreateSchedule(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ScheduleUpdate](body)
			if err != nil {
				return nil, err
			}
			s, err := env.SDK.Oncall.UpdateSchedule(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(s)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Oncall.DeleteSchedule(context.Background(), *orgID, id)
			return err
		},
	})

	register(&Resource{
		Name: "escalation-policy", Cols: escalationPolicyCols, Org: OrgArg, Fields: escalationPolicyFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			policies, err := env.SDK.Oncall.ListPolicies(context.Background(), *orgID, nil)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*policies))
			for _, p := range *policies {
				m, err := toMap(p)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			p, err := env.SDK.Oncall.GetPolicy(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.EscalationPolicyCreate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Oncall.CreatePolicy(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.EscalationPolicyUpdate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Oncall.UpdatePolicy(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Oncall.DeletePolicy(context.Background(), *orgID, id)
			return err
		},
	})
}
