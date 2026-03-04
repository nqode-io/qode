package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/detect"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newInitCmd() *cobra.Command {
	var (
		newProject bool
		scaffold   bool
		ide        []string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise qode in a project",
		Long: `Detect the project's tech stack and create qode.yaml.

For existing projects:
  qode init                # Auto-detect stack, create qode.yaml

For new (greenfield) projects:
  qode init --new          # Interactive wizard
  qode init --new --scaffold  # Wizard + scaffold directory structure

For multi-repo workspaces:
  qode init --workspace    # Detect sibling repos, create qode-workspace.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			if newProject {
				return runInitNew(root, scaffold, ide)
			}

			// Check if a workspace flag was set.
			ws, _ := cmd.Flags().GetBool("workspace")
			if ws {
				return runInitWorkspace(root)
			}

			return runInitExisting(root, ide)
		},
	}

	cmd.Flags().BoolVar(&newProject, "new", false, "start a new greenfield project (interactive wizard)")
	cmd.Flags().BoolVar(&scaffold, "scaffold", false, "scaffold project directory structure (use with --new)")
	cmd.Flags().BoolVar(new(bool), "workspace", false, "initialise a multi-repo workspace")
	cmd.Flags().StringSliceVar(&ide, "ide", []string{}, "IDEs to configure: cursor,vscode,claude (default: all enabled)")

	return cmd
}

// runInitExisting scans an existing project root and generates qode.yaml.
func runInitExisting(root string, ides []string) error {
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

	if len(ides) > 0 {
		applyIDEFlags(&cfg, ides)
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

// runInitNew starts the interactive new-project wizard.
func runInitNew(root string, scaffold bool, ides []string) error {
	fmt.Println("=== qode new project wizard ===")
	fmt.Println()

	name := readLine("Project name", filepath.Base(root))
	primaryFE := pickChoice("Frontend stack", []string{"react", "angular", "vue", "svelte", "none"}, "react")
	primaryBE := pickChoice("Backend stack", []string{"nextjs", "dotnet", "java", "python", "go", "none"}, "dotnet")
	ticketSys := pickChoice("Ticket system", []string{"jira", "azure-devops", "linear", "none"}, "none")

	cfg := config.DefaultConfig()
	cfg.Project.Name = name
	cfg.Project.Topology = config.TopologyMonorepo

	if primaryFE != "none" {
		tc := config.StackDefaults[primaryFE]
		cfg.Project.Layers = append(cfg.Project.Layers, config.LayerConfig{
			Name:  "frontend",
			Path:  "./frontend",
			Stack: primaryFE,
			Test:  tc,
		})
	}
	if primaryBE != "none" {
		tc := config.StackDefaults[primaryBE]
		cfg.Project.Layers = append(cfg.Project.Layers, config.LayerConfig{
			Name:  "backend",
			Path:  "./backend",
			Stack: primaryBE,
			Test:  tc,
		})
	}

	if ticketSys != "none" {
		cfg.TicketSystem = config.TicketSystemConfig{
			Type: ticketSys,
			Auth: config.AuthConfig{Method: "token", EnvVar: strings.ToUpper(strings.ReplaceAll(ticketSys, "-", "_")) + "_API_TOKEN"},
		}
	}

	if len(ides) > 0 {
		applyIDEFlags(&cfg, ides)
	}

	if scaffold {
		if err := scaffoldProject(root, cfg.Project.Layers); err != nil {
			return fmt.Errorf("scaffolding project: %w", err)
		}
	}

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

	fmt.Printf("\nGenerated: %s\n", outPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'qode ide setup' to generate IDE configs")
	fmt.Println("  2. Run 'qode branch create <name>' to start your first feature")

	return nil
}

// runInitWorkspace scans sibling directories and creates a workspace config.
func runInitWorkspace(root string) error {
	fmt.Printf("Scanning for workspace repos in %s ...\n\n", root)

	repos, err := workspace.DetectRepos(root)
	if err != nil {
		return fmt.Errorf("detecting repos: %w", err)
	}

	if len(repos) == 0 {
		fmt.Println("No additional repos found. Run 'qode init' for a single-repo project.")
		return nil
	}

	fmt.Printf("Found repos:\n")
	for _, r := range repos {
		fmt.Printf("  %-30s → %s\n", r.Name, r.Path)
	}
	fmt.Println()

	wsCfg := config.WorkspaceConfig{}
	for _, r := range repos {
		wsCfg.Repos = append(wsCfg.Repos, config.RepoRef{
			Name:   r.Name,
			Path:   r.Path,
			Branch: "main",
		})
	}

	rootCfg := config.DefaultConfig()
	rootCfg.Project.Name = filepath.Base(root)
	rootCfg.Project.Topology = config.TopologyMultirepo
	rootCfg.Workspace = wsCfg

	data, err := yaml.Marshal(&rootCfg)
	if err != nil {
		return err
	}

	outPath := filepath.Join(root, config.WorkspaceConfigFileName)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	fmt.Printf("Generated: %s\n", outPath)
	fmt.Println()
	fmt.Println("Each repo can also have its own qode.yaml. Run 'qode init' inside each.")

	return nil
}

func applyIDEFlags(cfg *config.Config, ides []string) {
	enabled := map[string]bool{}
	for _, ide := range ides {
		enabled[strings.ToLower(ide)] = true
	}
	cfg.IDE.Cursor.Enabled = enabled["cursor"]
	cfg.IDE.VSCode.Enabled = enabled["vscode"]
	cfg.IDE.ClaudeCode.Enabled = enabled["claude"] || enabled["claudecode"] || enabled["claude-code"]
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

func scaffoldProject(root string, layers []config.LayerConfig) error {
	for _, l := range layers {
		dir := filepath.Join(root, l.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		fmt.Printf("  Created: %s\n", dir)
	}
	return nil
}

// readLine is a minimal CLI prompt that reads a line from stdin.
func readLine(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	var input string
	_, _ = fmt.Scanln(&input)
	if input == "" {
		return defaultVal
	}
	return strings.TrimSpace(input)
}

// pickChoice presents a numbered menu and returns the chosen value.
func pickChoice(label string, choices []string, defaultVal string) string {
	fmt.Printf("%s:\n", label)
	for i, c := range choices {
		if c == defaultVal {
			fmt.Printf("  %d) %s (default)\n", i+1, c)
		} else {
			fmt.Printf("  %d) %s\n", i+1, c)
		}
	}
	fmt.Printf("Enter number [%s]: ", defaultVal)
	var input string
	_, _ = fmt.Scanln(&input)
	if input == "" {
		return defaultVal
	}
	for i, c := range choices {
		if fmt.Sprintf("%d", i+1) == strings.TrimSpace(input) {
			return c
		}
	}
	return defaultVal
}
