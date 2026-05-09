# Monocle

Local CLI/TUI review companion for AI coding agents. Everything is local: Unix sockets plus SQLite, not a web service, REST API, auth flow, or deployment.

## Commands

- `make build` builds `bin/monocle` with the `VERSION` ldflag.
- `make run` builds, then launches the TUI.
- `make test` is the full suite: temp `XDG_CONFIG_HOME`, then `go tool gotestsum -- -count=1 -cover -p 1 ./...`.
- Focused test: `tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT; XDG_CONFIG_HOME="$tmp" go test ./internal/core -run TestName -count=1`.
- `make lint` only runs `go vet ./...` and `go build ./...`; there is no golangci/staticcheck config.
- `make format` requires `golines` on `PATH`; it runs `golines -w --no-reformat-tags` before `gofmt`.
- `make check` runs `trivy fs --scanners vuln,secret,misconfig .`; do not treat it as normal lint.
- Avoid `make deploy` unless asked; it copies to hard-coded `/home/djipey/.local/bin/monocle`.

## Runtime Architecture

- `cmd/monocle` is one binary: TUI, headless `serve`, agent-facing `review` commands, registration, hooks, and hidden MCP server.
- `monocle` is a thin TUI client. It auto-spawns an owned `monocle serve` or attaches to a manually-started one; `serve` owns the engine, SQLite DB, sessions, feedback queue, and Unix socket.
- Socket pairing is repo-root based: default `/tmp/monocle-<12-char-sha>.sock`; override with `MONOCLE_SOCKET` or `--socket`, and use `-C/--workdir` when the agent/TUI working dirs differ.
- `core.EngineAPI` (`internal/core/engine.go`) is the TUI boundary; keep TUI code against that interface. Socket-backed clients live in `internal/client`.
- SQLite defaults to `$XDG_DATA_HOME/monocle/monocle.db` or `~/.local/share/monocle/monocle.db`; `MONOCLE_DB` overrides it.
- Config loads global `~/.config/monocle/config.json` first, then project `.monocle/config.json`; `SaveConfig` writes the global config.

## Package Map

- `internal/core`: engine, git/directory clients, formatter, feedback queue, socket server.
- `internal/protocol`: NDJSON socket message types.
- `internal/client`: socket client and `EngineAPI` proxy used by TUI/CLI clients.
- `internal/mcp`: Go MCP server; tool descriptions come from embedded `tools.json`.
- `internal/adapters`: agent registration; slash-command definitions come from embedded `commands.json`.
- `skills/`: source `SKILL.md` files copied by registration.
- `internal/tui`: Bubble Tea v2 UI; `internal/tui/register` is the registration wizard.
- Ignored `.opencode/`, `.claude/`, `.codex/`, `.gemini/`, `.agents/`, and `plugins/` are generated registration artifacts, not primary source.

## Agent Integration

- Agent-facing CLI commands are `monocle review status`, `get-feedback`, `send-artifact`, and `add-files`; `--wait` blocks for reviewer feedback.
- MCP tool names are `review_status`, `get_feedback`, `send_artifact`, and `add_files`.
- `monocle register` defaults Claude to MCP tools + channels; OpenCode, Codex, and Gemini default to skills. `--integration-mode mcp|skills` overrides this.
- When Monocle itself is running for review, send plans/artifacts with `/review-plan` or `/review-plan-wait`; normal changed files are picked up automatically.

## Go/TUI Gotchas

- Use Bubble Tea v2 imports (`charm.land/bubbletea/v2`): `View() tea.View`, `tea.KeyPressMsg`, non-generic `tea.Program`, and `tea.Quit` as a command function.
- `tea.KeyPressMsg.String()` returns `"esc"` and `"enter"`; do not check `"escape"` or `"return"`.
- Set alt-screen on the returned view (`v.AltScreen = true`), not via old program options.
- Use Lipgloss v2 (`charm.land/lipgloss/v2`); `lipgloss.Color()` returns `color.Color`, it is not a type.
- Keep terminal styling mostly 16-color ANSI. Nerd Font icons can measure wider than `lipgloss.Width()`, so follow existing `iconSlack` compensation.

## Docs Sync

- Keybindings: update `internal/tui/keys.go`, `internal/tui/help.go`, README keybinding table, `docs/reference/keybindings.mdx`, and `docs/configuration/keybindings.mdx`.
- Config: update `internal/types/config.go`, `internal/core/config.go`, README config section, and `docs/configuration/config-file.mdx`.
- CLI commands/flags: update `cmd/monocle/main.go`, README CLI sections, `docs/reference/cli.mdx`, and/or `docs/reference/agent-commands.mdx`.
- Skills or supported agents: update `skills/`, `internal/adapters/`, README/docs, and embedded definitions (`internal/adapters/commands.json`, `internal/mcp/tools.json`) as needed.
- Docs under `docs/` follow `docs/AGENTS.md`: use lowercase `monocle` in prose, document CLI/TUI behavior only, and do not document deprecated `install`/`uninstall` or hidden `serve-mcp-channel`.

## Conventions

- Wrap errors with context: `fmt.Errorf("description: %w", err)`.
- Tests are white-box and co-located; DB tests use `:memory:`, git tests use `setupTestRepo(t)`, and filesystem/config tests should isolate with `t.TempDir()` and `t.Setenv()`.
- DB schema lives in `internal/db/schema.go`; bump `schemaVersion` when changing `schemaSQL` and add query tests. Current migrations recreate stale schemas, so do not assume persisted backwards compatibility unless requested.
- Commit hooks run gitleaks and enforce conventional commit subjects (`feat:`, `fix:`, `docs:`, etc.).
