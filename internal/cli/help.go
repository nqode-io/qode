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
│  /qode-ticket-fetch <url>  (in Cursor/Claude Code)              │
│  → Auto-fetches ticket into context/ticket.md                   │
│  → Or manually edit .qode/branches/.../context/ticket.md        │
│  → Add mockups: cp design.png .qode/branches/.../context/       │
├─────────────────────────────────────────────────────────────────┤
│  STEP 3: REFINE REQUIREMENTS  (iterate to pass threshold)       │
│  /qode-plan-refine  (in Cursor/Claude Code)                     │
│  → AI reads context + researches codebase                       │
│  → Judge independently scores each configured dimension         │
│  → Iterate: answer open questions, re-run until pass threshold  │
├─────────────────────────────────────────────────────────────────┤
│  STEP 4: GENERATE SPEC                                          │
│  /qode-plan-spec  (in Cursor/Claude Code)                       │
│  → Creates spec.md from refined analysis                        │
│  → Tip: copy spec back to Jira/Azure DevOps ticket              │
├─────────────────────────────────────────────────────────────────┤
│  STEP 5: IMPLEMENT                                              │
│  /qode-start  (in Cursor/Claude Code)                           │
│  → Generates implementation prompt with spec + knowledge        │
├─────────────────────────────────────────────────────────────────┤
│  STEP 6: TEST LOCALLY                                           │
│  → Test the implementation manually                             │
│  → Use Cursor Chat / Claude Code chat for tweaks                │
├─────────────────────────────────────────────────────────────────┤
│  STEP 7: QUALITY GATES                                          │
│  /qode-check  (in Cursor/Claude Code)                           │
│  → AI detects test runner + linter from project structure       │
│  → Runs tests, then lint; proposes fixes on failure             │
├─────────────────────────────────────────────────────────────────┤
│  STEP 8: REVIEW                                                 │
│  /qode-review-code       (in Cursor/Claude Code)                │
│  /qode-review-security   (in Cursor/Claude Code)                │
│  → Apply suggested fixes; re-run until all gates pass           │
├─────────────────────────────────────────────────────────────────┤
│  STEP 9: CAPTURE LESSONS LEARNED                                │
│  /qode-knowledge-add-context (in Cursor/Claude Code)            │
│  → Capture insights and best practices from context             │
├─────────────────────────────────────────────────────────────────┤
│  STEP 10: SHIP                                                  │
│  git add . && git commit && git push                            │
│  gh pr create  (or az repos pr create)                          │
├─────────────────────────────────────────────────────────────────┤
│  STEP 11: CLEANUP                                               │
│  qode branch remove feat-user-dashboard                         │
└─────────────────────────────────────────────────────────────────┘

Scoring Rubric (refinement, default: 5 × 5 = 25 points):
  Default dimensions: Problem Understanding, Technical Analysis,
                      Risk & Edge Cases, Completeness, Actionability
  Configurable via scoring.rubrics.refine in qode.yaml

Review Scoring (default scales, configurable via scoring.rubrics):
  Code Review:     minimum 10.0/12 (configurable via review.min_code_score)
  Security Review: minimum 8.0/10  (configurable via review.min_security_score)

`
