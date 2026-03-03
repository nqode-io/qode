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
		vscode bool
		claude bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Generate IDE configs (Cursor, VS Code, Claude Code)",
		Long: `Generates IDE-specific configuration files based on qode.yaml.

Cursor:    .cursorrules/*.mdc + .cursor/commands/*.mdc
VS Code:   .vscode/launch.json, tasks.json, settings.json, extensions.json
Claude:    CLAUDE.md + .claude/commands/*.md

Existing configs are preserved; qode-managed sections are demarcated with
// qode:managed-start and // qode:managed-end markers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}

			// If specific IDE flags are set, override config.
			if cmd.Flags().Changed("cursor") {
				cfg.IDE.Cursor.Enabled = cursor
			}
			if cmd.Flags().Changed("vscode") {
				cfg.IDE.VSCode.Enabled = vscode
			}
			if cmd.Flags().Changed("claude") {
				cfg.IDE.ClaudeCode.Enabled = claude
			}

			return ide.Setup(root, cfg)
		},
	}

	cmd.Flags().BoolVar(&cursor, "cursor", false, "generate Cursor configs only")
	cmd.Flags().BoolVar(&vscode, "vscode", false, "generate VS Code configs only")
	cmd.Flags().BoolVar(&claude, "claude", false, "generate Claude Code configs only")

	return cmd
}

func newIDESyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Regenerate IDE configs from qode.yaml (non-destructive)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			if err := ide.Setup(root, cfg); err != nil {
				return err
			}
			fmt.Println("IDE configs synced.")
			return nil
		},
	}
}
