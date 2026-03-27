package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/prompt"
	"github.com/spf13/cobra"
)

func newPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan and refine feature requirements",
	}
	cmd.AddCommand(newPlanRefineCmd(), newPlanSpecCmd())
	return cmd
}

func newPlanRefineCmd() *cobra.Command {
	var (
		iterations int
		toFile     bool
	)

	cmd := &cobra.Command{
		Use:   "refine [ticket-url]",
		Short: "Generate a requirements refinement prompt (target: 25/25)",
		Long: `Generates a requirements refinement prompt and writes it to stdout.

The LLM reads the stdout output and executes it as the worker prompt.
Save your analysis to .qode/branches/<branch>/refined-analysis.md.

Use --to-file to write the prompt files to disk (worker + judge) for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ticketURL := ""
			if len(args) > 0 {
				ticketURL = args[0]
			}
			return runPlanRefine(ticketURL, iterations, toFile)
		},
	}
	cmd.Flags().IntVar(&iterations, "iterations", 0, "refinement iteration number (0 = auto-detect)")
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func newPlanSpecCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Generate a technical specification prompt from the refined analysis",
		Long: `Generates a tech spec prompt and writes it to stdout.

The LLM reads the stdout output and executes it to produce spec.md.
Use --to-file to write the prompt to disk for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlanSpec(toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runPlanRefine(ticketURL string, iterations int, toFile bool) error {
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

	if toFile {
		workerPath, judgePath, err := plan.SaveIterationFiles(root, branch, out)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Iteration %d — prompts ready:\n\n", out.Iteration)
		fmt.Fprintf(os.Stderr, "  Worker prompt:\n    %s\n\n", workerPath)
		if judgePath != "" {
			fmt.Fprintf(os.Stderr, "  Judge prompt:\n    %s\n\n", judgePath)
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, "# Prompt written to stdout — use --to-file to save.")
	_, err = fmt.Print(out.WorkerPrompt)
	return err
}

func runPlanSpec(toFile bool) error {
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
		fmt.Fprintln(os.Stderr, "No refined analysis found.")
		fmt.Fprintf(os.Stderr, "Run 'qode plan refine' first and save the AI output to:\n  .qode/branches/%s/refined-analysis.md\n", branch)
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

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Spec prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	fmt.Fprintln(os.Stderr, "# Prompt written to stdout — use --to-file to save.")
	_, err = fmt.Print(p)
	return err
}
