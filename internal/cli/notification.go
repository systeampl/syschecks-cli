package cli

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-go/models"
)

// notificationListCols is the column set for `notification list`.
var notificationListCols = []string{"id", "name", "channel_type", "is_active"}

// newNotificationCmd groups notification channel commands: list, test <id>.
func newNotificationCmd() *cobra.Command {
	c := &cobra.Command{Use: "notification", Short: "Notification channels"}
	c.AddCommand(newNotificationListCmd(), newNotificationTestCmd())
	return c
}

func newNotificationListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notification channels",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := optionalOrgID(cmd, env)
			if err != nil {
				return err
			}
			channels, err := env.SDK.NotificationChannels.ListChannels(cmd.Context(), &models.ListChannelsParams{OrgId: orgID})
			if err != nil {
				return clierr.Config("listing notification channels: %v", err)
			}
			var rows []output.Row
			for _, c := range *channels {
				rows = append(rows, output.Row{
					"id": c.Id, "name": c.Name,
					"channel_type": string(c.ChannelType), "is_active": c.IsActive,
				})
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: notificationListCols, Rows: rows})
		},
	}
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
