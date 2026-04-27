# Changelog

All notable changes to qode are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `CHANGELOG.md` (this file)
- `CODE_OF_CONDUCT.md`
- README trust badges (CI, Release, Latest Release, Go Report Card, License)

### Changed

- README "The Workflow" block now mirrors `qode workflow` CLI output (12 canonical steps)
- README "IDE Support" rewritten as canonical matrix (Cursor + Claude Code)

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
