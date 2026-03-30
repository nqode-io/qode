package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	var force bool
	cmd := &cobra.Command{
		Use:   "code",
		Short: "Generate a code review prompt for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("code", toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func newReviewSecurityCmd() *cobra.Command {
	var toFile bool
	var force bool
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Generate a security review prompt for the current changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview("security", toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func runReview(kind string, toFile, force bool) error {
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
	if diff == "" && !force {
		if cfg.Scoring.Strict {
			return fmt.Errorf("no changes detected: commit code first before running a review")
		}
		fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
		return nil
	}

	ctx, err := gocontext.Load(root, branch)
	if err != nil {
		return err
	}

	branchDir := filepath.Join(root, config.QodeDir, "branches", branch)

	diffPath := filepath.Join(branchDir, "diff.md")
	if err := os.WriteFile(diffPath, []byte(diff), 0600); err != nil {
		return fmt.Errorf("saving diff snapshot: %w", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(branchDir, fmt.Sprintf("%s-review.md", kind))
	promptPath := filepath.Join(branchDir, fmt.Sprintf(".%s-review-prompt.md", kind))

	var p string
	switch kind {
	case "code":
		p, err = review.BuildCodePrompt(engine, cfg, ctx, outputPath)
	case "security":
		p, err = review.BuildSecurityPrompt(engine, cfg, ctx, outputPath)
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

	_, err = fmt.Print(p)
	return err
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
