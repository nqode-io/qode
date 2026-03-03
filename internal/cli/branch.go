package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/spf13/cobra"
)

func newBranchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage feature branches and their context",
	}
	cmd.AddCommand(
		newBranchCreateCmd(),
		newBranchListCmd(),
		newBranchFocusCmd(),
		newBranchRemoveCmd(),
	)
	return cmd
}

func newBranchCreateCmd() *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a feature branch with a context folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			name := args[0]

			// Create git branch.
			if err := git.CreateBranch(root, name, base); err != nil {
				return fmt.Errorf("creating branch: %w", err)
			}

			// Create context folder.
			contextDir := filepath.Join(root, config.QodeDir, "branches", name, "context")
			if err := os.MkdirAll(contextDir, 0755); err != nil {
				return fmt.Errorf("creating context dir: %w", err)
			}

			// Stub files.
			stubs := map[string]string{
				"ticket.md":   "# Ticket\n\nPaste ticket content here, or run: qode ticket fetch <url>\n",
				"notes.md":    "# Notes\n\nAdd any additional context, decisions, or open questions here.\n",
			}
			for name, content := range stubs {
				p := filepath.Join(contextDir, name)
				if _, err := os.Stat(p); os.IsNotExist(err) {
					if err := os.WriteFile(p, []byte(content), 0644); err != nil {
						return err
					}
				}
			}

			fmt.Printf("Created branch: %s\n", name)
			fmt.Printf("Context folder: %s\n", contextDir)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  1. Add ticket: qode ticket fetch <url>\n")
			fmt.Printf("     Or paste into: %s/ticket.md\n", contextDir)
			fmt.Printf("  2. Add mockups / designs to: %s/\n", contextDir)
			fmt.Printf("  3. Run: qode plan refine\n")
			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", "", "base branch (default: current branch)")
	return cmd
}

func newBranchListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List feature branches with context folders",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			branchesDir := filepath.Join(root, config.QodeDir, "branches")
			entries, err := os.ReadDir(branchesDir)
			if os.IsNotExist(err) {
				fmt.Println("No feature branches found.")
				return nil
			}
			if err != nil {
				return err
			}

			current, _ := git.CurrentBranch(root)

			fmt.Printf("%-40s  %s\n", "Branch", "Context")
			fmt.Println("----------------------------------------  -------")
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				marker := "  "
				if e.Name() == current {
					marker = "→ "
				}
				ctxDir := filepath.Join(branchesDir, e.Name(), "context")
				files, _ := os.ReadDir(ctxDir)
				fmt.Printf("%s%-38s  %d file(s)\n", marker, e.Name(), len(files))
			}
			return nil
		},
	}
}

func newBranchFocusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "focus <name>",
		Short: "Switch to a branch and show its context summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			name := args[0]

			if err := git.CheckoutBranch(root, name); err != nil {
				return fmt.Errorf("checking out branch: %w", err)
			}

			contextDir := filepath.Join(root, config.QodeDir, "branches", name, "context")
			files, err := os.ReadDir(contextDir)
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			fmt.Printf("Switched to branch: %s\n\n", name)
			if len(files) > 0 {
				fmt.Println("Context files:")
				for _, f := range files {
					if !f.IsDir() {
						fmt.Printf("  %s\n", f.Name())
					}
				}
			} else {
				fmt.Println("No context files yet. Add them to:", contextDir)
			}
			return nil
		},
	}
}

func newBranchRemoveCmd() *cobra.Command {
	var keepBranch bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove branch context folder (optionally delete git branch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			name := args[0]

			contextDir := filepath.Join(root, config.QodeDir, "branches", name)
			if err := os.RemoveAll(contextDir); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing context: %w", err)
			}
			fmt.Printf("Removed context for branch: %s\n", name)

			if !keepBranch {
				if err := git.DeleteBranch(root, name); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not delete git branch: %v\n", err)
				} else {
					fmt.Printf("Deleted git branch: %s\n", name)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepBranch, "keep-branch", false, "keep the git branch, only remove context folder")
	return cmd
}
