// Package workflow implements step-ordering enforcement for the qode pipeline.
package workflow

import (
	"fmt"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/scoring"
)

// CheckResult is the output of CheckStep.
type CheckResult struct {
	Blocked bool
	Message string // actionable message including the slash command to run next
}

// CheckStep returns a CheckResult indicating whether the prerequisites for step
// are satisfied. It is pure — no side effects or I/O.
//
// Guarded steps: "spec", "start".
// Review steps ("review-code", "review-security") use the diff-empty check in
// runReview instead of this function. Unknown steps are treated as unblocked.
func CheckStep(step string, ctx *branchcontext.Context, cfg *config.Config) CheckResult {
	switch step {
	case "spec":
		return checkSpec(ctx, cfg)
	case "start":
		return checkStart(ctx)
	}
	return CheckResult{}
}

func checkSpec(ctx *branchcontext.Context, cfg *config.Config) CheckResult {
	if !ctx.HasRefinedAnalysis() {
		return CheckResult{
			Blocked: true,
			Message: "No refined-analysis.md found. Run /qode-plan-refine first.",
		}
	}
	if ctx.LatestScore() == 0 {
		return CheckResult{
			Blocked: true,
			Message: "refined-analysis.md is unscored. Run /qode-plan-judge first.",
		}
	}
	min := RefineMinScore(cfg)
	if score := ctx.LatestScore(); score < min {
		maxScore := scoring.BuildRubric(scoring.RubricRefine, cfg).Total()
		return CheckResult{
			Blocked: true,
			Message: fmt.Sprintf(
				"Refine score is %d/%d, minimum required is %d. Run /qode-plan-refine to improve.",
				score, maxScore, min,
			),
		}
	}
	return CheckResult{}
}

func checkStart(ctx *branchcontext.Context) CheckResult {
	if !ctx.HasSpec() {
		return CheckResult{
			Blocked: true,
			Message: "No spec.md found. Run /qode-plan-spec first.",
		}
	}
	return CheckResult{}
}

// RefineMinScore returns the configured minimum refinement score.
// Falls back to the rubric total when target_score is not set.
func RefineMinScore(cfg *config.Config) int {
	if cfg != nil && cfg.Scoring.TargetScore > 0 {
		return cfg.Scoring.TargetScore
	}
	return scoring.BuildRubric(scoring.RubricRefine, cfg).Total()
}

// RefineMaxScore returns the total points available for the refine rubric.
func RefineMaxScore(cfg *config.Config) int {
	return scoring.BuildRubric(scoring.RubricRefine, cfg).Total()
}
