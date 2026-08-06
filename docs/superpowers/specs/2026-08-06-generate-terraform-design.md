# syschecks CLI v0.3 — `generate terraform` design

**Date:** 2026-08-06
**Status:** approved (brainstorm), pre-plan
**Goal:** `syschecks generate terraform|opentofu` renders an account's live resources (read via `syschecks-go`) into HCL + `import{}` blocks that the published **`systeampl/systeam`** provider adopts on `plan` without recreating anything.

Scope of this phase: **Terraform + OpenTofu only** (one HCL renderer — identical output). Pulumi + Ansible = v0.3.x (separate plan). Builds on the v0.2 registry (read closures already exist).

## Command
```
syschecks generate terraform --org <org> [--project P] [--type check,notification-channel] [--check id,…] -o ./iac/
syschecks generate opentofu ...   # identical HCL; provider source resolves via the OpenTofu registry
```
Whole-org by default; filters narrow the set (incremental migration of an existing repo).

## Architecture
- `generate` iterates the in-scope resource types, calls each type's registry `listFn`/`getFn` (SDK typed → `map[string]any`), and feeds `(kind, attrs)` to a single **HCL renderer**.
- The renderer is driven by a per-resource **attribute spec** — the linchpin of correctness. For each `systeam_<type>` it declares: which attributes are emitted (the provider's Required+Optional set), each attribute's HCL type, whether it is a nested block, and whether it is **secret**.
- The attribute spec is **baked into the CLI** (`internal/generate/schema.go`), derived once from the `systeampl/systeam` provider source (its `schema.*Attribute` definitions). Only attributes the provider accepts are emitted → `plan` never errors on an unknown argument. Computed/read-only fields (`id`, `created_at`, `updated_at`, `status`, `uuid`, `last_*`) are excluded. Regeneration when the provider changes is a documented step (a small extractor over the provider repo).
  - Rationale: attribute names are snake_case = the SDK json keys (provider and SDK share the backend), so mapping is 1:1 by name; the spec's job is the allowlist + secret/nested flags, not renaming.

## Output layout (`-o <dir>`)
- `provider.tf` — `terraform { required_providers { systeam = { source = "systeampl/systeam", version = "~> 0.2" } } }` + an empty `provider "systeam" {}` (auth via env, matching the provider's config).
- `<type>.tf` per type — `checks.tf`, `notification_channels.tf`, `teams.tf`, … one `resource "systeam_<type>" "<label>" { … }` per resource.
- `imports.tf` — one `import { to = systeam_<type>.<label>, id = "<id>" }` per resource, so `plan` adopts existing state (no duplicate create).
- `variables.tf` — one `variable "<label>_<attr>" { type = string; sensitive = true }` per secret attribute referenced.

**Labels:** sanitize the resource's name/slug to a valid HCL identifier (`[a-z0-9_]`, lead-alpha), dedupe collisions with a numeric suffix. Stable within a run.

## Secrets
Attributes flagged `secret` in the spec (e.g. notification-channel `config.webhook_url`/tokens, check DB `auth_password`/`db_password`) are **never inlined**. Each is emitted as `var.<label>_<attr>`, declared `sensitive = true` in `variables.tf` with no default, and the renderer prints a stderr warning listing the variables the user must supply (`TF_VAR_...`). Safe to commit the generated HCL.

## Correctness proof (E2E)
Beyond unit tests (attrs map → expected HCL golden files), the real gate: generate against a live throwaway org, then `terraform init` + `terraform plan` (and `tofu plan`) and assert the plan is **"0 to add, 0 to change, 0 to destroy"** — i.e. every generated resource matches its imported live state. This is the same standard we held the providers to; a plan that wants to create/replace means the renderer or import wiring is wrong.

## Phasing within v0.3
- **Phase 1:** renderer + `provider.tf`/`imports.tf`/`variables.tf` + attribute spec for `check`, `notification-channel`, `team`; secrets handling; the `tofu plan == no-op` E2E on those three. Proves the whole pipeline.
- **Phase 2:** attribute spec + rendering for the remaining types (project, service, oncall-schedule, escalation-policy, maintenance-window, playbook, status-page, lifecycle-watch, contact-method, integration-key, organization), nested-block handling (rotations/steps/config), extend the E2E.

## Non-goals (v0.3)
- Pulumi/Ansible renderers (v0.3.x).
- Drift/`export` (v0.4).
- Rendering computed-only or action resources (`agent_registration_token`, `token`, `check_slo` if not a top-level provider resource in scope) unless trivially covered.

## Constraints / notes
- Built on the v0.2 registry read closures; no new SDK/backend work.
- `generate` is read-only against the account (list/get) + local file writes; never mutates the account.
- Output must be deterministic (stable ordering: resources sorted by id; attributes in spec order) so re-running produces a clean diff.
