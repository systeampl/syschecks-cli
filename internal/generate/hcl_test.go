package generate

import "testing"

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
