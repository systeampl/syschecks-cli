package cli

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-go/models"
)

// checkListCols/checkGetCols are the column sets for `check list` and
// `check get` respectively; get additionally shows the URL.
var (
	checkListCols = []string{"id", "name", "type", "status"}
	checkGetCols  = []string{"id", "name", "type", "status", "url"}
)

// checkListItem mirrors the fields we need from ListChecks' untyped
// json.RawMessage response: the SDK doesn't type this endpoint, so we decode
// only what the CLI displays / needs for name resolution.
type checkListItem struct {
	Id     int     `json:"id"`
	Name   string  `json:"name"`
	Type   *string `json:"type"`
	Status string  `json:"status"`
}

// checkSettlePollInterval is how often `check run --wait` re-polls
// GetCheckDetails while waiting for the check to settle.
const checkSettlePollInterval = 2 * time.Second

// newCheckCmd groups check commands: list, get, run, pause, resume,
// test-alert.
func newCheckCmd() *cobra.Command {
	c := &cobra.Command{Use: "check", Short: "Checks"}
	c.AddCommand(
		newCheckListCmd(),
		newCheckGetCmd(),
		newCheckRunCmd(),
		newCheckPauseCmd(),
		newCheckResumeCmd(),
		newCheckTestAlertCmd(),
	)
	return c
}

func newCheckListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List checks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			items, err := listChecks(cmd, env)
			if err != nil {
				return err
			}
			var rows []output.Row
			for _, it := range items {
				rows = append(rows, output.Row{"id": it.Id, "name": it.Name, "type": deref(it.Type), "status": it.Status})
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: checkListCols, Rows: rows})
		},
	}
}

func newCheckGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id|name>",
		Short: "Show a single check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			id, err := resolveCheckID(cmd, env, args[0])
			if err != nil {
				return err
			}
			c, err := env.SDK.Checks.GetCheckDetails(cmd.Context(), id)
			if err != nil {
				return clierr.Config("getting check %d: %v", id, err)
			}
			row := output.Row{
				"id": c.Id, "name": c.Name, "type": deref(c.Type),
				"status": string(c.Status), "url": deref(c.Url),
			}
			return output.Render(env.Out, env.Format, env.Quiet, output.Table{Cols: checkGetCols, Rows: []output.Row{row}})
		},
	}
}

func newCheckRunCmd() *cobra.Command {
	var wait bool
	var timeout time.Duration
	c := &cobra.Command{
		Use:   "run <id|name>",
		Short: "Trigger a check run now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			id, err := resolveCheckID(cmd, env, args[0])
			if err != nil {
				return err
			}
			if _, err := env.SDK.Checks.RunCheckNow(cmd.Context(), id); err != nil {
				return clierr.Config("running check %d: %v", id, err)
			}
			if !wait {
				cmd.Printf("triggered check %d\n", id)
				return nil
			}
			status, err := waitForCheckSettle(cmd.Context(), env, id, timeout)
			if err != nil {
				return err
			}
			if status == models.DOWN {
				return clierr.Fail("check %d is DOWN", id)
			}
			cmd.Printf("check %d is %s\n", id, status)
			return nil
		},
	}
	c.Flags().BoolVar(&wait, "wait", false, "wait for the check to settle after triggering")
	c.Flags().DurationVar(&timeout, "timeout", 120*time.Second, "max time to wait with --wait")
	return c
}

// waitForCheckSettle polls GetCheckDetails on checkSettlePollInterval until
// the check's status is terminal (UP, DOWN, DEGRADED) or timeout elapses.
// The first poll happens immediately, with no leading sleep, so a check that
// has already settled by the time run-now returns resolves without delay.
func waitForCheckSettle(ctx context.Context, env *cmdCtx, id int, timeout time.Duration) (models.CheckStatus, error) {
	deadline := time.Now().Add(timeout)
	for {
		c, err := env.SDK.Checks.GetCheckDetails(ctx, id)
		if err != nil {
			return "", clierr.Config("polling check %d: %v", id, err)
		}
		switch c.Status {
		case models.UP, models.DOWN, models.DEGRADED:
			return c.Status, nil
		}
		if time.Now().After(deadline) {
			return "", clierr.Config("timed out waiting for check %d to settle (last status %s)", id, c.Status)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(checkSettlePollInterval):
		}
	}
}

func newCheckPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <id|name>",
		Short: "Pause a check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setCheckActive(cmd, args[0], false, "paused")
		},
	}
}

func newCheckResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <id|name>",
		Short: "Resume a paused check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setCheckActive(cmd, args[0], true, "resumed")
		},
	}
}

func setCheckActive(cmd *cobra.Command, arg string, active bool, verb string) error {
	env, err := cmdEnv(cmd)
	if err != nil {
		return err
	}
	id, err := resolveCheckID(cmd, env, arg)
	if err != nil {
		return err
	}
	c, err := env.SDK.Checks.UpdateCheck(cmd.Context(), id, models.UpdateCheckJSONRequestBody{IsActive: boolPtr(active)})
	if err != nil {
		return clierr.Config("updating check %d: %v", id, err)
	}
	cmd.Printf("check %d (%s) %s\n", c.Id, c.Name, verb)
	return nil
}

func newCheckTestAlertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test-alert <id|name>",
		Short: "Send a test alert for a check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			id, err := resolveCheckID(cmd, env, args[0])
			if err != nil {
				return err
			}
			if _, err := env.SDK.Checks.SendTestAlert(cmd.Context(), id, models.SendTestAlertJSONRequestBody{}); err != nil {
				return clierr.Config("sending test alert for check %d: %v", id, err)
			}
			cmd.Printf("test alert sent for check %d\n", id)
			return nil
		},
	}
}

// resolveCheckID resolves a check id|name CLI arg to a numeric id: a bare
// integer is used as-is, anything else is looked up by exact name match
// against ListChecks. Zero or multiple matches are config errors (exit 2).
func resolveCheckID(cmd *cobra.Command, env *cmdCtx, arg string) (int, error) {
	if id, err := strconv.Atoi(arg); err == nil {
		return id, nil
	}
	items, err := listChecks(cmd, env)
	if err != nil {
		return 0, err
	}
	var matches []checkListItem
	for _, it := range items {
		if it.Name == arg {
			matches = append(matches, it)
		}
	}
	switch len(matches) {
	case 0:
		return 0, clierr.Config("no check named %q found", arg)
	case 1:
		return matches[0].Id, nil
	default:
		return 0, clierr.Config("multiple checks named %q found; use the numeric id", arg)
	}
}

// listChecks decodes ListChecks' untyped json.RawMessage response into
// checkListItems.
func listChecks(cmd *cobra.Command, env *cmdCtx) ([]checkListItem, error) {
	orgID, err := optionalOrgID(cmd, env)
	if err != nil {
		return nil, err
	}
	raw, err := env.SDK.Checks.ListChecks(cmd.Context(), &models.ListChecksParams{OrgId: orgID})
	if err != nil {
		return nil, clierr.Config("listing checks: %v", err)
	}
	var items []checkListItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, clierr.Config("decoding checks list: %v", err)
	}
	return items, nil
}

// boolPtr returns a pointer to b, for the SDK's optional *bool fields.
func boolPtr(b bool) *bool { return &b }
