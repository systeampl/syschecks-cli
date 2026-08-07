package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/generate"
)

// generateProviderTF is the fixed content of every generated provider.tf: it
// never varies with the live account, so it's a constant rather than
// something rendered per run.
const generateProviderTF = `terraform {
  required_providers {
    systeam = {
      source  = "systeampl/systeam"
      version = "~> 0.2"
    }
  }
}

provider "systeam" {}
`

// newGenerateCmd is the `generate` command group's parent. It has one child
// today (`terraform`); grouping it this way means a future `generate
// ansible`/`generate pulumi` (see the providers-sdk-refactor plan) slots in
// as a sibling subcommand without renaming this one.
func newGenerateCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate infrastructure-as-code from live resources",
	}
	c.AddCommand(newGenerateTerraformCmd())
	return c
}

// generateRegistryName maps a `generate` resource kind (the keys
// internal/generate's provider-derived specs use: "check",
// "notification-channel", "team") to its v0.2 CLI registry Resource.Name.
// The two naming schemes agree for "check" and "team" but diverge for
// "notification-channel", whose registry entry is named "notification" (see
// notification.go) -- the HCL-facing resource kind and the CLI's own command
// name were never required to match.
func generateRegistryName(kind string) string {
	if kind == "notification-channel" {
		return "notification"
	}
	return kind
}

// generateFileName is the output .tf file a resource kind's rendered blocks
// are grouped into.
func generateFileName(kind string) string {
	switch kind {
	case "check":
		return "checks.tf"
	case "notification-channel":
		return "notification_channels.tf"
	case "team":
		return "teams.tf"
	default:
		return strings.ReplaceAll(kind, "-", "_") + ".tf"
	}
}

// newGenerateTerraformCmd builds `generate terraform`: it only ever reads
// (listFn/getFn) from the registry -- never create/update/delete -- so
// running it against a live account is always safe.
func newGenerateTerraformCmd() *cobra.Command {
	var (
		outDir     string
		typesFlag  string
		checksFlag string
		projectID  int
	)
	c := &cobra.Command{
		Use:   "terraform",
		Short: "Generate Terraform HCL + import blocks for live resources (read-only)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if outDir == "" {
				return clierr.Config("generate terraform: --out <dir> is required")
			}
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			orgID, err := resolveOrgID(cmd, env)
			if err != nil {
				return err
			}
			kinds, err := generateKinds(typesFlag)
			if err != nil {
				return err
			}
			var checkIDs map[int]bool
			if checksFlag != "" {
				checkIDs, err = parseIDSet(checksFlag)
				if err != nil {
					return clierr.Config("generate terraform: invalid --check list: %v", err)
				}
			}
			// --project only ever filters checks (the only in-scope kind
			// that carries a project_id); for notification-channel/team it
			// is a documented no-op, matching --check.
			projectSet := cmd.Flags().Changed("project")

			labels := generate.NewLabelSet()
			fileBlocks := map[string][]string{}
			var importLines []string
			varDecls := map[string]bool{}

			for _, kind := range kinds {
				r, ok := registry[generateRegistryName(kind)]
				if !ok || r.listFn == nil {
					continue
				}
				items, err := r.listFn(env, &orgID)
				if err != nil {
					return clierr.Config("generate terraform: listing %s: %v", kind, err)
				}

				ids := make([]int, 0, len(items))
				for _, it := range items {
					id, ok := intFromAny(it["id"])
					if !ok {
						continue
					}
					if kind == "check" && checkIDs != nil && !checkIDs[id] {
						continue
					}
					ids = append(ids, id)
				}
				sort.Ints(ids)

				tfType, _ := generate.TFType(kind)

				for _, id := range ids {
					full, err := r.getFn(env, &orgID, id)
					if err != nil {
						return clierr.Config("generate terraform: getting %s %d: %v", kind, id, err)
					}
					if kind == "check" && projectSet {
						pid, ok := intFromAny(full["project_id"])
						if !ok || pid != projectID {
							continue
						}
					}

					label := labels.Unique(generate.HCLLabel(generateLabelBase(full, id)))
					block, vars, err := generate.RenderResource(kind, id, label, full)
					if err != nil {
						return clierr.Config("generate terraform: rendering %s %d: %v", kind, id, err)
					}
					fileBlocks[kind] = append(fileBlocks[kind], block)
					importLines = append(importLines, generate.RenderImport(tfType, label, id))
					for _, v := range vars {
						varDecls[v] = true
					}
				}
			}

			return writeGenerateOutput(cmd, outDir, kinds, fileBlocks, importLines, varDecls)
		},
	}
	c.Flags().StringVar(&outDir, "out", "", "output directory for generated .tf files (required; -o is taken by the global --output flag)")
	c.Flags().StringVar(&typesFlag, "type", "", "comma-separated resource types to generate: check,notification-channel,team (default: all)")
	c.Flags().StringVar(&checksFlag, "check", "", "comma-separated check ids to limit check generation to (no-op for other --type values)")
	c.Flags().IntVar(&projectID, "project", 0, "limit checks to this project id (no-op for other --type values)")
	return c
}

