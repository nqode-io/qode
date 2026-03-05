<!-- qode:iteration=1 score=18/25 -->

# Refined Requirements Analysis — Lessons Learned Feature

## 1. Problem Understanding

### Problem Restatement
qode currently supports a project-wide knowledge base (`qode knowledge add [path]`) that stores static documents in `.qode/knowledge/`. The feature request asks for two new commands that **automatically extract lessons learned** from AI-assisted development sessions and add them to this knowledge base:

1. **`/qode-knowledge-add-context`** — A slash-command-only action that summarizes the current IDE session context (conversation history, decisions made, problems encountered) and distills lessons learned from it.
2. **`/qode-knowledge-add-branch [branch1,branch2,...]`** — Works as both a terminal command (`qode knowledge add-branch`) and a slash command. It reads all relevant files from one or more branches (diffs, reviews, specs, tickets), summarizes them, and extracts lessons learned.

Each lesson learned is stored as an individual Markdown file in `.qode/knowledge/` with a specific format: kebab-case filename, heading, description (≤100 words), and optional positive/negative examples. Lessons must be deduplicated against existing ones.

### User Need & Business Value
- **Institutional memory** — Teams accumulate knowledge across sprints without manual documentation effort.
- **Reduced rework** — Lessons are injected into future prompts via `{{.KB}}`, so the AI avoids repeating past mistakes.
- **Lower token cost** — Fewer iteration cycles when the AI already knows project-specific pitfalls.

### Ambiguities & Open Questions
1. **Deduplication strategy** — The ticket says lessons must be "distinct." Should deduplication be semantic (AI-driven comparison) or title-based? **Recommendation:** Use AI-driven comparison during the extraction prompt — instruct the model to read existing lessons and only produce new, non-overlapping ones.
2. **Knowledge base subdirectory** — Should lessons live directly in `.qode/knowledge/` alongside other KB files, or in a subdirectory like `.qode/knowledge/lessons/`? **Recommendation:** Use `.qode/knowledge/lessons/` to keep them organized and distinguishable from manually-added KB files.
3. **Branch file scope for `add-branch`** — "All relevant files" is ambiguous. Which files? **Recommendation:** Read the branch context directory (`.qode/branches/$BRANCH/`) which includes ticket, notes, refined-analysis, spec, code-review, and security-review files, plus the git diff of the branch against main.
4. **Session context for `add-context`** — How is "current session context" accessed? In a slash command, the AI has the full conversation history. The prompt should instruct the AI to reflect on the session and extract lessons. No programmatic context extraction is needed.

---

## 2. Technical Analysis

### Affected Layers/Components

| Component | Change Type | Details |
|---|---|---|
| `internal/cli/knowledge_cmd.go` | **Modify** | Add `add-branch` subcommand to existing `knowledge` command group |
| `internal/knowledge/knowledge.go` | **Modify** | Add `LoadLessons()` to list existing lesson files for dedup; add `SaveLesson()` to write individual lesson files |
| `internal/prompt/templates/knowledge/` | **New** | Add `add-context.md.tmpl` and `add-branch.md.tmpl` templates |
| `.claude/commands/qode-knowledge-add-context.md` | **New** | Slash command definition for Claude Code |
| `.claude/commands/qode-knowledge-add-branch.md` | **New** | Slash command definition for Claude Code |
| `.cursor/commands/qode-knowledge-add-context.md` | **New** | Slash command definition for Cursor (if IDE commands directory exists) |
| `.cursor/commands/qode-knowledge-add-branch.md` | **New** | Slash command definition for Cursor |
| `internal/context/context.go` | **Minor modify** | May need helper to load context from arbitrary branch (not just current) |
| `CLAUDE.md` | **Modify** | Update workflow documentation to add recommended step after reviews |
| `internal/cli/plan.go` or equivalent | **No change** | Prompt building follows existing patterns |

### Key Technical Decisions

