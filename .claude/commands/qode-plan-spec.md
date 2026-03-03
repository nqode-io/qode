# Generate Technical Specification — qode

Read and execute the prompt in:
  .qode/branches/$BRANCH/.spec-prompt.md

Where $BRANCH is the current git branch name.

If the file does not exist, tell the user to run: qode plan spec

After generating the spec:
- Save it to: .qode/branches/$BRANCH/spec.md
- Suggest copying it to the ticket system for team review
