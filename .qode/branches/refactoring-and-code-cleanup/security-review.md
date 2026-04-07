# Security Review — qode

You are a security engineer performing a security-focused code review.
Read this diff as a map: trace every path from external input to persistent
state, external service, or sensitive data. Your job is to find where that
map leads somewhere it shouldn't.

## Working Assumptions

**What does this code trust?**
- The project root path (`--root` flag or working directory) — caller-supplied
- Branch names from git (`git.CurrentBranch()`) — trusts git output, passed through `git.SanitizeBranchName()`
- `QODE_LOG_LEVEL` environment variable — parsed against a closed `switch`, safe
- Config values from `qode.yaml` — now explicitly validated via `config.Validate()`
- User-provided file paths in `knowledge add <path>` — arbitrary local file read by design
- Context files in `.qode/branches/*/context/` — read via `iokit.ReadFileOrString`

**Enforced vs. merely expected:**
- Branch name sanitization: enforced (`git.SanitizeBranchName`)
- Base-branch flag injection: enforced — `base[0] == '-'` guard in `runBranchCreate`
- Config rubric validation: newly enforced on every `config.Load`
- Template data sourcing: user-controlled content is loaded as *string data*, never as template text — Go template engine does not recursively parse directives embedded in data values

---

## Vulnerabilities Found

### 1. Knowledge Base — Inadvertent Sensitive File Inclusion

- **Severity:** Medium
- **OWASP Category:** A05:2021 – Security Misconfiguration
- **File:** `internal/cli/knowledge_cmd.go` (`runKnowledgeAdd`)
- **Vulnerability:** `qode knowledge add <path>` reads any local file path and copies it into `.qode/knowledge/` with `0644` permissions. There is no allowlist on file extensions, no warning about sensitive files, and the destination directory is not in `.gitignore`. A developer who accidentally runs `qode knowledge add ~/.aws/credentials` or `qode knowledge add .env` silently copies credentials into a tracked directory.
- **Exploit Scenario:** Developer runs `qode knowledge add context/notes.md` but tab-completes to an adjacent `context/.env` file. The credentials are written to `.qode/knowledge/.env` at `0644`, potentially committed to git on the next `git add .`.
- **Remediation:** Add `.qode/knowledge/` to `.gitignore` (like `.qode/branches/*/context/ticket.md` is already excluded), and/or emit a warning when the source file has no extension or matches common credential patterns (`.env`, `credentials`, `*.pem`, `*.key`).

---

### 2. AtomicWrite — Permissions Rely on chmod-then-rename Ordering

- **Severity:** Low
- **OWASP Category:** A01:2021 – Broken Access Control
- **File:** `internal/iokit/iokit.go:74-93` (`AtomicWrite`)
- **Vulnerability:** `os.CreateTemp` creates the temp file with default mode `0600` (subject to umask). `os.Chmod` is then called *before* `os.Rename`. This is correct ordering, but if the process is interrupted between `Chmod` and `Rename` on a shared filesystem, a window exists. More concretely: on systems where umask is `000` (some CI containers), the temp file is created world-readable before `Chmod` narrows it. The rename then preserves the post-chmod permissions correctly, but a race exists during the write phase.
- **Exploit Scenario:** On a shared CI agent with `umask 000`, a concurrent process could read the temp file (`.tmp-XXXXXX`) while the prompt content is being written, before `Chmod(0600)` is called.
- **Remediation:** Call `os.Chmod` on the temp file *immediately after* `os.CreateTemp`, before writing any content:
  ```go
  tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
  if err != nil { return ... }
  if err := os.Chmod(tmp.Name(), perm); err != nil { ... } // chmod before write
  defer func() { _ = os.Remove(tmp.Name()) }()
  if _, err := tmp.Write(data); err != nil { ... }
  ```
  This closes the window entirely.

---

### 3. EnsureDir — World-Traversable Context Directories

