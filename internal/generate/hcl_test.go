package generate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// parseHCLAttrValue writes `attr = <hclSrc>` to a real HCL parser
// (hashicorp/hcl/v2, the library terraform itself uses) and returns the
// evaluated attribute value. It fails the test if the source doesn't parse
// or the attribute can't be evaluated, so callers can assert generated HCL
// is actually valid rather than merely matching an expected string.
func parseHCLAttrValue(t *testing.T, hclSrc string) cty.Value {
	t.Helper()
	src := fmt.Sprintf("attr = %s\n", hclSrc)
	p := hclparse.NewParser()
	f, diags := p.ParseHCL([]byte(src), "test.tf")
	if diags.HasErrors() {
		t.Fatalf("HCL did not parse (src=%q): %s", src, diags.Error())
	}
	attrs, diags := f.Body.JustAttributes()
	if diags.HasErrors() {
		t.Fatalf("HCL attributes error (src=%q): %s", src, diags.Error())
	}
	attr, ok := attrs["attr"]
	if !ok {
		t.Fatalf("HCL missing attr (src=%q)", src)
	}
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		t.Fatalf("HCL eval error (src=%q): %s", src, diags.Error())
	}
	return val
}

// assertHCLStringRoundTrips renders content via hclValue, parses the result
// with a real HCL parser, and checks the parsed value is the string
// "content" unchanged — i.e. the escaping neither corrupts the value nor
// (critically) lets it be reinterpreted as a template expression.
func assertHCLStringRoundTrips(t *testing.T, content string) {
	t.Helper()
	rendered := hclValue(content)
	val := parseHCLAttrValue(t, rendered)
	if val.Type() != cty.String {
		t.Fatalf("hclValue(%q) = %s parsed as non-string: %#v", content, rendered, val)
	}
	if got := val.AsString(); got != content {
		t.Errorf("hclValue(%q) = %s, round-tripped through HCL parser as %q, want %q", content, rendered, got, content)
	}
}

func TestHCLValueString(t *testing.T) {
	got := hclValue("a\"b")
	want := `"a\"b"`
	if got != want {
		t.Errorf("hclValue(%q) = %s, want %s", "a\"b", got, want)
	}
}

func TestHCLValueStringEscapesBackslashAndNewline(t *testing.T) {
	got := hclValue("a\\b\nc")
	want := `"a\\b\nc"`
	if got != want {
		t.Errorf("hclValue = %s, want %s", got, want)
	}
}

func TestHCLValueIntNotFloat(t *testing.T) {
	got := hclValue(300)
	if got != "300" {
		t.Errorf("hclValue(300) = %s, want 300", got)
	}
}

func TestHCLValueFloat64WholeNumberRendersAsInt(t *testing.T) {
	got := hclValue(float64(300))
	if got != "300" {
		t.Errorf("hclValue(float64(300)) = %s, want 300", got)
	}
}

func TestHCLValueFloatWithFraction(t *testing.T) {
	got := hclValue(1.5)
	if got != "1.5" {
		t.Errorf("hclValue(1.5) = %s, want 1.5", got)
	}
}

// TestHCLValueHugeWholeNumberFloatNoOverflow guards float64(int64(val)) for
// a whole-number float outside int64's range: converting it straight to
// int64 is an implementation-defined truncation in Go, not a safe cast, and
// can silently render the wrong number. The result must still be a valid
// HCL number literal that round-trips through a real parser to the same
// value.
func TestHCLValueHugeWholeNumberFloatNoOverflow(t *testing.T) {
	huge := 1e20 // whole number, far outside int64 range
	rendered := hclValue(huge)
	if strings.ContainsAny(rendered, "eE") {
		t.Errorf("hclValue(1e20) = %s, want plain decimal digits, not exponential notation", rendered)
	}
	val := parseHCLAttrValue(t, rendered)
	got, _ := val.AsBigFloat().Float64()
	if got != huge {
		t.Errorf("hclValue(1e20) = %s, round-tripped through HCL parser as %v, want %v", rendered, got, huge)
	}
}

func TestHCLValueBool(t *testing.T) {
	if got := hclValue(true); got != "true" {
		t.Errorf("hclValue(true) = %s, want true", got)
	}
	if got := hclValue(false); got != "false" {
		t.Errorf("hclValue(false) = %s, want false", got)
	}
}

