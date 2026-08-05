package cli

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// newResourceCmd turns a *Resource into its parent cobra command, adding a
// leaf subcommand for each verb the resource actually implements: a nil verb
// closure means that subcommand is not generated at all (e.g. no createFn ->
// no `create`), so there is never a leaf whose RunE would just fail.
func newResourceCmd(r *Resource) *cobra.Command {
	c := &cobra.Command{Use: r.Name, Aliases: r.Aliases, Short: strings.Title(r.Name) + " resources"} //nolint:staticcheck // simple ASCII resource names
	if r.listFn != nil {
		c.AddCommand(newListCmd(r))
	}
	if r.getFn != nil {
		c.AddCommand(newGetCmd(r))
	}
	if r.createFn != nil {
		c.AddCommand(newCreateCmd(r))
	}
	if r.updateFn != nil {
		c.AddCommand(newUpdateCmd(r))
	}
	if r.deleteFn != nil {
		c.AddCommand(newDeleteCmd(r))
	}
	return c
}

// resolveResourceOrg resolves the org scope for a resource's SDK calls per
// its OrgMode: OrgArg is a required path arg (resolveOrgID errors on an
// empty --org), OrgParam rides an optional query param (optionalOrgID, nil
// when --org is unset), OrgNone never takes one.
func resolveResourceOrg(cmd *cobra.Command, env *cmdCtx, mode OrgMode) (*int, error) {
	switch mode {
	case OrgArg:
		id, err := resolveOrgID(cmd, env)
		if err != nil {
			return nil, err
		}
		return &id, nil
	case OrgParam:
		return optionalOrgID(cmd, env)
	default:
		return nil, nil
	}
}

// renderMany renders a list of resource maps as a table using the resource's
// declared columns.
func renderMany(env *cmdCtx, cols []string, items []map[string]any) error {
	rows := make([]output.Row, 0, len(items))
	for _, m := range items {
		rows = append(rows, output.Row(m))
	}
	return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: cols, Rows: rows})
}

// renderOne renders a single resource map as a one-row table.
func renderOne(env *cmdCtx, cols []string, item map[string]any) error {
	return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: cols, Rows: []output.Row{output.Row(item)}})
}

func newListCmd(r *Resource) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List " + r.Name + " resources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveResourceOrg(cmd, env, r.Org)
			if err != nil {
				return err
			}
			items, err := r.listFn(env, orgID)
			if err != nil {
				return clierr.Config("listing %s: %v", r.Name, err)
			}
			return renderMany(env, r.Cols, items)
		},
	}
}

func newGetCmd(r *Resource) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show a single " + r.Name,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveResourceOrg(cmd, env, r.Org)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid %s id %q: %v", r.Name, args[0], err)
			}
			item, err := r.getFn(env, orgID, id)
			if err != nil {
				return clierr.Config("getting %s %d: %v", r.Name, id, err)
			}
			return renderOne(env, r.Cols, item)
		},
	}
}

// fieldValues holds the addressable storage cobra flags bind to, one map per
// Kind, keyed by Field.Name. Needed because cobra's *Var flag constructors
// take a pointer that must outlive flag parsing.
type fieldValues struct {
	strs  map[string]*string
	ints  map[string]*int
	bools map[string]*bool
}

func newFieldValues(fields []Field) *fieldValues {
	fv := &fieldValues{strs: map[string]*string{}, ints: map[string]*int{}, bools: map[string]*bool{}}
	for _, f := range fields {
		switch f.Kind {
		case "int":
			var v int
			fv.ints[f.Name] = &v
		case "bool":
			var v bool
			fv.bools[f.Name] = &v
		default: // "string" and anything unrecognized
			var v string
			fv.strs[f.Name] = &v
		}
	}
	return fv
}

