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
			items, err := listIncidents(cmd, env, orgID)
			if err != nil {
				return err
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
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: incidentListCols, Rows: rows})
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status")
	return c
}

// Incidents are paged too: the API defaults to 50 per page and caps at 200. The
// --status filter runs client-side (the API's own `status` takes a different
// vocabulary — ongoing/resolved — not the per-incident status shown here), so
// filtering a single page would answer over a slice of the data.
const (
	incidentListPageSize = 200
	incidentListMaxPages = 200
)

// listIncidents walks every page and returns the incidents as raw maps.
func listIncidents(cmd *cobra.Command, env *cmdCtx, orgID *int) ([]map[string]any, error) {
	var all []map[string]any
	for page := 0; page < incidentListMaxPages; page++ {
		offset := page * incidentListPageSize
		raw, err := env.SDK.Incidents.ListIncidents(cmd.Context(), &models.ListIncidentsParams{
			OrganizationId: orgID,
			Limit:          intPtr(incidentListPageSize),
			Offset:         intPtr(offset),
		})
		if err != nil {
			return nil, clierr.Config("listing incidents: %v", err)
		}
		items, err := extractItems(raw, "incidents")
		if err != nil {
			return nil, clierr.Config("decoding incidents list: %v", err)
		}
		all = append(all, items...)
		if len(items) < incidentListPageSize {
			return all, nil
		}
	}
	return all, nil
}
