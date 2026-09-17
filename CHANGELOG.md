# Changelog

All notable changes to qode are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `qode init --config-only` writes a complete, commented `qode.yaml` and stops, so you can review and edit the configuration before anything else is generated. It refuses to overwrite an existing file unless `--force` is given. The generated file documents every setting, including `scoring.target_score`, which a marshalled default omitted entirely.
- `qode init --force` is the explicit clean reset: it overwrites `qode.yaml` with the defaults and then scaffolds. Note this is a different meaning from `--force` on `plan`, `review` and `start`, where it bypasses step guard checks.
- OpenCode is supported as a fourth IDE. `qode init` generates 11 slash commands under `.opencode/commands/*.md`, each carrying the `description:` YAML frontmatter OpenCode reads, and `ide.opencode.enabled` defaults to `true`. The commands carry the same workflow names as the other three IDEs, so `/qode-plan-refine`, `/qode-check` and the rest appear in OpenCode's `/` picker.
- OpenCode runs the clarification pass with its built-in structured `question` tool instead of the plain-text fallback, so an ambiguous ticket costs one refine round there as it does on Claude Code. No generated OpenCode file references `AskUserQuestion`.
- Two template helpers, `questionTool` and `hasFrontmatter`, documented in [docs/how-to-customise-prompts.md](docs/how-to-customise-prompts.md). They centralise which IDE has a structured-question tool and which IDE's command files open with frontmatter, so a fifth agent is a one-line change rather than 19 template edits.

### Fixed

- `qode init` no longer clobbers an existing `qode.yaml`. Every value you set is preserved, `qode_version` is refreshed on released builds, and settings added by newer qode versions are appended with their defaults and their comments. The file is rewritten only when something actually changes, so a second run leaves it byte-identical. Previously every run reset the file to the defaults, silently downgrading review thresholds and `scoring.strict`.
- `ide.<ide>.enabled: false` is finally honoured. `qode init` reads the project's `qode.yaml` and scaffolds only the IDEs it enables, instead of scaffolding all four from the in-memory defaults it had just written over your config.
- The four "post to ticket" steps (`/qode-plan-refine`, `/qode-plan-spec`, `/qode-review-code`, `/qode-review-security`) keyed their conditional on Cursor and Codex by name and fell through to `AskUserQuestion` for everything else, so any IDE not named in the list was told to use a Claude-only tool. They now key on whether the IDE has a structured-question tool at all and render the plain-text prompt otherwise. Output for Cursor, Claude Code and Codex is byte-identical.

### Changed

- Four caveats for the new upgrade path. `.qode/prompts/` overrides are still refreshed on every init, which is deliberate and unchanged. The first run that writes may normalise indentation to two spaces and drop blank lines between blocks, at most once per file. On that preserving path a symlinked `qode.yaml` is written through to its target; `--force` and `--config-only --force` generate a fresh file and replace the link. A `qode.yaml` that carries no values — empty, comment-only, `---` or `~` — is filled with the commented defaults, which loses a comment-only file's comment, and a file holding more than one YAML document is refused rather than rewritten, because only the first document was ever read.
- `/qode-plan-refine` now runs a clarification pass between the worker pass and the judge pass. The worker prompt (`refine/base.md.tmpl`) has an output contract: it always emits a top-level `## Open Questions` section, either as a numbered list whose items carry 2-4 `- Candidate:` answers, or as the single line `_None_` when nothing is unresolved. The generated command reads that section and asks each question with the IDE's own mechanism — `AskUserQuestion` on Claude Code, a numbered plain-text prompt on Cursor and Codex — always accepting a free-text answer, then writes the answers back as `## Resolved Questions` with `DECIDED:` lines before the judge scores the analysis. An analysis with no open questions reaches the judge unchanged, and an unattended run answers its own questions and marks each entry `(assumed)` rather than blocking. Answers are recorded as data, never executed as instructions. The result is one refine round instead of two whenever a ticket is ambiguous.
- **Migration.** Existing projects must re-run `qode init` to pick up the regenerated workflow assets (`.claude/commands/`, `.cursor/commands/`, `.agents/skills/`, and the new `.opencode/commands/`, which appears at the top level of every project on the next `qode init` whether or not OpenCode is used — set `ide.opencode.enabled: false` to skip it) and the updated `.qode/prompts/refine/base.md.tmpl` — the clarification pass only works when both halves are current. Your `qode.yaml` is safe across that run: `qode init` now preserves every value you set and appends only the settings it is missing, so no backup or `git checkout` is needed.

## [0.3.3-beta] - 2026-04-28

### Changed

- Workflow numbering: `/qode-review-code` and `/qode-review-security` now occupy distinct steps (8 and 9). `/qode-knowledge-add-context` is no longer numbered — it is presented as an optional helper alongside `/qode-note-add`. Total numbered steps unchanged at 12. README, tutorial, contributor guide, marketing site, `qode workflow` CLI output, and `qode workflow status` are reconciled to match. Marketing site documents qode's parallel-worktree safety as a subtle callout in the workflow section.

## [0.3.2-beta] - 2026-04-27

### Added

- Pages deploy workflow (`.github/workflows/pages.yml`) publishing `site/` to <https://nqode-io.github.io/qode> on release publication and on pushes to `main` that touch `site/`

