package cli

import (
	"strconv"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// agentListCols is the column set for `agent list`.
var agentListCols = []string{"id", "name", "status", "last_seen"}

// agentTokenCols is the column set for `agent token`'s rendered output.
var agentTokenCols = []string{"token", "expires_at", "install_curl", "install_docker"}

// newAgentCmd groups agent commands: list, delete, and token. There is no
// `agent create`/`agent get`: agents self-register against the API using the
// token `agent token` mints, so the registration token is the only way an
// agent comes into existence from the CLI's side.
func newAgentCmd() *cobra.Command {
	c := &cobra.Command{Use: "agent", Short: "Agents"}
	c.AddCommand(
		newAgentListCmd(),
		newAgentDeleteCmd(),
		newAgentTokenCmd(),
	)
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
			items, err := extractItems(raw, "agents")
			if err != nil {
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
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: agentListCols, Rows: rows})
		},
	}
}

func newAgentDeleteCmd() *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:     "delete <agent_id>",
		Aliases: []string{"rm"},
		Short:   "Delete an agent from an organization",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid agent id %q: %v", args[0], err)
			}
			if !yes {
				if err := confirmDelete(cmd, "agent", id); err != nil {
					return err
				}
			}
			if _, err := env.SDK.Organizations.DeleteOrgAgent(cmd.Context(), orgID, id); err != nil {
				return clierr.Config("deleting agent %d: %v", id, err)
			}
			cmd.Printf("agent %d deleted\n", id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

// newAgentTokenCmd mints a registration token new agents use to self-register
// against the organization: it's the only way an agent gets created from the
// CLI's side, so this stands in for `agent create`.
func newAgentTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Create an agent registration token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			res, err := env.SDK.Organizations.CreateOrgAgentRegistrationToken(cmd.Context(), orgID)
			if err != nil {
				return clierr.Config("creating agent registration token: %v", err)
			}
			row := output.Row{
				"token":          res.Token,
				"expires_at":     res.ExpiresAt,
				"install_curl":   res.InstallCurl,
				"install_docker": res.InstallDocker,
			}
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: agentTokenCols, Rows: []output.Row{row}})
		},
	}
}
