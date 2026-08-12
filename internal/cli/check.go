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
// `check get` respectively; get additionally shows the URL. check keeps its
// own bespoke `get` (id-or-name, see newCheckGetCmd) instead of the
// registry's generic one, so these stay independent of Resource.Cols (which
// only drives `check list`/`check create`/`check update`).
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

// Listing checks is paged: the API caps a page at 500 and defaults to 100, so
// asking once returns a slice of a large organization with nothing to say so.
// checkListMaxPages bounds the walk — a server that ignored ?offset= would
// otherwise hand back the same page forever.
const (
	checkListPageSize = 500
	checkListMaxPages = 200
)

// intPtr returns a pointer to i, for the SDK's optional *int params.
func intPtr(i int) *int { return &i }

// checkFields is the flag/-f schema for `check create`/`check update`,
// generated from the flat scalar fields of models.CheckCreate: name and
// project_id are the only fields the API requires, everything else configures
// one check type or another and is optional. Nested/array fields (http
// headers, DNS expected ips, api scenario steps, ...) have no flag — they are
// -f-only.
var checkFields = []Field{
	{Name: "alert-after-failures", JSONKey: "alert_after_failures", Kind: "int", Required: false, Help: "alert after failures"},
	{Name: "assigned-agent-id", JSONKey: "assigned_agent_id", Kind: "int", Required: false, Help: "assigned agent id"},
	{Name: "auth-bearer-token", JSONKey: "auth_bearer_token", Kind: "string", Required: false, Help: "auth bearer token"},
	{Name: "auth-method", JSONKey: "auth_method", Kind: "string", Required: false, Help: "auth method"},
	{Name: "auth-password", JSONKey: "auth_password", Kind: "string", Required: false, Help: "auth password"},
	{Name: "auth-username", JSONKey: "auth_username", Kind: "string", Required: false, Help: "auth username"},
	{Name: "content-change-enabled", JSONKey: "content_change_enabled", Kind: "bool", Required: false, Help: "alert when page content changes"},
	{Name: "content-change-severity", JSONKey: "content_change_severity", Kind: "string", Required: false, Help: "content change severity: notify, degraded, down"},
	{Name: "content-match-case-sensitive", JSONKey: "content_match_case_sensitive", Kind: "bool", Required: false, Help: "content match case sensitive"},
	{Name: "content-match-enabled", JSONKey: "content_match_enabled", Kind: "bool", Required: false, Help: "content match enabled"},
	{Name: "content-match-text", JSONKey: "content_match_text", Kind: "string", Required: false, Help: "content match text"},
	{Name: "content-match-type", JSONKey: "content_match_type", Kind: "string", Required: false, Help: "content match type"},
	{Name: "cron-expression", JSONKey: "cron_expression", Kind: "string", Required: false, Help: "cron expression"},
	{Name: "cron-timezone", JSONKey: "cron_timezone", Kind: "string", Required: false, Help: "cron timezone"},
	{Name: "db-expected-result", JSONKey: "db_expected_result", Kind: "string", Required: false, Help: "db expected result"},
	{Name: "db-host", JSONKey: "db_host", Kind: "string", Required: false, Help: "db host"},
	{Name: "db-name", JSONKey: "db_name", Kind: "string", Required: false, Help: "db name"},
	{Name: "db-password", JSONKey: "db_password", Kind: "string", Required: false, Help: "db password"},
	{Name: "db-port", JSONKey: "db_port", Kind: "int", Required: false, Help: "db port"},
	{Name: "db-query", JSONKey: "db_query", Kind: "string", Required: false, Help: "db query"},
	{Name: "db-ssl-enabled", JSONKey: "db_ssl_enabled", Kind: "bool", Required: false, Help: "db ssl enabled"},
	{Name: "db-type", JSONKey: "db_type", Kind: "string", Required: false, Help: "db type"},
	{Name: "db-username", JSONKey: "db_username", Kind: "string", Required: false, Help: "db username"},
	{Name: "description", JSONKey: "description", Kind: "string", Required: false, Help: "description"},
	{Name: "dns-dkim-selector", JSONKey: "dns_dkim_selector", Kind: "string", Required: false, Help: "dns dkim selector"},
	{Name: "dns-expected-value", JSONKey: "dns_expected_value", Kind: "string", Required: false, Help: "dns expected value"},
	{Name: "dns-hijack-alert-channel-ids", JSONKey: "dns_hijack_alert_channel_ids", Kind: "string", Required: false, Help: "dns hijack alert channel ids"},
	{Name: "dns-hijack-alert-enabled", JSONKey: "dns_hijack_alert_enabled", Kind: "bool", Required: false, Help: "dns hijack alert enabled"},
	{Name: "dns-multi-record-enabled", JSONKey: "dns_multi_record_enabled", Kind: "bool", Required: false, Help: "dns multi record enabled"},
	{Name: "dns-record-type", JSONKey: "dns_record_type", Kind: "string", Required: false, Help: "dns record type"},
	{Name: "dns-server", JSONKey: "dns_server", Kind: "string", Required: false, Help: "dns server"},
	{Name: "dns-soa-alert-on-change", JSONKey: "dns_soa_alert_on_change", Kind: "bool", Required: false, Help: "dns soa alert on change"},
	{Name: "dns-txt-monitoring-enabled", JSONKey: "dns_txt_monitoring_enabled", Kind: "bool", Required: false, Help: "dns txt monitoring enabled"},
	{Name: "flap-detection-enabled", JSONKey: "flap_detection_enabled", Kind: "bool", Required: false, Help: "flap detection enabled"},
	{Name: "flap-threshold", JSONKey: "flap_threshold", Kind: "int", Required: false, Help: "flap threshold"},
	{Name: "flap-window-minutes", JSONKey: "flap_window_minutes", Kind: "int", Required: false, Help: "flap window minutes"},
	{Name: "ftp-passive", JSONKey: "ftp_passive", Kind: "bool", Required: false, Help: "ftp passive"},
	{Name: "ftp-password", JSONKey: "ftp_password", Kind: "string", Required: false, Help: "ftp password"},
	{Name: "ftp-path", JSONKey: "ftp_path", Kind: "string", Required: false, Help: "ftp path"},
	{Name: "ftp-protocol", JSONKey: "ftp_protocol", Kind: "string", Required: false, Help: "ftp protocol"},
	{Name: "ftp-username", JSONKey: "ftp_username", Kind: "string", Required: false, Help: "ftp username"},
	{Name: "geo-content-consistency-enabled", JSONKey: "geo_content_consistency_enabled", Kind: "bool", Required: false, Help: "alert when one location serves different content than its peers"},
	{Name: "geo-monitoring-enabled", JSONKey: "geo_monitoring_enabled", Kind: "bool", Required: false, Help: "geo monitoring enabled"},
	{Name: "grace-period", JSONKey: "grace_period", Kind: "int", Required: false, Help: "grace period"},
	{Name: "host", JSONKey: "host", Kind: "string", Required: false, Help: "host"},
	{Name: "http-body", JSONKey: "http_body", Kind: "string", Required: false, Help: "http body"},
	{Name: "http-body-type", JSONKey: "http_body_type", Kind: "string", Required: false, Help: "http body type"},
	{Name: "http-follow-redirects", JSONKey: "http_follow_redirects", Kind: "bool", Required: false, Help: "http follow redirects"},
	{Name: "http-form-check-after-login-url", JSONKey: "http_form_check_after_login_url", Kind: "string", Required: false, Help: "http form check after login url"},
	{Name: "http-form-login-enabled", JSONKey: "http_form_login_enabled", Kind: "bool", Required: false, Help: "http form login enabled"},
	{Name: "http-form-login-success-text", JSONKey: "http_form_login_success_text", Kind: "string", Required: false, Help: "http form login success text"},
	{Name: "http-form-login-url", JSONKey: "http_form_login_url", Kind: "string", Required: false, Help: "http form login url"},
	{Name: "http-method", JSONKey: "http_method", Kind: "string", Required: false, Help: "http method"},
	{Name: "interval", JSONKey: "interval", Kind: "int", Required: false, Help: "interval"},
	{Name: "is-active", JSONKey: "is_active", Kind: "bool", Required: false, Help: "is active"},
	{Name: "mail-blacklist-servers", JSONKey: "mail_blacklist_servers", Kind: "string", Required: false, Help: "mail blacklist servers"},
	{Name: "mail-check-blacklist", JSONKey: "mail_check_blacklist", Kind: "bool", Required: false, Help: "mail check blacklist"},
	{Name: "mail-check-dkim", JSONKey: "mail_check_dkim", Kind: "bool", Required: false, Help: "mail check dkim"},
	{Name: "mail-check-dmarc", JSONKey: "mail_check_dmarc", Kind: "bool", Required: false, Help: "mail check dmarc"},
	{Name: "mail-check-ptr", JSONKey: "mail_check_ptr", Kind: "bool", Required: false, Help: "mail check ptr"},
	{Name: "mail-check-spf", JSONKey: "mail_check_spf", Kind: "bool", Required: false, Help: "mail check spf"},
	{Name: "mail-dkim-selectors", JSONKey: "mail_dkim_selectors", Kind: "string", Required: false, Help: "mail dkim selectors"},
	{Name: "mail-domain", JSONKey: "mail_domain", Kind: "string", Required: false, Help: "mail domain"},
	{Name: "mail-imap-enabled", JSONKey: "mail_imap_enabled", Kind: "bool", Required: false, Help: "mail imap enabled"},
	{Name: "mail-imap-port", JSONKey: "mail_imap_port", Kind: "int", Required: false, Help: "mail imap port"},
	{Name: "mail-imap-ssl", JSONKey: "mail_imap_ssl", Kind: "bool", Required: false, Help: "mail imap ssl"},
	{Name: "mail-pop3-enabled", JSONKey: "mail_pop3_enabled", Kind: "bool", Required: false, Help: "mail pop3 enabled"},
	{Name: "mail-pop3-port", JSONKey: "mail_pop3_port", Kind: "int", Required: false, Help: "mail pop3 port"},
	{Name: "mail-pop3-ssl", JSONKey: "mail_pop3_ssl", Kind: "bool", Required: false, Help: "mail pop3 ssl"},
	{Name: "mail-smtp-enabled", JSONKey: "mail_smtp_enabled", Kind: "bool", Required: false, Help: "mail smtp enabled"},
	{Name: "mail-smtp-open-relay", JSONKey: "mail_smtp_open_relay", Kind: "bool", Required: false, Help: "mail smtp open relay"},
	{Name: "mail-smtp-port", JSONKey: "mail_smtp_port", Kind: "int", Required: false, Help: "mail smtp port"},
	{Name: "mail-smtp-starttls", JSONKey: "mail_smtp_starttls", Kind: "bool", Required: false, Help: "mail smtp starttls"},
	{Name: "name", JSONKey: "name", Kind: "string", Required: true, Help: "name"},
	{Name: "owner-user-id", JSONKey: "owner_user_id", Kind: "int", Required: false, Help: "owner user id"},
	{Name: "pagespeed-budget-load-ms", JSONKey: "pagespeed_budget_load_ms", Kind: "int", Required: false, Help: "pagespeed budget load ms"},
	{Name: "pagespeed-budget-score", JSONKey: "pagespeed_budget_score", Kind: "int", Required: false, Help: "pagespeed budget score"},
	{Name: "pagespeed-budget-size-kb", JSONKey: "pagespeed_budget_size_kb", Kind: "int", Required: false, Help: "pagespeed budget size kb"},
	{Name: "pagespeed-enabled", JSONKey: "pagespeed_enabled", Kind: "bool", Required: false, Help: "pagespeed enabled"},
	{Name: "pagespeed-interval", JSONKey: "pagespeed_interval", Kind: "int", Required: false, Help: "pagespeed interval"},
	{Name: "ping-count", JSONKey: "ping_count", Kind: "int", Required: false, Help: "ping count"},
	{Name: "port", JSONKey: "port", Kind: "int", Required: false, Help: "port"},
	{Name: "project-id", JSONKey: "project_id", Kind: "int", Required: true, Help: "project id"},
	{Name: "runbook-url", JSONKey: "runbook_url", Kind: "string", Required: false, Help: "runbook url"},
	{Name: "schedule-type", JSONKey: "schedule_type", Kind: "string", Required: false, Help: "schedule type"},
	{Name: "ssl-verify", JSONKey: "ssl_verify", Kind: "bool", Required: false, Help: "ssl verify"},
	{Name: "timeout", JSONKey: "timeout", Kind: "int", Required: false, Help: "timeout"},
	{Name: "traceroute-on-timeout", JSONKey: "traceroute_on_timeout", Kind: "bool", Required: false, Help: "traceroute on timeout"},
	{Name: "type", JSONKey: "type", Kind: "string", Required: false, Help: "type"},
	{Name: "url", JSONKey: "url", Kind: "string", Required: false, Help: "url"},
}

