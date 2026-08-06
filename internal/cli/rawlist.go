package cli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/systeampl/syschecks-cli/internal/clierr"
	"github.com/systeampl/syschecks-cli/internal/output"
)

// extractItems decodes an untyped list endpoint's body into a slice of rows.
// Some endpoints return a bare JSON array (e.g. /api/checks/), others wrap the
// array in an object under a named key (e.g. /api/incidents -> {"incidents":[...]},
// /api/organizations/{id}/agents -> {"agents":[...]}). This accepts either shape.
func extractItems(raw json.RawMessage, key string) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	inner, ok := obj[key]
	if !ok {
		return nil, fmt.Errorf("response has no %q array", key)
	}
	if err := json.Unmarshal(inner, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// renderRawObject decodes an untyped single-object json.RawMessage response
// (incident get/acknowledge/resolve, ...) into a map and renders it as a
// one-row table. These endpoints have no typed SDK model, so there is no
// fixed Resource.Cols to render against: the columns are the response's own
// keys, sorted for stable output.
func renderRawObject(env *cmdCtx, raw json.RawMessage) error {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return clierr.Config("decoding response: %v", err)
	}
	cols := make([]string, 0, len(m))
	for k := range m {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return renderTable(env.Out, env.Format, env.Quiet, env.NoColor, output.Table{Cols: cols, Rows: []output.Row{output.Row(m)}})
}
