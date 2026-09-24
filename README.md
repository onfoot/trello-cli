# trello

A command-line client for the [Trello REST API](https://developer.atlassian.com/cloud/trello/rest/), built for humans and agents.

- **Multi-board** — a global `--board` flag or a project-local config file selects the board, so the flag is optional.
- **Scriptable** — every command supports `--json`, documented exit codes, and no ANSI color when piped.
- **Shell-completable** — `trello completion bash|zsh|fish` and `trello completion install`.

## Requirements

- Go 1.23+ (to build from source)
- A Trello API key + token (see [Authentication](#authentication))

## Install / build

```sh
# Install the latest release into $GOBIN (requires the Go toolchain):
go install github.com/nomadicworks/trello-cli/cmd/trello@latest

# Or build from source:
git clone https://github.com/nomadicworks/trello-cli
cd trello-cli
go build -o trello ./cmd/trello
```

Verify:

```sh
trello whoami
```

## Authentication

Trello uses an API key + token. Create them at <https://trello.com/power-ups/admin>
(grant the token `read,write` scope).

Provide them in any of three ways (highest precedence first):

1. Flags — `trello --key <KEY> --token <TOKEN> …`
2. Environment — `TRELLO_API_KEY`, `TRELLO_TOKEN`
3. Config file — `key:` / `token:` (see [Configuration](#configuration))

## Configuration

The CLI reads an optional YAML file, resolved by walking from the current directory
up to `$HOME` (nearest wins), plus a global file:

- Project-local: `.trello.yaml` (or `.trello.yml`)
- Global: `<os user config dir>/trello/config.yaml`
  - Linux: `~/.config/trello/config.yaml`
  - macOS: `~/Library/Application Support/trello/config.yaml`
  - Windows: `%AppData%\trello\config.yaml`

```yaml
# .trello.yaml
board: My Board          # exact name, id, or shortLink
key: 3b7bd3…             # optional
token: 9eb76d…           # optional — don't commit this
```

Pin the current project to a board (and optionally store credentials):

```sh
trello config init --board "My Board"
trello config show        # resolved values + their source (token never printed)
trello config path
```

## Usage

```
trello [global flags] <command> [args]

Global flags:
  -b, --board    board to operate on (exact name, id, or shortLink)
      --key      Trello API key
      --token    Trello API token
      --json     emit machine-readable JSON
      --no-color disable colored output
      --timeout  HTTP request timeout (default 15s)
```

| Domain | Commands |
| --- | --- |
| Boards | `board list` |
| Lists | `list list get create update archive` |
| Cards | `card list get create update move archive delete` |
| Comments | `comment add list` |
| Checklists | `checklist list get create add-item check-item` |
| Labels | `label list get create update delete add remove` |
| Members | `member list get` |
| Attachments | `attachment list get add delete` |
| Custom fields | `customfield list get set create delete` |
| Actions | `action list get` |
| Notifications | `notification list read read-all` |
| Organizations | `org list get boards members` |
| Search | `search <query>` |
| Auth | `whoami` |
| Config | `config init show path` |
| Raw API | `raw <METHOD> <path> [-q k=v]` |
| Completions | `completion bash zsh fish install` |

Examples:

```sh
trello whoami
trello board list
trello list list --board "My Board"
trello card create "Fix the bug" --list "To Do" --desc "…"
trello card list --json
trello comment add <card-id-or-shortlink> --text "shipped"
trello search "invoice" --cards
trello raw GET /members/me
```

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | generic/runtime error |
| 2 | usage/flag error |
| 3 | config error (missing/invalid/ambiguous board or config) |
| 4 | authentication failure (HTTP 401) |
| 5 | not found (HTTP 404) |
| 6 | validation/conflict (HTTP 400/422) |
| 7 | rate limited (HTTP 429) |
| 8 | network/connection error |

## Shell completion

```sh
# Print a script to stdout:
trello completion bash      # or zsh / fish

# Install it to the shell's completion directory:
trello completion install bash    # or zsh / fish
```

For bash/zsh you can also source on demand:

```sh
source <(trello completion bash)
```

## Development

```sh
go test -race ./...    # unit + golden + acceptance tests, no live network
go vet ./...
staticcheck ./...
```

See `SPEC.md` for the full specification and `STATUS.md` for the task plan and progress.
