# Security Review — remove-unused-flags

**Branch:** remove-unused-flags
**Reviewer:** qode AI security review
**Date:** 2026-03-29

---

## Working Assumptions — What Does This Code Trust?

**Inputs arriving from outside the trust boundary:**
1. `args[0]` — branch name (user-supplied CLI argument; used for git operations and filesystem path)
2. `args[1]` — base branch/ref (new in this PR; user-supplied; passed directly to git)
3. `--keep-branch-context` flag value (boolean; benign)
4. `qode.yaml` contents (local file; partially trusted; could be modified by any user with write access)

**What this code assumes the caller has already validated:**
- That `args[0]` (branch name) is safe for filesystem use — enforced by `safeBranchDir`
- That `args[1]` (base ref) is a valid, safe git ref — **not enforced**
- That `qode.yaml` has not been tampered with — not enforced (no HMAC or signature)

**Enforced vs. merely expected:**
- `safeBranchDir` path traversal check on branch name: **enforced** (`branch.go:14–21`)
- `args[1]` contents: **not validated** before being passed to git subprocess
- No shell involved (`exec.Command` with variadic args): **enforced by design**

---

## Vulnerability Findings

---

**Severity:** Low
**OWASP Category:** A03:2021 – Injection (Argument Injection)
**File:** `internal/cli/branch.go:50–53`
**Vulnerability:** The new `base` positional argument is passed unsanitised to `git.CreateBranch`, which passes it as a distinct argument to `exec.Command("git", "checkout", "-b", name, base)`. Under normal CLI use, cobra's flag parser prevents leading-dash arguments from reaching `RunE` as positional args. However, the POSIX `--` separator disables cobra's flag parsing for subsequent tokens. A user running `qode branch create feat -- --orphan` results in `args = ["feat", "--orphan"]`, so `base = "--orphan"`. Git then receives `git checkout -b feat --orphan`, which creates an unparented (orphan) branch — not the intended behaviour.
**Exploit Scenario:** A crafted invocation `qode branch create feature -- --upload-pack=/path/to/binary` passes `--upload-pack=...` as a git flag. `git checkout` does not honour `--upload-pack`, so this specific variant is harmless. Other git flags that *are* honoured by `git checkout` (`--orphan`, `--no-track`, `--track`) could produce unexpected branch states. This requires deliberate user action; no remote exploit path exists in a developer CLI.
**Remediation:** Validate that `base` does not begin with `-` before passing to git:
```go
if len(base) > 0 && base[0] == '-' {
    return fmt.Errorf("invalid base branch %q: must not start with '-'", base)
}
```
This mirrors the principle used in `safeBranchDir` (rejecting malformed names before use).

---

**Severity:** Informational
**OWASP Category:** A04:2021 – Insecure Design (trust boundary note)
**File:** `internal/cli/branch.go:100`
**Vulnerability:** `branch remove` now calls `config.Load(root)` to read `cfg.Branch.KeepBranchContext`. In a shared environment where `qode.yaml` is committed to a repository, a malicious commit adding `branch: { keep_branch_context: true }` could cause `qode branch remove` to silently skip deleting context folders for all team members. No privilege escalation is possible; the consequence is data hoarding (context folders are not cleaned up), not data loss or code execution.
**Exploit Scenario:** Attacker has write access to the repo and merges a `qode.yaml` change setting `keep_branch_context: true`. Legitimate users running `qode branch remove` do not observe a change in exit code or stdout (branch is still deleted). Context folders accumulate. Impact: disk space, potential exposure of context folder contents to other team members if `.qode/branches/` is not in `.gitignore`.
**Remediation:** This is a design-level note, not a code defect. The context folder cleanup behaviour being config-driven is intentional. The risk is mitigated when `.qode/branches/` is added to `.gitignore` (tracked in a separate issue). No code change needed for this finding.

---

## Properties Verified Safe

**No shell injection via `exec.Command`** — `internal/git/git.go:102–113` uses `exec.Command("git", args...)` where args are passed as distinct elements (not concatenated into a shell string). Shell metacharacters in `name` or `base` have no special meaning. Verified by reading the `run()` function directly. ✅

