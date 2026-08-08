package generate

import (
	"fmt"
	"hash/fnv"
	"math"
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
		// float64(int64(val)) is only a safe round-trip check within
		// int64's range; converting an out-of-range float straight to
		// int64 is an implementation-defined truncation in Go, not a
		// value-preserving cast. Guard the range first, and for a
		// whole-number float outside it, print plain decimal digits (via
		// FormatFloat's 'f' verb) rather than int64-overflowing or falling
		// into 'g' verb's exponential notation.
		if !math.IsInf(val, 0) && val == math.Trunc(val) {
			if val >= math.MinInt64 && val <= math.MaxInt64 {
				return strconv.FormatInt(int64(val), 10)
			}
			return strconv.FormatFloat(val, 'f', -1, 64)
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
// backslashes, double quotes, newlines, carriage returns, tabs, and template
// interpolation/directive markers.
//
// HCL's template syntax treats the two-character sequences "${" and "%{" as
// the start of an interpolation or control directive, not the literal
// characters — e.g. `"${1+1}"` evaluates to the number 2. Left unescaped, an
// SDK value containing either sequence (a URL, a k8s/compose snippet, a
// description mentioning a shell variable, ...) gets silently reinterpreted
// as an expression instead of being written back verbatim, which can also
// leak whatever happens to be in scope.
//
// The escape is doubling ("$${", "%%{"), and it only applies when the
// dollar/percent sign is immediately followed by "{" — HCL does not fold
// "$$"/"%%" outside of that context, so a bare "$" or "%" (e.g. "$5",
// "50%", a Windows env-style "%PATH%") is left untouched. Verified against
// hashicorp/hcl/v2's parser: unconditionally doubling every "$"/"%" round-trips
// "a$b" back as "a$$b" — corrupting the very common case of a lone currency
// or percent sign — whereas the followed-by-"{" rule round-trips correctly
// for both plain values and interpolation/directive markers.
func quoteHCLString(s string) string {
	runes := []rune(s)
	var b strings.Builder
	b.WriteByte('"')
	for i, r := range runes {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '$', '%':
			if i+1 < len(runes) && runes[i+1] == '{' {
				b.WriteRune(r)
			}
			b.WriteRune(r)
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
		// Every input that sanitizes to nothing (empty string, "###", "@@@",
		// ...) would otherwise collapse to the same fixed fallback and
		// collide. hclLabel is pure (no shared state to draw a counter
		// from — see labelSet for that), so instead derive a fallback from
		// a hash of the original input: different empty-sanitizing inputs
		// get different fallback labels, deterministically.
		h := fnv.New32a()
		h.Write([]byte(s))
		return fmt.Sprintf("r_%d", h.Sum32())
	}
	if label[0] >= '0' && label[0] <= '9' {
		return "r_" + label
	}
	return label
}

// labelSet deduplicates HCL labels, appending "_2", "_3", ... on repeated
// calls with the same base string so generated resources never collide.
//
// Tracking a per-base counter alone is not enough: if some other base
// happens to collide with a suffixed name this set would have generated
// (e.g. unique("web") -> "web", then a literal unique("web_2") is handed
// in, then unique("web") again), a counter that only tracks how many times
// "web" itself was requested would hand out "web_2" a second time. So this
// tracks every label ever handed out and skips any candidate already taken,
// not just an increasing suffix count.
type labelSet struct {
	used map[string]bool
	next map[string]int // next suffix to try for a given base, once collided
}

// newLabelSet returns an empty, ready-to-use labelSet.
func newLabelSet() *labelSet {
	return &labelSet{used: make(map[string]bool), next: make(map[string]int)}
}

// unique returns base the first time it (or an equal candidate) is handed
// out, and otherwise the first of base_2, base_3, ... not already handed
// out by this labelSet — deterministic given the same sequence of calls.
func (ls *labelSet) unique(base string) string {
	if !ls.used[base] {
		ls.used[base] = true
		return base
	}
	n := ls.next[base]
	if n < 2 {
		n = 2
	}
	for {
		candidate := fmt.Sprintf("%s_%d", base, n)
		if !ls.used[candidate] {
			ls.used[candidate] = true
			ls.next[base] = n + 1
			return candidate
		}
		n++
	}
}
