# Technical Specification — Lessons Learned Feature

## 1. Feature Overview

This feature adds automated lesson extraction to qode's knowledge base, closing the feedback loop in the plan→implement→review cycle. Two new commands allow developers to distill actionable lessons from either their current IDE session (`/qode-knowledge-add-context`, slash-command only) or from branch artifacts (`qode knowledge add-branch`, terminal + slash command). Each lesson is stored as an individual Markdown file in `.qode/knowledge/lessons/`, automatically discovered by the existing knowledge system and injected into future AI prompts. Lessons are deduplicated against existing ones via AI-driven semantic comparison in the extraction prompt.

**Success criteria:**
- Developers can extract lessons from session context via slash command in Claude Code and Cursor
- Developers can extract lessons from branch artifacts via terminal or slash command
- Lessons are stored individually, auto-discovered by `knowledge.Load()`, and deduplicated
- Documentation reflects the new recommended workflow step

## 2. Scope

### In Scope
- Recursive file discovery in `knowledge.Load()` / `knowledge.List()` to support `.qode/knowledge/lessons/` subdirectory
- `Load()` header adjustment to avoid double `###` headings for lesson files
- New `LessonsDir()`, `ListLessons()`, `SaveLesson()`, `ToKebabCase()` functions in knowledge package
- New `Lessons` field in `prompt.TemplateData`
- Two new embedded prompt templates: `knowledge/add-context.md.tmpl` and `knowledge/add-branch.md.tmpl`
- New `qode knowledge add-branch` CLI command with `--prompt-only` flag
- New slash commands in Claude Code (`claudeSlashCommands()`) and Cursor (`slashCommands()`)
- Documentation updates: `CLAUDE.md`, `buildClaudeMD()`, `workflowRule()`
- Unit tests for all new and modified functions

### Out of Scope
- Automatic lesson extraction without explicit user trigger
- Lesson categorization, tagging, or quality scoring
- Lesson editing or deletion commands
- Cross-project lesson sharing
- VS Code slash command equivalent

### Assumptions
- The `claude` CLI is available for `dispatch.RunInteractive()` (existing requirement)
- `.qode/knowledge/` directory structure is under user control (gitignored or committed per project)
- AI model produces well-formatted lesson files when given a precise prompt (validated by prompt quality)

## 3. Architecture & Design

### Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    IDE (Claude Code / Cursor)            │
│                                                         │
│  /qode-knowledge-add-context    /qode-knowledge-add-branch
│  (inline prompt, session-only)  (delegates to CLI)      │
└──────────┬──────────────────────────────┬───────────────┘
           │                              │
           │                              ▼
           │                 ┌────────────────────────┐
           │                 │  qode knowledge         │
           │                 │  add-branch [branches]  │
           │                 │  (internal/cli)          │
           │                 └──────────┬─────────────┘
           │                            │
           │              ┌─────────────┼──────────────┐
           │              ▼             ▼              ▼
           │     ┌──────────┐  ┌──────────────┐  ┌──────────┐
           │     │ context   │  │ knowledge    │  │ git      │
           │     │ .Load()   │  │ .ListLessons │  │ .Diff    │
           │     └──────────┘  └──────────────┘  └──────────┘
           │                            │
           │                            ▼
           │                 ┌────────────────────────┐
           │                 │  prompt.Engine.Render() │
           │                 │  knowledge/add-branch   │
           │                 └──────────┬─────────────┘
           │                            │
           │                            ▼
           │                 ┌────────────────────────┐
           │                 │  dispatch.RunInteractive│
           │                 │  (launches claude CLI)  │
           │                 └──────────┬─────────────┘
           │                            │
           ▼                            ▼
