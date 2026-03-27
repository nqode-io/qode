package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/knowledge"
	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/prompt"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Generate an implementation prompt from the current spec",
		Long: `Reads spec.md and knowledge base, then generates an implementation prompt.

The prompt is written to stdout for the LLM to execute directly.
Use --to-file to write the prompt to .qode/branches/<branch>/.start-prompt.md for debugging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			ctx.WarnMissingPredecessors("start", os.Stderr)

			paths, listErr := knowledge.List(root, cfg)
			if listErr != nil && flagVerbose {
				fmt.Fprintf(os.Stderr, "Warning: knowledge base: %v\n", listErr)
			}
			var kb string
			refs := make([]string, 0, len(paths))
			for _, p := range paths {
				rel, relErr := filepath.Rel(root, p)
				if relErr != nil {
					rel = p
				}
				refs = append(refs, "- "+rel)
			}
			if len(refs) > 0 {
				kb = "Read the following knowledge base files:\n" + strings.Join(refs, "\n")
			}

			engine, err := prompt.NewEngine(root)
			if err != nil {
				return err
			}

			p, err := plan.BuildStartPrompt(engine, cfg, ctx, kb)
			if err != nil {
				return err
			}

			if toFile {
				outPath := filepath.Join(root, config.QodeDir, "branches", branch, ".start-prompt.md")
				if err := writePromptToFile(outPath, p); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Implementation prompt saved to:\n  %s\n", outPath)
				return nil
			}

			fmt.Fprintln(os.Stderr, "# Prompt written to stdout — use --to-file to save.")
			_, err = fmt.Print(p)
			return err
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}