- **Severity:** Informational
- **OWASP Category:** A01:2021 – Broken Access Control
- **File:** `internal/iokit/iokit.go:97-102` (`EnsureDir`)
- **Vulnerability:** All context directories are created with `0755` (world-traversable). On a shared development machine, other local users can list and enter `.qode/branches/*/context/` and read files stored there with `0644`.
- **Exploit Scenario:** On a shared developer workstation or CI node, a local user runs `ls ~/.../project/.qode/branches/feature-x/context/` and reads ticket descriptions or refined analysis containing proprietary requirements.
- **Remediation:** Consider creating context directories with `0700` and files with `0600` if the project targets shared-machine environments. For a typical single-user developer laptop, `0755`/`0644` is acceptable and matches existing project conventions.

---

## Adversary Simulation

1. **Attempt:** Inject a git flag via the `--base` argument of `branch create` | **Target:** `runBranchCreate` → `git.CreateBranch(root, name, "--delete")` | **Result:** **Blocked** — `base[0] == '-'` guard returns an error before `git.CreateBranch` is called.

2. **Attempt:** Path traversal in `knowledge add` to write outside the knowledge base | **Target:** `runKnowledgeAdd(out, "../../sensitive/file.md")` → `dest = filepath.Join(kbDir, filepath.Base(src))` | **Result:** **Blocked** — `filepath.Base("../../sensitive/file.md")` returns `"file.md"`, so the destination is always `<kbDir>/file.md`. The *source* file is read as-is (arbitrary local file read, by design), but the write destination is confined to the knowledge base directory.

3. **Attempt:** Template injection via adversarial ticket.md content containing Go template directives (e.g., `{{os.Exit 1}}`) | **Target:** `branchcontext.ReadFileOrString("context/ticket.md")` → `prompt.Engine.Render()` | **Result:** **Blocked** — User content is loaded as a `string` value into `TemplateData.Ticket` and rendered into template output as data (e.g., `{{.Ticket}}`). Go's `text/template` engine does not recursively parse template directives found *inside* a data value; they are emitted as literal text. No code is executed.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 1 |
| Informational | 1 |

**Overall security posture:** This is a structural refactoring PR (io.Writer injection for testability, `iokit` centralization, `branchcontext` package rename, `config.Validate` addition). No new attack surface is introduced. The `config.Validate` addition is a positive security improvement — previously malformed rubric configs were silently accepted. The flag-injection guard (`base[0] == '-'`) was preserved correctly through the refactor.

The principal residual risk is the knowledge base's lack of a gitignore entry and the absence of any guardrails against accidentally adding credential files.

**Must-fix before merge:** None (no Critical or High vulnerabilities).

**Recommended follow-up:**
- Add `.qode/knowledge/` to `.gitignore` to prevent accidental credential commits (Medium — #1 above)
- Apply `os.Chmod` before writing in `AtomicWrite` to eliminate the permission-race window (Low — #2 above)

---

## Rating

| Dimension | Score | Control or finding that determines this score |
|-----------|-------|------------------------------------------------|
| Command & Path Injection (0–3) | 2.5 | Flag injection blocked by `base[0] == '-'` at `internal/cli/branch.go:316`; path writes confined by `filepath.Base()` at `internal/cli/knowledge_cmd.go:1319`; git invocations use `exec.Command` args (no shell). Minor: `knowledge add` reads arbitrary source paths by design. |
| Credential Safety (0–3) | 2.5 | Prompt files written at `0600` via `iokit.AtomicWrite` at `internal/cli/util.go:246`; context/config files at `0644` (appropriate for local dev tool). Risk: no gitignore on `.qode/knowledge/` and no guard against credential files in `runKnowledgeAdd`. |
| Template Injection (0–3) | 3.0 | User content loaded as typed string fields in `TemplateData`, never as template text. `text/template` does not recursively execute directives within data values. Confirmed by tracing `ctx.Ticket` → `data.Ticket` → `{{.Ticket}}` in templates. |
| Input Validation & SSRF (0–2) | 2.0 | `config.Validate()` enforced on every load; rubric keys allowlisted; numeric fields range-checked. `QODE_LOG_LEVEL` parsed via closed switch. No network requests; SSRF not applicable. |
| Dependency Safety (0–1) | 1.0 | No new external dependencies introduced. Only stdlib additions (`log/slog`, `io`). |

**Total Score: 11.0/12**
**Minimum passing score: 10/12**
