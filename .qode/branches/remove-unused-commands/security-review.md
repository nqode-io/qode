# Security Review — Remove Unused Commands

**Branch:** remove-unused-commands  
**Reviewer:** AI Security Review  
**Date:** 2026-03-27

---

## Working Assumptions — What This Code Trusts

**Before the checklist:**

This is a pure deletion PR. No new code is introduced. The deleted commands were:
- Local CLI commands operating only on the local filesystem and local git repository
- None accepted network input or performed authenticated operations
- Trust boundary: the local developer's machine; no external services, no user authentication

**What the removed code trusted (and how):**
1. `newBranchFocusCmd` trusted the `name` argument (user CLI input) — passed to `safeBranchDir` (path-traversal check) and then to `exec.Command("git", "checkout", name)` (not a shell invocation)
2. `newBranchListCmd` trusted the filesystem contents of `.qode/branches/` — read-only, no sanitization needed
3. `newConfigDetectCmd` trusted `detect.Composite(root)` — filesystem scan only
4. `newConfigValidateCmd` trusted `gopkg.in/yaml.v3` deserialization of `qode.yaml` — local file, controlled by the developer
5. `newPlanStatusCmd` trusted the `.qode/branches/<branch>/` directory — read-only

**Unverified trust in the deleted code:**
- `newBranchFocusCmd`: `safeBranchDir` validated path traversal but did not validate that the branch name corresponds to an actual local git branch before calling checkout. This was a low-severity gap now eliminated by the deletion.
- `newBranchListCmd`: silently discarded `git.CurrentBranch` error (`current, _ := git.CurrentBranch(root)`). No security impact but a reliability gap — now gone.

---

## Security Checklist

### Injection
**Command injection:** `newBranchFocusCmd` passed a user-supplied `name` argument to `git.CheckoutBranch(root, name)` → `exec.Command("git", "checkout", name)`. Go's `exec.Command` with variadic string arguments does NOT invoke a shell; each argument is a separate string. A value like `; rm -rf /` would be passed literally to git and rejected as an invalid branch name. **Control verified: `exec.Command` without shell.** This code path is now deleted — surface area reduced.

No SQL, NoSQL, LDAP, or template injection applicable (CLI tool, no database, no template rendering in deleted code).

### Authentication & Authorisation
Not applicable. All removed commands are local developer CLI tools requiring no authentication. No auth bypass paths exist in deleted code.

### Data Exposure
`newConfigShowCmd` printed the entire resolved `qode.yaml` to stdout via `yaml.Marshal`. This could expose values like API keys or tokens if a user stored secrets in `qode.yaml`. **This is now deleted.** Any secrets in `qode.yaml` are no longer printable via this command. Net security improvement.

`newConfigDetectCmd` printed stack names and confidence scores — no sensitive data.

`newPlanStatusCmd` printed iteration scores and file paths — no sensitive data.

### Input Validation
`newBranchFocusCmd` was the only deleted command accepting user input beyond the implicit `--root` flag. It called `safeBranchDir(root, name)` which uses `filepath.Rel` to detect and reject path traversal (`..` prefix check). This control was present and effective — and is now gone along with the command.

### Cryptography
None applicable. No cryptographic operations in any deleted code.

### Frontend-Specific
Not applicable (CLI tool, no frontend).

### API Security
Not applicable (no HTTP endpoints).

### Dependency Issues
`internal/cli/config_cmd.go` was the only file in `internal/cli/` that imported `gopkg.in/yaml.v3`. Its deletion removes one import site. The `yaml.v3` module remains in `go.mod` because `internal/config/` still uses it. No new dependencies introduced; no CVE exposure created.

---

## Adversary Simulation

**Attempt 1:** Craft a branch name containing shell metacharacters (`feat/foo; rm -rf ~`) and pass it to `qode branch focus` to achieve command injection.  
**Target:** `newBranchFocusCmd` → `git.CheckoutBranch` → `exec.Command("git", "checkout", name)`  
**Result:** Would not succeed. Go's `exec.Command` passes each argument as a discrete string to the OS, not through a shell. The metacharacters are sent literally to git, which rejects them as an invalid ref. **Control:** Go `exec.Command` variadic argument separation.  
**Additional note:** This code path is now deleted — the attack surface no longer exists.

