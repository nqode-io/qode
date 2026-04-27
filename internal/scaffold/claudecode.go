package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

// SetupClaudeCode generates Claude Code configuration files.
func SetupClaudeCode(out io.Writer, root string) error {
	commandsDir := filepath.Join(root, ".claude", "commands")
	if err := iokit.EnsureDir(commandsDir); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root)).
		WithIDE("claude").
		Build()

	for _, workflow := range qodeWorkflows {
		content, err := engine.Render("scaffold/"+workflow.Name, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", workflow.Name, err)
		}
		if err := iokit.WriteFile(filepath.Join(commandsDir, workflow.Name+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  Claude Code: %d slash commands\n", len(qodeWorkflows))
	return nil
}
