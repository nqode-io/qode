# Security Review — qode / reimplement-init-command

## Working Assumptions

**What this code trusts:**

- `root` (the project directory) is derived entirely from `resolveRoot()` → `config.FindRoot()`, which walks upward from the process's current working directory. The caller cannot supply an arbitrary path — the trust boundary is: whoever starts the process controls the CWD.
- `.qode/scoring.yaml` is treated as a project-local file that team members may edit. The code assumes its contents are valid YAML; no signature or integrity check is performed.
- The project name embedded in IDE slash commands is `filepath.Base(root)` — the last path component of the resolved root directory. This value is untrusted in the sense that an attacker who can rename the directory controls it, but a rename requires the same access as editing any file in the repo.

**Enforced assumptions:**
- Path construction always uses `filepath.Join` with `root` as the base; no user-supplied path components are introduced in this diff.

**Merely expected assumptions:**
- The `.qode/scoring.yaml` file is not manipulated by a malicious process racing against `qode init`.

---

## Adversary Simulation

1. **Attempt:** Deliver a malicious `.qode/scoring.yaml` with a crafted YAML bomb (deeply nested anchors, billion-laughs) to cause CPU/memory exhaustion | **Target:** `mergeScoringFromFile` → `yaml.Unmarshal` | **Result:** Blocked — Go's `gopkg.in/yaml.v3` library does not expand anchors recursively without bound in this way; memory is bounded by the concrete target struct `ScoringFileConfig`. Unmarshaling into a typed struct discards keys that don't match, and the library imposes recursion limits.

2. **Attempt:** Name the project directory with YAML-special characters (`my: project {evil}`) so the generated `.mdc` frontmatter becomes syntactically invalid and causes Cursor to misparse the slash command, potentially substituting attacker-controlled content as a command description | **Target:** `slashCommands(name)` → `description:` field in `.cursor/commands/*.mdc` | **Result:** Partially blocked — `fmt.Sprintf` embeds the string verbatim; the resulting `description: my: project {evil}` is invalid YAML. Cursor would fail to parse the file and the command would be unavailable. No code is executed; the worst outcome is a broken slash command. An attacker with the ability to rename the project directory already has full write access to the repo.

3. **Attempt:** Race condition — during `qode init`, after `os.Stat(scoringPath)` returns `IsNotExist` but before `os.WriteFile(scoringPath, ...)` completes, write a malicious `scoring.yaml` to inject crafted rubric dimensions that weaken quality gates | **Target:** `runInitExisting` — the non-atomic check-then-write at `internal/cli/init.go:71` | **Result:** Would succeed in theory (TOCTOU), but the attacker would need write access to `.qode/` — the same access required to edit `qode.yaml` or any branch context file. The impact is limited to rubric manipulation (quality gate weakening), not code execution or privilege escalation. This is the same class of risk as any config file edit.

---

## Findings

### Finding 1 — Informational
**Severity:** Informational  
**OWASP Category:** A05:2021 – Security Misconfiguration  
**File:** `internal/cli/init.go:66–70`  
**Vulnerability:** TOCTOU on scoring.yaml first-run guard. `os.Stat` is called and then `os.WriteFile` is called separately; a concurrent write between these calls could replace the file with attacker-controlled content. Additionally, non-`IsNotExist` stat errors are silently swallowed — the write is skipped without returning an error.  
**Exploit Scenario:** An attacker with write access to `.qode/` races the gap between `os.Stat` and `os.WriteFile` to plant a crafted `scoring.yaml`. Impact: custom rubrics that lower quality gate pass thresholds.  
**Remediation:** This is documented as intentional (`qode init` is best-effort). For teams requiring integrity guarantees, use `os.O_CREATE|os.O_EXCL` via `os.OpenFile` to make the write atomic-create (fails if file already exists). This eliminates the TOCTOU window and makes the first-run semantics explicit at the OS level. Not blocking for a developer CLI.

---

### Finding 2 — Low
**Severity:** Low  
**OWASP Category:** A03:2021 – Injection (YAML Injection into generated config)  
**File:** `internal/scaffold/cursor.go:32–148` and `internal/scaffold/claudecode.go:82–187`  
**Vulnerability:** The project name — `filepath.Base(root)` — is embedded verbatim into YAML frontmatter inside `.mdc` files via `fmt.Sprintf`. If the directory name contains YAML-special characters (`:`, `{`, `}`, `[`, `]`, `#`, `|`), the generated frontmatter becomes syntactically invalid. Example: a project in directory `my:project` produces `description: Refine requirements for my:project` which YAML parsers may interpret as a mapping value rather than a plain scalar.  
**Exploit Scenario:** A developer with a non-standard directory name (common on some systems) silently generates broken `.mdc` files. Cursor may discard the malformed frontmatter entirely or mis-parse the description. No code execution; worst case is a non-functional slash command.  
**Remediation:** Quote the project name in YAML frontmatter: `description: "Refine requirements for %s"`. Wrap the `%s` replacement in a YAML-safe quoting function that escapes embedded double-quotes if needed. This is a one-line change in both `slashCommands` and `claudeSlashCommands`.

