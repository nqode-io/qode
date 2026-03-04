# Technical Specification
# Feature: /qode-ticket-fetch Slash Command

*Branch: feat-qode-ticket-fetch-slash-command*
*Generated: 2026-03-04*

---

## 1. Feature Overview

This feature adds a `/qode-ticket-fetch` IDE command across Claude Code, Cursor, and VS Code that allows developers to fetch a remote ticket (GitHub issue, Jira, Linear, or Azure DevOps) and save it to their branch context without leaving the IDE. Currently, developers must switch to a terminal and run `qode ticket fetch <url>` manually. By surfacing this command in all three supported IDEs, the full qode workflow — from branch creation through shipping — becomes executable without leaving the editor.

**Success criteria**: After the change, running `/qode-ticket-fetch <url>` (Claude Code / Cursor) or the `qode: fetch ticket` task (VS Code) results in `.qode/branches/<branch>/context/ticket.md` being populated with the ticket content, and the developer can immediately proceed to `/qode-plan-refine`.

---

## 2. Scope

### In Scope
- Add `"qode-ticket-fetch"` entry to `claudeSlashCommands()` in `internal/ide/claudecode.go` — generates `.claude/commands/qode-ticket-fetch.md`
- Add `"qode-ticket-fetch"` entry to `slashCommands()` in `internal/ide/cursor.go` — generates `.cursor/commands/qode-ticket-fetch.mdc`
- Add `qode: fetch ticket` task (with `${input:ticketUrl}`) to `buildTasksJSON()` in `internal/ide/vscode.go` — updates `.vscode/tasks.json`
- Regenerate the IDE config artifacts for the current project via `qode ide sync`

### Out of Scope
- Any changes to `internal/cli/ticket.go` or `internal/ticket/` — the CLI command is already complete
- Authentication token management or setup flows
- Fetching multiple tickets in one invocation
- Cursor rules files (`.cursorrules/*.mdc`) — no workflow rule changes needed
- `qode start` prompt changes — ticket context is already read from `context/ticket.md`

### Assumptions
- Claude Code's `!` prefix for direct bash execution is supported and stable
- Cursor's `.mdc` slash commands support `$ARGUMENTS` substitution
- VS Code's `${input:promptString}` input variable mechanism is used for parameterization
- `qode ticket fetch` already works correctly for all four providers

---

## 3. Architecture & Design

### Component Diagram

```
IDE User Interaction
        │
        ├── Claude Code: /qode-ticket-fetch <url>
        │     └── .claude/commands/qode-ticket-fetch.md
        │           content: "!qode ticket fetch $ARGUMENTS"
        │           └── [direct bash exec, no AI]
        │
        ├── Cursor: /qode-ticket-fetch <url>
        │     └── .cursor/commands/qode-ticket-fetch.mdc
        │           └── Cursor AI agent runs: qode ticket fetch $ARGUMENTS
        │
        └── VS Code: Tasks > qode: fetch ticket
              └── .vscode/tasks.json
                    └── prompts for ticketUrl input
                          └── shell: qode ticket fetch ${input:ticketUrl}
                                        │
                              ┌─────────▼─────────────────────────────┐
                              │  qode ticket fetch <url>  (CLI)        │
                              │  internal/cli/ticket.go                │
                              │    → ticket.DetectProvider(url)        │
                              │    → provider.Fetch(url)               │
                              │    → os.WriteFile(context/ticket.md)   │
                              └────────────────────────────────────────┘
```

### Affected Layers

| File | Change Type | Description |
|---|---|---|
| `internal/ide/claudecode.go` | Modified | Add `"qode-ticket-fetch"` to `claudeSlashCommands()` map |
| `internal/ide/cursor.go` | Modified | Add `"qode-ticket-fetch"` to `slashCommands()` map |
| `internal/ide/vscode.go` | Modified | Add task + inputs entry to `buildTasksJSON()` |
| `.claude/commands/qode-ticket-fetch.md` | New (generated) | Claude Code slash command artifact |
| `.cursor/commands/qode-ticket-fetch.mdc` | New (generated) | Cursor slash command artifact |
| `.vscode/tasks.json` | Modified (generated) | Updated by `qode ide sync` |

