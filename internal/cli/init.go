package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/detect"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise qode in a project",
		Long: `Detect the project's tech stack and create qode.yaml.

Scans the current directory, detects the tech stack, and writes qode.yaml.
Run 'qode ide setup' afterwards to generate IDE configs.`,
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

// runInitExisting scans an existing project root and generates qode.yaml.
func runInitExisting(root string) error {
	fmt.Printf("Scanning project at %s ...\n\n", root)

	topo, err := workspace.Detect(root)
	if err != nil {
		return fmt.Errorf("detecting workspace topology: %w", err)
	}

	layers, err := detect.Composite(root)
	if err != nil {
		return fmt.Errorf("detecting tech stacks: %w", err)
	}

	if len(layers) == 0 {
		fmt.Println("No recognised tech stacks detected.")
		fmt.Println("You can manually set the stack in qode.yaml.")
		layers = []detect.DetectedLayer{{Name: "default", Path: ".", Stack: "unknown"}}
	}

	// Report findings.
	fmt.Printf("Detected topology: %s\n", topo)
	fmt.Printf("Detected layers:\n")
	for _, l := range layers {
		fmt.Printf("  %-20s → %-12s (confidence: %.0f%%)\n", l.Path+"/", l.Stack, l.Confidence*100)
	}
	fmt.Println()

	cfg := config.DefaultConfig()
	cfg.Project.Name = filepath.Base(root)
	cfg.Project.Topology = config.Topology(topo)

	for _, l := range layers {
		tc := l.Test
		if tc.Unit == "" {
			tc = config.StackDefaults[l.Stack]
		}
		cfg.Project.Layers = append(cfg.Project.Layers, config.LayerConfig{
			Name:  l.Name,
			Path:  l.Path,
			Stack: l.Stack,
			Test:  tc,
		})
	}

	// Write qode.yaml.
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	outPath := filepath.Join(root, config.ConfigFileName)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	// Create .qode directory structure and copy prompt templates.
	for _, dir := range []string{
		filepath.Join(root, config.QodeDir, "branches"),
		filepath.Join(root, config.QodeDir, "knowledge"),
		filepath.Join(root, config.QodeDir, "prompts"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if err := copyEmbeddedTemplates(root); err != nil {
		return err
	}

	fmt.Printf("Generated: %s\n", outPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review qode.yaml and adjust layer paths / test commands if needed")
	fmt.Println("  2. Run 'qode ide setup' to generate IDE configs")
	fmt.Println("  3. Run 'qode branch create <name>' to start your first feature")

	return nil
}

// copyEmbeddedTemplates writes all built-in prompt templates into
// .qode/prompts/ so users can edit them directly. Existing files are
// not overwritten.
func copyEmbeddedTemplates(root string) error {
	templates, err := prompt.EmbeddedTemplates()
	if err != nil {
		return fmt.Errorf("reading embedded templates: %w", err)
	}
	for name, content := range templates {
		dst := filepath.Join(root, config.QodeDir, "prompts", name+".md.tmpl")
		if _, err := os.Stat(dst); err == nil {
			continue // already exists — do not overwrite
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0644); err != nil {
			return fmt.Errorf("writing template %s: %w", dst, err)
		}
	}
	return nil
}
