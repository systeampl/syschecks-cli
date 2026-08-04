package cli

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
)

// newVerifyCmd is the CI assertion command: GET --url, check the response
// status against --expect-status, and optionally assert a gojq expression
// over the decoded JSON body. It is deliberately client-side (like probe):
// it hits an arbitrary target, not the SysChecks API, so it needs no
// configured context or token.
//
// Exit-code contract: anything that's the TARGET's fault -- wrong status,
// a response body that isn't valid JSON, an --expect-json expression that
// evaluates to non-true -- is a CI-meaningful assertion failure ->
// clierr.Fail (exit 1). Anything that's the USER's fault -- a missing/bad
// flag, or an --expect-json EXPRESSION that fails to parse or errors during
// evaluation -- is a usage/config problem -> clierr.Config (exit 2).
func newVerifyCmd() *cobra.Command {
	var target string
	var expectStatus int
	var expectJSON string
	var timeout time.Duration

	c := &cobra.Command{
		Use:   "verify",
		Short: "HTTP GET a URL and assert status (and optionally a gojq expression) for CI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if target == "" {
				return clierr.Config("--url is required")
			}

			var query *gojq.Query
			if expectJSON != "" {
				q, err := gojq.Parse(expectJSON)
				if err != nil {
					return clierr.Config("invalid --expect-json: %v", err)
				}
				query = q
			}

			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, target, nil)
			if err != nil {
				return clierr.Config("building request for %s: %v", target, err)
			}
			// A bounded client, not http.DefaultClient: a target that accepts the
			// connection and never answers would otherwise hang the CI job that
			// this command exists to fail.
			resp, err := (&http.Client{Timeout: timeout}).Do(req)
			if err != nil {
				return clierr.Fail("requesting %s: %v", target, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != expectStatus {
				return clierr.Fail("expected status %d, got %d", expectStatus, resp.StatusCode)
			}

			if query != nil {
				var decoded any
				if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
					return clierr.Fail("decoding response body as JSON: %v", err)
				}

				iter := query.Run(decoded)
				v, ok := iter.Next()
				if !ok {
					return clierr.Fail("expect-json produced no result")
				}
				if e, isErr := v.(error); isErr {
					return clierr.Config("expect-json eval: %v", e)
				}
				if b, _ := v.(bool); !b {
					return clierr.Fail("expect-json was not true (got %v)", v)
				}
			}

			cmd.Printf("OK: %s -> %d\n", target, resp.StatusCode)
			return nil
		},
	}

	c.Flags().StringVar(&target, "url", "", "URL to GET (required)")
	c.Flags().IntVar(&expectStatus, "expect-status", http.StatusOK, "expected HTTP status code")
	c.Flags().StringVar(&expectJSON, "expect-json", "", "gojq expression asserted true over the decoded JSON body")
	c.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "abort the request after this long")
	return c
}
