# End-to-End: `syschecks generate` Acceptance Test (Terraform/OpenTofu)

This runbook describes how to validate that the `syschecks generate` command produces correct, adoptable infrastructure-as-code. It is the acceptance test for the `generate` feature (Phase 1: `check`, `notification-channel`, `team` resources).

## Prerequisites

- **`syschecks` binary v0.3 or later**, installed and available on `PATH`
- **`terraform` or `tofu` (OpenTofu)** installed and available on `PATH`
  - Either one works; `syschecks generate terraform` and `syschecks generate opentofu` produce byte-identical output
  - `tofu` is preferred for this runbook (substitute `terraform` if using the Terraform CLI instead)
- **API credentials:**
  - A valid `SYSCHECKS_API_URL` (e.g., `https://api.example.com`)
  - A valid `SYSCHECKS_TOKEN` (a Personal Access Token with read permission)
  - Either set as environment variables or stored in a named context (see `syschecks auth login --help` and `syschecks config set-context --help`)
- **A target SysChecks account** (ideally a throwaway org for testing):
  - At least one **check** resource (any type; HTTP/DNS/TLS all work)
  - At least one **notification channel** (with at least one secret field like `webhook_url`)
  - At least one **team** resource
  - These will be read (not modified) and adopted into Terraform state

## Steps

### 1. Generate HCL from live resources

```bash
syschecks generate terraform --org <org> --out ./iac
```

Replace `<org>` with the slug or name of your target organization. This will:
- Read all live `check`, `notification-channel`, and `team` resources in the organization
- Create the directory `./iac` if it doesn't exist
- Write four files:
  - `provider.tf` — Terraform provider configuration (fixed, same every run)
  - `checks.tf` — one `resource "systeam_check"` block per live check
  - `notification_channels.tf` — one `resource "systeam_notification_channel"` block per live notification channel
  - `teams.tf` — one `resource "systeam_team"` block per live team
  - `imports.tf` — one `import` block per generated resource (links live id to local resource name)
  - `variables.tf` — sensitive variable declarations for secret fields (only if secrets were found)

The command will print a `WARNING` listing all variables it expects you to supply, e.g.:
```
WARNING: set the following sensitive variables before running plan/apply:
  - TF_VAR_slack_webhook_url (notification_channels.tf line 5)
  - TF_VAR_pagerduty_token (notification_channels.tf line 12)
```

### 2. Navigate to the generated directory

```bash
cd ./iac
```

### 3. Set sensitive variables

For each secret the generator warned about, set a `TF_VAR_<name>` environment variable with its **live value** from your SysChecks account. For example:

```bash
export TF_VAR_slack_webhook_url="https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXX"
export TF_VAR_pagerduty_token="<your-pagerduty-api-key>"
```

Alternatively, create a `terraform.tfvars` file (note: `.gitignore` this file in production):

```hcl
slack_webhook_url = "https://hooks.slack.com/services/..."
pagerduty_token   = "<your-api-key>"
```

If you want to dry-run this without the real secrets, you can use any placeholder string for each variable (e.g., `export TF_VAR_slack_webhook_url="placeholder"`); the plan will still validate HCL correctness, though it won't match live state for secret-carrying resources.

### 4. Initialize the Terraform working directory

```bash
tofu init
```

This downloads the `systeampl/systeam` provider from the Terraform Registry and sets up the local state backend.

### 5. Run the acceptance test: plan

```bash
tofu plan
```

## Acceptance Criteria

The plan **must** report:

```
No changes. Your infrastructure matches the configuration.
```

This means:
- **0 to add** (no resources the generator missed)
- **0 to change** (no attributes diverge between live state and generated config)
- **0 to destroy** (no stray resources in state)

### Why a no-op plan proves correctness

The `generate` command produces:
1. **Resource blocks** with live attributes rendered as HCL literals or variable references
2. **`import` blocks** that adopt each live resource into Terraform state by its live id

When you run `tofu plan`:
- `tofu init` pulled the provider and created an empty state
- `tofu import` (triggered by the import blocks) hydrated state with the live resources
- `tofu plan` compared the live-state values to the generated HCL
- If the generated HCL matches live state exactly, the plan reports no changes

A no-op plan is the strongest proof that the generator is correct: it means every resource was found, every attribute was rendered, and no values diverge.

## Troubleshooting Plan Outcomes

