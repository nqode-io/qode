# Code Review — qode

Read and execute the prompt in:
  .qode/branches/$BRANCH/.code-review-prompt.md

Where $BRANCH is the current git branch name.

If the file does not exist, tell the user to run: qode review code

After completing the review:
- Save to: .qode/branches/$BRANCH/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
