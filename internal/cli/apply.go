package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systeampl/syschecks-cli/internal/clierr"
	yaml "sigs.k8s.io/yaml"
)

// bodyFromFlagsAndFile builds a request body from an optional -f document
// overlaid by any changed field flags: the file supplies a base
// map[string]any (via sigs.k8s.io/yaml, which converts YAML to JSON first so
// numbers/bools land as the same Go types the SDK models expect), then each
// Field whose flag was Changed replaces that key — flags win over the file.
// enforceRequired gates the missing-required-field check (on for create, off
// for update); a Required field satisfied only by the file, with no matching
// flag set, still counts as satisfied. This supersedes Task 1's
// flags-only bodyFromFlags.
func bodyFromFlagsAndFile(cmd *cobra.Command, fields []Field, fv *fieldValues, file string, enforceRequired bool) (map[string]any, error) {
	body := map[string]any{}
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, clierr.Config("reading -f %s: %v", file, err)
		}
		if err := yaml.Unmarshal(b, &body); err != nil {
			return nil, clierr.Config("parsing -f %s: %v", file, err)
		}
		if body == nil {
			body = map[string]any{}
		}
	}
	for _, f := range fields {
		if !cmd.Flags().Changed(f.Name) {
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
	if enforceRequired {
		for _, f := range fields {
			if !f.Required {
				continue
			}
			if _, ok := body[f.JSONKey]; !ok {
				return nil, clierr.Config("missing required flag --%s (and no -f value for %q)", f.Name, f.JSONKey)
			}
		}
	}
	return body, nil
}

// newApplyCmd builds the `apply -f <file>` command: it splits file into one
// or more YAML documents (separated by a line that is exactly "---", the
// standard YAML document separator), and for each non-empty document reads
// its "kind" field, looks up that name in the resource registry, and
// dispatches to the resource's updateFn (when the document carries an "id")
// or createFn (otherwise). Org scope for each document is resolved from the
// dispatched resource's own OrgMode, exactly like the generic create/update
// leaves in crud.go.
func newApplyCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "apply",
		Short: "Create or update resources from a YAML/JSON file (kind: <resource> per document)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			b, err := os.ReadFile(file)
			if err != nil {
				return clierr.Config("reading -f %s: %v", file, err)
			}
			env, err := cmdEnv(cmd)
			if err != nil {
				return err
			}
			for i, doc := range splitYAMLDocs(b) {
				if len(bytes.TrimSpace(doc)) == 0 {
					continue
				}
				item, err := applyDoc(cmd, env, doc)
				if err != nil {
					return clierr.Config("document %d: %v", i+1, err)
				}
				cmd.Printf("%v\n", item["id"])
			}
			return nil
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "", "path to a YAML/JSON file with one or more documents to apply")
	_ = c.MarkFlagRequired("file")
	return c
}

// splitYAMLDocs splits b into documents on any line that is exactly "---"
// (surrounding whitespace ignored), the standard YAML document separator.
// A file with no separator at all yields a single document (the whole file).
func splitYAMLDocs(b []byte) [][]byte {
	var docs [][]byte
	var cur bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			docs = append(docs, append([]byte(nil), cur.Bytes()...))
			cur.Reset()
			continue
		}
		cur.WriteString(line)
		cur.WriteByte('\n')
	}
	docs = append(docs, append([]byte(nil), cur.Bytes()...))
	return docs
}

// applyDoc parses one document, resolves its resource by "kind", and
// dispatches to createFn/updateFn depending on whether "id" is present. It
// returns whatever the dispatched call returns, for the caller to report.
func applyDoc(cmd *cobra.Command, env *cmdCtx, doc []byte) (map[string]any, error) {
	var m map[string]any
	if err := yaml.Unmarshal(doc, &m); err != nil {
		return nil, fmt.Errorf("parsing document: %w", err)
	}
	kind, _ := m["kind"].(string)
	if kind == "" {
		return nil, clierr.Config("document missing required \"kind\" field")
	}
	r, ok := registry[kind]
	if !ok {
		return nil, clierr.Config("unknown kind %q", kind)
	}
	orgID, err := resolveResourceOrg(cmd, env, r.Org)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	for k, v := range m {
		if k == "kind" || k == "id" {
			continue
		}
		body[k] = v
	}
	if idRaw, present := m["id"]; present && idRaw != nil {
		id, err := toIntID(idRaw)
		if err != nil {
			return nil, clierr.Config("invalid id %v: %v", idRaw, err)
		}
		if r.updateFn == nil {
			return nil, clierr.Config("%s does not support update", kind)
		}
		return r.updateFn(env, orgID, id, body)
	}
	if r.createFn == nil {
		return nil, clierr.Config("%s does not support create", kind)
	}
	return r.createFn(env, orgID, body)
}

// toIntID normalizes a document's "id" value (a float64 after the YAML->JSON
// conversion for a bare number, or a string) to an int.
func toIntID(v any) (int, error) {
	switch t := v.(type) {
	case float64:
		return int(t), nil
	case int:
		return t, nil
	case string:
		return strconv.Atoi(t)
	default:
		return 0, fmt.Errorf("unsupported id type %T", v)
	}
}
