# Security Review — qode (optimize-prompts)

## Working Assumptions

**What this code trusts:**
- `ctx.ContextDir` (and thus `BranchDir`) is a safe filesystem path — confirmed: constructed via `filepath.Join(root, config.QodeDir, "branches", branch)` where `root` is `filepath.Abs(os.Getwd())` and `branch` comes from git.
- The `diff` string passed to `os.WriteFile` contains no executable content — it is `git diff` output treated as bytes, not interpreted.
- Template values substituted for `{{.BranchDir}}` are plain strings in rendered output — Go's `text/template` engine is single-pass; rendered output is not re-parsed.
- `WarnMissingPredecessors` format strings are hardcoded — no user input reaches `fmt.Fprintln` format arguments.

**Which assumptions are enforced vs. merely expected:**
- Enforced: `filepath.Join` / `filepath.Clean` resolves `..` traversals in any path component.
- Enforced: Go's `text/template` does not re-execute rendered output — language guarantee.
- Enforced: `git check-ref-format` rejects branch names containing `..` (consecutive dots disallowed).
- Merely expected: Developer does not accidentally stage secrets before running `qode review code` — the new `diff.md` snapshot persists those secrets to disk.

**Unverified trust:** The `diff` content written to `diff.md` is written with `0644` permissions — world-readable. If the diff contains accidentally-staged credentials, they persist in a new file at `.qode/branches/<branch>/diff.md`.

---

## Adversary Simulation

**Attempt 1: Template injection via malicious branch name**
- **Attempt:** Create a branch named `{{printf "%s" .Project.Name}}` to inject Go template directives into the rendered `{{.BranchDir}}` substitution
- **Target:** All five `BuildXxxPrompt` functions where `BranchDir: ctx.ContextDir` is set and expanded in templates
- **Result:** Would NOT succeed. Go's `text/template` engine is single-pass — `{{.BranchDir}}` is replaced by its string value; the double-brace characters in the resulting path string are plain text, not re-parsed by the engine. Verified by reading `internal/prompt/engine.go` — `Render` calls `t.Execute` once with no second parse cycle.

**Attempt 2: Path traversal via branch name to write `diff.md` outside the project**
- **Attempt:** Create a branch named `../../etc/cron.d/evil` and run `qode review code` to write `diff.md` to an arbitrary path via `os.WriteFile(diffPath, ...)`
- **Target:** `internal/cli/review.go` — `diffPath := filepath.Join(branchDir, "diff.md")`
- **Result:** Would NOT succeed. Two independent controls block this: (1) `filepath.Join` calls `filepath.Clean` which resolves `..` sequences — `filepath.Join("/proj/.qode/branches/../../etc/cron.d/evil", "diff.md")` resolves within the project root, not to `/etc`; (2) git's `check-ref-format` rejects branch names containing `..`, so such a branch cannot be created in the first place.

**Attempt 3: Read staged credentials from persistent `diff.md`**
- **Attempt:** As a second local user on a shared machine, read `.qode/branches/<branch>/diff.md` after a developer runs `qode review code`, extracting accidentally-staged API keys or tokens from the diff snapshot
- **Target:** `os.WriteFile(diffPath, []byte(diff), 0644)` — mode `0644` (world-readable)
- **Result:** Would succeed on a shared machine if the diff contains accidentally-staged secrets. `0644` permits read by all local users. Controls that partially mitigate: the file is in the project directory (not `/tmp`), so the attacker needs filesystem access to the project; the diff is also accessible via `git diff` which has the same permissions. This is a new persistent exposure not present before this branch (previously the diff was held in memory and passed inline to the prompt builder).

---

## Security Checklist

### Injection
- **Template injection:** Not present. `{{.BranchDir}}` and all new template references expand to plain strings. Go's `text/template` engine renders output once; no double-evaluation. Zero new `{{` directives added to template source. Verified by reading both `review/code.md.tmpl` and `review/security.md.tmpl` diffs — only `{{.BranchDir}}` substitutions added, no new executable directives.
- **Command injection:** Not present. `WarnMissingPredecessors` writes hardcoded strings to an `io.Writer`. `os.WriteFile` writes bytes to disk — no shell execution. No new `exec.Command` calls in this diff.
- **Path traversal:** Not present as an exploitable risk. `filepath.Join` + `filepath.Clean` resolves traversal attempts; git validation rejects `..` in branch names.

### Authentication & Authorisation
- No authentication or authorisation logic modified. qode is a local CLI tool; no HTTP endpoints, no privilege boundaries crossed. `WarnMissingPredecessors` writes to the caller's `io.Writer` — no escalation path.

