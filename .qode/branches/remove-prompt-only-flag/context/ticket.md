# FEAT: Remove --prompt-only flag and make that the default behavior

## Problem / motivation

qode is meant to be used as a support tool in the process of writing code. While some functionalities should be called directly by the user from the terminal, most should only ever be used by AI.

In the beginning, commands like `qode plan refine` and `qode plan spec` were used to generate a prompt file, then drop into Claude Code CLI and execute the prompt. The slash command equivalents, namely `/qode-plan-refine` and `/qode-plan-spec`, instead first called `qode plan refine --prompt-only` (or `qode plan spec --prompt-only`) to generate the corresponding prompt file then executed the prompt directly.

Entering the Claude Code CLI is superfluous as the engineer's choice of IDE should be irrelevant and we also want to be able to support all IDEs and all LLMs.

## Proposed solution

We remove the call to the Claude Code CLI in qode altogether. Instead we are just left with the generated prompt file. As this is equivalent to always calling the command with `--prompt-only` flag this flag should also be removed.

The solution must then:

- Change the behavior of the command called (`qode plan refine`, `qode plan spec`, `qode start`, `qode review code`, `qode review security`, `qode review all` -- last of which should be removed as it is no longer necessary). All other commands that accept `--prompt-only` should be reviewed and a decision per case should be made as to how to deal with them
- Update the slash commands to reflect the removal of the `--prompt-only` flag
- Update all documentation which is applicable which should remove all uses of `--prompt-only` flag but also any mention of the use of affected terminal commands as the only proper way to use the tool from now on is through an IDE

## Alternatives considered

While keeping the current behavior where the prompt is generated and saved to its own prompt file and then executed by the slash command is acceptable, an even better solution would be to output the prompt directly to the `os.Stdout`. This is only applicable if the LLM can be instructed to call a terminal command, read its output and use it as a prompt. In this case adding a new `--to-file` flag would be useful for debugging the prompt templates if changed.

The solution must then:

- Change the behavior of the command called (`qode plan refine`, `qode plan spec`, `qode start`, `qode review code`, `qode review security`, `qode review all` -- the last of which should be removed as it is no longer necessary) to write the generated prompt to `os.Stdout` instead of to a prompt file. All other commands that accept `--prompt-only` should be reviewed and a decision per case should be made as to how to deal with them
- For the same commands add a `--to-file` flag to generate the prompt by applying the data to the template then save it to the prompt file as it does currently and if there are any errors while applying the data to the template any errors should be caught and written out to `os.Stderr` before gracefully terminating the process.
- Update the slash commands to reflect the removal of the `--prompt-only` flag
- Update all documentation which is applicable which should remove all uses of `--prompt-only` flag but also any mention of the use of affected terminal commands as the only proper way to use the tool from now on is through an IDE, and addition of the `--to-file` flag.

## Additional context

Since this is already a big refactor, review any dead code which may arise afterwards and decide the proper course of action.

Ensure there is a way that the slash commands or their equivalents for all IDEs is up to date as well as appropriate documentation. For now handle this change in such a way that all the relevant ide specific files are overwritten in both `qode ide setup` and `qode ide sync`, but leave a TODO comment to ensure proper behavior after we go into beta to implement a `--force` flag to explicitly allow overwriting those files.

When updating documentation consider the `.md` files in the `/` and the `doc/` directories, but also the inline docummentation shouch as the one produced by the `qode workflow` command or via the `--help` flag.

Review if any of the lessons learned in this repository need to be updated or removed.

Commands for:

- `qode init` with applicable flags
- `qode branch` and sub-commands
- `qode ticket fetch`
- `qode plan status`
- `qode check` with applicable flags
- `qode knowledge` and sub-commands
- `qode config` and sub-commands

should not be changed.
