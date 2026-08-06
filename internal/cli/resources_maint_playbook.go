package cli

import (
	"context"

	"github.com/systeampl/syschecks-go/models"
)

// maintenanceWindowCols is the column set for `maintenance-window`
// list/get/create/update: every verb renders models.MaintenanceWindowResponse
// (Maintenance.ListMaintenanceWindows returns *[]MaintenanceWindowResponse,
// same shape as the single-item get/create/update responses), so one Cols
// (no ListCols) covers all four verbs, same as team/oncall-schedule.
var maintenanceWindowCols = []string{"id", "name", "start_time", "end_time", "is_active", "is_recurring"}

// maintenanceWindowFields is the flag/-f schema for `maintenance-window
// create`/`maintenance-window update`, generated from the flat scalar fields
// of models.MaintenanceWindowCreate: name/start-time/end-time are required.
// check_ids/project_ids ([]int) and recurrence_pattern (map) are
// nested/array fields with no flag — they are -f-only.
var maintenanceWindowFields = []Field{
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "end-time", JSONKey: "end_time", Kind: "string", Required: true, Help: "end time"},
	{Name: "is-recurring", JSONKey: "is_recurring", Kind: "bool", Required: false, Help: "is recurring"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "notify-before-minutes", JSONKey: "notify_before_minutes", Kind: "int", Required: false, Help: "notify before minutes"},
	{Name: "organization-id", JSONKey: "organization_id", Kind: "int", Required: false, Help: "organization id"},
	{Name: "start-time", JSONKey: "start_time", Kind: "string", Required: true, Help: "start time"},
	{Name: "timezone", JSONKey: "timezone", Kind: "string", Required: false, Help: "timezone"},
}

// playbookCols is the column set for `playbook` list/get/create/update: every
// verb renders models.PlaybookResponse (Playbooks.ListPlaybooks returns
// *[]PlaybookResponse, same shape as the single-item get/create/update
// responses), so one Cols (no ListCols) covers all four verbs.
var playbookCols = []string{"id", "name", "trigger_type", "is_active", "service_id"}

// playbookFields is the flag/-f schema for `playbook create`/`playbook
// update`, generated from the flat scalar fields of models.PlaybookCreate:
// name and trigger-type are required. steps ([]PlaybookStepSchema) and
// trigger_conditions (map) are nested/array fields with no flag — they are
// -f-only.
var playbookFields = []Field{
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "service-id", JSONKey: "service_id", Kind: "int", Required: false, Help: "service id"},
	{Name: "suppress-default-notifications", JSONKey: "suppress_default_notifications", Kind: "bool", Required: false, Help: "suppress default notifications"},
	{Name: "trigger-type", JSONKey: "trigger_type", Kind: "string", Required: true, Help: "trigger type"},
}

func init() {
	register(&Resource{
		Name: "maintenance-window", Cols: maintenanceWindowCols, Org: OrgParam, Fields: maintenanceWindowFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			windows, err := env.SDK.Maintenance.ListMaintenanceWindows(context.Background(), &models.ListMaintenanceWindowsParams{OrganizationId: orgID})
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*windows))
			for _, w := range *windows {
				m, err := toMap(w)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, _ *int, id int) (map[string]any, error) {
			w, err := env.SDK.Maintenance.GetMaintenanceWindow(context.Background(), id)
			if err != nil {
				return nil, err
			}
			return toMap(w)
		},
		createFn: func(env *cmdCtx, _ *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.MaintenanceWindowCreate](body)
			if err != nil {
				return nil, err
			}
			w, err := env.SDK.Maintenance.CreateMaintenanceWindow(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(w)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.MaintenanceWindowUpdate](body)
			if err != nil {
				return nil, err
			}
			w, err := env.SDK.Maintenance.UpdateMaintenanceWindow(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(w)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.Maintenance.DeleteMaintenanceWindow(context.Background(), id)
			return err
		},
	})

	register(&Resource{
		Name: "playbook", Cols: playbookCols, Org: OrgArg, Fields: playbookFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			playbooks, err := env.SDK.Playbooks.ListPlaybooks(context.Background(), *orgID, nil)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*playbooks))
			for _, p := range *playbooks {
				m, err := toMap(p)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			p, err := env.SDK.Playbooks.GetPlaybook(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.PlaybookCreate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Playbooks.CreatePlaybook(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		updateFn: func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.PlaybookUpdate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.Playbooks.UpdatePlaybook(context.Background(), *orgID, id, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Playbooks.DeletePlaybook(context.Background(), *orgID, id)
			return err
		},
	})
}
