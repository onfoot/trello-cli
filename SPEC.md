# Trello CLI — Project Specification

> Status: APPROVED for implementation planning.
> This document is the single source of truth for scope, requirements, and acceptance criteria.

## 1. Overview & Goals

A command-line client for the Trello REST API (`https://api.trello.com/1`), written in
modern Go, for power users and AI agents.

Primary goals:

1. **Reliable** — errors are never swallowed; connection/network failures are surfaced
   with actionable detail (status code, response body, cause).
2. **Testable** — business logic and the HTTP client are decoupled and unit-testable
   without a live Trello account.
3. **Multi-board** — operate against any of the user's boards, selected by an explicit
   argument or resolved automatically from a project-local config file.
4. **Human- and agent-friendly** — a memorable command tree, genuinely useful help
   (`--help` on every node), deterministic machine-readable output (`--json`), and
   correct exit codes.
5. **Shell-completable** — the command structure is designed to be
   completion-compatible from day one (Cobra), with bash/zsh/fish completion scripts
   delivered as a Phase-3 item (§2).

## 2. Scope

The upstream OpenAPI spec defines **261 endpoints**. v1 covers the core workflows only;
the long tail remains reachable via a generic passthrough so the CLI is never a dead end.

### Phase 1 — MVP (deliberately minimal)

| Domain    | Commands |
|-----------|----------|
| Boards    | list (discovery only — used to find the board to pin) |
| Lists     | list, get, create, update, archive |
| Cards     | list, get, create, update, move, archive, delete |
| Comments  | add, list (on a card) |
| Checklists | list, get, create, add-item, check-item |
| Search    | search |
| Config    | init, show, path |
| Auth      | whoami (validates key/token) |

> `config` is CLI infrastructure (required for the config-file requirement), not a
> Trello API domain. `whoami` is a single `GET /members/me` call kept purely to
> validate credentials. `board list` exists in Phase 1 solely so a user can find the
> id/shortLink of the existing board they want to pin to the current project — we
> assume boards already exist (creation/management is Phase 3).

### Phase 2 — deferred (explicitly out of MVP)

Labels, Members (list/get), Actions (full), Notifications, Webhooks, Custom Fields,
Attachments, Stickers, Card covers, Members' personal prefs/backgrounds/stars/
saved-searches, Organizations, Enterprises, Plugins, Emoji, Batch, Exports, Board
power-ups.

### Phase 3 — deferred

- **Board management** — `board get`, `board create`, `board update`, `board delete`.
- **Completions** — `completion bash|zsh|fish` generation + install helper.

> Any uncovered endpoint remains reachable via `trello raw <METHOD> <path> [--query k=v]`.

## 3. Functional Requirements

### 3.1 Authentication
- Auth = API key + token, both passed as query params (`key`, `token`) per the spec's
  `APIKey` + `APIToken` security schemes.
- Credential precedence (highest first):
  1. `--key` / `--token` flags
  2. `TRELLO_API_KEY` / `TRELLO_TOKEN` environment variables
  3. config file `key` / `token` (resolved per §3.3 hierarchy)
- CLI must produce a clear error if credentials are missing, and distinguish an
  explicit auth failure (HTTP 401) from other errors.

### 3.2 Board selection
- A global `--board` / `-b` flag accepts a board **exact name**, **id** (24-hex), or
  **shortLink**. Matching is exact (no prefix/substring guessing).
- Resolution precedence (highest first):
  1. `--board` flag
  2. project-local config `board` (walking up to `$HOME`, §3.3)
  3. global config `board` (§3.3)
- There is **no** board environment variable.
- If no board resolves where one is required, the command errors with a hint to
  run `trello board list` or set a default.
- No match for an explicit `--board` value is an error (exit 3), never a silent fallback.

### 3.3 Configuration files
- **Format:** YAML only.
- **Project-local:** `.trello.yaml` (alias `.trello.yml`), discovered by walking from
  the current working directory upward to `$HOME` (inclusive). For each key, the
  nearest ancestor file that defines the key wins (per-key nearest-wins).
- **Global (user) config:** `<UserConfigDir>/trello/config.yaml`, where
  `<UserConfigDir>` is Go's `os.UserConfigDir()` (honors the OS-default config path):
  - Linux: `~/.config/trello/config.yaml` (respects `$XDG_CONFIG_HOME`)
  - macOS: `~/Library/Application Support/trello/config.yaml`
  - Windows: `%AppData%\trello\config.yaml`
- **Keys:** `board`, `key`, `token`. Unknown keys must warn (typo guard), not fail.
- `trello config show` prints the *resolved* values (board; credential presence only —
  never the token itself) and the file each value came from — critical for
  agent/operator debugging.

### 3.4 Output modes
- Default: human-readable tables/columns, color auto-disabled when not a TTY.
- `--json` global flag: stable, documented JSON schema per command.
- `--no-color` global flag.
- All timestamps in RFC3339; IDs always emitted as raw strings (never lossy numbers).

### 3.5 Exit codes
| Code | Meaning |
|------|---------|
| 0    | success |
| 1    | generic/runtime error |
| 2    | usage/flag error (Cobra default) |
| 3    | config error (missing/invalid/ambiguous board or config) |
| 4    | authentication failure (HTTP 401) |
| 5    | not found (HTTP 404) |
| 6    | validation/conflict (HTTP 400/422) |
| 7    | rate limited (HTTP 429) |
| 8    | network/connection error (DNS, TLS, timeout, EOF) |

## 4. Non-Functional Requirements

