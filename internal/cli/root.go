package cli

import "github.com/spf13/cobra"

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
	root.AddCommand(newVersionCmd())
	return root
}

func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		return 2
	}
	return 0
}
