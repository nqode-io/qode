# Security Review — strict-mode branch

**Reviewer:** Claude Sonnet 4.6
**Date:** 2026-03-30
**Diff:** `.qode/branches/strict-mode/diff.md`

---

## Working Assumptions

**What this code trusts:**
- The local git installation and its branch naming validation (branch names come from `git rev-parse --abbrev-ref HEAD`)
- The developer's `qode.yaml` configuration file — rubric dimension names, weights, and score levels flow into prompt templates unescaped
- The developer's `.qode/prompts/` directory — any `.md.tmpl` file placed there is executed as a full Go template by `text/template`
- AI judge output text — fed into regex parsers in `scoring/extract.go` and `scoring/scoring.go`

**Enforcement vs. expectation:**
- Branch name safety: enforced by git ref naming rules (no `..`, no control chars) — not by the application
- `qode.yaml` content safety: expected, not enforced — the developer is the sole author
- AI output content: expected to conform to the scoring format — not validated beyond regex matching

---

## Adversary Simulation

1. **Attempt:** Craft a `qode.yaml` rubric dimension with name `{{.SomeField}}` or Go template syntax to execute arbitrary template logic during `qode plan judge`. **Target:** `prompt.Engine.Render` via `TemplateData.Rubric.Dimensions[i].Name`. **Result:** Blocked. `text/template` renders `{{.Name}}` as a plain string — data values are not re-evaluated as template source. The output would contain the literal string `{{.SomeField}}`, not an execution result.

2. **Attempt:** Create a branch named `../../attack` to write `diff.md` outside `.qode/` via path traversal in `review.go:88-91`. **Target:** `filepath.Join(branchDir, "diff.md")` where `branchDir` embeds the branch name. **Result:** Blocked by git ref naming rules. Git forbids `..` in branch names (enforced at `git checkout -b` time). `git rev-parse --abbrev-ref HEAD` returns the sanitized stored name. Even if bypassed, `filepath.Join` with `..` in the branch component can traverse at most to the project root directory.

3. **Attempt:** Place a recursive template in `.qode/prompts/refine/base.md.tmpl` that calls itself to trigger a stack overflow or infinite loop when `qode plan refine` is run. **Target:** `Engine.Render("refine/base", data)` at `engine.go:76-84`. **Result:** Partially effective as a local DoS. `template.New(name).Parse(content)` accepts recursive template definitions and `t.Execute` will stack-overflow or spin. Requires write access to the repository's `.qode/prompts/` directory. In a team setting, any developer with write access can trigger this against other developers. Not exploitable remotely.

---

## Vulnerabilities

### Low

