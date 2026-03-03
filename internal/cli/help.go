package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHelpWorkflowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "workflow",
		Short: "Show the full qode workflow diagram",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(workflowDiagram)
		},
	}
}

const workflowDiagram = `
qode Workflow
=============

┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: CREATE BRANCH                                          │
│  qode branch create feat-user-dashboard                         │
│  → Creates git branch + .qode/branches/feat-user-dashboard/     │
├─────────────────────────────────────────────────────────────────┤
│  STEP 2: ADD CONTEXT                                            │
│  qode ticket fetch <url>                                        │
│  → Auto-fetches ticket into context/ticket.md                   │
│  → Or manually edit .qode/branches/.../context/ticket.md        │
│  → Add mockups: cp design.png .qode/branches/.../context/       │
├─────────────────────────────────────────────────────────────────┤
│  STEP 3: REFINE REQUIREMENTS  (target: 25/25, ~3-5 iterations)  │
│  /qode-plan-refine  (in Cursor/Claude Code)                     │
│  → AI reads context + researches codebase                       │
│  → Judge independently scores 5 dimensions × 5 points          │
│  → Iterate: answer open questions, re-run until 25/25           │
│  → Check status: qode plan status                               │
├─────────────────────────────────────────────────────────────────┤
│  STEP 4: GENERATE SPEC                                          │
│  /qode-plan-spec  (in Cursor/Claude Code)                       │
│  → Creates spec.md from refined analysis                        │
│  → Tip: copy spec back to Jira/Azure DevOps ticket              │
├─────────────────────────────────────────────────────────────────┤
│  STEP 5: IMPLEMENT                                              │
│  qode start                                                     │
│  → Generates implementation prompt with spec + knowledge        │
│  → Paste into Cursor/Claude Code; AI writes baseline code       │
├─────────────────────────────────────────────────────────────────┤
│  STEP 6: TEST LOCALLY                                           │
│  → Test the implementation manually                             │
│  → Use Cursor Chat / Claude Code chat for tweaks                │
├─────────────────────────────────────────────────────────────────┤
│  STEP 7: REVIEW + QUALITY GATES                                 │
│  /qode-review-code       (in Cursor/Claude Code)                │
│  /qode-review-security   (in Cursor/Claude Code)                │
│  — or —                                                         │
│  qode check    (runs tests + lint + both reviews per layer)     │
│  → Apply suggested fixes; re-run until all gates pass           │
├─────────────────────────────────────────────────────────────────┤
│  STEP 8: SHIP                                                   │
│  git add . && git commit && git push                            │
│  gh pr create  (or az repos pr create)                          │
├─────────────────────────────────────────────────────────────────┤
│  STEP 9: CLEANUP                                                │
│  qode branch remove feat-user-dashboard                         │
└─────────────────────────────────────────────────────────────────┘

Scoring Rubric (refinement, 5 × 5 = 25 points):
  1. Problem Understanding      (0-5)
  2. Technical Analysis         (0-5)
  3. Risk & Edge Cases          (0-5)
  4. Completeness               (0-5)
  5. Actionability              (0-5)

Review Scoring (10-point scale):
  Code Review:     minimum 8.0/10 (configurable via review.min_code_score)
  Security Review: minimum 8.0/10 (configurable via review.min_security_score)

`
