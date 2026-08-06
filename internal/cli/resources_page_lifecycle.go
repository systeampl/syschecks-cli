package cli

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-go/models"
)

// statusPageCols is the column set for `status-page` list/get/create/update:
// every verb renders models.StatusPageResponse (StatusPages.ListStatusPages
// returns *[]StatusPageResponse, same shape as the single-item get/create/
// update responses), so one Cols (no ListCols) covers all four verbs, same as
// team/oncall-schedule.
var statusPageCols = []string{"id", "name", "slug", "is_active", "is_public", "custom_domain"}

// statusPageFields is the flag/-f schema for `status-page create`/`status-page
// update`, generated from the flat scalar fields of models.StatusPageCreate:
// name and slug are the only required fields. check_ids ([]int) is a
// nested/array field with no flag — it is -f-only.
var statusPageFields = []Field{
	{Name: "custom-domain", JSONKey: "custom_domain", Kind: "string", Required: false, Help: "custom domain"},
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "inbound-min-severity", JSONKey: "inbound_min_severity", Kind: "string", Required: false, Help: "inbound min severity"},
	{Name: "is-public", JSONKey: "is_public", Kind: "bool", Required: false, Help: "is public"},
	{Name: "logo-url", JSONKey: "logo_url", Kind: "string", Required: false, Help: "logo url"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "show-inbound-incidents", JSONKey: "show_inbound_incidents", Kind: "bool", Required: false, Help: "show inbound incidents"},
	{Name: "slug", JSONKey: "slug", Kind: "string", Required: true, Help: "slug"},
}

// lifecycleWatchCols is the column set for `lifecycle-watch`
// list/get/create (there is no `update`, see the registration below):
// get/list render models.LifecycleWatchResponse, and create (backed by the
// untyped Upsert endpoint, see upsertLifecycleWatch below) decodes into the
// same shape. channel_ids/channel_names ([]int/[]string) are left off, same
// as playbook's steps.
var lifecycleWatchCols = []string{
	"id", "resource_type", "resource_id", "vendor", "platform",
	"notify_on_new", "notify_7d", "notify_30d", "notify_90d",
}

// lifecycleWatchFields is the flag/-f schema for `lifecycle-watch create`
// (there is no `update` — see the registration below): generated from the
// flat scalar fields of models.LifecycleWatchUpsert, plus vendor/
// resource-type/resource-id promoted to Required even though the SDK model
// only marks Vendor non-optional. UpsertLifecycleWatch has no id path arg —
// the server keys the upsert by exactly these three fields, so a `create`
// that omitted one would silently upsert against an incomplete/wrong key
// instead of failing client-side. channel_ids ([]int) is a nested/array
// field with no flag — it is -f-only.
var lifecycleWatchFields = []Field{
	{Name: "notify-30d", JSONKey: "notify_30d", Kind: "bool", Required: false, Help: "notify 30 days before end of life"},
	{Name: "notify-7d", JSONKey: "notify_7d", Kind: "bool", Required: false, Help: "notify 7 days before end of life"},
	{Name: "notify-90d", JSONKey: "notify_90d", Kind: "bool", Required: false, Help: "notify 90 days before end of life"},
	{Name: "notify-on-new", JSONKey: "notify_on_new", Kind: "bool", Required: false, Help: "notify on newly discovered resource"},
	{Name: "platform", JSONKey: "platform", Kind: "string", Required: false, Help: "platform"},
	{Name: "resource-id", JSONKey: "resource_id", Kind: "string", Required: true, Help: "resource id (part of the upsert key)"},
	{Name: "resource-type", JSONKey: "resource_type", Kind: "string", Required: true, Help: "resource type (part of the upsert key)"},
	{Name: "vendor", JSONKey: "vendor", Kind: "string", Required: true, Help: "vendor (part of the upsert key)"},
}

