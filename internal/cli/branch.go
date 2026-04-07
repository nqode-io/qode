package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/iokit"
	"github.com/spf13/cobra"
)

func safeBranchDir(root, name string) (string, error) {
	sanitized := git.SanitizeBranchName(name)
	base := filepath.Join(root, config.QodeDir, "branches")
	target := filepath.Join(base, sanitized)
	rel, err := filepath.Rel(base, target)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return "", fmt.Errorf("invalid branch name %q: path traversal detected", name)
	}
	return target, nil
}

func newBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage feature branches and their context",
	}
	cmd.AddCommand(
		newBranchCreateCmd(),
		newBranchRemoveCmd(),
	)
	return cmd
}

func newBranchCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name> [base]",
		Short: "Create a feature branch with a context folder",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			base := ""
			if len(args) > 1 {
				base = args[1]
			}
			return runBranchCreate(os.Stdout, name, base)
		},
	}
	return cmd
}

func runBranchCreate(out io.Writer, name, base string) error {
	if len(base) > 0 && base[0] == '-' {
		return fmt.Errorf("invalid base branch %q: must not start with '-'", base)
	}

	root, err := resolveRoot()
	if err != nil {
		return err
	}

	// Create git branch.
	if err := git.CreateBranch(root, name, base); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	// Create context folder.
	branchDir, err := safeBranchDir(root, name)
	if err != nil {
		return err
	}
	contextDir := filepath.Join(branchDir, "context")
	if err := iokit.EnsureDir(contextDir); err != nil {
		return fmt.Errorf("creating context dir: %w", err)
	}

	// Stub files.
	stubs := map[string]string{
		"ticket.md": "# Ticket\n\nPaste ticket content here, or use /qode-ticket-fetch <url> in your IDE.\n",
		"notes.md":  "# Notes\n\nAdd any additional context, decisions, or open questions here.\n",
	}
	for name, content := range stubs {
		p := filepath.Join(contextDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := iokit.WriteFile(p, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(out, "Created branch: %s\n", name)
	fmt.Fprintf(out, "Context folder: %s\n", contextDir)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintf(out, "  1. Fetch ticket: /qode-ticket-fetch <url>  (in IDE)\n")
	fmt.Fprintf(out, "     Or paste into: %s/ticket.md\n", contextDir)
	fmt.Fprintf(out, "  2. Add mockups / designs to: %s/\n", contextDir)
	fmt.Fprintf(out, "  3. Run: qode plan refine\n")
	return nil
}

func newBranchRemoveCmd() *cobra.Command {
	var keepBranchCtx bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove branch context folder and delete git branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBranchRemove(os.Stdout, os.Stderr, args[0], keepBranchCtx)
		},
	}
	cmd.Flags().BoolVar(&keepBranchCtx, "keep-branch-context", false, "keep the .qode/branches/ context folder")
	return cmd
}

func runBranchRemove(out, errOut io.Writer, name string, keepCtx bool) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	branchDir, err := safeBranchDir(root, name)
	if err != nil {
		return err
	}

	keepCtx = cfg.Branch.KeepBranchContext || keepCtx
	if !keepCtx {
		if err := os.RemoveAll(branchDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing context: %w", err)
		}
		fmt.Fprintf(out, "Removed context for branch: %s\n", name)
	}

	if err := git.DeleteBranch(root, name); err != nil {
		fmt.Fprintf(errOut, "Warning: could not delete git branch: %v\n", err)
	} else {
		fmt.Fprintf(out, "Deleted git branch: %s\n", name)
	}
	return nil
}
