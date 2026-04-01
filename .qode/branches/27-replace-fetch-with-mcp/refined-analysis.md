<!-- qode:iteration=2 score=25/25 -->

# Requirements Analysis — #27 Replace qode ticket fetch with MCP server integration

## 1. Problem Understanding

**Restated:** `qode ticket fetch` ships five bespoke HTTP clients (Jira, Azure DevOps, Linear, GitHub, Notion in `internal/ticket/`) that each extract only a ticket's title and body into a flat `context/ticket.md`. This loses comments, attachments, linked designs (Figma, Notion docs, Google Sheets), and images — all of which may contain critical acceptance criteria. Production-ready MCP servers exist for all five systems and surface that richness natively via IDE tool calls. The goal is to:
1. Add `ticket_system.mode: mcp` to `qode.yaml` so users can opt in to MCP-based fetching
2. Update the `/qode-ticket-fetch` slash command (in both Claude Code and Cursor) to emit AI instructions when `mode: mcp`, instead of a shell command
3. Mark `qode ticket fetch` as a no-op with a deprecation warning when `mode: mcp`

**Phase 1 only** — internal provider files stay. Phase 2 (removal) is a follow-on PR.

**User need:** AI-assisted spec and plan generation requires the full ticket context (acceptance criteria often live in comments, designs in Figma links, screenshots in attachments). Today those are silently dropped.

**Business value:** Eliminates maintenance of five API integrations; enables richer AI context; aligns with IDE MCP ecosystem.

**Open questions resolved:**
- Default remains `mode: api` (empty string = api). No breaking change.
- CLI command in `mode: mcp` is a **no-op** (exit 0, warning to stderr). It does NOT run any MCP logic in Go.
- Claude Code slash command type changes: `!shell-command` → plain markdown prompt (no `!`).
- Scope: Phase 1 only. No provider file deletion.

---

## 2. Technical Analysis

### Affected components

| Component | File | Change |
|---|---|---|
| Config schema | `internal/config/schema.go` | Add `Mode string` to `TicketSystemConfig` |
| CLI command | `internal/cli/ticket.go` | Early-exit with deprecation warning when `mode == "mcp"` |
| Claude Code generator | `internal/ide/claudecode.go` | Conditional content in `claudeSlashCommands()` |
| Cursor generator | `internal/ide/cursor.go` | Conditional content in `slashCommands()` |
| Generated slash commands | `.claude/commands/qode-ticket-fetch.md` | Regenerated via `qode ide sync` |
| Generated slash commands | `.cursor/commands/qode-ticket-fetch.mdc` | Same |
| Docs | `docs/how-to-use-ticket-fetch.md` | Add MCP section; keep API section with deprecation note |
| Docs | `docs/qode-yaml-reference.md` | Document `mode` field |
| Docs | `README.md` | Update ticket system section |
| Tests | `internal/ide/ide_test.go` | MCP mode assertions (slash command content) |

**Not affected:**
- `internal/ticket/` — providers unchanged in Phase 1
- `internal/context/context.go` — already loads all files in `context/` via `Extra []string`; `Mockups []string` captures images. New MCP-written files auto-populate both fields without any code change.
- Prompt templates — `ctx.Ticket` still points to `context/ticket.md`; `ctx.Extra` includes comments/links files automatically.

### Key technical decisions

**1. `Mode string` field in `TicketSystemConfig`**
```go
// internal/config/schema.go
type TicketSystemConfig struct {
    Mode       string     `yaml:"mode,omitempty"` // "" or "api" (default) | "mcp"
    Type       string     `yaml:"type,omitempty"`
    URL        string     `yaml:"url,omitempty"`
    ProjectKey string     `yaml:"project_key,omitempty"`
    Auth       AuthConfig `yaml:"auth,omitempty"`
}
```
Inline check: `cfg.TicketSystem.Mode == "mcp"` (no helper method; keeps schema package simple).

**2. Claude Code slash command — format change**

In `api` mode (current): `!qode ticket fetch $ARGUMENTS`
The `!` prefix causes Claude Code to execute a shell command directly; the AI is not involved.

In `mcp` mode: plain markdown prompt. No `!` prefix. The AI reads the prompt and calls MCP tools.

In `internal/ide/claudecode.go`, `claudeSlashCommands()` returns `map[string]string`. Change the `qode-ticket-fetch` value:

```go
"qode-ticket-fetch": func() string {
    if cfg.TicketSystem.Mode == "mcp" {
        return fmt.Sprintf(`# Fetch Ticket via MCP — %s

Fetch the ticket at the URL provided in $ARGUMENTS using your available MCP tools.

**Steps:**
1. Use whatever MCP tool is available to read the ticket (e.g. a Jira, Linear, GitHub, Notion, or Azure DevOps MCP server). If no MCP server is available, open the URL in a browser tool and extract the content.
2. Collect: title, description, all comments (with author names and timestamps), any linked resources (Figma, Notion docs, Google Docs, spreadsheets), and attachment summaries.
3. Write the following files, creating the directory if needed:

