### Review prompt generator may produce partial diffs
When running `qode review code --prompt-only` or `qode review security --prompt-only`, the generated prompt file may contain only a partial diff (e.g., only the last commit rather than the full branch diff). When performing code or security reviews, always verify the diff scope by running `git diff main --stat` separately to understand the full changeset, then read all changed files directly rather than relying solely on the diff in the generated prompt.

**Example 1:** Verify full scope before reviewing
```bash
git diff main --stat           # see all changed files
git diff main -- path/to/file  # read full diff per file
```