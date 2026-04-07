# Issue #4: Duplication Between Cursor and Claude Code Scaffold

## Summary

The scaffold package contains significant code duplication between `claudecode.go` and `cursor.go`. Each IDE generates slash commands by maintaining its own complete copy of command definitions, differing only in IDE-specific frontmatter (Claude Code uses Markdown headings; Cursor uses YAML frontmatter). Every slash command update must be applied in two places, risking inconsistencies. Currently only `qodeCheckBody` and `ticketFetchMCPBody` are shared — the remaining 8 commands are fully duplicated.

## Affected Files

- `internal/scaffold/claudecode.go`
- `internal/scaffold/cursor.go`
- `internal/scaffold/ticket_fetch.go` (contains `ticketFetchMCPBody`, one of the two shared constants)

## Current State

### Only 2 of 10 commands are shared

`qodeCheckBody` (~48 lines) is defined in `claudecode.go` and reused in both files.  
`ticketFetchMCPBody` (~21 lines) is defined in `ticket_fetch.go` and reused in both files.

### 8 commands are fully duplicated

The following commands appear in both files with only frontmatter differences:

| Command | Claude Code wrapper | Cursor wrapper |
|---------|---------------------|----------------|
| `qode-plan-refine` | `# Refine Requirements — %s\n\n` | `---\ndescription: Refine requirements for %s\n---\n\n` |
| `qode-plan-spec` | `# Generate Technical Specification — %s\n\n` | `---\ndescription: Generate technical specification for %s\n---\n\n` |
| `qode-review-code` | `# Code Review — %s\n\n` | `---\ndescription: Code review for %s\n---\n\n` |
| `qode-review-security` | `# Security Review — %s\n\n` | `---\ndescription: Security review for %s\n---\n\n` |
| `qode-start` | `# Start Implementation — %s\n\n` | `---\ndescription: Start implementation session for %s\n---\n\n` |
| `qode-knowledge-add-context` | `# Extract Lessons Learned — %s\n\n` | `---\ndescription: Extract lessons learned from current session for %s\n---\n\n` |
| `qode-knowledge-add-branch` | `# Extract Lessons from Branch — %s\n\n` | `---\ndescription: Extract lessons learned from branch context for %s\n---\n\n` |

### Setup functions also duplicate logic

`SetupClaudeCode()` and `SetupCursor()` share the same structure:

```
os.MkdirAll(ideDir)
name := filepath.Base(root)
cmds := ideSlashCommands(name)
for cmdName, content := range cmds { write file }
fmt.Printf(status message)
```

Differences: directory path (`.claude/commands/` vs `.cursor/commands/`), file extension (`.md` vs `.mdc`), and status message format.

Additionally, the cursor command generator is named `slashCommands()` instead of `cursorSlashCommands()` — an inconsistency with `claudeSlashCommands()`.

## Proposed Fix

### Step 1: Define a shared command struct

```go
// internal/scaffold/commands.go (new file)
type SlashCommand struct {
    Name        string // e.g. "qode-plan-refine"
    Title       string // used in Claude Code heading
    Description string // used in Cursor frontmatter (may include %s for project name)
    Body        string // shared body text
}
```

### Step 2: Extract all command bodies as package-level constants

Move the body text for all 8 duplicated commands to constants (alongside existing `qodeCheckBody` and `ticketFetchMCPBody`). Create a `getSharedCommands()` function returning `[]SlashCommand` with all 10 commands.

### Step 3: Thin IDE-specific wrappers

**In `claudecode.go`:**
```go
func claudeSlashCommands(name string) map[string]string {
    result := make(map[string]string)
    for _, cmd := range getSharedCommands() {
        result[cmd.Name] = fmt.Sprintf("# %s — %s\n\n%s", cmd.Title, name, cmd.Body)
    }
    return result
}
```

**In `cursor.go`:**
```go
func cursorSlashCommands(name string) map[string]string {
    result := make(map[string]string)
    for _, cmd := range getSharedCommands() {
        desc := fmt.Sprintf(cmd.Description, name)
        result[cmd.Name] = fmt.Sprintf("---\ndescription: %s\n---\n\n%s", desc, cmd.Body)
    }
    return result
}
```

Rename `slashCommands()` in cursor.go to `cursorSlashCommands()` for consistency.

### Step 4: Consolidate setup logic

```go
func setupIDECommands(root, ideDir, ext string, cmds map[string]string) error {
    if err := os.MkdirAll(filepath.Join(root, ideDir), 0755); err != nil {
        return err
    }
    for name, content := range cmds {
        path := filepath.Join(root, ideDir, name+ext)
        if err := os.WriteFile(path, []byte(content), 0644); err != nil {
            return err
        }
    }
    return nil
}
```

Both `SetupClaudeCode` and `SetupCursor` become thin callers of `setupIDECommands`.

## Impact

- **Update cost today**: changing one command body requires editing two files (~18 lines each) with risk of divergence
- **After fix**: editing one constant automatically applies to both IDEs
- **Adding a third IDE** (e.g., Windsurf): currently requires ~250 lines of duplicated code; after fix requires ~10 lines of adapter code
- **Adding a new slash command**: currently requires editing two map functions; after fix requires one entry in `getSharedCommands()`
