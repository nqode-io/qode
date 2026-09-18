package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

// openCodeCommandsDir is the project-local directory OpenCode reads slash commands from.
const openCodeCommandsDir = ".opencode/commands"

// SetupOpenCode generates OpenCode slash-command files.
func SetupOpenCode(out io.Writer, root string) error {
	if err := iokit.EnsureDir(filepath.Join(root, openCodeCommandsDir)); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root)).
		WithAgent("opencode").
		Build()

	for _, workflow := range qodeWorkflows {
		content, err := engine.Render("scaffold/"+workflow.Name, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", workflow.Name, err)
		}
		p := filepath.Join(root, openCodeCommandsDir, workflow.Name+".md")
		if err := iokit.WriteFile(p, []byte(content), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  OpenCode: .opencode/commands/ (%d commands)\n", len(qodeWorkflows))
	return nil
}
