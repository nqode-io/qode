# Roadmap

Planned features for qode, in recommended implementation order. Items marked with the same step number can be worked on in parallel.

## Done

- [x] [#24](https://github.com/nqode-io/qode/issues/24) — **Harden review prompts** — Enforcement mechanisms for code and security reviews, slim IDE commands
- [x] [#25](https://github.com/nqode-io/qode/issues/25) — **Optimize prompts for token usage** — Reference files instead of inlining previous-step outputs into templates
- [x] [#39](https://github.com/nqode-io/qode/issues/39) — **Split `qode plan refine` into `qode plan refine` + `qode plan judge`** — Judge pass is a dedicated subcommand; worker prompt no longer generates the judge file
- [x] [#42](https://github.com/nqode-io/qode/issues/42) — **Remove unused commands** — Removed `plan status`, `branch list`, `branch focus`, `config show/detect/validate`
- [x] [#44](https://github.com/nqode-io/qode/issues/44) — **Remove unused CLI flags** — Removed `--verbose`, `--base`, `--keep-branch`, `--skip-tests`, `--layer`, `--new`, `--scaffold`, `--workspace`, `--ide`, `--cursor`, `--claude`, `--iterations`
- [x] [#45](https://github.com/nqode-io/qode/issues/45) — **Replace `qode check` with `/qode-check` IDE slash command** — AI-driven interactive quality gate; detects test runner and linter from project structure

## In progress

## Up Next (parallel — start after #24)

- [ ] [#26](https://github.com/nqode-io/qode/issues/26) — **Configurable scoring rubrics** — Extract hardcoded rubrics into `qode.yaml`, support custom dimensions and weights
- [ ] [#27](https://github.com/nqode-io/qode/issues/27) — **Replace ticket fetch with MCP** — Use MCP servers instead of built-in HTTP clients; support comments, attachments, linked resources
- [ ] [#29](https://github.com/nqode-io/qode/issues/29) — **Rethink qode init** — Simplify setup; let the AI read project configs instead of hardcoding test/lint/build commands
- [ ] [#41](https://github.com/nqode-io/qode/issues/41) — **`qode init`: append gitignore rules** — Add qode-specific `.gitignore` entries (temp prompt files, ticket snapshots, scored iteration copies) during init
- [ ] [#33](https://github.com/nqode-io/qode/issues/33) — **Worktree support** — Config flag `branch.use_worktrees` to create git worktrees via worktrunk, phantom, or `git worktree`; enables parallel task development
- [ ] [#34](https://github.com/nqode-io/qode/issues/34) — **Add Codex IDE support** — Slash commands, IDE setup, templates, and documentation for OpenAI Codex, following the same convention as Cursor and Claude Code
- [ ] [#35](https://github.com/nqode-io/qode/issues/35) — **Auto-commit after completed tasks** — Config flag `workflow.auto_commit`; instructs AI to commit after each task in `qode start`, after review fixes, and after PR comment resolution

## After Dependencies

- [ ] [#28](https://github.com/nqode-io/qode/issues/28) — **Post step outputs as ticket comments** — Publish analysis, spec, and review outputs to the original ticket via MCP *(requires #27)*
- [ ] [#30](https://github.com/nqode-io/qode/issues/30) — **Strict mode** — Block workflow steps when prerequisites are missing or scores are below configured minimums *(requires #26)*
- [ ] [#36](https://github.com/nqode-io/qode/issues/36) — **Add qode pr create command** — Generate PR/MR with AI-written title and description from branch context; store PR URL for subsequent steps *(requires #27)*
- [ ] [#31](https://github.com/nqode-io/qode/issues/31) — **PR/MR review comments step** — Read and address PR review comments using MCP *(requires #27, #36)*

## Release Preparation (after all features above)

- [ ] [#37](https://github.com/nqode-io/qode/issues/37) — **Prepare qode for public beta release** — Documentation review, README badges, install script, binary signing, GitHub Pages site, GoReleaser setup, version bump to beta, automatic release notes

## Dependency Graph

```
#24 Harden review prompts ✅
 ├── #25 Optimize prompts for token usage ✅
 ├── #26 Configurable scoring rubrics
 │    └── #30 Strict mode
 ├── #27 Replace ticket fetch with MCP
 │    ├── #28 Post step outputs as ticket comments
 │    ├── #36 qode pr create
 │         └── #31 PR/MR review comments step
 └── #29 Rethink qode init

Independent (can run in parallel with any of the above):
 #33 Worktree support
 #34 Codex IDE support
 #35 Auto-commit after completed tasks
 #41 qode init: append gitignore rules
 #45 Replace qode check with /qode-check  ✅

All of the above → #37 Prepare for public beta release
```

---

## Long-term Goals

These are future product directions beyond the current software engineer workflow scope. They are tracked here for planning purposes and will be broken into issues when the time comes.

### qode-qa — QA Engineer workflow

A companion tool for QA engineers, built on the same conventions as qode:

- Generate test cases (manual and automated) from tickets, specs, and code
- Optionally automate test execution where tooling allows (Playwright, Cypress, k6, etc.)
- Run test suites without human involvement and report results
- File bug reports directly in the ticketing system (Jira, Linear, GitHub Issues, etc.) with steps to reproduce, environment info, and severity
- Generate structured test reports for stakeholders

### qode-devops — DevOps / Platform Engineer workflow

A companion tool for DevOps and platform engineers:

- Generate infrastructure-as-code scaffolding (Terraform, Helm, Docker Compose) from descriptions
- Review infrastructure changes for security and best-practice violations
- Automate deployment pipeline setup for common stacks
- Integrate with incident management tools to create and triage incidents
- Generate runbooks and postmortems from incident context

### qode-po — Product Owner workflow

A companion tool for Product Owners and product managers:

- Expand brief feature descriptions into detailed, development-ready tickets
- Support attaching screenshots, mockups, Figma links, and design documents to tickets
- Validate tickets for completeness against a configurable definition-of-ready rubric
- Generate acceptance criteria from user stories
- Create epics, break them into stories, and maintain backlog structure in the ticketing system
