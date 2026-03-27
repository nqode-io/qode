<!-- qode:iteration=2 score=25/25 -->

# Requirements Analysis — Remove Unused Commands (Iteration 2)

## 1. Problem Understanding

The qode CLI has accumulated six commands that are never invoked by any IDE slash command or prompt template: `qode plan status`, `qode branch list`, `qode branch focus`, `qode config show`, `qode config detect`, and `qode config validate`. These orphan commands add noise to the CLI surface area, making it harder for new contributors to understand which commands are part of the actual workflow.

Removing them simplifies the codebase, tightens the CLI to its documented workflow, and removes dead code that must otherwise be maintained.

**Business value:** Smaller public API, less maintenance burden, clearer onboarding.

**Open questions:** None — the ticket names every command to remove and every workflow command to keep.

---

## 2. Technical Analysis

### Affected files

| File | Change |
|---|---|
| `internal/cli/plan.go` | Delete `newPlanStatusCmd()` (lines 172–223); remove from `newPlanCmd()` line 21 |
| `internal/cli/branch.go` | Delete `newBranchListCmd()` (lines 95–134) and `newBranchFocusCmd()` (lines 136–176); update `newBranchCmd()` AddCommand call |
| `internal/cli/config_cmd.go` | Delete entire file |
| `internal/cli/root.go` | Remove `newConfigCmd()` from `rootCmd.AddCommand(...)` (line 71) |
| `internal/cli/help.go` | Remove `│  → Check status: qode plan status                               │` (line 40) |
| `README.md` | Remove rows for all six commands (lines 125–126, 135, 155–157) |
| `CLAUDE.md` | No changes needed (grep confirmed no references) |

### Secondary effects verified

1. **`internal/detect` package is safe.** `detect.Composite` is still called by `internal/cli/init.go:75`. Deleting `config_cmd.go` does not orphan the import.

2. **`context.Iterations` field is safe.** Still consumed by `plan.BuildRefinePromptWithOutput` in `internal/plan/refine.go` via `len(ctx.Iterations) + 1` for auto-iteration numbering.

3. **`safeBranchDir` helper is safe.** Still called by `newBranchCreateCmd` and `newBranchRemoveCmd`.

4. **`git.CheckoutBranch` is the only dead-code risk.** Its sole CLI caller is `newBranchFocusCmd`. Grep `internal/git/` before deleting; if uncalled, remove the function (and any private helpers it delegates to).

5. **`internal/context.Iterations` field in `plan status` loop** — the `ctx.Iterations` range in `newPlanStatusCmd` (lines 200–209) is the only place `Iteration.Score` is rendered as a formatted string. Removing this does not affect anything else.

### Patterns / conventions

- Remove factory functions entirely — don't leave empty stubs.
- Update `AddCommand` call-sites atomically with the function removal.
- No test files exist for the deleted commands (`config_cmd_test.go`, `plan_status_test.go` absent), so no test cleanup.

### Dependencies

None external. The `cli` package is consumed only by `main.go` via `Execute()`.

---

## 3. Risk & Edge Cases

| Risk | Likelihood | Mitigation |
|---|---|---|
| `git.CheckoutBranch` left as dead exported function | Low | Grep `internal/git/` for callers; delete if zero results |
| `detect` package orphaned | None | `init.go` still imports it |
| `context.Iterations` field orphaned | None | `plan/refine.go` still reads it |
| README table structure broken after row deletions | Low | Verify table header/footer rows unchanged |
| `go build ./...` fails due to missed import cleanup | Low | Run as final step |
| CLAUDE.md contains hidden reference | None | Grep confirms clean |
| Users scripting `qode config validate` in CI | Very low | Ticket explicitly classifies these as unused |

**Security:** None — all removed commands are read-only.

**Performance:** None.

---

## 4. Completeness Check

### Acceptance criteria

- [ ] `newPlanStatusCmd()` deleted from `internal/cli/plan.go`, removed from registration
- [ ] `newBranchListCmd()` deleted from `internal/cli/branch.go`, removed from registration
- [ ] `newBranchFocusCmd()` deleted from `internal/cli/branch.go`, removed from registration
- [ ] `internal/cli/config_cmd.go` deleted
- [ ] `newConfigCmd()` removed from `rootCmd.AddCommand(...)` in `root.go`
- [ ] `internal/cli/help.go` workflow diagram updated
- [ ] `README.md` command table updated (6 rows removed)
- [ ] `CLAUDE.md` checked — no change needed

### Implicit requirements

- Grep `internal/git/` for `CheckoutBranch`; delete if no callers remain after the branch.go edit.
- `go build ./...` and `go vet ./...` must pass with zero errors.
- Root `Long` description in `root.go` does not reference any removed command — confirmed, no change needed.

### Out of scope

- `qode plan refine`, `qode plan spec`, `qode branch create`, `qode branch remove`, `qode ticket`, `qode check`, `qode init`, `qode knowledge`, `qode workflow` — all unchanged.
- Prompt templates and slash command definitions — unchanged.
- Adding aliases or deprecation shims for removed commands.

---

## 5. Actionable Implementation Plan

### Task 1 — Remove `plan status` (`internal/cli/plan.go`)
- Delete lines 172–223 (`newPlanStatusCmd` body)
- Line 21: `cmd.AddCommand(newPlanRefineCmd(), newPlanSpecCmd(), newPlanStatusCmd())` → `cmd.AddCommand(newPlanRefineCmd(), newPlanSpecCmd())`

### Task 2 — Remove `branch list` and `branch focus` (`internal/cli/branch.go`)
- Delete lines 95–134 (`newBranchListCmd`)
- Delete lines 136–176 (`newBranchFocusCmd`)
- Update `newBranchCmd()` `AddCommand` call (remove both references)
- Grep `internal/git/` for `CheckoutBranch`; delete function if zero callers

### Task 3 — Remove `config` command group
- Delete `internal/cli/config_cmd.go`
- `internal/cli/root.go` line 71: remove `newConfigCmd()` from `AddCommand` list

### Task 4 — Update help diagram (`internal/cli/help.go`)
- Remove line 40: `│  → Check status: qode plan status                               │`

### Task 5 — Update `README.md`
- Remove 6 rows: `qode branch list`, `qode branch focus`, `qode plan status`, `qode config show`, `qode config detect`, `qode config validate`

### Task 6 — Verify build
- `go build ./...` — must succeed
- `go vet ./...` — must succeed

### Order
Tasks 1–5 are independent and can be done in any order. Task 6 runs last.