// writeGenerateOutput creates outDir and writes provider.tf, one .tf file
// per in-scope kind that produced at least one block, imports.tf (only if
// there's at least one resource at all), and variables.tf (only if any
// secret attribute was redacted into a variable) -- printing a stderr
// warning naming those variables so the caller knows to set them before
// `terraform apply`.
func writeGenerateOutput(cmd *cobra.Command, outDir string, kinds []string, fileBlocks map[string][]string, importLines []string, varDecls map[string]bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return clierr.Config("generate terraform: creating output directory %s: %v", outDir, err)
	}
	if err := writeGenerateFile(outDir, "provider.tf", generateProviderTF); err != nil {
		return err
	}
	for _, kind := range kinds {
		blocks := fileBlocks[kind]
		if len(blocks) == 0 {
			continue
		}
		if err := writeGenerateFile(outDir, generateFileName(kind), strings.Join(blocks, "\n\n")+"\n"); err != nil {
			return err
		}
	}
	if len(importLines) > 0 {
		if err := writeGenerateFile(outDir, "imports.tf", strings.Join(importLines, "\n\n")+"\n"); err != nil {
			return err
		}
	}
	if len(varDecls) == 0 {
		cmd.Printf("generated Terraform configuration in %s\n", outDir)
		return nil
	}

	decls := make([]string, 0, len(varDecls))
	for d := range varDecls {
		decls = append(decls, d)
	}
	sort.Strings(decls) // every decl starts `variable "<name>" {...`, so string sort == sort by name
	if err := writeGenerateFile(outDir, "variables.tf", strings.Join(decls, "\n\n")+"\n"); err != nil {
		return err
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: the generated configuration references sensitive variables that are never written to disk. Set each one (e.g. via TF_VAR_<name>) before running terraform:")
	for _, d := range decls {
		fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", generateVariableName(d))
	}
	cmd.Printf("generated Terraform configuration in %s\n", outDir)
	return nil
}

func writeGenerateFile(dir, name, content string) error {
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		return clierr.Config("generate terraform: writing %s: %v", name, err)
	}
	return nil
}

// generateLabelBase picks the human-readable seed for a resource's HCL
// label: its name, else its slug, else a synthetic "r_<id>" -- hclLabel then
// sanitizes whichever is chosen into a valid identifier.
func generateLabelBase(attrs map[string]any, id int) string {
	if name, ok := attrs["name"].(string); ok && name != "" {
		return name
	}
	if slug, ok := attrs["slug"].(string); ok && slug != "" {
		return slug
	}
	return "r_" + strconv.Itoa(id)
}

// generateVariableName extracts the variable name from a rendered
// `variable "<name>" { ... }` declaration, for the stderr warning list.
func generateVariableName(decl string) string {
	const prefix = `variable "`
	i := strings.Index(decl, prefix)
	if i < 0 {
		return decl
	}
	rest := decl[i+len(prefix):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return decl
	}
	return rest[:j]
}

// intFromAny normalizes a JSON-decoded numeric value (float64 from
// map[string]any decoding, or a plain int when a listFn built its rows by
// hand rather than round-tripping through JSON) to an int.
func intFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// parseIDSet parses a comma-separated list of integer ids (as taken by
// --check) into a set.
func parseIDSet(s string) (map[int]bool, error) {
	out := map[int]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q: %w", part, err)
		}
		out[id] = true
	}
	return out, nil
}

// generateKinds resolves the --type flag (a comma-separated subset of
// generate.ResourceKinds, or empty for all of them) into the ordered list of
// kinds in scope for this run. An unrecognized type name is a config error
// (exit 2) rather than a silent no-op.
func generateKinds(typesFlag string) ([]string, error) {
	if typesFlag == "" {
		return append([]string(nil), generate.ResourceKinds...), nil
	}
	want := map[string]bool{}
	for _, t := range strings.Split(typesFlag, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		want[t] = true
	}
	var kinds []string
	for _, k := range generate.ResourceKinds {
		if want[k] {
			kinds = append(kinds, k)
			delete(want, k)
		}
	}
	if len(want) > 0 {
		unknown := make([]string, 0, len(want))
		for k := range want {
			unknown = append(unknown, k)
		}
		sort.Strings(unknown)
		return nil, clierr.Config("generate terraform: unknown --type value(s): %s (want %s)",
			strings.Join(unknown, ", "), strings.Join(generate.ResourceKinds, ","))
	}
	return kinds, nil
}
