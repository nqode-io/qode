package cli

import (
	"fmt"
	"io"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/qodecontext"
	"github.com/nqode/qode/internal/scoring"
	"github.com/nqode/qode/internal/workflow"
	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Show or inspect the qode workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(cmd.OutOrStdout())
		},
	}
	cmd.AddCommand(newWorkflowShowCmd(), newWorkflowStatusCmd())
	return cmd
}

func runWorkflow(out io.Writer) error {
	_, _ = fmt.Fprint(out, workflowList)
	return nil
}

func newWorkflowShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the full qode workflow steps",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), workflowList)
		},
	}
}

func newWorkflowStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show live completion status for each workflow step",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowStatus(cmd.OutOrStdout())
		},
	}
}

func runWorkflowStatus(out io.Writer) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}

	diff := runDiffCommand(sess.Root, sess.Config.Diff.Command)

	lines, upNext := buildStatusLines(sess.Context, sess.Config, diff)
	for _, line := range lines {
		_, _ = fmt.Fprintln(out, line)
	}
	if upNext != "" {
		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "Up next:", upNext)
	}
	return nil
}

func buildStatusLines(ctx *qodecontext.Context, cfg *config.Config, diff string) (lines []string, upNext string) {
	step := func(n int, label, status string) string {
		return fmt.Sprintf("%d. %s - %s", n, label, status)
	}

	// Step 1: always complete once the context exists.
	lines = append(lines, step(1, "Create context", "Completed."))

	// Step 2: Add context.
	if ctx.Ticket != "" {
		lines = append(lines, step(2, "Add context", "Completed."))
	} else {
		lines = append(lines, step(2, "Add context", "Not started."))
		if upNext == "" {
			upNext = "Run the `qode-ticket-fetch` step with the ticket URL."
		}
	}

	// Step 3: Refine requirements.
	lines = append(lines, step(3, "Refine requirements", refineStatus(ctx, cfg, &upNext)))

	// Step 4: Generate spec.
	if ctx.HasSpec() {
		lines = append(lines, step(4, "Generate spec", "Completed."))
	} else {
		lines = append(lines, step(4, "Generate spec", "Not started."))
		if upNext == "" {
			upNext = "Run the `qode-plan-spec` step."
		}
	}

	// Step 5: Implement.
	if diff != "" {
		lines = append(lines, step(5, "Implement", "Completed."))
	} else {
		lines = append(lines, step(5, "Implement", "Not started."))
		if upNext == "" {
			upNext = "Run the `qode-start` step to generate the implementation prompt."
		}
	}

	// Steps 6–7: always manual.
	lines = append(lines, step(6, "Test locally", "Always done by the user."))
	lines = append(lines, step(7, "Quality gates", "Always done by the user."))

	// Step 8: Reviews.
	lines = append(lines, reviewStatus(ctx, cfg, &upNext)...)

	// Step 9: Create pull request — always manual.
	lines = append(lines, step(9, "Create pull request", "Always done by the user — run the `qode-pr-create` step."))

	// Step 10: Resolve PR review comments — always manual.
	lines = append(lines, step(10, "Resolve PR review comments", "Always done by the user — run the `qode-pr-resolve` step."))

	// Step 11: Lessons learned — always optional.
	lines = append(lines, step(11, "Capture lessons learned", "Always optional — run the `qode-knowledge-add-context` step."))

	return lines, upNext
}

func refineStatus(ctx *qodecontext.Context, cfg *config.Config, upNext *string) string {
	if !ctx.HasRefinedAnalysis() {
		if *upNext == "" {
			*upNext = "Run the `qode-plan-refine` step."
		}
		return "Not started."
	}
	n := len(ctx.Iterations)
	score := ctx.LatestScore()
	if score == 0 {
		if *upNext == "" {
			*upNext = "Run the `qode-plan-judge` step to score the analysis."
		}
		return fmt.Sprintf("%d iteration(s), unscored — run the `qode-plan-judge` step.", n)
	}
	maxScore := workflow.RefineMaxScore(cfg)
	minScore := workflow.RefineMinScore(cfg)
	if score < minScore {
		if *upNext == "" {
			*upNext = fmt.Sprintf("Score %d/%d is below minimum %d. Run the `qode-plan-refine` step to improve it.", score, maxScore, minScore)
		}
		return fmt.Sprintf("%d iteration(s), latest score: %d/%d (below minimum %d).", n, score, maxScore, minScore)
	}
	return fmt.Sprintf("%d iteration(s), latest score: %d/%d.", n, score, maxScore)
}

func reviewStatus(ctx *qodecontext.Context, cfg *config.Config, upNext *string) []string {
	var lines []string
	codeMax := scoring.BuildRubric(scoring.RubricReview, cfg).Total()
	secMax := scoring.BuildRubric(scoring.RubricSecurity, cfg).Total()
	codeStatus := reviewItemStatus(
		ctx.HasCodeReview(), ctx.CodeReviewScore(), cfg.Review.MinCodeScore,
		"qode-review-code", codeMax, upNext,
	)
	secStatus := reviewItemStatus(
		ctx.HasSecurityReview(), ctx.SecurityReviewScore(), cfg.Review.MinSecurityScore,
		"qode-review-security", secMax, upNext,
	)
	lines = append(lines,
		fmt.Sprintf("8. Review - Code review: %s", codeStatus),
		fmt.Sprintf("   Security review: %s", secStatus),
	)
	return lines
}

func reviewItemStatus(present bool, score, min float64, cmd string, maxScore int, upNext *string) string {
	if !present {
		if *upNext == "" {
			*upNext = fmt.Sprintf("Run the `%s` step.", cmd)
		}
		return "Not started."
	}
	if score < min {
		if *upNext == "" {
			*upNext = fmt.Sprintf("Score %.1f/%d is below minimum %.1f. Consider fixing issues and re-running the `%s` step.", score, maxScore, min, cmd)
		}
		return fmt.Sprintf("Score %.1f/%d (below minimum %.1f).", score, maxScore, min)
	}
	return fmt.Sprintf("Passed with score %.1f/%d.", score, maxScore)
}

const workflowList = `qode Workflow
=============

IDE invocation surface:
  Cursor / Claude Code: /qode-*
  Codex:                $qode-*  (skills generated under .agents/skills/)

1.  Manually create a branch, then initialise the qode context
    qode context init <name>

2.  Add context
    qode-ticket-fetch <url>  (in IDE)
    Optional helper: qode-note-add  (in IDE, followed by free-form notes)

3.  Refine requirements  (iterate until pass threshold)
    qode-plan-refine   — worker and scoring pass

4.  Generate spec
    qode-plan-spec

5.  Implement
    qode-start

6.  Test locally
    (manual)

7.  Quality gates
    qode-check

8.  Review
    qode-review-code
    qode-review-security

9.  Create pull request
    qode-pr-create

10. Resolve PR review comments
    qode-pr-resolve

11. Capture lessons learned
    qode-knowledge-add-context  (optional)

12. Cleanup
    qode context remove

Run 'qode workflow status' to see live completion status for the current context.
`
