package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// agentListCols is the column set for `agent list`.
var agentListCols = []string{"id", "name", "status", "last_seen"}

// newAgentCmd groups agent read commands: list. Org-scoped, resolved via
// resolveOrgID from the --org global flag.
func newAgentCmd() *cobra.Command {
	c := &cobra.Command{Use: "agent", Short: "Agents"}
	c.AddCommand(newAgentListCmd())
	return c
}

func newAgentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List agents in an organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			raw, err := env.SDK.Organizations.ListOrgAgents(cmd.Context(), orgID)
			if err != nil {
				return clierr.Config("listing agents: %v", err)
			}
			var items []map[string]any
			if err := json.Unmarshal(raw, &items); err != nil {
				return clierr.Config("decoding agents list: %v", err)
			}
			var rows []output.Row
			for _, it := range items {
				rows = append(rows, output.Row{
					"id":        it["id"],
					"name":      it["name"],
					"status":    it["status"],
					"last_seen": it["last_seen"],
				})
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: agentListCols, Rows: rows})
		},
	}
}
