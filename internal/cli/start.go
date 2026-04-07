package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/knowledge"
	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/workflow"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var toFile bool
	var force bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Generate an implementation prompt from the current spec",
		Long: `Reads spec.md and knowledge base, then generates an implementation prompt.

The prompt is written to stdout for the LLM to execute directly.
Use --to-file to write the prompt to .qode/branches/<branch>/.start-prompt.md for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(os.Stdout, os.Stderr, toFile, force)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "bypass step guard checks")
	return cmd
}

func runStart(out, errOut io.Writer, toFile, force bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}
	if flagStrict {
		sess.Config.Scoring.Strict = true
	}

	if !toFile && !force {
		if result := workflow.CheckStep("start", sess.Context, sess.Config); result.Blocked {
			if sess.Config.Scoring.Strict {
				return fmt.Errorf("%s", result.Message)
			}
			fmt.Fprintf(out, "STOP. Do not continue with this prompt.\n\n%s\n\nInform the user: %q and wait for further instructions.\n", result.Message, result.Message)
			return nil
		}
	}

	if !sess.Context.HasSpec() {
		fmt.Fprintln(errOut, "No spec.md found.")
		fmt.Fprintf(errOut, "Run /qode-plan-spec first and save the output to:\n  %s/spec.md\n", sess.Context.ContextDir)
		return fmt.Errorf("no spec")
	}

	paths, _ := knowledge.List(sess.Root, sess.Config)
	var kb string
	refs := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, relErr := filepath.Rel(sess.Root, p)
		if relErr != nil {
			rel = p
		}
		refs = append(refs, "- "+rel)
	}
	if len(refs) > 0 {
		kb = "Read the following knowledge base files:\n" + strings.Join(refs, "\n")
	}

	p, err := plan.BuildStartPrompt(sess.Engine, sess.Config, sess.Context, kb)
	if err != nil {
		return err
	}

	if toFile {
		outPath := filepath.Join(sess.Context.ContextDir, ".start-prompt.md")
		if err := writePromptToFile(outPath, p); err != nil {
			return err
		}
		fmt.Fprintf(errOut, "Implementation prompt saved to:\n  %s\n", outPath)
		return nil
	}

	_, err = fmt.Fprint(out, p)
	return err
}