func init() {
	register(&Resource{
		Name: "check", Cols: checkListCols, Org: OrgParam, Fields: checkFields,
		listFn: func(env *cmdCtx, orgID *int) ([]map[string]any, error) {
			items, err := listChecksPaged(context.Background(), env, orgID)
			if err != nil {
				return nil, err
			}
			rows := make([]map[string]any, 0, len(items))
			for _, it := range items {
				rows = append(rows, map[string]any{"id": it.Id, "name": it.Name, "type": deref(it.Type), "status": it.Status})
			}
			return rows, nil
		},
		getFn: func(env *cmdCtx, _ *int, id int) (map[string]any, error) {
			c, err := env.SDK.Checks.GetCheckDetails(context.Background(), id)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		createFn: func(env *cmdCtx, _ *int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.CheckCreate](body)
			if err != nil {
				return nil, err
			}
			c, err := env.SDK.Checks.CreateNewCheck(context.Background(), b)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		updateFn: func(env *cmdCtx, _ *int, id int, body map[string]any) (map[string]any, error) {
			b, err := mapToBody[models.CheckUpdate](body)
			if err != nil {
				return nil, err
			}
			c, err := env.SDK.Checks.UpdateCheck(context.Background(), id, b)
			if err != nil {
				return nil, err
			}
			return toMap(c)
		},
		deleteFn: func(env *cmdCtx, _ *int, id int) error {
			_, err := env.SDK.Checks.RemoveCheck(context.Background(), id)
			return err
		},
	})
}

// newCheckCmd builds the `check` command from the registry (list/create/
// update/delete), swaps in the id-or-name-aware `get` that predates the
// generic factory, and adds the non-CRUD verbs (run/pause/resume/test-alert)
// the registry has no notion of.
func newCheckCmd() *cobra.Command {
	c := newResourceCmd(registry["check"])
	swapGetCmd(c, newCheckGetCmd())
	c.AddCommand(
		newCheckRunCmd(),
		newCheckPauseCmd(),
		newCheckResumeCmd(),
		newCheckTestAlertCmd(),
	)
	return c
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
			return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: checkGetCols, Rows: []output.Row{row}})
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
// checkListItems, resolving org scope from the --org global flag/context.
// Used by resolveCheckID and the bespoke `get`/`run`/`pause`/`resume`/
// `test-alert` commands, which all take *cobra.Command directly rather than
// going through the registry.
func listChecks(cmd *cobra.Command, env *cmdCtx) ([]checkListItem, error) {
	orgID, err := optionalOrgID(cmd, env)
	if err != nil {
		return nil, err
	}
	return listChecksPaged(cmd.Context(), env, orgID)
}

// listChecksPaged is listChecks' paging loop, factored out so the registry's
// check listFn (which gets orgID already resolved by resolveResourceOrg, and
// no *cobra.Command to pull a context from) can drive it directly.
func listChecksPaged(ctx context.Context, env *cmdCtx, orgID *int) ([]checkListItem, error) {
	var all []checkListItem
	for page := 0; page < checkListMaxPages; page++ {
		offset := page * checkListPageSize
		raw, err := env.SDK.Checks.ListChecks(ctx, &models.ListChecksParams{
			OrgId:  orgID,
			Limit:  intPtr(checkListPageSize),
			Offset: intPtr(offset),
		})
		if err != nil {
			return nil, clierr.Config("listing checks: %v", err)
		}
		var items []checkListItem
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, clierr.Config("decoding checks list: %v", err)
		}
		all = append(all, items...)
		if len(items) < checkListPageSize {
			return all, nil
		}
	}
	return all, nil
}

// boolPtr returns a pointer to b, for the SDK's optional *bool fields.
func boolPtr(b bool) *bool { return &b }
