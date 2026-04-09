<!-- qode:iteration=4 score=25/25 -->
# Refined Analysis: qode init — Append Gitignore Rules

## 1. Problem Understanding

`qode init` sets up `.qode/` directory structure, writes `qode.yaml`, copies templates, and generates IDE configs, but never touches `.gitignore`. Developers either manually add ignore rules or accidentally commit ephemeral branch artifacts (scored analysis snapshots, fetched tickets, temp prompt files, diff summaries) into git history.

**User need:** Run `qode init` once and have `.gitignore` automatically configured so no qode-generated ephemeral files appear in `git status`.

**Business value:** Removes a manual, error-prone setup step; prevents noisy git history; makes the tool fully self-contained on first run.

**Ambiguities resolved (from notes):**
- The notes expand the rule list beyond the ticket: adds `.qode/branches/*/diff.md` and `.qode/prompts/scaffold/` — notes are authoritative.
- The notes require per-rule idempotency ("each one must be checked individually"), not just a one-shot block-marker guard.
- The notes say rules should live in "a config file or template" — resolved as package-level exported vars in `internal/scaffold/gitignore.go`.
- The ticket says print to stderr; the existing `runInitExisting` exclusively uses `io.Writer out` (callers wire to stdout). For consistency with all other init output, the confirmation is written to `out`. This is an intentional deviation from ticket wording.

---

## 2. Technical Analysis

### Layers/components affected

| Component | Change |
|---|---|
| `internal/scaffold/gitignore.go` | **New file** — `GitignoreMarker`, `GitignoreRules`, `AppendGitignoreRules` |
| `internal/scaffold/gitignore_test.go` | **New file** — unit tests |
| `internal/cli/init.go` `runInitExisting` | **Modified** — add `"context"` import, call `scaffold.AppendGitignoreRules` |
| `internal/cli/init_test.go` | **Modified** — two new tests |

Dependency layer: `cli` already imports `scaffold`; `scaffold` already imports `iokit`. No new deps, no circular imports.

**No new CLI flags are introduced.** The `init` command signature and its `RunE` handler are unchanged. No prompt templates are added or modified.

**IDE integrations (Cursor, Claude Code) are unaffected.** `scaffold.Setup` (line 91) is the sole IDE-touching step; it generates `.cursor/rules/` and `.claude/commands/`. `AppendGitignoreRules` runs after `scaffold.Setup` returns and only writes to `.gitignore`. Neither Cursor nor Claude Code reads `.gitignore`, so neither integration is affected.

**Override system unaffected.** The template override system (`.qode/prompts/` → embedded fallback) is not touched. The rule `.qode/prompts/scaffold/` added to `.gitignore` prevents scaffold prompt overrides from being committed, but the directory itself is already removed by `os.RemoveAll(scaffoldPromptsDir)` at line 99. Adding the rule to `.gitignore` is belt-and-suspenders: it guards against partial `qode init` runs where `scaffold.Setup` succeeds but the `os.RemoveAll` step has not yet executed.

### Key technical decisions

**D1 — Context threading (CLAUDE.md requirement):** `AppendGitignoreRules` performs file I/O, so its signature must be `AppendGitignoreRules(ctx context.Context, out io.Writer, root string) error`. Use `iokit.WriteFileCtx` for the write. `runInitExisting` does not accept context; the caller passes `context.Background()` per CLAUDE.md ("callers without a context pass `context.Background()`"). The signature of `runInitExisting` itself is NOT changed.

**D2 — Location of rules:** Package-level exported vars in `internal/scaffold/gitignore.go`:
```go
const GitignoreMarker = "# qode temp files"

var GitignoreRules = []string{
    ".qode/branches/*/.*.md",
    ".qode/branches/*/context/ticket.md",
    ".qode/branches/*/refined-analysis-*-score-*.md",
    ".qode/branches/*/diff.md",
    ".qode/prompts/scaffold/",
}
```

**D3 — Per-rule idempotency:** Check each rule individually with `strings.Contains(content, rule)`. Do not rely solely on the block marker. This handles the case where the user has deleted individual rules from an existing block.

**D4 — Block structure logic:**
- If marker absent: append `"\n" + GitignoreMarker + "\n" + strings.Join(allRules, "\n") + "\n"` to existing content.
- If marker present but some rules missing: append `"\n" + strings.Join(missingRules, "\n") + "\n"` to content (missing rules land after the existing block; no duplication of the marker).
- If all rules present: no-op, no output.

