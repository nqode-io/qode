# Issue #15: Move Slash Command Bodies into the Template Engine

## Summary

`internal/scaffold/claudecode.go` and `internal/scaffold/cursor.go` embed large multi-line slash command strings as raw Go string literals (with `fmt.Sprintf` for project name injection). These strings cannot be locally overridden by users — unlike all prompt templates, which are resolved via the `Engine`'s local-override-first strategy (`.qode/prompts/<name>.md.tmpl` → embedded fallback).

Moving slash command bodies into `internal/prompt/templates/scaffold/` would give users the same override capability they have for prompt templates, and would unify all user-visible text under one system.

## Affected Files

**Source of the problem:**

- `internal/scaffold/claudecode.go` lines 30–187 — `claudeSlashCommands()`, `qodeCheckBody` const, `ticketFetchClaudeCommand()`
- `internal/scaffold/cursor.go` lines 30–143 — `slashCommands()`, `ticketFetchCursorCommand()`
- `internal/scaffold/ticket_fetch.go` lines 7–27 — `ticketFetchMCPBody` const

**Template system (reference):**

- `internal/prompt/engine.go` lines 79–138 — `loadTemplate()` implements local-override-first resolution
- `internal/prompt/engine.go:17` — `//go:embed templates` already covers the entire `templates/` directory recursively
- `internal/prompt/templates/` — existing template structure (refine/, review/, spec/, start/, etc.)

**Tests (will need updates):**

- `internal/scaffold/scaffold_test.go` — currently tests internal functions that would be removed

## Current State

Slash command bodies are Go string literals concatenated with the project name:

```go
// claudecode.go lines 82–99
"qode-plan-refine": fmt.Sprintf(`# Refine Requirements — %s

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine
...
`, name),
```

The only shared constants today are `qodeCheckBody` (48 lines in `claudecode.go`) and `ticketFetchMCPBody` (21 lines in `ticket_fetch.go`) — referenced in both files. The remaining 7 command bodies are duplicated between the two scaffold files.

**Template engine resolution** (engine.go lines 124–138):

```go
func (e *Engine) loadTemplate(name string) (string, error) {
    // 1. Check .qode/prompts/<name>.md.tmpl (user override)
    localPath := filepath.Join(e.root, config.QodeDir, "prompts", name+".md.tmpl")
    if data, err := os.ReadFile(localPath); err == nil {
        return string(data), nil
    }
    // 2. Fall back to embedded templates
    return embeddedFS.ReadFile("templates/" + name + ".md.tmpl")
}
```

Scaffold strings bypass this system entirely — they can never be user-overridden.

## Proposed Fix

### Step 1: Create template files in `internal/prompt/templates/scaffold/`

Add 18 template files (9 commands × 2 IDEs), using `{{.Project.Name}}` instead of `%s`:

```
internal/prompt/templates/scaffold/
├── qode-check.claude.md.tmpl
├── qode-check.cursor.md.tmpl
├── qode-knowledge-add-branch.claude.md.tmpl
├── qode-knowledge-add-branch.cursor.md.tmpl
├── qode-knowledge-add-context.claude.md.tmpl
├── qode-knowledge-add-context.cursor.md.tmpl
├── qode-plan-refine.claude.md.tmpl
├── qode-plan-refine.cursor.md.tmpl
├── qode-plan-spec.claude.md.tmpl
├── qode-plan-spec.cursor.md.tmpl
├── qode-review-code.claude.md.tmpl
├── qode-review-code.cursor.md.tmpl
├── qode-review-security.claude.md.tmpl
├── qode-review-security.cursor.md.tmpl
├── qode-start.claude.md.tmpl
├── qode-start.cursor.md.tmpl
├── qode-ticket-fetch.claude.md.tmpl
└── qode-ticket-fetch.cursor.md.tmpl
```

Example `qode-plan-refine.claude.md.tmpl`:

```
# Refine Requirements — {{.Project.Name}}

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine
...
```

Example `qode-plan-refine.cursor.md.tmpl`:

```
---
description: Refine requirements for {{.Project.Name}}
---

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine
...
```

The `//go:embed templates` directive in `engine.go:17` already covers this directory recursively — no build changes needed.

### Step 2: Rewrite SetupClaudeCode and SetupCursor to use engine.Render

```go
// claudecode.go (simplified after refactor)
func SetupClaudeCode(root string) error {
    commandsDir := filepath.Join(root, ".claude", "commands")
    if err := os.MkdirAll(commandsDir, 0755); err != nil {
        return err
    }
    engine, err := prompt.NewEngine(root)
    if err != nil {
        return err
    }
    data := prompt.TemplateData{Project: prompt.TemplateProject{Name: filepath.Base(root)}}
    commands := []string{
        "qode-plan-refine", "qode-plan-spec", "qode-review-code",
        "qode-review-security", "qode-check", "qode-start",
        "qode-ticket-fetch", "qode-knowledge-add-context", "qode-knowledge-add-branch",
    }
    for _, cmd := range commands {
        content, err := engine.Render("scaffold/"+cmd+".claude", data)
        if err != nil {
            return fmt.Errorf("rendering scaffold/%s.claude: %w", cmd, err)
        }
        if err := writeFile(filepath.Join(commandsDir, cmd+".md"), content); err != nil {
            return err
        }
    }
    fmt.Printf("  Claude Code: %d slash commands\n", len(commands))
    return nil
}
```

Cursor setup is identical except template suffix is `.cursor` and file extension is `.mdc`.

### Step 3: Delete old code

- `claudeSlashCommands()` (lines 80–187 of claudecode.go)
- `slashCommands()` (lines 30–143 of cursor.go)
- `ticketFetchClaudeCommand()`, `ticketFetchCursorCommand()`
- `qodeCheckBody` const, `ticketFetchMCPBody` const
- Delete `ticket_fetch.go` entirely (content moved to templates)

### Step 4: Update tests

Replace internal-function unit tests with integration tests of `SetupClaudeCode()`/`SetupCursor()` that verify file count, content keywords, and local override behavior:

```go
func TestSetupClaudeCode_UsesLocalOverride(t *testing.T) {
    dir := t.TempDir()
    overrideDir := filepath.Join(dir, ".qode", "prompts", "scaffold")
    os.MkdirAll(overrideDir, 0755)
    os.WriteFile(filepath.Join(overrideDir, "qode-check.claude.md.tmpl"),
        []byte("# My Custom Check — {{.Project.Name}}\nCustom behavior."), 0644)

    SetupClaudeCode(dir)

    content, _ := os.ReadFile(filepath.Join(dir, ".claude", "commands", "qode-check.md"))
    if !strings.Contains(string(content), "My Custom Check") {
        t.Error("local override template was not used")
    }
}
```

## Impact

- **User customizability**: any slash command can be overridden via `.qode/prompts/scaffold/<command>.<ide>.md.tmpl` — same mechanism as prompt templates
- **Unified system**: all user-visible text lives under the template engine; no special-case Go string literals
- **Maintainability**: command body changes are made in `.md.tmpl` files, not inside Go string literals with escape sequences
- **No breaking changes**: generated files in `.claude/commands/` and `.cursor/commands/` are content-identical after refactoring
- **Code reduction**: ~700 lines of Go string literals removed; replaced by 18 small template files and ~30 lines of loop logic
