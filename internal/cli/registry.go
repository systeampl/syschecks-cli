package cli

import "encoding/json"

// OrgMode describes how a resource type takes its organization scope, so the
// generic command factory (crud.go) knows which org helper to call and
// whether org is required at all.
type OrgMode int

const (
	OrgNone  OrgMode = iota // not org-scoped (organizations, status pages, contact methods)
	OrgArg                  // org id is a required path arg to the SDK call (teams, services, oncall, playbooks, lifecycle, projects, integration keys)
	OrgParam                // org id rides in a *ListXParams.OrgId and is optional (checks, notification channels, maintenance, incidents)
)

// Field describes one create/update input: the cobra flag to generate, the
// key it lands under in the request body / -f document, and whether create
// requires it. Kind drives which cobra flag constructor is used.
type Field struct {
	Name     string // flag name, kebab-case (e.g. "url", "interval")
	JSONKey  string // key in the -f document / body (e.g. "url", "interval_seconds")
	Kind     string // "string" | "int" | "bool"
	Required bool   // enforced on create when no flag/-f value supplies it
	Help     string
}

// Resource is one registry entry: the shape the generic CRUD command factory
// (newResourceCmd, crud.go) needs to build list/get/create/update/delete
// subcommands for a single API resource type. The closures adapt that type's
// (non-uniform) SDK calls to a common shape; any nil verb means the factory
// does not generate that subcommand at all (e.g. agents have no createFn).
type Resource struct {
	Name    string // singular command name, e.g. "team"
	Aliases []string
	Cols    []string // table columns for get/create/update (and list, when ListCols is empty)
	// ListCols is the table columns for `list` specifically, when the list
	// response is a different (typically richer) shape than the single-item
	// get/create/update responses — e.g. service's list endpoint returns
	// health_status/checks_count that its detail/create/update responses
	// don't have. Leave nil/empty to have `list` render Cols, same as every
	// other verb; only set it when list's response type actually differs.
	ListCols []string
	Org      OrgMode
	Fields   []Field // create/update flag set + -f schema

	listFn   func(env *cmdCtx, orgID *int) ([]map[string]any, error)
	getFn    func(env *cmdCtx, orgID *int, id int) (map[string]any, error)
	createFn func(env *cmdCtx, orgID *int, body map[string]any) (map[string]any, error)
	updateFn func(env *cmdCtx, orgID *int, id int, body map[string]any) (map[string]any, error)
	deleteFn func(env *cmdCtx, orgID *int, id int) error
}

// registry is the single source of truth for resource types: name/alias ->
// *Resource. Populated by register calls in each resources_*.go file (later
// tasks); Task 1 only defines the machinery, it registers nothing itself.
var registry = map[string]*Resource{}

// register adds r to the registry under its Name and every Alias.
func register(r *Resource) {
	registry[r.Name] = r
	for _, a := range r.Aliases {
		registry[a] = r
	}
}

// toMap round-trips v through JSON into a map[string]any: the standard way a
// typed SDK response becomes the registry's normalized shape for rendering.
func toMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// mapToBody round-trips a -f/flag body map into a typed SDK request body T
// via JSON, the inverse of toMap: it's how a createFn/updateFn closure turns
// the registry's normalized map[string]any into the concrete
// models.<X>Create/<X>Update struct the SDK call expects, without any
// hand-written field-by-field mapping.
func mapToBody[T any](m map[string]any) (T, error) {
	var t T
	b, err := json.Marshal(m)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(b, &t); err != nil {
		return t, err
	}
	return t, nil
}
