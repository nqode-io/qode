package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scaffold"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise qode in a project",
		Long: `Initialise qode in the current directory.

Writes a minimal qode.yaml with defaults, creates the .qode/ directory
structure, copies embedded prompt templates, and generates IDE workflow assets
for Cursor, Claude Code, and Codex.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			return runInitExisting(cmd.OutOrStdout(), root)
		},
	}
	return cmd
}

// runInitExisting writes qode.yaml with defaults, creates .qode/ dirs, copies
// prompt templates, and generates IDE configs. .qode/scoring.yaml is only
// written on first run so user-customised rubrics are never overwritten.
func runInitExisting(out io.Writer, root string) error {
	cfg := config.DefaultConfig()
	cfg.QodeVersion = rootCmd.Version
	if cfg.QodeVersion == "" {
		cfg.QodeVersion = "dev"
	}

	// Always write qode.yaml — rubrics live in .qode/scoring.yaml, not here.
	cfgForYaml := cfg
	cfgForYaml.Scoring.Rubrics = nil
	data, err := yaml.Marshal(&cfgForYaml)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	outPath := filepath.Join(root, config.ConfigFileName)
	if err := iokit.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	_, _ = fmt.Fprintf(out, "Generated: %s\n", outPath)

	// Create .qode directory structure.
	for _, dir := range []string{
		filepath.Join(root, config.QodeDir, "contexts"),
		filepath.Join(root, config.QodeDir, "knowledge"),
		filepath.Join(root, config.QodeDir, "prompts"),
	} {
		if err := iokit.EnsureDir(dir); err != nil {
			return err
		}
	}

	// Write .qode/scoring.yaml only on first run; re-runs preserve custom rubrics.
	scoringPath := filepath.Join(root, config.QodeDir, config.ScoringFileName)
	if _, statErr := os.Stat(scoringPath); os.IsNotExist(statErr) {
		scoringFile := config.ScoringFileConfig{Rubrics: config.DefaultRubricConfigs()}
		scoringData, err := yaml.Marshal(&scoringFile)
		if err != nil {
			return fmt.Errorf("marshaling scoring config: %w", err)
		}
		if err := iokit.WriteFile(scoringPath, scoringData, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", scoringPath, err)
		}
		_, _ = fmt.Fprintf(out, "Generated: %s\n", scoringPath)
	}

	// Copy embedded prompt templates.
	if err := copyEmbeddedTemplates(root); err != nil {
		return err
	}

	// Generate IDE configs and workflow assets using the loaded (or default) config.
	if err := scaffold.Setup(out, root, &cfg); err != nil {
		return fmt.Errorf("setting up IDE configs: %w", err)
	}

	if err := scaffold.AppendGitignoreRules(context.Background(), out, root); err != nil {
		return fmt.Errorf("appending gitignore rules: %w", err)
	}

	// Remove scaffold prompt overrides: these templates are one-time scaffolding
	// tools used only during init. Deleting them ensures future qode versions
	// can update the generated workflow assets without local overrides blocking the update.
	scaffoldPromptsDir := filepath.Join(root, config.QodeDir, "prompts", "scaffold")
	if err := os.RemoveAll(scaffoldPromptsDir); err != nil {
		return fmt.Errorf("removing scaffold prompts: %w", err)
	}

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
		if err := iokit.WriteFile(dst, content, 0644); err != nil {
			return fmt.Errorf("writing template %s: %w", dst, err)
		}
	}
	return nil
}
