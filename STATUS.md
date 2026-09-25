# STATUS — Trello CLI

> Living plan + progress tracker. Spec: `SPEC.md` (authoritative). Keep this file current.

**Status:** Phase 1 & Phase 2 COMPLETE (M1–M8 APPROVED). Phase 3 completion part DONE (M9 APPROVED). Remaining: board management (Phase 3). Production fixes P1 (raw auth) and P2 (card labels) **DONE & APPROVED** — see "Production issue fixes".

## Non-Functional Requirements (NFRs)

Cross-cutting constraints every task must respect (from SPEC §1, §3, §4):

- **NFR-1 Error surfacing** — no swallowed errors (`_ = err`). Network-layer errors (DNS,
  TLS, timeout, reset/EOF) are wrapped with context and surfaced on stderr with a
  non-zero exit. HTTP 4xx/5xx print status code + response body.
- **NFR-2 Testability** — HTTP behind an interface; unit tests + `httptest` fixtures;
  no live network in tests.
- **NFR-3 Determinism** — stable output ordering; `--json` byte-identical for identical
  input; IDs emitted as raw strings; timestamps RFC3339.
- **NFR-4 Agent-friendliness** — `--json` on every data command; documented exit-code
  contract (0–8); no ANSI color when piped/non-TTY or `--no-color`.
- **NFR-5 Human-friendliness** — real `--help` with usage + examples on every command
  and subcommand; memorable command tree.
- **NFR-6 Completion-compatibility** — Cobra command tree kept dynamic-free so generated
  completions are accurate (scripts themselves are a Phase-3 deliverable).
- **NFR-7 Performance** — one HTTP request per invocation where possible; cheap config
  resolution; no global mutable state.
- **NFR-8 Timeouts** — configurable request timeout (default 15s) via context; clear
  timeout error.
- **NFR-9 Code quality** — Go 1.23+, idiomatic; `go vet` and `go test -race ./...`
  clean; no unused/blank imports.

## Task breakdown

| ID  | Task | Milestone | Status |
|-----|------|-----------|--------|
| T0  | Scaffolding: go.mod, `cmd/trello/main.go`, root Cobra cmd + real help, global flags, exit-code plumbing, package layout | M1 | done |
| T1  | Output layer (`internal/output`): human table + JSON writers, TTY/color detection, `--no-color`, RFC3339/raw-ID conventions, exit-code type + mapper | M1 | done |
| T2  | HTTP client (`internal/trello`): `Client`, `HTTPClient` interface, `Do()`, key/token injection, error wrapping, status→typed-error mapping, typed response structs, context timeout | M1 | done |
| T3  | Config (`internal/config`): YAML, project-local walk-up (per-key nearest-wins), global `os.UserConfigDir()`, credential/board precedence, unknown-key warnings, resolved view (never expose token) | M1 | done |
| T4  | `whoami` command (end-to-end vertical slice: creds → `GET /members/me` → render) | M1 | done |
| T5  | Board resolution (exact name/id/shortLink) + `board list` | M1 | done |
| T6  | `config init|show|path` (interactive board pinning; show resolved values + source) | M1 | done |
| T7  | `list` commands (list/get/create/update/archive) | M2 | done |
| T8  | `card` commands (list/get/create/update/move/archive/delete) | M2 | done |
| T9  | `comment` commands (add/list) | M3 | done |
| T10 | `checklist` commands (list/get/create/add-item/check-item) | M3 | done |
| T11 | `search` | M3 | done |
| T12 | `raw <method> <path> [-q k=v]` passthrough | M3 | done |
| T13 | Polish: help+examples on every command, golden tests, `go vet` + `staticcheck` + `-race` clean, acceptance pass (SPEC §9) | M4 | done |
| T14 | `label` commands (list/get/create/update/delete on board; add/remove label on card) | M5 | done |
| T15 | `member` commands (list board members, get) | M5 | done |
| T16 | `attachment` commands (list/add url+file/delete on card) | M6 | done |
| T17 | `customfield` commands (list board custom fields; set value on card) | M6 | done |
| T18 | `action` commands (list on board/card, get) | M7 | done |
| T19 | `notification` commands (list; mark read) | M7 | done |
| T20 | `org` commands (list, get, boards, members) | M8 | done |
| T21 | `completion bash|zsh|fish` generation (replace Cobra default) | M9 | done |
| T22 | `completion install <shell>` helper (write script to shell completion dir) | M9 | done |

## Milestones

- **M1 — Foundation (T0–T6):** runnable vertical slice — scaffold, client, config,
  output, `whoami`, `board list`, `config init|show|path`. Proves the full stack
  (creds + config resolution + API + output + exit codes).
- **M2 — Core data (T7–T8):** lists + cards.
- **M3 — Remaining features (T9–T12):** comments, checklists, search, raw.
- **M4 — Polish (T13):** help/examples, golden tests, lint/race clean, acceptance.
- **M5 — Labels + Members (T14–T15).**
- **M6 — Attachments + Custom Fields (T16–T17).**
- **M7 — Actions + Notifications (T18–T19).**
- **M8 — Organizations (T20).**
- **M9 — Completion (T21–T22):** `completion bash|zsh|fish` + `completion install <shell>`.

## Current status

- **M1:** APPROVED (reviewer) — T0–T6 complete; build/vet/race clean; all review items closed.
- **M2:** APPROVED (reviewer) — T7–T8 complete; exit-code contract holds end-to-end.
- **M3:** APPROVED (reviewer) — T9–T12 complete; exit-code contract holds for all command classes.
- **M4:** APPROVED (reviewer) — T13 polish + acceptance complete; build/vet/staticcheck/race clean; SPEC §9 acceptance codified.
- **Phase 1 MVP: DONE.**
- **Phase 2: COMPLETE** — M5 (labels + members) **APPROVED**; M6 (attachments + custom fields) **APPROVED**;
  M7 (actions + notifications) **APPROVED**; M8 (organizations) **APPROVED**.
  Niche long-tail (enterprises, plugins, batch, exports, stickers, card covers, emoji,
  member personal prefs) is served by `raw`.
