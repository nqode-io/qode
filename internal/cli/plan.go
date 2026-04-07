package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/workflow"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan and refine feature requirements",
	}
	cmd.AddCommand(newPlanRefineCmd(), newPlanJudgeCmd(), newPlanSpecCmd())
	return cmd
}

func newPlanRefineCmd() *cobra.Command {
	var toFile bool

	cmd := &cobra.Command{
		Use:   "refine [ticket-url]",
		Short: "Generate a requirements refinement prompt",
		Long: `Generates a requirements refinement prompt and writes it to stdout.

The LLM reads the stdout output and executes it as the worker prompt.
Save your analysis to .qode/branches/<branch>/refined-analysis.md.

Use --to-file to write the prompt files to disk (worker + judge) for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ticketURL := ""
			if len(args) > 0 {
				ticketURL = args[0]
			}
			return runPlanRefine(ticketURL, toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func newPlanSpecCmd() *cobra.Command {
	var toFile bool
	var force bool
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Generate a technical specification prompt from the refined analysis",
		Long: `Generates a tech spec prompt and writes it to stdout.

The LLM reads the stdout output and executes it to produce spec.md.
Use --to-file to write the prompt to disk for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanSpec(toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func newPlanJudgeCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "judge",
		Short: "Generate the judge scoring prompt from the current refined analysis",
		Long: `Generates the judge scoring prompt and writes it to stdout.

The LLM reads the stdout output and scores the refined analysis.
Requires refined-analysis.md to exist in the branch directory.

Use --to-file to write the prompt to disk for debugging the judge template.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanJudge(toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runPlanJudge(toFile bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}

	if !sess.Context.HasRefinedAnalysis() {
		fmt.Fprintln(os.Stderr, "No refined analysis found.")
		fmt.Fprintf(os.Stderr, "Run 'qode plan refine' first and save the AI output to:\n  %s/refined-analysis.md\n", sess.Context.ContextDir)
		return fmt.Errorf("no refined analysis")
	}

	p, err := plan.BuildJudgePrompt(sess.Engine, sess.Config, sess.Context)
	if err != nil {
		return err
	}

	branchDir := sess.Context.ContextDir
	promptPath := filepath.Join(branchDir, ".refine-judge-prompt.md")

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Judge prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Print(p)
	return err
}

func runPlanRefine(ticketURL string, toFile bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}

	branchDir := sess.Context.ContextDir
	analysisPath := filepath.Join(branchDir, "refined-analysis.md")

	out, err := plan.BuildRefinePromptWithOutput(sess.Engine, sess.Config, sess.Context, ticketURL, 0, analysisPath)
	if err != nil {
		return err
	}

	if toFile {
		workerPath, err := plan.SaveIterationFiles(sess.Root, sess.Branch, out)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Iteration %d — worker prompt saved to:\n  %s\n", out.Iteration, workerPath)
		return nil
	}

	_, err = fmt.Print(out.WorkerPrompt)
	return err
}

func runPlanSpec(toFile, force bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}
	if flagStrict {
		sess.Config.Scoring.Strict = true
	}

	if !toFile && !force {
		if result := workflow.CheckStep("spec", sess.Context, sess.Config); result.Blocked {
			if sess.Config.Scoring.Strict {
				return fmt.Errorf("%s", result.Message)
			}
			fmt.Printf("STOP. Do not continue with this prompt.\n\n%s\n\nInform the user: %q and wait for further instructions.\n", result.Message, result.Message)
			return nil
		}
	}

	if !sess.Context.HasRefinedAnalysis() {
		fmt.Fprintln(os.Stderr, "No refined analysis found.")
		fmt.Fprintf(os.Stderr, "Run 'qode plan refine' first and save the AI output to:\n  %s/refined-analysis.md\n", sess.Context.ContextDir)
		return fmt.Errorf("no refined analysis")
	}

	branchDir := sess.Context.ContextDir
	specPath := filepath.Join(branchDir, "spec.md")
	promptPath := filepath.Join(branchDir, ".spec-prompt.md")

	p, err := plan.BuildSpecPromptWithOutput(sess.Engine, sess.Config, sess.Context, specPath)
	if err != nil {
		return err
	}

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Spec prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Print(p)
	return err
}
