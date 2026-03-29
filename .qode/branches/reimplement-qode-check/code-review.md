# Code Review — reimplement-qode-check (round 2)

## Pre-read incident report

This code shipped. `qode check` removal breaks direct CLI callers and CI pipelines. Accepted by design — the ticket is explicit, and every documentation surface has been updated in the same PR. Risk bounded to direct CLI callers in a workflow the project treats as IDE-first.

Now reading the diff to find anything new introduced by the round-1 fixes.

---

## Round-1 fixes verified

### `internal/ide/claudecode.go` — `qodeCheckBody` constant

**What the fix does:** Extracts the 52-line prompt body into a package-level `const qodeCheckBody`. The Claude Code entry becomes `fmt.Sprintf("# Quality Gates — %s\n\n", name) + qodeCheckBody`; the Cursor entry becomes `fmt.Sprintf("---\ndescription: Run quality gates for %s\n---\n\n", cfg.Project.Name) + qodeCheckBody`.

**Verified correct:**
1. Constant is accessible from `cursor.go` — both files are `package ide`. Confirmed: `grep -c "qodeCheckBody"` returns 3 (definition + 1 use) in `claudecode.go` and 1 (use) in `cursor.go`.
2. Output is structurally identical to the prior inline strings — the header/frontmatter produces the same prefix, then `qodeCheckBody` provides the identical body. The raw string starts immediately with `Run quality gates...` matching the prior inline string at that position.
3. Constant does not contain `%s` placeholders — no risk of `fmt.Sprintf` being accidentally omitted. The project name is injected only in the per-generator header, which is correct: the body contains no project-specific text.
4. Doc comment explains the design constraint (only header/frontmatter differs). ✅

**Verified no regressions:** `go test ./internal/ide/...` passes with 0 failures.

### `internal/ide/ide_test.go` — new content tests

**What the fix does:** Adds `TestClaudeSlashCommands_IncludesQodeCheck` and `TestCursorSlashCommands_IncludesQodeCheck`.

**Verified correct:**
1. Both tests verify key presence, Phase 1/2 headings, and absence of `test.unit`/`test.lint` field references. The prohibited-field check is the right proxy for the design constraint (AI must not use qode.yaml config fields) — it is more precise than checking for the string `"qode.yaml"` which would have falsely fired on the explicit "Do NOT read qode.yaml" instruction in the prompt.
2. Cursor test additionally verifies `description:` frontmatter is present. ✅
3. Tests run and pass (`-run QodeCheck -v` output confirms both PASS).
4. Test naming (`IncludesQodeCheck`) is consistent with the `IncludesKnowledge` convention already in the file. ✅

**Remaining minor gap (not a defect):** Neither test verifies that `cfg.Project.Name` appears in the generated content (i.e., the `%s` in the header is substituted correctly). This is the same gap that exists for `TestClaudeSlashCommands_IncludesKnowledge`. Since the substitution is a single `fmt.Sprintf` call and the integration tests (`TestSetupClaudeCode_WritesTicketFetchCommand`) already verify file-write behaviour, the risk of this gap is negligible. Not flagging as a new issue.

### `CONTRIBUTING.md`

**Verified correct:** "full 9-step workflow" → "full workflow". All three `qode check` references replaced. Pre-existing typo `/ qode-review-code` (extra space) fixed. Dangling backtick removed. ✅

### `CLAUDE.md`

**Verified correct:** Trailing newline added. File now ends with `\n`. ✅

### `internal/cli/help.go`

**Verified correct:** Stray `│       │` artifact removed from STEP 5 line. ✅

---

## New issues introduced by the round-1 fixes

None found.

The constant extraction, test additions, and documentation fixes are clean. No new error paths, no new assumptions, no regressions.

---

## Issues summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 0 |
| Medium   | 0 |
| Low      | 0 |
| Nit      | 0 |

---

## Top 3 before merging

All previously identified issues are resolved. The branch is ready to merge.

1. Run `qode ide setup` one final time before opening the PR to ensure the generated files (`.claude/commands/qode-check.md`, `.cursor/commands/qode-check.mdc`) reflect the final `qodeCheckBody` constant. The constant content is identical to what was previously inline, so the generated files are already correct — but a final `qode ide setup` confirms it.
2. Confirm the PR description notes that `qode check` is removed without a deprecation path, so reviewers are aware this is intentional.
3. Move #45 from "In progress" to "Done" in ROADMAP.md once the PR is merged.

---

## Rating

| Dimension      | Score (0-2) | What I verified (not what I assumed) |
|----------------|-------------|---------------------------------------|
| Correctness    | 2           | No dangling references in Go (`grep` empty); `qodeCheckBody` produces structurally identical output to prior inline strings; workflow ordering (check at step 6, before review at step 7) consistent across root.go, help.go, cursor workflow rule, README, CONTRIBUTING |
| Code Quality   | 2           | Duplication eliminated via documented package-level constant; no `%s` placeholders in body (safe from missing `fmt.Sprintf`); all doc surfaces updated; no stale references remain |
| Architecture   | 2           | Package-level constant accessible to both generators without cross-package dependency; follows established inline-content pattern; no new abstractions beyond what the refactor required |
| Error Handling | 2           | No new error paths introduced; deleted code removed completely; constant concatenation cannot fail at runtime |
| Testing        | 1.5         | New tests verify key exists, prohibited field references absent, structure present, Cursor frontmatter correct; no integration-level test for the generated `qode-check.md`/`.mdc` files (the existing `TestSetupClaudeCode_WritesTicketFetchCommand` pattern is not replicated for `qode-check`) — acceptable given unit-level coverage |

**Total Score: 9.5/10**

No Critical, High, Medium, or Low issues. The 0.5 deduction reflects the absence of a file-write integration test for the `qode-check` command (the same gap that exists for all other commands except ticket-fetch). This is consistent with the existing test convention and not a regression introduced by this PR.
