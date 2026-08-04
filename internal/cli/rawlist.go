package cli

import (
	"encoding/json"
	"fmt"
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
