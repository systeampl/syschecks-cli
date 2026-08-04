package cli

import (
	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-go/models"
)

// incidentListCols is the column set for `incident list`.
var incidentListCols = []string{"id", "check", "status", "started_at"}

// newIncidentCmd groups incident read commands: list.
func newIncidentCmd() *cobra.Command {
	c := &cobra.Command{Use: "incident", Short: "Incidents"}
	c.AddCommand(newIncidentListCmd())
	return c
}

func newIncidentListCmd() *cobra.Command {
	var status string
	c := &cobra.Command{
		Use:   "list",
		Short: "List incidents",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := optionalOrgID(cmd, env)
			if err != nil {
				return err
			}
			raw, err := env.SDK.Incidents.ListIncidents(cmd.Context(), &models.ListIncidentsParams{OrganizationId: orgID})
			if err != nil {
				return clierr.Config("listing incidents: %v", err)
			}
			items, err := extractItems(raw, "incidents")
			if err != nil {
				return clierr.Config("decoding incidents list: %v", err)
			}
			var rows []output.Row
			for _, it := range items {
				if status != "" {
					if s, _ := it["status"].(string); s != status {
						continue
					}
				}
				rows = append(rows, output.Row{
					"id":         it["id"],
					"check":      it["check"],
					"status":     it["status"],
					"started_at": it["started_at"],
				})
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: incidentListCols, Rows: rows})
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status")
	return c
}