func TestHCLValueList(t *testing.T) {
	got := hclValue([]any{"x", "y"})
	want := `["x", "y"]`
	if got != want {
		t.Errorf("hclValue([]any{x,y}) = %s, want %s", got, want)
	}
}

func TestHCLValueEmptyList(t *testing.T) {
	got := hclValue([]any{})
	if got != "[]" {
		t.Errorf("hclValue([]any{}) = %s, want []", got)
	}
}

func TestHCLValueMapSortedMultiline(t *testing.T) {
	got := hclValue(map[string]any{"k": "v"})
	want := "{\n  k = \"v\"\n}"
	if got != want {
		t.Errorf("hclValue(map) = %q, want %q", got, want)
	}
}

func TestHCLValueMapSortedKeys(t *testing.T) {
	got := hclValue(map[string]any{"b": "2", "a": "1"})
	want := "{\n  a = \"1\"\n  b = \"2\"\n}"
	if got != want {
		t.Errorf("hclValue(map) = %q, want %q", got, want)
	}
}

func TestHCLValueEmptyMap(t *testing.T) {
	got := hclValue(map[string]any{})
	if got != "{}" {
		t.Errorf("hclValue(empty map) = %s, want {}", got)
	}
}

func TestHCLValueUnknownTypeNoPanic(t *testing.T) {
	type weird struct{ X int }
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("hclValue panicked on unknown type: %v", r)
		}
	}()
	got := hclValue(weird{X: 1})
	if got == "" {
		t.Errorf("hclValue(weird{}) returned empty string")
	}
}

func TestHCLLabelSanitizesAndLowercases(t *testing.T) {
	got := hclLabel("Web Prod #1")
	if got != "web_prod_1" {
		t.Errorf("hclLabel(%q) = %q, want %q", "Web Prod #1", got, "web_prod_1")
	}
}

func TestHCLLabelCollapsesRepeatedSeparators(t *testing.T) {
	got := hclLabel("a---b")
	if got != "a_b" {
		t.Errorf("hclLabel(a---b) = %q, want a_b", got)
	}
}

func TestHCLLabelTrimsLeadingTrailingUnderscore(t *testing.T) {
	got := hclLabel("  hello  ")
	if got != "hello" {
		t.Errorf("hclLabel(%q) = %q, want hello", "  hello  ", got)
	}
}

func TestHCLLabelPrefixesWhenLeadingDigit(t *testing.T) {
	got := hclLabel("123abc")
	if got != "r_123abc" {
		t.Errorf("hclLabel(123abc) = %q, want r_123abc", got)
	}
}

func TestHCLLabelFallbackWhenEmpty(t *testing.T) {
	got := hclLabel("###")
	if got == "" {
		t.Errorf("hclLabel(###) returned empty string")
	}
	if got[0] < 'a' || got[0] > 'z' {
		t.Errorf("hclLabel(###) = %q, does not start with a lowercase letter", got)
	}
}

// TestHCLLabelFallbackDiffersByInput guards against the empty-sanitization
// fallback collapsing every unrelated input to the same fixed label (e.g.
// always "r_1"), which would silently merge distinct resources' labels.
func TestHCLLabelFallbackDiffersByInput(t *testing.T) {
	a := hclLabel("###")
	b := hclLabel("@@@")
	c := hclLabel("")
	if a == b || a == c || b == c {
		t.Errorf("fallback labels not distinct: hclLabel(###)=%q hclLabel(@@@)=%q hclLabel(\"\")=%q", a, b, c)
	}
}

func TestHCLLabelFallbackDeterministic(t *testing.T) {
	if hclLabel("###") != hclLabel("###") {
		t.Errorf("hclLabel(###) not deterministic across calls")
	}
}

func TestLabelSetDedupesOnCollision(t *testing.T) {
	ls := newLabelSet()
	first := ls.unique("web")
	second := ls.unique("web")
	third := ls.unique("web")

	if first != "web" {
		t.Errorf("first unique(web) = %q, want web", first)
	}
	if second != "web_2" {
		t.Errorf("second unique(web) = %q, want web_2", second)
	}
	if third != "web_3" {
		t.Errorf("third unique(web) = %q, want web_3", third)
	}
}