### Data Exposure
- **New disk artifact:** `os.WriteFile(filepath.Join(branchDir, "diff.md"), []byte(diff), 0644)` persists the full `git diff` output to disk with world-readable permissions. This is a new exposure — previously the diff was in-memory only. On single-user machines, no impact. On shared machines, another local user can read the diff including any accidentally-staged secrets.
- **`WarnMissingPredecessors` stderr:** Discloses whether `spec.md` and `refined-analysis.md` exist. This is intentional (user-facing warning), not a data exposure concern. No sensitive content included.
- **Removed `Notes` field:** `context.Notes` is no longer read from disk or inlined in prompts. This is a reduction in data sent to AI — neutral to positive impact.

### Input Validation
- `BranchDir` is derived from git-validated branch name passed through `filepath.Join`. No direct user input.
- `diff` content comes from the git library (`git diff` output). Written to disk as opaque bytes — not parsed, not executed.
- `WarnMissingPredecessors` step parameter comes from hardcoded call sites (`"spec"`, `"start"`, `"review"`). Unknown values fall through the switch silently — correct, safe behavior.

### Cryptography
- Not applicable. No cryptographic operations added or modified.

### Frontend-Specific
- Not applicable. Go CLI tool.

### API Security
- Not applicable. No HTTP endpoints introduced or modified.

### Dependency Issues
- Only `"io"` (stdlib) added as a new import in `context.go`. `go.mod` and `go.sum` are unchanged. Zero new external dependencies.

---

## Summary

**Vulnerabilities by severity:**
- Critical: 0
- High: 0
- Medium: 0
- Low: 1
- Informational: 1

**Low — `diff.md` written world-readable (`0644`)**
- **OWASP Category:** A02:2021 – Cryptographic Failures (sensitive data protection at rest)
- **File:** `internal/cli/review.go` — `os.WriteFile(diffPath, []byte(diff), 0644)`
- **Vulnerability:** The git diff snapshot is written with mode `0644`, making it readable by all local users. This is a new persistent artifact: previously the diff was held in memory and never written to disk by this code path. If the diff contains accidentally-staged credentials, they persist in `.qode/branches/<branch>/diff.md` until the file is deleted or the branch is cleaned up.
- **Exploit Scenario:** On a shared developer machine, a second local user reads `.qode/branches/<branch>/diff.md` after `qode review code` is run, extracting staged API keys or tokens.
- **Remediation:** Use `os.WriteFile(diffPath, []byte(diff), 0600)` (owner-read/write only) instead of `0644`. This matches the principle of least privilege for a file containing developer work product.

**Informational — `diff.md` may be accidentally committed**
- **File:** `internal/cli/review.go`
- **Note:** The new `diff.md` file is written to `.qode/branches/<branch>/`, the same directory where `code-review.md` and `spec.md` are committed as part of the workflow. Developers may accidentally `git add .qode/` and commit `diff.md`. At review time the diff represents uncommitted work; a committed `diff.md` would be stale and potentially misleading. Consider adding `diff.md` to `.gitignore` or documenting that it should not be committed.

**Must-fix before merge:** No Critical or High findings. The Low finding (file permission) is a one-character fix.

**Overall security posture:** The changeset introduces no new attack surface for injection, auth bypass, or privilege escalation. The primary new risk is a data-at-rest concern: a world-readable file containing code changes that may include accidentally-staged secrets. All other new code (template references, `WarnMissingPredecessors`, field removals) is security-neutral or a net improvement.

---

## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score |
|-----------------------|-------------|------------------------------------------------|
| Injection Prevention  | 2           | Template injection: Go `text/template` is single-pass — verified by reading engine.go `Render` method; no re-parse cycle. Path traversal: `filepath.Join`/`filepath.Clean` resolves `..`; git `check-ref-format` blocks `..` branch names. Command injection: no new `exec.Command`; `WarnMissingPredecessors` uses hardcoded format strings. |
| Auth & Access Control | 2           | No auth logic modified; no HTTP endpoints; no new privilege boundaries. CLI tool executes as current user throughout. |
| Data Protection       | 1           | New `diff.md` artifact written with `0644` (world-readable) — finding above. All other data handling is unchanged or improved (Notes field removed reduces data inlined into prompts). |
| Input Validation      | 2           | `BranchDir` derived from git-validated branch name through `filepath.Join`; `diff` from git library treated as opaque bytes; `WarnMissingPredecessors` step arg is always a hardcoded string at call sites. |
| Dependency Security   | 2           | Only `io` stdlib import added. `go.mod` and `go.sum` unchanged. Zero new external dependencies. |

**Total Score: 9.0/10**

Constraints check: No Critical or High vulnerabilities — no score cap applies. Score ≥ 8.0 justified by: Injection Prevention verified by reading `engine.go` and confirming single-pass execution; Auth & Access Control verified by confirming no HTTP endpoints or auth code changes; Input Validation verified by reading all three new call sites. Deduction on Data Protection for `0644` file permissions on new `diff.md` artifact.