### Data Flow

1. User invokes the IDE command with a ticket URL
2. The IDE executes `qode ticket fetch <url>` as a shell command
3. `newTicketFetchCmd()` in `internal/cli/ticket.go`:
   - Calls `resolveRoot()` → finds project root
   - Calls `config.Load(root)` → reads `qode.yaml`
   - Calls `git.CurrentBranch(root)` → determines branch name
   - Calls `ticket.DetectProvider(url, cfg.TicketSystem)` → selects provider
   - Calls `provider.Fetch(url)` → HTTP API call, returns `Ticket{Title, Body}`
   - Writes `# <Title>\n\n<Body>\n` to `.qode/branches/<branch>/context/ticket.md`
4. CLI prints `Fetched ticket: <title>` and `Saved to: <path>` to stdout
5. IDE surfaces the output to the user

---

## 4. API / Interface Contracts

No new CLI commands, HTTP endpoints, or Go interfaces are introduced. The only interface change is the map returned by the three generator functions:

### `claudeSlashCommands()` — `internal/ide/claudecode.go`

**Before**: returns 4 keys (`qode-plan-refine`, `qode-plan-spec`, `qode-review-code`, `qode-review-security`)
**After**: returns 5 keys (adds `qode-ticket-fetch`)

New entry:
```go
"qode-ticket-fetch": `!qode ticket fetch $ARGUMENTS`,
```

Generated file `.claude/commands/qode-ticket-fetch.md`:
```
!qode ticket fetch $ARGUMENTS
```

### `slashCommands()` — `internal/ide/cursor.go`

**Before**: returns 4 keys
**After**: returns 5 keys (adds `qode-ticket-fetch`)

New entry:
```go
"qode-ticket-fetch": fmt.Sprintf(`---
description: Fetch a ticket into branch context for %s
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
`, cfg.Project.Name),
```

Generated file `.cursor/commands/qode-ticket-fetch.mdc`:
```markdown
---
description: Fetch a ticket into branch context for <project-name>
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
```

### `buildTasksJSON()` — `internal/ide/vscode.go`

**Before**: returns map with `"version"` and `"tasks"` keys
**After**: returns map with `"version"`, `"tasks"`, and `"inputs"` keys

New task entry (appended to the existing qode tasks slice):
```go
map[string]interface{}{
    "label":   "qode: fetch ticket",
    "type":    "shell",
    "command": "qode ticket fetch ${input:ticketUrl}",
    "group":   "build",
},
```

New `"inputs"` key in the returned map:
```go
"inputs": []map[string]interface{}{
    {
        "id":          "ticketUrl",
        "type":        "promptString",
        "description": "Ticket URL (GitHub issue, Jira, Linear, Azure DevOps)",
    },
},
```

---

## 5. Data Model Changes

No data model changes. No new files are persisted beyond:
- The generated IDE config artifacts (`.claude/commands/`, `.cursor/commands/`, `.vscode/tasks.json`) — these are already managed by the existing `ide.Setup()` / `writeFile()` infrastructure
- The existing `context/ticket.md` written by the CLI — no change to this path or format

No migrations required. No backward compatibility concerns — adding new map entries is purely additive.

---

## 6. Implementation Tasks

- [ ] **Task 1** (ide/claudecode): Add `"qode-ticket-fetch": '!qode ticket fetch $ARGUMENTS'` to `claudeSlashCommands()` map in `internal/ide/claudecode.go`
- [ ] **Task 2** (ide/cursor): Add `"qode-ticket-fetch"` entry with `.mdc` frontmatter to `slashCommands()` map in `internal/ide/cursor.go`
- [ ] **Task 3** (ide/vscode): Add `qode: fetch ticket` task and `"inputs"` array to `buildTasksJSON()` in `internal/ide/vscode.go`
- [ ] **Task 4** (artifacts): Run `qode ide sync` to regenerate `.claude/commands/qode-ticket-fetch.md`, `.cursor/commands/qode-ticket-fetch.mdc`, and `.vscode/tasks.json`; commit all generated artifacts

