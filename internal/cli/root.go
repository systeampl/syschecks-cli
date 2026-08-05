package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
)

// GlobalFlags holds the persistent flags every command shares.
type GlobalFlags struct {
	Output  string
	Quiet   bool
	NoColor bool
	Context string
	Org     string
	APIURL  string
	Token   string
	Verbose bool
}

func NewRootCmd() *cobra.Command {
	gf := &GlobalFlags{}
	root := &cobra.Command{
		Use:           "syschecks",
		Short:         "Operational CLI for SysChecks monitoring",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&gf.Output, "output", "o", "table", "output format: table|json|yaml")
	pf.BoolVarP(&gf.Quiet, "quiet", "q", false, "only essential output (ids)")
	pf.BoolVar(&gf.NoColor, "no-color", false, "disable ANSI color")
	pf.StringVar(&gf.Context, "context", "", "config context to use")
	pf.StringVar(&gf.Org, "org", "", "organization id or slug")
	pf.StringVar(&gf.APIURL, "api-url", "", "API base URL")
	pf.StringVar(&gf.Token, "token", "", "PAT token (prefer SYSCHECKS_TOKEN env)")
	pf.BoolVar(&gf.Verbose, "verbose", false, "verbose logging (token redacted)")
	root.SetContext(withGlobals(root.Context(), gf))
	root.AddCommand(newVersionCmd(), newAuthCmd(), newConfigCmd(), newOrgCmd(), newProjectCmd(), newCheckCmd(),
		newIncidentCmd(), newAgentCmd(), newNotificationCmd(), newProbeCmd(), newVerifyCmd(), newApplyCmd(),
		newResourceCmd(registry["team"]), newResourceCmd(registry["service"]))
	return root
}

// Execute builds the root command, runs it, and returns the process exit
// code. It is the sole entry point main.go calls.
func Execute() int {
	return execute(NewRootCmd())
}

// execute runs root and, on failure, writes the error to root's stderr
// writer before mapping it to an exit code. Split out from Execute so tests
// can inject a captured Err writer via root.SetErr and assert on it directly
// — root.Execute() alone (SilenceErrors: true) never prints anything, which
// is exactly the bug this function fixes: a failing command must not exit
// with a bare code and empty stderr (design spec §7).
func execute(root *cobra.Command) int {
	err := root.Execute()
	if err != nil {
		printErr(root, err)
	}
	return clierr.Code(err)
}

// printErr writes err to cmd's stderr writer (os.Stderr in production,
// unless overridden via SetErr): a `{"error":"..."}` line when the
// --output global flag is "json", else a plain "Error: ..." line.
func printErr(cmd *cobra.Command, err error) {
	w := cmd.ErrOrStderr()
	if globalsFrom(cmd).Output == "json" {
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Fprintf(w, "Error: %v\n", err)
}
