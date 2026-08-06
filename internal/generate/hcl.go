package generate

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// hclValue renders a Go value — as decoded from JSON (string, float64/int,
// bool, []any, map[string]any) — as an HCL literal. It is total: unknown
// types fall back to a quoted %v representation rather than panicking, so
// callers never need to guard against surprising SDK payload shapes.
func hclValue(v any) string {
	switch val := v.(type) {
	case string:
		return quoteHCLString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'g', -1, 64)
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		items := make([]string, len(val))
		for i, item := range val {
			items[i] = hclValue(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("{\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s = %s\n", k, hclValue(val[k]))
		}
		b.WriteString("}")
		return b.String()
	default:
		return quoteHCLString(fmt.Sprintf("%v", val))
	}
}

// quoteHCLString renders s as an HCL double-quoted string literal, escaping
// backslashes, double quotes, and newlines.
func quoteHCLString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// hclLabel sanitizes s into a valid HCL identifier label: lowercased, with
// every run of characters outside [a-z0-9_] collapsed to a single
// underscore, leading/trailing underscores trimmed, and — if the result
// starts with a digit or is empty — prefixed with "r_" so it starts with a
// letter.
func hclLabel(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	prevUnderscore := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			prevUnderscore = r == '_'
			continue
		}
		if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	label := strings.Trim(b.String(), "_")

	if label == "" {
		return "r_1"
	}
	if label[0] >= '0' && label[0] <= '9' {
		return "r_" + label
	}
	return label
}

// labelSet deduplicates HCL labels, appending "_2", "_3", ... on repeated
// calls with the same base string so generated resources never collide.
type labelSet struct {
	counts map[string]int
}

// newLabelSet returns an empty, ready-to-use labelSet.
func newLabelSet() *labelSet {
	return &labelSet{counts: make(map[string]int)}
}

// unique returns base the first time it is seen, and base_2, base_3, ... on
// each subsequent call with the same base.
func (ls *labelSet) unique(base string) string {
	n := ls.counts[base]
	ls.counts[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, n+1)
}
