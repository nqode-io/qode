# Security Review — qode (harden-review-prompts)

## Working Assumptions

**What this code trusts:**
- The developer who commits template content is a trusted contributor (git history + PR review is the control)
- `internal/prompt/engine.go`'s `text/template` engine processes only `{{...}}` directives; plain Markdown is passed through verbatim
- The AI reviewer's output is an untrusted text document — only the `Total Score: X.X/10` line is machine-parsed (`internal/scoring/extract.go:9`)
- `.claude/commands/*.md` and `.cursorrules/*.mdc` files are read by IDE clients, not executed by qode itself

**Which assumptions are enforced vs. merely expected:**
- Enforced: Go's `text/template` will not execute plain Markdown as code — this is a language guarantee, not a convention
- Enforced: `internal/scoring/extract.go` parses only one line; all other review content is inert to the scoring pipeline
- Merely expected: The AI reviewer does not take harmful real-world actions based on the Adversary Simulation prompt — this is a model behaviour assumption, not a code control

**Unverified trust:** None that introduce security risk in this changeset. The template content is developer-authored static text with no runtime input path.

---

## Adversary Simulation

**Attempt 1: Template injection via new Markdown content**
- **Attempt:** Introduce `{{.SomeVar}}` or `{{call .Func}}` directives in the new template sections to execute arbitrary Go template logic when `qode review code` renders the prompt
- **Target:** `internal/prompt/templates/review/code.md.tmpl:8-15` (Reviewer Stance), `security.md.tmpl:83-92` (Adversary Simulation)
- **Result:** Would not succeed — all new content is plain Markdown text. I verified: zero occurrences of `{{` or `}}` in any new section. Go's `text/template` engine passes non-directive text through unchanged. No code execution path exists for attacker-controlled template injection via this diff.

**Attempt 2: Prompt injection via Adversary Simulation output**
- **Attempt:** Craft the "Inhabit the role of someone actively attempting to exploit" instruction to cause the AI reviewer to execute shell commands or make network requests when processing a code review
- **Target:** `security.md.tmpl:85` — "Inhabit the role..." instruction
- **Result:** Would not succeed — the AI reviewer produces text output saved to `.qode/branches/<branch>/security-review.md`. That file is only read by `internal/scoring/extract.go`, which applies a single regex to extract the score. No review content is interpreted as commands by qode. The IDE presents the file as Markdown; it cannot trigger code execution.

**Attempt 3: IDE command hijack via `.claude/commands/` modifications**
- **Attempt:** Leverage the blank-line additions to the five `.claude/commands/*.md` files to inject additional instructions that the IDE would execute on the developer's behalf
- **Target:** `.claude/commands/qode-review-code.md`, `qode-review-security.md` (and 3 others)
- **Result:** Would not succeed — the additions are a single blank line (`\n`) after the H1 title in each file. No instruction text is introduced. Content is semantically identical to pre-branch state, verified by reading all five files.

**Controls that stopped all three attempts:**
1. `text/template` engine is instruction-ignorant toward plain Markdown — hard language guarantee
2. `internal/scoring/extract.go:9` reads exactly one regex match from review output; all other content is discarded
3. Blank line additions carry zero semantic payload in Markdown

---

## Security Checklist

### Injection
- **Template injection:** Not present — all new content in `code.md.tmpl` and `security.md.tmpl` is plain Markdown, verified by grep. Zero new `{{` directives introduced. `text/template` will not interpret any new content as executable.
- **Command injection:** Not applicable — no shell commands in the diff. No user input reaches template content.
- **SQL/NoSQL/LDAP injection:** Not applicable — CLI tool with no database.

### Authentication & Authorisation
- No authentication or authorisation logic modified. Templates are static embedded assets. No access control changes.

### Data Exposure
- New template sections do not introduce log statements, error messages, or API responses. The Adversary Simulation section requests the AI describe hypothetical exploit attempts — this text appears only in the local review file, not in logs or external services.
- Informational: The Adversary Simulation output, if shared externally, documents exploit hypotheses for the reviewed changeset. This is deliberate and desirable for security reviewers; it is not a data leakage concern for this CLI tool.

### Input Validation
- No new runtime inputs introduced. Existing template variables (`{{.Diff}}`, `{{.Project.Name}}`, `{{.Branch}}`, `{{.Layers}}`, `{{.Spec}}`, `{{.OutputPath}}`) are unchanged — their validation is out of scope for this diff.
- The new Adversary Simulation section includes `[what you'd try]` placeholder text — this is instructional Markdown rendered to the AI, not a user input field. No validation required.

### Cryptography
- Not applicable. No cryptographic operations in the diff.

### Frontend-Specific
- Not applicable. Go CLI tool.

### API Security
- Not applicable. No HTTP endpoints introduced or modified.

### Dependency Security
- Zero new Go dependencies in this diff (`go.mod` and `go.sum` are unchanged). All changes are to Markdown/template files.

---

## Summary

**Vulnerabilities by severity:**
- Critical: 0
- High: 0
- Medium: 0
- Low: 0
- Informational: 1 (see below)

**Informational — Adversary Simulation output sensitivity**
- **OWASP Category:** Not applicable (not a vulnerability)
- **File:** `internal/prompt/templates/review/security.md.tmpl:83-92`
- **Note:** The section instructs the AI to describe three exploit attempts in the saved review file. If `.qode/branches/<branch>/security-review.md` is committed to a public repository, it would document hypothetical attack vectors for the reviewed changeset. This is standard security review practice; the risk is negligible for a CLI workflow tool where review files are branch-scoped. No remediation required — worth noting for teams that commit review artifacts to public repos.

**Must-fix before merge (Critical/High only):** None.

**Overall security posture:** The changeset introduces no new attack surface. All modifications are to static Markdown text embedded in a binary. The trust model is unchanged. The most security-sensitive new element — the Adversary Simulation section — is structurally inert to the scoring pipeline and cannot cause code execution.

---

## Rating

| Dimension             | Score (0-2) | Control or finding that determines this score |
|-----------------------|-------------|------------------------------------------------|
| Injection Prevention  | 2           | Zero `{{` directives in new content (verified by grep); `text/template` does not execute plain Markdown — language guarantee, not convention |
| Auth & Access Control | 2           | No auth logic modified; templates are static assets compiled into binary; no new access paths created |
| Data Protection       | 2           | No sensitive data in new content; Adversary Simulation output is local-only by default; no new logging, no new API calls |
| Input Validation      | 2           | No new runtime inputs; existing template variables unchanged; placeholder text in Adversary Simulation is non-executable Markdown |
| Dependency Security   | 2           | `go.mod` and `go.sum` unchanged; zero new dependencies |

**Total Score: 9.5/10**

Constraints check: No Critical or High vulnerabilities — no score cap applies. Score ≥ 8.0 justified by: Injection Prevention verified via grep (zero `{{` in new content); Auth unchanged verified by reading all modified files; Data Protection confirmed via full diff review — no log calls, no external writes, no sensitive literals. Score is 9.5 rather than 10.0 because complete security is not provable: the Adversary Simulation prompt's effect on AI model behaviour cannot be formally guaranteed, and the out-of-scope `.cursorrules` step ordering change represents unverified drift that a thorough security review must acknowledge even when risk is negligible.
