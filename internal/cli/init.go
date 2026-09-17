package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	var configOnly bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise qode in a project",
		Long: `Initialise qode in the current directory.

Generates qode.yaml with commented defaults when it is absent. When it already
exists, every value you set is kept, qode_version is refreshed (on released
builds), and settings
added by newer qode versions are appended with their defaults — nothing is
reset. Creates the .qode/ directory structure, copies embedded prompt
templates, and generates IDE workflow assets for the IDEs enabled in qode.yaml
(Cursor, Claude Code, Codex, OpenCode).

Use --config-only to write qode.yaml and stop, so you can review and edit it
before anything else is generated. --force overwrites an existing qode.yaml
with the defaults instead of preserving it (unlike --force on plan, review and
start, which bypasses step guard checks).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			return runInitExisting(cmd.Context(), cmd.OutOrStdout(), root, rootCmd.Version, configOnly, force)
		},
	}
	cmd.Flags().BoolVar(&configOnly, "config-only", false, "write qode.yaml with commented defaults and stop")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing qode.yaml with the defaults")
	return cmd
}

// runInitExisting generates or upgrades qode.yaml and then scaffolds the project
// against the configuration that file actually carries. With configOnly it writes
// qode.yaml and stops.
func runInitExisting(ctx context.Context, out io.Writer, root, binaryVersion string, configOnly, force bool) error {
	if configOnly {
		if err := config.WriteDefault(ctx, root, binaryVersion, force); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Generated: %s\n", filepath.Join(root, config.ConfigFileName))
		return nil
	}
	cfg, err := ensureConfig(ctx, out, root, binaryVersion, force)
	if err != nil {
		return err
	}
	return scaffoldFromConfig(ctx, out, root, cfg)
}

// ensureConfig generates, or preserves-and-upgrades, the project qode.yaml and
// returns the loaded configuration. A file that cannot be parsed or validated is
// left byte-identical.
func ensureConfig(ctx context.Context, out io.Writer, root, binaryVersion string, force bool) (*config.Config, error) {
	path := filepath.Join(root, config.ConfigFileName)
	_, statErr := os.Stat(path)
	switch {
	case errors.Is(statErr, fs.ErrNotExist) || force:
		if err := config.WriteDefault(ctx, root, binaryVersion, true); err != nil {
			return nil, err
		}
		_, _ = fmt.Fprintf(out, "Generated: %s\n", path)
	case statErr != nil:
		return nil, fmt.Errorf("checking %s: %w", path, statErr)
	default:
		changed, err := config.Upgrade(ctx, root, binaryVersion)
		if err != nil {
			return nil, withOverwriteHint(err)
		}
		if changed {
			_, _ = fmt.Fprintf(out, "Updated: %s\n", path)
		}
	}
	// Load unconditionally: one code path, and a freshly written file is parsed and
	// validated before anything is scaffolded against it. Errors here may name
	// .qode/scoring.yaml or ~/.qode/config.yaml, so they are returned undecorated.
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// withOverwriteHint appends the escape hatch for a broken project qode.yaml. Only
// config.Upgrade returns ErrConfigInvalid, so a malformed .qode/scoring.yaml can
// never be answered with "overwrite your qode.yaml".
func withOverwriteHint(err error) error {
	if !errors.Is(err, config.ErrConfigInvalid) {
		return err
	}
	return fmt.Errorf("%w\nqode.yaml could not be read; fix it, or overwrite it with 'qode init --config-only --force'", err)
}

// scaffoldFromConfig creates the .qode/ directory structure, writes the first-run
// scoring rubrics, copies prompt templates, and generates workflow assets for the
// IDEs cfg enables. .qode/scoring.yaml is only written on first run so
// user-customised rubrics are never overwritten.
func scaffoldFromConfig(ctx context.Context, out io.Writer, root string, cfg *config.Config) error {
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
	if err := scaffold.Setup(out, root, cfg); err != nil {
		return fmt.Errorf("setting up IDE configs: %w", err)
	}

	if err := scaffold.AppendGitignoreRules(ctx, out, root); err != nil {
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