**context/ticket.md** — title, description, and metadata:
\`\`\`
# <Title>

**URL:** $ARGUMENTS
**Status:** <status if available>
**Assignee:** <assignee if available>

## Description

<full description>
\`\`\`

**context/ticket-comments.md** — all comments in chronological order:
\`\`\`
# Comments

## <Author> — <timestamp>
<comment body>

## <Author> — <timestamp>
<comment body>
\`\`\`
(Omit this file if there are no comments.)

**context/ticket-links.md** — linked resources with summaries:
\`\`\`
# Linked Resources

## <Title or URL>
<1-2 sentence summary of what the link contains, fetched via MCP if accessible>
\`\`\`
(Omit this file if there are no links.)

4. Report a one-line summary: "Fetched: <title> — <N> comments, <M> links."
`, name)
    }
    return `!qode ticket fetch $ARGUMENTS`
}(),
```

**3. Cursor slash command — format change**

Cursor commands are already AI prompts (no `!` prefix), so the change is smaller. In `internal/ide/cursor.go`, `slashCommands()`:

```go
"qode-ticket-fetch": func() string {
    if cfg.TicketSystem.Mode == "mcp" {
        return fmt.Sprintf(`---
description: Fetch ticket via MCP for %s
---

Fetch the ticket at the URL provided in $ARGUMENTS using your available MCP tools.

Use whatever MCP tool is available (Jira, Linear, GitHub, Notion, or Azure DevOps server).
If no MCP tool is available, open the URL with a browser tool and extract the content.

Collect: title, description, all comments (author + timestamp), linked resources, attachment summaries.

Write to the context directory for the current branch:
- context/ticket.md — title, description, metadata
- context/ticket-comments.md — all comments (omit if none)
- context/ticket-links.md — linked resource summaries (omit if none)

Report: "Fetched: <title> — <N> comments, <M> links."
`, name)
    }
    return fmt.Sprintf(`---
description: Fetch a ticket into branch context for %s
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
`, name)
}(),
```

**4. MCP tool discovery phrasing**
The instruction uses "use whatever MCP tool is available" rather than hardcoding tool names like `jira_get_issue`. This is intentional: MCP server tool names vary by implementation. The AI's tool-calling mechanism will surface available tools and the instruction leaves selection to the model. This is the established pattern in Claude's tool use — natural-language descriptions let the model pick the right tool.

**5. `qode ticket fetch` deprecation in `internal/cli/ticket.go`**

Add at the start of `RunE`, immediately after config load:
```go
if cfg.TicketSystem.Mode == "mcp" {
    fmt.Fprintln(os.Stderr, "Warning: ticket_system.mode is \"mcp\".")
    fmt.Fprintln(os.Stderr, "Use /qode-ticket-fetch in your IDE to fetch tickets via MCP.")
    fmt.Fprintln(os.Stderr, "qode ticket fetch is a no-op when mode: mcp.")
    return nil
}
```
Exit 0 (return nil), not an error — backward-compat for scripts.

**6. Patterns to follow**
- All non-prompt output → `os.Stderr`; generated prompt content → `os.Stdout` (consistent with `remove-prompt-only-flag` changes on this repo)
- `fmt.Sprintf` with `name` variable for project name interpolation in slash command strings (existing convention in `claudecode.go`)
- `qode ide sync` must be run as a separate explicit step to regenerate generated files

---

## 3. Risks & Edge Cases

**R1 — MCP server not configured**
The AI will fail to call any MCP tool. Mitigated: prompt includes "If no MCP tool is available, open the URL with a browser tool" as a fallback.

**R2 — Claude Code `!` prefix regression**
If the MCP prompt accidentally starts with `!` it will be treated as a shell command. Guard: the prompt starts with `# Fetch Ticket via MCP`. Verified by `TestClaudeSlashCommands_MCPMode_NoExclamationPrefix` test.

**R3 — MCP tool name variability**
Addressed by natural-language instruction (see §2, decision 4).

**R4 — Large comment threads / attachments**
Prompt instructs "summaries" for attachments and linked resources — not full raw content — to bound context size.

**R5 — `context/ticket-comments.md` absent when no comments**
Prompt says "Omit this file if there are no comments." `context.go` handles missing files gracefully (file not found → empty string in `Extra`).

**R6 — `qode ide sync` not re-run after code change**
The generated `.claude/commands/qode-ticket-fetch.md` lags behind code changes until sync is run. Documented as an explicit implementation task.

**R7 — `mode: mcp` with `mode: api` providers still registered**
In Phase 1, both modes co-exist. If a user sets `mode: mcp` but accidentally runs `qode ticket fetch`, they get the warning and exit — providers are never invoked. Safe.

---

## 4. Completeness Check

### Acceptance criteria (all testable)

