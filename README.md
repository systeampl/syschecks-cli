# syschecks

`syschecks` is an operational command-line client for [SysChecks](https://github.com/systeampl) monitoring: full CRUD over organizations, projects, checks, teams, services, on-call schedules, escalation policies, maintenance windows, playbooks, status pages, lifecycle watches, contact methods and integration keys, plus incident/agent actions and notification channels, plus client-side HTTP/DNS/TLS diagnostics — all from the terminal, scriptable in CI.

It is built on the [`syschecks-go`](https://github.com/systeampl/syschecks-go) SDK, which is the shared client underneath the CLI and (eventually) the Terraform/Ansible/Pulumi/Salt providers.

## Install

### curl (prebuilt binary)

Download the archive for your platform from the [Releases](https://github.com/systeampl/syschecks-cli/releases) page and place the `syschecks` binary on your `PATH`, e.g.:

```bash
curl -sL https://github.com/systeampl/syschecks-cli/releases/latest/download/syschecks_<version>_linux_amd64.tar.gz \
  | tar -xz syschecks
sudo mv syschecks /usr/local/bin/
```

### Homebrew

_Coming soon_ (a `systeampl/homebrew-tap` is planned). For now use the prebuilt binary above or `go install`.

### Go install

```bash
go install github.com/systeampl/syschecks-cli@latest
```

## Authentication

`syschecks` authenticates with a Personal Access Token (PAT). Either:

- export `SYSCHECKS_TOKEN` in the environment (preferred for CI), or
- store one under a named context:

```bash
syschecks auth login --with-token          # reads the token from stdin
syschecks auth login --with-token <token>  # or pass it as an argument
syschecks auth whoami
syschecks auth logout
```

Tokens are written to `$XDG_CONFIG_HOME/syschecks/token-<context>` (or `~/.config/syschecks/...`) with `0600` permissions and are never printed to stdout/stderr.

### Config contexts

`syschecks` uses kubectl-style named contexts (`~/.config/syschecks/config.yaml`) to hold an API URL and organization per environment:

```bash
syschecks config set-context prod --api-url https://api.syschecks.example.com --org acme
syschecks config use-context prod
syschecks config get-contexts
syschecks config current-context
```

Resolution precedence for a given invocation (each field resolves independently — see `internal/config/resolve.go`):

| Field     | Precedence (highest first)                                     |
|-----------|------------------------------------------------------------------|
| token     | `SYSCHECKS_TOKEN` env > `--token` flag > active context's token file |
| api_url   | `--api-url` flag > active context > `SYSCHECKS_API_URL` env       |
| org       | `--org` flag > active context                                     |

Note the token is the one field where the environment variable wins over the flag — this lets a CI job's `SYSCHECKS_TOKEN` env override a stray `--token` in a shared script without editing it.

## Commands

```
syschecks version
syschecks completion {bash|zsh|fish|powershell}

syschecks auth login --with-token [<token>]
syschecks auth logout
syschecks auth whoami

syschecks config set-context <name> --api-url <url> --org <org>
syschecks config use-context <name>
syschecks config get-contexts
syschecks config current-context

syschecks probe http <url> [--save --project <id> [--interval <dur>]] [--timeout <dur>]
syschecks probe dns <host> [--timeout <dur>]
syschecks probe tls <host:port> [--timeout <dur>]

syschecks verify --url <url> [--expect-status <code>] [--expect-json <gojq-expr>] [--timeout <dur>]
```

Global flags (available on every command): `-o, --output table|json|yaml`, `-q, --quiet`, `--no-color`, `--context`, `--org`, `--api-url`, `--token`, `--verbose`.

### CRUD resources (v0.2)

Every resource below follows the same shape:

```
syschecks <resource> list
syschecks <resource> get <id>
syschecks <resource> create [--<field> <value> ...] [-f <file.yaml>]
syschecks <resource> update <id> [--<field> <value> ...] [-f <file.yaml>]
syschecks <resource> delete <id> [--yes]
```

Only the subcommands the SysChecks API actually exposes for that resource are generated — there is never a leaf that would just fail on the wire. `--<field>` flags exist for every flat scalar field of that resource's create/update payload (run `syschecks <resource> create --help` for the full flag list — `check create` alone has ~70, one per check-type setting); nested/array fields (HTTP headers, escalation steps, rotation members, DNS expected IPs, ...) have no flag and are `-f`-only. `create`/`update` accept `--field` flags and/or `-f <file>` together: the file supplies a base document, then any flag that was actually passed overrides the same key — see [Hybrid input](#hybrid-input-flags--f) below.

| resource | list | get | create | update | delete | org scope | notes |
|---|---|---|---|---|---|---|---|
| `org` | ✓ | ✓ (by slug) | ✓ | ✓ | ✓ | none | `org get <slug>`, not an id |
| `project` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | |
| `check` | ✓ | ✓ (by id or name) | ✓ | ✓ | ✓ | optional `--org` | plus `run`/`pause`/`resume`/`test-alert` (below) |
| `notification` | ✓ | ✓ | ✓ | ✓ | ✓ | optional `--org` | plus `test` (below) |
| `team` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | |
| `service` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | `list` shows extra `health_status`/`checks_count` columns |
| `oncall-schedule` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | rotations are `-f`-only |
| `escalation-policy` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | steps are `-f`-only |
| `maintenance-window` | ✓ | ✓ | ✓ | ✓ | ✓ | optional `--org` | |
| `playbook` | ✓ | ✓ | ✓ | ✓ | ✓ | required `--org` | steps are `-f`-only |
| `status-page` | ✓ | ✓ | ✓ | ✓ | ✓ | none | |
| `lifecycle-watch` | ✓ | ✓ | ✓ (upsert) | – | ✓ | required `--org` | **no update**: `create` is an idempotent upsert keyed by `--vendor`/`--resource-type`/`--resource-id` — run it again with the same key to change notification settings |
| `contact-method` | ✓ | – | ✓ | ✓ | ✓ | none | **no get**: the API has no single-contact-method lookup |
| `integration-key` | ✓ | – | ✓ | – | ✓ (revoke) | required `--org` | **no get/update**: create-then-revoke only |

### `apply -f` (declarative, multi-doc)

```
syschecks apply -f <file.yaml>
syschecks apply -f -   # read the document(s) from stdin
```

`apply` accepts one or more YAML/JSON documents separated by a line containing exactly `---`. Each document must carry a `kind: <resource>` field (one of the resource names above); a document with an `id` is routed to that resource's `update`, one without is routed to `create`. `-f -` reads from stdin instead of a file, so `syschecks <resource> get <id> -o yaml` output can be piped straight into `apply` — a real get response doesn't carry a `kind` field yet, so a `kind: <resource>` line has to be added to the piped document first (e.g. with `yq` or a small wrapper script); once it does, `get -o yaml | apply -f -` reproduces the same `update` request body a direct `syschecks <resource> update <id>` call would have sent (see `internal/cli/roundtrip_test.go`, which asserts exactly this for `check` and `team`).

Example:

```yaml
kind: check
id: 42
url: https://example.com/health
---
kind: team
name: platform
```

### Hybrid input: flags + `-f`

`create`/`update` build their request body by loading `-f <file>` (if given) as the base document, then overlaying any `--field` flag that was actually passed on the command line — flags win over the file, and a field present only in the file still satisfies a `create`-time required field. This lets a script keep the bulk of a resource's config in a checked-in YAML file while overriding one value per invocation:

```
syschecks check update 42 -f check.yaml --interval 30
```

### Non-CRUD actions

```
syschecks check run <id|name> [--wait] [--timeout <dur>]
syschecks check pause <id|name>
syschecks check resume <id|name>
syschecks check test-alert <id|name>

syschecks notification test <id>

syschecks incident list [--status <status>]
syschecks incident get <check_id> <log_id>
syschecks incident acknowledge <check_id> <log_id> [--note <text>]   # alias: ack
syschecks incident resolve <check_id> <log_id>

syschecks agent list
syschecks agent token
syschecks agent delete <agent_id>   # alias: rm
```

Incidents are addressed by the `(check_id, log_id)` pair the API uses, not a single id, so they don't fit the generic CRUD factory. Agents have no `create`/`get`: they self-register against the API using the token `agent token` mints, and there is no single-agent lookup endpoint.

## Output and exit codes

Every command renders through the same output layer, selected with `-o`/`--output`:

- `table` (default) — tab-aligned columns; check/incident statuses are colour-coded when stdout is a terminal, off under `--no-color` or when piped
- `json` — the row set as a JSON array, `[]` when empty
- `yaml` — the row set as YAML
- `-q`/`--quiet` — only the first column (typically an id or name), one per line, for shell scripting

Exit codes follow a fixed contract so `syschecks` composes cleanly in CI:

| Code | Meaning |
|------|---------|
| `0`  | success |
| `1`  | an assertion/check failed on its own terms (e.g. `verify` got the wrong status, `check run --wait` settled DOWN) |
| `2`  | everything else: config, auth, API, or usage errors |

## `probe` and `verify` are client-side

Every client-side command is bounded by `--timeout` (30s by default), so an unresponsive target fails the job instead of hanging it.

`syschecks probe {http,dns,tls}` and `syschecks verify` talk directly to the target you point them at (via `net/http`, `net`, `crypto/tls`) — they do **not** go through the SysChecks API and need no configured context or token. `probe http --save` is the one exception: it uses the SDK to persist the probed URL as a monitored check, which does require a resolved context/token and `--project <id>`.

## Development

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

A smoke end-to-end script against a local dev backend lives at `test/smoke_test.sh` (see `docs/LOCAL_DEV.md` for bringing that backend up):

```bash
SYSCHECKS_TOKEN=<a local PAT> bash test/smoke_test.sh
```
