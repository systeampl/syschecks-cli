# syschecks CLI — full-coverage design (v0.2 → v0.4)

**Date:** 2026-08-05
**Status:** approved (brainstorm), pre-plan
**Goal:** close the CLI to 100% of what `syschecks-go` + the backend already expose — full imperative CRUD for all resource types, IaC code generation, and org export/backup — leaving only `probe from <geo>` (a genuine backend gap) out of scope.

Input: coverage audit `healtchecks/docs/plans/2026-08-03-syschecks-cli-audit.md` (338 API paths / 448 ops, 23 SDK namespaces, 15 resource types — all with typed list/get/create/update/delete in the SDK).

## Non-goals
- `probe from <geo>` (on-demand geo-agent dispatch) — needs a new backend endpoint + agent protocol. Tracked separately, not part of "100%".
- Reimplementing HTTP — the CLI stands on `syschecks-go`. API → SDK → CLI.

## Principle: one resource registry
The 15 resource types are **not** hand-coded 15×. A single **registry** declares each type once:
- SDK bindings: `List / Get / Create / Update / Delete` method handles.
- Field descriptors: which fields map to flags (common) vs `-f` only (nested/full), types, required.
- Canonical (de)serialization to/from YAML/JSON.
- Identity: how to name/reference it (id, slug), and for generate: the IaC resource address + import id.

Three features consume the **same** registry, so they never diverge (audit's key requirement):
1. **CRUD** — generic `create/get/list/update/delete` + `apply -f` iterate the registry.
2. **export/backup** — iterate registry → typed reads → canonical YAML.
3. **generate** — iterate registry → per-tool renderer.

The 15 types: `organization, project, check, notification-channel, team, service, oncall(-schedule), escalation(-policy), maintenance(-window), playbook, status-page, lifecycle(-watch), contact-method, integration-key`, plus read/actions for `incident` and `agent`.

## Feature area 1 — CRUD surface (v0.2)
Uniform verbs across every registered type:
```
syschecks <resource> list [--org] [filters]
syschecks <resource> get <id>
syschecks <resource> create [--flags…] [-f file.yaml]
syschecks <resource> update <id> [--flags…] [-f file.yaml]
syschecks <resource> delete <id> [--yes]
syschecks apply -f file.yaml         # create-or-update, one or many docs (---), any type
```
- **Hybrid input:** common fields as flags (quick one-offs); full/nested via `-f` (YAML/JSON). Flags overlay `-f` when both given. Flag sets are generated from the registry field descriptors, so flags↔schema cannot drift.
- **Round-trip:** `get -o yaml` / `export` produce documents `apply -f` accepts.
- **Resource actions** beyond CRUD:
  - `incident get|acknowledge|resolve <id>` (list already exists).
  - `agent get|create|delete <id>` + `agent token <id>` (registration token).
  - `check` keeps its `run/pause/resume/test-alert`.
- **Delete safety:** `--yes` required non-interactively; TTY prompts otherwise.

## Feature area 2 — generate (v0.3)
```
syschecks generate terraform|opentofu|pulumi|ansible --org <org>
    [--project P] [--type check,notification] [--check id,…] -o ./iac/
```
- Data from typed SDK reads (list/get) via the registry — no backend work.
- **Output = directory**, one file per type (`checks.tf`, `notifications.tf`, …).
- **Import wiring is mandatory** so `plan`/`up` does not try to recreate existing resources:
  - terraform + opentofu = **one HCL renderer**, emits `import { to = …, id = "…" }` blocks (`imports.tf`).
  - pulumi = program + a `pulumi import` script / resource options with import ids.
  - ansible = playbook with `state: present` (idempotent, no import concept).
- **Whole-org by default**, `--project/--type/--check` narrow the set (for incremental migration of an existing repo).
- Renderers are registry-driven: each type contributes its address + attribute mapping; the per-tool renderer is a thin formatter over that.

## Feature area 3 — export / backup / import / drift (v0.4)
```
syschecks export org <org> -o backup.yaml        # all 15 types, from typed reads (richer than the MVP export endpoint)
syschecks apply -f backup.yaml --dry-run          # client-side diff (= drift), then apply for real
syschecks drift detect -f backup.yaml             # alias of apply --dry-run scoped to a file
```
- **Built from typed reads via the registry**, not the backend `/export` endpoint (which is MVP: 7/15 types, no dry-run). Same canonical YAML as `get -o yaml` / generate source.
- **Secrets:** redacted by default; `--include-secrets` is an explicit, warned flag. (A restore of redacted config needs secret re-entry — documented.)
- **`--dry-run`:** client-side plan — fetch current state, diff against the file, print create/update/delete/no-op per resource; exit 0 if in sync, non-zero if drift (CI-usable). This subsumes `drift detect`.

## Phasing (independent, each shippable)
- **v0.2** — registry + full CRUD surface + `apply` + incident/agent actions. Foundation for the rest.
- **v0.3** — `generate` (all four tools, import wiring).
- **v0.4** — `export`/`import`/`drift`.
Each phase gets its own implementation plan; v0.2 is planned and built first because the registry it establishes underpins v0.3/v0.4.

## Cross-cutting (unchanged contracts from v0.1)
- Output `-o table|json|yaml`, `-q`; stable exit codes 0 (ok) / 1 (assertion/target failure) / 2 (config/auth/API/usage). Errors to stderr (`{"error":…}` under `-o json`).
- Auth/config contexts, `--timeout`, `--verbose` (never logs Authorization), `--no-color` — all from v0.1.
- **Testing:** TDD, subagent-driven-development like v0.1. The registry is the single source for contract tests (every registered type must satisfy list/get/create/update/delete round-trip against a fake API).
- Built on `syschecks-go`; bump SDK dep as needed. Single static binary, goreleaser (Homebrew tap still deferred).

## Risks / notes
- SDK export/import are `json.RawMessage` today — we generate from typed list/get instead (richer, avoids the untyped blob). If a type's Create/Update signature is awkward, the registry absorbs the per-type quirk in one place.
- Nested types (oncall rotations, escalation steps, playbook steps) are `-f`-only in v0.2 (no flag explosion); flags cover the flat common fields.