**D5 — File creation:** If `.gitignore` does not exist, `os.ReadFile` returns an `os.IsNotExist` error; treat content as `""` and proceed as "marker absent" case. `iokit.WriteFileCtx` creates parent directories — the parent is always the repo root, which exists, so no issue.

**D6 — Output:** Single confirmation message printed to `out` when any rules are appended: `"Appended qode ignore rules to .gitignore\n"`. Silent on no-op. The message does not contain "Detected", "Scanning", or "qode ide setup", so it does not violate `TestRunInitExisting_NoDetectionOutput`.

**D7 — Write primitive:** `iokit.WriteFileCtx(ctx, path, []byte(newContent), 0644)` — context-aware; not `AtomicWrite` since `.gitignore` is not consumed by a subsequent workflow step.

**D8 — Trailing newline:** If existing `.gitignore` content is non-empty and does not end with `\n`, prepend `\n` before the appended block to avoid merging with the last existing line.

**D9 — Call placement in `runInitExisting`:** Insert after `scaffold.Setup` returns successfully (line 91–93), before `os.RemoveAll` (line 95). This ordering ensures IDE configs are fully generated before `.gitignore` is touched, and any `.gitignore` error is reported before the scaffold prompts cleanup and "Next steps:" message.

### Patterns/conventions to follow

- `t.Helper()`, `t.TempDir()`, `t.Parallel()`, table-driven tests with `t.Run(tc.name, ...)`
- `%w` for all error wrapping
- Named constants/exported vars (no inline magic strings in logic functions)
- Functions ≤ 50 lines, single responsibility

### Dependencies

