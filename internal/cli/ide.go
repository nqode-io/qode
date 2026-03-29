package cli

import (
	"fmt"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/ide"
	"github.com/spf13/cobra"
)

func newIDECmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ide",
		Short: "Manage IDE configurations",
	}
	cmd.AddCommand(newIDESetupCmd(), newIDESyncCmd())
	return cmd
}

func newIDESetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Generate IDE configs (Cursor, Claude Code)",
		Long: `Generates IDE-specific configuration files based on qode.yaml.

Cursor:  .cursorrules/*.mdc + .cursor/commands/*.mdc
Claude:  CLAUDE.md + .claude/commands/*.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			return ide.Setup(root, cfg)
		},
	}
}

func newIDESyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Regenerate IDE configs from qode.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			// TODO: add --force flag before beta to make overwriting opt-in; currently always overwrites.
			if err := ide.Setup(root, cfg); err != nil {
				return err
			}
			fmt.Println("IDE configs synced.")
			return nil
		},
	}
}
