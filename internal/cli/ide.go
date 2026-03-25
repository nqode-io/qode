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
	var (
		cursor bool
		claude bool
	)

	cmd := &cobra.Command{
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

			if cmd.Flags().Changed("cursor") {
				cfg.IDE.Cursor.Enabled = cursor
			}
			if cmd.Flags().Changed("claude") {
				cfg.IDE.ClaudeCode.Enabled = claude
			}

			// TODO: add --force flag before beta to make overwriting opt-in; currently always overwrites.
			return ide.Setup(root, cfg)
		},
	}

	cmd.Flags().BoolVar(&cursor, "cursor", false, "generate Cursor configs only")
	cmd.Flags().BoolVar(&claude, "claude", false, "generate Claude Code configs only")

	return cmd
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
