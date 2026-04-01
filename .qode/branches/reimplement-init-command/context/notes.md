# Notes

Workspace configuration should be removed as well as the possibility to init a new repo or a workspace. Init is init, no need to complicate things.

## Open questions -- Answers

- Project name in IDE generators: derive via `filepath.Base(root)` at call time; no config field. **Answer**: This is correct.
- Templates that render project name: replace `{{.Project.Name}}` with `**Project:** (derived from directory)` text or remove it entirely — removing the `## Project Context` block is cleaner because the project name adds no value to the AI reading the prompt. **Answer**: Considering the previous, keep the `{{.Project.Name}}` and populate it by `filepath.Base(root)`. This should be done through templates and template variables because this will save the AI from running a tool to find the project name.

## Decisions

- **Merge `qode init` and `qode ide setup` into a single command**: The user should not have to run both `qode init` and `qode ide setup` to start using qode. `qode init` should do everything: generate the minimal `qode.yaml` with defaults, create the `.qode/` directories if they do not exist, copy the embedded prompt templates, and generate IDE configs and commands. `qode ide setup` (and `qode ide sync`) can remain as a separate command for regenerating IDE configs after manual edits to `qode.yaml`, but it is no longer a required step after `qode init`.

- **Omit `.cursorrules/` generation entirely**: The `.cursorrules/` directory serves the same purpose as `CLAUDE.md` — project-level AI assistant instructions. This is the user's domain to author and maintain. `qode init` (and `qode ide setup`) must not create or modify any files under `.cursorrules/`. Only `.cursor/commands/` (slash commands) is written by qode.

- **Update all documentation** to reflect the merged init flow and the removal of `.cursorrules/` generation. The README, `docs/qode-yaml-reference.md`, and any other docs referencing the two-step `qode init` + `qode ide setup` flow must be updated.

- **Remove `qode ide` entirely**: `qode ide setup` and `qode ide sync` are removed. There is no separate `qode ide` subcommand. IDE configs are generated exclusively by `qode init`. Delete `internal/cli/ide.go` and all associated tests.

- **Rename `internal/ide` to `internal/scaffold`**: The `ide` package was named after the `qode ide` CLI subcommand, which has been removed. The package's actual responsibility is generating IDE config files during project initialisation (a scaffolding concern), not exposing an IDE-facing interface. `internal/scaffold` matches the domain (`scaffold` is the standard term in developer tooling for generating project file structure), is forward-compatible if `qode init` ever scaffolds more than IDE files, and follows the existing convention of naming packages after their domain concern rather than the CLI command they serve.

- **Extract scoring rubrics to `.qode/scoring.yaml`**: The `scoring.rubrics` section is removed from `qode.yaml` and written instead to `.qode/scoring.yaml`. `qode init` writes `.qode/scoring.yaml` with defaults only when the file does not already exist, so re-runs never overwrite user-customised rubrics. `config.Load` merges `.qode/scoring.yaml` after `qode.yaml` so custom rubrics take priority over defaults. This resolves H1 (silent overwrite of user scoring config on re-run). `qode.yaml` itself is also only written on first run (when the file is absent) — re-runs preserve all other user customisations there too. H2 (IDE preferences ignored on re-run) is fixed by calling `config.Load(root)` at the top of `runInitExisting` instead of always constructing `DefaultConfig()`, so the loaded `ide.cursor.enabled`/`ide.claude_code.enabled` values are passed to `ide.Setup`.
