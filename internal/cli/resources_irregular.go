package cli

import (
	"context"
	"encoding/json"

	"github.com/systeampl/syschecks-go/models"
)

// contactMethodCols is the column set for `contact-method` list/create/
// update, all of which render models.ContactMethodResponse (there is no
// `get` — see the registration below, ContactMethods has no
// GetContactMethod endpoint at all).
var contactMethodCols = []string{"id", "kind", "label", "value", "enabled", "verified"}

// contactMethodFields is the flag/-f schema for `contact-method create`
// (and reused, same as every other resource in this registry, for
// `update`), generated from the flat scalar fields of
// models.ContactMethodCreate: kind is the only required field. Update's
// shape (models.ContactMethodUpdate: enabled/label) genuinely diverges from
// create's (kind/label/value) rather than merely narrowing it — mapToBody's
// JSON round-trip silently drops whichever of these flags don't exist on
// the target struct for a given verb (e.g. --kind is accepted but ignored
// on `update`), the same tradeoff already made for playbook/maintenance-
// window's Fields (Task 6/7) rather than modeling two divergent flag sets.
var contactMethodFields = []Field{
	{Name: "kind", JSONKey: "kind", Kind: "string", Required: true, Help: "contact method kind (e.g. email, sms, push)"},
	{Name: "label", JSONKey: "label", Kind: "string", Required: false, Help: "label"},
	{Name: "value", JSONKey: "value", Kind: "string", Required: false, Help: "value (e.g. the email address or phone number)"},
}

// integrationKeyCols is the column set for `integration-key` list/create:
// unlike every other resource in the registry, IntegrationKeys has no typed
// response model at all (ListIntegrationKeys/CreateIntegrationKey/
// RevokeIntegrationKey all return a bare json.RawMessage per the generated
// SDK, and the backend's own OpenAPI schema for these three endpoints is
// empty ({}) with no components/schemas reference to confirm field names
// against). id/name/key/created_at are the reasonable guess for an API-key
// resource shape (matching how every other create/list response in this API
// names its fields), but are NOT verified against a real response — see the
// task report's concerns.
var integrationKeyCols = []string{"id", "name", "key", "created_at"}

// integrationKeyFields is the flag/-f schema for `integration-key create`,
// generated from the flat scalar fields of models.IntegrationKeyCreate:
// name and escalation-policy-id are required. There is no `update` (see the
// registration below), so unlike every other Fields var in the registry
// this one is create-only, not shared with an update verb.
var integrationKeyFields = []Field{
	{Name: "escalation-policy-id", JSONKey: "escalation_policy_id", Kind: "int", Required: true, Help: "escalation policy id"},
	{Name: "grouping-type", JSONKey: "grouping_type", Kind: "string", Required: false, Help: "grouping type"},
	{Name: "grouping-window-seconds", JSONKey: "grouping_window_seconds", Kind: "int", Required: false, Help: "grouping window seconds"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
}

func init() {
	register(&Resource{
		Name: "contact-method", Cols: contactMethodCols, Org: OrgNone, Fields: contactMethodFields,
		// listFn ignores orgID: contact methods are scoped to the
		// authenticated user/token, not to an organization at all.
		listFn: func(env *cmdCtx, _ *int) ([]map[string]any, error) {
			methods, err := env.SDK.ContactMethods.ListContactMethods(context.Background())
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*methods))
			for _, m := range *methods {
				row, err := toMap(m)
				if err != nil {
					return nil, err
				}
				items = append(items, row)
			}
			return items, nil
		},
		// getFn is deliberately nil: there is no GetContactMethod endpoint in
		// the SDK, so newResourceCmd (crud.go) omits `contact-method get`
		// entirely rather than generating a leaf with nowhere to call.
		createFn: func(env *cmdCtx, _ *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ContactMethodCreate](body)
			if err != nil {
				return nil, err
			}
			m, err := env.SDK.ContactMethods.CreateContactMethod(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(m)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.ContactMethodUpdate](body)
			if err != nil {
				return nil, err
			}
			m, err := env.SDK.ContactMethods.UpdateContactMethod(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(m)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.ContactMethods.DeleteContactMethod(context.Background(), id)
			return err
		},
	})

	register(&Resource{
		Name: "integration-key", Cols: integrationKeyCols, Org: OrgArg, Fields: integrationKeyFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			raw, err := env.SDK.IntegrationKeys.ListIntegrationKeys(context.Background(), *orgID)
			if err != nil {
				return nil, err
			}
			return extractItems(raw, "integration_keys")
		},
		// getFn and updateFn are deliberately nil: the SDK exposes no
		// GetIntegrationKey/UpdateIntegrationKey endpoint, so newResourceCmd
		// (crud.go) omits both `integration-key get` and `integration-key
		// update` rather than generating leaves with nowhere to call.
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.IntegrationKeyCreate](body)
			if err != nil {
				return nil, err
			}
			raw, err := env.SDK.IntegrationKeys.CreateIntegrationKey(context.Background(), *orgID, b)
			if err != nil {
				return nil, err
			}
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				return nil, err
			}
			return m, nil
		},
		// deleteFn maps to RevokeIntegrationKey: there is no separate
		// "delete" endpoint for an integration key, only revoke, which the
		// generated SDK already treats any 2xx status as success for (see
		// IntegrationKeys.RevokeIntegrationKey in resources_gen.go) — no
		// further status check is needed here.
		deleteFn: func(env *cmdCtx, orgID *int, id int) error {
			_, err := env.SDK.IntegrationKeys.RevokeIntegrationKey(context.Background(), *orgID, id)
			return err
		},
	})
}
