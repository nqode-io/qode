<!-- qode:iteration=3 score=25/25 -->

# Requirements Analysis — #27 Replace qode ticket fetch with MCP server integration

## 1. Problem Understanding

**Restated:** `qode ticket fetch` ships five hand-rolled HTTP clients in `internal/ticket/` that extract only a ticket's title and description into a flat `context/ticket.md`. Comments, attachments, linked design docs (Figma, draw.io, Google Docs, SharePoint, etc.) are silently dropped. Production-ready MCP servers exist for all five ticketing systems and surface that richness natively. Since qode is in alpha, breaking changes are expected.

**Decision (from notes.md):** Always use MCP — no `mode` config field, no API fallback, no phased approach. The `qode ticket fetch` CLI command is deleted entirely. The `/qode-ticket-fetch` IDE slash command always emits an MCP-driven AI prompt. When the ticket contains links to other services that have MCP servers configured (Figma, draw.io, Google Docs, SharePoint, Confluence, Miro, etc.), the AI must use those MCP tools to fetch the linked content too.

**User need:** The AI generating a spec or plan needs the full ticket — comments that clarify acceptance criteria, Figma links with design details, attachments with edge-case specs, linked Google Docs with requirements. Today all of that is lost before the AI ever sees it.

**Business value:** Eliminates maintenance of five diverging API clients; unlocks rich ticket context including linked resources; aligns with IDE MCP ecosystem; reduces codebase complexity.

**Resolved ambiguities:**
- No `mode` field — alpha stage, clean break.
- `qode ticket fetch` is deleted outright (not deprecated with a warning).
- `TicketSystemConfig` becomes dead code after deletion — remove from `internal/config/schema.go` entirely.
- Linked resources (Figma, draw.io, Google Docs, SharePoint, Confluence, Miro) must be fetched via their own MCP servers if configured — this is part of the slash command prompt, not a separate Go feature.
- `qode init` is the correct command to regenerate IDE configs (not `qode ide sync`, which is deprecated).
- Docs rewritten to cover official MCP server setup per ticketing system and per linked-resource service.

---

## 2. Technical Analysis

### Affected components

| Component | File | Change |
|---|---|---|
| Ticket providers | `internal/ticket/` (entire directory, 8 files) | **Delete** |
| CLI command | `internal/cli/ticket.go` | **Delete** |
| CLI root | `internal/cli/root.go:68` | Remove `newTicketCmd()` call; remove any now-unused imports |
| Config schema | `internal/config/schema.go` | Remove `TicketSystem TicketSystemConfig` from `Config`; delete `TicketSystemConfig` struct; verify `AuthConfig` has no other consumers before removing |
| Scaffold — Claude Code | `internal/scaffold/claudecode.go:151` | Replace `!qode ticket fetch $ARGUMENTS` with MCP prompt via `ticketFetchClaudeCommand(name string) string` helper |
| Scaffold — Cursor | `internal/scaffold/cursor.go:106-112` | Replace CLI-run content with MCP prompt via `ticketFetchCursorCommand(name string) string` helper |
| Tests — scaffold | `internal/scaffold/scaffold_test.go` | Update 4 failing tests; add `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix` |
| Tests — ticket | `internal/ticket/github_test.go`, `notion_test.go` | Deleted with the directory |
| Docs | `docs/how-to-use-ticket-fetch.md` | **Rewrite entirely** — MCP setup per ticketing system + per linked-resource service |
| Docs | `docs/qode-yaml-reference.md` | Remove `ticket_system` section |
| Docs | `README.md` | Update ticket section to point to MCP approach |
| Generated files | `.claude/commands/qode-ticket-fetch.md` | Regenerated via `qode init` |
| Generated files | `.cursor/commands/qode-ticket-fetch.mdc` | Same |

