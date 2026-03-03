# Code Review — qode

First, run this command to generate the prompt:
  qode review code --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.code-review-prompt.md

Where $BRANCH is the current git branch name.

After completing the review:
- Save to: .qode/branches/$BRANCH/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
