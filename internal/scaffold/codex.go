package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

const (
	codexLegacyCommandsDir = ".codex/commands"
	codexSkillsDir         = ".agents/skills"
)

// SetupCodex generates Codex skills under <root>/.agents/skills/.
// It writes one SKILL.md file per workflow plus minimal OpenAI skill metadata.
func SetupCodex(out io.Writer, root string) error {
	if err := os.RemoveAll(filepath.Join(root, codexLegacyCommandsDir)); err != nil {
		return fmt.Errorf("remove legacy codex commands: %w", err)
	}
	skillsDir := filepath.Join(root, codexSkillsDir)
	if err := iokit.EnsureDir(skillsDir); err != nil {
		return fmt.Errorf("codex setup: %w", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root)).
		WithIDE("codex").
		Build()

	for _, workflow := range qodeWorkflows {
		content, err := engine.Render("scaffold/"+workflow.Name, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", workflow.Name, err)
		}
		skillDir := filepath.Join(skillsDir, workflow.Name)
		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := iokit.WriteFile(skillPath, []byte(renderCodexSkill(workflow, content)), 0644); err != nil {
			return err
		}
		metadataPath := filepath.Join(skillDir, "agents", "openai.yaml")
		if err := iokit.WriteFile(metadataPath, []byte(renderCodexSkillMetadata(workflow)), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  Codex: .agents/skills/ (%d skills)\n", len(qodeWorkflows))
	return nil
}

func renderCodexSkill(workflow workflowDefinition, content string) string {
	return fmt.Sprintf(
		"---\nname: %q\ndescription: %q\n---\n\n%s",
		workflow.Name,
		workflow.Description,
		content,
	)
}

func renderCodexSkillMetadata(workflow workflowDefinition) string {
	return fmt.Sprintf(
		"interface:\n  display_name: %q\n  short_description: %q\npolicy:\n  allow_implicit_invocation: false\n",
		workflow.Name,
		workflow.Description,
	)
}
