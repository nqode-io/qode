# Requirements Refinement Analysis — bug-plan-refine-not-reentrant

## 1. Problem Understanding

### Restatement

The `/qode-plan-refine` slash command (defined in `internal/ide/claudecode.go`) orchestrates the refinement workflow by:
1. Instructing the AI to run `qode plan refine --prompt-only`
2. Reading and executing `.qode/branches/$BRANCH/.refine-prompt.md`
3. Writing analysis output to `.qode/branches/$BRANCH/refined-analysis.md`

On first invocation, this works correctly. On second invocation, `qode plan refine --prompt-only` always generates an "Iteration 1" prompt instead of "Iteration 2", causing the AI to re-do the same iteration rather than building on the previous one.

### User Need and Business Value

The refinement workflow is iterative by design — users are expected to run multiple rounds until they reach 25/25. If subsequent rounds can only be triggered via terminal (`qode plan refine`), the IDE slash command experience is broken. This forces context-switching out of the editor and disrupts the intended in-IDE workflow.

### Ambiguities / Open Questions

- **None blocking.** The root cause is clearly identified in the code. No ambiguity about the expected behavior: each call to `/qode-plan-refine` must produce the correct next iteration number.

---

## 2. Technical Analysis

### Root Cause

**File:** `internal/context/context.go`, `Load()` function (lines 77–94)

Iteration tracking uses a glob: `refined-analysis-*.md`. This only matches *numbered* iteration files (e.g., `refined-analysis-1-score-22.md`).

**File:** `internal/plan/refine.go`, `BuildRefinePromptWithOutput()` (lines 30–33)

```go
if iteration == 0 {
    iteration = len(ctx.Iterations) + 1
}
```

The iteration number is derived from `len(ctx.Iterations)`. If no numbered files exist, iteration is always `1`.

**The gap:** The slash command tells Claude to write directly to `refined-analysis.md`. This file contains an HTML comment header with the iteration number:

```
<!-- qode:iteration=1 score=22/25 -->
```

...but `context.Load()` never reads this header back. The numbered file (`refined-analysis-1-score-22.md`) is only created when `ParseIterationFromOutput()` is called — which only happens in the non-`--prompt-only` flow (i.e., terminal `qode plan refine` with AI dispatch).

**Result:** After a slash command cycle that writes to `refined-analysis.md` without going through judge scoring, the glob finds zero numbered files → iteration resets to 1.

### Affected Components

| Component | File | Role |
|---|---|---|
| Iteration tracking | `internal/context/context.go` | `Load()` parses iteration history from glob |
| Iteration calculation | `internal/plan/refine.go` | `BuildRefinePromptWithOutput()` calculates next iteration |
| Iteration persistence | `internal/plan/refine.go` | `ParseIterationFromOutput()` creates numbered files |
| Slash command definition | `internal/ide/claudecode.go` | `claudeSlashCommands()` generates command text |
| Canonical analysis file | `.qode/branches/$BRANCH/refined-analysis.md` | Written by Claude, has iteration header |

### Key Technical Decisions

1. **Fix in `context.Load()`:** Parse the iteration number from `refined-analysis.md`'s HTML comment header when the file exists. This is the minimal-impact fix — no schema changes, no new files.

   Specifically: read `refined-analysis.md`, extract `<!-- qode:iteration=N score=S/M -->` via regex, and use `N` to compute the next iteration if it is higher than what the glob found.

2. **Do NOT change the numbered file creation logic.** The `refined-analysis-N-score-S.md` files still serve as the immutable audit history. The fix should be additive: `refined-analysis.md` becomes an additional source of iteration truth for the case where numbered files lag behind.

### Patterns to Follow

- `internal/context/context.go` already uses `strings.Split` and `strconv.Atoi` for parsing. Use the same approach for the HTML comment.
- No new dependencies. The HTML comment format is simple enough for `strings.Contains` / `fmt.Sscanf` or a minimal regex.
- Adhere to the 50-line function limit — extract a helper `parseIterationFromAnalysis(path string) (int, int, bool)` if needed.

### Dependencies

None. This is fully self-contained within the `internal/context` and `internal/plan` packages.

---

## 3. Risk & Edge Cases

