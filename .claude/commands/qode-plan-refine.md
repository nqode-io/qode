# Refine Requirements — qode

First, run this command to generate the prompt:
  qode plan refine --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.refine-prompt.md

Where $BRANCH is the current git branch name.

After completing the analysis:
- Save the output to: .qode/branches/$BRANCH/refined-analysis.md
- Tell the user the analysis is complete and to run qode plan refine again for judge scoring
