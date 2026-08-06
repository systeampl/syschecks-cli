# syschecks CLI v0.3 `generate terraform` — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** `syschecks generate terraform|opentofu --org <org> -o <dir>` renders `check`, `notification-channel`, `team` (Phase 1) into HCL + `import{}` blocks the published `systeampl/systeam` provider adopts on `plan` with **0 to add/change/destroy**. Secrets → sensitive variables.

**Architecture:** iterate in-scope resources via the v2 registry read closures (`listFn`/`getFn` → `map[string]any`); a per-resource **attribute spec** (writable attrs + type + secret/nested, baked from the provider) drives a single HCL renderer that writes `provider.tf`/`<type>.tf`/`imports.tf`/`variables.tf`.

**Tech stack:** Go 1.24, on branch `impl/v0.2-crud` (v0.2 registry). New package `internal/generate`. Provider source for the spec: `/home/destine/GIT/wlasne/terraform-provider-systeam`.

## Global Constraints
- Read-only against the account (list/get) + local file writes; NEVER mutate the account.
- Emit only attributes the provider accepts (its Required+Optional set); exclude pure-Computed (`id`, `created_at`, `updated_at`, `status`, `uuid`, `last_*`). Emitting an unknown arg breaks `plan`.
- For `plan == no-op`: emit every writable attribute that is **non-null** in the SDK read (matches live state).
- Secrets (notification `config` secret keys, check `auth_password`/`db_password`) → `var.<label>_<attr>` (sensitive), never inlined.
- Deterministic output: resources sorted by id, attributes in spec order.
- Attribute names are snake_case = SDK json keys (1:1); the spec is an allowlist + flags, not a rename table.
- TDD; keep `go build/vet/test ./...` green.

## Interfaces (Task 1)
```go
// internal/generate/schema.go
type AttrKind int
const ( AttrString AttrKind = iota; AttrInt; AttrBool; AttrList; AttrMap )
type Attr struct {
    Name   string   // snake_case HCL attribute = SDK json key
    Kind   AttrKind
    Secret bool     // route to a sensitive variable, never inline
}
type ResourceSpec struct {
    TFType string   // e.g. "systeam_check"
    Attrs  []Attr   // provider Required+Optional, computed excluded, in provider order
    SecretMapKeys []string // for map attrs (notification config): keys treated as secret
}
var specs = map[string]*ResourceSpec{} // keyed by CLI resource name ("check", "notification-channel", "team")
```

---

### Task 1: Attribute spec from the provider (check, notification-channel, team)

**Files:** Create `internal/generate/schema.go` (the `specs` map + types), `internal/generate/schema_test.go`. Also create `hack/extract-provider-schema.sh` (or a Go `//go:generate` helper) documenting how `specs` was derived from `/home/destine/GIT/wlasne/terraform-provider-systeam/internal/resources/<res>/resource.go` (parse `"<name>": schema.<Kind>Attribute{ Required|Optional|Computed }` — include Required+Optional, drop pure-Computed).

- [ ] **Step 1:** Write a failing test asserting `specs["check"].TFType == "systeam_check"`, that `specs["check"]` contains `name/type/project_id` (Required) and `url/interval/is_active/...` (Optional) but NOT `id`/`created_at`/`status`, and `specs["notification-channel"]` has a Map attr `config` with `SecretMapKeys` including `webhook_url`.
- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 3:** Extract the writable attribute lists for the three resources from the provider source (run the hack script or read the schema files) and hand them into `specs` as Go literals. For notification `config` (a `MapAttribute`), set `Kind: AttrMap` and `SecretMapKeys: []string{"webhook_url","url","token","api_key","auth_token","integration_key","routing_key","password"}` (the config keys that carry secrets). Mark check `auth_password`,`db_password`,`auth_bearer_token` `Secret: true`.
- [ ] **Step 4:** Run — PASS. **Step 5:** Commit `feat(generate): provider-aligned attribute spec for check/notification/team`.

### Task 2: HCL value + label rendering

**Files:** `internal/generate/hcl.go`, `internal/generate/hcl_test.go`.
- [ ] **Step 1:** Failing tests: `hclValue("a\"b")` → `"a\"b"` (quoted, escaped); `hclValue(300)`→`300`; `hclValue(true)`→`true`; `hclValue([]any{"x","y"})`→`["x", "y"]`; `hclValue(map[string]any{"k":"v"})`→ a multiline `{ k = "v" }`; `hclLabel("Web Prod #1")`→`web_prod_1`; `dedupeLabel` adds `_2` on collision.
- [ ] **Step 2:** FAIL. **Step 3:** Implement `hclValue(any) string`, `hclLabel(string) string` (lowercase, `[^a-z0-9_]`→`_`, collapse repeats, lead-alpha, fallback `r_<id>`), and a label deduper. **Step 4:** PASS. **Step 5:** Commit `feat(generate): HCL value + label rendering`.

