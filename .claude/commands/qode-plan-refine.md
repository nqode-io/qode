# Refine Requirements — qode

Read and execute the prompt in:
  .qode/branches/$BRANCH/.refine-prompt.md

Where $BRANCH is the current git branch name.

If the file does not exist, tell the user to run: qode plan refine [ticket-url]

After completing the analysis:
- Save the output to: .qode/branches/$BRANCH/refined-analysis.md
- Tell the user the analysis is complete and to run qode plan refine again for judge scoring