┌──────────────────────────────────────────────────────┐
│              AI writes lesson files                   │
│   .qode/knowledge/lessons/<kebab-case-title>.md      │
└──────────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────┐
│  knowledge.Load() discovers lessons via recursive    │
│  listDir() → injected into future prompts as {{.KB}} │
└──────────────────────────────────────────────────────┘
```

### Affected Layers

| Layer | Component | Change Type |
|---|---|---|
| Knowledge | `internal/knowledge/knowledge.go` | Modify (`listDir`, `Load`) + add functions |
| Prompt | `internal/prompt/engine.go` | Modify (`TemplateData`) |
| Prompt | `internal/prompt/templates/knowledge/` | New (2 templates) |
| CLI | `internal/cli/knowledge_cmd.go` | Modify + add command |
| IDE | `internal/ide/claudecode.go` | Modify (slash commands, `buildClaudeMD`) |
| IDE | `internal/ide/cursor.go` | Modify (slash commands, `workflowRule`) |
| Docs | `CLAUDE.md` | Modify (workflow step) |

### Data Flow

**`/qode-knowledge-add-context` (slash command):**
1. User invokes slash command in IDE
2. IDE AI receives inline prompt with instructions
3. AI runs `qode knowledge list` to see existing lessons
4. AI reflects on session conversation history
5. AI writes 1-5 lesson files to `.qode/knowledge/lessons/`

**`qode knowledge add-branch` (terminal):**
1. CLI parses branch names from args (comma-split + positional)
2. Loads branch context via `context.Load()` + direct reads for review files
3. Gets git diff via `git.DiffFromBase()`
4. Loads existing lessons via `knowledge.ListLessons()`
5. Renders `knowledge/add-branch` template with all data
6. Dispatches via `dispatch.RunInteractive()` (or writes prompt file with `--prompt-only`)
7. AI writes lesson files to `.qode/knowledge/lessons/`

**`/qode-knowledge-add-branch` (slash command):**
1. User invokes slash command with branch name(s)
2. Slash command runs `qode knowledge add-branch --prompt-only $ARGUMENTS`
3. AI reads generated prompt file
4. AI writes lesson files

## 4. API / Interface Contracts

### CLI Command

```
qode knowledge add-branch [branches...] [--prompt-only]
```

**Arguments:**
- `branches` — One or more branch names (positional args and/or comma-separated)
  - `qode knowledge add-branch feat-a feat-b` — two branches as separate args
  - `qode knowledge add-branch feat-a,feat-b` — two branches comma-separated
  - `qode knowledge add-branch feat-a,feat-b feat-c` — mixed (3 branches)

**Flags:**
- `--prompt-only` — Write prompt file without dispatching to claude CLI

**Output (normal mode):**
```
Extracting lessons from branches: feat-a, feat-b
Prompt dispatched to claude CLI.
```

**Output (--prompt-only):**
```
Lesson extraction prompt written to:
  .qode/branches/feat-lessons-learned/.knowledge-add-branch-prompt.md

Use slash command: /qode-knowledge-add-branch
```

**Errors:**
- `branch context not found: .qode/branches/<name>/` — Branch has no context directory (warning, continues with git diff only)
- `no branches provided` — No arguments (cobra enforces `MinimumNArgs(1)`)
- `claude CLI not found` — Claude binary not on PATH (suggest `--prompt-only`)

### Knowledge Package Functions

```go
// LessonsDir returns the lessons subdirectory path.
func LessonsDir(root string) string

// ListLessons returns summaries of all existing lessons for dedup.
func ListLessons(root string) ([]LessonSummary, error)

// SaveLesson writes a lesson file with kebab-case filename.
// Handles filename collision by appending -2, -3, etc.
func SaveLesson(root, title, content string) error

// ToKebabCase converts a string to kebab-case for filenames.
func ToKebabCase(s string) string
```

```go
type LessonSummary struct {
    Title   string // parsed from ### heading
    Path    string // absolute file path
    Summary string // first paragraph after title
}
```

### TemplateData Addition

```go
type TemplateData struct {
    // ... existing fields ...
    KB         string
    Lessons    string // NEW: compact lesson listing for dedup
    OutputPath string
}
```

### Lesson File Format

**Filename:** `<kebab-case-title>.md` (e.g., `avoid-nested-goroutines-in-handlers.md`)

**Content:**
```markdown
### Avoid nested goroutines in HTTP handlers
When spawning goroutines inside HTTP handlers, always pass context and use errgroup for lifecycle management. Naked goroutines leak when the request is cancelled and produce race conditions in tests. This applies to any handler that performs concurrent I/O operations.

**Example 1:** Incorrect — goroutine leak on cancelled request
```go
func handler(w http.ResponseWriter, r *http.Request) {
    go fetchData() // leaked if request cancelled
}
```

**Example 2:** Correct — use errgroup with context
```go
func handler(w http.ResponseWriter, r *http.Request) {
    g, ctx := errgroup.WithContext(r.Context())
    g.Go(func() error { return fetchData(ctx) })
    if err := g.Wait(); err != nil { ... }
}
```
```

## 5. Data Model Changes

### File System Structure

**New directory:**
```
.qode/knowledge/lessons/
├── avoid-nested-goroutines-in-handlers.md
├── always-validate-config-before-dispatch.md
└── use-table-driven-tests-for-edge-cases.md
```

**No database, migration, or backward compatibility concerns.** The knowledge system is file-based. The only structural change is that `listDir()` now recurses into subdirectories, which is additive — existing flat-directory layouts continue to work identically.

### `Load()` Output Change

**Before (all files):**
```
### my-file.md

<file content>

---
```

**After (lesson files skip the filename header):**
```
### Avoid nested goroutines in handlers
<lesson content with examples>

---
```

**After (non-lesson files unchanged):**
```
### my-file.md

<file content>

---
```

## 6. Implementation Tasks

- [ ] **Task 1:** (knowledge) Make `listDir()` recursive using `filepath.WalkDir()` — skip hidden dirs, collect all files. Modify `Load()` to skip `### filename` header for files in `lessons/` subdirectory. Add tests for both behaviors.

- [ ] **Task 2:** (knowledge) Add `LessonSummary` type, `LessonsDir()`, `ListLessons()`, `SaveLesson()`, `ToKebabCase()`, and `parseLessonHeader()` helper. Add `fileExists()` helper for collision detection. Add tests: save, list, kebab-case conversion, collision handling.