---

### Finding 3 — Informational
**Severity:** Informational  
**OWASP Category:** A05:2021 – Security Misconfiguration  
**File:** `internal/cli/init.go:47`, `internal/config/config.go:44`  
**Vulnerability:** Generated files (`qode.yaml`, `.qode/scoring.yaml`) are written with permissions `0644`. On multi-user systems, this makes them world-readable. These files contain no credentials, but they do expose project topology and quality gate thresholds to all local users.  
**Exploit Scenario:** On a shared build server, another user reads `.qode/scoring.yaml` to learn what rubric dimensions are active, potentially crafting AI-generated content that scores well under the custom rubric. No direct code execution.  
**Remediation:** Not required for a developer CLI typically used on single-user workstations. If the project repository is on a shared server, file permissions are controlled by the repo's umask anyway. Acceptable as-is.

---

## Positive Security Properties Observed

- **No user input reaches shell commands.** All shell-adjacent code paths (`filepath.Join`, `os.MkdirAll`, `os.WriteFile`) use programmatically constructed paths rooted at `resolveRoot()`. No raw user string is passed to `exec.Command` or similar anywhere in this diff.
- **YAML deserialization into typed structs.** `mergeScoringFromFile` unmarshals into `ScoringFileConfig` — a concrete Go struct. Unknown keys are silently ignored; no `interface{}` deserialization that could be used to inject executable content.
- **`filepath.Base` prevents path traversal.** The project name used in IDE config content is derived from `filepath.Base(root)`, which strips directory separators. A path like `../../etc/passwd` becomes `passwd` — the traversal component is discarded before the value is used.
- **Idempotent scoring guard.** The `os.IsNotExist` check on `scoring.yaml` ensures the file is written at most once, protecting against re-run overwrites of user-customised rubrics. The guard also prevents blind writes to pre-existing files.
- **No new dependencies introduced.** The `go.mod` file is unchanged. No new external packages with unknown security histories are introduced.
- **IDE command files contain only hardcoded `qode` CLI invocations.** The generated `.md`/`.mdc` slash commands call `qode plan refine`, `qode review code`, etc. — the qode binary itself, not arbitrary shell. No user-supplied argument is interpolated into these command strings.

---

## Summary

| Severity | Count |
| --- | --- |
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 1 |
| Informational | 2 |

**Overall security posture:** Strong for a developer CLI tool. The attack surface of this diff is narrow: local filesystem writes using programmatically constructed paths, YAML config loading into typed structs, and static IDE command file generation. No network calls, no credential handling, no shell command construction from user input.

**Must-fix before merge (Critical/High):** None.

The Low finding (YAML frontmatter quoting) is a correctness issue as much as a security one — it should be fixed, but it is not a blocker for merge.

---

## Rating

| Dimension | Score | Control or finding that determines this score |
| --- | --- | --- |
| Command & Path Injection (0–3) | 2.8 | No user input reaches shell; `filepath.Base` strips traversal. Minor gap: project name from directory embedded verbatim in YAML frontmatter (L2 above) — cosmetic breakage, not code execution |
| Credential Safety (0–3) | 3.0 | No credentials in any written file; `qode.yaml` and `scoring.yaml` contain only non-sensitive config; 0644 permissions observed at `init.go:47` and `config.go:44` for non-sensitive files |
| Template Injection (0–3) | 2.7 | No `text/template` or `html/template` execution in this diff; `fmt.Sprintf` used for static content; project name from `filepath.Base` embedded in Markdown description strings (text-only context, not executed). Gap: YAML-special characters in project name (Finding 2) |
| Input Validation & SSRF (0–2) | 1.8 | No network calls; paths derived from CWD; `yaml.Unmarshal` into typed struct (`ScoringFileConfig`) rejects arbitrary keys; non-`IsNotExist` stat errors silently swallowed in scoring guard (intentional, documented in notes) |
| Dependency Safety (0–1) | 1.0 | No changes to `go.mod`; no new external packages introduced |

**Total Score: 11.3/12**  
**Minimum passing score: 10/12**