Tasks 1–3 are independent and can be implemented in a single commit. Task 4 follows.

---

## 7. Testing Strategy

### Unit Tests
- `TestClaudeSlashCommands`: assert the returned map contains key `"qode-ticket-fetch"` with value `"!qode ticket fetch $ARGUMENTS"`
- `TestCursorSlashCommands`: assert the returned map contains key `"qode-ticket-fetch"` and the value contains `qode ticket fetch $ARGUMENTS` and valid YAML frontmatter
- `TestBuildTasksJSON`: assert the returned map contains a task with label `"qode: fetch ticket"` and command `"qode ticket fetch ${input:ticketUrl}"`, and an `"inputs"` array with id `"ticketUrl"`

### Integration Tests
- `TestSetupClaudeCode`: call `SetupClaudeCode()` against a temp dir; assert `.claude/commands/qode-ticket-fetch.md` is written with correct content
- `TestSetupCursor`: call `SetupCursor()` against a temp dir; assert `.cursor/commands/qode-ticket-fetch.mdc` is written with correct content
- `TestSetupVSCode`: call `SetupVSCode()` against a temp dir; assert `.vscode/tasks.json` contains the new task and inputs

### E2E / Manual Tests
1. Run `qode ide sync` on the qode project itself; verify all three artifacts exist and contain expected content
2. **Claude Code**: Invoke `/qode-ticket-fetch https://github.com/nqode-io/qode/issues/1` — verify `ticket.md` is written, CLI output appears in chat
3. **Cursor**: Invoke `/qode-ticket-fetch <url>` — verify agent executes the command and `ticket.md` is created
4. **VS Code**: Run `Tasks: Run Task > qode: fetch ticket` — verify URL prompt appears, command executes, `ticket.md` is created

### Edge Cases to Test
- No URL provided (empty `$ARGUMENTS`) — CLI error from cobra's `ExactArgs(1)` is surfaced
- Unsupported URL format — `ticket.DetectProvider()` error is surfaced
- `GITHUB_TOKEN` not set — GitHub provider auth error is surfaced
- `ticket.md` already exists — file is silently overwritten (desired behavior)

---

## 8. Security Considerations

### Input Validation
- The URL from `$ARGUMENTS` / `${input:ticketUrl}` is passed directly to `qode ticket fetch`. The CLI validates it via `ticket.DetectProvider()` before making any HTTP request.
- No shell interpolation risk: Claude Code executes `!`-prefixed commands as structured subprocesses; VS Code tasks use the `shell` type with the command string treated as a literal command, not interpolated by a shell glob.

### Authentication
- No new authentication mechanisms. Existing provider tokens (`GITHUB_TOKEN`, `JIRA_API_TOKEN`, etc.) are read from environment variables by the provider implementations. No credentials are written to disk.

### Data Sensitivity
- Ticket content (title + body) is written to `.qode/branches/<branch>/context/ticket.md` as plain text. This file is local to the developer's machine and not transmitted anywhere. If ticket bodies contain sensitive information, they are already handled the same way as if the developer had manually written `context/ticket.md`.

---

## 9. Open Questions

None. All ambiguities from the requirements analysis have been resolved:

| Question | Resolution |
|---|---|
| Direct execution vs `--prompt-only` pattern? | Direct execution (`!` prefix in Claude Code, agent instruction in Cursor, `tasks.json` in VS Code) |
| Post-fetch AI summary? | Not needed — CLI stdout is sufficient |
| VS Code parameterization mechanism? | `${input:ticketUrl}` with `promptString` type |
| Cursor `.mdc` format? | Follows existing `slashCommands()` pattern with YAML frontmatter |

---

*Spec generated by qode. Copy to ticket system for team review.*
