package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/ticket"
	"github.com/spf13/cobra"
)

func newTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Fetch and manage ticket context",
	}
	cmd.AddCommand(newTicketFetchCmd())
	return cmd
}

func newTicketFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <url>",
		Short: "Fetch a ticket and save it to the branch context folder",
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
			branch, err := git.CurrentBranch(root)
			if err != nil {
				return err
			}

			url := args[0]
			provider, err := ticket.DetectProvider(url, cfg.TicketSystem)
			if err != nil {
				return fmt.Errorf("detecting ticket provider: %w", err)
			}

			t, err := provider.Fetch(url)
			if err != nil {
				return fmt.Errorf("fetching ticket: %w", err)
			}

			contextDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch), "context")
			if err := os.MkdirAll(contextDir, 0755); err != nil {
				return err
			}

			outPath := filepath.Join(contextDir, "ticket.md")
			content := fmt.Sprintf("# %s\n\n%s\n", t.Title, t.Body)
			if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
				return err
			}

			fmt.Printf("Fetched ticket: %s\n", t.Title)
			fmt.Printf("Saved to: %s\n", outPath)
			return nil
		},
	}
}