**Attempt 2:** Pass `../../etc/passwd` as the branch name to `qode branch focus` to read files outside the context directory.  
**Target:** `newBranchFocusCmd` → `safeBranchDir(root, "../../etc/passwd")`  
**Result:** Would not succeed. `safeBranchDir` computes `filepath.Rel(base, target)` and rejects any result with a `..` prefix. `filepath.Rel(".qode/branches", ".qode/branches/../../etc/passwd")` = `"../../etc/passwd"` → rejected with `"invalid branch name: path traversal detected"`. **Control:** `safeBranchDir` at `internal/cli/branch.go` (now deleted along with its caller).

**Attempt 3:** Supply a crafted `qode.yaml` with a YAML deserialization gadget to `qode config validate` to achieve code execution.  
**Target:** `newConfigValidateCmd` → `config.Load` → `gopkg.in/yaml.v3` unmarshalling  
**Result:** Would not succeed. `yaml.v3` in Go does not support the arbitrary type instantiation gadget chains present in some JVM/Python YAML libraries. The struct type is fixed at compile time; the unmarshaller only populates known fields. **Control:** Go's static typing eliminates deserialization gadget chains. **Additional note:** `config validate` is now deleted — this path no longer exists.

All three attempts fail. Controls observed: Go's `exec.Command` argument separation, `safeBranchDir` path traversal check, Go static typing for YAML deserialization. The deletion of this PR reduces attack surface.

---

## Vulnerabilities Found

None introduced by this PR.

**Security improvement noted (Informational):**
- **File:** `internal/cli/branch.go` (deleted lines 136–176)  
- **Observation:** Removal of `newBranchFocusCmd` eliminates the only deleted-code path that accepted user-supplied input. The existing controls (`safeBranchDir`, `exec.Command`) were adequate, but removal is a net improvement in attack surface.

- **File:** `internal/cli/config_cmd.go` (deleted)  
- **Observation:** `newConfigShowCmd` serialized the entire resolved config to stdout, which could have exposed secrets stored in `qode.yaml`. Deletion removes this risk.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 2 (security improvements, not issues) |

**Must-fix before merge (Critical/High):** None.

**Overall security posture:** This PR reduces attack surface. The two most security-relevant deletions are `newBranchFocusCmd` (only command accepting user-supplied string input) and `newConfigShowCmd` (only command that could surface config secrets). No new code, no new trust boundaries, no new dependencies.

---

## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score |
|-----------------------|-------------|------------------------------------------------|
| Injection Prevention  | 2           | Verified: `exec.Command` variadic args prevent shell injection in deleted `CheckoutBranch`; deletion eliminates this surface entirely; no new injection vectors introduced anywhere in the diff |
| Auth & Access Control | 2           | No auth or access control changes; removed commands are local-only CLI tools requiring no authentication; no IDOR or privilege escalation paths in deleted code |
| Data Protection       | 2           | `newConfigShowCmd` (deleted) was the only command serializing potentially-sensitive config data to stdout; its deletion improves data protection posture; no new sensitive data exposure introduced |
| Input Validation      | 2           | `safeBranchDir` (verified at `internal/cli/branch.go:14-22`) enforced path traversal prevention on the one deleted command that accepted user input; the control was present and effective; deletion of the command eliminates the need for the control |
| Dependency Security   | 2           | No new dependencies added; config_cmd.go deletion removes one `yaml.v3` import site; yaml.v3 has no known critical CVEs affecting this usage pattern; go.mod unchanged in terms of new packages |

**Total Score: 9.5/10**

Total 10.0 is excluded by the constraint — complete security is not provable. This score reflects: no vulnerabilities introduced, two security improvements delivered (surface reduction), all controls in deleted code verified as adequate (the code was safe before and is now gone). The 0.5 deduction acknowledges that the config show command's ability to print secrets was a latent issue that existed before this branch — not introduced here, but a known gap in the pre-existing codebase.