| # | Criterion | Test location / strategy |
|---|---|---|
| AC1 | `qode.yaml` accepts `ticket_system.mode: mcp` without error | `internal/config/` YAML round-trip (existing test infra) |
| AC2 | `qode ticket fetch <url>` with `mode: mcp` exits 0, prints warning to stderr, does NOT write `ticket.md` | `internal/cli/` integration test: capture stderr, assert no file written |
| AC3 | `qode ticket fetch <url>` with `mode: api` (or empty) behaves exactly as before | Existing tests pass unchanged |
| AC4 | `/qode-ticket-fetch` in Claude Code with `mode: mcp` generates a markdown prompt (no `!` prefix) containing `$ARGUMENTS` | `internal/ide/ide_test.go` — `TestClaudeSlashCommands_MCPMode_TicketFetch` |
| AC5 | `/qode-ticket-fetch` in Claude Code with `mode: api` still generates `!qode ticket fetch $ARGUMENTS` | `TestClaudeSlashCommands_APIMode_TicketFetch` (or existing `ContainsTicketFetch` test with api config) |
| AC6 | `/qode-ticket-fetch` in Cursor with `mode: mcp` generates a prompt with `$ARGUMENTS` and no shell command | `TestCursorSlashCommands_MCPMode_TicketFetch` |
| AC7 | `/qode-ticket-fetch` in Cursor with `mode: api` behaves as before | Existing `TestCursorSlashCommands_ContainsTicketFetch` passes |
| AC8 | `qode ide sync` regenerates `.claude/commands/qode-ticket-fetch.md` correctly for both modes | Manual verification; existing `SetupClaudeCode` integration test confirms file is written |
| AC9 | `docs/how-to-use-ticket-fetch.md` has an MCP section and a legacy API section | Doc review |
| AC10 | `docs/qode-yaml-reference.md` documents `mode` field with allowed values `api` / `mcp` | Doc review |

### Implicit requirements
- The MCP prompt must specify output file paths explicitly (`context/ticket.md` etc.) so the AI knows where to write without guessing
- Comments file should include author names and timestamps (not just raw text) for AI to assess recency and authority
- The prompt must work with `$ARGUMENTS` as a placeholder — both Claude Code and Cursor replace this with the actual URL at invocation time

### Explicitly out of scope (Phase 1)
- Removing `internal/ticket/jira.go`, `azuredevops.go`, `linear.go`, `github.go`, `notion.go`
- Building a Go MCP client
- Changing `internal/context/context.go` or prompt templates in `internal/prompt/templates/`
- Making the context file structure (`ticket.md`, `ticket-comments.md`, etc.) configurable
- Workspace or multi-repo topology considerations (feature is topology-agnostic)
- Changes to `qode check`, `qode review`, or `qode plan`

---

## 5. Actionable Implementation Plan

**Task 1 — Add `Mode` field to `TicketSystemConfig`** (`internal/config/schema.go`)
Add `Mode string \`yaml:"mode,omitempty"\`` to `TicketSystemConfig`. One commit. No new tests needed (YAML unmarshalling covered by existing config package tests).

**Task 2 — CLI deprecation warning** (`internal/cli/ticket.go`)
Add early-exit block (code in §2, decision 5) after config load in `RunE`. One commit.

**Task 3 — Claude Code MCP slash command** (`internal/ide/claudecode.go`)
Wrap `qode-ticket-fetch` value in `claudeSlashCommands()` with the conditional (full content in §2, decision 2). One commit.

**Task 4 — Cursor MCP slash command** (`internal/ide/cursor.go`)
Same conditional in `slashCommands()` (full content in §2, decision 3). One commit.

**Task 5 — Tests** (`internal/ide/ide_test.go`)
Add four new test functions:
- `TestClaudeSlashCommands_MCPMode_TicketFetch` — `mode: mcp` config; assert content does not start with `!`; assert contains `$ARGUMENTS`; assert contains `context/ticket.md`
- `TestClaudeSlashCommands_MCPMode_NoExclamationPrefix` — assert `strings.HasPrefix(content, "!")` is false for `qode-ticket-fetch` in mcp mode (R2 guard)
- `TestCursorSlashCommands_MCPMode_TicketFetch` — same for Cursor
- `TestTicketFetch_MCPMode_IsNoop` — in `internal/cli/` (new test file `ticket_test.go`): set `cfg.TicketSystem.Mode = "mcp"`, call handler directly, assert no file created, assert stderr contains deprecation warning

One commit.

**Task 6 — Run `qode ide sync`**
```bash
go run ./cmd/qode ide sync
```
Regenerates `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc`. The project's own `qode.yaml` uses `mode: api` (or no mode), so generated files stay in API format — correct. One commit.

**Task 7 — Update docs**
- `docs/how-to-use-ticket-fetch.md`: Add MCP section above existing content; add `> **Note:** The API-based approach below is deprecated when `mode: mcp` is set.`
- `docs/qode-yaml-reference.md`: Add `mode` field entry with `api | mcp` values
- `README.md`: Update ticket system section to mention MCP option

One commit.

**Prerequisite order:** Task 1 → Task 2 (needs `Mode` field) → Tasks 3+4 (independent of 2, depend on 1) → Task 5 (needs all code in place) → Task 6 (needs 3+4) → Task 7 (independent, can be last).