**Not affected:**
- `internal/scaffold/scaffold.go` — no signature changes needed (no mode to thread)
- `claudeSlashCommands(name string)` and `slashCommands(name string)` — signatures unchanged; only the `qode-ticket-fetch` map value changes
- `internal/context/context.go` — already auto-loads all files in `context/` and images via `Mockups`; new MCP-written files are captured automatically
- Prompt templates — `ctx.Ticket` still points to `context/ticket.md`; new files populate `ctx.Extra`

### Key technical decisions

**1. No function signature changes in scaffold**

Since there is no mode branching, `claudeSlashCommands(name string)` and `slashCommands(name string)` keep their current signatures. The only change is the string value for the `"qode-ticket-fetch"` key, extracted to a named helper to keep the map readable.

**2. Claude Code slash command — always MCP, follow linked resources**

Plain markdown (no `!` prefix), contains `$ARGUMENTS`, specifies output file paths, instructs the AI to follow linked resources using their own MCP servers:

```markdown
# Fetch Ticket via MCP — <project-name>

Fetch the ticket at the URL or ID provided in $ARGUMENTS using your available MCP tools.

**Steps:**
1. Use the MCP tool for the appropriate ticketing system (Jira, Linear, GitHub Issues,
   Notion, or Azure DevOps). Prefer the official MCP server for the system. If no MCP
   server is available, open the URL in a browser tool and extract the content.
2. Collect: title, description, all comments (with author names and timestamps), any
   linked resources, and attachment summaries.
3. For each linked resource, fetch its content using the appropriate MCP server if one
   is configured — e.g. Figma (designs), draw.io (diagrams), Google Docs, SharePoint,
   Confluence, Miro boards. If no MCP server is available for a linked resource, record
   its URL and title only.
4. Write the following files under context/, creating the directory if needed:
   - context/ticket.md — title, description, metadata
   - context/ticket-comments.md — all comments in chronological order (omit if no comments)
   - context/ticket-links.md — linked resources with summaries or fetched content (omit if no links)
5. Report a one-line summary: "Fetched: <title> — <N> comments, <M> links."
```

**3. Cursor slash command — always MCP, same linked-resource instructions**

Cursor format keeps `description:` frontmatter and project name. Identical MCP steps:

```markdown
---
description: Fetch a ticket into branch context for <project-name>
---

Fetch the ticket at the URL or ID provided in $ARGUMENTS using your available MCP tools.

[same steps as Claude Code above]
```

**4. `TicketSystemConfig` removal**

Only referenced in:
- `internal/config/schema.go` (definition + `Config.TicketSystem`)
- `internal/cli/ticket.go` (`cfg.TicketSystem`)
- `internal/ticket/ticket.go` (`DetectProvider(rawURL string, cfg config.TicketSystemConfig)`)

After deleting `internal/cli/ticket.go` and `internal/ticket/`, zero non-test references remain. Remove `TicketSystem TicketSystemConfig` from `Config` and delete the type. Verify `AuthConfig` is not used elsewhere before removing it too (grep `AuthConfig` across `internal/`).

**5. Official MCP server documentation — ticketing systems**

| System | Official MCP server | Notes |
|---|---|---|
| GitHub | `@modelcontextprotocol/server-github` | `npx -y @modelcontextprotocol/server-github` |
| Jira | `@atlassian/jira-mcp` | Atlassian developer portal; requires API token |
| Linear | Linear official MCP | Via Linear Settings → API → MCP |
| Azure DevOps | `@microsoft/azure-devops-mcp` | Via Azure DevOps Marketplace |
| Notion | `@notionhq/notion-mcp-server` | `npx -y @notionhq/notion-mcp-server`; requires integration token |

**6. Official MCP server documentation — linked-resource services**

| Service | Official MCP server | Notes |
|---|---|---|
| Figma | `@figma/mcp` (Figma official) | Via Figma → Settings → MCP; requires Figma access token |
| Google Docs / Drive | `@modelcontextprotocol/server-gdrive` | OAuth 2.0; requires Google Cloud credentials |
| SharePoint / OneDrive | `@microsoft/sharepoint-mcp` | Microsoft 365 OAuth; requires Azure App Registration |
| Confluence | `@atlassian/confluence-mcp` | Atlassian developer portal; requires API token |
| Miro | Miro official MCP | Via Miro → Apps → MCP |
| draw.io | No official MCP server as of writing — record URL only |