1. **Lesson file storage path:** `.qode/knowledge/lessons/<kebab-case-title>.md` — This keeps lessons organized while still being auto-discovered by the existing `knowledge.Load()` function (which walks `.qode/knowledge/` recursively).

2. **Terminal command structure:** `qode knowledge add-branch <branch1> [branch2] [branch3]` — Follows the existing `qode knowledge add <path>` pattern. Branch names as positional args, comma-separated also accepted per ticket.

3. **Slash command pattern:** The slash commands should generate a prompt that:
   - Lists existing lessons (for dedup awareness)
   - Provides the relevant context (session or branch files)
   - Instructs the AI to output lessons in the specified format
   - Instructs the AI to write each lesson to its own file

4. **Prompt execution:** For the terminal `add-branch` command, use `dispatch.RunInteractive()` to launch claude with the rendered prompt (same pattern as `qode start`). For slash commands, the prompt is executed directly by the IDE's AI.

### Patterns & Conventions to Follow
- CLI commands use Cobra, registered in `internal/cli/root.go` via `newKnowledgeCmd()`
- Prompts are Go templates in `internal/prompt/templates/` with `.md.tmpl` extension
- Template data uses `prompt.TemplateData` struct
- Slash commands in `.claude/commands/` reference `qode` CLI for prompt generation
- File I/O uses `os.ReadFile`/`os.WriteFile` with `0644` permissions
- Branch context loaded via `context.Load(root, branch)`

### Dependencies
- Existing `knowledge.Load()` function — must continue to work with new lesson files in subdirectory
- `context.Load()` — used to gather branch context for `add-branch`
- `dispatch.RunInteractive()` — used for terminal command execution
- Git CLI — needed to compute diff for branch summary

---

## 3. Risk & Edge Cases

### What Could Go Wrong
1. **Duplicate lessons** — Without proper dedup, repeated runs could create overlapping lessons. **Mitigation:** Prompt includes all existing lesson titles/summaries; instruct model to skip duplicates.
2. **Empty context** — Running `add-context` at session start with no meaningful work done. **Mitigation:** Prompt should handle gracefully; if no lessons can be extracted, output a message saying so.
3. **Invalid branch names** — User provides non-existent branch name to `add-branch`. **Mitigation:** Validate branch exists (check `.qode/branches/<name>/` directory or `git branch --list`).
4. **Large diffs** — Branch with hundreds of changed files could exceed context window. **Mitigation:** Summarize diff rather than including raw diff; or limit to key files (reviews, specs, refined analysis).

### Edge Cases
- Branch with no context files (no ticket, no reviews) — should still work by reading git diff
- Multiple branches with overlapping lessons — dedup across all
- Lesson title collision (two different lessons generate same kebab-case filename) — append numeric suffix
- Existing lesson file with same name but different content — skip or prompt for overwrite
- Non-ASCII characters in lesson titles — sanitize for filename

### Security Considerations
- No user input is passed to shell commands unsanitized (branch names validated)
- Lesson files are local only, no network exposure
- No secrets should appear in lessons (git diffs might contain them) — prompt should instruct model to exclude sensitive data

### Performance Implications
- Loading all existing lessons for dedup comparison adds I/O but is negligible (small files)
- Large branch diffs may require truncation to fit context window
- No persistent processes or background jobs

---

## 4. Completeness Check

### Acceptance Criteria (Derived from Ticket)