- [ ] **Task 3:** (prompt) Add `Lessons string` field to `TemplateData` struct in `internal/prompt/engine.go`.

- [ ] **Task 4:** (prompt) Create `internal/prompt/templates/knowledge/add-context.md.tmpl` and `internal/prompt/templates/knowledge/add-branch.md.tmpl` with lesson format spec, dedup instructions, and file-write instructions.

- [ ] **Task 5:** (cli) Add `newKnowledgeAddBranchCmd()` with `parseBranchArgs()` helper. Register in `newKnowledgeCmd()`. Implementation: load context + reviews + diff, list existing lessons, render template, dispatch or write prompt file.

- [ ] **Task 6:** (ide) Add `qode-knowledge-add-context` and `qode-knowledge-add-branch` to `claudeSlashCommands()` in `claudecode.go` and `slashCommands()` in `cursor.go`.

- [ ] **Task 7:** (docs) Update `buildClaudeMD()` and `workflowRule()` to add recommended lesson-extraction step after reviews. Manually update `CLAUDE.md`.

- [ ] **Task 8:** (test) Add IDE tests verifying new slash commands appear in generated output. Verify `qode ide setup` produces the expected command files.

## 7. Testing Strategy

### Unit Tests (`internal/knowledge/knowledge_test.go`)

| Test | Description |
|---|---|
| `TestListDir_Recursive` | Create `.qode/knowledge/` with files in root and `lessons/` subdirectory. Verify all files found. |
| `TestListDir_SkipsHiddenDirs` | Create `.hidden/` subdirectory with files. Verify skipped. |
| `TestListDir_EmptyDir` | Empty knowledge dir returns empty slice, no error. |
| `TestLoad_LessonFilesNoDoubleHeader` | Create lesson file with `### Title`. Verify `Load()` output has one heading, not two. |
| `TestLoad_RegularFilesKeepHeader` | Create non-lesson file. Verify `Load()` output has `### filename` header. |
| `TestLoad_MixedFiles` | Mix of regular KB files and lesson files. Verify correct headers for each. |
| `TestSaveLesson_WritesCorrectFile` | Save lesson, verify filename is kebab-case, content matches. |
| `TestSaveLesson_CreatesDir` | Save lesson when `lessons/` doesn't exist. Verify dir created. |
| `TestSaveLesson_HandlesCollision` | Save two lessons with same title. Verify second gets `-2` suffix. |
| `TestListLessons_ParsesTitleAndSummary` | Create lesson files, verify parsed titles and summaries. |
| `TestListLessons_EmptyDir` | No lessons dir returns nil, nil. |
| `TestToKebabCase_Spaces` | `"Hello World"` → `"hello-world"` |
| `TestToKebabCase_SpecialChars` | `"Don't use this!"` → `"don-t-use-this"` |
| `TestToKebabCase_ConsecutiveHyphens` | `"foo---bar"` → `"foo-bar"` |
| `TestToKebabCase_NonASCII` | `"über cool"` → `"ber-cool"` (or similar) |

### Integration Tests

| Test | Description |
|---|---|
| `TestKnowledgeList_IncludesLessons` | End-to-end: save lesson, run `knowledge.List()`, verify lesson appears. |
| `TestKnowledgeSearch_FindsInLessons` | Save lesson with keyword, run `knowledge.Search()`, verify match. |

### IDE Tests (`internal/ide/ide_test.go`)

| Test | Description |
|---|---|
| `TestClaudeSlashCommands_IncludesKnowledge` | Verify `claudeSlashCommands()` output contains both new command keys. |
| `TestCursorSlashCommands_IncludesKnowledge` | Verify `slashCommands()` output contains both new command keys. |

### Edge Cases to Test Explicitly
- Lesson file with no `###` heading (graceful fallback in `parseLessonHeader`)
- Lesson file with empty content
- Branch name with slashes (`feat/sub/feature`)
- Branch name with `..` (should be rejected or handled safely)
- Very long lesson title (verify filename truncation or handling)

## 8. Security Considerations

### Input Validation
- **Branch names:** Validated against directory traversal — reject names containing `..`. Branch names are used in `filepath.Join()` which normalizes paths, but explicit validation prevents `.qode/branches/../../etc/passwd` style attacks.
- **Lesson content:** Written by AI, not directly by user input. No injection risk since files are Markdown stored locally.

### Secrets Prevention
- Prompt templates explicitly instruct: "Do NOT include credentials, API keys, tokens, or other secrets in lessons."
- Git diffs may contain secrets — the prompt prioritizes structured context (ticket, spec, reviews) and treats the diff as supplementary. Diff truncation (500 lines max) limits exposure.

### Data Sensitivity
- Lesson files are local (`.qode/knowledge/lessons/`), not transmitted to any service
- Whether lessons are committed to git is the user's decision (`.qode/` can be gitignored)
- No PII handling required

## 9. Open Questions

None — all ambiguities were resolved during requirements refinement (iteration 3, score 25/25).

---
*Spec generated by qode. Copy to ticket system for team review.*
