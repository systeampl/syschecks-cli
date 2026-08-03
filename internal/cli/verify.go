package cli

import (
	"encoding/json"
	"net/http"

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
// Exit-code contract: a status or assertion mismatch is a CI-meaningful
// failure -> clierr.Fail (exit 1). A bad flag or a gojq parse/eval error is a
// usage/config problem, not something the target did wrong -> clierr.Config
// (exit 2).
func newVerifyCmd() *cobra.Command {
	var target string
	var expectStatus int
	var expectJSON string

	c := &cobra.Command{
		Use:   "verify",
		Short: "HTTP GET a URL and assert status (and optionally a gojq expression) for CI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if target == "" {
				return clierr.Config("--url is required")
			}

			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, target, nil)
			if err != nil {
				return clierr.Config("building request for %s: %v", target, err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return clierr.Fail("requesting %s: %v", target, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != expectStatus {
				return clierr.Fail("expected status %d, got %d", expectStatus, resp.StatusCode)
			}

			if expectJSON != "" {
				var decoded any
				if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
					return clierr.Config("decoding response body as JSON: %v", err)
				}

				query, err := gojq.Parse(expectJSON)
				if err != nil {
					return clierr.Config("invalid --expect-json: %v", err)
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
	return c
}