func init() {
	register(&Resource{
		Name: "status-page", Cols: statusPageCols, Org: OrgNone, Fields: statusPageFields,
		listFn: func(env *cmdCtx, _ *int) ([]map[string]any, error) {
			pages, err := env.SDK.StatusPages.ListStatusPages(context.Background(), &models.ListStatusPagesParams{})
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*pages))
			for _, p := range *pages {
				m, err := toMap(p)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, _ *int, id int) (map[string]any, error) {
			p, err := env.SDK.StatusPages.GetStatusPage(context.Background(), id)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		createFn: func(env *cmdCtx, _ *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.StatusPageCreate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.StatusPages.CreateStatusPage(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.StatusPageUpdate](body)
			if err != nil {
				return nil, err
			}
			p, err := env.SDK.StatusPages.UpdateStatusPage(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(p)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.StatusPages.DeleteStatusPage(context.Background(), id)
			return err
		},
	})

	register(&Resource{
		Name: "lifecycle-watch", Cols: lifecycleWatchCols, Org: OrgArg, Fields: lifecycleWatchFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			watches, err := env.SDK.Lifecycle.ListLifecycleWatches(context.Background(), *orgID)
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*watches))
			for _, w := range *watches {
				m, err := toMap(w)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, orgID *int, id int) (map[string]any, error) {
			w, err := env.SDK.Lifecycle.GetLifecycleWatch(context.Background(), *orgID, id)
			if err != nil {
				return nil, err
			}
			return toMap(w)
		},
		// createFn maps to UpsertLifecycleWatch: the API has no separate
		// create/update for lifecycle watches, only an idempotent upsert keyed
		// by (vendor, resource_type, resource_id) in the body — not by an id
		// path arg. There is deliberately no updateFn: an id-addressable
		// `lifecycle-watch update <id>` would be misleading (the id is
		// discarded; the SDK call has nowhere to put it) and, worse, unsafe —
		// bodyFromFlagsAndFile only enforces Required on create, so an
		// `update <id>` with no flags at all would silently upsert an
		// empty-keyed watch rather than touching <id>. Re-running `create` with
		// the same key fields is the supported way to change an existing
		// watch's notification settings; newLifecycleWatchCmd documents this on
		// the `create` subcommand. See newApplyCmd/applyDoc (apply.go), which
		// already turns a nil updateFn into a clierr.Config error for any `-f`
		// document that carries an "id" for this kind, rather than panicking.
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			return upsertLifecycleWatch(env, *orgID, body)
		},
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.Lifecycle.DeleteLifecycleWatch(context.Background(), *orgID, id)
			return err
		},
	})
}

// newLifecycleWatchCmd builds the `lifecycle-watch` command from the registry
// (list/get/create/delete — no `update`, see the registration in init above)
// and documents the upsert semantics of `create` on its own Long text, since
// the generic factory's "Create a lifecycle-watch" Short is misleading on its
// own: running `create` again with the same vendor/resource-type/resource-id
// updates the existing watch's notification settings in place rather than
// erroring or duplicating it.
func newLifecycleWatchCmd() *cobra.Command {
	c := newResourceCmd(registry["lifecycle-watch"])
	for _, sc := range c.Commands() {
		if sc.Name() == "create" {
			sc.Long = "Create or update a lifecycle watch.\n\n" +
				"This is an idempotent upsert keyed by --vendor/--resource-type/--resource-id " +
				"(all three required): the underlying API has no id-addressable update, so " +
				"running `create` again with the same vendor/resource-type/resource-id changes " +
				"that watch's notification settings in place instead of creating a duplicate. " +
				"There is no `lifecycle-watch update` command."
		}
	}
	return c
}

// upsertLifecycleWatch drives Lifecycle.UpsertLifecycleWatch, which — unlike
// every other create/update endpoint in the registry — returns an untyped
// json.RawMessage rather than a typed response struct: it is decoded straight
// into a map[string]any rather than round-tripped through toMap.
func upsertLifecycleWatch(env *cmdCtx, orgID int, body map[string]any) (map[string]any, error) {
	b, err := mapToBody[models.LifecycleWatchUpsert](body)
	if err != nil {
		return nil, err
	}
	raw, err := env.SDK.Lifecycle.UpsertLifecycleWatch(context.Background(), orgID, b)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}
