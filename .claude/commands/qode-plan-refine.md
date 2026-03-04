# Refine Requirements — qode

First, run this command to generate the prompts:
  qode plan refine --prompt-only

Where $BRANCH is the current git branch name.

**Worker pass:** Read and execute the worker prompt in:
  .qode/branches/$BRANCH/.refine-prompt.md

Save the worker output to:
  .qode/branches/$BRANCH/refined-analysis.md

**Judge pass (scoring):**
1. Read .qode/branches/$BRANCH/.refine-judge-prompt.md
2. Replace the placeholder line "[Worker output will be pasted here by the user after running the worker prompt above]" with the full content of refined-analysis.md
3. Execute the modified judge prompt
4. Parse the "**Total Score:** N/25" line from the judge output
5. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
6. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/25 -->
7. Write a copy to: .qode/branches/$BRANCH/refined-analysis-N-score-S.md
8. Report the score to the user. If S >= 25, suggest running "qode plan spec". Otherwise suggest re-running /qode-plan-refine.