func TestLabelSetDeterministicAcrossInstances(t *testing.T) {
	ls1 := newLabelSet()
	ls2 := newLabelSet()

	a1 := ls1.unique("x")
	a2 := ls1.unique("x")

	b1 := ls2.unique("x")
	b2 := ls2.unique("x")

	if a1 != b1 || a2 != b2 {
		t.Errorf("labelSet not deterministic: (%q,%q) vs (%q,%q)", a1, a2, b1, b2)
	}
}

func TestLabelSetIndependentBases(t *testing.T) {
	ls := newLabelSet()
	web := ls.unique("web")
	db := ls.unique("db")
	if web != "web" || db != "db" {
		t.Errorf("independent bases collided: web=%q db=%q", web, db)
	}
}

// TestLabelSetCrossBaseCollision reproduces the review-reported bug: a
// per-base counter alone lets a name that the set would itself generate for
// one base (e.g. "web_2" from repeated "web") get handed out a second time
// once "web" collides again, if some other call already claimed that exact
// suffixed string. All three calls below must yield distinct labels.
func TestLabelSetCrossBaseCollision(t *testing.T) {
	ls := newLabelSet()
	first := ls.unique("web")    // "web"
	second := ls.unique("web_2") // literal "web_2", handed in directly
	third := ls.unique("web")    // must NOT collide with "web_2" above

	if first != "web" {
		t.Errorf("first unique(web) = %q, want web", first)
	}
	if second != "web_2" {
		t.Errorf("unique(web_2) = %q, want web_2", second)
	}
	if third == second {
		t.Errorf("unique(web) second call collided with unique(web_2): both %q", third)
	}
	seen := map[string]bool{}
	for _, label := range []string{first, second, third} {
		if seen[label] {
			t.Errorf("label %q handed out more than once: %v", label, []string{first, second, third})
		}
		seen[label] = true
	}
}

// --- HCL-parser-verified escaping (regression coverage for review findings) ---

func TestHCLValueTemplateInterpolationEscaped(t *testing.T) {
	// Unescaped, HCL would evaluate "${1+1}" as the expression 1+1 = 2
	// instead of keeping it as a literal string.
	assertHCLStringRoundTrips(t, "${1+1}")
}

func TestHCLValueTemplateDirectiveEscaped(t *testing.T) {
	assertHCLStringRoundTrips(t, "%{if true}x%{endif}")
}

func TestHCLValueDoubleDollarAndPercentRoundTrip(t *testing.T) {
	assertHCLStringRoundTrips(t, "price is $$5 and 50%%")
}

func TestHCLValueBareDollarAndPercentNotOverEscaped(t *testing.T) {
	// A lone "$" or "%" not immediately followed by "{" is not special in
	// HCL and must survive unchanged — doubling it unconditionally would
	// corrupt extremely common values like prices and percentages.
	assertHCLStringRoundTrips(t, "$5 off, save 50%")
}

func TestHCLValueMixedQuoteBackslashInterpolation(t *testing.T) {
	assertHCLStringRoundTrips(t, `say "hi" \ then ${boom}`)
}

func TestHCLValueCarriageReturnEscaped(t *testing.T) {
	assertHCLStringRoundTrips(t, "line1\r\nline2")
}

func TestHCLValueTabEscaped(t *testing.T) {
	assertHCLStringRoundTrips(t, "a\tb")
}

func TestHCLValueNestedMapParsesAsValidHCL(t *testing.T) {
	nested := map[string]any{
		"outer1": "v1",
		"outer2": map[string]any{
			"inner1": "v2",
			"inner2": "v3",
		},
	}
	rendered := hclValue(nested)
	val := parseHCLAttrValue(t, rendered)
	if val.Type().IsObjectType() && !val.Type().HasAttribute("outer2") {
		t.Errorf("hclValue(nested map) = %s, parsed object missing outer2", rendered)
	}
	outer2 := val.GetAttr("outer2")
	if !outer2.Type().IsObjectType() || !outer2.Type().HasAttribute("inner1") {
		t.Errorf("hclValue(nested map) = %s, outer2 is not a nested object with inner1: %#v", rendered, outer2)
	}
	if got := val.GetAttr("outer1").AsString(); got != "v1" {
		t.Errorf("nested map outer1 = %q, want v1", got)
	}
	if got := outer2.GetAttr("inner2").AsString(); got != "v3" {
		t.Errorf("nested map outer2.inner2 = %q, want v3", got)
	}
}
