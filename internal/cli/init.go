package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/ide"
	"github.com/nqode/qode/internal/prompt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise qode in a project",
		Long: `Initialise qode in the current directory.

Writes a minimal qode.yaml with defaults, creates the .qode/ directory
structure, copies embedded prompt templates, and generates IDE slash commands
for Cursor and Claude Code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			return runInitExisting(root)
		},
	}
	return cmd
}

// runInitExisting writes a minimal qode.yaml, creates .qode/ dirs,
// copies prompt templates, and generates IDE configs.
func runInitExisting(root string) error {
	cfg := config.DefaultConfig()

	// Write qode.yaml.
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	outPath := filepath.Join(root, config.ConfigFileName)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	// Create .qode directory structure.
	for _, dir := range []string{
		filepath.Join(root, config.QodeDir, "branches"),
		filepath.Join(root, config.QodeDir, "knowledge"),
		filepath.Join(root, config.QodeDir, "prompts"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Copy embedded prompt templates.
	if err := copyEmbeddedTemplates(root); err != nil {
		return err
	}

	// Generate IDE configs and slash commands.
	if err := ide.Setup(root, &cfg); err != nil {
		return fmt.Errorf("setting up IDE configs: %w", err)
	}

	fmt.Printf("Generated: %s\n", outPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'qode branch create <name>' to start your first feature")
	fmt.Println("  2. Fetch your ticket with 'qode ticket fetch <url>' or edit .qode/branches/<name>/context/ticket.md")
	fmt.Println("  3. Use /qode-plan-refine in your IDE to begin requirements refinement")

	return nil
}

// copyEmbeddedTemplates writes all built-in prompt templates into
// .qode/prompts/ so users can edit them directly. Existing files are
// overwritten so projects stay in sync with the embedded defaults.
func copyEmbeddedTemplates(root string) error {
	templates, err := prompt.EmbeddedTemplates()
	if err != nil {
		return fmt.Errorf("reading embedded templates: %w", err)
	}
	for name, content := range templates {
		dst := filepath.Join(root, config.QodeDir, "prompts", name+".md.tmpl")
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0644); err != nil {
			return fmt.Errorf("writing template %s: %w", dst, err)
		}
	}
	return nil
}
