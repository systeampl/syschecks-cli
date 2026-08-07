package cli

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-go/models"
)

// notificationListCols is the column set for `notification list`/`get`.
var notificationListCols = []string{"id", "name", "channel_type", "is_active"}

// notificationFields is the flag/-f schema for `notification create`/
// `notification update`, generated from the flat scalar fields of
// models.NotificationChannelCreate; config (the per-channel-type payload) is
// -f-only since it's a nested object, not a flag.
var notificationFields = []Field{
	{Name: "channel-type", JSONKey: "channel_type", Kind: "string", Required: true, Help: "channel type"},
	{Name: "is-active", JSONKey: "is_active", Kind: "bool", Required: false, Help: "is active"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "organization-id", JSONKey: "organization_id", Kind: "int", Required: false, Help: "organization id"},
}

func init() {
	register(&Resource{
		Name: "notification", Cols: notificationListCols, Org: OrgParam, Fields: notificationFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			channels, err := env.SDK.NotificationChannels.ListChannels(context.Background(), &models.ListChannelsParams{OrgId: orgID})
			if err != nil {
				return nil, err
			}
			items := make([]map[string]any, 0, len(*channels))
			for _, c := range *channels {
				m, err := toMap(c)
				if err != nil {
					return nil, err
				}
				items = append(items, m)
			}
			return items, nil
		},
		getFn: func(env *cmdCtx, _ *int, id int) (map[string]any, error) {
			c, err := env.SDK.NotificationChannels.GetChannel(context.Background(), id)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		createFn: func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error) {
			if orgID != nil {
				if _, ok := body["organization_id"]; !ok {
					body["organization_id"] = *orgID
				}
			}
			b, err := mapToBody[models.NotificationChannelCreate](body)
			if err != nil {
				return nil, err
			}
			c, err := env.SDK.NotificationChannels.CreateChannel(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.NotificationChannelUpdate](body)
			if err != nil {
				return nil, err
			}
			c, err := env.SDK.NotificationChannels.UpdateChannel(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.NotificationChannels.DeleteChannel(context.Background(), id)
			return err
		},
	})
}

// newNotificationCmd builds the `notification` command from the registry
// (list/get/create/update/delete) plus the bespoke `test` subcommand, which
// has no registry equivalent (it isn't a CRUD verb).
func newNotificationCmd() *cobra.Command {
	c := newResourceCmd(registry["notification"])
	c.AddCommand(newNotificationTestCmd())
	return c
}

func newNotificationTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Send a test notification through a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid channel id %q: %v", args[0], err)
			}
			res, err := env.SDK.NotificationChannels.TestChannel(cmd.Context(), id)
			if err != nil {
				return clierr.Config("testing notification channel %d: %v", id, err)
			}
			cmd.Printf("success=%t: %s\n", res.Success, res.Message)
			return nil
		},
	}
}
