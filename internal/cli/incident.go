package cli

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-go/models"
)

// incidentListCols is the column set for `incident list`.
var incidentListCols = []string{"id", "check", "status", "started_at"}

// The API names these differently from the columns we print: an incident is identified
// by the log row that opened it, the check by name, and the status is the worst one the
// incident reached. Reading `id`/`check`/`status` straight off the payload yielded empty
// cells on every row.
const (
	incidentIDField     = "start_log_id"
	incidentCheckField  = "check_name"
	incidentStatusField = "max_status"
)

// newIncidentCmd groups incident commands: list plus the composite-id
// actions (get/acknowledge/resolve) that address an incident by the
// (check_id, log_id) pair the API uses, rather than a single resource id —
// they don't fit the generic CRUD verb factory, so they're hand-written
// leaves here, same as check's run/pause/resume in check.go.
func newIncidentCmd() *cobra.Command {
	c := &cobra.Command{Use: "incident", Short: "Incidents"}
	c.AddCommand(
		newIncidentListCmd(),
		newIncidentGetCmd(),
		newIncidentAcknowledgeCmd(),
		newIncidentResolveCmd(),
	)
	return c
}

// parseIncidentIDs parses the <check_id> <log_id> positional args shared by
// incident get/acknowledge/resolve into ints, as clierr.Config errors (exit
// code 2) on a non-numeric arg rather than a panic.
func parseIncidentIDs(args []string) (checkID, logID int, err error) {
	checkID, err = strconv.Atoi(args[0])
	if err != nil {
		return 0, 0, clierr.Config("invalid check id %q: %v", args[0], err)
	}
	logID, err = strconv.Atoi(args[1])
	if err != nil {
		return 0, 0, clierr.Config("invalid log id %q: %v", args[1], err)
	}
	return checkID, logID, nil
}

func newIncidentGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <check_id> <log_id>",
		Short: "Show a single incident's detail",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			checkID, logID, err := parseIncidentIDs(args)
			if err != nil {
				return err
			}
			raw, err := env.SDK.Checks.GetCheckIncidentDetail(cmd.Context(), checkID, logID)
			if err != nil {
				return clierr.Config("getting incident %d/%d: %v", checkID, logID, err)
			}
			return renderRawObject(env, raw)
		},
	}
}

func newIncidentAcknowledgeCmd() *cobra.Command {
	var note string
	c := &cobra.Command{
		Use:     "acknowledge <check_id> <log_id>",
		Aliases: []string{"ack"},
		Short:   "Acknowledge an incident",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			checkID, logID, err := parseIncidentIDs(args)
			if err != nil {
				return err
			}
			var body models.AcknowledgeIncidentJSONRequestBody
			if cmd.Flags().Changed("note") {
				body.Note = &note
			}
			raw, err := env.SDK.Incidents.AcknowledgeIncident(cmd.Context(), checkID, logID, body)
			if err != nil {
				return clierr.Config("acknowledging incident %d/%d: %v", checkID, logID, err)
			}
			return renderRawObject(env, raw)
		},
	}
	c.Flags().StringVar(&note, "note", "", "optional note to attach to the acknowledgement")
	return c
}

func newIncidentResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <check_id> <log_id>",
		Short: "Resolve an incident",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			checkID, logID, err := parseIncidentIDs(args)
			if err != nil {
				return err
			}
			raw, err := env.SDK.Checks.ResolveCheckIncident(cmd.Context(), checkID, logID)
			if err != nil {
				return clierr.Config("resolving incident %d/%d: %v", checkID, logID, err)
			}
			return renderRawObject(env, raw)
		},
	}
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
					if s, _ := it[incidentStatusField].(string); s != status {
						continue
					}
				}
				rows = append(rows, output.Row{
					"id":         it[incidentIDField],
					"check":      it[incidentCheckField],
					"status":     it[incidentStatusField],
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
