# Notes

The following gitignore rules must be added to the `.gitignore` file:

```text
# qode temp files
.qode/branches/*/.*.md
.qode/branches/*/context/ticket.md
.qode/branches/*/context/ticket-comments.md
.qode/branches/*/context/ticket-links.md
.qode/branches/*/refined-analysis-*-score-*.md
.qode/branches/*/diff.md
.qode/prompts/scaffold/
```

Each one must be checked individually and added only if not present, ideally under that same content heading. The implementation should be such that all of these are added to a config file or template and then used by the `qode init` command from there.

Open questions - Answers

Q1: Should the `.qode/prompts/` directory be added to `.gitignore`? **Answer**: No.
Q2: Should workspace topology be handled (running `qode init` outside a git repo) in any special way? **Answer**: No.