**S1 — `.qode/prompts/` custom templates execute without a recursion guard**
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** `internal/prompt/engine.go:70-85`
- **Vulnerability:** `Engine.Render` loads any template from `.qode/prompts/<name>.md.tmpl` and executes it with `text/template`. Go's `text/template` has no depth limit for recursive template calls. A template containing `{{template "itself" .}}` will recurse until the goroutine stack is exhausted. Since `.qode/prompts/` is a repository-level directory, any team member with push access can place such a template.
- **Exploit Scenario:** Developer A commits a template to `.qode/prompts/start/base.md.tmpl` with a `{{define "start/base"}}...{{template "start/base" .}}{{end}}` recursive definition. Every team member running `qode start` crashes.
- **Remediation:** Add a configurable execution timeout to `Engine.Render`:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  // execute template inside a goroutine, signal via channel, check ctx.Done()
  ```
  Alternatively, document that `.qode/prompts/` overrides are trusted code and gate template changes through code review — if the project already does this, add a comment to `engine.go` stating the trust model.

---

### Informational

**I1 — Branch name used in `filepath.Join` without application-level sanitization**
- **File:** `internal/cli/review.go:86`, `internal/cli/plan.go:162`, `internal/cli/start.go:83`
- **Observation:** `branchDir := filepath.Join(root, config.QodeDir, "branches", branch)` where `branch` is the raw output of `git rev-parse --abbrev-ref HEAD`. The application relies on git's ref naming validation (which forbids `..`, `/` at start/end, and other problematic sequences) rather than validating the branch name itself. If the git binary were compromised or a future code path accepted branch names from a source other than `git rev-parse`, path traversal into the project root directory would be possible.
- **Note:** Not exploitable with standard git. No remediation required; the trust delegation to git is appropriate for a local developer tool. Add a comment if this expands to accept branch names from user-supplied arguments.

**I2 — `text/template` used instead of `html/template` for all prompt rendering**
- **File:** `internal/prompt/engine.go:76`
- **Observation:** `template.New(name).Funcs(e.funcMap).Parse(tmplContent)` uses `text/template`. This means HTML special characters in `qode.yaml` values (rubric names, levels) and context files (ticket.md, spec.md) are NOT escaped when rendered into prompts. This is intentional — prompts are Markdown text fed to an AI, not HTML rendered in a browser. No remediation needed. Noted for completeness.

**I3 — Git diff content written to `diff.md` before content inspection**
- **File:** `internal/cli/review.go:88-91`
- **Observation:** `os.WriteFile(diffPath, []byte(diff), 0644)` persists the full working-tree diff to a file with world-readable permissions (0644). If the diff contains accidentally staged secrets (API keys, tokens), they are durably written to `diff.md` in the repository. This is pre-existing behavior (not introduced in this diff) and is inherent to any tool that captures diffs. File mode 0644 is appropriate for a local developer tool; no change needed.

---

## Verified safe

- **`exec.Command` usage in `git/git.go:102-113`:** Arguments passed as a slice (`exec.Command("git", args...)`), not via a shell. Branch names, SHAs, and paths cannot inject shell metacharacters. Verified all call sites in `git.go` use this pattern.
- **`scoring/extract.go:9`:** Updated regex `(?i)total\s*score[:\s*]*(\d+(?:\.\d+)?)\s*/\s*(\d+)` — the `\d+`, `\s*`, and `[:\s*]*` quantifiers have linear backtracking. No nested quantifiers on overlapping character classes. Not vulnerable to ReDoS even on adversarially crafted input.
- **`scoring/scoring.go:55-58`:** `totalRe.FindStringSubmatch(judgeOutput)` applied to AI-generated text. Linear regex, bounded output. The `dimRe` pattern at line 62 is similarly linear.
- **`prompt/engine.go:40-43`:** New funcmap functions `add` and `pct` are pure arithmetic — no I/O, no subprocess, no filesystem access. Cannot be leveraged for injection or privilege escalation from within a template.
- **`internal/config/schema.go:96-99`:** New `ScoringConfig.Strict bool` field. Boolean flag, zero value `false`. No user-controlled string processing. No injection surface.
- **`internal/workflow/guard.go`:** Pure function with no I/O or external calls. No new attack surface. `CheckStep` operates only on in-memory `Context` and `Config` values.
- **YAML parsing (`qode.yaml` + `config.Load`):** Uses `gopkg.in/yaml.v3` which does not support arbitrary type instantiation (unlike Python's `yaml.load`). Rubric dimension strings are parsed as `string` and `int` scalars — no deserialization of executable code.
- **`context.go:132-151` (`HasCodeReview`, `HasSecurityReview`):** `os.Stat` calls on known paths under `.qode/branches/<branch>/`. No user-controlled path components beyond the branch name (already discussed in I1). Return value is a boolean — no file content is read.
- **No new network calls:** The entire diff introduces zero new HTTP/TLS/network operations. All new code paths are local filesystem and subprocess.
- **No new dependencies:** `go.mod` and `go.sum` are unchanged in this diff. No new transitive dependency vulnerabilities introduced.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 1 |
| Informational | 3 |

**Must-fix before merge:** None. No Critical or High vulnerabilities found.

**Overall security posture:** Good. The diff introduces no new external network calls, no new credential processing, no new shell execution paths, and no new user-input-to-filesystem flows beyond what existed pre-branch. The one Low finding (S1, recursive template DoS) requires write access to the repository and affects only local developer machines, not users or production systems. Documenting the trust model for `.qode/prompts/` is a sufficient mitigation.

---

## Rating

| Dimension | Score | Control or finding that determines this score |
|-----------|-------|------------------------------------------------|
| Command & Path Injection (0–3) | 3 | Verified `exec.Command("git", args...)` at `git.go:103` — slice args, no shell expansion. All file paths use `filepath.Join`. Branch name traversal blocked by git ref validation (not application-level). `kind` parameter in `runReview` is a hard-coded string at `review.go:102-107`, never user-supplied. |
| Credential Safety (0–3) | 3 | No new credential processing in any code path. `qode.yaml` rubric fields (names, weights, levels) are display strings only — no credential fields. `diff.md` write at `review.go:89-91` is pre-existing behavior; secrets in diffs are a developer responsibility, not an application vulnerability. No tokens written to stdout. |
| Template Injection (0–3) | 2 | Funcmap `add`/`pct` at `engine.go:40-43` verified safe (pure arithmetic, no I/O). Template data strings are not re-evaluated — verified by tracing `Rubric.Dimensions[i].Name` through `Render` with `text/template`. Deduction: `.qode/prompts/` custom templates can recurse without bound (S1) — no execution timeout or depth limit guards `Engine.Render`. |
| Input Validation & SSRF (0–2) | 2 | No new HTTP calls introduced. Regex patterns at `extract.go:9` and `scoring.go:55,62` verified non-catastrophic. YAML parsed as typed scalars — no code execution. Integer weight accumulation in `Rubric.Total()` theoretically overflowable but requires attacker-authored `qode.yaml`. |
| Dependency Safety (0–1) | 1 | `go.mod` and `go.sum` not modified. No new transitive dependencies. Verified by inspecting the complete diff. |

**Total Score: 11.0/12**
**Minimum passing score: 10/12**

Constraints:
- A Critical vulnerability voids a high score — total cannot exceed 6.0
- A High vulnerability caps the total at 9.0
- Total ≥ 9.6 requires citing specific controls observed (e.g. parameterized queries at line X, input allowlist at line Y) — not just the absence of known bugs ✓
- Total 12.0 is not a valid security score; complete security is not provable
