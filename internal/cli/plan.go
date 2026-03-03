package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/dispatch"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scoring"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan and refine feature requirements",
	}
	cmd.AddCommand(newPlanRefineCmd(), newPlanSpecCmd(), newPlanStatusCmd())
	return cmd
}

func newPlanRefineCmd() *cobra.Command {
	var (
		iterations int
		promptOnly bool
	)

	cmd := &cobra.Command{
		Use:   "refine [ticket-url]",
		Short: "Refine requirements through iterative AI analysis (target: 25/25)",
		Long: `Generates and dispatches a requirements refinement prompt.

By default the prompt is sent to the claude CLI (if available) and the
analysis is saved to .qode/branches/<branch>/refined-analysis.md.
When two-pass scoring is enabled, a judge prompt is also dispatched.

Use --prompt-only to write the prompt file and copy to clipboard instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ticketURL := ""
			if len(args) > 0 {
				ticketURL = args[0]
			}
			return runPlanRefine(ticketURL, iterations, promptOnly)
		},
	}
	cmd.Flags().IntVar(&iterations, "iterations", 0, "refinement iteration number (0 = auto-detect)")
	cmd.Flags().BoolVar(&promptOnly, "prompt-only", false, "write prompt file and copy to clipboard without dispatching")
	return cmd
}

func newPlanSpecCmd() *cobra.Command {
	var promptOnly bool
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Generate a technical specification from the refined analysis",
		Long: `Generates and dispatches a tech spec prompt.

By default the prompt is sent to the claude CLI (if available) and the spec
is saved to .qode/branches/<branch>/spec.md.

Use --prompt-only to write the prompt file and copy to clipboard instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanSpec(promptOnly)
		},
	}
	cmd.Flags().BoolVar(&promptOnly, "prompt-only", false, "write prompt file and copy to clipboard without dispatching")
	return cmd
}

func runPlanRefine(ticketURL string, iterations int, promptOnly bool) error {
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

	ctx, err := gocontext.Load(root, branch)
	if err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)
	analysisPath := filepath.Join(branchDir, "refined-analysis.md")

	out, err := plan.BuildRefinePromptWithOutput(engine, cfg, ctx, ticketURL, iterations, analysisPath)
	if err != nil {
		return err
	}

	workerPath, judgePath, err := plan.SaveIterationFiles(root, branch, out)
	if err != nil {
		return err
	}

	if promptOnly {
		return refinePromptOnly(branch, workerPath, judgePath, out)
	}
	return refineDispatch(root, branch, out, workerPath, judgePath, analysisPath, cfg)
}

func refinePromptOnly(branch, workerPath, judgePath string, out *plan.RefineOutput) error {
	fmt.Printf("Iteration %d — prompts ready:\n\n", out.Iteration)
	fmt.Printf("  Worker prompt (do this first):\n    %s\n\n", workerPath)
	if judgePath != "" {
		fmt.Printf("  Judge prompt  (score the worker output):\n    %s\n\n", judgePath)
	}
	if !flagNoClipboard {
		if err := copyToClipboard(out.WorkerPrompt); err != nil && flagVerbose {
			fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
		} else {
			fmt.Println("Worker prompt copied to clipboard.")
		}
		fmt.Println()
	}
	fmt.Println("Next steps:")
	fmt.Println("  1. Paste worker prompt into your IDE (or use /qode-plan-refine)")
	fmt.Printf("  2. Copy the AI's analysis to: .qode/branches/%s/refined-analysis.md\n", branch)
	fmt.Println("  3. Paste judge prompt, get score → iterate until 25/25")
	fmt.Println("  4. When done: qode plan spec")
	return nil
}

func refineDispatch(root, branch string, out *plan.RefineOutput, workerPath, judgePath, analysisPath string, cfg *config.Config) error {
	d := dispatch.Resolve()
	fmt.Printf("Running refinement (iteration %d) via %s...\n", out.Iteration, d.Name())

	workerOutput, err := d.Run(context.Background(), out.WorkerPrompt, dispatch.Options{WorkingDir: root})
	if err != nil {
		if errors.Is(err, dispatch.ErrManualDispatch) {
			return refinePromptOnly(branch, workerPath, judgePath, out)
		}
		return fmt.Errorf("refine worker: %w", err)
	}

	// Use the file the AI wrote, or persist the captured output.
	if _, statErr := os.Stat(analysisPath); os.IsNotExist(statErr) {
		if workerOutput == "" {
			return fmt.Errorf("refine: AI returned no output")
		}
		if err := os.WriteFile(analysisPath, []byte(workerOutput), 0644); err != nil {
			return err
		}
	}
	savedAnalysis, _ := os.ReadFile(analysisPath)

	if out.JudgePrompt == "" || judgePath == "" {
		fmt.Println("\nRefinement analysis saved.")
		fmt.Printf("Analysis: %s\n", analysisPath)
		fmt.Println("Run: qode plan spec")
		return nil
	}

	fmt.Printf("Scoring via %s...\n", d.Name())
	judgeOutput, judgeErr := d.Run(context.Background(), out.JudgePrompt, dispatch.Options{WorkingDir: root})

	result, _ := plan.ParseIterationFromOutput(root, branch, out.Iteration, string(savedAnalysis))
	if judgeErr == nil && judgeOutput != "" {
		result = scoring.ParseScore(judgeOutput, scoring.RefineRubric)
	}

	fmt.Printf("\nRefinement iteration %d complete.\n", out.Iteration)
	fmt.Printf("Score: %d/25\n", result.TotalScore)
	if result.TotalScore >= 25 {
		fmt.Println("Target reached! Run: qode plan spec")
	} else {
		fmt.Printf("Need %d more points. Run: qode plan refine\n", 25-result.TotalScore)
	}
	return nil
}

