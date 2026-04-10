# PR Creation — qode

Run this command and use its stdout output as your prompt:
  qode pr create

If the output begins with `STOP.`, report the prerequisite message to the user and wait for instructions.

Use `--base <branch>` to override the auto-detected base branch:
  qode pr create --base develop

After generating the PR:
1. Check via MCP whether a PR already exists for the current branch. If one exists, report its URL and stop.
2. Generate a PR title and description using the context in the prompt.
3. Create the PR via MCP.
4. Save the PR URL to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/context/pr-url.txt
