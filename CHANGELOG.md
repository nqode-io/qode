# Changelog

All notable changes to qode are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `CHANGELOG.md` (this file) — Keep a Changelog 1.1.0 format
- `CODE_OF_CONDUCT.md` — Contributor Covenant 2.1
- `docs/tutorial.md` — end-to-end walkthrough covering branch + multi-context workflow, `notes.md` usage, mid-run course-correction, worktrees, and per-IDE best practices
- README trust badges row (CI, Release, Latest Release, Go Report Card, License)
- `docs/qode-yaml-reference.md` field descriptions for `ide.cursor.enabled`, `ide.claude_code.enabled`, `ide.codex.enabled`, and `knowledge.path` so every key in `internal/config/schema.go` is documented
- Code of Conduct link in `README.md` `## Contributing` and at the top of `CONTRIBUTING.md`
- Codex IDE support documented across `README.md`, `CONTRIBUTING.md`, `docs/tutorial.md`, and `docs/qode-yaml-reference.md` after Codex scaffolder landed in [#34](https://github.com/nqode-io/qode/issues/34)

### Changed

- `README.md` `## The Workflow` rewritten to mirror `qode workflow` CLI output (12 canonical steps in `internal/cli/help.go:191-232`)
- `README.md` `## IDE Support` rewritten as canonical matrix — Cursor, Claude Code, and Codex, with the slash-command catalog inline
- `CONTRIBUTING.md` development-workflow snippet reconciled to the canonical 12-step list (added `Test locally`, `/qode-pr-create`, `/qode-pr-resolve`)
- `CLAUDE.md` package references updated from the obsolete `branchcontext` to `qodecontext` (renamed in [#33](https://github.com/nqode-io/qode/issues/33)); minimum security-review score aligned with the documented default
- `.github/workflows/ci.yml` now also runs on `push: branches: [main]` so the new CI badge has a status to display on the default branch
- Codex integration now generates explicit-invocation skills under `.agents/skills/` instead of legacy `.codex/commands/`, and the docs/help text now describe cross-IDE workflow invocation accordingly

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