| Case | Risk | Handling |
|---|---|---|
| `refined-analysis.md` exists but has no HTML comment header (manually written or from old version) | Iteration would fall back to glob count | Safe — treat missing header as iteration 0; glob count still applies |
| `refined-analysis.md` header says iteration=3 but glob finds 2 numbered files | Discrepancy. Numbered files may be from a different session | Use `max(glob_count, header_iteration)` as the canonical count |
| `refined-analysis.md` exists from branch with no prior numbered files | Header says iteration=1 → next is 2 | Correct |
| Corrupted header (e.g., bad parse) | Parse returns 0 | Fall back to glob count; no regression |
| Concurrent writes | Slash command runs in-IDE AI context; only one session writes at a time | Not a concern for this use case |
| `refined-analysis.md` score header has 0 score (unscored) | Score=0 is valid for first draft | Fine — iteration number is what matters, not score |

### Security

No user input is parsed from external sources. File content is project-local. No security concern.

### Performance

Single additional file read in `context.Load()`. Negligible.

---

## 4. Completeness Check

### Acceptance Criteria

1. Calling `/qode-plan-refine` twice in sequence generates "Iteration 1" then "Iteration 2" (not "Iteration 1" again).
2. The behavior is correct regardless of whether `qode plan refine` (judge scoring) was run between slash command calls.
3. Calling the slash command after terminal `qode plan refine` judge scoring still works correctly (no regression).
4. The fix does not break the numbered file audit trail (`refined-analysis-N-score-S.md`).
5. `qode check` passes.

### Implicit Requirements

- The fix must be backward-compatible: branches that have numbered files but no `refined-analysis.md` header must still work correctly.
- Old `refined-analysis.md` files without the HTML comment header must not cause a crash or incorrect iteration.

### Out of Scope

- Changing the slash command text itself (it already correctly instructs the workflow).
- Adding judge scoring to the slash command flow.
- Changing `ParseIterationFromOutput()` or numbered file naming.

---

## 5. Actionable Implementation Plan

### Task 1 — Add `parseIterationFromAnalysis()` helper in `internal/context/context.go`

Write a function that reads `refined-analysis.md` from the branch directory, finds the line `<!-- qode:iteration=N score=S/M -->`, and returns `(iteration int, score int, ok bool)`.

Implementation sketch:
```go
func parseIterationFromAnalysis(branchDir string) (int, int, bool) {
    data, err := os.ReadFile(filepath.Join(branchDir, "refined-analysis.md"))
    if err != nil {
        return 0, 0, false
    }
    var n, score, max int
    for _, line := range strings.SplitN(string(data), "\n", 5) {
        if _, err := fmt.Sscanf(line, "<!-- qode:iteration=%d score=%d/%d -->", &n, &score, &max); err == nil {
            return n, score, true
        }
    }
    return 0, 0, false
}
```

### Task 2 — Update `context.Load()` to use the header as a fallback

After the existing glob loop in `Load()`, add:

```go
if n, score, ok := parseIterationFromAnalysis(dir); ok {
    // If the canonical file represents a newer iteration than the glob found,
    // use it to backfill the iteration list.
    existing := maxIterationNumber(ctx.Iterations) // helper
    if n > existing {
        ctx.Iterations = append(ctx.Iterations, Iteration{Number: n, Score: score})
    }
}
```

Or simpler — just ensure `len(ctx.Iterations)` reflects the correct count without duplicates.

### Task 3 — Add `maxIterationNumber()` helper (if needed)

Tiny helper to find the maximum `Number` in `ctx.Iterations`. May not be needed if we use a simpler approach (e.g., check if any existing iteration has `Number == n` before appending).

### Task 4 — Write unit tests

File: `internal/context/context_test.go`

Test cases:
- Directory with only `refined-analysis.md` (no numbered files) → `ctx.Iterations` has length 1 with correct iteration/score.
- Directory with numbered files AND `refined-analysis.md` header matching highest numbered file → no duplicate.
- Directory with numbered files AND `refined-analysis.md` with higher iteration (unscored cycle) → header wins.
- `refined-analysis.md` with no HTML header → graceful fallback.
- Missing `refined-analysis.md` → graceful fallback.

### Task 5 — Run `qode check`

Verify quality gates pass. No TODOs, no magic numbers.

### Implementation Order

1 → 2 → 3 → 4 → 5

All tasks are in one commit. The fix is self-contained in `internal/context/context.go`.
