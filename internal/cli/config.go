package cli

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/config"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// configCols is the column set for `config get-contexts`. "name" is first so
// --quiet (which prints only the first column) yields plain context names.
var configCols = []string{"name", "current", "api_url", "organization"}

// newConfigCmd groups context management: set-context, use-context,
// get-contexts, current-context. Unlike most command groups here, these
// never touch the SDK client — they only read/write config.yaml.
func newConfigCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Manage CLI contexts (config.yaml)"}
	c.AddCommand(newSetContextCmd(), newUseContextCmd(), newGetContextsCmd(), newCurrentContextCmd())
	return c
}

func newSetContextCmd() *cobra.Command {
	var apiURL, org string
	cmd := &cobra.Command{
		Use:   "set-context <name>",
		Short: "Create or update a named context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			f, err := config.Load(config.Dir())
			if err != nil {
				return clierr.Config("loading config: %v", err)
			}
			ctx := f.Contexts[name]
			if apiURL != "" {
				ctx.APIURL = apiURL
			}
			if org != "" {
				ctx.Organization = org
			}
			f.Contexts[name] = ctx
			if err := config.Save(config.Dir(), f); err != nil {
				return clierr.Config("saving config: %v", err)
			}
			cmd.Printf("Context %q set\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&apiURL, "api-url", "", "API base URL for this context")
	cmd.Flags().StringVar(&org, "org", "", "organization id or slug for this context")
	return cmd
}

func newUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context <name>",
		Short: "Set the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			f, err := config.Load(config.Dir())
			if err != nil {
				return clierr.Config("loading config: %v", err)
			}
			if _, ok := f.Contexts[name]; !ok {
				return clierr.Config("no such context %q", name)
			}
			f.CurrentContext = name
			if err := config.Save(config.Dir(), f); err != nil {
				return clierr.Config("saving config: %v", err)
			}
			cmd.Printf("Switched to context %q\n", name)
			return nil
		},
	}
}

func newGetContextsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-contexts",
		Short: "List configured contexts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gf := globalsFrom(cmd)
			f, err := config.Load(config.Dir())
			if err != nil {
				return clierr.Config("loading config: %v", err)
			}
			names := make([]string, 0, len(f.Contexts))
			for name := range f.Contexts {
				names = append(names, name)
			}
			sort.Strings(names)
			var rows []output.Row
			for _, name := range names {
				ctx := f.Contexts[name]
				current := ""
				if name == f.CurrentContext {
					current = "*"
				}
				rows = append(rows, output.Row{
					"name":         name,
					"current":      current,
					"api_url":      ctx.APIURL,
					"organization": ctx.Organization,
				})
			}
			return output.Render(cmd.OutOrStdout(), gf.Output, gf.Quiet, output.Table{Cols: configCols, Rows: rows})
		},
	}
}

func newCurrentContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current-context",
		Short: "Print the active context name",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.Load(config.Dir())
			if err != nil {
				return clierr.Config("loading config: %v", err)
			}
			if f.CurrentContext == "" {
				return clierr.Config("current-context is not set")
			}
			cmd.Println(f.CurrentContext)
			return nil
		},
	}
}
