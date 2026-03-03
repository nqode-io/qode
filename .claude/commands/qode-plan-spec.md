# Generate Technical Specification — qode

First, run this command to generate the prompt:
  qode plan spec --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.spec-prompt.md

Where $BRANCH is the current git branch name.

After generating the spec:
- Save it to: .qode/branches/$BRANCH/spec.md
- Suggest copying it to the ticket system for team review