- **Phase 3:** completion part **APPROVED** (M9 — `completion bash|zsh|fish` + `completion install <shell>`);
  board management (`board get/create/update/delete`) still deferred per SPEC §2.

## Post-completion hardening (real-board validation)

Found and fixed during live testing against a real Trello account (all read-only):

1. **Missing-credential error** now names which credential is missing (key/token/both)
   instead of a vague "credentials not found" — `internal/cli/root.go`.
2. **Empty search groups** now render their header + `(no results)` instead of silently
   printing nothing — `internal/cli/search.go`.
3. **Search `--json`** now always emits all four group keys as arrays (`[]`), never
   `null`, for a stable agent-facing schema — `internal/trello/search.go`
   (custom `UnmarshalJSON` normalizes missing/`null` groups).

All fixes covered by regression tests; build/vet/staticcheck/race clean.

## Phase 2 playground validation (board `z7ynt1JB`)

Full-feature test (read + write + cleanup) against a dedicated playground board found
and fixed three bugs (all verified live + regression-tested):

1. **`checklist check-item` wrong endpoint (404)** — now does `GET /checklists/{id}`
   (learn `idCard`) → `PUT /cards/{idCard}/checkItem/{idCheckItem}?state=…`
   (was `PUT /checklists/{id}/checkItems/{idCheckItem}`). `Checklist` gained `idCard`.
2. **`card create/move/update/list --list` now accept a 24-hex list id directly**
   (no `--board` required); names still resolve against the board. `IsTrelloID` exported.
3. **`card get` renders `Desc:`** (omitted when empty).

Also noted (not code bugs): `config init` persists key/token to `.trello.yaml` (0600)
by design — add a `.gitignore` to avoid committing secrets; Custom Fields returned 403
because the Power-Up is disabled on that board.

Final: 385 tests passing; build/vet/staticcheck/race clean.

## Production issue fixes (2026-09-25)

Two defects surfaced during live use (headlines-project board reconciliation).
Both are fixed with regression tests and reviewer sign-off (APPROVED).

| ID | Issue | Root cause | Fix | Status |
|----|-------|-----------|-----|--------|
| P1 | `raw` passthrough returns HTTP 401 `invalid key` for REST-shaped paths while native verbs succeed under identical credentials | `Client.do` builds `baseURL+path`, then blindly appends `?`+encoded query. `key`/`token` are always injected, so a path that already contains `?` (e.g. `/cards/x?fields=labels`) becomes `…?fields=labels?key=…&token=…` — the credentials are swallowed into the first param's value. Native verbs never embed `?`, so they are unaffected. | `Client.do` now `url.Parse`s `baseURL+path`, merges any embedded query with the caller query, and sets `key`/`token` last (credentials win); `raw` strips a leading `/1` REST version prefix. | **done** |
| P2 | Card read output omits labels entirely, so label attachment cannot be verified through the CLI | `cardFields` (`cards.go`) excludes `labels` and `Card` (`types.go`) has no `Labels` field. | `Card` gains `Labels []Label json:"labels,omitempty"`; `labels` is requested on card reads **and** writes; human output adds a `Labels` row / `LABELS` column and `--json` carries the array; card goldens regenerated. | **done** |

**Verification (reviewer):** `gofmt -l .` clean; `go build ./...`, `go vet ./...`,
`go test -race -count=1 ./...` all pass (cli/config/output/trello ok); `staticcheck`
not installed. The reviewer independently reverted the production files to HEAD
(keeping the new tests) and confirmed every new test fails against the old code
(P1: `key=""`, `fields="labels?key=k"`; P2: `fields` assertion), then restored the
tree. No typed-call regressions; query encoding sorted (deterministic); goldens
stable and only the four card goldens changed.

Residual/optional follow-ups (none blocking):
- `normalizeRawPath` does not strip `/1?query` (e.g. `/1?fields=x`); realistic forms
  (`/1`, `/1/`, `/1/cards/x?…`) work. Help wording slightly overpromises.
- `fields=cardFields` is sent on card writes where `trello-api.json` doesn't declare
  it; Trello ignores unknown query params and its default card serialization already
  includes `labels`, so this is almost certainly harmless. A live smoke of
  `card create/update/move/archive` would close the question.
- Pre-existing (not a regression): `NetworkError` text can include the full request
  URL with `key`/`token` on transport failures. Worth redacting in a separate change.

Constraints: credentials are never logged/inlined (read from config per call); keep `go vet` / `staticcheck` / `go test -race ./...` clean; no live network in tests.

## Polish backlog (for M4)

- `card update --due <null|"">` — allow clearing a due date (currently rejects non-RFC3339). — **DONE (M4)**: `--due null`/`--due ""` sends `due=null`.
- `config init --board <ref>` — consider validating the ref via `ResolveBoard` when creds present (currently persisted unvalidated). — **DONE (M4)**: validated when creds present (persists resolved id); warn + persist as-is without creds.
- (No action) `PUT /cards/{id}/idList` is the canonical move endpoint but isn't declared in trello-api.json — hand-written client already handles it.

## How the loop works

For each milestone: coder implements → reviewer reviews against SPEC + NFRs → iterate
until reviewer is satisfied → advance. STATUS.md updated after each milestone.