### Task 3: Resource-block + import + variable rendering

**Files:** `internal/generate/render.go`, `render_test.go`.
- [ ] **Step 1:** Failing golden test: given `renderResource("check", id=5, attrs=map{name:"web",type:"http",project_id:3,url:"https://x",interval:300,status:"UP",id:5,created_at:"…"})` and `specs["check"]`, assert the emitted block is exactly:
```
resource "systeam_check" "web" {
  name       = "web"
  type       = "http"
  project_id = 3
  url        = "https://x"
  interval   = 300
}
```
(only non-null spec attrs, in spec order, `status`/`id`/`created_at` excluded, aligned `=`), that `renderImport` yields `import {\n  to = systeam_check.web\n  id = "5"\n}`, and that a secret attr (e.g. notification `config.webhook_url`) renders as `webhook_url = var.<label>_webhook_url` plus a returned `variable "<label>_webhook_url" { type = string\n  sensitive = true\n}`.
- [ ] **Step 2:** FAIL. **Step 3:** Implement `renderResource(kind, id, attrs, spec) (block string, vars []string)`, `renderImport(tfType, label, id) string`. Non-null check against the SDK map; secrets → var refs + collected variable decls; map attrs render nested (secret keys → vars). **Step 4:** PASS. **Step 5:** Commit `feat(generate): resource/import/variable rendering`.

### Task 4: `generate terraform` command

**Files:** `internal/cli/generate.go`, `generate_test.go`; register in `root.go`.
- [ ] **Step 1:** Failing test: against `newFakeAPI` returning one check + one notification-channel + one team for org `acme`, run `generate terraform --org acme -o <tmpdir>`; assert files exist — `provider.tf` (contains `source = "systeampl/systeam"`), `checks.tf`/`notification_channels.tf`/`teams.tf` (contain the `resource` blocks), `imports.tf` (contains an `import` block per resource), `variables.tf` (a sensitive `variable` for the channel's secret); and stderr warns which vars to set. Also `--type check` emits only `checks.tf`+its imports.
- [ ] **Step 2:** FAIL. **Step 3:** Implement: resolve org; for each in-scope type (`check`,`notification-channel`,`team`; narrowed by `--type`/`--project`/`--check`), call the registry `listFn` then `getFn` per id (full attrs), render, group by type into files; write `provider.tf`/`imports.tf`/`variables.tf`; print the secret-var warning. Deterministic ordering. **Step 4:** PASS. **Step 5:** Commit `feat(generate): generate terraform command (whole-org + filters)`.

### Task 5: `opentofu` alias + README

**Files:** `internal/cli/generate.go` (add `opentofu` subcommand → same renderer), `README.md`.
- [ ] Add `generate opentofu` producing identical HCL; test asserts it writes the same files as `terraform`. Document `generate` in README (usage, output files, secrets-as-vars, `import`, `--type/--project/--check`). Commit `feat(generate): opentofu subcommand + README`.

### Task 6: real `tofu plan == no-op` E2E (manual/gated)

**Files:** `docs/e2e-generate.md` (a runbook), no code.
- [ ] Document the acceptance E2E (run by a human/CI with tofu installed + a real token): `syschecks generate terraform --org <throwaway> -o /tmp/iac` → `cd /tmp/iac && tofu init && tofu plan` → expect **"No changes. Your infrastructure matches the configuration."** (after supplying the sensitive vars). This is the real correctness gate; unit golden tests stand in for the automated suite. Commit `docs(generate): tofu-plan no-op E2E runbook`.

## Self-review
- Correctness hinges on Task 1's allowlist matching the provider and Task 3 emitting every non-null writable attr → the golden tests + the Task 6 `tofu plan` no-op are the proof.
- Nested/complex attrs beyond notification `config` (check `api_scenario_steps`, etc.) are Phase 2 — Phase 1 covers scalar + the config map. If a Phase-1 resource has a set nested attr with no spec entry, it's omitted (plan may show drift on that attr → flagged for Phase 2, not a Phase-1 failure for the 3 simple shapes chosen).
- Phase 2 (remaining 11 types + nested blocks) is a separate plan.
