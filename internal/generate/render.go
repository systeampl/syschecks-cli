package generate

import (
	"fmt"
	"sort"
	"strings"
)

// renderResource renders one live resource's writable attributes as an HCL
// `resource` block, in the order defined by specs[cliName].Attrs.
//
// Only attributes present and non-nil in attrs are emitted — a spec attr
// that's absent or nil from the live payload is left out entirely rather
// than written as null, so the generated block matches live state and a
// `terraform plan` against it is a no-op. Read-only attributes (id, status,
// created_at, ...) are never emitted because they're simply not part of the
// spec's Attrs list.
//
// Any Attr marked Secret never has its plaintext value written to the
// block: it's replaced with a reference to a variable named
// "<label>_<attr>", and the corresponding `variable { ... sensitive = true
// }` declaration is appended to the returned vars slice for the caller to
// emit alongside the resource block. The same substitution applies, key by
// key, to AttrMap attributes whose keys appear in spec.SecretMapKeys (e.g.
// notification-channel's config.webhook_url).
//
// An unrecognized cliName returns an error rather than panicking, so a
// caller iterating over arbitrary live resources can skip/report a kind it
// doesn't know how to render instead of crashing the whole generate run.
func renderResource(cliName string, id int, label string, attrs map[string]any) (block string, vars []string, err error) {
	spec, ok := specs[cliName]
	if !ok {
		return "", nil, fmt.Errorf("generate: no resource spec for kind %q (id=%d)", cliName, id)
	}

	type attrLine struct {
		name  string
		value string
	}
	var lines []attrLine
	maxName := 0

	for _, a := range spec.Attrs {
		val, present := attrs[a.Name]
		if !present || val == nil {
			continue
		}

		var rendered string
		switch {
		case a.Secret:
			varName := label + "_" + a.Name
			rendered = "var." + varName
			vars = append(vars, renderVariable(varName))
		case a.Kind == AttrMap:
			rendered = renderMapAttr(val, spec.SecretMapKeys, label, &vars)
		default:
			rendered = hclValue(val)
		}

		lines = append(lines, attrLine{name: a.Name, value: rendered})
		if len(a.Name) > maxName {
			maxName = len(a.Name)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "resource %q %q {\n", spec.TFType, label)
	for _, l := range lines {
		fmt.Fprintf(&b, "  %-*s = %s\n", maxName, l.name, l.value)
	}
	b.WriteString("}")

	return b.String(), vars, nil
}

// renderImport renders a Terraform 1.5+ `import` block that binds an
// existing live resource (identified by id) to the resource address the
// matching renderResource call just generated.
func renderImport(tfType, label string, id int) string {
	return fmt.Sprintf("import {\n  to = %s.%s\n  id = %q\n}", tfType, label, fmt.Sprintf("%d", id))
}

// renderVariable renders the sensitive string variable declaration used to
// back a redacted secret attribute's `var.<name>` reference.
func renderVariable(name string) string {
	return fmt.Sprintf("variable %q {\n  type      = string\n  sensitive = true\n}", name)
}

// renderMapAttr renders an AttrMap attribute's value as a nested HCL object
// literal, one level more indented than the enclosing resource block's
// attributes. Keys are sorted for determinism. Any key present in
// secretKeys is redacted behind a `var.<label>_<key>` reference — with the
// variable declaration appended to *vars — instead of being rendered via
// hclValue like the rest of the map's keys.
//
// val is accepted as either map[string]any (the common JSON-decoded shape)
// or map[string]string; any other shape (missing attr, wrong type from an
// unexpected SDK payload) renders as an empty object rather than panicking.
func renderMapAttr(val any, secretKeys []string, label string, vars *[]string) string {
	m, ok := asAnyMap(val)
	if !ok || len(m) == 0 {
		return "{}"
	}

	secret := make(map[string]bool, len(secretKeys))
	for _, k := range secretKeys {
		secret[k] = true
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{\n")
	for _, k := range keys {
		if secret[k] {
			varName := label + "_" + k
			fmt.Fprintf(&b, "    %s = var.%s\n", k, varName)
			*vars = append(*vars, renderVariable(varName))
			continue
		}
		fmt.Fprintf(&b, "    %s = %s\n", k, hclValue(m[k]))
	}
	b.WriteString("  }")
	return b.String()
}

// asAnyMap normalizes a map-shaped attribute value — map[string]any or
// map[string]string, the two shapes an SDK/JSON payload realistically hands
// us — to map[string]any so renderMapAttr has one representation to walk.
func asAnyMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, s := range m {
			out[k] = s
		}
		return out, true
	default:
		return nil, false
	}
}
