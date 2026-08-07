package generate

// ResourceKinds is the fixed, deterministic set of syschecks-cli resource
// kinds `generate terraform` knows how to emit, in the order their files are
// considered when no --type filter narrows the set. It mirrors the keys of
// specs; kept as an explicit slice (rather than derived by ranging specs,
// whose map iteration order is randomized) so callers get a stable default
// order without having to sort themselves.
var ResourceKinds = []string{"check", "notification-channel", "team"}

// TFType returns the Terraform resource type name for a syschecks-cli
// resource kind (e.g. "check" -> "systeam_check") and whether kind is one
// generate knows about.
func TFType(kind string) (string, bool) {
	spec, ok := specs[kind]
	if !ok {
		return "", false
	}
	return spec.TFType, true
}

// RenderResource is the exported entry point onto the package-private
// renderResource, for callers outside package generate (internal/cli's
// `generate terraform` command).
func RenderResource(cliName string, id int, label string, attrs map[string]any) (block string, vars []string, err error) {
	return renderResource(cliName, id, label, attrs)
}

// RenderImport is the exported entry point onto the package-private
// renderImport.
func RenderImport(tfType, label string, id int) string {
	return renderImport(tfType, label, id)
}

// HCLLabel is the exported entry point onto the package-private hclLabel.
func HCLLabel(s string) string {
	return hclLabel(s)
}

// LabelSet deduplicates HCL labels across every resource a `generate`
// invocation emits — exported wrapper around labelSet so a single instance
// can be shared across all resource kinds by internal/cli, keeping labels
// collision-free across the whole generated directory, not just within one
// file.
type LabelSet struct {
	inner *labelSet
}

// NewLabelSet returns an empty, ready-to-use LabelSet.
func NewLabelSet() *LabelSet {
	return &LabelSet{inner: newLabelSet()}
}

// Unique returns a collision-free label derived from base (see labelSet.unique).
func (ls *LabelSet) Unique(base string) string {
	return ls.inner.unique(base)
}
