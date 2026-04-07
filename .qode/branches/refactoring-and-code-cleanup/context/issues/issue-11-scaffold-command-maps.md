# Issue #11: Use Table-Driven Patterns for the Scaffold Command Maps

## Summary

`claudeSlashCommands()` and `slashCommands()` (cursor) each return a `map[string]string` built by assembling IDE-specific frontmatter around command bodies. 8 of 9 command bodies are identical between the two functions; only the frontmatter wrapper differs. The two maps share no data structure — adding a new command or changing a body requires editing both functions. A `CommandDef` struct and a single shared command slice would make the maps table-driven and reduce a new-command addition to one entry.

## Affected Files

- `internal/scaffold/claudecode.go` lines 80–187 — `claudeSlashCommands()`
- `internal/scaffold/cursor.go` lines 30–143 — `slashCommands()` (note: inconsistently named vs `claudeSlashCommands`)
- `internal/scaffold/ticket_fetch.go` — `ticketFetchMCPBody` (already extracted as shared constant — correct pattern)
- `internal/scaffold/scaffold_test.go` — tests for both setup functions

## Current State

Both functions build maps with the same keys and near-identical bodies:

**`claudeSlashCommands()` (claudecode.go)**
```go
func claudeSlashCommands(name string) map[string]string {
    return map[string]string{
        "qode-plan-refine": fmt.Sprintf("# Refine Requirements — %s\n\n**Worker pass:** ...", name),
        "qode-plan-spec":   fmt.Sprintf("# Generate Technical Specification — %s\n\n...", name),
        "qode-review-code": fmt.Sprintf("# Code Review — %s\n\n...", name),
        // ... 6 more entries, bodies duplicated from cursor.go
    }
}
```

**`slashCommands()` (cursor.go)**
```go
func slashCommands(name string) map[string]string {
    return map[string]string{
        "qode-plan-refine": fmt.Sprintf("---\ndescription: Refine requirements for %s\n---\n\n**Worker pass:** ...", name),
        "qode-plan-spec":   fmt.Sprintf("---\ndescription: Generate technical specification for %s\n---\n\n...", name),
        "qode-review-code": fmt.Sprintf("---\ndescription: Code review for %s\n---\n\n...", name),
        // ... 6 more entries with identical bodies, different frontmatter
    }
}
```

The existing `qodeCheckBody` and `ticketFetchMCPBody` constants already demonstrate the correct extraction pattern — they're defined once and referenced in both functions. The remaining 7 command bodies are not extracted.

## Proposed Fix

### Step 1: Define a shared command struct

```go
// internal/scaffold/commands.go (new file)
type CommandDef struct {
    Name        string // e.g. "qode-plan-refine"
    Title       string // used in Claude Code: "# Title — <project>"
    Description string // used in Cursor: "description: <desc> for <project>"
    Body        string // shared body text
}

var allCommands = []CommandDef{
    {
        Name:        "qode-plan-refine",
        Title:       "Refine Requirements",
        Description: "Refine requirements for %s",
        Body:        qodePlanRefineBody,
    },
    {
        Name:        "qode-plan-spec",
        Title:       "Generate Technical Specification",
        Description: "Generate technical specification for %s",
        Body:        qodePlanSpecBody,
    },
    // ... one entry per command; bodies defined as package-level constants
}
```

### Step 2: Extract remaining bodies as constants

Alongside `qodeCheckBody` and `ticketFetchMCPBody`, add:
- `qodePlanRefineBody`
- `qodePlanSpecBody`
- `qodeReviewCodeBody`
- `qodeReviewSecurityBody`
- `qodeStartBody`
- `qodeKnowledgeAddContextBody`
- `qodeKnowledgeAddBranchBody`

### Step 3: Rewrite both command map functions

```go
// claudecode.go
func claudeSlashCommands(name string) map[string]string {
    m := make(map[string]string, len(allCommands))
    for _, cmd := range allCommands {
        m[cmd.Name] = fmt.Sprintf("# %s — %s\n\n%s", cmd.Title, name, cmd.Body)
    }
    return m
}

// cursor.go — also rename to cursorSlashCommands for consistency
func cursorSlashCommands(name string) map[string]string {
    m := make(map[string]string, len(allCommands))
    for _, cmd := range allCommands {
        desc := fmt.Sprintf(cmd.Description, name)
        m[cmd.Name] = fmt.Sprintf("---\ndescription: %s\n---\n\n%s", desc, cmd.Body)
    }
    return m
}
```

Update `SetupCursor()` to call `cursorSlashCommands()` instead of `slashCommands()`.

## Impact

| Task | Before | After |
|------|--------|--------|
| Update a command body | Edit 2 files | Edit 1 constant |
| Add a new command | Add entry to 2 map functions | Add 1 `CommandDef` entry + 1 constant |
| Add a 3rd IDE | Copy ~100 lines | Write a 5-line wrapper function |
| Code review for command change | Verify both files are identical | Review one change |

The rename of `slashCommands` → `cursorSlashCommands` also removes a naming inconsistency that exists today.