## [0.3.1-beta] - 2026-04-27

### Changed

- `.goreleaser.yml` cosign `signs` block migrated to the Sigstore bundle format — releases now publish a single `checksums.txt.bundle` (signature + certificate combined) instead of the separate `checksums.txt.sig` + `checksums.txt.pem` pair, fixing the `create bundle file: open : no such file or directory` failure introduced when cosign 2.6 made `--new-bundle-format` the default and silently ignored `--output-signature` / `--output-certificate`
- `README.md` supply-chain verification snippet now downloads `checksums.txt.bundle` and verifies with `cosign verify-blob --bundle …`, with a note covering pre-0.3.1-beta releases that still ship the legacy `.sig` + `.pem` artifacts
- `docs/versioning.md` tagged-release pipeline description updated to reflect the bundle-format signing asset

## [0.3.0-beta] - 2026-04-27

### Added

- `CHANGELOG.md` (this file) — Keep a Changelog 1.1.0 format
- `CODE_OF_CONDUCT.md` — Contributor Covenant 2.1
- `docs/tutorial.md` — end-to-end walkthrough covering branch + multi-context workflow, `notes.md` usage, mid-run course-correction, worktrees, and per-IDE best practices
- README trust badges row (CI, Release, Latest Release, Go Report Card, License)
- `docs/qode-yaml-reference.md` field descriptions for `ide.cursor.enabled`, `ide.claude_code.enabled`, `ide.codex.enabled`, and `knowledge.path` so every key in `internal/config/schema.go` is documented
- Code of Conduct link in `README.md` `## Contributing` and at the top of `CONTRIBUTING.md`
- Codex IDE support documented across `README.md`, `CONTRIBUTING.md`, `docs/tutorial.md`, and `docs/qode-yaml-reference.md` after Codex scaffolder landed in [#34](https://github.com/nqode-io/qode/issues/34)
- Generated `qode-note-add` workflow assets for Claude Code, Cursor, and Codex, with free-form note capture into `.qode/contexts/current/notes.md`

### Changed

- `README.md` `## The Workflow` rewritten to mirror `qode workflow` CLI output (12 canonical steps in `internal/cli/help.go:191-232`)
- `README.md` `## IDE Support` rewritten as canonical matrix — Cursor, Claude Code, and Codex, with the slash-command catalog inline
- `CONTRIBUTING.md` development-workflow snippet reconciled to the canonical 12-step list (added `Test locally`, `/qode-pr-create`, `/qode-pr-resolve`)
- `CLAUDE.md` package references updated from the obsolete `branchcontext` to `qodecontext` (renamed in [#33](https://github.com/nqode-io/qode/issues/33)); minimum security-review score aligned with the documented default
- `.github/workflows/ci.yml` now also runs on `push: branches: [main]` so the new CI badge has a status to display on the default branch
- Codex integration now generates explicit-invocation skills under `.agents/skills/` instead of legacy `.codex/commands/`, and the docs/help text now describe cross-IDE workflow invocation accordingly
- Workflow docs and help now present `qode-note-add` as an optional helper with free-form note text rather than a numbered workflow step

### Fixed

- `docs/versioning.md` — corrected the snapshot-version description: snapshots derive the version from the most recent `v*` Git tag plus the GitHub Actions run number, not a hardcoded `0.1.0-alpha+<run>`

## [0.2.1-beta] - 2026-04-27

### Fixed

- Ship qode via a Homebrew formula instead of a cask so `brew install nqode-io/tap/qode` works on Linuxbrew and on macOS without Gatekeeper prompts.

## [0.2.0-beta] - 2026-04-27

### Added

- Initial qode CLI: structured AI-prompt generation for a standardised developer workflow
- Named contexts for parallel work streams ([#33](https://github.com/nqode-io/qode/issues/33))
- MCP-based ticket fetch replacing the built-in HTTP fetcher ([#27](https://github.com/nqode-io/qode/issues/27))
- Strict mode for refined-analysis scoring ([#30](https://github.com/nqode-io/qode/issues/30))
- Configurable scoring rubrics ([#26](https://github.com/nqode-io/qode/issues/26))
- `qode pr create` ([#36](https://github.com/nqode-io/qode/issues/36))
- `/qode-pr-resolve` slash command ([#31](https://github.com/nqode-io/qode/issues/31))
- Split worker/judge two-pass refinement ([#39](https://github.com/nqode-io/qode/issues/39))
- Install scripts (`install.sh`, `install.ps1`) and GoReleaser config ([#37](https://github.com/nqode-io/qode/issues/37))

### Changed

- Simplified `qode init` ([#29](https://github.com/nqode-io/qode/issues/29))
- Replaced branch-based context with VCS-agnostic named contexts ([#33](https://github.com/nqode-io/qode/issues/33))
- Lowered minimum coverage gate from 75% to 70%

### Removed

- `plan status`, `branch list`, `branch focus`, `config show/detect/validate` ([#42](https://github.com/nqode-io/qode/issues/42))
- Unused CLI flags ([#44](https://github.com/nqode-io/qode/issues/44))
- Built-in HTTP ticket fetcher (replaced by MCP, [#27](https://github.com/nqode-io/qode/issues/27))