| # | Criterion | Source |
|---|---|---|
| AC1 | `/qode-knowledge-add-context` slash command exists for Claude Code | Ticket |
| AC2 | `/qode-knowledge-add-context` works in Cursor and other IDEs | Ticket ("equivalents in all IDEs") |
| AC3 | `/qode-knowledge-add-context` summarizes current session and extracts lessons | Ticket |
| AC4 | `qode knowledge add-branch <branches>` terminal command exists | Ticket |
| AC5 | `/qode-knowledge-add-branch` slash command exists for all IDEs | Ticket |
| AC6 | `add-branch` accepts one or more branch names (comma-separated) | Ticket |
| AC7 | `add-branch` reads relevant branch files and extracts lessons | Ticket |
| AC8 | Lessons are deduplicated against existing lessons | Ticket |
| AC9 | Each lesson is stored in its own file with kebab-case filename | Ticket |
| AC10 | Lesson content follows the specified format (heading, description ≤100 words, optional examples) | Ticket |
| AC11 | Documentation updated with recommended post-review step | Ticket |
| AC12 | Template prompt(s) added for lesson extraction | Ticket |

### Implicit Requirements
- Existing `knowledge.Load()` must still discover and include lesson files in prompts
- Lesson files must be valid Markdown
- The `qode knowledge list` command should show lesson files
- Error handling for missing branches, empty context, filesystem errors

### Out of Scope
- Automatic lesson extraction (without explicit user trigger)
- Lesson categorization or tagging
- Lesson editing or deletion commands (can be done manually via filesystem)
- Cross-project lesson sharing
- Lesson quality scoring

---

## 5. Actionable Implementation Plan

### Prerequisites
- None — all required infrastructure (knowledge base, prompt engine, dispatch, CLI framework) already exists.

### Implementation Tasks (in order)

**Task 1: Add `lessons/` subdirectory support to knowledge package**
- File: `internal/knowledge/knowledge.go`
- Add `LessonsDir()` function returning `.qode/knowledge/lessons/`
- Add `ListLessons(root string)` to return existing lesson filenames and titles
- Add `SaveLesson(root string, title string, content string)` to write a lesson file
- Ensure `Load()` recursively walks subdirectories (verify this already works)

**Task 2: Create lesson extraction prompt template**
- File: `internal/prompt/templates/knowledge/add-context.md.tmpl`
- Template instructs AI to: review session context, identify lessons, check against existing lessons (injected), output each as a separate file write
- File: `internal/prompt/templates/knowledge/add-branch.md.tmpl`
- Template instructs AI to: review branch context (ticket, analysis, spec, reviews, diff), identify lessons, check against existing, output file writes
- Both templates include the lesson file format specification from the ticket

**Task 3: Add `knowledge add-branch` terminal command**
- File: `internal/cli/knowledge_cmd.go`
- Add `add-branch` subcommand accepting branch names (positional args, also support comma-separated)
- Load context for each branch via `context.Load()`
- Load existing lessons for dedup
- Render `knowledge/add-branch.md.tmpl` with branch context + existing lessons
- Dispatch interactively via `dispatch.RunInteractive()`

**Task 4: Create slash command for `/qode-knowledge-add-context`**
- File: `.claude/commands/qode-knowledge-add-context.md`
- Instructions for the AI to: run `qode knowledge list` to see existing lessons, summarize session context, extract lessons, write each to `.qode/knowledge/lessons/<name>.md`
- File: `.cursor/commands/qode-knowledge-add-context.md` (if Cursor commands directory exists, otherwise follow IDE setup pattern)

**Task 5: Create slash command for `/qode-knowledge-add-branch`**
- File: `.claude/commands/qode-knowledge-add-branch.md`
- Instructions to run `qode knowledge add-branch <branch>` which handles everything
- File: `.cursor/commands/qode-knowledge-add-branch.md`

**Task 6: Update documentation**
- File: `CLAUDE.md` — Add step after reviews recommending `/qode-knowledge-add-context`
- File: Any other workflow documentation (README, help text) — Update workflow steps
- Update `qode help workflow` output if applicable

**Task 7: Add tests**
- Test `SaveLesson()` writes correct filename and content
- Test `ListLessons()` returns existing lessons
- Test filename sanitization (kebab-case conversion, special characters)
- Test `add-branch` command with valid and invalid branch names
- Test that `Load()` includes lessons from subdirectory