Each section in the rewritten doc must cover: install/enable command, required auth/env vars, IDE configuration snippet (both `.claude/mcp.json` for Claude Code and `mcp.json` for Cursor).

**7. Regenerating generated files**

After updating `internal/scaffold/claudecode.go` and `cursor.go`, run `qode init` (not `qode ide sync` — deprecated) to regenerate `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc`.

### Patterns to follow
- Functions ≤ 50 lines, single responsibility
- Delete dead code entirely — no deprecation wrappers in alpha
- `os.Stderr` for warnings, never `os.Stdout`

### Dependencies
- Issues #28 (post step outputs as ticket comments) and #31 (PR/MR review) depend on this PR
- No external Go dependencies introduced

---

## 3. Risks & Edge Cases

**R1 — Users with `ticket_system:` in `qode.yaml`**
After removing `TicketSystemConfig`, Go's `yaml.v3` silently ignores unknown fields during unmarshalling — no parse error. Users should clean up their config manually; the docs change covers this.

**R2 — Import cleanup after deletion**
After deleting `internal/ticket/` and `internal/cli/ticket.go`, verify `internal/cli/root.go` no longer imports either package. The `newTicketCmd()` removal at line 68 is the required code change; if `ticket` was only imported transitively via `ticket.go`, the import will be stale and must be removed. Verify with `go build ./...`.

**R3 — MCP server not configured for ticket system**
AI cannot call MCP tools for the ticketing system. Mitigated: prompt instructs "If no MCP server is available, open the URL in a browser tool and extract the content."

**R4 — MCP server not configured for a linked resource**
AI cannot fetch Figma/Google Docs/etc. Mitigated: prompt instructs "If no MCP server is available for a linked resource, record its URL and title only."

**R5 — Claude Code `!` prefix regression**
If the MCP prompt accidentally starts with `!`, Claude Code treats it as a shell command. The prompt starts with `# Fetch Ticket via MCP` — safe. Caught by `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix`.

**R6 — Four existing tests assert old CLI command string**
- `TestClaudeSlashCommands_ContainsTicketFetch` — exact equality `!qode ticket fetch $ARGUMENTS`
- `TestSetupClaudeCode_WritesTicketFetchCommand` — exact equality
- `TestCursorSlashCommands_ContainsTicketFetch` — `strings.Contains` for `"qode ticket fetch $ARGUMENTS"`
- `TestSetupCursor_WritesTicketFetchCommand` — same

All four must be updated to assert MCP prompt content.

**R7 — `AuthConfig` used outside `TicketSystemConfig`**
Grep `AuthConfig` across `internal/` before deletion; remove only if no other consumers.

---

## 4. Completeness Check

### Acceptance criteria

| # | Criterion | Verification |
|---|---|---|
| AC1 | `internal/ticket/` directory deleted (8 files) | `go build ./...` passes; directory absent |
| AC2 | `internal/cli/ticket.go` deleted; `qode ticket fetch` no longer in `qode --help` | `go build ./...`; CLI help output |
| AC3 | `TicketSystemConfig` removed from `internal/config/schema.go` | `go build ./...`; grep confirms absence |
| AC4 | `/qode-ticket-fetch` in Claude Code: no `!` prefix, contains `$ARGUMENTS`, contains `context/ticket.md`, contains linked-resource fetching instructions | `TestClaudeSlashCommands_TicketFetchIsPrompt`, `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix` |
| AC5 | `/qode-ticket-fetch` in Cursor: contains `$ARGUMENTS`, `context/ticket.md`, `description:` frontmatter, linked-resource fetching instructions | `TestCursorSlashCommands_TicketFetchIsPrompt` |
| AC6 | Both prompts instruct AI to fetch linked resources (Figma, Google Docs, SharePoint, Confluence, Miro) via MCP if available | Content assertions in AC4/AC5 tests |
| AC7 | `docs/how-to-use-ticket-fetch.md` rewritten: MCP setup for all 5 ticketing systems + Figma, Google Docs, SharePoint, Confluence, Miro | Doc review |
| AC8 | `docs/qode-yaml-reference.md` has no `ticket_system` section | Doc review |
| AC9 | `README.md` ticket section points to MCP approach | Doc review |
| AC10 | Generated `.claude/commands/qode-ticket-fetch.md` contains MCP prompt (run `qode init`) | File inspection |
| AC11 | Generated `.cursor/commands/qode-ticket-fetch.mdc` contains MCP prompt | File inspection |

