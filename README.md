# syschecks

`syschecks` is an operational command-line client for [SysChecks](https://github.com/systeampl) monitoring: manage organizations, projects, checks, incidents, agents and notification channels, plus run client-side HTTP/DNS/TLS diagnostics — all from the terminal, scriptable in CI.

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

```bash
brew install systeampl/tap/syschecks
```

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

Resolution precedence for a given invocation: `--flag` > active context > environment variable (`SYSCHECKS_API_URL`, `SYSCHECKS_TOKEN`) > token file.

## Commands (v0.1)

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

syschecks org list
syschecks org get <slug>

syschecks project list
syschecks project get <id>

syschecks check list
syschecks check get <id|name>
syschecks check run <id|name> [--wait] [--timeout <dur>]
syschecks check pause <id|name>
syschecks check resume <id|name>
syschecks check test-alert <id|name>

syschecks incident list [--status <status>]

syschecks agent list

syschecks notification list
syschecks notification test <id>

syschecks probe http <url> [--save --project <id> [--interval <dur>]]
syschecks probe dns <host>
syschecks probe tls <host:port>

syschecks verify --url <url> [--expect-status <code>] [--expect-json <gojq-expr>]
```

Global flags (available on every command): `-o, --output table|json|yaml`, `-q, --quiet`, `--no-color`, `--context`, `--org`, `--api-url`, `--token`, `--verbose`.

## Output and exit codes

Every command renders through the same output layer, selected with `-o`/`--output`:

- `table` (default) — tab-aligned columns
- `json` — the row set as a JSON array
- `yaml` — the row set as YAML
- `-q`/`--quiet` — only the first column (typically an id or name), one per line, for shell scripting

Exit codes follow a fixed contract so `syschecks` composes cleanly in CI:

| Code | Meaning |
|------|---------|
| `0`  | success |
| `1`  | an assertion/check failed on its own terms (e.g. `verify` got the wrong status, `check run --wait` settled DOWN) |
| `2`  | everything else: config, auth, API, or usage errors |

## `probe` and `verify` are client-side

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
