package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/knowledge"
	"github.com/spf13/cobra"
)

func newKnowledgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Manage the project knowledge base",
	}
	cmd.AddCommand(newKnowledgeListCmd(), newKnowledgeAddCmd(), newKnowledgeSearchCmd())
	return cmd
}

func newKnowledgeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List knowledge base files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			files, err := knowledge.List(root, cfg)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Println("Knowledge base is empty.")
				fmt.Printf("Add files with: qode knowledge add <path>\n")
				return nil
			}
			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		},
	}
}

func newKnowledgeAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a file to the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			src := args[0]

			kbDir := filepath.Join(root, config.QodeDir, "knowledge")
			if err := os.MkdirAll(kbDir, 0755); err != nil {
				return err
			}

			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("reading %s: %w", src, err)
			}

			dest := filepath.Join(kbDir, filepath.Base(src))
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return err
			}
			fmt.Printf("Added to knowledge base: %s\n", dest)
			return nil
		},
	}
}

func newKnowledgeSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			results, err := knowledge.Search(root, cfg, args[0])
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Printf("No results for %q\n", args[0])
				return nil
			}
			for _, r := range results {
				fmt.Printf("%s: %s\n", r.File, r.Snippet)
			}
			return nil
		},
	}
}
