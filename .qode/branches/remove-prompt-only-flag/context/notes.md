# Notes

## Open questions with answers
- `knowledge add-branch` has `--prompt-only` but is listed among commands that "should not be changed" in Additional Context. However it IS in `knowledge_cmd.go` which is not `knowledge` or `config` — need to confirm: does "knowledge and sub-commands should not be changed" cover `knowledge add-branch`? The ticket lists `qode knowledge` under "should not be changed", so `knowledge add-branch` should be preserved as-is. **Answer**: this should be changed, you're right
- `qode review all` is to be removed — confirm it is not referenced anywhere externally (docs, CI, user scripts) before deletion. **Answer**: completly agree
- The TODO comment about a future `--force` flag for ide setup/sync overwrite behavior needs a precise location to be useful. **Answer**: internal/cli/ide.go would be a perfect place for this because it already handles both `setup` and `sync` subcommands


## Additional comments
- support for VSCode IDE can be dropped also now that qode subcommands don't enter the interactive claude code CLI
- security review found one low level issue, but it was manually fixed