### Implicit requirements
- `$ARGUMENTS` placeholder present in both IDE slash commands
- Output file paths explicitly specified in prompts
- Comments must include author names and timestamps
- Linked-resource fallback behaviour (record URL/title if no MCP server) must be in prompts
- Each doc section includes both Claude Code (`.claude/mcp.json`) and Cursor (`mcp.json`) config snippets

### Explicitly out of scope
- Building a Go MCP client — IDE handles MCP
- Making the context file structure configurable
- Changes to `internal/context/context.go` or prompt templates
- draw.io MCP support (no official server exists as of writing)
- Keeping backward compat for `ticket_system:` config

---

## 5. Actionable Implementation Plan

**Task 1** — Delete `internal/ticket/` directory (8 files: `ticket.go`, `jira.go`, `azuredevops.go`, `linear.go`, `github.go`, `notion.go`, `github_test.go`, `notion_test.go`). One commit.

**Task 2** — Delete `internal/cli/ticket.go`. Remove `newTicketCmd()` at `internal/cli/root.go:68`. Remove stale imports from `root.go`. Run `go build ./...` to confirm clean. One commit.

**Task 3** — `internal/config/schema.go`: Remove `TicketSystem TicketSystemConfig` from `Config`. Delete `TicketSystemConfig` type. Grep `AuthConfig` — remove if no other consumers. Run `go build ./...`. One commit.

**Task 4** — `internal/scaffold/claudecode.go`: Replace inline `"qode-ticket-fetch": \`!qode ticket fetch $ARGUMENTS\`` with `"qode-ticket-fetch": ticketFetchClaudeCommand(name)`. Add `ticketFetchClaudeCommand(name string) string` helper returning the full MCP markdown prompt including linked-resource instructions. One commit.

**Task 5** — `internal/scaffold/cursor.go`: Replace `"qode-ticket-fetch"` value with `ticketFetchCursorCommand(name)`. Add `ticketFetchCursorCommand(name string) string` returning MCP prompt with `description:` frontmatter and linked-resource instructions. One commit.

**Task 6** — `internal/scaffold/scaffold_test.go`: Update the 4 failing tests to assert MCP prompt content (no `!` prefix, contains `$ARGUMENTS`, contains `context/ticket.md`, contains linked-resource keyword). Add `TestClaudeSlashCommands_TicketFetchNoExclamationPrefix`. One commit.

**Task 7** — Run `qode init` to regenerate `.claude/commands/qode-ticket-fetch.md` and `.cursor/commands/qode-ticket-fetch.mdc`. Verify generated content. One commit.

**Task 8** — Rewrite `docs/how-to-use-ticket-fetch.md` (MCP setup for all 5 ticketing systems + Figma, Google Docs, SharePoint, Confluence, Miro — each with install, auth, and IDE config snippets). Update `docs/qode-yaml-reference.md` (remove `ticket_system`). Update `README.md`. One commit.

**Task order:** 1 → 2 → 3 (deletion tasks, independent but ordered for clean build verification) → 4 → 5 (scaffold changes, independent of each other) → 6 (depends on 4+5) → 7 (depends on 4+5) → 8 (independent, can be done in parallel with 4–7).
