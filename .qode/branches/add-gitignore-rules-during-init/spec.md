# Technical Specification: qode init — Append Gitignore Rules

*Branch: add-gitignore-rules-during-init*
*Generated: 2026-04-09*

---

## 1. Feature Overview

`qode init` sets up the `.qode/` directory structure, writes `qode.yaml`, copies templates, and generates IDE configs, but currently never touches `.gitignore`. This means developers must manually add ignore rules after initialization or risk accidentally committing ephemeral qode-generated artifacts (scored analysis snapshots, fetched tickets, diff summaries, scaffold prompts) into git history.

This feature extends `qode init` to automatically append qode-specific `.gitignore` patterns on first run, and to idempotently repair a partial configuration on re-runs. The implementation is purely additive: no existing flags, commands, or templates change.

**Business value:** Removes a manual, error-prone setup step; prevents noisy git history from ephemeral qode artifacts; makes the tool fully self-contained on first run.

**Success criteria:**
- Running `qode init` on a fresh repo creates or updates `.gitignore` with all 5 qode patterns.
- Re-running on a fully configured repo is a silent no-op.
- Re-running on a partially configured repo appends only missing rules without duplication.
- All existing `qode init` tests continue to pass.

---

## 2. Scope

### In scope
- New `internal/scaffold/gitignore.go` with exported `GitignoreMarker`, `GitignoreRules`, and `AppendGitignoreRules`.
- New `internal/scaffold/gitignore_test.go` with full unit coverage.
- Modification of `internal/cli/init.go` (`runInitExisting`) to call `AppendGitignoreRules`.
- Two new integration-style tests in `internal/cli/init_test.go`.
- Per-rule idempotency: each of the 5 rules is checked and appended individually.

### Out of scope
- Removing rules from `.gitignore` on uninstall or `qode deinit`.
- Managing `.gitignore` in subdirectories or global gitignore.
- Modifying or validating a `# qode temp files` block the user has customized.
- Checking whether `root` is a git repository (no special handling required).
- New CLI flags, subcommands, or prompt templates.
- Changes to Cursor or Claude Code IDE integrations.

### Assumptions
- The `iokit.WriteFileCtx` function already exists and creates parent directories if needed.
- `internal/cli` already imports `internal/scaffold`; no new dependency edges are introduced.
- The repo root (`root`) always exists when `runInitExisting` is called.
- `.gitignore` is a plain text file; binary or directory-named `.gitignore` entries are edge cases handled via error propagation.

---

## 3. Architecture & Design

### Component diagram

```
internal/cli/init.go
  └── runInitExisting(root, out)
        ├── scaffold.Setup(...)          [existing — IDE configs]
        ├── scaffold.AppendGitignoreRules(ctx, out, root)   [NEW]
        └── os.RemoveAll(scaffoldPromptsDir)                [existing]

internal/scaffold/gitignore.go          [NEW FILE]
  ├── const GitignoreMarker
  ├── var   GitignoreRules
  └── func  AppendGitignoreRules(ctx, out, root) error
        ├── os.ReadFile(".gitignore")
        ├── per-rule strings.Contains check
        ├── block assembly
        └── iokit.WriteFileCtx(ctx, path, content, 0644)
```

### Layers affected

| Layer | File | Change |
|---|---|---|
| Domain (`scaffold`) | `internal/scaffold/gitignore.go` | **New file** |
| Domain (`scaffold`) | `internal/scaffold/gitignore_test.go` | **New file** |
| Top-level (`cli`) | `internal/cli/init.go` | **Modified** — one new call site |
| Top-level (`cli`) | `internal/cli/init_test.go` | **Modified** — two new tests |

No new dependency edges. `scaffold` already transitively depends on `iokit`. `cli` already imports `scaffold`.

### Data flow

1. `runInitExisting` calls `scaffold.Setup` (IDE configs generated).
2. `runInitExisting` calls `scaffold.AppendGitignoreRules(context.Background(), out, root)`.
3. `AppendGitignoreRules` reads `.gitignore` (or treats as empty if missing).
4. For each rule in `GitignoreRules`, checks presence via `strings.Contains`.
5. If all rules present → returns nil (silent).
6. If any missing → assembles new content, writes via `iokit.WriteFileCtx`, prints confirmation to `out`.
7. `runInitExisting` proceeds to `os.RemoveAll(scaffoldPromptsDir)` and "Next steps:" output.

---

## 4. API / Interface Contracts

### New function: `scaffold.AppendGitignoreRules`

```go
// AppendGitignoreRules adds qode-specific patterns to .gitignore in root.
// Each rule is checked individually; only missing rules are appended.
// Prints a confirmation to out when rules are written; silent on no-op.
func AppendGitignoreRules(ctx context.Context, out io.Writer, root string) error
```

**Inputs:**
- `ctx context.Context` — forwarded to `iokit.WriteFileCtx`; cancellation respected before write.
- `out io.Writer` — receives confirmation message when rules are appended; silent on no-op.
- `root string` — filesystem path to the project root; `.gitignore` resolved as `filepath.Join(root, ".gitignore")`.

**Output:**
- `nil` on success or no-op.
- Wrapped error on read failure (`"reading .gitignore: %w"`) or write failure (`"updating .gitignore: %w"`).

**Exported constants/vars:**

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

**Confirmation output (written to `out`):**
```
Appended qode ignore rules to .gitignore
```

