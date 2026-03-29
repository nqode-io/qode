# Security Review — reimplement-qode-check

## Working Assumptions

**What this code trusts:**

- `qode.yaml` is a checked-in project configuration file — treated as trusted. Any party who can modify `qode.yaml` already has write access to the repository.
- The `root` parameter passed to `SetupCursor`/`SetupClaudeCode` derives from the process working directory — trusted OS-level input.
- The AI executing the generated `/qode-check` prompt trusts the project's build system (e.g., `npm test` executes whatever `scripts.test` is in `package.json`). This is the inherent trust model for "AI runs your tests" and is accepted by design — not a new assumption in this PR.

**What is NOT trusted:** user-supplied arguments at the command line, HTTP responses, external data. None appear in this diff.

**Assumptions that are enforced vs. merely expected:**

- The `root`/write-path assumption is enforced by using hardcoded constants (`cursorRulesDir`, `cursorCommandsDir`) rather than config-driven values — this is a **regression fix** compared to the prior code (see finding below).
- The config-trust assumption is merely expected — no schema validation prevents `knowledge.path: ../../etc` (see informational finding).

---

## Security Checklist

### Injection

**Command injection:** No `exec.Command` or shell invocation appears anywhere in the diff. The prior `internal/runner/runner.go` (deleted in this branch) was the only location where config-derived strings were passed to `exec.Command`. Its deletion is a net security improvement — this attack surface no longer exists.

**Path traversal via config-driven write paths (eliminated):** The prior `cursor.go` used `cfg.IDE.Cursor.RulesDir` and `cfg.IDE.Cursor.CommandsDir` as directory components in `filepath.Join(root, rulesDir, "file")`. Since `filepath.Join` does not sanitize `..` segments against `root`, a crafted `qode.yaml` with `rules_dir: ../../.ssh` would have caused `writeFile` to write files to arbitrary paths outside the project directory. The new code replaces these config-driven paths with hardcoded constants:

```go
const cursorRulesDir = ".cursorrules"
const cursorCommandsDir = ".cursor/commands"
```

This eliminates the write-path traversal. **Net improvement.**

**Template/prompt injection into generated files:** `cfg.Project.Name` is injected via `fmt.Sprintf` into markdown content (e.g., `claudecode.go:138`). The result is written to `.md`/`.mdc` files. These files are read by an AI as a text prompt — they are not evaluated by a shell or template engine. Injecting shell metacharacters into the project name would have no effect.

**SQL/NoSQL/LDAP injection:** Not applicable — no database or directory queries.

### Authentication & Authorisation

Not applicable. This is a local-filesystem CLI tool with no network requests, no authentication tokens, and no access control decisions. No changes to auth surfaces.

### Data Exposure

`writeFile` (cursor.go:298) uses mode `0644` — appropriate for IDE configuration files (markdown/text). Generated content is AI prompts. No secrets, credentials, or PII in `qodeCheckBody` (compile-time constant) or any other generated content.

No logging of user data. `fmt.Printf` output is limited to counts: `"Claude Code: %d slash commands\n"` and `"Cursor: %s/ (%d rules, %d commands)\n"`.

### Input Validation

**Residual finding — `knowledge.path` traversal (Informational):** `knowledge.go:242` constructs the knowledge directory path as:

```go
kbDir := filepath.Join(root, kbPath)
```

where `kbPath` comes from `cfg.Knowledge.Path` (from `qode.yaml`). Setting `path: ../../etc` in `qode.yaml` would cause `List()` to scan outside the project root. However:

1. This is read-only — `List` returns file paths; it does not write.
2. The config is a trusted checked-in file; modifying it requires repository write access.
3. The prior code had the same class of issue via `cfg.Knowledge.Paths []string`.

Severity: **Informational** — the attacker who can modify `qode.yaml` already has full repository access.

All write paths (`claudecode.go:20`, `cursor.go:25`, `cursor.go:30`, `cursor.go:37`) use hardcoded constant directory strings, not user-controlled values.

### Cryptography

Not applicable. No cryptographic operations in the diff.

### API Security

Not applicable. No HTTP, network, or API calls in the diff.

### Dependency Security

No new entries in `go.mod` — confirmed by diff (no `go.mod` changes). No new dependencies introduced.

---

## Adversary Simulation

1. **Attempt:** Craft `qode.yaml` with `rules_dir: ../../.ssh` to overwrite `~/.ssh/authorized_keys` when the victim runs `qode ide setup` | **Target:** old `cursor.go:SetupCursor` → `writeFile` | **Result:** BLOCKED — the `rules_dir` config field has been removed from `CursorIDEConfig` (schema.go); `SetupCursor` now uses the hardcoded constant `cursorRulesDir = ".cursorrules"`. The field is silently ignored even if present in `qode.yaml`.

2. **Attempt:** Craft `qode.yaml` with `knowledge: path: ../../etc` to exfiltrate `/etc/passwd` via `qode knowledge list` | **Target:** `knowledge.go:242` `filepath.Join(root, kbPath)` | **Result:** Limited — `List()` would scan `/etc` and return file paths to the caller, which then includes them in prompts. However: (a) this is read-only — no write occurs, (b) modifying `qode.yaml` requires repo write access, (c) an attacker with repo write access has far more direct exfiltration paths. Informational only; no fix required.

3. **Attempt:** Inject shell commands into `cfg.Project.Name` (e.g., `$(rm -rf ~)`) to achieve code execution when `qode ide setup` is run | **Target:** `claudecode.go:138` `fmt.Sprintf("# Quality Gates — %s\n\n", name)` | **Result:** BLOCKED — the injected string is written to a `.md` file via `os.WriteFile`. No shell interpreter reads this file during `qode ide setup`. The AI that later reads the file interprets it as a text prompt in a sandboxed conversation, not as a shell command.

---

## Summary

| Severity      | Count |
|---------------|-------|
| Critical      | 0     |
| High          | 0     |
| Medium        | 0     |
| Low           | 0     |
| Informational | 1     |

**Overall security posture:** This branch is a net security improvement. The primary change — deleting `runner.go` and removing config-driven write paths in `cursor.go` — eliminates the two most significant attack surfaces in the prior code. The `qodeCheckBody` constant introduces no new attack surface. The one informational finding (read-only path traversal via `knowledge.path`) predates this branch and does not require a fix before merge.

**Must-fix before merge:** None.

---

## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score |
|-----------------------|-------------|------------------------------------------------|
| Injection Prevention  | 2           | `exec.Command` removed (runner.go deleted); write paths use hardcoded constants `cursorRulesDir`/`cursorCommandsDir` (cursor.go:12-13); `qodeCheckBody` is a compile-time Go constant (claudecode.go:31); `fmt.Sprintf` injects project name into markdown, not a shell |
| Auth & Access Control | 2           | No network calls, no auth tokens, no privileged operations — local filesystem CLI only |
| Data Protection       | 2           | `os.WriteFile` at cursor.go:298 uses mode `0644`; generated content is AI prompts (no secrets); `fmt.Printf` output limited to numeric counts |
| Input Validation      | 1.5         | `filepath.Join(root, kbPath)` at knowledge.go:242 uses untrusted config path without `..` sanitization (read-only, informational; predates this PR); all write paths use hardcoded constants |
| Dependency Security   | 2           | No new Go modules added (go.mod unchanged in diff) |

**Total Score: 9.5/10**

The 0.5 deduction reflects the residual read-only path traversal via `cfg.Knowledge.Path`. This is an informational finding that predates this branch and is accepted given the config-trust model.