If the plan is **not** a no-op, check the plan output to see what it wants to change. This table maps plan outcomes to likely generator issues:

| Plan outcome | Likely cause | Fix |
|---|---|---|
| **Create new resource** (e.g., `+ resource "systeam_check" "example"`) | Missing or incorrect `import` block in `imports.tf` for this resource | Check that `imports.tf` includes an `import` block for this resource's type and id. Verify the `id` matches the live resource id (use `syschecks check get <name>` to confirm). Rerun `tofu init && tofu plan`. |
| **Change attribute** (e.g., `~ url = "old" -> "new"`) | Generated HCL attribute diverges from live value | (a) If the live value changed after generation, rerun `syschecks generate ...` to re-fetch. (b) If the generated value is wrong, it's a generator bug — check `checks.tf` / `notification_channels.tf` / `teams.tf` for a mismatch and report it. (c) If an attribute is marked `Computed` in the provider schema, it may be set by the server on create/read; omit it from generated HCL or accept the diff. |
| **Replace resource** (e.g., `~ resource "systeam_check" "x" [delete then create]`) | Resource id mismatch or a required attribute changed | (a) Check that the `id` in the import block matches the live resource id. (b) Verify that required attributes (e.g., `type` for a check) are present and correct in the resource block. (c) If a required field was omitted by the generator, it's a generator bug. |
| **Destroy resource** (e.g., `- resource "systeam_check" "example"`) | An import block is missing for a live resource | Check that `imports.tf` includes an import block for every live resource. Rerun `syschecks generate ...` and inspect `imports.tf`. |

## Phase 1 Scope

This test covers only the resources available in Phase 1:
- `check`
- `notification-channel`
- `team`

Other resource types (e.g., `service`, `escalation-policy`, `on-call-schedule`, `maintenance-window`, etc.) are not generated in Phase 1 and will not appear in the generated HCL. If your test organization contains other resources, **this is expected and not a failure** — a mixed org with un-generated types will legitimately have HCL only for the three Phase-1 types.

To validate only a subset of resources (e.g., only checks), use the `--type` flag:

```bash
syschecks generate terraform --org <org> --type check --out ./iac
```

Or limit by project or check id:

```bash
syschecks generate terraform --org <org> --project <project_id> --out ./iac
syschecks generate terraform --org <org> --check <id1>,<id2> --out ./iac
```

## Read-Only Safety

`syschecks generate` **never** creates, updates, or deletes any resources. It only calls the read (`list` / `get`) side of the API. It is always safe to point at a live, production account.

## Debugging

If the plan fails for reasons unrelated to the generator (e.g., provider download, credential issues, syntax errors in generated HCL), check:

1. **Provider availability**: Verify the `systeampl/systeam` provider is available in the Terraform Registry (or your private registry).
2. **Provider version**: The generated `provider.tf` pins a version; check that your registry has that version published.
3. **Credentials**: Ensure `SYSCHECKS_TOKEN` and `SYSCHECKS_API_URL` are set and valid. Run `syschecks auth whoami` to verify.
4. **HCL syntax**: Run `tofu fmt -check ./iac` to detect formatting issues (though the generator should produce valid syntax).
5. **Generated files**: Inspect `checks.tf`, `notification_channels.tf`, `teams.tf`, `imports.tf`, `variables.tf` by hand to spot rendering errors (attribute names, types, missing fields, etc.).

## Example: Full walkthrough

```bash
# 1. Generate HCL
syschecks generate terraform --org my-test-org --out ./iac

# Output:
# Generated 2 checks, 1 notification_channel, 1 team.
# WARNING: set the following sensitive variables before running plan/apply:
#   - TF_VAR_webhook_url (notification_channels.tf line 5)

# 2. Move to the generated directory
cd ./iac

# 3. Set secrets
export TF_VAR_webhook_url="https://hooks.slack.com/services/..."

# 4. Init
tofu init
# Initializing the backend...
# Downloading hashicorp/null v3.2.2...
# ...

# 5. Plan (the acceptance test)
tofu plan
# data.systeam_provider_schema.main: Refreshing state...
# systeam_check.http_probe: Refreshing state... [id=42]
# systeam_check.dns_probe: Refreshing state... [id=43]
# systeam_notification_channel.slack_alerts: Refreshing state... [id=1]
# systeam_team.platform: Refreshing state... [id=5]
#
# No changes. Your infrastructure matches the configuration.

# Success! The generator is correct.
```