**`safeBranchDir` path traversal control remains intact** — `internal/cli/branch.go:14–21` (unchanged from main):
```go
rel, err := filepath.Rel(base, target)
if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
    return "", fmt.Errorf("invalid branch name %q: path traversal detected", name)
}
```
This control was not touched by this PR. Branch name path traversal is blocked before any filesystem operation. ✅

**No new secrets, credentials, or sensitive data in code paths** — the new `config.Load` call reads local YAML. `BranchConfig.KeepBranchContext` is a boolean preference. No tokens, API keys, or user credentials flow through changed code. ✅

**Removal of `runInitNew`, `readLine`, `pickChoice` reduces stdin attack surface** — these functions read from `fmt.Scanln` and could have been a vector for unexpected input if called from a non-interactive context. Their deletion tightens the input surface. ✅

**No new dependencies introduced** — `gopkg.in/yaml.v3` import added to `config_test.go` was already a transitive dependency. No new packages added to `go.mod`. ✅

**`config.Load` gracefully handles missing `qode.yaml`** — `config.go:31` swallows `os.IsNotExist` errors, returning safe defaults. No path for a missing config to cause a panic or security-relevant error. ✅

---

## Adversary Simulation

1. **Attempt:** Pass a git flag as the `base` arg via `--` separator to alter branch creation behaviour | **Target:** `newBranchCreateCmd` / `git.CreateBranch` | **Result:** Partially succeeds — `qode branch create feat -- --orphan` reaches git as `git checkout -b feat --orphan`. Creates an unparented branch. No code execution, no privilege escalation. Blocked for external-facing scenarios because `qode` is a local developer tool invoked by the same user who owns the repository.

2. **Attempt:** Path traversal via branch name to write `.qode/branches/../../etc/cron.d/evil` | **Target:** `safeBranchDir` in `branch.go:14–21` | **Result:** Blocked. `filepath.Rel` check catches `..` components and returns error before any filesystem operation. Branch name `../../etc/cron.d/evil` is rejected at validation.

3. **Attempt:** Modify `qode.yaml` to inject a malicious `branch:` section that sets `keep_branch_context: true` to prevent cleanup of sensitive context files | **Target:** `config.Load` in `branch remove` | **Result:** Succeeds at preventing cleanup (context folder retained) but requires pre-existing write access to `qode.yaml` — same privilege as the legitimate user. No escalation of privilege. Consequence bounded to disk space and potential accidental context exposure before `.gitignore` fix lands.

**Why no higher-severity findings:** The changed code is a local CLI tool with no network I/O, no authentication, no shared secrets, and no database. The primary attack surface is subprocess invocation (git) and local filesystem operations, both of which are governed by the process owner's filesystem permissions. The `exec.Command` arg-array API prevents shell injection by design. The one meaningful injection vector (leading-dash `base` arg) requires the user to deliberately use `--` separator and produces bounded, non-escalating consequences.

---

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 1 |
| Informational | 1 |

**Overall security posture:** The changes introduce no new attack surface that could be exploited remotely or by an unprivileged attacker. The single Low finding (leading-dash argument injection into `base`) is bounded to local git behavior manipulation by the user themselves. The removal of stdin-reading wizard code is a net security improvement. The `config.Load` addition to `branch remove` is safe per the `IsNotExist` guard in `config.go`.

**Must-fix before merge (Critical/High):** None.

---

## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score                                                                                                      |
|-----------------------|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| Injection Prevention  | 1           | `exec.Command` variadic args prevent shell injection (verified `git.go:102`); `safeBranchDir` blocks filesystem path traversal (verified `branch.go:14`); `base` arg not validated against leading-dash git flag injection reachable via `--` separator — Low finding |
| Auth & Access Control | 2           | No auth or access control surfaces in scope; config-driven behaviour change requires write access to `qode.yaml` (same trust level as user running the tool) |
| Data Protection       | 2           | No secrets, PII, or credentials in changed code paths; `BranchConfig` contains only a boolean; no new logging of user input |
| Input Validation      | 1           | Branch name validated by `safeBranchDir` (unchanged); `base` arg lacks leading-dash sanitisation before subprocess invocation; wizard stdin-reading code deleted (reduces surface) |
| Dependency Security   | 2           | No new packages added to `go.mod`; `gopkg.in/yaml.v3` already present; no CVEs introduced |

**Total Score: 8.0/10**
