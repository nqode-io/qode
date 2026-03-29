package cli

import (
	"fmt"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/runner"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run all quality gates (test → lint → code review → security review)",
		Long: `Runs quality gates for every configured layer.

Per layer:
  1. Unit tests
  2. Lint
  3. Code review (AI, two-pass scoring)
  4. Security review (AI, two-pass scoring)

Fails if any mandatory gate does not meet the configured minimum score.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}

			branch, err := git.CurrentBranch(root)
			if err != nil {
				return err
			}

			results := runner.RunCheck(root, branch, cfg, cfg.Layers())

			printCheckResults(results, cfg)

			for _, r := range results {
				if !r.Passed {
					return fmt.Errorf("quality gates failed")
				}
			}
			return nil
		},
	}

	return cmd
}

func printCheckResults(results []runner.LayerResult, cfg *config.Config) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║           QUALITY GATES CHECK             ║")
	fmt.Println("╠═══════════════════════════════════════════╣")

	allPassed := true
	for _, r := range results {
		fmt.Println("║                                           ║")
		fmt.Printf("║  Layer: %-34s║\n", fmt.Sprintf("%s (%s)", r.Layer, r.Stack))

		printGate("Tests", r.Tests)
		printGate("Lint", r.Lint)
		printScoreGate("Code Review", r.CodeReview, cfg.Review.MinCodeScore)
		printScoreGate("Security Review", r.SecurityReview, cfg.Review.MinSecurityScore)

		if !r.Passed {
			allPassed = false
		}
	}

	fmt.Println("║                                           ║")
	if allPassed {
		fmt.Println("║  OVERALL: ✅ ALL GATES PASSED             ║")
	} else {
		fmt.Println("║  OVERALL: ❌ FAILED                       ║")
	}
	fmt.Println("╚═══════════════════════════════════════════╝")
	fmt.Println()
}

func printGate(name string, r runner.GateResult) {
	status := "✅"
	detail := r.Detail
	if !r.Passed {
		status = "❌"
	}
	if r.Skipped {
		status = "⏭"
		detail = "skipped"
	}
	fmt.Printf("║  ├── %-16s %s %-17s║\n", name+":", status, detail)
}

func printScoreGate(name string, score float64, min float64) {
	status := "✅"
	detail := fmt.Sprintf("%.1f/10", score)
	if score < min {
		status = "❌"
		detail = fmt.Sprintf("%.1f/10 (min %.1f)", score, min)
	}
	if score == 0 {
		status = "⏭"
		detail = "skipped"
	}
	fmt.Printf("║  └── %-16s %s %-17s║\n", name+":", status, detail)
}