None beyond stdlib (`context`, `fmt`, `io`, `os`, `path/filepath`, `strings`) and `iokit` (already in scaffold's transitive deps).

---

## 3. Risk & Edge Cases

| Risk / Edge Case | Mitigation |
|---|---|
| `.gitignore` does not exist | `os.IsNotExist` → treat content as empty; create via `iokit.WriteFileCtx` |
| `.gitignore` is read-only | `iokit.WriteFileCtx` returns error; propagate with `"updating .gitignore: %w"` |
| Marker present, all rules present | Per-rule check detects all present; return nil with no output |
| Marker present, some rules missing | `strings.Contains` on each rule individually; only missing ones appended, no marker duplication |
| `.gitignore` ends without trailing newline | Check `!strings.HasSuffix(content, "\n")` before appending; if true, prepend `"\n"` |
| Re-running `qode init` multiple times | Per-rule idempotency ensures no duplicate lines regardless of how many runs |
| Rule strings contain glob chars (`*`, `?`) | Stored as exact literals; `strings.Contains` matches the literal string — correct for `.gitignore` syntax |
| Windows line endings (`\r\n`) in existing file | `strings.Contains` works for substring matching; appended lines use `\n` (standard for `.gitignore`) |
| Context cancelled before write | `iokit.WriteFileCtx` propagates context error before attempting write |
| `.gitignore` is a directory (unusual) | `os.ReadFile` returns an error; propagate with `"reading .gitignore: %w"` |
| User has customized the `# qode temp files` block | Per-rule check only appends missing rules; user additions are untouched |
| Running `qode init` outside a git repo | No special handling needed (Q2 in notes confirmed: no). The function only reads/writes a `.gitignore` file relative to `root`; it does not require or check git state. |

**Security:** No user input is interpolated into file content. Rules are hardcoded constants. No injection risk.

**Performance:** Reads/writes one small file (typically < 2 KB). Negligible.

---

## 4. Completeness Check

### Acceptance criteria (ticket + notes)

1. `qode init` creates `.gitignore` if absent, writes all 5 rules under `# qode temp files`.
2. `qode init` on existing `.gitignore` with no marker appends the full block.
3. Re-running when all 5 rules are present is a silent no-op.
4. Re-running when marker is present but some rules are missing appends only the missing ones.
5. All 5 rules are defined in a single authoritative location in `internal/scaffold/gitignore.go`.
6. Confirmation message printed to `out` when any rules are appended; silent on no-op.
7. Errors propagated with `%w` wrapping.
8. `AppendGitignoreRules` accepts `context.Context` and forwards it to `iokit.WriteFileCtx`.

### Implicit requirements

- All existing `init_test.go` tests must continue to pass — no behavior regression.
- `TestRunInitExisting_NoDetectionOutput` checks forbidden strings ("Detected", "Scanning", "qode ide setup") and required strings ("Generated:", "Next steps:"). The new message "Appended qode ignore rules to .gitignore" passes both checks.
- `AppendGitignoreRules` must be called after `scaffold.Setup` so IDE setup errors are reported before `.gitignore` is touched.

### Explicitly out of scope

- Removing rules from `.gitignore` on uninstall or `qode deinit`.
- Managing `.gitignore` in subdirectories or `.gitignore_global`.
- Modifying or validating a `# qode temp files` block the user has customized.
- Running `qode init` in a non-git directory requires no special handling (Q2: confirmed no-op on git state).

---

## 5. Actionable Implementation Plan

### Task 1 — `internal/scaffold/gitignore.go` (new file)

```go
package scaffold

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/nqode/qode/internal/iokit"
)

const GitignoreMarker = "# qode temp files"

var GitignoreRules = []string{
    ".qode/branches/*/.*.md",
    ".qode/branches/*/context/ticket.md",
    ".qode/branches/*/refined-analysis-*-score-*.md",
    ".qode/branches/*/diff.md",
    ".qode/prompts/scaffold/",
}

// AppendGitignoreRules adds qode-specific patterns to .gitignore in root.
// Each rule is checked individually; only missing rules are appended.
// Prints a confirmation to out when rules are written; silent on no-op.
func AppendGitignoreRules(ctx context.Context, out io.Writer, root string) error {
    path := filepath.Join(root, ".gitignore")
    data, err := os.ReadFile(path)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("reading .gitignore: %w", err)
    }
    content := string(data)

    var missing []string
    for _, rule := range GitignoreRules {
        if !strings.Contains(content, rule) {
            missing = append(missing, rule)
        }
    }
    if len(missing) == 0 {
        return nil
    }

    var block strings.Builder
    if content != "" && !strings.HasSuffix(content, "\n") {
        block.WriteString("\n")
    }
    if !strings.Contains(content, GitignoreMarker) {
        block.WriteString(GitignoreMarker + "\n")
    }
    for _, rule := range missing {
        block.WriteString(rule + "\n")
    }

    newContent := content + block.String()
    if err := iokit.WriteFileCtx(ctx, path, []byte(newContent), 0644); err != nil {
        return fmt.Errorf("updating .gitignore: %w", err)
    }
    _, _ = fmt.Fprintln(out, "Appended qode ignore rules to .gitignore")
    return nil
}
```

### Task 2 — `internal/scaffold/gitignore_test.go` (new file)

Table-driven, parallel tests covering:

- `no .gitignore` → all 5 rules present in new file, marker present, output contains "Appended"
- `.gitignore exists, no marker` → full block appended, all rules present, output contains "Appended"
- `.gitignore exists, all rules present` → no-op, file unchanged, output empty
- `.gitignore exists, marker present, 2 rules missing` → only missing rules appended, no duplicates, output contains "Appended"
- `.gitignore exists, no trailing newline` → appended block starts on new line (not merged with last line)

Each test asserts presence of every rule individually via `strings.Contains` (sentinel assertions). Idempotency test uses `strings.Count(content, rule) == 1` for each rule.

### Task 3 — `internal/cli/init.go` — wire the call

Add `"context"` to the import block (currently missing from `init.go`).

After the `scaffold.Setup` block (lines 91–93), before `os.RemoveAll` (line 95):

```go
if err := scaffold.AppendGitignoreRules(context.Background(), out, root); err != nil {
    return err
}
```

### Task 4 — `internal/cli/init_test.go` — two new tests

- `TestRunInitExisting_AppendsGitignoreRules`: call `runInitExisting` on a fresh `t.TempDir()`; read `.gitignore`; assert each of the 5 rules is present via `strings.Contains`.
- `TestRunInitExisting_GitignoreIsIdempotent`: call `runInitExisting` twice on the same dir; read `.gitignore`; for each rule, assert `strings.Count(content, rule) == 1`.

### Implementation order

1. Task 1 (`scaffold/gitignore.go`) — pure logic, no external task dependencies
2. Task 2 (`scaffold/gitignore_test.go`) — validates Task 1 in isolation
3. Task 3 (`cli/init.go`) — wires end-to-end
4. Task 4 (`cli/init_test.go`) — end-to-end validation

Each task is independently committable with passing tests.
