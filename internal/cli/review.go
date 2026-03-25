package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/review"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "AI-assisted code and security reviews",
	}
	cmd.AddCommand(newReviewCodeCmd(), newReviewSecurityCmd())
	return cmd
}

func newReviewCodeCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "code",
		Short: "Generate a code review prompt for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("code", toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func newReviewSecurityCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Generate a security review prompt for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("security", toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runReview(kind string, toFile bool) error {
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
		fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
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
	outputPath := filepath.Join(branchDir, fmt.Sprintf("%s-review.md", kind))
	promptPath := filepath.Join(branchDir, fmt.Sprintf(".%s-review-prompt.md", kind))

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

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "%s review prompt saved to:\n  %s\n", capitalize(kind), promptPath)
		return nil
	}

	fmt.Fprintln(os.Stderr, "# Prompt written to stdout — use --to-file to save.")
	_, err = fmt.Print(p)
	return err
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