**Error responses:** propagated as Go errors; no status codes (CLI context).

---

## 5. Data Model Changes

No database, no structured data store. The only data artifact is the `.gitignore` text file.

**Schema of the appended block:**
```
\n                           ← only if existing content non-empty and lacks trailing newline
# qode temp files\n         ← only if marker not already present
.qode/branches/*/.*.md\n    ← only if rule not already present
.qode/branches/*/context/ticket.md\n
.qode/branches/*/refined-analysis-*-score-*.md\n
.qode/branches/*/diff.md\n
.qode/prompts/scaffold/\n
```

**Backward compatibility:** purely additive; no existing lines are removed or modified. User customizations within the `# qode temp files` block are preserved.

**Migration:** none — this is a first-time write, not a schema migration.

---

## 6. Implementation Tasks

- [ ] Task 1: (`scaffold`) Create `internal/scaffold/gitignore.go` with `GitignoreMarker`, `GitignoreRules`, and `AppendGitignoreRules` implementation.
- [ ] Task 2: (`scaffold`) Create `internal/scaffold/gitignore_test.go` with table-driven unit tests covering all cases (no file, no marker, all present, partial, no trailing newline).
- [ ] Task 3: (`cli`) Modify `internal/cli/init.go` — add `"context"` import, call `scaffold.AppendGitignoreRules(context.Background(), out, root)` after `scaffold.Setup` succeeds and before `os.RemoveAll`.
- [ ] Task 4: (`cli`) Add `TestRunInitExisting_AppendsGitignoreRules` and `TestRunInitExisting_GitignoreIsIdempotent` to `internal/cli/init_test.go`.

**Implementation order:** Task 1 → Task 2 → Task 3 → Task 4. Each task is independently committable with passing tests.

---

## 7. Testing Strategy

### Unit tests (`internal/scaffold/gitignore_test.go`)

Table-driven, `t.Parallel()` on parent and subtests, `t.TempDir()` for filesystem isolation.

| Test case | Setup | Assertions |
|---|---|---|
| No `.gitignore` exists | Empty `t.TempDir()` | File created; all 5 rules present; marker present; output contains "Appended" |
| `.gitignore` exists, no marker | File with unrelated content | All 5 rules appended; marker present; output contains "Appended" |
| All rules already present | File with marker + all 5 rules | File content unchanged; output empty (no-op) |
| Marker present, 2 rules missing | File with marker + 3 rules | Only 2 missing rules appended; no rule duplicated (`strings.Count == 1`); output contains "Appended" |
| No trailing newline in existing file | File content `"foo"` (no `\n`) | Block starts on new line (block does not begin with `foo...`); rules present |
| Context cancelled | Use cancelled context | `iokit.WriteFileCtx` returns error; function returns wrapped error |

Sentinel assertions: `strings.Contains(content, rule)` for each rule individually.
Idempotency assertion: `strings.Count(content, rule) == 1` after double-append case.

### Integration tests (`internal/cli/init_test.go`)

- `TestRunInitExisting_AppendsGitignoreRules`: call `runInitExisting` on a fresh `t.TempDir()`; read resulting `.gitignore`; assert each of the 5 rules is present.
- `TestRunInitExisting_GitignoreIsIdempotent`: call `runInitExisting` twice on the same dir; read `.gitignore`; assert `strings.Count(content, rule) == 1` for each rule.

### Regression

All existing tests in `internal/cli/init_test.go` and `internal/scaffold/` must continue to pass without modification. In particular:
- `TestRunInitExisting_NoDetectionOutput` must not flag "Appended qode ignore rules to .gitignore" (it does not contain forbidden strings "Detected", "Scanning", or "qode ide setup").

### E2E / edge cases (explicitly covered)

- `.gitignore` is read-only → error propagated.
- `.gitignore` is a directory → `os.ReadFile` error propagated.
- Running `qode init` outside a git repo → no special handling; function completes normally.
- Windows line endings in existing file → `strings.Contains` substring match works correctly; appended rules use `\n`.

---

## 8. Security Considerations

**Input validation:** No user input is interpolated into `.gitignore` content. `GitignoreRules` and `GitignoreMarker` are hardcoded constants. No injection risk.

**Authentication / authorisation:** None — this is a local filesystem operation within the project root. No network access, no elevated privileges required.

**Data sensitivity:** `.gitignore` is a non-sensitive plain text file. The appended patterns are glob patterns for qode's own ephemeral artifacts; they contain no secrets or PII.

**File permissions:** Written with mode `0644` (owner read/write, group/other read) — standard for source-controlled text files.

---

## 9. Open Questions

None. All ambiguities from the original ticket and notes have been resolved in the refined analysis:

- **Rule list**: `.qode/branches/*/diff.md` and `.qode/prompts/scaffold/` are included (notes are authoritative over ticket).
- **Output destination**: confirmation written to `out` (not stderr), consistent with all other `runInitExisting` output.
- **Idempotency granularity**: per-rule, not block-marker-only.
- **Rule location**: exported vars in `internal/scaffold/gitignore.go`.
- **Context**: `context.Background()` passed from `runInitExisting`; `AppendGitignoreRules` signature accepts `context.Context`.
- **Non-git repo**: no special handling required.

---

*Spec generated by qode. Copy to Jira/Azure DevOps ticket for team review.*
