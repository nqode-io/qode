package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/branchcontext"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/iokit"
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
			return runReview(os.Stdout, os.Stderr, "code", toFile, force)
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
			return runReview(os.Stdout, os.Stderr, "security", toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func runReview(out, errOut io.Writer, kind string, toFile, force bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}
	if flagStrict {
		sess.Config.Scoring.Strict = true
	}

	diff, err := git.DiffFromBase(sess.Root, "")
	if err != nil {
		return fmt.Errorf("getting diff: %w", err)
	}
	if diff == "" && !force {
		if sess.Config.Scoring.Strict {
			return fmt.Errorf("no changes detected: commit code first before running a review")
		}
		fmt.Fprintln(errOut, "No changes detected. Commit some code first.")
		return nil
	}

	branchDir := sess.Context.ContextDir

	if err := branchcontext.EnsureContextDir(sess.Root, sess.Branch); err != nil {
		return fmt.Errorf("creating context directory: %w", err)
	}

	diffPath := filepath.Join(branchDir, "diff.md")
	if err := iokit.WriteFile(diffPath, []byte(diff), 0600); err != nil {
		return fmt.Errorf("saving diff snapshot: %w", err)
	}

	outputPath := filepath.Join(branchDir, fmt.Sprintf("%s-review.md", kind))
	promptPath := filepath.Join(branchDir, fmt.Sprintf(".%s-review-prompt.md", kind))

	var p string
	switch kind {
	case "code":
		p, err = review.BuildCodePrompt(sess.Engine, sess.Config, sess.Context, outputPath)
	case "security":
		p, err = review.BuildSecurityPrompt(sess.Engine, sess.Config, sess.Context, outputPath)
	}
	if err != nil {
		return err
	}

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(errOut, "%s review prompt saved to:\n  %s\n", capitalize(kind), promptPath)
		return nil
	}

	_, err = fmt.Fprint(out, p)
	return err
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
