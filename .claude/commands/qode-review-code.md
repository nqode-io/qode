# Code Review — qode

Run this command and use its stdout output as your prompt:
  qode review code

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `qode review code --force` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/contexts/current/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