1. **Error surfacing (hard requirement):** no `_ = err` anywhere. Network-layer errors
   (DNS, TLS handshake, timeouts, connection reset) are wrapped with context and
   surfaced to stderr with a non-zero exit. HTTP 4xx/5xx print status + response body.
2. **Testability:** HTTP transport abstracted behind an interface so tests use a
   `httptest` server or a fake transport; no live network in unit tests.
3. **Determinism:** output key ordering stable; JSON output identical across runs for
   identical input (agent-scriptable).
4. **Performance:** one HTTP request per CLI invocation where possible; config
   resolution is cheap; no global mutable state.
5. **Timeouts:** configurable request timeout (default 15s), with a clear timeout error.

## 5. Technology Stack

- **Language:** Go 1.23 or later.
- **CLI framework:** [spf13/cobra](https://github.com/spf13/cobra) (help text,
  subcommand tree, and completion generation out of the box) + `pflag`.
- **Config:** `gopkg.in/yaml.v3` for YAML parsing; `os.UserConfigDir()` for the global
  config location.
- **TTY detection:** `golang.org/x/term` (color suppression when piped/redirected).
- **HTTP:** stdlib `net/http` with a custom `*http.Client` (timeout, `context`).
- **Completions:** generated by Cobra for bash/zsh/fish; shipped as files + an
  install helper.
- **Client strategy:** hand-written, typed client for Phase-1 resources (deliberately
  not OpenAPI codegen, given the spec's 261 endpoints, inconsistent path-param
  declarations, query-param write convention, and stub schemas), plus a generic
  passthrough for everything else.
- **Binary/module:** binary `trello`; module `github.com/nomadicworks/trello-cli`.

## 6. API Client Design

- `Client` struct holds `Key`, `Token`, `BaseURL`, `HTTPClient` (interface), and
  request/response helpers.
- All requests go through a single `Do(ctx, method, path, query, body)` that:
  - injects `key`/`token` query params,
  - sets `Accept: application/json`,
  - checks status, decodes JSON, and wraps errors with status + body,
  - maps status → typed errors (`ErrAuth`, `ErrNotFound`, `ErrRateLimited`, …).
- Typed response structs for Board, List, Card, Member, Label, Checklist, CheckItem,
  Comment/Action (Phase-1 subset of the schema fields, documented).
- Query-param write convention honored (writes send params as query, per the spec).

## 7. Command Structure

```
trello                          # Phase 1
├── board list                  # (Phase 3: get|create|update|delete)
├── list  list|get|create|update|archive
├── card  list|get|create|update|move|archive|delete
├── comment add|list
├── checklist list|get|create|add-item|check-item
├── search
├── config init|show|path
├── whoami
└── raw <method> <path>         # passthrough
                                # Phase 3: completion bash|zsh|fish
```
Global flags: `--board/-b`, `--key`, `--token`, `--json`, `--no-color`, `--timeout`.

## 8. Testing Strategy

- Unit tests: config resolution (hierarchy, per-key nearest-wins, typo warnings),
  board resolution (exact name/id/shortLink, no-match error), flag parsing, output
  formatting, exit-code mapping, client error wrapping.
- HTTP tests: `httptest.Server` serving recorded fixtures; verify query-param
  injection, error mapping, JSON decoding.
- Golden tests for `--json` and human output (guard against accidental schema drift).
- `go vet` + `staticcheck` clean; CI runs `go test ./...` with `-race`.

## 9. Acceptance Criteria

For each Phase-1 command, a scriptable scenario passes:

1. `trello board list --json` returns valid JSON and exit 0.
2. Commands honor `--board` and config-file default identically.
3. Missing/invalid credentials → clear error, correct exit code, no panic.
4. `--help` on every command/subcommand prints real usage + examples (not stub text).
5. Network failure (unreachable host) → distinct connection error, exit 8.
6. HTTP 401/404/429 → mapped exit codes 4/5/7 with body surfaced.
7. `go test ./...` passes with no live network; race detector clean.
8. Piped output contains no ANSI color codes.
9. (Phase 3) `trello completion bash|zsh|fish` emits valid, installable scripts;
   completion resolves subcommands accurately.

## 10. Risks / Notes

- **Spec quirks:** many endpoints don't declare path params formally; writes use query
  params not JSON bodies; `Checklist` schema is a stub; pagination inconsistent.
  Mitigation: hand-written client + passthrough; don't rely on codegen.
- **Rate limiting:** spec has no explicit 429/retry contract. Mitigation: explicit
  `ErrRateLimited` + documented; no silent retry in MVP.
- **Scope creep:** 261 endpoints. Mitigation: strict Phase-1 gate (§2), passthrough
  covers the long tail.

## 11. Decisions (resolved)

1. **Scope** — Phase-1 command set: board `list` only; lists, cards, comments,
   checklists, search (+ `config`, `whoami`). Board management (get/create/update/
   delete) and completions moved to Phase 3; Labels and Members in Phase 2.
2. **CLI framework** — Cobra.
3. **Go version** — 1.23+ (default; bump if a later version is required).
4. **Config** — YAML only; project-local `.trello.yaml` walk-up + global
   `os.UserConfigDir()/trello/config.yaml` (covers Linux/macOS/Windows).
5. **Env vars** — `TRELLO_API_KEY` / `TRELLO_TOKEN` only; **no** board env var.
6. **Board matching** — exact name, id, or shortLink only.
7. **Output/exit codes** — as specified in §3.4 / §3.5.
8. **Completions** — bash, zsh, and fish (Phase 3 deliverable; command structure kept
   completion-compatible in Phase 1).
9. **Naming** — binary `trello`; module `github.com/nomadicworks/trello-cli`.
10. **Client** — hand-written typed client + passthrough (no codegen).