func runPlanSpec(promptOnly bool) error {
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

	ctx, err := gocontext.Load(root, branch)
	if err != nil {
		return err
	}

	if !ctx.HasRefinedAnalysis() {
		fmt.Println("No refined analysis found.")
		fmt.Println("Run 'qode plan refine' first and save the AI output to:")
		fmt.Printf("  .qode/branches/%s/refined-analysis.md\n", branch)
		return fmt.Errorf("no refined analysis")
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)
	specPath := filepath.Join(branchDir, "spec.md")
	promptPath := filepath.Join(branchDir, ".spec-prompt.md")

	p, err := plan.BuildSpecPromptWithOutput(engine, cfg, ctx, specPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(promptPath, []byte(p), 0644); err != nil {
		return err
	}

	if promptOnly {
		return specPromptOnly(branch, promptPath, p)
	}
	return specDispatch(root, branch, p, specPath)
}

func specPromptOnly(branch, promptPath, p string) error {
	if !flagNoClipboard {
		if err := copyToClipboard(p); err != nil && flagVerbose {
			fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
		}
	}
	fmt.Printf("Spec prompt written to:\n  %s\n\n", promptPath)
	fmt.Println("Paste into your IDE (or use /qode-plan-spec).")
	fmt.Printf("Save the AI's spec to: .qode/branches/%s/spec.md\n", branch)
	return nil
}

func specDispatch(root, branch, p, specPath string) error {
	d := dispatch.Resolve()
	fmt.Printf("Generating spec via %s...\n", d.Name())

	output, err := d.Run(context.Background(), p, dispatch.Options{WorkingDir: root})
	if err != nil {
		if errors.Is(err, dispatch.ErrManualDispatch) {
			promptPath := filepath.Join(config.QodeDir, "branches", branch, ".spec-prompt.md")
			return specPromptOnly(branch, promptPath, p)
		}
		return fmt.Errorf("plan spec: %w", err)
	}

	if _, statErr := os.Stat(specPath); os.IsNotExist(statErr) {
		if output == "" {
			return fmt.Errorf("plan spec: AI returned no output")
		}
		if err := os.WriteFile(specPath, []byte(output), 0644); err != nil {
			return err
		}
	}

	fmt.Println("\nSpec generated.")
	fmt.Printf("Spec saved to:\n  %s\n", specPath)
	fmt.Println("Run: qode start")
	return nil
}

func newPlanStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show refinement iterations and scores for the current branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			branch, err := git.CurrentBranch(root)
			if err != nil {
				return err
			}

			ctx, err := gocontext.Load(root, branch)
			if err != nil {
				return err
			}

			fmt.Printf("Branch: %s\n\n", branch)

			if len(ctx.Iterations) == 0 {
				fmt.Println("No refinement iterations yet.")
				fmt.Println("Run 'qode plan refine' to start.")
				return nil
			}

			fmt.Printf("%-4s  %-8s  %s\n", "Iter", "Score", "File")
			fmt.Println("----  --------  ----")
			for _, it := range ctx.Iterations {
				scoreStr := fmt.Sprintf("%d/25", it.Score)
				if it.Score == 25 {
					scoreStr += " ✅"
				} else if it.Score >= 20 {
					scoreStr += " 🔄"
				}
				fmt.Printf("%-4d  %-8s  %s\n", it.Number, scoreStr, it.File)
			}

			latest := ctx.LatestScore()
			if latest > 0 {
				fmt.Printf("\nLatest score: %d/25", latest)
				if latest >= 25 {
					fmt.Println(" — Ready for spec generation! Run: qode plan spec")
				} else {
					fmt.Printf(" — Need %d more points. Keep iterating.\n", 25-latest)
				}
			}
			return nil
		},
	}
}
