package cli

import (
	"fmt"
	"io"
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
			return runPlanRefine(cmd.OutOrStdout(), cmd.ErrOrStderr(), ticketURL, toFile)
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
			return runPlanSpec(cmd.OutOrStdout(), cmd.ErrOrStderr(), toFile, force)
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
			return runPlanJudge(cmd.OutOrStdout(), cmd.ErrOrStderr(), toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runPlanJudge(out, errOut io.Writer, toFile bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}
	if flagStrict {
		sess.Config.Scoring.Strict = true
	}

	if !sess.Context.HasRefinedAnalysis() {
		_, _ = fmt.Fprintln(errOut, "No refined analysis found.")
		_, _ = fmt.Fprintf(errOut, "Run 'qode plan refine' first and save the AI output to:\n  %s/refined-analysis.md\n", sess.Context.ContextDir)
		return ErrNoAnalysis
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
		_, _ = fmt.Fprintf(errOut, "Judge prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Fprint(out, p)
	return err
}

func runPlanRefine(out, errOut io.Writer, ticketURL string, toFile bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}

	branchDir := sess.Context.ContextDir
	analysisPath := filepath.Join(branchDir, "refined-analysis.md")

	refOut, err := plan.BuildRefinePromptWithOutput(sess.Engine, sess.Config, sess.Context, ticketURL, 0, analysisPath)
	if err != nil {
		return err
	}

	if toFile {
		workerPath, err := plan.SaveIterationFiles(sess.Root, sess.Branch, refOut)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(errOut, "Iteration %d — worker prompt saved to:\n  %s\n", refOut.Iteration, workerPath)
		return nil
	}

	_, err = fmt.Fprint(out, refOut.WorkerPrompt)
	return err
}

func runPlanSpec(out, errOut io.Writer, toFile, force bool) error {
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
				return fmt.Errorf("step guard: %s", result.Message)
			}
			_, _ = fmt.Fprintf(out, "STOP. Do not continue with this prompt.\n\n%s\n\nInform the user: %q and wait for further instructions.\n", result.Message, result.Message)
			return nil
		}
	}

	if !sess.Context.HasRefinedAnalysis() {
		_, _ = fmt.Fprintln(errOut, "No refined analysis found.")
		_, _ = fmt.Fprintf(errOut, "Run 'qode plan refine' first and save the AI output to:\n  %s/refined-analysis.md\n", sess.Context.ContextDir)
		return ErrNoAnalysis
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
		_, _ = fmt.Fprintf(errOut, "Spec prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Fprint(out, p)
	return err
}