// registerFieldFlags declares one cobra flag per Field on c, generated
// straight from Resource.Fields — the single source for both create/update
// flags and (from Task 2) the -f schema, so flags and schema never drift.
func registerFieldFlags(c *cobra.Command, fields []Field, fv *fieldValues) {
	for _, f := range fields {
		switch f.Kind {
		case "int":
			c.Flags().IntVar(fv.ints[f.Name], f.Name, 0, f.Help)
		case "bool":
			c.Flags().BoolVar(fv.bools[f.Name], f.Name, false, f.Help)
		default:
			c.Flags().StringVar(fv.strs[f.Name], f.Name, "", f.Help)
		}
	}
}

// bodyFromFlags builds a request body from the flags registered by
// registerFieldFlags: only flags the user actually set (Changed) are
// included, so update issues a partial body. enforceRequired gates the
// missing-required-flag check, on for create and off for update. Task 2
// replaces this with bodyFromFlagsAndFile, which overlays these same flags
// on top of a -f document; this flags-only version is Task 1's scope.
func bodyFromFlags(cmd *cobra.Command, fields []Field, fv *fieldValues, enforceRequired bool) (map[string]any, error) {
	body := map[string]any{}
	for _, f := range fields {
		if !cmd.Flags().Changed(f.Name) {
			if enforceRequired && f.Required {
				return nil, clierr.Config("missing required flag --%s", f.Name)
			}
			continue
		}
		switch f.Kind {
		case "int":
			body[f.JSONKey] = *fv.ints[f.Name]
		case "bool":
			body[f.JSONKey] = *fv.bools[f.Name]
		default:
			body[f.JSONKey] = *fv.strs[f.Name]
		}
	}
	return body, nil
}

func newCreateCmd(r *Resource) *cobra.Command {
	fv := newFieldValues(r.Fields)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a " + r.Name,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveResourceOrg(cmd, env, r.Org)
			if err != nil {
				return err
			}
			body, err := bodyFromFlags(cmd, r.Fields, fv, true)
			if err != nil {
				return err
			}
			item, err := r.createFn(env, orgID, body)
			if err != nil {
				return clierr.Config("creating %s: %v", r.Name, err)
			}
			return renderOne(env, r.Cols, item)
		},
	}
	registerFieldFlags(c, r.Fields, fv)
	return c
}

func newUpdateCmd(r *Resource) *cobra.Command {
	fv := newFieldValues(r.Fields)
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a " + r.Name,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveResourceOrg(cmd, env, r.Org)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid %s id %q: %v", r.Name, args[0], err)
			}
			body, err := bodyFromFlags(cmd, r.Fields, fv, false)
			if err != nil {
				return err
			}
			item, err := r.updateFn(env, orgID, id, body)
			if err != nil {
				return clierr.Config("updating %s %d: %v", r.Name, id, err)
			}
			return renderOne(env, r.Cols, item)
		},
	}
	registerFieldFlags(c, r.Fields, fv)
	return c
}

func newDeleteCmd(r *Resource) *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a " + r.Name,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveResourceOrg(cmd, env, r.Org)
			if err != nil {
				return err
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return clierr.Config("invalid %s id %q: %v", r.Name, args[0], err)
			}
			if !yes {
				if err := confirmDelete(cmd, r.Name, id); err != nil {
					return err
				}
			}
			if err := r.deleteFn(env, orgID, id); err != nil {
				return clierr.Config("deleting %s %d: %v", r.Name, id, err)
			}
			cmd.Printf("%s %d deleted\n", r.Name, id)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}

// confirmDelete prompts y/N on a TTY and returns an error unless the answer
// is affirmative. Without --yes and without a TTY to prompt on (piped/CI
// invocation), it refuses rather than silently deleting.
func confirmDelete(cmd *cobra.Command, name string, id int) error {
	in, ok := cmd.InOrStdin().(*os.File)
	if !ok || !isTerminalFile(in) {
		return clierr.Config("refusing to delete %s %d without --yes (no interactive terminal to confirm on)", name, id)
	}
	cmd.Printf("delete %s %d? [y/N]: ", name, id)
	line, _ := bufio.NewReader(in).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(line)); a == "y" || a == "yes" {
		return nil
	}
	return clierr.Config("aborted")
}

// isTerminalFile reports whether f is a character device (a terminal), the
// same check output.ShouldColor uses for its own TTY detection.
func isTerminalFile(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
