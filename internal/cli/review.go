package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/dispatch"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/review"
	"github.com/nqode/qode/internal/scoring"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "AI-assisted code and security reviews",
	}
	cmd.AddCommand(newReviewCodeCmd(), newReviewSecurityCmd(), newReviewAllCmd())
	return cmd
}

func newReviewCodeCmd() *cobra.Command {
	var promptOnly bool
	cmd := &cobra.Command{
		Use:   "code",
		Short: "Run a code review for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("code", promptOnly)
		},
	}
	cmd.Flags().BoolVar(&promptOnly, "prompt-only", false, "write prompt file without dispatching")
	return cmd
}

func newReviewSecurityCmd() *cobra.Command {
	var promptOnly bool
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Run a security review for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("security", promptOnly)
		},
	}
	cmd.Flags().BoolVar(&promptOnly, "prompt-only", false, "write prompt file without dispatching")
	return cmd
}

func newReviewAllCmd() *cobra.Command {
	var promptOnly bool
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Run both code and security reviews",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runReview("code", promptOnly); err != nil {
				return err
			}
			return runReview("security", promptOnly)
		},
	}
	cmd.Flags().BoolVar(&promptOnly, "prompt-only", false, "write prompt files without dispatching")
	return cmd
}

func runReview(kind string, promptOnly bool) error {
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

	diff, err := git.DiffFromBase(root, "")
	if err != nil {
		return fmt.Errorf("getting diff: %w", err)
	}
	if diff == "" {
		fmt.Println("No changes detected. Commit some code first.")
		return nil
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
	if err := os.MkdirAll(branchDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(branchDir, fmt.Sprintf("%s-review.md", kind))
	promptPath := filepath.Join(branchDir, fmt.Sprintf(".%s-review-prompt.md", kind))

	// Build prompt. Include outputPath so the AI knows where to write.
	var p string
	switch kind {
	case "code":
		p, err = review.BuildCodePrompt(engine, cfg, ctx, diff, outputPath)
	case "security":
		p, err = review.BuildSecurityPrompt(engine, cfg, ctx, diff, outputPath)
	}
	if err != nil {
		return err
	}

	// Always persist the prompt file for debugging and slash-command use.
	if err := os.WriteFile(promptPath, []byte(p), 0644); err != nil {
		return err
	}

	if promptOnly {
		return reviewPromptOnly(kind, promptPath)
	}
	return reviewDispatch(root, kind, p, outputPath)
}

func reviewPromptOnly(kind, promptPath string) error {
	fmt.Printf("%s review prompt written to:\n  %s\n\n", capitalize(kind), promptPath)
	fmt.Printf("Use slash command: /qode-review-%s\n", kind)
	return nil
}

func reviewDispatch(root, kind, p, outputPath string) error {
	// Remove stale output so the AI always writes a fresh review.
	// This prevents a re-run from silently reporting the previous score.
	_ = os.Remove(outputPath)

	if err := dispatch.RunInteractive(context.Background(), p, dispatch.Options{WorkingDir: root}); err != nil {
		return fmt.Errorf("%s review: %w", kind, err)
	}

	score := scoring.ExtractScoreFromFile(outputPath)
	fmt.Printf("\n%s review complete.\n", capitalize(kind))
	if score > 0 {
		fmt.Printf("Score: %.1f/10\n", score)
	} else {
		fmt.Println("Score: not found — check review file.")
	}
	fmt.Printf("Review saved to:\n  %s\n", outputPath)
	return nil
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
