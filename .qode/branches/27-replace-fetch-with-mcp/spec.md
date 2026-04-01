# Technical Specification — #27 MCP-based Ticket Fetch (Phase 1)

## 1. Feature Overview

Add `ticket_system.mode: mcp` to `qode.yaml` so that the `/qode-ticket-fetch` IDE slash command emits AI instructions (leveraging the IDE's configured MCP servers) instead of delegating to qode's built-in HTTP clients. When `mode: mcp` is set, `qode ticket fetch` becomes a no-op that prints a deprecation warning. All five existing provider implementations remain in place in Phase 1 — no breaking changes. The change unlocks rich ticket context (comments, attachments, linked Figma/Notion/GDocs) that today's text-only scrape silently drops, enabling more accurate AI-generated specs and plans.

**Business value:** Eliminates maintenance of five bespoke API integrations; enables richer AI context from MCP-native tools; aligns qode with the IDE MCP ecosystem direction.

**Success criteria:**
- Setting `ticket_system.mode: mcp` in `qode.yaml` causes `/qode-ticket-fetch` to emit an AI prompt (not a shell invocation) that writes structured context files
- `qode ticket fetch` with `mode: mcp` exits 0 with a deprecation warning — no ticket.md written
- All existing behaviour with `mode: api` (or no mode) is unchanged

---

## 2. Scope

### In scope
- Add `Mode string` field to `TicketSystemConfig` in `internal/config/schema.go`
- Deprecation no-op in `internal/cli/ticket.go` when `mode: mcp`
- Conditional slash command content in `internal/ide/claudecode.go` (`claudeSlashCommands`)
- Conditional slash command content in `internal/ide/cursor.go` (`slashCommands`)
- Regenerate `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc` via `qode ide sync`
- Tests: `internal/ide/ide_test.go` (slash command content), `internal/cli/ticket_test.go` (deprecation path)
- Docs: `docs/how-to-use-ticket-fetch.md`, `docs/qode-yaml-reference.md`, `README.md`

### Out of scope (Phase 1)
- Removing `internal/ticket/jira.go`, `azuredevops.go`, `linear.go`, `github.go`, `notion.go`
- Building a Go MCP client
- Changes to `internal/context/context.go` or prompt templates
- Making the context file structure configurable
- Workspace / multi-repo topology handling (feature is topology-agnostic)
- Changes to `qode check`, `qode review`, or `qode plan`

### Assumptions
- IDEs (Claude Code, Cursor) already have MCP servers configured by the user for their ticketing system
- `internal/context/context.go` already loads all files in `context/` into `ctx.Extra` (text) and `ctx.Mockups` (images) — no changes needed
- The `$ARGUMENTS` placeholder in slash commands is replaced with the actual URL at invocation time in both Claude Code and Cursor
- Default `Mode == ""` is treated identically to `mode: api`

---

## 3. Architecture & Design

### Component diagram

```
qode.yaml (ticket_system.mode: mcp | api)
    │
    ├── internal/config/schema.go     ← add Mode field
    │
    ├── internal/cli/ticket.go        ← early-exit + warning when mode==mcp
    │
    └── internal/ide/
            ├── claudecode.go         ← claudeSlashCommands(): branch on mode
            └── cursor.go             ← slashCommands(): branch on mode

Generated files (via qode ide sync):
    .claude/commands/qode-ticket-fetch.md     ← shell cmd (api) OR AI prompt (mcp)
    .cursor/commands/qode-ticket-fetch.mdc    ← AI prompt in both modes (different content)
```

### Slash command format by mode

| IDE | mode: api | mode: mcp |
|---|---|---|
| Claude Code | `!qode ticket fetch $ARGUMENTS` (shell, no AI) | Plain markdown prompt (AI reads + calls MCP) |
| Cursor | Instructional markdown: "run `qode ticket fetch $ARGUMENTS`" | Instructional markdown: "use MCP tools to fetch…" |

### Data flow — MCP mode

```
User: /qode-ticket-fetch https://linear.app/team/issue/ENG-123
    │
    └── IDE reads .claude/commands/qode-ticket-fetch.md (AI prompt)
            │
            └── AI executes prompt:
                    ├── calls MCP tool (Linear/Jira/GitHub/etc. server)
                    ├── fetches title, description, comments, links
                    └── writes:
                            context/ticket.md
                            context/ticket-comments.md  (if comments exist)
                            context/ticket-links.md     (if links exist)
```

### Data flow — API mode (unchanged)

```
User: /qode-ticket-fetch https://linear.app/team/issue/ENG-123
    │
    └── IDE runs shell: qode ticket fetch <url>
            │
            └── internal/cli/ticket.go
                    → ticket.DetectProvider(url, cfg.TicketSystem)
                    → provider.Fetch(url)
                    → writes context/ticket.md
```

---

## 4. API / Interface Contracts

### `TicketSystemConfig` (internal/config/schema.go)

**Before:**
```go
type TicketSystemConfig struct {
    Type       string     `yaml:"type,omitempty"`
    URL        string     `yaml:"url,omitempty"`
    ProjectKey string     `yaml:"project_key,omitempty"`
    Auth       AuthConfig `yaml:"auth,omitempty"`
}
```

**After:**
```go
type TicketSystemConfig struct {
    Mode       string     `yaml:"mode,omitempty"` // "" | "api" | "mcp"
    Type       string     `yaml:"type,omitempty"`
    URL        string     `yaml:"url,omitempty"`
    ProjectKey string     `yaml:"project_key,omitempty"`
    Auth       AuthConfig `yaml:"auth,omitempty"`
}
```

`Mode` is optional. Empty string and `"api"` are treated identically everywhere.

### `claudeSlashCommands` return value — `qode-ticket-fetch` key

**mode: api (unchanged):**
```
!qode ticket fetch $ARGUMENTS
```

**mode: mcp (new):**
```markdown
# Fetch Ticket via MCP — <project-name>

Fetch the ticket at the URL provided in $ARGUMENTS using your available MCP tools.

**Steps:**
1. Use whatever MCP tool is available to read the ticket (e.g. a Jira, Linear, GitHub,
   Notion, or Azure DevOps MCP server). If no MCP server is available, open the URL
   in a browser tool and extract the content.
2. Collect: title, description, all comments (with author names and timestamps), any
   linked resources (Figma, Notion docs, Google Docs, spreadsheets), and attachment
   summaries.
3. Write the following files, creating the directory if needed:

**context/ticket.md** — title, description, metadata

**context/ticket-comments.md** — all comments in chronological order
(Omit this file if there are no comments.)

**context/ticket-links.md** — linked resources with summaries
(Omit this file if there are no links.)

4. Report a one-line summary: "Fetched: <title> — <N> comments, <M> links."
```

### `slashCommands` return value — `qode-ticket-fetch` key (Cursor)

**mode: api (unchanged):**
```markdown
---
description: Fetch a ticket into branch context for <project-name>
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
```

**mode: mcp (new):**
```markdown
---
description: Fetch ticket via MCP for <project-name>
---

Fetch the ticket at the URL provided in $ARGUMENTS using your available MCP tools.

Use whatever MCP tool is available (Jira, Linear, GitHub, Notion, or Azure DevOps server).
If no MCP tool is available, open the URL with a browser tool and extract the content.

Collect: title, description, all comments (author + timestamp), linked resources, attachment
summaries.

Write to the context directory for the current branch:
- context/ticket.md — title, description, metadata
- context/ticket-comments.md — all comments (omit if none)
- context/ticket-links.md — linked resource summaries (omit if none)

Report: "Fetched: <title> — <N> comments, <M> links."
```

### `qode ticket fetch` CLI — behaviour change when `mode: mcp`

**Before:** Runs provider detection and fetch, writes `context/ticket.md`.
**After:** Prints deprecation warning to stderr, returns `nil` (exit 0). No file written.

Stderr output:
```
Warning: ticket_system.mode is "mcp".
Use /qode-ticket-fetch in your IDE to fetch tickets via MCP.
qode ticket fetch is a no-op when mode: mcp.
```

---

## 5. Data Model Changes

### `qode.yaml` — new optional field

```yaml
ticket_system:
  mode: mcp   # new; "" or "api" preserves existing behaviour
  type: linear
  ...
```

No migration needed — field is optional and defaults to existing behaviour.

### Context directory — new output files (written by AI in MCP mode)

| File | Content | Consumed by |
|---|---|---|
| `context/ticket.md` | Title, description, metadata | `ctx.Ticket` (existing field) |
| `context/ticket-comments.md` | All comments with author/timestamp | `ctx.Extra` (auto-loaded) |
| `context/ticket-links.md` | Linked resource summaries | `ctx.Extra` (auto-loaded) |

No schema or code changes to `internal/context/context.go`.

---

## 6. Implementation Tasks

- [ ] **Task 1** (config): Add `Mode string \`yaml:"mode,omitempty"\`` to `TicketSystemConfig` in `internal/config/schema.go`
- [ ] **Task 2** (cli): Add early-exit deprecation block in `internal/cli/ticket.go` `RunE` after config load; write 3-line warning to `os.Stderr`, return `nil`
- [ ] **Task 3** (ide): Wrap `qode-ticket-fetch` value in `claudeSlashCommands()` (`internal/ide/claudecode.go`) with `if cfg.TicketSystem.Mode == "mcp"` conditional returning the AI prompt; else return existing shell command
- [ ] **Task 4** (ide): Same conditional in `slashCommands()` (`internal/ide/cursor.go`)
- [ ] **Task 5** (test): Add `internal/ide/ide_test.go` tests: `TestClaudeSlashCommands_MCPMode_TicketFetch`, `TestClaudeSlashCommands_MCPMode_NoExclamationPrefix`, `TestCursorSlashCommands_MCPMode_TicketFetch`
- [ ] **Task 6** (test): Add `internal/cli/ticket_test.go` with `TestTicketFetch_MCPMode_IsNoop`: configure `mode: mcp`, call handler, assert exit 0, assert stderr contains warning, assert `context/ticket.md` not written
- [ ] **Task 7** (generated): Run `go run ./cmd/qode ide sync` and commit regenerated `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc`
- [ ] **Task 8** (docs): Update `docs/how-to-use-ticket-fetch.md` (add MCP section), `docs/qode-yaml-reference.md` (add `mode` field), `README.md` (mention MCP option in ticket system section)

**Task order:** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 (linear; each task depends on the prior schema/code being in place)

---

## 7. Testing Strategy

### Unit tests (`internal/ide/ide_test.go`)

| Test | Config | Assertion |
|---|---|---|
| `TestClaudeSlashCommands_MCPMode_TicketFetch` | `mode: mcp` | Content does not start with `!`; contains `$ARGUMENTS`; contains `context/ticket.md` |
| `TestClaudeSlashCommands_MCPMode_NoExclamationPrefix` | `mode: mcp` | `!strings.HasPrefix(content, "!")` for `qode-ticket-fetch` key |
| `TestCursorSlashCommands_MCPMode_TicketFetch` | `mode: mcp` | Contains `$ARGUMENTS`; contains `context/ticket.md`; contains `description:` frontmatter |
| `TestClaudeSlashCommands_ContainsTicketFetch` (existing) | `mode: api` / default | Still returns `!qode ticket fetch $ARGUMENTS` — no regression |
| `TestCursorSlashCommands_ContainsTicketFetch` (existing) | `mode: api` / default | Still contains `qode ticket fetch $ARGUMENTS` — no regression |

### Integration tests (`internal/cli/ticket_test.go`)

| Test | Setup | Assertion |
|---|---|---|
| `TestTicketFetch_MCPMode_IsNoop` | `cfg.TicketSystem.Mode = "mcp"`; temp dir | Returns `nil`; stderr contains `"no-op"`; `context/ticket.md` not created |
| `TestTicketFetch_APIMode_StillWorks` | `cfg.TicketSystem.Mode = "api"` (or `""`) | Proceeds to provider detection (may error on bad URL — that's expected) |

### Edge cases
- `mode: ""` (unset): treated as `api` — existing tests cover this
- `mode: "API"` (wrong casing): treated as api via `==` check — warn in docs, no special handling
- `context/` directory does not exist when AI writes: AI creates it (documented in prompt); `context.go` handles gracefully if dir absent

### Manual verification
- Set `mode: mcp` in local `qode.yaml`, run `qode ide sync`, confirm `.claude/commands/qode-ticket-fetch.md` contains the AI prompt (no `!`)
- In Claude Code with a Linear MCP server, invoke `/qode-ticket-fetch <url>`, confirm three context files are written

---

## 8. Security Considerations

- No authentication or authorisation changes. MCP credentials are managed by the IDE/MCP server configuration, not by qode.
- The `Mode` field is free-form string in YAML; invalid values (e.g. `mode: foo`) silently default to API mode. No injection risk — value is only compared with `==`.
- Prompt content written by AI to `context/` files is local filesystem only; no data leaves the machine via qode.
- No new environment variables or secrets.
- The deprecation warning path in `qode ticket fetch` does no network I/O when `mode: mcp` — reduces attack surface.

---

## 9. Open Questions

None. All ambiguities from requirements analysis are resolved:

1. **Phase scope** → Phase 1 only (no provider deletion).
2. **Default mode** → `""` = api; no breaking change.
3. **CLI behaviour in mcp mode** → no-op, exit 0, stderr warning.
4. **Slash command format** → Claude Code: `!` dropped, becomes AI prompt. Cursor: already AI prompt, content updated.
5. **MCP tool naming** → natural language ("use whatever MCP tool is available") to be provider-agnostic.
6. **Context file structure** → `ticket.md` + optional `ticket-comments.md` + optional `ticket-links.md`; all auto-loaded by existing `context.go`.

---

*Spec generated by qode. Copy to GitHub issue #27 for team review.*
