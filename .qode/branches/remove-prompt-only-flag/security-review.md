# Security Review — remove-prompt-only-flag

**Branch:** remove-prompt-only-flag
**Reviewer:** Claude (qode-review-security)
**Date:** 2026-03-18
**Score:** 9.5/10

---

## Summary

This branch removes `--prompt-only` / interactive dispatch from all prompt-generating commands and deletes the `internal/dispatch/` package and VSCode IDE support entirely. From a security perspective this is a **net positive change** — it eliminates subprocess execution and reduces the codebase's attack surface. The only finding is a minor permission issue on directories created by `writePromptToFile`.

> **Diff scope note:** `git review security` pulls the full `git diff main` which includes committed `.qode/branches/feat-support-apps-directory/` artifacts from the previous PR. Those are not part of this branch's changes. Review focused on `internal/` source changes per `git diff main --stat -- internal/`.

---

## Critical Vulnerabilities

None.

---

## High Vulnerabilities

None.

---

## Medium Vulnerabilities

None.

---

## Low Vulnerabilities

### L1 — Branch directory created with world-readable permissions (0755)

**OWASP Category:** A05:2021 – Security Misconfiguration
**Location:** [internal/cli/root.go:93](internal/cli/root.go#L93)

```go
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
```

**Issue:** The `.qode/branches/<branch>/` directory is created with mode `0755` (world-readable, world-executable). Prompt files written here can contain sensitive data: ticket content (credentials, internal URLs, PII in descriptions), full code diffs (potential trade secrets), internal architecture details from `spec.md`, and LLM instructions.

While the prompt **files** themselves are created with mode `0600` (via `os.CreateTemp` which uses `O_EXCL | 0600`), the directory is `0755`, so any local user on a shared machine can:
- List the directory and see file names (revealing branch names and activity)
- Attempt to read files (blocked by `0600` file permission, but names are exposed)

On developer laptops this is typically not exploitable (single user), but on shared CI runners or multi-user development servers this is a meaningful exposure.

**Remediation:**
```go
// Change 0755 → 0700 (owner-only access to the directory)
if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
```
This is consistent with how tools like `ssh`, `gpg`, and `git` protect sensitive local directories.

---

## Informational / Positive Findings

### I1 — Dispatch removal eliminates subprocess injection surface (OWASP A03)

The deleted `internal/dispatch/claude.go` previously ran `claude -p <prompt>` as a subprocess. The prompt content was passed directly to an external process and could, in edge cases (e.g. shell metacharacters in ticket content, path handling), create unintended behavior. Removing subprocess execution entirely eliminates this attack vector.

### I2 — `writePromptToFile` uses correct atomic write pattern

The temp-file-then-rename pattern (`os.CreateTemp` → write → close → `os.Rename`) prevents:
- Partial writes being read by concurrent processes
- TOCTOU races on the output file
- Leftover temp files on crash (via `defer os.Remove`)

`os.CreateTemp` creates files with mode `0600` (owner read/write only), which is the correct restrictive default for files that may contain sensitive prompt content.

### I3 — Branch name used in file path is safe

`branch` is sourced from `git.CurrentBranch()` which reads `git rev-parse --abbrev-ref HEAD`. Git validates branch names at creation time and rejects path-traversal sequences (`..`), so no path traversal is possible via branch name.

### I4 — VSCode setup removal reduces file system write scope

Removing `internal/ide/vscode.go` eliminates writes to `.vscode/` (tasks.json, settings.json, extensions.json, launch.json). Each removed write is one less opportunity for misconfiguration or unintended file creation in user-visible directories.

---

## Verdict

The change is security-positive. The only finding (L1) is a minor directory permissions issue that should be fixed before the feature goes to production environments. No critical, high, or medium vulnerabilities found.
