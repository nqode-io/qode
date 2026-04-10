package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/workflow"
	"github.com/spf13/cobra"
)

func newPrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request commands",
	}
	cmd.AddCommand(newPrCreateCmd())
	return cmd
}

func newPrCreateCmd() *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Generate a PR prompt and create via MCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrCreate(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), base)
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "Base branch (overrides config and auto-detection)")
	return cmd
}

func runPrCreate(ctx context.Context, out, errOut io.Writer, base string) error {
	sess, err := loadSessionCtx(ctx)
	if err != nil {
		return err
	}

	if result := workflow.CheckStep("pr", sess.Context, sess.Config); result.Blocked {
		_, _ = fmt.Fprintln(errOut, result.Message)
		return nil
	}

	baseBranch, err := resolveBaseBranch(ctx, sess.Root, base, sess.Config.PR.BaseBranch)
	if err != nil {
		return err
	}

	branchDir := sess.Context.ContextDir
	codeReview := iokit.ReadFileOrString(filepath.Join(branchDir, "code-review.md"), "")
	secReview := iokit.ReadFileOrString(filepath.Join(branchDir, "security-review.md"), "")
	diff := iokit.ReadFileOrString(filepath.Join(branchDir, "diff.md"), "")

	data := prompt.NewTemplateData(sess.Engine.ProjectName(), sess.Branch).
		WithBranchDir(branchDir).
		WithTicket(sess.Context.Ticket).
		WithSpec(sess.Context.Spec).
		WithDiff(diff).
		WithBaseBranch(baseBranch).
		WithCodeReview(codeReview).
		WithSecurityReview(secReview).
		WithDraftPR(sess.Config.PR.Draft).
		Build()

	p, err := sess.Engine.Render("pr/create", data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(out, p)
	return err
}

// resolveBaseBranch returns the base branch using the priority chain:
// flag > config > git auto-detection > "main".
func resolveBaseBranch(ctx context.Context, root, flagVal, configVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if configVal != "" {
		return configVal, nil
	}
	detected, err := git.DefaultBranch(ctx, root)
	if err == nil && detected != "" {
		return detected, nil
	}
	return "main", nil
}
