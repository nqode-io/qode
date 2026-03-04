# FEAT: Create /qode-start slash command

Currently the user needs to call the `qode start` from the terminal which should not be the case.

Also, current behavior of `qode start` is such that it creates the prompt file but does nothing more. It of course should behave just like other commands.

In addition to this, after it writes the prompt file it notifies the user to `Paste into Cursor/Claude Code, or use: qode start --open` but this flag does not and should not exist.

Make sure all IDE's have equivalent behavior, not just claude code.
