package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/knowledge"
	"github.com/nqode/qode/internal/plan"
	"github.com/nqode/qode/internal/prompt"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Generate an implementation prompt from the current spec",
		Long: `Reads spec.md and knowledge base, then generates an implementation prompt.

The prompt is written to .qode/branches/<branch>/.start-prompt.md.
It includes the full spec, knowledge base fragments, and stack-specific
clean code requirements.`,
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

			ctx, err := context.Load(root, branch)
			if err != nil {
				return err
			}

			kb, err := knowledge.Load(root, cfg)
			if err != nil && flagVerbose {
				fmt.Fprintf(os.Stderr, "Warning: knowledge base: %v\n", err)
			}

			engine, err := prompt.NewEngine(root)
			if err != nil {
				return err
			}

			p, err := plan.BuildStartPrompt(engine, cfg, ctx, kb)
			if err != nil {
				return err
			}

			outPath := filepath.Join(root, config.QodeDir, "branches", branch, ".start-prompt.md")
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(outPath, []byte(p), 0644); err != nil {
				return err
			}

			if err := copyToClipboard(p); err != nil && flagVerbose {
				fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
			}

			fmt.Printf("Implementation prompt written to:\n  %s\n\n", outPath)
			fmt.Println("Paste into Cursor/Claude Code, or use: qode start --open")
			return nil
		},
	}
}
