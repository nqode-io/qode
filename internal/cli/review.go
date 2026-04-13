package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
			return runReview(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "code", toFile, force)
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
			return runReview(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "security", toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func runReview(ctx context.Context, out, errOut io.Writer, kind string, toFile, force bool) error {
	sess, err := loadSessionCtx(ctx)
	if err != nil {
		return err
	}
	if flagStrict {
		sess.Config.Scoring.Strict = true
	}

	diff := runDiffCommandCtx(ctx, sess.Root, sess.Config.Diff.Command)
	if diff == "" && !force {
		if sess.Config.Scoring.Strict {
			return ErrNoChanges
		}
		_, _ = fmt.Fprintln(errOut, "No changes detected on this branch.")
		return nil
	}

	ctxDir := sess.Context.ContextDir

	diffPath := filepath.Join(ctxDir, "diff.md")
	if err := iokit.WriteFile(diffPath, []byte(diff), 0600); err != nil {
		return fmt.Errorf("saving diff snapshot: %w", err)
	}

	outputPath := filepath.Join(ctxDir, fmt.Sprintf("%s-review.md", kind))
	promptPath := filepath.Join(ctxDir, fmt.Sprintf(".%s-review-prompt.md", kind))

	var p string
	switch kind {
	case "code":
		p, err = review.BuildCodePrompt(sess.Engine, sess.Config, sess.Context, outputPath)
	case "security":
		p, err = review.BuildSecurityPrompt(sess.Engine, sess.Config, sess.Context, outputPath)
	default:
		return fmt.Errorf("unknown review kind %q", kind)
	}
	if err != nil {
		return err
	}

	if toFile {
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(errOut, "%s review prompt saved to:\n  %s\n", capitalize(kind), promptPath)
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
