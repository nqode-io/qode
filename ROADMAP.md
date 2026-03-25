# Roadmap

Planned features for qode, in recommended implementation order. Items marked with the same step number can be worked on in parallel.

## In Progress

- [ ] [#24](https://github.com/nqode-io/qode/issues/24) — **Harden review prompts** — Enforcement mechanisms for code and security reviews, slim IDE commands

## Up Next (parallel — start after #24)

- [ ] [#25](https://github.com/nqode-io/qode/issues/25) — **Optimize prompts for token usage** — Reference files instead of inlining previous-step outputs into templates
- [ ] [#26](https://github.com/nqode-io/qode/issues/26) — **Configurable scoring rubrics** — Extract hardcoded rubrics into `qode.yaml`, support custom dimensions and weights
- [ ] [#27](https://github.com/nqode-io/qode/issues/27) — **Replace ticket fetch with MCP** — Use MCP servers instead of built-in HTTP clients; support comments, attachments, linked resources
- [ ] [#29](https://github.com/nqode-io/qode/issues/29) — **Rethink qode init** — Simplify setup; let the AI read project configs instead of hardcoding test/lint/build commands

## After Dependencies

- [ ] [#28](https://github.com/nqode-io/qode/issues/28) — **Post step outputs as ticket comments** — Publish analysis, spec, and review outputs to the original ticket via MCP *(requires #27)*
- [ ] [#30](https://github.com/nqode-io/qode/issues/30) — **Strict mode** — Block workflow steps when prerequisites are missing or scores are below configured minimums *(requires #26)*
- [ ] [#31](https://github.com/nqode-io/qode/issues/31) — **PR/MR review comments step** — Read and address PR review comments using MCP *(requires #27)*

## Dependency Graph

```
#24 Harden review prompts
 ├── #25 Optimize prompts for token usage
 ├── #26 Configurable scoring rubrics
 │    └── #30 Strict mode
 ├── #27 Replace ticket fetch with MCP
 │    ├── #28 Post step outputs as ticket comments
 │    └── #31 PR/MR review comments step
 └── #29 Rethink qode init
```
