# Refine Requirements — qode

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine

Save the worker output to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/refined-analysis.md

**Judge pass (scoring):** Run this command and use its stdout output as your prompt:
  qode plan judge

Then:
1. Parse the "**Total Score:** S/M" line and the "**Pass threshold:** T/M" line from the judge output to get score S, max M, and pass threshold T
2. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
3. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/M -->
4. Write a copy to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/refined-analysis-N-score-S.md
5. Report the score to the user. If S >= T, suggest running /qode-plan-spec. Otherwise suggest re-running /qode-plan-refine.
