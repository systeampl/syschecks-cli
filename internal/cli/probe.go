package cli

import (
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
	"github.com/systeampl/syschecks-cli/internal/probe"
	"github.com/systeampl/syschecks-go/models"
)

// probeHTTPCols/probeDNSCols/probeTLSCols are the column sets for `probe
// http`, `probe dns`, and `probe tls` respectively.
var (
	probeHTTPCols = []string{"status", "dns", "connect", "tls", "ttfb", "total", "cert_expiry"}
	probeDNSCols  = []string{"addr", "duration"}
	probeTLSCols  = []string{"version", "cipher", "cert_expiry", "dns_names"}
)

// newProbeCmd groups client-side diagnostics: http, dns, tls. Unlike other
// command groups these talk directly to the target (net/http, crypto/tls,
// net) rather than through the SDK, so plain probes work without any
// configured context or token. `probe http --save` is the one exception: it
// uses the SDK to persist the probed URL as a monitored check, and needs
// config resolved via cmdEnv for that.
func newProbeCmd() *cobra.Command {
	c := &cobra.Command{Use: "probe", Short: "Client-side HTTP/DNS/TLS diagnostics"}
	c.AddCommand(newProbeHTTPCmd(), newProbeDNSCmd(), newProbeTLSCmd())
	return c
}

func newProbeHTTPCmd() *cobra.Command {
	var save bool
	var project int
	var interval time.Duration
	c := &cobra.Command{
		Use:   "http <url>",
		Short: "Measure a single GET's DNS/connect/TLS/TTFB timings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			r, err := probe.HTTP(cmd.Context(), target)
			if err != nil {
				return clierr.Config("probing %s: %v", target, err)
			}

			gf := globalsFrom(cmd)
			row := output.Row{
				"status":      r.Status,
				"dns":         r.DNS.String(),
				"connect":     r.Connect.String(),
				"tls":         r.TLS.String(),
				"ttfb":        r.TTFB.String(),
				"total":       r.Total.String(),
				"cert_expiry": certExpiryString(r.CertExpiry),
			}
			if err := output.Render(cmd.OutOrStdout(), gf.Output, gf.Quiet, output.Table{Cols: probeHTTPCols, Rows: []output.Row{row}}); err != nil {
				return err
			}
			if !save {
				return nil
			}

			if project <= 0 {
				return clierr.Config("--save requires --project <id>")
			}
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			chk, err := env.SDK.Checks.CreateNewCheck(cmd.Context(), models.CreateNewCheckJSONRequestBody{
				Name:      checkNameFromURL(target),
				ProjectId: project,
				Type:      strptr("uptime"),
				Url:       &target,
				Interval:  intptr(int(interval.Seconds())),
			})
			if err != nil {
				return clierr.Config("saving check for %s: %v", target, err)
			}
			cmd.Printf("created check %d (%s)\n", chk.Id, chk.Name)
			return nil
		},
	}
	c.Flags().BoolVar(&save, "save", false, "create a check from this probe")
	c.Flags().IntVar(&project, "project", 0, "project id to create the check in (required with --save)")
	c.Flags().DurationVar(&interval, "interval", 60*time.Second, "check interval, used with --save")
	return c
}

func newProbeDNSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dns <host>",
		Short: "Resolve a hostname and time the lookup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := probe.DNS(args[0])
			if err != nil {
				return clierr.Config("resolving %s: %v", args[0], err)
			}
			var rows []output.Row
			for _, a := range r.Addrs {
				rows = append(rows, output.Row{"addr": a, "duration": r.Duration.String()})
			}
			gf := globalsFrom(cmd)
			return output.Render(cmd.OutOrStdout(), gf.Output, gf.Quiet, output.Table{Cols: probeDNSCols, Rows: rows})
		},
	}
}

func newProbeTLSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tls <host:port>",
		Short: "Perform a raw TLS handshake and report cert/cipher info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := probe.TLS(args[0])
			if err != nil {
				return clierr.Config("TLS handshake with %s: %v", args[0], err)
			}
			row := output.Row{
				"version":     r.Version,
				"cipher":      r.Cipher,
				"cert_expiry": r.CertExpiry.Format(time.RFC3339),
				"dns_names":   strings.Join(r.DNSNames, ","),
			}
			gf := globalsFrom(cmd)
			return output.Render(cmd.OutOrStdout(), gf.Output, gf.Quiet, output.Table{Cols: probeTLSCols, Rows: []output.Row{row}})
		},
	}
}

// certExpiryString formats an optional cert expiry for output: RFC3339 when
// present, "" for plain http probes where no TLS handshake happened.
func certExpiryString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// checkNameFromURL derives a check name from rawURL's host, falling back to
// the raw URL string if it doesn't parse or has no host.
func checkNameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Host
}

// strptr returns a pointer to s, for the SDK's optional *string fields.
func strptr(s string) *string { return &s }

// intptr returns a pointer to i, for the SDK's optional *int fields.
func intptr(i int) *int { return &i }
